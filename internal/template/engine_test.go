package template_test

import (
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/frherrer/GoE2E-DocSyncer/internal/domain"
	tmpl "github.com/frherrer/GoE2E-DocSyncer/internal/template"
)

var _ = Describe("TemplateEngine", func() {
	var engine *tmpl.DefaultEngine

	BeforeEach(func() {
		var err error
		engine, err = tmpl.NewEngine(filepath.Join("..", "..", "templates"), "ginkgo_default")
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

	It("should return error for nonexistent template directory", func() {
		_, err := tmpl.NewEngine("nonexistent_dir", "default")
		Expect(err).To(HaveOccurred())
	})
})
