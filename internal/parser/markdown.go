package parser

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	"github.com/frherrer/GoE2E-DocSyncer/internal/domain"
)

// MarkdownParser parses Markdown documents using goldmark.
type MarkdownParser struct{}

// NewMarkdownParser creates a new MarkdownParser.
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{}
}

// SupportedExtensions returns the file extensions this parser handles.
func (p *MarkdownParser) SupportedExtensions() []string {
	return []string{".md", ".markdown"}
}

// Parse parses a Markdown document and extracts tagged code blocks and headings.
func (p *MarkdownParser) Parse(filePath string, content []byte, tags []string) (*domain.ParsedDocument, error) {
	md := goldmark.New()
	reader := text.NewReader(content)
	doc := md.Parser().Parse(reader)

	parsed := &domain.ParsedDocument{
		FilePath: filePath,
		FileType: "markdown",
		Metadata: make(map[string]string),
	}

	// Build a set for quick tag lookup
	tagSet := make(map[string]bool)
	for _, t := range tags {
		tagSet[t] = true
	}

	// Walk the AST to extract headings and code blocks
	var currentHeading string
	var currentTestFile string
	var currentStepGroup string
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Heading:
			headingText := extractText(node, content)
			lineNum := 0
			if node.Lines().Len() > 0 {
				lineNum = lineNumber(content, node.Lines().At(0).Start)
			} else if node.HasChildren() {
				// For ATX headings, use the child text segment position
				if first, ok := node.FirstChild().(*ast.Text); ok {
					lineNum = lineNumber(content, first.Segment.Start)
				}
			}
			parsed.Headings = append(parsed.Headings, domain.Heading{
				Level: node.Level,
				Text:  headingText,
				Line:  lineNum,
			})
			currentHeading = headingText

		case *ast.FencedCodeBlock:
			lang := string(node.Language(content))
			// Parse info string: "tag attr1=val1 attr2=val2"
			var info string
			if node.Info != nil {
				info = string(node.Info.Segment.Value(content))
			}
			parts := parseInfoString(info)
			tag := parts["_tag"]

			if tagSet[tag] || tagSet[lang] {
				// Extract code content
				var buf bytes.Buffer
				lines := node.Lines()
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					buf.Write(line.Value(content))
				}

				// Remove _tag from attributes
				attrs := make(map[string]string)
				for k, v := range parts {
					if k != "_tag" {
						attrs[k] = v
					}
				}

				block := domain.CodeBlock{
					Tag:        tag,
					Content:    strings.TrimRight(buf.String(), "\n"),
					LineNumber: lineNumber(content, node.Lines().At(0).Start),
					Attributes: attrs,
					Context:    currentHeading,
					TestFile:   currentTestFile,
				StepGroup:  currentStepGroup,
				}
				parsed.Blocks = append(parsed.Blocks, block)
			}

		case *ast.HTMLBlock:
			// Check for test-start / test-end / test-step-start / test-step-end comments
			var buf bytes.Buffer
			lines := node.Lines()
			for i := 0; i < lines.Len(); i++ {
				line := lines.At(i)
				buf.Write(line.Value(content))
			}
			htmlText := strings.TrimSpace(buf.String())
			if strings.HasPrefix(htmlText, "<!-- test-start:") {
				// Extract test name from comment
				name := strings.TrimPrefix(htmlText, "<!-- test-start:")
				name = strings.TrimSuffix(name, "-->")
				name = strings.TrimSpace(name)
				currentTestFile = name
				// Keep backward-compatible metadata (stores the last seen test-start)
				parsed.Metadata["test-start"] = name
			} else if strings.HasPrefix(htmlText, "<!-- test-end") {
				currentTestFile = ""
			} else if strings.HasPrefix(htmlText, "<!-- test-step-start:") {
				name := strings.TrimPrefix(htmlText, "<!-- test-step-start:")
				name = strings.TrimSuffix(name, "-->")
				name = strings.TrimSpace(name)
				currentStepGroup = name
			} else if strings.HasPrefix(htmlText, "<!-- test-step-end") {
				currentStepGroup = ""
			}
		}

		return ast.WalkContinue, nil
	})

	if err != nil {
		return nil, domain.NewErrorWithSuggestion("parse", filePath, 0,
			"failed to walk markdown AST",
			"check the markdown file for syntax issues — ensure fenced code blocks use triple backticks",
			err)
	}

	return parsed, nil
}

// parseInfoString parses a fenced code block info string like:
//
//	"go-e2e-step step-name=\"Deploy\" timeout=60s"
//
// Returns map with _tag for the language tag and other key-value pairs.
func parseInfoString(info string) map[string]string {
	result := make(map[string]string)
	info = strings.TrimSpace(info)
	if info == "" {
		return result
	}

	// First token is the language tag
	parts := splitInfoString(info)
	if len(parts) == 0 {
		return result
	}

	result["_tag"] = parts[0]

	// Remaining tokens are key=value pairs
	for _, part := range parts[1:] {
		if idx := strings.Index(part, "="); idx > 0 {
			key := part[:idx]
			val := part[idx+1:]
			// Remove surrounding quotes
			val = strings.Trim(val, "\"'")
			result[key] = val
		}
	}

	return result
}

// splitInfoString splits the info string respecting quoted values.
func splitInfoString(s string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote {
			if c == quoteChar {
				inQuote = false
				current.WriteByte(c)
			} else {
				current.WriteByte(c)
			}
		} else {
			if c == '"' || c == '\'' {
				inQuote = true
				quoteChar = c
				current.WriteByte(c)
			} else if c == ' ' || c == '\t' {
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

// extractText gets the text content of a heading node.
func extractText(n ast.Node, source []byte) string {
	var buf bytes.Buffer
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(*ast.Text); ok {
			buf.Write(t.Segment.Value(source))
		}
	}
	return buf.String()
}

// lineNumber calculates the 1-based line number for a byte offset.
func lineNumber(content []byte, offset int) int {
	return bytes.Count(content[:offset], []byte("\n")) + 1
}
