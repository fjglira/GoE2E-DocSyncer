package converter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/frherrer/GoE2E-DocSyncer/internal/config"
	"github.com/frherrer/GoE2E-DocSyncer/internal/converter"
	"github.com/frherrer/GoE2E-DocSyncer/internal/domain"
)

var _ = Describe("Converter", func() {
	var (
		conv   *converter.DefaultConverter
		cmdCfg *config.CommandConfig
		tagCfg *config.TagConfig
	)

	BeforeEach(func() {
		cmdCfg = &config.CommandConfig{
			DefaultTimeout:          "30s",
			DefaultExpectedExitCode: 0,
			BlockedPatterns:         []string{"rm -rf /", "mkfs"},
			Shell:                   "/bin/sh",
			ShellFlag:               "-c",
		}
		tagCfg = &config.TagConfig{
			StepTags: []string{"go-e2e-step"},
			Attributes: map[string][]string{
				"step_name":          {"step-name", "name"},
				"timeout":           {"timeout"},
				"expected_exit_code": {"expected", "exit-code"},
				"skip_on_failure":   {"skip-on-failure"},
				"template":         {"template"},
				"retry":            {"retry", "retries", "retry-count"},
				"retry_interval":   {"retry-interval", "retry-delay"},
			},
		}
		conv = converter.NewConverter(cmdCfg)
	})

	Describe("Convert", func() {
		It("should convert a document with blocks to TestSpecs", func() {
			doc := &domain.ParsedDocument{
				FilePath: "test.md",
				FileType: "markdown",
				Blocks: []domain.CodeBlock{
					{
						Tag:        "go-e2e-step",
						Content:    "kubectl apply -f deploy.yaml",
						LineNumber: 10,
						Attributes: map[string]string{"step-name": "Deploy app"},
					},
				},
				Headings: []domain.Heading{
					{Level: 1, Text: "Deployment Guide", Line: 1},
				},
				Metadata: map[string]string{},
			}

			specs, err := conv.Convert(doc, tagCfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(specs).To(HaveLen(1))
			Expect(specs[0].DescribeBlock).To(Equal("Deployment Guide"))
			Expect(specs[0].Steps).To(HaveLen(1))
			Expect(specs[0].Steps[0].Name).To(Equal("Deploy app"))
			Expect(specs[0].Steps[0].GoCode).To(ContainSubstring("exec.Command"))
		})

		It("should return nil for document with no blocks", func() {
			doc := &domain.ParsedDocument{
				FilePath: "empty.md",
				FileType: "markdown",
				Blocks:   nil,
				Metadata: map[string]string{},
			}

			specs, err := conv.Convert(doc, tagCfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(specs).To(BeNil())
		})

		It("should use TestGroup as test name when set", func() {
			doc := &domain.ParsedDocument{
				FilePath: "test.md",
				FileType: "markdown",
				Blocks: []domain.CodeBlock{
					{Tag: "go-e2e-step", Content: "echo hello", Attributes: map[string]string{}, TestGroup: "My Custom Test"},
				},
				Headings: []domain.Heading{{Level: 1, Text: "Title", Line: 1}},
				Metadata: map[string]string{"test-start": "My Custom Test"},
			}

			specs, err := conv.Convert(doc, tagCfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(specs[0].TestName).To(Equal("My Custom Test"))
		})

		It("should produce multiple TestSpecs for different TestGroups", func() {
			doc := &domain.ParsedDocument{
				FilePath: "multi.md",
				FileType: "markdown",
				Blocks: []domain.CodeBlock{
					{Tag: "go-e2e-step", Content: "echo step1", Attributes: map[string]string{}, TestGroup: "Group A"},
					{Tag: "go-e2e-step", Content: "echo step2", Attributes: map[string]string{}, TestGroup: "Group A"},
					{Tag: "go-e2e-step", Content: "echo step3", Attributes: map[string]string{}, TestGroup: "Group B"},
				},
				Headings: []domain.Heading{{Level: 1, Text: "Title", Line: 1}},
				Metadata: map[string]string{},
			}

			specs, err := conv.Convert(doc, tagCfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(specs).To(HaveLen(2))
			Expect(specs[0].TestName).To(Equal("Group A"))
			Expect(specs[0].Steps).To(HaveLen(2))
			Expect(specs[1].TestName).To(Equal("Group B"))
			Expect(specs[1].Steps).To(HaveLen(1))
		})

		It("should use filename for ungrouped blocks", func() {
			doc := &domain.ParsedDocument{
				FilePath: "test.md",
				FileType: "markdown",
				Blocks: []domain.CodeBlock{
					{Tag: "go-e2e-step", Content: "echo hello", Attributes: map[string]string{}},
				},
				Headings: []domain.Heading{{Level: 1, Text: "Title", Line: 1}},
				Metadata: map[string]string{},
			}

			specs, err := conv.Convert(doc, tagCfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(specs[0].TestName).To(Equal("test"))
		})

		It("should reject blocked commands", func() {
			doc := &domain.ParsedDocument{
				FilePath: "test.md",
				FileType: "markdown",
				Blocks: []domain.CodeBlock{
					{Tag: "go-e2e-step", Content: "rm -rf /", Attributes: map[string]string{}},
				},
				Headings: []domain.Heading{},
				Metadata: map[string]string{},
			}

			_, err := conv.Convert(doc, tagCfg)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("blocked"))
		})
	})

	Describe("GenerateGoCode", func() {
		It("should generate simple exec.Command for basic commands", func() {
			code := converter.GenerateGoCode("kubectl get pods", 0, "30s", 0, "", cmdCfg)
			Expect(code).To(ContainSubstring("exec.Command"))
			Expect(code).To(ContainSubstring("kubectl"))
			Expect(code).To(ContainSubstring("get"))
			Expect(code).To(ContainSubstring("pods"))
		})

		It("should use shell for complex commands with pipes", func() {
			code := converter.GenerateGoCode("cat file | grep test", 0, "30s", 0, "", cmdCfg)
			Expect(code).To(ContainSubstring("/bin/sh"))
			Expect(code).To(ContainSubstring("-c"))
		})

		It("should wrap with timeout", func() {
			code := converter.GenerateGoCode("echo hello", 0, "60s", 0, "", cmdCfg)
			Expect(code).To(ContainSubstring("time.ParseDuration"))
			Expect(code).To(ContainSubstring("context.WithTimeout"))
			Expect(code).To(ContainSubstring("CommandContext"))
		})

		It("should handle expected exit code", func() {
			code := converter.GenerateGoCode("false", 1, "0s", 0, "", cmdCfg)
			Expect(code).To(ContainSubstring("ExitCode"))
			Expect(code).To(ContainSubstring("Equal(1)"))
		})

		It("should not produce retry wrapper when retry=0", func() {
			code := converter.GenerateGoCode("echo hello", 0, "0s", 0, "", cmdCfg)
			Expect(code).ToNot(ContainSubstring("attempt"))
			Expect(code).ToNot(ContainSubstring("time.Sleep"))
			Expect(code).ToNot(ContainSubstring("lastErr"))
		})

		It("should produce a retry loop with 4 attempts when retry=3", func() {
			code := converter.GenerateGoCode("kubectl get pods", 0, "0s", 3, "2s", cmdCfg)
			Expect(code).To(ContainSubstring("attempt <= 4"))
			Expect(code).To(ContainSubstring("time.Sleep(2 * time.Second)"))
			Expect(code).To(ContainSubstring("lastErr"))
			Expect(code).To(ContainSubstring("lastOutput"))
			Expect(code).To(ContainSubstring("Expect(lastErr).ToNot(HaveOccurred()"))
		})

		It("should use custom retry interval", func() {
			code := converter.GenerateGoCode("echo test", 0, "0s", 2, "5s", cmdCfg)
			Expect(code).To(ContainSubstring("attempt <= 3"))
			Expect(code).To(ContainSubstring("time.Sleep(5 * time.Second)"))
		})

		It("should wrap retry inside timeout", func() {
			code := converter.GenerateGoCode("kubectl get pods", 0, "60s", 3, "2s", cmdCfg)
			// Timeout should be the outermost wrapper
			Expect(code).To(ContainSubstring("context.WithTimeout"))
			// Retry loop should be inside
			Expect(code).To(ContainSubstring("attempt <= 4"))
			Expect(code).To(ContainSubstring("time.Sleep"))
		})
	})

	Describe("Retry attribute resolution", func() {
		It("should resolve retry attribute from block attributes", func() {
			doc := &domain.ParsedDocument{
				FilePath: "test.md",
				FileType: "markdown",
				Blocks: []domain.CodeBlock{
					{
						Tag:        "go-e2e-step",
						Content:    "kubectl get pods",
						LineNumber: 10,
						Attributes: map[string]string{
							"retry":          "3",
							"retry-interval": "5s",
						},
					},
				},
				Headings: []domain.Heading{{Level: 1, Text: "Test", Line: 1}},
				Metadata: map[string]string{},
			}

			specs, err := conv.Convert(doc, tagCfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(specs).To(HaveLen(1))
			Expect(specs[0].Steps[0].RetryCount).To(Equal(3))
			Expect(specs[0].Steps[0].RetryInterval).To(Equal("5s"))
			Expect(specs[0].Steps[0].GoCode).To(ContainSubstring("attempt <= 4"))
			Expect(specs[0].Steps[0].GoCode).To(ContainSubstring("time.Sleep(5 * time.Second)"))
		})

		It("should default retry interval to 2s when not specified", func() {
			doc := &domain.ParsedDocument{
				FilePath: "test.md",
				FileType: "markdown",
				Blocks: []domain.CodeBlock{
					{
						Tag:        "go-e2e-step",
						Content:    "kubectl get pods",
						LineNumber: 10,
						Attributes: map[string]string{
							"retry": "2",
						},
					},
				},
				Headings: []domain.Heading{{Level: 1, Text: "Test", Line: 1}},
				Metadata: map[string]string{},
			}

			specs, err := conv.Convert(doc, tagCfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(specs[0].Steps[0].RetryInterval).To(Equal("2s"))
			Expect(specs[0].Steps[0].GoCode).To(ContainSubstring("time.Sleep(2 * time.Second)"))
		})

		It("should not add retry when attribute is absent", func() {
			doc := &domain.ParsedDocument{
				FilePath: "test.md",
				FileType: "markdown",
				Blocks: []domain.CodeBlock{
					{
						Tag:        "go-e2e-step",
						Content:    "echo hello",
						LineNumber: 10,
						Attributes: map[string]string{},
					},
				},
				Headings: []domain.Heading{{Level: 1, Text: "Test", Line: 1}},
				Metadata: map[string]string{},
			}

			specs, err := conv.Convert(doc, tagCfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(specs[0].Steps[0].RetryCount).To(Equal(0))
			Expect(specs[0].Steps[0].GoCode).ToNot(ContainSubstring("attempt"))
		})

		It("should accept retry-count alias", func() {
			doc := &domain.ParsedDocument{
				FilePath: "test.md",
				FileType: "markdown",
				Blocks: []domain.CodeBlock{
					{
						Tag:        "go-e2e-step",
						Content:    "kubectl get pods",
						LineNumber: 10,
						Attributes: map[string]string{
							"retry-count":  "2",
							"retry-delay":  "3s",
						},
					},
				},
				Headings: []domain.Heading{{Level: 1, Text: "Test", Line: 1}},
				Metadata: map[string]string{},
			}

			specs, err := conv.Convert(doc, tagCfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(specs[0].Steps[0].RetryCount).To(Equal(2))
			Expect(specs[0].Steps[0].RetryInterval).To(Equal("3s"))
		})
	})

	Describe("ValidateCommand", func() {
		It("should pass for safe commands", func() {
			err := converter.ValidateCommand("kubectl get pods", []string{"rm -rf /"})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should block dangerous commands", func() {
			err := converter.ValidateCommand("rm -rf /", []string{"rm -rf /"})
			Expect(err).To(HaveOccurred())
		})
	})
})
