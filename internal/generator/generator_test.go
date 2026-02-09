package generator_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/frherrer/GoE2E-DocSyncer/internal/config"
	"github.com/frherrer/GoE2E-DocSyncer/internal/converter"
	"github.com/frherrer/GoE2E-DocSyncer/internal/generator"
	"github.com/frherrer/GoE2E-DocSyncer/internal/parser"
	"github.com/frherrer/GoE2E-DocSyncer/internal/scanner"
	tmpl "github.com/frherrer/GoE2E-DocSyncer/internal/template"
)

var _ = Describe("Generator", func() {
	var (
		gen       *generator.DefaultGenerator
		cfg       *config.Config
		outputDir string
		log       *logrus.Logger
	)

	BeforeEach(func() {
		log = logrus.New()
		log.SetLevel(logrus.DebugLevel)

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
		engine, engineErr := tmpl.NewEngine(cfg.Templates.Directory, cfg.Templates.Default)
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
		Expect(len(entries)).To(BeNumerically(">=", 2))

		// Check file names
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		Expect(names).To(ContainElement("generated_simple_test.go"))
		Expect(names).To(ContainElement("generated_multi-step_test.go"))
	})

	It("should generate valid Go code", func() {
		err := gen.Generate(cfg)
		Expect(err).ToNot(HaveOccurred())

		content, err := os.ReadFile(filepath.Join(outputDir, "generated_simple_test.go"))
		Expect(err).ToNot(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("package e2e_test"))
		Expect(string(content)).To(ContainSubstring("Describe"))
		Expect(string(content)).To(ContainSubstring("It"))
	})

	It("should respect dry-run mode", func() {
		cfg.DryRun = true
		err := gen.Generate(cfg)
		Expect(err).ToNot(HaveOccurred())

		// No files should be written in dry-run mode
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
