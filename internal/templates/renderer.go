package templates

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

// TemplateRenderer wraps html/template for Echo
type TemplateRenderer struct {
	templates map[string]*template.Template
}

// NewTemplateRenderer creates a new template renderer
// It loads all templates from the specified directory
func NewTemplateRenderer(templatesDir string) (*TemplateRenderer, error) {
	templates := make(map[string]*template.Template)

	// Get layout and component patterns
	layoutPattern := filepath.Join(templatesDir, "layouts", "*.html")
	componentPattern := filepath.Join(templatesDir, "components", "*.html")

	// Parse base templates (layouts and components) ONCE
	baseTmpl, err := template.New("base").ParseGlob(layoutPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse layouts: %w", err)
	}

	baseTmpl, err = baseTmpl.ParseGlob(componentPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to parse components: %w", err)
	}

	// Get all page files
	pagePattern := filepath.Join(templatesDir, "pages", "*.html")
	pages, err := filepath.Glob(pagePattern)
	if err != nil {
		return nil, err
	}

	// For each page, clone the base template and add the page-specific defines
	for _, page := range pages {
		pageName := filepath.Base(page)

		// Clone the base template to create an isolated template set for this page
		pageTmpl, err := baseTmpl.Clone()
		if err != nil {
			return nil, fmt.Errorf("failed to clone base template for %s: %w", pageName, err)
		}

		// Parse the page file into the cloned template
		pageTmpl, err = pageTmpl.ParseFiles(page)
		if err != nil {
			return nil, fmt.Errorf("failed to parse page %s: %w", page, err)
		}

		// Store the template set with the base filename as key
		templates[pageName] = pageTmpl
	}

	return &TemplateRenderer{
		templates: templates,
	}, nil
}

// Render renders a template with data
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, ok := t.templates[name]
	if !ok {
		return fmt.Errorf("template %s not found", name)
	}
	// Execute the page template directly
	// The page starts with {{template "base" .}} which will use the page's defined blocks
	return tmpl.ExecuteTemplate(w, name, data)
}
