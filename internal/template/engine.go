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
	ListTemplates() []string
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
		return domain.NewError("template", e.templateDir, 0, "failed to read template directory", err)
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
			return domain.NewError("template", path, 0, "failed to parse template", err)
		}

		e.templates[name] = tmpl
	}

	if len(e.templates) == 0 {
		return domain.NewError("template", e.templateDir, 0, "no templates found", nil)
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
		return "", domain.NewError("template", "", 0,
			fmt.Sprintf("template %q not found (available: %s)", tmplName, strings.Join(e.ListTemplates(), ", ")), nil)
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
		return "", domain.NewError("template", spec.SourceFile, 0, "failed to execute template", err)
	}

	// Format with go/format
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Return unformatted if go/format fails (might be useful for debugging)
		return buf.String(), domain.NewError("template", spec.SourceFile, 0,
			"generated code failed go/format validation", err)
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
