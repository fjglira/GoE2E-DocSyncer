package generator_test

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fjglira/GoE2E-DocSyncer/internal/config"
	"github.com/fjglira/GoE2E-DocSyncer/internal/converter"
	"github.com/fjglira/GoE2E-DocSyncer/internal/generator"
	"github.com/fjglira/GoE2E-DocSyncer/internal/parser"
	"github.com/fjglira/GoE2E-DocSyncer/internal/scanner"
	tmpl "github.com/fjglira/GoE2E-DocSyncer/internal/template"
)

var _ = Describe("Generator", func() {
	var (
		gen       *generator.DefaultGenerator
		cfg       *config.Config
		outputDir string
		log       *slog.Logger
	)

	BeforeEach(func() {
		log = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))

		var err error
		outputDir, err = os.MkdirTemp("", "docsyncer-test-*")
		Expect(err).ToNot(HaveOccurred())

		cfg = config.DefaultConfig()
		cfg.Input.Directories = []string{
			filepath.Join("..", "..", "testdata", "markdown"),
		}
		cfg.Input.Include = []string{"*.md"}
		cfg.Output.Directory = outputDir
		cfg.Output.FilePrefix = "generated_"
		cfg.Output.FileSuffix = "_test.go"
		cfg.Output.PackageName = "e2e_test"
		cfg.Templates.Directory = filepath.Join("..", "..", "templates")
		cfg.Templates.Default = "ginkgo_default"

		// Set up components
		s := scanner.NewScanner(true)
		registry := parser.NewRegistry()
		registry.Register(parser.NewMarkdownParser())
		conv := converter.NewConverter(&cfg.Commands)
		engine, engineErr := tmpl.NewEngine(cfg.Templates.Directory, cfg.Templates.Default, cfg.Output.BuildTag)
		Expect(engineErr).ToNot(HaveOccurred())

		gen = generator.NewGenerator(s, registry, conv, engine, log)
	})

	AfterEach(func() {
		os.RemoveAll(outputDir)
	})

	It("should generate test files from markdown docs", func() {
		err := gen.Generate(cfg)
		Expect(err).ToNot(HaveOccurred())

		// Check that output files were created
		entries, err := os.ReadDir(outputDir)
		Expect(err).ToNot(HaveOccurred())

		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		// simple.md → 1 file (TestFile-based), multi-step.md → 2 files (TestFile-based)
		Expect(names).To(ContainElement("generated_simple_deployment_test_test.go"))
		Expect(names).To(ContainElement("generated_infrastructure_provisioning_test.go"))
		Expect(names).To(ContainElement("generated_application_deployment_test.go"))
	})

	It("should generate valid Go code", func() {
		err := gen.Generate(cfg)
		Expect(err).ToNot(HaveOccurred())

		content, err := os.ReadFile(filepath.Join(outputDir, "generated_simple_deployment_test_test.go"))
		Expect(err).ToNot(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("package e2e_test"))
		Expect(string(content)).To(ContainSubstring("Describe"))
		Expect(string(content)).To(ContainSubstring("It"))
	})

	It("should generate separate files for each test-start/end pair in multi-step.md", func() {
		err := gen.Generate(cfg)
		Expect(err).ToNot(HaveOccurred())

		// Infrastructure provisioning file should have 2 It blocks (from test-step-start/end)
		content, err := os.ReadFile(filepath.Join(outputDir, "generated_infrastructure_provisioning_test.go"))
		Expect(err).ToNot(HaveOccurred())
		contentStr := string(content)
		Expect(contentStr).To(ContainSubstring(`It("Setup Database"`))
		Expect(contentStr).To(ContainSubstring(`It("Wait for Ready"`))
		Expect(contentStr).To(ContainSubstring(`Describe("Infrastructure provisioning"`))

		// Application deployment file should have a single It block (no test-step-start/end)
		content2, err := os.ReadFile(filepath.Join(outputDir, "generated_application_deployment_test.go"))
		Expect(err).ToNot(HaveOccurred())
		contentStr2 := string(content2)
		Expect(contentStr2).To(ContainSubstring(`It("Application deployment"`))
		Expect(contentStr2).To(ContainSubstring(`Describe("Application deployment"`))
	})

	It("should generate suite_test.go in the output directory", func() {
		err := gen.Generate(cfg)
		Expect(err).ToNot(HaveOccurred())

		suitePath := filepath.Join(outputDir, "suite_test.go")
		content, err := os.ReadFile(suitePath)
		Expect(err).ToNot(HaveOccurred())

		contentStr := string(content)
		Expect(contentStr).To(ContainSubstring("package e2e_test"))
		Expect(contentStr).To(ContainSubstring("func TestE2eTest(t *testing.T)"))
		Expect(contentStr).To(ContainSubstring("RunSpecs(t,"))
		Expect(contentStr).To(ContainSubstring(`"testing"`))
		Expect(contentStr).To(ContainSubstring(`. "github.com/onsi/ginkgo/v2"`))
		Expect(contentStr).To(ContainSubstring(`. "github.com/onsi/gomega"`))
	})

	It("should not overwrite existing suite_test.go", func() {
		// Disable clean so our pre-existing file survives
		cfg.Output.CleanBeforeGenerate = false

		suitePath := filepath.Join(outputDir, "suite_test.go")
		customContent := "// custom suite file\npackage e2e_test\n"
		err := os.MkdirAll(outputDir, 0755)
		Expect(err).ToNot(HaveOccurred())
		err = os.WriteFile(suitePath, []byte(customContent), 0644)
		Expect(err).ToNot(HaveOccurred())

		err = gen.Generate(cfg)
		Expect(err).ToNot(HaveOccurred())

		content, err := os.ReadFile(suitePath)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(content)).To(Equal(customContent))
	})

	It("should respect dry-run mode", func() {
		cfg.DryRun = true
		err := gen.Generate(cfg)
		Expect(err).ToNot(HaveOccurred())

		// No files should be written in dry-run mode (including suite_test.go)
		entries, err := os.ReadDir(outputDir)
		Expect(err).ToNot(HaveOccurred())
		Expect(entries).To(BeEmpty())
	})

	It("should handle empty directory gracefully", func() {
		emptyDir, err := os.MkdirTemp("", "docsyncer-empty-*")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(emptyDir)

		cfg.Input.Directories = []string{emptyDir}
		err = gen.Generate(cfg)
		Expect(err).ToNot(HaveOccurred())
	})
})
