package converter

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/frherrer/GoE2E-DocSyncer/internal/config"
	"github.com/frherrer/GoE2E-DocSyncer/internal/domain"
)

// Converter transforms parsed documents into TestSpec domain models.
type Converter interface {
	Convert(doc *domain.ParsedDocument, tagCfg *config.TagConfig) ([]domain.TestSpec, error)
}

// DefaultConverter implements Converter.
type DefaultConverter struct {
	cmdConfig *config.CommandConfig
}

// NewConverter creates a new DefaultConverter.
func NewConverter(cmdCfg *config.CommandConfig) *DefaultConverter {
	return &DefaultConverter{cmdConfig: cmdCfg}
}

// Convert transforms a ParsedDocument into a slice of TestSpecs.
// Blocks are grouped by their TestGroup field. Each group produces a separate TestSpec.
func (c *DefaultConverter) Convert(doc *domain.ParsedDocument, tagCfg *config.TagConfig) ([]domain.TestSpec, error) {
	if len(doc.Blocks) == 0 {
		return nil, nil
	}

	// Determine describe block from the first heading
	describeBlock := inferDescribeBlock(doc)
	contextBlock := inferContextBlock(doc)

	// Group blocks by TestGroup, maintaining insertion order
	var groupOrder []string
	groupBlocks := make(map[string][]domain.CodeBlock)
	for _, block := range doc.Blocks {
		group := block.TestGroup
		if _, seen := groupBlocks[group]; !seen {
			groupOrder = append(groupOrder, group)
		}
		groupBlocks[group] = append(groupBlocks[group], block)
	}

	// Fallback test name from filename
	base := filepath.Base(doc.FilePath)
	ext := filepath.Ext(base)
	fileTestName := strings.TrimSuffix(base, ext)

	var specs []domain.TestSpec
	for _, group := range groupOrder {
		blocks := groupBlocks[group]

		// Convert blocks to steps
		var steps []domain.TestStep
		for i, block := range blocks {
			// Validate command security
			if err := ValidateCommand(block.Content, c.cmdConfig.BlockedPatterns); err != nil {
				return nil, domain.NewError("convert", doc.FilePath, block.LineNumber, err.Error(), nil)
			}

			step := c.blockToStep(block, i, tagCfg)
			steps = append(steps, step)
		}

		// Determine test name: use group name if set, otherwise fallback to filename
		testName := group
		if testName == "" {
			testName = fileTestName
		}

		spec := domain.TestSpec{
			SourceFile:    doc.FilePath,
			SourceType:    doc.FileType,
			TestName:      testName,
			DescribeBlock: describeBlock,
			ContextBlock:  contextBlock,
			Steps:         steps,
			TemplateName:  "",
		}

		// Check for template override in any block attribute
		if tagCfg.Attributes != nil {
			templateKeys := tagCfg.Attributes["template"]
			for _, block := range blocks {
				for _, key := range templateKeys {
					if val, ok := block.Attributes[key]; ok {
						spec.TemplateName = val
						break
					}
				}
			}
		}

		specs = append(specs, spec)
	}

	return specs, nil
}

// blockToStep converts a single CodeBlock to a TestStep.
func (c *DefaultConverter) blockToStep(block domain.CodeBlock, index int, tagCfg *config.TagConfig) domain.TestStep {
	step := domain.TestStep{
		Command:    block.Content,
		LineNumber: block.LineNumber,
	}

	// Resolve step name from attributes
	step.Name = resolveAttribute(block.Attributes, tagCfg.Attributes["step_name"])
	if step.Name == "" {
		// Auto-generate from command
		step.Name = autoStepName(block.Content, index)
	}

	// Resolve timeout
	timeout := resolveAttribute(block.Attributes, tagCfg.Attributes["timeout"])
	if timeout == "" {
		timeout = c.cmdConfig.DefaultTimeout
	}
	step.Timeout = timeout

	// Resolve expected exit code
	exitCodeStr := resolveAttribute(block.Attributes, tagCfg.Attributes["expected_exit_code"])
	if exitCodeStr != "" {
		if code, err := strconv.Atoi(exitCodeStr); err == nil {
			step.ExpectedExit = code
		}
	} else {
		step.ExpectedExit = c.cmdConfig.DefaultExpectedExitCode
	}

	// Resolve skip on failure
	skipStr := resolveAttribute(block.Attributes, tagCfg.Attributes["skip_on_failure"])
	step.SkipOnFailure = skipStr == "true" || skipStr == "yes"

	// Generate Go code
	step.GoCode = GenerateGoCode(block.Content, step.ExpectedExit, step.Timeout, c.cmdConfig)

	return step
}

// resolveAttribute looks up an attribute value using a list of possible key names.
func resolveAttribute(attrs map[string]string, keys []string) string {
	for _, key := range keys {
		if val, ok := attrs[key]; ok {
			return val
		}
	}
	return ""
}

// autoStepName generates a step name from the command content.
func autoStepName(command string, index int) string {
	lines := strings.Split(strings.TrimSpace(command), "\n")
	if len(lines) == 0 {
		return fmt.Sprintf("Step %d", index+1)
	}
	first := strings.TrimSpace(lines[0])
	parts := strings.Fields(first)
	if len(parts) == 0 {
		return fmt.Sprintf("Step %d", index+1)
	}

	// Use the first command word for naming
	cmd := parts[0]
	switch {
	case cmd == "kubectl" && len(parts) > 1:
		return fmt.Sprintf("kubectl %s", parts[1])
	case cmd == "helm" && len(parts) > 1:
		return fmt.Sprintf("helm %s", parts[1])
	case cmd == "docker" && len(parts) > 1:
		return fmt.Sprintf("docker %s", parts[1])
	case cmd == "curl":
		return "curl request"
	default:
		if len(first) > 50 {
			return first[:50]
		}
		return first
	}
}

// inferDescribeBlock extracts the top-level heading for the Describe block.
func inferDescribeBlock(doc *domain.ParsedDocument) string {
	for _, h := range doc.Headings {
		if h.Level == 1 {
			return h.Text
		}
	}
	// Fallback: use the first heading regardless of level
	if len(doc.Headings) > 0 {
		return doc.Headings[0].Text
	}
	// Last resort: use filename
	return strings.TrimSuffix(filepath.Base(doc.FilePath), filepath.Ext(doc.FilePath))
}

// inferContextBlock extracts a context block from level-2 headings.
func inferContextBlock(doc *domain.ParsedDocument) string {
	for _, h := range doc.Headings {
		if h.Level == 2 {
			return h.Text
		}
	}
	return ""
}
