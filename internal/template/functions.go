package template

import (
	"strings"
	"text/template"
)

// CustomFuncMap returns the custom template functions available in templates.
func CustomFuncMap() template.FuncMap {
	return template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"toLower":   strings.ToLower,
		"toUpper":   strings.ToUpper,
		"toTitle":   strings.ToTitle,
		"replace":   strings.ReplaceAll,
		"trimSpace": strings.TrimSpace,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"join":      strings.Join,
		"indent": func(spaces int, s string) string {
			pad := strings.Repeat(" ", spaces)
			lines := strings.Split(s, "\n")
			for i, line := range lines {
				if line != "" {
					lines[i] = pad + line
				}
			}
			return strings.Join(lines, "\n")
		},
		"parseDuration": func(s string) string {
			// This is a helper for templates; actual parsing happens at runtime
			return s
		},
	}
}
