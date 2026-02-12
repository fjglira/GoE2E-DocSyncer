package parser_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fjglira/GoE2E-DocSyncer/internal/parser"
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

	Describe("Parse test-start/end markers", func() {
		It("should assign TestFile from test-start markers", func() {
			content := []byte(`= My Guide

// test-start: First Test

[source,go-e2e-step]
----
echo step1
----

// test-end

// test-start: Second Test

[source,go-e2e-step]
----
echo step2
----

// test-end
`)
			doc, err := p.Parse("test.adoc", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks).To(HaveLen(2))
			Expect(doc.Blocks[0].TestFile).To(Equal("First Test"))
			Expect(doc.Blocks[1].TestFile).To(Equal("Second Test"))
		})

		It("should clear TestFile after test-end", func() {
			content := []byte(`= My Guide

// test-start: Grouped

[source,go-e2e-step]
----
echo grouped
----

// test-end

[source,go-e2e-step]
----
echo ungrouped
----
`)
			doc, err := p.Parse("test.adoc", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks).To(HaveLen(2))
			Expect(doc.Blocks[0].TestFile).To(Equal("Grouped"))
			Expect(doc.Blocks[1].TestFile).To(BeEmpty())
		})
	})

	Describe("Parse test-step-start/end markers", func() {
		It("should assign StepGroup from test-step-start markers", func() {
			content := []byte(`= My Guide

// test-start: My Test

// test-step-start: Setup

[source,go-e2e-step]
----
echo setup
----

// test-step-end

// test-step-start: Verify

[source,go-e2e-step]
----
echo verify
----

// test-step-end

// test-end
`)
			doc, err := p.Parse("test.adoc", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks).To(HaveLen(2))
			Expect(doc.Blocks[0].TestFile).To(Equal("My Test"))
			Expect(doc.Blocks[0].StepGroup).To(Equal("Setup"))
			Expect(doc.Blocks[1].TestFile).To(Equal("My Test"))
			Expect(doc.Blocks[1].StepGroup).To(Equal("Verify"))
		})

		It("should clear StepGroup after test-step-end", func() {
			content := []byte(`= My Guide

// test-start: My Test

// test-step-start: Setup

[source,go-e2e-step]
----
echo setup
----

// test-step-end

[source,go-e2e-step]
----
echo no-group
----

// test-end
`)
			doc, err := p.Parse("test.adoc", content, []string{"go-e2e-step"})
			Expect(err).ToNot(HaveOccurred())
			Expect(doc.Blocks).To(HaveLen(2))
			Expect(doc.Blocks[0].StepGroup).To(Equal("Setup"))
			Expect(doc.Blocks[1].StepGroup).To(BeEmpty())
		})
	})
})
