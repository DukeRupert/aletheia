package templates

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgtype"
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

	// Define custom template functions
	funcMap := template.FuncMap{
		// dict creates a dictionary/map for passing multiple values to templates
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict requires an even number of arguments")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
		// list creates a slice for passing arrays to templates
		"list": func(values ...interface{}) []interface{} {
			return values
		},
		// slice is an alias for list (for convenience)
		"slice": func(values ...interface{}) []interface{} {
			return values
		},
		// cond returns the first value if condition is true, otherwise the second value
		"cond": func(condition bool, trueVal, falseVal interface{}) interface{} {
			if condition {
				return trueVal
			}
			return falseVal
		},
		"mul": func(a, b interface{}) float64 {
			var aFloat, bFloat float64

			// Handle pgtype.Numeric
			if numeric, ok := a.(pgtype.Numeric); ok {
				floatVal, _ := numeric.Float64Value()
				aFloat = floatVal.Float64
			} else {
				switch v := a.(type) {
				case float64:
					aFloat = v
				case float32:
					aFloat = float64(v)
				case int:
					aFloat = float64(v)
				case int64:
					aFloat = float64(v)
				default:
					return 0
				}
			}

			// Handle pgtype.Numeric
			if numeric, ok := b.(pgtype.Numeric); ok {
				floatVal, _ := numeric.Float64Value()
				bFloat = floatVal.Float64
			} else {
				switch v := b.(type) {
				case float64:
					bFloat = v
				case float32:
					bFloat = float64(v)
				case int:
					bFloat = float64(v)
				case int64:
					bFloat = float64(v)
				default:
					return 0
				}
			}

			return aFloat * bFloat
		},
		// iterate creates a sequence of numbers from 0 to n-1 for range loops
		"iterate": func(n int) []int {
			result := make([]int, n)
			for i := 0; i < n; i++ {
				result[i] = i
			}
			return result
		},
		// sub subtracts two integers
		"sub": func(a, b int) int {
			return a - b
		},
		// add adds two integers
		"add": func(a, b int) int {
			return a + b
		},
		// div divides two integers
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		// toJSON converts a value to JSON for use in Alpine.js or JavaScript
		"toJSON": func(v interface{}) template.JS {
			b, err := json.Marshal(v)
			if err != nil {
				return template.JS("[]")
			}
			return template.JS(b)
		},
	}

	// Get layout and component patterns
	layoutPattern := filepath.Join(templatesDir, "layouts", "*.html")
	componentPattern := filepath.Join(templatesDir, "components", "*.html")

	// Parse base templates (layouts and components) ONCE with custom functions
	baseTmpl, err := template.New("base").Funcs(funcMap).ParseGlob(layoutPattern)
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
