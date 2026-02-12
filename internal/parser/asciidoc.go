package parser

import (
	"regexp"
	"strings"

	"github.com/fjglira/GoE2E-DocSyncer/internal/domain"
)

// AsciiDocParser parses AsciiDoc documents using regex patterns.
type AsciiDocParser struct{}

// NewAsciiDocParser creates a new AsciiDocParser.
func NewAsciiDocParser() *AsciiDocParser {
	return &AsciiDocParser{}
}

// SupportedExtensions returns the file extensions this parser handles.
func (p *AsciiDocParser) SupportedExtensions() []string {
	return []string{".adoc", ".asciidoc"}
}

var (
	// Matches [source,tag,attr1="val1",attr2="val2"]
	asciidocSourceRe = regexp.MustCompile(`^\[source,([^,\]]+)(?:,(.+))?\]\s*$`)
	// Matches ---- delimiter
	asciidocDelimRe = regexp.MustCompile(`^----+\s*$`)
	// Matches == Heading, === Subheading, etc.
	asciidocHeadingRe = regexp.MustCompile(`^(={2,6})\s+(.+)$`)
)

// Parse parses an AsciiDoc document and extracts tagged code blocks and headings.
func (p *AsciiDocParser) Parse(filePath string, content []byte, tags []string) (*domain.ParsedDocument, error) {
	lines := strings.Split(string(content), "\n")

	tagSet := make(map[string]bool)
	for _, t := range tags {
		tagSet[t] = true
	}

	parsed := &domain.ParsedDocument{
		FilePath: filePath,
		FileType: "asciidoc",
		Metadata: make(map[string]string),
	}

	var currentHeading string
	var currentTestFile string
	var currentStepGroup string

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Check for test-start / test-end comment markers
		// AsciiDoc single-line comments start with //
		if strings.HasPrefix(trimmed, "// test-start:") {
			name := strings.TrimPrefix(trimmed, "// test-start:")
			name = strings.TrimSpace(name)
			currentTestFile = name
			parsed.Metadata["test-start"] = name
			continue
		} else if strings.HasPrefix(trimmed, "// test-end") {
			currentTestFile = ""
			continue
		} else if strings.HasPrefix(trimmed, "// test-step-start:") {
			name := strings.TrimPrefix(trimmed, "// test-step-start:")
			name = strings.TrimSpace(name)
			currentStepGroup = name
			continue
		} else if strings.HasPrefix(trimmed, "// test-step-end") {
			currentStepGroup = ""
			continue
		}

		// Check for headings
		if m := asciidocHeadingRe.FindStringSubmatch(line); m != nil {
			level := len(m[1]) - 1 // == is level 1, === is level 2
			parsed.Headings = append(parsed.Headings, domain.Heading{
				Level: level,
				Text:  strings.TrimSpace(m[2]),
				Line:  i + 1,
			})
			currentHeading = strings.TrimSpace(m[2])
			continue
		}

		// Check for [source,tag,...] directive
		if m := asciidocSourceRe.FindStringSubmatch(line); m != nil {
			tag := strings.TrimSpace(m[1])
			if !tagSet[tag] {
				continue
			}

			// Parse attributes from the directive
			attrs := make(map[string]string)
			if m[2] != "" {
				attrs = parseAsciidocAttrs(m[2])
			}

			directiveLine := i + 1

			// Expect ---- delimiter on next line
			i++
			if i >= len(lines) {
				break
			}
			if !asciidocDelimRe.MatchString(lines[i]) {
				continue
			}

			// Read content until closing ----
			i++
			var contentLines []string
			contentStartLine := i + 1
			for i < len(lines) && !asciidocDelimRe.MatchString(lines[i]) {
				contentLines = append(contentLines, lines[i])
				i++
			}

			_ = directiveLine // used for error reporting if needed
			block := domain.CodeBlock{
				Tag:        tag,
				Content:    strings.Join(contentLines, "\n"),
				LineNumber: contentStartLine,
				Attributes: attrs,
				Context:    currentHeading,
				TestFile:   currentTestFile,
				StepGroup:  currentStepGroup,
			}
			parsed.Blocks = append(parsed.Blocks, block)
		}
	}

	return parsed, nil
}

// parseAsciidocAttrs parses comma-separated key="value" or key=value attributes.
func parseAsciidocAttrs(s string) map[string]string {
	attrs := make(map[string]string)
	// Split on comma, but respect quotes
	parts := splitAsciidocAttrs(s)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if idx := strings.Index(part, "="); idx > 0 {
			key := strings.TrimSpace(part[:idx])
			val := strings.TrimSpace(part[idx+1:])
			val = strings.Trim(val, "\"'")
			attrs[key] = val
		}
	}
	return attrs
}

// splitAsciidocAttrs splits on commas, respecting quoted values.
func splitAsciidocAttrs(s string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote {
			if c == quoteChar {
				inQuote = false
			}
			current.WriteByte(c)
		} else {
			if c == '"' || c == '\'' {
				inQuote = true
				quoteChar = c
				current.WriteByte(c)
			} else if c == ',' {
				if current.Len() > 0 {
					parts = append(parts, current.String())
					current.Reset()
				}
			} else {
				current.WriteByte(c)
			}
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}
