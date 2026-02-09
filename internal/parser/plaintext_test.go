package parser_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/frherrer/GoE2E-DocSyncer/internal/parser"
)

var _ = Describe("PlaintextParser", func() {
	var p *parser.PlaintextParser

	BeforeEach(func() {
		var err error
		p, err = parser.NewPlaintextParser(
			`^\s*@begin\((\S+)(?:\s+(.*))?\)\s*$`,
			`^\s*@end\s*$`,
		)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("SupportedExtensions", func() {
		It("should support .txt, .rst, .rtf", func() {
			exts := p.SupportedExtensions()
			Expect(exts).To(ContainElements(".txt", ".rst", ".rtf"))
		})
	})

	Describe("Parse generic.txt", func() {
		var content []byte

		BeforeEach(func() {
			var err error
			content, err = os.ReadFile(filepath.Join("..", "..", "testdata", "plaintext", "generic.txt"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should extract 5 code blocks", func() {
			doc, err := p.Parse("generic.txt", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks).To(HaveLen(5))
		})

		It("should set file type to plaintext", func() {
			doc, err := p.Parse("generic.txt", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.FileType).To(Equal("plaintext"))
		})

		It("should extract step-name attributes", func() {
			doc, err := p.Parse("generic.txt", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks[0].Attributes["step-name"]).To(Equal("Create test namespace"))
		})

		It("should extract timeout attributes", func() {
			doc, err := p.Parse("generic.txt", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks[1].Attributes["timeout"]).To(Equal("60s"))
		})

		It("should extract command content", func() {
			doc, err := p.Parse("generic.txt", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks[0].Content).To(ContainSubstring("kubectl create namespace e2e-test"))
		})

		It("should extract headings from underline-style", func() {
			doc, err := p.Parse("generic.txt", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Headings).ToNot(BeEmpty())
		})
	})

	Describe("Parse sample.rtf", func() {
		var content []byte

		BeforeEach(func() {
			var err error
			content, err = os.ReadFile(filepath.Join("..", "..", "testdata", "plaintext", "sample.rtf"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should strip RTF control words and extract blocks", func() {
			doc, err := p.Parse("sample.rtf", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks).To(HaveLen(2))
		})

		It("should extract step-name attribute from RTF content", func() {
			doc, err := p.Parse("sample.rtf", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks[0].Attributes["step-name"]).To(Equal("Check cluster status"))
		})

		It("should extract command content from RTF", func() {
			doc, err := p.Parse("sample.rtf", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks[0].Content).To(ContainSubstring("kubectl cluster-info"))
		})

		It("should extract timeout attribute from RTF content", func() {
			doc, err := p.Parse("sample.rtf", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks[1].Attributes["timeout"]).To(Equal("30s"))
		})
	})

	It("should return error for invalid regex", func() {
		_, err := parser.NewPlaintextParser("[invalid", `^\s*@end\s*$`)
		Expect(err).To(HaveOccurred())
	})
})
