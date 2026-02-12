package template_test

import (
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fjglira/GoE2E-DocSyncer/internal/domain"
	tmpl "github.com/fjglira/GoE2E-DocSyncer/internal/template"
)

var _ = Describe("TemplateEngine", func() {
	var engine *tmpl.DefaultEngine

	BeforeEach(func() {
		var err error
		engine, err = tmpl.NewEngine(filepath.Join("..", "..", "templates"), "ginkgo_default", "")
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("ListTemplates", func() {
		It("should list available templates", func() {
			templates := engine.ListTemplates()
			Expect(templates).To(ContainElement("ginkgo_default"))
		})
	})

	Describe("Render", func() {
		It("should render a TestSpec to valid Go code", func() {
			spec := domain.TestSpec{
				SourceFile:    "test.md",
				SourceType:    "markdown",
				TestName:      "Simple test",
				DescribeBlock: "My Feature",
				Steps: []domain.TestStep{
					{
						Name:   "Run command",
						GoCode: `cmd := exec.Command("echo", "hello")` + "\n" + `output, err := cmd.CombinedOutput()` + "\n" + `Expect(err).ToNot(HaveOccurred(), string(output))`,
					},
				},
			}

			result, err := engine.Render(spec, "e2e_test")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ContainSubstring("package e2e_test"))
			Expect(result).To(ContainSubstring(`Describe("My Feature"`))
			Expect(result).To(ContainSubstring(`It("Simple test"`))
			Expect(result).To(ContainSubstring(`By("Run command")`))
		})

		It("should include context imports when needed", func() {
			spec := domain.TestSpec{
				SourceFile:    "test.md",
				SourceType:    "markdown",
				TestName:      "Timeout test",
				DescribeBlock: "Timeout Feature",
				Steps: []domain.TestStep{
					{
						Name:   "Run with timeout",
						GoCode: `ctx, cancel := context.WithTimeout(context.Background(), dur)` + "\n" + `defer cancel()` + "\n" + `cmd := exec.CommandContext(ctx, "echo", "hello")` + "\n" + `output, err := cmd.CombinedOutput()` + "\n" + `Expect(err).ToNot(HaveOccurred(), string(output))`,
					},
				},
			}

			result, err := engine.Render(spec, "e2e_test")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ContainSubstring(`"context"`))
			Expect(result).To(ContainSubstring(`"time"`))
		})

		It("should render context block when present", func() {
			spec := domain.TestSpec{
				SourceFile:    "test.md",
				SourceType:    "markdown",
				TestName:      "My test",
				DescribeBlock: "Feature",
				ContextBlock:  "Sub-feature",
				Steps: []domain.TestStep{
					{
						Name:   "Step 1",
						GoCode: `cmd := exec.Command("ls")` + "\n" + `output, err := cmd.CombinedOutput()` + "\n" + `Expect(err).ToNot(HaveOccurred(), string(output))`,
					},
				},
			}

			result, err := engine.Render(spec, "e2e_test")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ContainSubstring(`Context("Sub-feature"`))
		})

		It("should produce go/format compliant code", func() {
			spec := domain.TestSpec{
				SourceFile:    "test.md",
				SourceType:    "markdown",
				TestName:      "Format test",
				DescribeBlock: "Feature",
				Steps: []domain.TestStep{
					{
						Name:   "Step",
						GoCode: `cmd := exec.Command("echo")` + "\n" + `output, err := cmd.CombinedOutput()` + "\n" + `Expect(err).ToNot(HaveOccurred(), string(output))`,
					},
				},
			}

			result, err := engine.Render(spec, "e2e_test")
			Expect(err).ToNot(HaveOccurred())
			// go/format output should not have excessive whitespace
			Expect(strings.Contains(result, "\t\t\t\t\t\t")).To(BeFalse())
		})
	})

	Describe("RenderMulti", func() {
		It("should render multiple specs into a single file with multiple It blocks", func() {
			specs := []domain.TestSpec{
				{
					SourceFile:    "multi.md",
					SourceType:    "markdown",
					TestName:      "Group A",
					DescribeBlock: "My Feature",
					Steps: []domain.TestStep{
						{
							Name:   "Step A1",
							GoCode: `cmd := exec.Command("echo", "a1")` + "\n" + `output, err := cmd.CombinedOutput()` + "\n" + `Expect(err).ToNot(HaveOccurred(), string(output))`,
						},
					},
				},
				{
					SourceFile:    "multi.md",
					SourceType:    "markdown",
					TestName:      "Group B",
					DescribeBlock: "My Feature",
					Steps: []domain.TestStep{
						{
							Name:   "Step B1",
							GoCode: `cmd := exec.Command("echo", "b1")` + "\n" + `output, err := cmd.CombinedOutput()` + "\n" + `Expect(err).ToNot(HaveOccurred(), string(output))`,
						},
					},
				},
			}

			result, err := engine.RenderMulti(specs, "e2e_test")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ContainSubstring(`Describe("My Feature"`))
			Expect(result).To(ContainSubstring(`It("Group A"`))
			Expect(result).To(ContainSubstring(`It("Group B"`))
			Expect(result).To(ContainSubstring(`By("Step A1")`))
			Expect(result).To(ContainSubstring(`By("Step B1")`))
		})

		It("should detect context imports across all specs", func() {
			specs := []domain.TestSpec{
				{
					SourceFile:    "multi.md",
					SourceType:    "markdown",
					TestName:      "No timeout",
					DescribeBlock: "Feature",
					Steps: []domain.TestStep{
						{Name: "Simple", GoCode: `cmd := exec.Command("echo")` + "\n" + `output, err := cmd.CombinedOutput()` + "\n" + `Expect(err).ToNot(HaveOccurred(), string(output))`},
					},
				},
				{
					SourceFile:    "multi.md",
					SourceType:    "markdown",
					TestName:      "Has timeout",
					DescribeBlock: "Feature",
					Steps: []domain.TestStep{
						{Name: "With timeout", GoCode: `ctx, cancel := context.WithTimeout(context.Background(), dur)` + "\n" + `defer cancel()` + "\n" + `cmd := exec.CommandContext(ctx, "echo")` + "\n" + `output, err := cmd.CombinedOutput()` + "\n" + `Expect(err).ToNot(HaveOccurred(), string(output))`},
					},
				},
			}

			result, err := engine.RenderMulti(specs, "e2e_test")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ContainSubstring(`"context"`))
			Expect(result).To(ContainSubstring(`"time"`))
		})
	})

	Describe("BuildTag", func() {
		It("should include build tag when configured", func() {
			engine, err := tmpl.NewEngine(filepath.Join("..", "..", "templates"), "ginkgo_default", "e2e")
			Expect(err).ToNot(HaveOccurred())

			spec := domain.TestSpec{
				SourceFile:    "test.md",
				SourceType:    "markdown",
				TestName:      "Tag test",
				DescribeBlock: "Feature",
				Steps: []domain.TestStep{
					{
						Name:   "Step",
						GoCode: `cmd := exec.Command("echo")` + "\n" + `output, err := cmd.CombinedOutput()` + "\n" + `Expect(err).ToNot(HaveOccurred(), string(output))`,
					},
				},
			}

			result, err := engine.Render(spec, "e2e_test")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HavePrefix("//go:build e2e\n"))
		})

		It("should omit build tag when empty", func() {
			spec := domain.TestSpec{
				SourceFile:    "test.md",
				SourceType:    "markdown",
				TestName:      "No tag test",
				DescribeBlock: "Feature",
				Steps: []domain.TestStep{
					{
						Name:   "Step",
						GoCode: `cmd := exec.Command("echo")` + "\n" + `output, err := cmd.CombinedOutput()` + "\n" + `Expect(err).ToNot(HaveOccurred(), string(output))`,
					},
				},
			}

			result, err := engine.Render(spec, "e2e_test")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(ContainSubstring("//go:build"))
		})
	})

	Describe("Embedded template fallback", func() {
		It("should fall back to embedded template for nonexistent directory", func() {
			engine, err := tmpl.NewEngine("nonexistent_dir", "ginkgo_default", "")
			Expect(err).ToNot(HaveOccurred())
			Expect(engine.ListTemplates()).To(ContainElement("ginkgo_default"))
		})

		It("should fall back to embedded template when directory is empty string", func() {
			engine, err := tmpl.NewEngine("", "ginkgo_default", "")
			Expect(err).ToNot(HaveOccurred())
			Expect(engine.ListTemplates()).To(ContainElement("ginkgo_default"))
		})

		It("should render using embedded template", func() {
			engine, err := tmpl.NewEngine("", "ginkgo_default", "")
			Expect(err).ToNot(HaveOccurred())

			spec := domain.TestSpec{
				SourceFile:    "test.adoc",
				SourceType:    "asciidoc",
				TestName:      "Embedded test",
				DescribeBlock: "Embedded Feature",
				Steps: []domain.TestStep{
					{
						Name:   "Run command",
						GoCode: `cmd := exec.Command("echo", "hello")` + "\n" + `output, err := cmd.CombinedOutput()` + "\n" + `Expect(err).ToNot(HaveOccurred(), string(output))`,
					},
				},
			}

			result, err := engine.Render(spec, "e2e_test")
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ContainSubstring("package e2e_test"))
			Expect(result).To(ContainSubstring(`Describe("Embedded Feature"`))
			Expect(result).To(ContainSubstring(`It("Embedded test"`))
		})
	})
})
