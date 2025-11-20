# Web Templates & Static Assets

This directory contains the frontend templates and static assets for Aletheia.

## Directory Structure

```
web/
├── templates/           # Go html/template files
│   ├── layouts/        # Base layouts (base.html)
│   ├── components/     # Reusable components (nav.html)
│   └── pages/          # Page templates (home.html, 404.html, etc.)
└── static/             # Static assets served at /static
    ├── css/            # Stylesheets (main.css)
    └── images/         # Images and icons
```

## Template System

### Base Layout

All pages extend `layouts/base.html` which provides:
- HTML5 boilerplate
- HTMX and Alpine.js scripts
- Navigation (when authenticated)
- Flash message display
- Modal container for HTMX interactions

### Using Templates

To create a new page:

1. Create a new template file in `templates/pages/`
2. Extend the base layout:

```html
{{template "base" .}}

{{define "title"}}Page Title - Aletheia{{end}}

{{define "content"}}
<div class="container">
  <!-- Your content here -->
</div>
{{end}}
```

3. Render from handler:

```go
func (h *Handler) MyPage(c echo.Context) error {
    data := map[string]interface{}{
        "IsAuthenticated": true,
        "User": user,
    }
    return c.Render(http.StatusOK, "mypage.html", data)
}
```

## Styling

Styles follow a minimal, semantic HTML-first approach. See [STYLE_GUIDE.md](/STYLE_GUIDE.md) for:
- Color palette
- Typography scale
- Component styles
- Utility classes

## HTMX Patterns

HTMX is loaded in the base template and available on all pages.

### Common Patterns

**Inline Edit:**
```html
<div hx-get="/api/items/123/edit" hx-target="this" hx-swap="outerHTML">
  <span>{{.Content}}</span>
  <button>Edit</button>
</div>
```

**Form Submission:**
```html
<form hx-post="/api/items" hx-target="#item-list" hx-swap="afterbegin">
  <input type="text" name="name">
  <button type="submit">Create</button>
</form>
```

**Polling:**
```html
<div hx-get="/api/jobs/{{.JobID}}/status"
     hx-trigger="every 2s"
     hx-swap="outerHTML">
  Processing... {{.Progress}}%
</div>
```

## Alpine.js Patterns

Alpine.js is used for lightweight client-side interactivity.

**Dropdown:**
```html
<div x-data="{ open: false }">
  <button @click="open = !open">Menu</button>
  <div x-show="open" @click.away="open = false">
    <!-- Dropdown content -->
  </div>
</div>
```

**Tabs:**
```html
<div x-data="{ tab: 'details' }">
  <button @click="tab = 'details'">Details</button>
  <button @click="tab = 'photos'">Photos</button>
  <div x-show="tab === 'details'"><!-- Content --></div>
  <div x-show="tab === 'photos'"><!-- Content --></div>
</div>
```

## Development

The template renderer automatically loads all templates from:
- `templates/layouts/*.html`
- `templates/components/*.html`
- `templates/pages/*.html`

Templates are parsed at application startup. Restart the server to see template changes.

Static files are served at `/static/` and are accessible directly:
- `/static/css/main.css`
- `/static/images/logo.png`

## Guidelines

1. **Semantic HTML First** - Use the right HTML element for the job
2. **Minimal CSS** - Only add styles when necessary for clarity
3. **Progressive Enhancement** - Start with HTML, add HTMX, then Alpine.js if needed
4. **Mobile First** - Design for small screens first
5. **Accessibility** - Always include proper labels, ARIA attributes, and keyboard navigation
