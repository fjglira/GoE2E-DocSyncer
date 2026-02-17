package generator

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/fjglira/GoE2E-DocSyncer/internal/config"
	"github.com/fjglira/GoE2E-DocSyncer/internal/converter"
	"github.com/fjglira/GoE2E-DocSyncer/internal/domain"
	"github.com/fjglira/GoE2E-DocSyncer/internal/parser"
	"github.com/fjglira/GoE2E-DocSyncer/internal/scanner"
	tmpl "github.com/fjglira/GoE2E-DocSyncer/internal/template"
)

// Generator is the top-level orchestrator.
type Generator interface {
	Generate(cfg *config.Config) error
}

// DefaultGenerator implements Generator by wiring all components together.
type DefaultGenerator struct {
	scanner    scanner.Scanner
	registry   parser.ParserRegistry
	converter  converter.Converter
	engine     tmpl.TemplateEngine
	log        *slog.Logger
}

// NewGenerator creates a new DefaultGenerator with all dependencies.
func NewGenerator(
	s scanner.Scanner,
	r parser.ParserRegistry,
	c converter.Converter,
	e tmpl.TemplateEngine,
	log *slog.Logger,
) *DefaultGenerator {
	return &DefaultGenerator{
		scanner:   s,
		registry:  r,
		converter: c,
		engine:    e,
		log:       log,
	}
}

// Generate runs the full pipeline: scan → parse → convert → render → write.
func (g *DefaultGenerator) Generate(cfg *config.Config) error {
	// Step 1: Clean output directory if configured
	if cfg.Output.CleanBeforeGenerate && !cfg.DryRun {
		g.log.Debug("Cleaning output directory", "path", cfg.Output.Directory)
		if err := cleanOutputDir(cfg.Output.Directory); err != nil {
			return domain.NewErrorWithSuggestion("write", cfg.Output.Directory, 0,
				"failed to clean output directory",
				"check file permissions or set output.clean_before_generate to false in docsyncer.yaml",
				err)
		}
	}

	// Step 2: Scan for documentation files
	var allFiles []string
	for _, dir := range cfg.Input.Directories {
		g.log.Debug("Scanning directory", "path", dir)
		files, err := g.scanner.Scan(dir, cfg.Input.Include, cfg.Input.Exclude)
		if err != nil {
			g.log.Warn("Failed to scan directory", "path", dir, "error", err)
			continue
		}
		allFiles = append(allFiles, files...)
	}

	if len(allFiles) == 0 {
		g.log.Warn("No documentation files found")
		return nil
	}

	g.log.Info("Found documentation file(s)", "count", len(allFiles))

	// Step 3: Parse each file and convert to TestSpecs
	var allSpecs []domain.TestSpec
	for _, filePath := range allFiles {
		g.log.Debug("Processing", "path", filePath)

		// Read file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			return domain.NewErrorWithSuggestion("parse", filePath, 0,
				"failed to read file",
				"check that the file exists and has read permissions",
				err)
		}

		// Select parser based on file extension
		ext := filepath.Ext(filePath)
		p, err := g.registry.ParserFor(ext)
		if err != nil {
			g.log.Warn("No parser found, skipping", "ext", ext, "path", filePath)
			continue
		}

		// Parse document
		doc, err := p.Parse(filePath, content, cfg.Tags.StepTags)
		if err != nil {
			return err
		}

		if len(doc.Blocks) == 0 {
			g.log.Debug("No tagged blocks found", "path", filePath)
			continue
		}

		g.log.Debug("Found tagged block(s)", "count", len(doc.Blocks), "path", filePath)

		// Convert to TestSpecs
		specs, err := g.converter.Convert(doc, &cfg.Tags)
		if err != nil {
			return err
		}

		allSpecs = append(allSpecs, specs...)
	}

	if len(allSpecs) == 0 {
		g.log.Warn("No test specs generated from documentation")
		return nil
	}

	// Populate labels on each spec: default labels + DescribeBlock name (deduplicated)
	for i := range allSpecs {
		allSpecs[i].Labels = buildLabels(cfg.Output.DefaultLabels, allSpecs[i].DescribeBlock)
	}

	g.log.Info("Generated test spec(s)", "count", len(allSpecs))

	// Step 4: Group specs by output key.
	// If spec has TestFile set, use TestFile as the grouping key (each unique TestFile → separate output file).
	// Otherwise, fall back to SourceFile (existing behavior).
	var keyOrder []string
	specsByKey := make(map[string][]domain.TestSpec)
	for _, spec := range allSpecs {
		key := spec.SourceFile
		if spec.TestFile != "" {
			key = spec.TestFile
		}
		if _, seen := specsByKey[key]; !seen {
			keyOrder = append(keyOrder, key)
		}
		specsByKey[key] = append(specsByKey[key], spec)
	}

	// Step 5: Ensure output directory exists
	if !cfg.DryRun {
		if err := os.MkdirAll(cfg.Output.Directory, 0755); err != nil {
			return domain.NewErrorWithSuggestion("write", cfg.Output.Directory, 0,
				"failed to create output directory",
				"check that the parent directory exists and has write permissions",
				err)
		}
	}

	// Step 6: Render and write output, one file per grouping key
	for _, key := range keyOrder {
		specs := specsByKey[key]

		var rendered string
		var err error
		if len(specs) > 1 {
			rendered, err = g.engine.RenderMulti(specs, cfg.Output.PackageName)
		} else {
			rendered, err = g.engine.Render(specs[0], cfg.Output.PackageName)
		}
		if err != nil {
			return err
		}

		// Build output filename — use TestFile-based name when available
		isTestFile := specs[0].TestFile != ""
		outputFile := buildOutputFilename(key, isTestFile, cfg.Output)
		outputPath := filepath.Join(cfg.Output.Directory, outputFile)

		if cfg.DryRun {
			g.log.Info("[DRY-RUN] Would write", "path", outputPath)
			g.log.Debug("[DRY-RUN] Content", "content", rendered)
			continue
		}

		g.log.Info("Writing", "path", outputPath)
		if err := os.WriteFile(outputPath, []byte(rendered), 0644); err != nil {
			return domain.NewErrorWithSuggestion("write", outputPath, 0,
				"failed to write output file",
				"check disk space and write permissions for the output directory",
				err)
		}
	}

	// Step 7: Generate suite_test.go if it doesn't already exist
	if err := writeSuiteFile(cfg, g.log); err != nil {
		return err
	}

	g.log.Info("Generation complete")
	return nil
}

// buildOutputFilename constructs the output filename.
// When isTestFile is true, the key is a TestFile name that gets sanitized
// (lowercase, spaces→underscores, strip non-alphanum). Otherwise, the key
// is a source file path and we use the basename without extension.
func buildOutputFilename(key string, isTestFile bool, output config.OutputConfig) string {
	var name string
	if isTestFile {
		name = sanitizeTestFileName(key)
	} else {
		base := filepath.Base(key)
		ext := filepath.Ext(base)
		name = strings.TrimSuffix(base, ext)
	}
	return fmt.Sprintf("%s%s%s", output.FilePrefix, name, output.FileSuffix)
}

// sanitizeTestFileName converts a test-start name into a valid filename component.
// e.g. "Istiod HA ReplicaCount" → "istiod_ha_replicacount"
func sanitizeTestFileName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	var b strings.Builder
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			b.WriteRune(c)
		}
	}
	result := b.String()
	// Collapse multiple underscores
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}
	return strings.Trim(result, "_")
}

// writeSuiteFile generates a suite_test.go bootstrap file in the output directory.
// It skips writing if the file already exists to avoid overwriting user-maintained files.
func writeSuiteFile(cfg *config.Config, log *slog.Logger) error {
	suitePath := filepath.Join(cfg.Output.Directory, "suite_test.go")

	// Skip if file already exists
	if _, err := os.Stat(suitePath); err == nil {
		log.Debug("suite_test.go already exists, skipping", "path", suitePath)
		return nil
	}

	testFunc := packageNameToTestFunc(cfg.Output.PackageName)
	suiteDesc := strings.ReplaceAll(testFunc, "Test", "")
	// If stripping "Test" prefix leaves it empty, use the full name
	if suiteDesc == "" {
		suiteDesc = testFunc
	}

	var buildTag string
	if cfg.Output.BuildTag != "" {
		buildTag = fmt.Sprintf("//go:build %s\n\n", cfg.Output.BuildTag)
	}

	content := fmt.Sprintf(`%spackage %s

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func %s(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "%s Suite")
}

var _ = BeforeSuite(func() {
	// Add setup code here
})

var _ = AfterSuite(func() {
	// Add teardown code here
})
`, buildTag, cfg.Output.PackageName, testFunc, suiteDesc)

	if cfg.DryRun {
		log.Info("[DRY-RUN] Would write", "path", suitePath)
		log.Debug("[DRY-RUN] Content", "content", content)
		return nil
	}

	log.Info("Writing", "path", suitePath)
	if err := os.WriteFile(suitePath, []byte(content), 0644); err != nil {
		return domain.NewErrorWithSuggestion("write", suitePath, 0,
			"failed to write suite file",
			"check disk space and write permissions for the output directory",
			err)
	}
	return nil
}

// packageNameToTestFunc converts a Go package name to a Test function name.
// e.g. "e2e_generated" → "TestE2eGenerated", "e2e_test" → "TestE2eTest"
func packageNameToTestFunc(pkg string) string {
	parts := strings.Split(pkg, "_")
	var b strings.Builder
	b.WriteString("Test")
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]) + strings.ToLower(part[1:]))
	}
	return b.String()
}

// buildLabels creates a deduplicated label list from default labels plus the test name.
func buildLabels(defaults []string, testName string) []string {
	seen := make(map[string]bool, len(defaults)+1)
	var labels []string
	for _, l := range defaults {
		if !seen[l] {
			seen[l] = true
			labels = append(labels, l)
		}
	}
	if testName != "" && !seen[testName] {
		labels = append(labels, testName)
	}
	return labels
}

// cleanOutputDir removes all generated files from the output directory.
func cleanOutputDir(dir string) error {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil // Nothing to clean
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), "_test.go") && entry.Name() != "suite_test.go" {
			path := filepath.Join(dir, entry.Name())
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}

	return nil
}
