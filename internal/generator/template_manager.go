package generator

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
)

// TemplateManager handles loading and executing embedded templates
type TemplateManager struct {
	templates map[string]*template.Template
	fs        embed.FS
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(fs embed.FS) *TemplateManager {
	return &TemplateManager{
		templates: make(map[string]*template.Template),
		fs:        fs,
	}
}

// LoadTemplate loads and parses a template from the embedded filesystem
func (tm *TemplateManager) LoadTemplate(name string) (*template.Template, error) {
	// Check cache first
	if tmpl, exists := tm.templates[name]; exists {
		return tmpl, nil
	}

	// Read template file
	content, err := tm.fs.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("failed to read template %s: %w", name, err)
	}

	// Parse template
	tmpl, err := template.New(name).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	// Cache template
	tm.templates[name] = tmpl
	return tmpl, nil
}

// ExecuteTemplate executes a template with given data
func (tm *TemplateManager) ExecuteTemplate(name string, data interface{}) (string, error) {
	tmpl, err := tm.LoadTemplate(name)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return result.String(), nil
}
