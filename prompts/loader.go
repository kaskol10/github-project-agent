package prompts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Loader loads and renders prompt templates from multiple locations
type Loader struct {
	templates map[string]*template.Template
	basePaths []string // Multiple paths to search for templates
}

// NewLoader creates a new prompt loader with a single base path
func NewLoader(basePath string) (*Loader, error) {
	return NewMultiPathLoader([]string{basePath})
}

// NewMultiPathLoader creates a new prompt loader that searches multiple paths
// Templates are loaded in order, with later paths overriding earlier ones
func NewMultiPathLoader(basePaths []string) (*Loader, error) {
	loader := &Loader{
		templates: make(map[string]*template.Template),
		basePaths: basePaths,
	}

	// Load all templates from all paths
	for _, basePath := range basePaths {
		if basePath == "" {
			continue
		}
		if err := loader.loadTemplatesFromPath(basePath); err != nil {
			// Log error but continue with other paths
			fmt.Printf("Warning: failed to load prompts from %s: %v\n", basePath, err)
		}
	}

	return loader, nil
}

// loadTemplatesFromPath loads all .md files from a specific path
func (l *Loader) loadTemplatesFromPath(basePath string) error {
	// Check if path exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return nil // Path doesn't exist, skip silently
	}

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return fmt.Errorf("failed to read prompts directory %s: %w", basePath, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		// Skip README
		if entry.Name() == "README.md" {
			continue
		}

		filePath := filepath.Join(basePath, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", entry.Name(), err)
		}

		// Extract template name (filename without .md)
		templateName := strings.TrimSuffix(entry.Name(), ".md")

		// Parse template
		tmpl, err := template.New(templateName).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", entry.Name(), err)
		}

		// Later paths override earlier ones (allows customization)
		l.templates[templateName] = tmpl
	}

	return nil
}

// Render renders a template with the given data
func (l *Loader) Render(templateName string, data interface{}) (string, error) {
	tmpl, ok := l.templates[templateName]
	if !ok {
		return "", fmt.Errorf("template %s not found", templateName)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}

// HasTemplate checks if a template exists
func (l *Loader) HasTemplate(templateName string) bool {
	_, ok := l.templates[templateName]
	return ok
}

// ListTemplates returns all available template names
func (l *Loader) ListTemplates() []string {
	var names []string
	for name := range l.templates {
		names = append(names, name)
	}
	return names
}
