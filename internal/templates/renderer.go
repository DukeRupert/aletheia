package templates

import (
	"html/template"
	"io"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

// TemplateRenderer wraps html/template for Echo
type TemplateRenderer struct {
	templates *template.Template
}

// NewTemplateRenderer creates a new template renderer
// It loads all templates from the specified directory
func NewTemplateRenderer(templatesDir string) (*TemplateRenderer, error) {
	// Parse all template files
	// We need to parse layouts, components, and pages together
	tmpl := template.New("")

	// Parse layout templates
	layoutPattern := filepath.Join(templatesDir, "layouts", "*.html")
	tmpl, err := tmpl.ParseGlob(layoutPattern)
	if err != nil {
		return nil, err
	}

	// Parse component templates
	componentPattern := filepath.Join(templatesDir, "components", "*.html")
	tmpl, err = tmpl.ParseGlob(componentPattern)
	if err != nil {
		return nil, err
	}

	// Parse page templates
	pagePattern := filepath.Join(templatesDir, "pages", "*.html")
	tmpl, err = tmpl.ParseGlob(pagePattern)
	if err != nil {
		return nil, err
	}

	return &TemplateRenderer{
		templates: tmpl,
	}, nil
}

// Render renders a template with data
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}
