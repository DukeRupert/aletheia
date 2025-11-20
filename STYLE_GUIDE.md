# Aletheia Style Guide

Minimal, semantic HTML-first styling for MVP development.

## Philosophy

1. **Semantic HTML first** - Use the right HTML element for the job
2. **Minimal custom CSS** - Leverage browser defaults where possible
3. **Functional over decorative** - Clarity trumps aesthetics
4. **Mobile-first** - Design for small screens, enhance for large
5. **Fast to change** - Avoid complex CSS that's hard to modify

## Color Palette

Keep it simple - only 5 colors needed for MVP:

```css
:root {
  /* Neutrals */
  --color-text: #1a1a1a;
  --color-bg: #ffffff;
  --color-border: #e5e5e5;

  /* Functional */
  --color-primary: #2563eb;    /* Blue - primary actions */
  --color-danger: #dc2626;     /* Red - destructive actions, errors */
  --color-success: #16a34a;    /* Green - success states */
  --color-warning: #ea580c;    /* Orange - warnings */

  /* Severity (for violations) */
  --color-critical: #dc2626;   /* Red */
  --color-high: #ea580c;       /* Orange */
  --color-medium: #eab308;     /* Yellow */
  --color-low: #64748b;        /* Gray */
}
```

## Typography

**System Font Stack** - No custom fonts, use what users already have:

```css
body {
  font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI',
               Roboto, sans-serif;
  font-size: 16px;
  line-height: 1.5;
  color: var(--color-text);
}
```

**Type Scale:**
```css
h1 { font-size: 2rem; }      /* 32px */
h2 { font-size: 1.5rem; }    /* 24px */
h3 { font-size: 1.25rem; }   /* 20px */
h4 { font-size: 1rem; }      /* 16px */
p, li { font-size: 1rem; }   /* 16px */
small { font-size: 0.875rem; } /* 14px */
```

## Spacing

Use a simple 8px grid:

```css
:root {
  --space-xs: 0.25rem;  /* 4px */
  --space-sm: 0.5rem;   /* 8px */
  --space-md: 1rem;     /* 16px */
  --space-lg: 1.5rem;   /* 24px */
  --space-xl: 2rem;     /* 32px */
  --space-2xl: 3rem;    /* 48px */
}
```

## Layout

### Container

```html
<div class="container">
  <!-- content -->
</div>
```

```css
.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 var(--space-md);
}
```

### Stack (Vertical Spacing)

```html
<div class="stack">
  <div>Item 1</div>
  <div>Item 2</div>
</div>
```

```css
.stack > * + * {
  margin-top: var(--space-md);
}
```

### Grid (Simple)

```html
<div class="grid">
  <div>Column 1</div>
  <div>Column 2</div>
</div>
```

```css
.grid {
  display: grid;
  gap: var(--space-md);
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
}
```

## Components

### Buttons

Use semantic HTML buttons:

```html
<!-- Primary action -->
<button type="submit" class="btn-primary">Save Changes</button>

<!-- Secondary action -->
<button type="button" class="btn-secondary">Cancel</button>

<!-- Destructive action -->
<button type="button" class="btn-danger">Delete</button>
```

```css
button, .btn {
  padding: var(--space-sm) var(--space-md);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  font-size: 1rem;
  cursor: pointer;
  background: white;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
  border-color: var(--color-primary);
}

.btn-secondary {
  background: white;
  color: var(--color-text);
}

.btn-danger {
  background: var(--color-danger);
  color: white;
  border-color: var(--color-danger);
}

/* Touch-friendly sizing on mobile */
@media (max-width: 768px) {
  button, .btn {
    min-height: 44px;
    min-width: 44px;
  }
}
```

### Forms

Keep forms simple and semantic:

```html
<form class="stack">
  <div class="form-field">
    <label for="email">Email</label>
    <input type="email" id="email" name="email" required>
    <small class="help-text">We'll never share your email.</small>
  </div>

  <div class="form-field">
    <label for="password">Password</label>
    <input type="password" id="password" name="password" required>
  </div>

  <button type="submit" class="btn-primary">Sign In</button>
</form>
```

```css
.form-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

label {
  font-weight: 500;
}

input, textarea, select {
  padding: var(--space-sm);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  font-size: 1rem;
  font-family: inherit;
}

input:focus, textarea:focus, select:focus {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}

.help-text {
  color: #666;
}

.error {
  color: var(--color-danger);
}
```

### Cards

```html
<div class="card">
  <h3>Card Title</h3>
  <p>Card content goes here.</p>
  <button class="btn-primary">Action</button>
</div>
```

```css
.card {
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: var(--space-lg);
  background: white;
}
```

### Tables

Use semantic table elements:

```html
<table>
  <thead>
    <tr>
      <th>Name</th>
      <th>Status</th>
      <th>Actions</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>Inspection #123</td>
      <td><span class="badge badge-success">Completed</span></td>
      <td><a href="/inspections/123">View</a></td>
    </tr>
  </tbody>
</table>
```

```css
table {
  width: 100%;
  border-collapse: collapse;
}

th {
  text-align: left;
  padding: var(--space-sm);
  border-bottom: 2px solid var(--color-border);
  font-weight: 600;
}

td {
  padding: var(--space-sm);
  border-bottom: 1px solid var(--color-border);
}

/* Responsive tables - scroll on mobile */
@media (max-width: 768px) {
  table {
    display: block;
    overflow-x: auto;
  }
}
```

### Badges

```html
<span class="badge badge-critical">Critical</span>
<span class="badge badge-high">High</span>
<span class="badge badge-medium">Medium</span>
<span class="badge badge-low">Low</span>
```

```css
.badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.875rem;
  font-weight: 500;
}

.badge-critical {
  background: #fecaca;
  color: var(--color-critical);
}

.badge-high {
  background: #fed7aa;
  color: var(--color-high);
}

.badge-medium {
  background: #fef3c7;
  color: #92400e;
}

.badge-low {
  background: #e2e8f0;
  color: var(--color-low);
}

.badge-success {
  background: #bbf7d0;
  color: var(--color-success);
}
```

### Navigation

```html
<nav>
  <div class="container">
    <div class="nav-content">
      <a href="/" class="logo">Aletheia</a>
      <ul class="nav-links">
        <li><a href="/dashboard">Dashboard</a></li>
        <li><a href="/inspections">Inspections</a></li>
        <li><a href="/projects">Projects</a></li>
      </ul>
    </div>
  </div>
</nav>
```

```css
nav {
  border-bottom: 1px solid var(--color-border);
  background: white;
}

.nav-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-md) 0;
}

.logo {
  font-weight: 700;
  font-size: 1.25rem;
  text-decoration: none;
  color: var(--color-text);
}

.nav-links {
  display: flex;
  gap: var(--space-lg);
  list-style: none;
  margin: 0;
  padding: 0;
}

.nav-links a {
  text-decoration: none;
  color: var(--color-text);
}

.nav-links a:hover {
  color: var(--color-primary);
}
```

### Modals

```html
<!-- HTMX will swap content into this -->
<div id="modal" class="modal">
  <div class="modal-backdrop"></div>
  <div class="modal-content">
    <!-- Dynamic content -->
  </div>
</div>
```

```css
.modal {
  position: fixed;
  inset: 0;
  display: none;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal.active {
  display: flex;
}

.modal-backdrop {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
}

.modal-content {
  position: relative;
  background: white;
  border-radius: 8px;
  padding: var(--space-xl);
  max-width: 500px;
  width: 90%;
  max-height: 90vh;
  overflow-y: auto;
}
```

## Utility Classes

Keep it minimal - only what's actually needed:

```css
/* Visibility */
.hidden { display: none; }
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border-width: 0;
}

/* Text alignment */
.text-center { text-align: center; }
.text-right { text-align: right; }

/* Flexbox helpers */
.flex { display: flex; }
.flex-col { flex-direction: column; }
.items-center { align-items: center; }
.justify-between { justify-content: space-between; }
.gap-sm { gap: var(--space-sm); }
.gap-md { gap: var(--space-md); }

/* Spacing */
.mt-sm { margin-top: var(--space-sm); }
.mt-md { margin-top: var(--space-md); }
.mt-lg { margin-top: var(--space-lg); }
.mb-sm { margin-bottom: var(--space-sm); }
.mb-md { margin-bottom: var(--space-md); }
.mb-lg { margin-bottom: var(--space-lg); }
```

## Accessibility

### Focus States

Always visible focus indicators:

```css
*:focus {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}

*:focus:not(:focus-visible) {
  outline: none;
}

*:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}
```

### Skip Links

```html
<a href="#main" class="skip-link">Skip to main content</a>
```

```css
.skip-link {
  position: absolute;
  top: -40px;
  left: 0;
  background: var(--color-primary);
  color: white;
  padding: var(--space-sm);
  text-decoration: none;
}

.skip-link:focus {
  top: 0;
}
```

## Icons

Use inline SVG for simplicity (no icon library needed for MVP):

```html
<!-- Example: Close icon -->
<button aria-label="Close">
  <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
    <path d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"/>
  </svg>
</button>
```

## Responsive Design

**Mobile-first approach** - Base styles are for mobile, enhance for desktop:

```css
/* Base (mobile) */
.grid {
  grid-template-columns: 1fr;
}

/* Tablet and up */
@media (min-width: 768px) {
  .grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

/* Desktop */
@media (min-width: 1024px) {
  .grid {
    grid-template-columns: repeat(3, 1fr);
  }
}
```

**Breakpoints:**
- Mobile: < 768px (default)
- Tablet: 768px - 1024px
- Desktop: > 1024px

## Loading States

```html
<!-- Button loading state -->
<button class="btn-primary" disabled>
  <span class="spinner"></span>
  Loading...
</button>
```

```css
.spinner {
  display: inline-block;
  width: 1rem;
  height: 1rem;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
```

## Error States

```html
<div class="alert alert-error">
  <strong>Error:</strong> Unable to upload photo.
</div>

<div class="alert alert-success">
  <strong>Success:</strong> Inspection created.
</div>
```

```css
.alert {
  padding: var(--space-md);
  border-radius: 4px;
  border-left: 4px solid;
}

.alert-error {
  background: #fef2f2;
  border-color: var(--color-danger);
  color: #991b1b;
}

.alert-success {
  background: #f0fdf4;
  border-color: var(--color-success);
  color: #166534;
}

.alert-warning {
  background: #fff7ed;
  border-color: var(--color-warning);
  color: #9a3412;
}
```

## Print Styles

For reports:

```css
@media print {
  nav, button, .no-print {
    display: none !important;
  }

  body {
    font-size: 12pt;
  }

  .page-break {
    page-break-after: always;
  }

  a {
    text-decoration: none;
    color: inherit;
  }
}
```

---

## Quick Reference

**When to add CSS:**
- ✅ Clarity and usability improvements
- ✅ Accessibility (focus states, contrast)
- ✅ Responsive behavior
- ❌ Decorative animations
- ❌ Complex gradients or shadows
- ❌ Pixel-perfect spacing

**Decision Tree:**
1. Can I use semantic HTML? → Yes? Do that.
2. Can I use a simple utility class? → Yes? Do that.
3. Do I need custom CSS? → Keep it minimal and reusable.

**Remember:** You can always add more styling later. For MVP, clarity and speed are more important than aesthetics.
