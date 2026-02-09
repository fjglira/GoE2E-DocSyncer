package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/frherrer/GoE2E-DocSyncer/internal/domain"
)

// PlaintextParser parses generic text files using configurable regex patterns.
type PlaintextParser struct {
	blockStartPattern *regexp.Regexp
	blockEndPattern   *regexp.Regexp
}

// NewPlaintextParser creates a new PlaintextParser with the given regex patterns.
func NewPlaintextParser(blockStart, blockEnd string) (*PlaintextParser, error) {
	startRe, err := regexp.Compile(blockStart)
	if err != nil {
		return nil, fmt.Errorf("invalid block_start pattern: %w", err)
	}
	endRe, err := regexp.Compile(blockEnd)
	if err != nil {
		return nil, fmt.Errorf("invalid block_end pattern: %w", err)
	}
	return &PlaintextParser{
		blockStartPattern: startRe,
		blockEndPattern:   endRe,
	}, nil
}

// SupportedExtensions returns the file extensions this parser handles.
// The plaintext parser acts as a fallback, so it supports common text extensions.
func (p *PlaintextParser) SupportedExtensions() []string {
	return []string{".txt", ".rst", ".rtf"}
}

// Parse parses a plaintext document using regex patterns to find tagged blocks.
func (p *PlaintextParser) Parse(filePath string, content []byte, tags []string) (*domain.ParsedDocument, error) {
	lines := strings.Split(string(content), "\n")

	tagSet := make(map[string]bool)
	for _, t := range tags {
		tagSet[t] = true
	}

	parsed := &domain.ParsedDocument{
		FilePath: filePath,
		FileType: "plaintext",
		Metadata: make(map[string]string),
	}

	// Detect simple headings: lines followed by --- or === underlines
	for i := 0; i < len(lines)-1; i++ {
		line := strings.TrimSpace(lines[i])
		underline := strings.TrimSpace(lines[i+1])
		if line != "" && len(underline) >= 3 {
			if allChar(underline, '=') || allChar(underline, '-') {
				level := 1
				if allChar(underline, '-') {
					level = 2
				}
				parsed.Headings = append(parsed.Headings, domain.Heading{
					Level: level,
					Text:  line,
					Line:  i + 1,
				})
			}
		}
	}

	// Find tagged blocks using regex
	var currentHeading string
	for _, h := range parsed.Headings {
		if currentHeading == "" {
			currentHeading = h.Text
		}
	}

	i := 0
	for i < len(lines) {
		// Update current heading context
		for _, h := range parsed.Headings {
			if h.Line == i+1 {
				currentHeading = h.Text
			}
		}

		m := p.blockStartPattern.FindStringSubmatch(lines[i])
		if m == nil {
			i++
			continue
		}

		tag := m[1]
		if !tagSet[tag] {
			i++
			continue
		}

		// Parse attributes from capture group 2
		attrs := make(map[string]string)
		if len(m) > 2 && m[2] != "" {
			attrs = parsePlaintextAttrs(m[2])
		}

		startLine := i + 1
		i++

		// Collect content until block end
		var contentLines []string
		for i < len(lines) && !p.blockEndPattern.MatchString(lines[i]) {
			contentLines = append(contentLines, lines[i])
			i++
		}

		block := domain.CodeBlock{
			Tag:        tag,
			Content:    strings.Join(contentLines, "\n"),
			LineNumber: startLine + 1, // content starts on next line after @begin
			Attributes: attrs,
			Context:    currentHeading,
		}
		parsed.Blocks = append(parsed.Blocks, block)

		i++ // skip the @end line
	}

	return parsed, nil
}

// parsePlaintextAttrs parses space-separated key=value or key="value" attributes.
func parsePlaintextAttrs(s string) map[string]string {
	attrs := make(map[string]string)
	parts := splitInfoString(s) // reuse from markdown parser
	for _, part := range parts {
		if idx := strings.Index(part, "="); idx > 0 {
			key := part[:idx]
			val := part[idx+1:]
			val = strings.Trim(val, "\"'")
			attrs[key] = val
		}
	}
	return attrs
}

// allChar checks if s consists entirely of character c.
func allChar(s string, c byte) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] != c {
			return false
		}
	}
	return true
}
