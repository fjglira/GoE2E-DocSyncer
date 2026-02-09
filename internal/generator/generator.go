package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/frherrer/GoE2E-DocSyncer/internal/config"
	"github.com/frherrer/GoE2E-DocSyncer/internal/converter"
	"github.com/frherrer/GoE2E-DocSyncer/internal/domain"
	"github.com/frherrer/GoE2E-DocSyncer/internal/parser"
	"github.com/frherrer/GoE2E-DocSyncer/internal/scanner"
	tmpl "github.com/frherrer/GoE2E-DocSyncer/internal/template"
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
	log        *logrus.Logger
}

// NewGenerator creates a new DefaultGenerator with all dependencies.
func NewGenerator(
	s scanner.Scanner,
	r parser.ParserRegistry,
	c converter.Converter,
	e tmpl.TemplateEngine,
	log *logrus.Logger,
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
		g.log.Debugf("Cleaning output directory: %s", cfg.Output.Directory)
		if err := cleanOutputDir(cfg.Output.Directory); err != nil {
			return domain.NewError("write", cfg.Output.Directory, 0, "failed to clean output directory", err)
		}
	}

	// Step 2: Scan for documentation files
	var allFiles []string
	for _, dir := range cfg.Input.Directories {
		g.log.Debugf("Scanning directory: %s", dir)
		files, err := g.scanner.Scan(dir, cfg.Input.Include, cfg.Input.Exclude)
		if err != nil {
			g.log.Warnf("Failed to scan directory %s: %v", dir, err)
			continue
		}
		allFiles = append(allFiles, files...)
	}

	if len(allFiles) == 0 {
		g.log.Warn("No documentation files found")
		return nil
	}

	g.log.Infof("Found %d documentation file(s)", len(allFiles))

	// Step 3: Parse each file and convert to TestSpecs
	var allSpecs []domain.TestSpec
	for _, filePath := range allFiles {
		g.log.Debugf("Processing: %s", filePath)

		// Read file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			return domain.NewError("parse", filePath, 0, "failed to read file", err)
		}

		// Select parser based on file extension
		ext := filepath.Ext(filePath)
		p, err := g.registry.ParserFor(ext)
		if err != nil {
			g.log.Warnf("No parser for %s, skipping %s", ext, filePath)
			continue
		}

		// Parse document
		doc, err := p.Parse(filePath, content, cfg.Tags.StepTags)
		if err != nil {
			return err
		}

		if len(doc.Blocks) == 0 {
			g.log.Debugf("No tagged blocks found in %s", filePath)
			continue
		}

		g.log.Debugf("Found %d tagged block(s) in %s", len(doc.Blocks), filePath)

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

	g.log.Infof("Generated %d test spec(s)", len(allSpecs))

	// Step 4: Group specs by source file so multiple test groups from one doc
	// are rendered together in a single output file.
	var sourceOrder []string
	specsBySource := make(map[string][]domain.TestSpec)
	for _, spec := range allSpecs {
		if _, seen := specsBySource[spec.SourceFile]; !seen {
			sourceOrder = append(sourceOrder, spec.SourceFile)
		}
		specsBySource[spec.SourceFile] = append(specsBySource[spec.SourceFile], spec)
	}

	// Step 5: Ensure output directory exists
	if !cfg.DryRun {
		if err := os.MkdirAll(cfg.Output.Directory, 0755); err != nil {
			return domain.NewError("write", cfg.Output.Directory, 0, "failed to create output directory", err)
		}
	}

	// Step 6: Render and write output, one file per source document
	for _, sourceFile := range sourceOrder {
		specs := specsBySource[sourceFile]

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

		// Build output filename
		outputFile := buildOutputFilename(sourceFile, cfg.Output)
		outputPath := filepath.Join(cfg.Output.Directory, outputFile)

		if cfg.DryRun {
			g.log.Infof("[DRY-RUN] Would write: %s", outputPath)
			g.log.Debugf("[DRY-RUN] Content:\n%s", rendered)
			continue
		}

		g.log.Infof("Writing: %s", outputPath)
		if err := os.WriteFile(outputPath, []byte(rendered), 0644); err != nil {
			return domain.NewError("write", outputPath, 0, "failed to write output file", err)
		}
	}

	g.log.Info("Generation complete")
	return nil
}

// buildOutputFilename constructs the output filename from the source file path.
func buildOutputFilename(sourceFile string, output config.OutputConfig) string {
	base := filepath.Base(sourceFile)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return fmt.Sprintf("%s%s%s", output.FilePrefix, name, output.FileSuffix)
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
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), "_test.go") {
			path := filepath.Join(dir, entry.Name())
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}

	return nil
}
