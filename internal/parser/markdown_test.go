package parser_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/frherrer/GoE2E-DocSyncer/internal/parser"
)

var _ = Describe("MarkdownParser", func() {
	var p *parser.MarkdownParser

	BeforeEach(func() {
		p = parser.NewMarkdownParser()
	})

	Describe("SupportedExtensions", func() {
		It("should support .md and .markdown", func() {
			exts := p.SupportedExtensions()
			Expect(exts).To(ContainElements(".md", ".markdown"))
		})
	})

	Describe("Parse simple.md", func() {
		var content []byte

		BeforeEach(func() {
			var err error
			content, err = os.ReadFile(filepath.Join("..", "..", "testdata", "markdown", "simple.md"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should extract 3 code blocks", func() {
			doc, err := p.Parse("simple.md", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks).To(HaveLen(3))
		})

		It("should set file type to markdown", func() {
			doc, err := p.Parse("simple.md", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.FileType).To(Equal("markdown"))
		})

		It("should extract step name attribute", func() {
			doc, err := p.Parse("simple.md", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks[0].Attributes["step-name"]).To(Equal("Apply deployment manifests"))
		})

		It("should extract timeout attribute", func() {
			doc, err := p.Parse("simple.md", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks[1].Attributes["timeout"]).To(Equal("60s"))
		})

		It("should extract headings", func() {
			doc, err := p.Parse("simple.md", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Headings).ToNot(BeEmpty())
			Expect(doc.Headings[0].Text).To(Equal("Simple Deployment Guide"))
		})

		It("should extract test-start metadata", func() {
			doc, err := p.Parse("simple.md", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Metadata["test-start"]).To(Equal("Simple deployment test"))
		})

		It("should not extract blocks with non-matching tags", func() {
			doc, err := p.Parse("simple.md", content, []string{"other-tag"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks).To(BeEmpty())
		})
	})

	Describe("Parse multi-step.md", func() {
		var content []byte

		BeforeEach(func() {
			var err error
			content, err = os.ReadFile(filepath.Join("..", "..", "testdata", "markdown", "multi-step.md"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should extract 5 code blocks", func() {
			doc, err := p.Parse("multi-step.md", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks).To(HaveLen(5))
		})

		It("should set context from nearest heading", func() {
			doc, err := p.Parse("multi-step.md", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			// Each block should have context set from the nearest heading
			Expect(doc.Blocks[0].Context).ToNot(BeEmpty())
		})

		It("should ignore bash code blocks", func() {
			doc, err := p.Parse("multi-step.md", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			for _, block := range doc.Blocks {
				Expect(block.Tag).To(Equal("go-e2e-step"))
			}
		})
	})
})
