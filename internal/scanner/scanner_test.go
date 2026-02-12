package scanner_test

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/fjglira/GoE2E-DocSyncer/internal/scanner"
)

var _ = Describe("Scanner", func() {
	var s *scanner.FileScanner

	BeforeEach(func() {
		s = scanner.NewScanner(true)
	})

	It("should find markdown files in testdata", func() {
		files, err := s.Scan(filepath.Join("..", "..", "testdata", "markdown"), []string{"*.md"}, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(files).To(HaveLen(2))
	})

	It("should find asciidoc files in testdata", func() {
		files, err := s.Scan(filepath.Join("..", "..", "testdata", "asciidoc"), []string{"*.adoc"}, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(files).To(HaveLen(1))
	})

	It("should return sorted file paths", func() {
		files, err := s.Scan(filepath.Join("..", "..", "testdata", "markdown"), []string{"*.md"}, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(files).To(HaveLen(2))
		// Sorted alphabetically
		Expect(filepath.Base(files[0])).To(Equal("multi-step.md"))
		Expect(filepath.Base(files[1])).To(Equal("simple.md"))
	})

	It("should respect exclude patterns", func() {
		files, err := s.Scan(filepath.Join("..", "..", "testdata", "markdown"), []string{"*.md"}, []string{"simple.md"})
		Expect(err).ToNot(HaveOccurred())
		Expect(files).To(HaveLen(1))
		Expect(filepath.Base(files[0])).To(Equal("multi-step.md"))
	})

	It("should handle non-recursive mode", func() {
		s = scanner.NewScanner(false)
		files, err := s.Scan(filepath.Join("..", "..", "testdata"), []string{"*.md", "*.adoc", "*.yaml"}, nil)
		Expect(err).ToNot(HaveOccurred())
		// Non-recursive: only files directly in testdata (none match, all are in subdirs)
		Expect(files).To(BeEmpty())
	})

	It("should return error for nonexistent directory", func() {
		_, err := s.Scan("nonexistent_dir", []string{"*.md"}, nil)
		Expect(err).To(HaveOccurred())
	})
})
