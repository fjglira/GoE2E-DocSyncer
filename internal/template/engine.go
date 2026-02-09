package template

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/frherrer/GoE2E-DocSyncer/internal/domain"
)

// TemplateEngine renders TestSpec into Go source code strings.
type TemplateEngine interface {
	Render(spec domain.TestSpec, packageName string) (string, error)
	RenderMulti(specs []domain.TestSpec, packageName string) (string, error)
	ListTemplates() []string
}

// testCase represents a single It() block within a Describe.
type testCase struct {
	TestName string
	Steps    []domain.TestStep
}

// templateData is the struct passed to templates.
type templateData struct {
	PackageName   string
	SourceFile    string
	SourceType    string
	DescribeBlock string
	ContextBlock  string
	TestName      string
	Steps         []domain.TestStep
	Tests         []testCase
	NeedsContext  bool
}

// DefaultEngine implements TemplateEngine.
type DefaultEngine struct {
	templates   map[string]*template.Template
	defaultName string
	templateDir string
}

// NewEngine creates a new template engine, loading templates from the given directory.
func NewEngine(templateDir string, defaultTemplate string) (*DefaultEngine, error) {
	engine := &DefaultEngine{
		templates:   make(map[string]*template.Template),
		defaultName: defaultTemplate,
		templateDir: templateDir,
	}

	if err := engine.loadTemplates(); err != nil {
		return nil, err
	}

	return engine, nil
}

// loadTemplates reads all .tmpl files from the template directory.
func (e *DefaultEngine) loadTemplates() error {
	entries, err := os.ReadDir(e.templateDir)
	if err != nil {
		return domain.NewErrorWithSuggestion("template", e.templateDir, 0,
			"failed to read template directory",
			"ensure the templates directory exists and contains .tmpl files — check templates.directory in docsyncer.yaml",
			err)
	}

	funcMap := CustomFuncMap()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tmpl") {
			continue
		}

		path := filepath.Join(e.templateDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return domain.NewError("template", path, 0, "failed to read template file", err)
		}

		name := strings.TrimSuffix(entry.Name(), ".tmpl")
		tmpl, err := template.New(name).Funcs(funcMap).Parse(string(content))
		if err != nil {
			return domain.NewErrorWithSuggestion("template", path, 0,
				"failed to parse template",
				"check Go template syntax — ensure all {{}} blocks are properly closed and function names are valid",
				err)
		}

		e.templates[name] = tmpl
	}

	if len(e.templates) == 0 {
		return domain.NewErrorWithSuggestion("template", e.templateDir, 0,
			"no templates found",
			"add at least one .tmpl file to the templates directory — see templates/ginkgo_default.tmpl for an example",
			nil)
	}

	return nil
}

// Render renders a TestSpec into a formatted Go source string.
func (e *DefaultEngine) Render(spec domain.TestSpec, packageName string) (string, error) {
	// Select template
	tmplName := e.defaultName
	if spec.TemplateName != "" {
		tmplName = spec.TemplateName
	}

	tmpl, ok := e.templates[tmplName]
	if !ok {
		return "", domain.NewErrorWithSuggestion("template", "", 0,
			fmt.Sprintf("template %q not found (available: %s)", tmplName, strings.Join(e.ListTemplates(), ", ")),
			"check templates.default in docsyncer.yaml or ensure the .tmpl file exists in the templates directory",
			nil)
	}

	// Determine if any step uses context/timeout
	needsContext := false
	for _, step := range spec.Steps {
		if strings.Contains(step.GoCode, "context.WithTimeout") {
			needsContext = true
			break
		}
	}

	data := templateData{
		PackageName:   packageName,
		SourceFile:    spec.SourceFile,
		SourceType:    spec.SourceType,
		DescribeBlock: spec.DescribeBlock,
		ContextBlock:  spec.ContextBlock,
		TestName:      spec.TestName,
		Steps:         spec.Steps,
		NeedsContext:  needsContext,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", domain.NewErrorWithSuggestion("template", spec.SourceFile, 0,
			"failed to execute template",
			"check the template syntax — the template may reference fields that don't exist in the data model",
			err)
	}

	// Format with go/format
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Return unformatted if go/format fails (might be useful for debugging)
		return buf.String(), domain.NewErrorWithSuggestion("template", spec.SourceFile, 0,
			"generated code failed go/format validation",
			"the template may produce invalid Go syntax — check template output with --dry-run --verbose",
			err)
	}

	return string(formatted), nil
}

// RenderMulti renders multiple TestSpecs (from the same source file) into a single
// formatted Go source string with multiple It() blocks inside one Describe().
func (e *DefaultEngine) RenderMulti(specs []domain.TestSpec, packageName string) (string, error) {
	if len(specs) == 0 {
		return "", domain.NewError("template", "", 0, "no specs to render", nil)
	}

	// Use the first spec for shared fields
	first := specs[0]

	// Select template
	tmplName := e.defaultName
	if first.TemplateName != "" {
		tmplName = first.TemplateName
	}

	tmpl, ok := e.templates[tmplName]
	if !ok {
		return "", domain.NewErrorWithSuggestion("template", "", 0,
			fmt.Sprintf("template %q not found (available: %s)", tmplName, strings.Join(e.ListTemplates(), ", ")),
			"check templates.default in docsyncer.yaml or ensure the .tmpl file exists in the templates directory",
			nil)
	}

	// Build test cases and check for context usage
	needsContext := false
	var tests []testCase
	for _, spec := range specs {
		for _, step := range spec.Steps {
			if strings.Contains(step.GoCode, "context.WithTimeout") {
				needsContext = true
			}
		}
		tests = append(tests, testCase{
			TestName: spec.TestName,
			Steps:    spec.Steps,
		})
	}

	data := templateData{
		PackageName:   packageName,
		SourceFile:    first.SourceFile,
		SourceType:    first.SourceType,
		DescribeBlock: first.DescribeBlock,
		ContextBlock:  first.ContextBlock,
		TestName:      first.TestName,
		Steps:         first.Steps,
		Tests:         tests,
		NeedsContext:  needsContext,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", domain.NewErrorWithSuggestion("template", first.SourceFile, 0,
			"failed to execute template",
			"check the template syntax — the template may reference fields that don't exist in the data model",
			err)
	}

	// Format with go/format
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return buf.String(), domain.NewErrorWithSuggestion("template", first.SourceFile, 0,
			"generated code failed go/format validation",
			"the template may produce invalid Go syntax — check template output with --dry-run --verbose",
			err)
	}

	return string(formatted), nil
}

// ListTemplates returns the names of all loaded templates.
func (e *DefaultEngine) ListTemplates() []string {
	names := make([]string, 0, len(e.templates))
	for name := range e.templates {
		names = append(names, name)
	}
	return names
}
