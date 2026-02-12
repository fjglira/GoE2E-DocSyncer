package config_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/frherrer/GoE2E-DocSyncer/internal/config"
)

var _ = Describe("Config", func() {
	Describe("Load", func() {
		It("should load minimal config", func() {
			cfg, err := config.Load(filepath.Join("..", "..", "testdata", "configs", "minimal.yaml"))
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())
			Expect(cfg.Input.Directories).To(ContainElement("docs"))
			Expect(cfg.Tags.StepTags).To(ContainElement("go-e2e-step"))
			Expect(cfg.Output.PackageName).To(Equal("e2e_generated"))
		})

		It("should load full config", func() {
			cfg, err := config.Load(filepath.Join("..", "..", "testdata", "configs", "full.yaml"))
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())
			Expect(cfg.Input.Directories).To(HaveLen(3))
			Expect(cfg.Input.Include).To(ContainElement("*.adoc"))
			Expect(cfg.Input.Exclude).To(ContainElement("vendor/**"))
			Expect(cfg.Tags.StepTags).To(ContainElements("go-e2e-step", "e2e-test", "test-step"))
			Expect(cfg.Commands.Shell).To(Equal("/bin/sh"))
			Expect(cfg.Commands.BlockedPatterns).To(ContainElement("rm -rf /"))
		})

		It("should return error for nonexistent file", func() {
			_, err := config.Load("nonexistent.yaml")
			Expect(err).To(HaveOccurred())
		})

		It("should return error for invalid YAML", func() {
			tmpFile := filepath.Join(os.TempDir(), "invalid_docsyncer.yaml")
			err := os.WriteFile(tmpFile, []byte("{{invalid yaml}}"), 0644)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(tmpFile)

			_, loadErr := config.Load(tmpFile)
			Expect(loadErr).To(HaveOccurred())
		})
	})

	Describe("DefaultConfig", func() {
		It("should return config with sensible defaults", func() {
			cfg := config.DefaultConfig()
			Expect(cfg).ToNot(BeNil())
			Expect(cfg.Input.Directories).To(ContainElement("docs"))
			Expect(cfg.Input.Include).To(ContainElement("*.md"))
			Expect(*cfg.Input.Recursive).To(BeTrue())
			Expect(cfg.Tags.StepTags).To(ContainElement("go-e2e-step"))
			Expect(cfg.Output.Directory).To(Equal("tests/e2e/generated"))
			Expect(cfg.Output.PackageName).To(Equal("e2e_generated"))
			Expect(cfg.Output.FileSuffix).To(Equal("_test.go"))
			Expect(cfg.Commands.DefaultTimeout).To(Equal("30s"))
			Expect(cfg.Logging.Level).To(Equal("info"))
		})
	})

	Describe("Validate", func() {
		It("should pass for valid config", func() {
			cfg, err := config.Load(filepath.Join("..", "..", "testdata", "configs", "full.yaml"))
			Expect(err).ToNot(HaveOccurred())
			Expect(config.Validate(cfg)).To(Succeed())
		})

		It("should fail if directories are empty", func() {
			cfg := config.DefaultConfig()
			cfg.Input.Directories = nil
			err := config.Validate(cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("input.directories"))
		})

		It("should fail if step_tags are empty", func() {
			cfg := config.DefaultConfig()
			cfg.Tags.StepTags = nil
			err := config.Validate(cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("tags.step_tags"))
		})

		It("should fail if output directory is empty", func() {
			cfg := config.DefaultConfig()
			cfg.Output.Directory = ""
			err := config.Validate(cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("output.directory"))
		})

		It("should fail if file suffix doesn't end with .go", func() {
			cfg := config.DefaultConfig()
			cfg.Output.FileSuffix = "_test.txt"
			err := config.Validate(cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("file_suffix"))
		})

		It("should fail for invalid log level", func() {
			cfg := config.DefaultConfig()
			cfg.Logging.Level = "verbose"
			err := config.Validate(cfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("logging.level"))
		})
	})
})
