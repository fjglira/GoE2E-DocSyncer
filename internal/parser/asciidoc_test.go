package parser_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/frherrer/GoE2E-DocSyncer/internal/parser"
)

var _ = Describe("AsciiDocParser", func() {
	var p *parser.AsciiDocParser

	BeforeEach(func() {
		p = parser.NewAsciiDocParser()
	})

	Describe("SupportedExtensions", func() {
		It("should support .adoc and .asciidoc", func() {
			exts := p.SupportedExtensions()
			Expect(exts).To(ContainElements(".adoc", ".asciidoc"))
		})
	})

	Describe("Parse sample.adoc", func() {
		var content []byte

		BeforeEach(func() {
			var err error
			content, err = os.ReadFile(filepath.Join("..", "..", "testdata", "asciidoc", "sample.adoc"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should extract 5 code blocks", func() {
			doc, err := p.Parse("sample.adoc", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks).To(HaveLen(5))
		})

		It("should set file type to asciidoc", func() {
			doc, err := p.Parse("sample.adoc", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.FileType).To(Equal("asciidoc"))
		})

		It("should extract step-name attributes", func() {
			doc, err := p.Parse("sample.adoc", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks[0].Attributes["step-name"]).To(Equal("Build Docker image"))
		})

		It("should extract timeout attributes", func() {
			doc, err := p.Parse("sample.adoc", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks[1].Attributes["timeout"]).To(Equal("5m"))
		})

		It("should extract headings", func() {
			doc, err := p.Parse("sample.adoc", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Headings).ToNot(BeEmpty())
		})

		It("should set context from nearest heading", func() {
			doc, err := p.Parse("sample.adoc", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks[0].Context).ToNot(BeEmpty())
		})
	})
})
