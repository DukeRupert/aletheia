# Catalyst UI Kit Migration Guide

A comprehensive guide for refining Aletheia's UI using the Catalyst design system by Tailwind Labs.

## Overview

This guide documents the 5-phase migration from our current minimal CSS approach to the polished Catalyst design language. We'll adapt Catalyst's React components to work with our Go templates + HTMX + Alpine.js stack while maintaining all the visual polish and accessibility features.

**Design Philosophy:**
- Accessibility first (ARIA, focus states, high contrast mode)
- Dark mode native (every component has thoughtful dark mode styling)
- Touch optimized (44px minimum touch targets on mobile)
- Performance focused (CSS over JavaScript where possible)
- Professional polish (subtle shadows, transitions, hover states)

---

## Phase 1: Foundation & Design Tokens ✓ COMPLETED

**Goal**: Establish the core design system foundation that all components will use.

**Status**: ✅ Complete (2025-11-21)
**Implementation**: Tailwind v4 browser CDN with inline `@theme` configuration

### 1.1 Tailwind CSS v4 Migration

**Current State**: Tailwind CSS v3 with traditional config file
**Target State**: Tailwind CSS v4 with CSS variable-based theming

**Tasks:**
- [ ] Update to Tailwind CSS v4
- [ ] Migrate `tailwind.config.js` to `@theme` directive in CSS
- [ ] Set up CSS variable system for theming
- [ ] Configure content paths for Go templates

**Files to Update:**
- `package.json` - Update Tailwind version
- `web/static/css/main.css` - Add `@theme` configuration
- Remove `tailwind.config.js` (replaced by CSS-based config)

**Example `@theme` Configuration:**
```css
@theme {
  --font-family-display: 'Inter', ui-sans-serif, system-ui, sans-serif;
  --font-family-sans: 'Inter', ui-sans-serif, system-ui, sans-serif;

  --spacing-gutter: 1rem;

  --color-zinc-50: oklch(0.985 0 0);
  --color-zinc-100: oklch(0.967 0 0);
  /* ... full zinc scale */

  --color-red-600: oklch(0.577 0.245 27.325);
  /* ... full color palette */
}
```

### 1.2 Typography System

**Current State**: System font stack with basic type scale
**Target State**: Inter font with refined type scale and font features

**Tasks:**
- [ ] Add Inter font (Google Fonts or self-hosted)
- [ ] Enable font-feature-settings: 'cv11' (curved 1)
- [ ] Update base typography styles
- [ ] Create heading, subheading, and text utility classes
- [ ] Set up responsive type scale (base/sm breakpoint)

**Font Loading:**
```html
<!-- In layouts/base.html <head> -->
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
```

**Base Typography Styles:**
```css
body {
  font-family: 'Inter', ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont,
               'Segoe UI', Roboto, sans-serif;
  font-feature-settings: 'cv11';
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

/* Mobile-first with sm: breakpoint for desktop */
.text-base { font-size: 1rem; line-height: 1.5rem; }
@media (min-width: 640px) {
  .text-base { font-size: 0.875rem; line-height: 1.5rem; }
}
```

**Type Scale:**
- Base text: `text-base/6 sm:text-sm/6` (16px → 14px)
- Headings: `text-2xl/8 sm:text-xl/8` (24px → 20px)
- Subheadings: `text-base/7 sm:text-sm/6`
- Small text: `text-sm/5 sm:text-xs/5`
- Font weights: 400 (normal), 500 (medium), 600 (semibold)

### 1.3 Color System

**Current State**: Simple 5-color functional palette
**Target State**: Zinc-based neutral palette with full color spectrum

**Tasks:**
- [ ] Replace current colors with zinc-based neutrals
- [ ] Add full color spectrum (red, orange, amber, yellow, lime, green, emerald, teal, cyan, sky, blue, indigo, violet, purple, fuchsia, pink, rose)
- [ ] Update CSS custom properties for light/dark modes
- [ ] Update existing components to use new color system
- [ ] Create color variant utilities

**Light Mode Colors:**
```css
--color-text-primary: var(--color-zinc-950);
--color-text-secondary: var(--color-zinc-500);
--color-border: color-mix(in srgb, var(--color-zinc-950) 10%, transparent);
--color-bg-surface: var(--color-white);
--color-bg-hover: color-mix(in srgb, var(--color-zinc-950) 2.5%, transparent);
--color-bg-active: color-mix(in srgb, var(--color-zinc-950) 5%, transparent);
```

**Dark Mode Colors:**
```css
@media (prefers-color-scheme: dark) {
  --color-text-primary: var(--color-white);
  --color-text-secondary: var(--color-zinc-400);
  --color-border: color-mix(in srgb, var(--color-white) 10%, transparent);
  --color-bg-surface: var(--color-zinc-900);
  --color-bg-hover: color-mix(in srgb, var(--color-white) 5%, transparent);
  --color-bg-active: color-mix(in srgb, var(--color-white) 10%, transparent);
}
```

**Status Colors:**
- Error: `text-red-600 dark:text-red-500`
- Success: `text-green-600 dark:text-green-400`
- Warning: `text-amber-700 dark:text-amber-400`
- Info: `text-blue-600 dark:text-blue-400`

### 1.4 Spacing & Layout

**Current State**: Simple 8px spacing grid
**Target State**: Refined spacing with gutter system

**Tasks:**
- [ ] Adopt Catalyst spacing patterns
- [ ] Implement gutter system for page margins
- [ ] Update container max-widths
- [ ] Refine gap utilities for common layouts

**Spacing Patterns:**
- Touch targets: `px-[calc(theme(spacing.3.5)-1px)] py-[calc(theme(spacing.2.5)-1px)]` (mobile)
- Touch targets: `sm:px-[calc(theme(spacing.3)-1px)] sm:py-[calc(theme(spacing.1.5)-1px)]` (desktop)
- Gutter: `px-[--gutter,theme(spacing.2)]` for responsive page margins
- Common gaps: `gap-3`, `gap-4`, `gap-8` (12px, 16px, 32px)

### 1.5 Border Radius

**Current State**: Simple 4px/8px radius
**Target State**: Refined radius scale for different component types

**Tasks:**
- [ ] Update border radius utilities
- [ ] Apply consistent radius to components by type

**Border Radius Scale:**
- Buttons/Inputs: `rounded-lg` (8px)
- Cards/Dialogs: `rounded-2xl` (16px), `rounded-3xl` (24px)
- Badges: `rounded-md` (6px)
- Avatars: `rounded-full` (circle) or `rounded-[20%]` (squircle)

### 1.6 Shadows & Depth

**Current State**: Minimal use of shadows
**Target State**: Subtle shadow system for visual hierarchy

**Tasks:**
- [ ] Implement ring-based borders: `ring-1 ring-zinc-950/10`
- [ ] Add button shadows using pseudo-elements
- [ ] Create card elevation patterns
- [ ] Add inset shadows for inputs

**Shadow Patterns:**
```css
/* Button shadow (light mode only) */
.btn::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background: var(--btn-bg);
  box-shadow: 0 1px 1px rgba(0, 0, 0, 0.05);
  display: none;
}
@media not (prefers-color-scheme: dark) {
  .btn::before {
    display: block;
  }
}

/* Ring borders instead of traditional borders */
.ring-border {
  @apply ring-1 ring-zinc-950/10 dark:ring-white/10;
}
```

### 1.7 Responsive Breakpoints

**Current State**: Mobile (< 768px), Tablet (768-1024px), Desktop (> 1024px)
**Target State**: Mobile-first with primary `sm:` breakpoint (640px)

**Tasks:**
- [ ] Refactor responsive utilities to use `sm:` as primary breakpoint
- [ ] Update component styles for mobile-first approach
- [ ] Ensure 44px touch targets on mobile with `pointer-fine:` variant

**Breakpoint Strategy:**
- Default: Mobile styles (< 640px)
- `sm:`: Desktop/tablet styles (≥ 640px)
- `pointer-fine:`: Mouse/trackpad users (hide large touch targets)

### 1.8 Dark Mode

**Current State**: No dark mode support
**Target State**: Native dark mode with thoughtful color choices

**Tasks:**
- [ ] Enable dark mode in Tailwind config: `darkMode: 'media'`
- [ ] Add dark mode variants to all components
- [ ] Test all pages in dark mode
- [ ] Ensure sufficient contrast in both modes

**Dark Mode Pattern:**
```html
<!-- Every component needs dark mode consideration -->
<button class="bg-zinc-900 text-white
               dark:bg-white dark:text-zinc-950">
  Button
</button>

<input class="border-zinc-950/10 bg-transparent
              dark:border-white/10 dark:bg-white/5">
```

### 1.9 Accessibility Foundation

**Current State**: Basic focus states
**Target State**: Comprehensive accessibility features

**Tasks:**
- [ ] Implement `data-*` attributes for state styling
- [ ] Add forced colors mode support for Windows High Contrast
- [ ] Enhance focus indicators
- [ ] Ensure touch target minimums
- [ ] Add skip links and landmark regions

**Data Attributes for State:**
```html
<!-- Use data attributes instead of :hover/:focus pseudo-classes -->
<button class="data-hover:bg-zinc-950/5
               data-focus:outline data-focus:outline-2
               data-active:bg-zinc-950/10
               data-disabled:opacity-50">
  Button
</button>
```

**Forced Colors Mode:**
```css
/* Windows High Contrast support */
.btn {
  @apply forced-colors:outline forced-colors:[--btn-icon:ButtonText];
  @apply forced-colors:data-focus:bg-[Highlight] forced-colors:data-focus:text-[HighlightText];
}
```

### Phase 1 Success Criteria

- [x] Tailwind CSS v4 installed and configured (via CDN)
- [x] Inter font loaded with font features enabled
- [x] Zinc color palette implemented with dark mode
- [x] Spacing and typography scales updated (gutter system)
- [x] Border radius and shadow patterns established (Tailwind defaults + theme vars)
- [x] All design tokens accessible via CSS variables
- [x] Dark mode working across entire app (media query based)
- [x] No visual regressions in existing UI (old CSS backed up)

**Actual Time**: ~1 hour
**Files Modified**:
- `web/templates/layouts/base.html` - Added Tailwind v4 CDN, Inter font, @theme configuration
- `web/static/css/main.css.backup` - Old CSS preserved as backup

---

## Phase 2: Core Components - NEXT UP

**Goal**: Build the fundamental UI components that will be used throughout the application.

**Status**: 🔜 Ready to start

### 2.1 Button Component

**Current State**: Simple button styles with 3 variants
**Target State**: Sophisticated button with multiple styles, colors, and states

**Tasks:**
- [ ] Create button base component template
- [ ] Implement solid, outline, and plain variants
- [ ] Add color support (all accent colors)
- [ ] Add size variants (sm, md, lg)
- [ ] Implement sophisticated hover/active states using pseudo-elements
- [ ] Add loading state with spinner
- [ ] Ensure 44px touch targets on mobile

**Go Template Structure:**
```go
{{define "button"}}
<button type="{{.Type | default "button"}}"
        class="{{template "button-classes" .}}"
        {{if .Disabled}}disabled{{end}}
        {{if .HXPost}}hx-post="{{.HXPost}}"{{end}}
        {{if .HXGet}}hx-get="{{.HXGet}}"{{end}}>
  {{if .Loading}}
    {{template "spinner" .}}
  {{end}}
  {{.Content}}
</button>
{{end}}

{{define "button-classes"}}
relative isolate inline-flex items-baseline justify-center gap-x-2
rounded-lg text-base/6 font-semibold
px-[calc(theme(spacing.3.5)-1px)] py-[calc(theme(spacing.2.5)-1px)]
sm:px-[calc(theme(spacing.3)-1px)] sm:py-[calc(theme(spacing.1.5)-1px)]
sm:text-sm/6
{{if eq .Variant "solid"}}
  {{template "button-solid" .}}
{{else if eq .Variant "outline"}}
  {{template "button-outline" .}}
{{else if eq .Variant "plain"}}
  {{template "button-plain" .}}
{{end}}
{{end}}
```

**Button Variants:**

1. **Solid** (default):
```css
/* Base styles */
relative isolate inline-flex items-baseline justify-center gap-x-2
rounded-lg border text-base/6 font-semibold
px-[calc(theme(spacing.3.5)-1px)] py-[calc(theme(spacing.2.5)-1px)]

/* Dark button (default) */
border-transparent bg-[--btn-border]
dark:bg-[--btn-bg]
before:absolute before:inset-0 before:rounded-[calc(theme(borderRadius.lg)-1px)]
before:bg-[--btn-bg] before:shadow dark:before:hidden
after:absolute after:inset-0 after:rounded-[calc(theme(borderRadius.lg)-1px)]
after:shadow-[shadow:inset_0_1px_theme(colors.white/15%)]

data-hover:before:bg-[--btn-hover-overlay]
data-active:after:bg-[--btn-active-overlay]
```

2. **Outline**:
```css
border-zinc-950/10 text-zinc-950
data-hover:border-zinc-950/20 data-active:bg-zinc-950/[2.5%]
dark:border-white/15 dark:text-white
dark:data-hover:border-white/25 dark:data-active:bg-white/5
```

3. **Plain** (ghost):
```css
border-transparent text-zinc-950
data-hover:bg-zinc-950/5 data-active:bg-zinc-950/10
dark:text-white dark:data-hover:bg-white/5 dark:data-active:bg-white/10
```

**Color Support:**
```go
{{if .Color}}
  {{/* Generate color-specific CSS variables */}}
  [--btn-bg:theme(colors.{{.Color}}.600)]
  [--btn-border:theme(colors.{{.Color}}.700)]
  [--btn-hover-overlay:theme(colors.white/10%)]
{{end}}
```

**Files to Create/Update:**
- `web/templates/components/button.html`
- Update all existing buttons to use new component

### 2.2 Input Components

**Current State**: Basic input styling
**Target State**: Polished inputs with sophisticated focus states

**Components to Build:**
- Text input
- Textarea
- Select dropdown
- Checkbox
- Radio button
- Switch (Alpine.js)

**Tasks:**
- [ ] Create Field wrapper component (label + input + description + error)
- [ ] Build text input with focus states
- [ ] Create textarea component
- [ ] Build select component (native + custom styled)
- [ ] Create checkbox with visual indicator
- [ ] Build radio button component
- [ ] Implement switch with Alpine.js
- [ ] Add validation error states
- [ ] Add disabled states

**Field Wrapper Template:**
```go
{{define "field"}}
<div class="space-y-1">
  {{if .Label}}
    <label for="{{.ID}}" class="text-base/6 text-zinc-950 data-disabled:opacity-50
                                 sm:text-sm/6 dark:text-white font-medium">
      {{.Label}}
      {{if .Required}}<span class="text-red-600">*</span>{{end}}
    </label>
  {{end}}

  {{if .Description}}
    <p class="text-base/6 text-zinc-500 data-disabled:opacity-50
              sm:text-sm/6 dark:text-zinc-400">
      {{.Description}}
    </p>
  {{end}}

  {{.Input}}

  {{if .Error}}
    <p class="text-base/6 text-red-600 data-disabled:opacity-50
              sm:text-sm/6 dark:text-red-500">
      {{.Error}}
    </p>
  {{end}}
</div>
{{end}}
```

**Input Base Classes:**
```css
relative block w-full appearance-none rounded-lg
px-[calc(theme(spacing.3.5)-1px)] py-[calc(theme(spacing.2.5)-1px)]
sm:px-[calc(theme(spacing.3)-1px)] sm:py-[calc(theme(spacing.1.5)-1px)]
text-base/6 text-zinc-950 placeholder:text-zinc-500 sm:text-sm/6
border border-zinc-950/10 data-hover:border-zinc-950/20
dark:border-white/10 dark:data-hover:border-white/20
bg-transparent dark:bg-white/5
focus:outline focus:outline-2 focus:-outline-offset-1 focus:outline-blue-500
invalid:border-red-500 invalid:data-hover:border-red-500
disabled:border-zinc-950/20 disabled:opacity-50 dark:disabled:border-white/15
dark:disabled:bg-white/[2.5%] dark:disabled:opacity-50
```

**Checkbox Template:**
```html
<label class="flex items-center gap-3 cursor-pointer group">
  <input type="checkbox"
         name="{{.Name}}"
         value="{{.Value}}"
         {{if .Checked}}checked{{end}}
         class="size-4 rounded border-zinc-950/15 text-blue-600
                focus:ring-2 focus:ring-blue-500 focus:ring-offset-2
                dark:border-white/15 dark:bg-white/5">
  <span class="text-base/6 text-zinc-950 sm:text-sm/6 dark:text-white">
    {{.Label}}
  </span>
</label>
```

**Switch Component (Alpine.js):**
```html
<button type="button"
        role="switch"
        x-data="{ checked: {{.Checked}} }"
        @click="checked = !checked"
        :aria-checked="checked.toString()"
        :class="checked ? 'bg-blue-600' : 'bg-zinc-200 dark:bg-zinc-700'"
        class="relative inline-flex h-6 w-11 items-center rounded-full
               transition-colors focus:outline focus:outline-2
               focus:outline-offset-2 focus:outline-blue-500">
  <span :class="checked ? 'translate-x-6' : 'translate-x-1'"
        class="inline-block size-4 transform rounded-full bg-white
               transition-transform"></span>
</button>
```

**Files to Create/Update:**
- `web/templates/components/field.html`
- `web/templates/components/input.html`
- `web/templates/components/textarea.html`
- `web/templates/components/select.html`
- `web/templates/components/checkbox.html`
- `web/templates/components/radio.html`
- `web/templates/components/switch.html`
- Update all forms to use new field components

### 2.3 Badge Component

**Current State**: Simple badge with severity colors
**Target State**: Polished badge with color variants and proper typography

**Tasks:**
- [ ] Create badge component template
- [ ] Add color variants (all accent colors)
- [ ] Add size variants (sm, md)
- [ ] Update severity badges (critical, high, medium, low)
- [ ] Update status badges (pending, confirmed, dismissed)

**Badge Template:**
```go
{{define "badge"}}
<span class="{{template "badge-classes" .}}">
  {{if .Icon}}
    <svg class="size-4" fill="currentColor">{{.Icon}}</svg>
  {{end}}
  {{.Content}}
</span>
{{end}}

{{define "badge-classes"}}
inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5
text-sm/5 font-medium sm:text-xs/5 forced-colors:outline
{{if eq .Color "zinc"}}
  bg-zinc-600/10 text-zinc-700 dark:bg-white/5 dark:text-zinc-400
{{else if eq .Color "red"}}
  bg-red-600/10 text-red-700 dark:bg-red-400/10 dark:text-red-400
{{else if eq .Color "orange"}}
  bg-orange-600/10 text-orange-700 dark:bg-orange-400/10 dark:text-orange-400
{{else if eq .Color "amber"}}
  bg-amber-400/20 text-amber-700 dark:bg-amber-400/10 dark:text-amber-400
{{else if eq .Color "green"}}
  bg-green-600/10 text-green-700 dark:bg-green-400/10 dark:text-green-400
{{else if eq .Color "blue"}}
  bg-blue-600/10 text-blue-700 dark:bg-blue-400/10 dark:text-blue-400
{{end}}
{{end}}
```

**Severity Badge Mapping:**
- Critical: `red`
- High: `orange`
- Medium: `amber`
- Low: `zinc`

**Status Badge Mapping:**
- Pending: `blue`
- Confirmed: `green`
- Dismissed: `zinc`

**Files to Create/Update:**
- `web/templates/components/badge.html`
- Update violation cards to use new badges
- Update inspection status badges

### 2.4 Typography Components

**Current State**: Raw HTML elements
**Target State**: Semantic components with consistent styling

**Components to Build:**
- Heading (h1-h6)
- Subheading
- Text (body, small, code, strong, link)
- Description list

**Tasks:**
- [ ] Create heading component with levels
- [ ] Create subheading component
- [ ] Create text component with variants
- [ ] Create description list component
- [ ] Update all page headings to use components

**Heading Template:**
```go
{{define "heading"}}
{{$level := .Level | default "1"}}
<h{{$level}} class="text-2xl/8 font-semibold text-zinc-950
                    sm:text-xl/8 dark:text-white">
  {{.Content}}
</h{{$level}}>
{{end}}

{{define "subheading"}}
<h2 class="text-base/7 font-semibold text-zinc-950
           sm:text-sm/6 dark:text-white">
  {{.Content}}
</h2>
{{end}}
```

**Text Variants:**
```html
<!-- Body text -->
<p class="text-base/6 text-zinc-500 sm:text-sm/6 dark:text-zinc-400">
  {{.Content}}
</p>

<!-- Strong/emphasized -->
<strong class="font-medium text-zinc-950 dark:text-white">
  {{.Content}}
</strong>

<!-- Code -->
<code class="rounded border border-zinc-950/10 bg-zinc-950/[2.5%]
             px-0.5 text-sm font-medium text-zinc-950
             dark:border-white/20 dark:bg-white/5 dark:text-white
             sm:text-xs">
  {{.Content}}
</code>

<!-- Link -->
<a href="{{.Href}}"
   class="text-zinc-950 underline decoration-zinc-950/50
          data-hover:decoration-zinc-950 dark:text-white
          dark:decoration-white/50 dark:data-hover:decoration-white">
  {{.Content}}
</a>
```

**Files to Create/Update:**
- `web/templates/components/heading.html`
- `web/templates/components/text.html`
- Update all pages to use semantic typography components

### 2.5 Divider Component

**Current State**: Simple `<hr>` elements
**Target State**: Polished divider with text support

**Tasks:**
- [ ] Create divider component
- [ ] Add variant with centered text
- [ ] Update page layouts to use dividers

**Divider Template:**
```go
{{define "divider"}}
{{if .Text}}
  <div class="relative">
    <div class="absolute inset-0 flex items-center" aria-hidden="true">
      <div class="w-full border-t border-zinc-950/10 dark:border-white/10"></div>
    </div>
    <div class="relative flex justify-center text-sm/6">
      <span class="bg-white px-6 text-zinc-500 dark:bg-zinc-900 dark:text-zinc-400">
        {{.Text}}
      </span>
    </div>
  </div>
{{else}}
  <hr class="w-full border-t border-zinc-950/10 dark:border-white/10">
{{end}}
{{end}}
```

**Files to Create/Update:**
- `web/templates/components/divider.html`

### 2.6 Avatar Component

**Current State**: No avatar component
**Target State**: Avatar with size variants and fallback initials

**Tasks:**
- [ ] Create avatar component
- [ ] Add size variants (xs, sm, md, lg, xl)
- [ ] Support image or initials
- [ ] Add square/circle variants

**Avatar Template:**
```go
{{define "avatar"}}
<span class="{{template "avatar-classes" .}}">
  {{if .ImageURL}}
    <img src="{{.ImageURL}}" alt="{{.Alt}}" class="size-full">
  {{else}}
    <span class="{{template "avatar-initials-classes" .}}">
      {{.Initials}}
    </span>
  {{end}}
</span>
{{end}}

{{define "avatar-classes"}}
inline-grid shrink-0 align-middle
{{if eq .Shape "square"}}
  rounded-[20%]
{{else}}
  rounded-full
{{end}}
{{if eq .Size "xs"}}
  size-6
{{else if eq .Size "sm"}}
  size-8
{{else if eq .Size "lg"}}
  size-12
{{else if eq .Size "xl"}}
  size-16
{{else}}
  size-10
{{end}}
forced-colors:outline
{{end}}

{{define "avatar-initials-classes"}}
flex items-center justify-center
bg-zinc-900 text-white dark:bg-white dark:text-zinc-900
text-[calc(var(--avatar-size)/2.5)] font-medium uppercase
{{end}}
```

**Files to Create/Update:**
- `web/templates/components/avatar.html`
- Update navigation to show user avatar
- Consider adding to violation cards (inspector avatar)

### Phase 2 Success Criteria

- [ ] Button component with all variants working
- [ ] All form input components created and functional
- [ ] Badge component with color variants
- [ ] Typography components implemented
- [ ] Divider and avatar components ready
- [ ] All components have dark mode support
- [ ] All components are accessible (ARIA, focus states)
- [ ] Components work with HTMX and Alpine.js
- [ ] Documentation for each component

**Estimated Time**: 2-3 days

---

## Phase 3: Layout & Navigation

**Goal**: Refine the overall page layout, navigation, and container structures.

### 3.1 Sidebar Layout

**Current State**: No sidebar navigation
**Target State**: Optional sidebar layout for main app sections

**Tasks:**
- [ ] Create sidebar layout component
- [ ] Implement mobile hamburger menu
- [ ] Add smooth transitions for sidebar toggle
- [ ] Create navigation item components
- [ ] Add active state indicators
- [ ] Integrate with existing navigation

**Sidebar Layout Structure:**
```go
{{define "sidebar-layout"}}
<div class="relative isolate flex min-h-screen w-full bg-white dark:bg-zinc-900
            lg:bg-zinc-100 lg:dark:bg-zinc-950">

  <!-- Sidebar -->
  <div x-data="{ open: false }" class="fixed inset-y-0 left-0 z-50 w-64">
    <!-- Mobile overlay -->
    <div x-show="open"
         x-transition.opacity
         @click="open = false"
         class="fixed inset-0 bg-zinc-950/50 lg:hidden"></div>

    <!-- Sidebar content -->
    <nav :class="open ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'"
         class="flex h-full flex-col gap-y-5 bg-white px-6 pb-4
                dark:bg-zinc-900 transition-transform duration-200">
      <!-- Logo -->
      <div class="flex h-16 shrink-0 items-center">
        <a href="/dashboard">
          <span class="text-xl font-bold text-zinc-950 dark:text-white">
            Aletheia
          </span>
        </a>
      </div>

      <!-- Navigation items -->
      <nav class="flex flex-1 flex-col">
        {{range .NavItems}}
          {{template "sidebar-nav-item" .}}
        {{end}}
      </nav>

      <!-- User profile at bottom -->
      {{template "sidebar-user-profile" .User}}
    </nav>
  </div>

  <!-- Main content -->
  <main class="flex flex-1 flex-col lg:pl-64">
    {{.Content}}
  </main>
</div>
{{end}}
```

**Navigation Item:**
```go
{{define "sidebar-nav-item"}}
<a href="{{.Href}}"
   class="group flex gap-x-3 rounded-lg p-2 text-sm leading-6 font-semibold
          {{if .Current}}
            bg-zinc-950/5 text-zinc-950 dark:bg-white/5 dark:text-white
          {{else}}
            text-zinc-700 hover:bg-zinc-950/5 hover:text-zinc-950
            dark:text-zinc-400 dark:hover:bg-white/5 dark:hover:text-white
          {{end}}">
  {{if .Icon}}
    <svg class="size-6 shrink-0 {{if .Current}}text-zinc-950 dark:text-white
                                 {{else}}text-zinc-400 group-hover:text-zinc-950
                                 dark:text-zinc-500 dark:group-hover:text-white{{end}}"
         fill="currentColor">
      {{.Icon}}
    </svg>
  {{end}}
  {{.Label}}
</a>
{{end}}
```

**Files to Create/Update:**
- `web/templates/layouts/sidebar-layout.html`
- `web/templates/components/sidebar-nav-item.html`
- Update main pages to use sidebar layout

### 3.2 Top Navigation Refinement

**Current State**: Basic horizontal navigation
**Target State**: Polished navigation with better active states and dropdowns

**Tasks:**
- [ ] Refine navigation bar styling
- [ ] Add better active state indicators
- [ ] Implement user profile dropdown (Alpine.js)
- [ ] Add mobile responsive menu
- [ ] Improve logout button styling

**Navigation Bar:**
```html
<nav class="flex items-center justify-between gap-4 border-b border-zinc-950/10
            bg-white px-6 py-4 dark:border-white/10 dark:bg-zinc-900">
  <!-- Logo -->
  <div class="flex items-center gap-8">
    <a href="/dashboard" class="text-xl font-bold text-zinc-950 dark:text-white">
      Aletheia
    </a>

    <!-- Desktop navigation -->
    <div class="hidden lg:flex lg:gap-6">
      {{range .NavItems}}
        <a href="{{.Href}}"
           class="relative px-3 py-2 text-sm/6 font-medium transition
                  {{if .Current}}
                    text-zinc-950 dark:text-white
                  {{else}}
                    text-zinc-500 hover:text-zinc-950
                    dark:text-zinc-400 dark:hover:text-white
                  {{end}}">
          {{.Label}}
          {{if .Current}}
            <span class="absolute inset-x-0 -bottom-4 h-0.5 bg-zinc-950
                         dark:bg-white"></span>
          {{end}}
        </a>
      {{end}}
    </div>
  </div>

  <!-- User menu -->
  <div x-data="{ open: false }" class="relative">
    <button @click="open = !open"
            class="flex items-center gap-3 rounded-lg p-2
                   hover:bg-zinc-950/5 dark:hover:bg-white/5">
      {{template "avatar" .User}}
      <span class="hidden text-sm font-medium text-zinc-950 dark:text-white lg:block">
        {{.User.DisplayName}}
      </span>
    </button>

    <!-- Dropdown menu -->
    <div x-show="open"
         x-transition
         @click.away="open = false"
         class="absolute right-0 mt-2 w-56 rounded-xl bg-white p-1 shadow-lg
                ring-1 ring-zinc-950/10 dark:bg-zinc-900 dark:ring-white/10">
      {{template "dropdown-menu" .UserMenuItems}}
    </div>
  </div>
</nav>
```

**Files to Create/Update:**
- `web/templates/components/nav.html`
- Update navigation active state logic

### 3.3 Page Container & Gutter System

**Current State**: Simple container with max-width
**Target State**: Sophisticated gutter system for consistent page margins

**Tasks:**
- [ ] Implement gutter system using CSS variables
- [ ] Create page header component
- [ ] Create page content wrapper
- [ ] Add breadcrumb navigation
- [ ] Ensure responsive behavior

**Page Structure:**
```html
<div class="mx-auto max-w-7xl">
  <!-- Page header -->
  <header class="px-[--gutter,theme(spacing.6)] py-8 sm:py-12">
    {{template "heading" .Title}}

    {{if .Description}}
      <p class="mt-2 text-base/7 text-zinc-600 dark:text-zinc-400">
        {{.Description}}
      </p>
    {{end}}

    {{if .Actions}}
      <div class="mt-6 flex gap-4">
        {{range .Actions}}
          {{template "button" .}}
        {{end}}
      </div>
    {{end}}
  </header>

  <!-- Page content -->
  <main class="px-[--gutter,theme(spacing.6)] pb-16">
    {{.Content}}
  </main>
</div>
```

**Gutter CSS Variables:**
```css
@media (min-width: 640px) {
  :root {
    --gutter: theme(spacing.8);
  }
}

@media (min-width: 1024px) {
  :root {
    --gutter: theme(spacing.10);
  }
}
```

**Files to Create/Update:**
- `web/templates/layouts/page-container.html`
- `web/static/css/main.css` - Add gutter variables
- Update all page templates to use new structure

### 3.4 Grid Layouts

**Current State**: Simple auto-fit grid
**Target State**: Refined grid with consistent gaps and responsive behavior

**Tasks:**
- [ ] Create reusable grid component
- [ ] Add variant for card grids (organizations, projects)
- [ ] Add variant for photo gallery
- [ ] Add variant for violation cards
- [ ] Implement proper gap handling

**Grid Component:**
```go
{{define "grid"}}
<div class="grid gap-8
            {{if eq .Cols "1"}}
              grid-cols-1
            {{else if eq .Cols "2"}}
              grid-cols-1 sm:grid-cols-2
            {{else if eq .Cols "3"}}
              grid-cols-1 sm:grid-cols-2 lg:grid-cols-3
            {{else if eq .Cols "4"}}
              grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4
            {{else}}
              grid-cols-1 sm:grid-cols-2 lg:grid-cols-3
            {{end}}">
  {{.Content}}
</div>
{{end}}
```

**Files to Create/Update:**
- `web/templates/components/grid.html`
- Update organization, project, and photo gallery layouts

### 3.5 Stacked Layout (Vertical Lists)

**Current State**: Manual spacing between elements
**Target State**: Consistent vertical rhythm using stack component

**Tasks:**
- [ ] Create stack component for vertical spacing
- [ ] Add size variants (sm, md, lg)
- [ ] Add dividers option
- [ ] Use in forms and detail pages

**Stack Component:**
```go
{{define "stack"}}
<div class="space-y-{{.Gap | default "6"}}
            {{if .Dividers}}divide-y divide-zinc-950/10 dark:divide-white/10{{end}}">
  {{.Content}}
</div>
{{end}}
```

**Files to Create/Update:**
- `web/templates/components/stack.html`
- Update forms and detail pages to use stack

### Phase 3 Success Criteria

- [ ] Sidebar layout implemented (optional)
- [ ] Navigation bar refined with dropdown
- [ ] Page container and gutter system working
- [ ] Grid and stack layouts available
- [ ] All layouts responsive across breakpoints
- [ ] Mobile menu functional with Alpine.js
- [ ] Active navigation states working correctly
- [ ] Consistent spacing throughout the app

**Estimated Time**: 2-3 days

---

## Phase 4: Interactive Components

**Goal**: Build sophisticated interactive components using Alpine.js and HTMX.

### 4.1 Dropdown Component

**Current State**: No dropdown menus
**Target State**: Polished dropdown with proper positioning and transitions

**Tasks:**
- [ ] Create dropdown wrapper (Alpine.js)
- [ ] Build dropdown button
- [ ] Create dropdown menu with items
- [ ] Add divider support
- [ ] Implement proper positioning (anchoring)
- [ ] Add keyboard navigation
- [ ] Add smooth transitions

**Dropdown Structure:**
```html
<div x-data="{ open: false }" class="relative">
  <!-- Trigger button -->
  <button @click="open = !open"
          type="button"
          class="inline-flex items-center gap-2 rounded-lg border border-zinc-950/10
                 px-3 py-2 text-sm font-semibold text-zinc-950
                 hover:border-zinc-950/20 dark:border-white/10
                 dark:text-white dark:hover:border-white/20">
    <span>{{.Label}}</span>
    <svg class="size-4" :class="open && 'rotate-180'" fill="none"
         viewBox="0 0 24 24" stroke="currentColor">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M19 9l-7 7-7-7" />
    </svg>
  </button>

  <!-- Dropdown menu -->
  <div x-show="open"
       x-transition
       @click.away="open = false"
       class="absolute right-0 z-50 mt-2 w-56 origin-top-right
              rounded-xl bg-white p-1 shadow-lg
              ring-1 ring-zinc-950/10
              dark:bg-zinc-900 dark:ring-white/10">
    {{range .Items}}
      {{if eq .Type "divider"}}
        <hr class="my-1 border-zinc-950/5 dark:border-white/5">
      {{else if .HTMX}}
        <button hx-{{.HTMX.Method}}="{{.HTMX.URL}}"
                hx-target="{{.HTMX.Target}}"
                @click="open = false"
                class="flex w-full items-center gap-2 rounded-lg px-3 py-2
                       text-left text-sm text-zinc-950
                       hover:bg-zinc-950/5 dark:text-white
                       dark:hover:bg-white/5">
          {{if .Icon}}
            <svg class="size-4">{{.Icon}}</svg>
          {{end}}
          {{.Label}}
        </button>
      {{else}}
        <a href="{{.Href}}"
           @click="open = false"
           class="flex items-center gap-2 rounded-lg px-3 py-2 text-sm
                  text-zinc-950 hover:bg-zinc-950/5
                  dark:text-white dark:hover:bg-white/5">
          {{if .Icon}}
            <svg class="size-4">{{.Icon}}</svg>
          {{end}}
          {{.Label}}
        </a>
      {{end}}
    {{end}}
  </div>
</div>
```

**Keyboard Navigation (Alpine.js):**
```javascript
// Add to Alpine component
{
  open: false,
  activeIndex: 0,
  items: [],
  init() {
    this.items = this.$el.querySelectorAll('[role="menuitem"]');
  },
  onKeydown(e) {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      this.activeIndex = (this.activeIndex + 1) % this.items.length;
      this.items[this.activeIndex].focus();
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      this.activeIndex = (this.activeIndex - 1 + this.items.length) % this.items.length;
      this.items[this.activeIndex].focus();
    } else if (e.key === 'Escape') {
      this.open = false;
    }
  }
}
```

**Files to Create/Update:**
- `web/templates/components/dropdown.html`
- Update user menu to use dropdown
- Add action dropdowns to tables

### 4.2 Dialog/Modal Component

**Current State**: Basic modal structure
**Target State**: Polished dialog with transitions, sizes, and proper accessibility

**Tasks:**
- [ ] Create dialog wrapper (Alpine.js)
- [ ] Build dialog content structure (title, description, body, actions)
- [ ] Add size variants (sm, md, lg, xl, full)
- [ ] Implement backdrop with transition
- [ ] Add close button
- [ ] Handle focus trapping
- [ ] Add keyboard support (Escape to close)

**Dialog Structure:**
```html
<div x-data="{ open: false }"
     x-on:open-dialog.window="open = true"
     x-on:keydown.escape.window="open = false">

  <!-- Backdrop -->
  <div x-show="open"
       x-transition.opacity
       class="fixed inset-0 z-50 bg-zinc-950/25 dark:bg-zinc-950/50"></div>

  <!-- Dialog -->
  <div x-show="open"
       x-transition
       class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <div @click.away="open = false"
         class="relative w-full {{template "dialog-size" .Size}}
                rounded-3xl bg-white p-8 shadow-lg
                dark:bg-zinc-900 dark:ring-1 dark:ring-white/10">

      <!-- Close button -->
      <button @click="open = false"
              type="button"
              class="absolute right-4 top-4 text-zinc-400
                     hover:text-zinc-950 dark:hover:text-white">
        <svg class="size-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>

      <!-- Dialog title -->
      <h2 class="text-2xl/8 font-semibold text-zinc-950 dark:text-white">
        {{.Title}}
      </h2>

      <!-- Dialog description -->
      {{if .Description}}
        <p class="mt-2 text-base/6 text-zinc-500 dark:text-zinc-400">
          {{.Description}}
        </p>
      {{end}}

      <!-- Dialog body -->
      <div class="mt-6">
        {{.Body}}
      </div>

      <!-- Dialog actions -->
      {{if .Actions}}
        <div class="mt-8 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          {{range .Actions}}
            {{template "button" .}}
          {{end}}
        </div>
      {{end}}
    </div>
  </div>
</div>

{{define "dialog-size"}}
  {{if eq . "sm"}}max-w-sm
  {{else if eq . "md"}}max-w-md
  {{else if eq . "lg"}}max-w-lg
  {{else if eq . "xl"}}max-w-xl
  {{else if eq . "full"}}max-w-7xl
  {{else}}max-w-md{{end}}
{{end}}
```

**Opening Dialog via HTMX:**
```html
<!-- Trigger button -->
<button hx-get="/violations/new"
        hx-target="#dialog-content"
        hx-swap="innerHTML"
        @click="$dispatch('open-dialog')"
        class="...">
  Add Violation
</button>

<!-- Dialog container -->
<div id="dialog-container">
  <div id="dialog-content">
    <!-- HTMX will inject dialog content here -->
  </div>
</div>
```

**Files to Create/Update:**
- `web/templates/components/dialog.html`
- Update manual violation creation to use dialog
- Update photo delete confirmation to use dialog
- Add dialogs for other confirmations

### 4.3 Table Component

**Current State**: Basic table with borders
**Target State**: Sophisticated table with variants, clickable rows, and proper styling

**Tasks:**
- [ ] Create table wrapper component
- [ ] Build table head, body, row, cell components
- [ ] Add variants (striped, grid, dense, bleed)
- [ ] Implement clickable rows with HTMX
- [ ] Add hover states
- [ ] Make responsive (horizontal scroll on mobile)
- [ ] Add loading states

**Table Structure:**
```go
{{define "table"}}
<div class="flow-root">
  <div class="-mx-[--gutter] overflow-x-auto whitespace-nowrap">
    <div class="inline-block min-w-full align-middle px-[--gutter]">
      <table class="min-w-full text-left text-sm/6 text-zinc-950 dark:text-white">
        <thead class="text-zinc-500 dark:text-zinc-400">
          {{.Head}}
        </thead>
        <tbody>
          {{.Body}}
        </tbody>
      </table>
    </div>
  </div>
</div>
{{end}}
```

**Table Row (Clickable with HTMX):**
```html
<tr hx-get="/inspections/{{.ID}}"
    hx-push-url="true"
    class="cursor-pointer hover:bg-zinc-950/[2.5%] dark:hover:bg-white/[2.5%]">
  <td class="relative px-4 py-4 first:pl-[--gutter] last:pr-[--gutter]">
    {{.Column1}}
  </td>
  <td class="relative px-4 py-4">
    {{.Column2}}
  </td>
</tr>
```

**Table Variants:**

1. **Striped**:
```css
tbody tr:nth-child(odd) {
  @apply bg-zinc-950/[2.5%] dark:bg-white/[2.5%];
}
```

2. **Grid** (all borders):
```css
td, th {
  @apply border-b border-zinc-950/5 dark:border-white/5;
}
td:not(:last-child), th:not(:last-child) {
  @apply border-r border-zinc-950/5 dark:border-white/5;
}
```

3. **Dense** (tighter spacing):
```css
td, th {
  @apply px-3 py-2;
}
```

4. **Bleed** (extends to edges):
```css
.table-bleed {
  @apply -mx-[--gutter];
}
```

**Files to Create/Update:**
- `web/templates/components/table.html`
- Update inspection list to use new table
- Update organization/project lists to use table (or keep as cards)

### 4.4 Tabs Component

**Current State**: Simple Alpine.js tabs
**Target State**: Polished tabs with active indicators and transitions

**Tasks:**
- [ ] Create tabs wrapper (Alpine.js)
- [ ] Build tab list and tab buttons
- [ ] Create tab panels with transitions
- [ ] Add active state indicator
- [ ] Implement keyboard navigation
- [ ] Style tab variants (line, pills)

**Tabs Structure:**
```html
<div x-data="{ activeTab: '{{.DefaultTab}}' }">
  <!-- Tab list -->
  <div class="border-b border-zinc-950/10 dark:border-white/10">
    <nav class="-mb-px flex gap-8 px-[--gutter]" role="tablist">
      {{range .Tabs}}
        <button @click="activeTab = '{{.ID}}'"
                :class="activeTab === '{{.ID}}' ?
                        'border-zinc-950 text-zinc-950 dark:border-white dark:text-white' :
                        'border-transparent text-zinc-500 hover:border-zinc-300 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300'"
                class="border-b-2 px-1 py-4 text-sm font-semibold transition-colors"
                role="tab"
                :aria-selected="activeTab === '{{.ID}}'"
                :tabindex="activeTab === '{{.ID}}' ? 0 : -1">
          {{.Label}}
          {{if .Count}}
            <span class="ml-2 rounded-full bg-zinc-100 px-2.5 py-0.5 text-xs
                         dark:bg-zinc-800"
                  :class="activeTab === '{{.ID}}' && 'bg-zinc-900 text-white dark:bg-white dark:text-zinc-900'">
              {{.Count}}
            </span>
          {{end}}
        </button>
      {{end}}
    </nav>
  </div>

  <!-- Tab panels -->
  <div class="mt-8">
    {{range .Tabs}}
      <div x-show="activeTab === '{{.ID}}'"
           x-transition
           role="tabpanel"
           :aria-hidden="activeTab !== '{{.ID}}'">
        {{.Content}}
      </div>
    {{end}}
  </div>
</div>
```

**Keyboard Navigation:**
```javascript
// Add to Alpine component
{
  activeTab: 'all',
  tabs: ['all', 'pending', 'confirmed', 'dismissed'],
  onKeydown(e) {
    const currentIndex = this.tabs.indexOf(this.activeTab);
    if (e.key === 'ArrowRight') {
      e.preventDefault();
      this.activeTab = this.tabs[(currentIndex + 1) % this.tabs.length];
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault();
      this.activeTab = this.tabs[(currentIndex - 1 + this.tabs.length) % this.tabs.length];
    }
  }
}
```

**Files to Create/Update:**
- `web/templates/components/tabs.html`
- Update violation filtering to use polished tabs
- Add tabs to other sections as needed

### 4.5 Alert/Toast Component

**Current State**: Flash messages in base layout
**Target State**: Polished alert with dismiss and auto-hide

**Tasks:**
- [ ] Create alert component (Alpine.js)
- [ ] Add variants (success, error, warning, info)
- [ ] Add dismiss button
- [ ] Implement auto-hide after timeout
- [ ] Add smooth transitions
- [ ] Position as toast (top-right corner)

**Alert Structure:**
```html
<div x-data="{
       show: true,
       timeout: {{.Timeout | default 5000}}
     }"
     x-init="setTimeout(() => show = false, timeout)"
     x-show="show"
     x-transition
     class="rounded-xl p-4 shadow-lg ring-1
            {{if eq .Type "success"}}
              bg-green-50 text-green-800 ring-green-600/20
              dark:bg-green-500/10 dark:text-green-400 dark:ring-green-500/20
            {{else if eq .Type "error"}}
              bg-red-50 text-red-800 ring-red-600/20
              dark:bg-red-500/10 dark:text-red-400 dark:ring-red-500/20
            {{else if eq .Type "warning"}}
              bg-amber-50 text-amber-800 ring-amber-600/20
              dark:bg-amber-500/10 dark:text-amber-400 dark:ring-amber-500/20
            {{else}}
              bg-blue-50 text-blue-800 ring-blue-600/20
              dark:bg-blue-500/10 dark:text-blue-400 dark:ring-blue-500/20
            {{end}}">
  <div class="flex items-start gap-3">
    {{if .Icon}}
      <svg class="size-5 shrink-0" fill="currentColor">{{.Icon}}</svg>
    {{end}}

    <div class="flex-1">
      {{if .Title}}
        <h3 class="text-sm font-semibold">{{.Title}}</h3>
      {{end}}
      <p class="text-sm {{if .Title}}mt-1{{end}}">{{.Message}}</p>
    </div>

    <button @click="show = false"
            type="button"
            class="shrink-0 hover:opacity-75">
      <svg class="size-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M6 18L18 6M6 6l12 12" />
      </svg>
    </button>
  </div>
</div>
```

**Toast Container (Fixed Position):**
```html
<!-- In base layout -->
<div class="fixed top-4 right-4 z-50 flex flex-col gap-3 w-full max-w-md">
  {{range .Alerts}}
    {{template "alert" .}}
  {{end}}
</div>
```

**Files to Create/Update:**
- `web/templates/components/alert.html`
- Update flash message display in base layout
- Add toast notifications for HTMX responses

### 4.6 Loading States

**Current State**: Basic spinner
**Target State**: Polished loading indicators for various contexts

**Tasks:**
- [ ] Create spinner component with sizes
- [ ] Create skeleton loaders for content
- [ ] Add loading states to buttons
- [ ] Add loading overlays for forms/sections
- [ ] Integrate with HTMX loading states

**Spinner Component:**
```html
<svg class="{{if eq .Size "sm"}}size-4{{else if eq .Size "lg"}}size-8{{else}}size-6{{end}}
            animate-spin text-{{.Color | default "zinc-950"}}
            dark:text-{{.DarkColor | default "white"}}"
     xmlns="http://www.w3.org/2000/svg"
     fill="none"
     viewBox="0 0 24 24">
  <circle class="opacity-25"
          cx="12" cy="12" r="10"
          stroke="currentColor"
          stroke-width="4"></circle>
  <path class="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
</svg>
```

**Button Loading State:**
```html
<button {{if .Loading}}disabled{{end}} class="...">
  {{if .Loading}}
    <span class="flex items-center gap-2">
      {{template "spinner" (dict "Size" "sm")}}
      Loading...
    </span>
  {{else}}
    {{.Label}}
  {{end}}
</button>
```

**HTMX Loading Indicator:**
```html
<!-- Show spinner during HTMX request -->
<div hx-get="/api/data"
     hx-indicator="#loading-spinner">
  <!-- Content -->
</div>

<div id="loading-spinner" class="htmx-indicator">
  {{template "spinner" .}}
</div>

<style>
.htmx-indicator {
  display: none;
}
.htmx-request .htmx-indicator,
.htmx-request.htmx-indicator {
  display: block;
}
</style>
```

**Skeleton Loader:**
```html
<div class="animate-pulse space-y-4">
  <div class="h-4 bg-zinc-200 rounded dark:bg-zinc-700 w-3/4"></div>
  <div class="h-4 bg-zinc-200 rounded dark:bg-zinc-700"></div>
  <div class="h-4 bg-zinc-200 rounded dark:bg-zinc-700 w-5/6"></div>
</div>
```

**Files to Create/Update:**
- `web/templates/components/spinner.html`
- `web/templates/components/skeleton.html`
- Update buttons to support loading state
- Add HTMX loading indicators throughout

### Phase 4 Success Criteria

- [ ] Dropdown component working with Alpine.js
- [ ] Dialog/modal component functional
- [ ] Table component with all variants
- [ ] Tabs component with keyboard navigation
- [ ] Alert/toast notifications working
- [ ] Loading states implemented throughout
- [ ] All interactive components accessible
- [ ] Smooth transitions on all interactions
- [ ] HTMX integration working seamlessly

**Estimated Time**: 3-4 days

---

## Phase 5: Pages Polish

**Goal**: Apply all new components to existing pages and refine the overall user experience.

### 5.1 Authentication Pages

**Current State**: Basic functional pages
**Target State**: Polished auth pages with refined layout and components

**Tasks:**
- [ ] Refine login page layout
- [ ] Update register page with new form components
- [ ] Polish password reset flow
- [ ] Update email verification page
- [ ] Add better error states
- [ ] Improve success messages

**Auth Page Layout:**
```html
<div class="flex min-h-screen items-center justify-center px-4 py-12
            bg-zinc-50 dark:bg-zinc-950">
  <div class="w-full max-w-md">
    <!-- Logo -->
    <div class="flex justify-center mb-8">
      <span class="text-2xl font-bold text-zinc-950 dark:text-white">
        Aletheia
      </span>
    </div>

    <!-- Card -->
    <div class="rounded-3xl bg-white p-8 shadow-lg ring-1 ring-zinc-950/5
                dark:bg-zinc-900 dark:ring-white/10">
      <h1 class="text-2xl/8 font-semibold text-zinc-950 dark:text-white">
        {{.Title}}
      </h1>

      {{if .Description}}
        <p class="mt-2 text-sm/6 text-zinc-500 dark:text-zinc-400">
          {{.Description}}
        </p>
      {{end}}

      <form hx-post="{{.Action}}"
            hx-target="this"
            hx-swap="outerHTML"
            class="mt-8 space-y-6">
        {{.FormFields}}

        {{template "button" (dict "Type" "submit" "Variant" "solid"
                                   "Content" .SubmitLabel "Class" "w-full")}}
      </form>

      {{if .Links}}
        <div class="mt-6 text-center text-sm text-zinc-500 dark:text-zinc-400">
          {{range .Links}}
            <a href="{{.Href}}"
               class="font-medium text-zinc-950 hover:text-zinc-700
                      dark:text-white dark:hover:text-zinc-300">
              {{.Label}}
            </a>
          {{end}}
        </div>
      {{end}}
    </div>
  </div>
</div>
```

**Files to Update:**
- `web/templates/pages/login.html`
- `web/templates/pages/register.html`
- `web/templates/pages/forgot-password.html`
- `web/templates/pages/reset-password.html`
- `web/templates/pages/verify.html`

### 5.2 Dashboard Page

**Current State**: Basic dashboard with navigation
**Target State**: Polished dashboard with widgets and quick actions

**Tasks:**
- [ ] Add stats cards (total inspections, pending violations, etc.)
- [ ] Create recent inspections widget
- [ ] Add violation summary chart/visualization
- [ ] Implement quick actions section
- [ ] Add empty state for new users
- [ ] Make responsive

**Dashboard Layout:**
```html
<div class="mx-auto max-w-7xl">
  <header class="px-[--gutter] py-8">
    <h1 class="text-2xl/8 font-semibold text-zinc-950 dark:text-white">
      Dashboard
    </h1>
    <p class="mt-2 text-base/7 text-zinc-600 dark:text-zinc-400">
      Welcome back, {{.User.DisplayName}}
    </p>
  </header>

  <div class="px-[--gutter] pb-16 space-y-8">
    <!-- Stats cards -->
    <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
      {{range .Stats}}
        <div class="rounded-2xl bg-white p-6 ring-1 ring-zinc-950/5
                    dark:bg-zinc-900 dark:ring-white/10">
          <div class="text-sm/6 text-zinc-500 dark:text-zinc-400">
            {{.Label}}
          </div>
          <div class="mt-2 text-3xl font-semibold text-zinc-950 dark:text-white">
            {{.Value}}
          </div>
          {{if .Change}}
            <div class="mt-2 text-sm/6
                        {{if .ChangePositive}}text-green-600 dark:text-green-400
                        {{else}}text-red-600 dark:text-red-400{{end}}">
              {{.Change}}
            </div>
          {{end}}
        </div>
      {{end}}
    </div>

    <!-- Recent inspections -->
    <section>
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-base/7 font-semibold text-zinc-950 dark:text-white">
          Recent Inspections
        </h2>
        <a href="/inspections"
           class="text-sm/6 font-medium text-zinc-950 hover:text-zinc-700
                  dark:text-white dark:hover:text-zinc-300">
          View all →
        </a>
      </div>

      {{if .RecentInspections}}
        {{template "table" .RecentInspections}}
      {{else}}
        {{template "empty-state" (dict "Title" "No inspections yet"
                                       "Description" "Create your first inspection to get started"
                                       "Action" (dict "Label" "Create Inspection" "Href" "/inspections/new"))}}
      {{end}}
    </section>
  </div>
</div>
```

**Files to Update:**
- `web/templates/pages/dashboard.html`
- Add stats calculation in dashboard handler

### 5.3 Organization & Project Pages

**Current State**: Basic list and form pages
**Target State**: Polished pages with refined cards and forms

**Tasks:**
- [ ] Refine organization list with better cards
- [ ] Update organization creation form
- [ ] Polish project list layout
- [ ] Refine project creation form
- [ ] Improve project detail page
- [ ] Add better empty states

**Organization Card:**
```html
<a href="/organizations/{{.ID}}"
   class="group relative rounded-2xl bg-white p-6 ring-1 ring-zinc-950/5
          hover:bg-zinc-50 dark:bg-zinc-900 dark:ring-white/10
          dark:hover:bg-zinc-800 transition-colors">
  <h3 class="text-base/7 font-semibold text-zinc-950 dark:text-white">
    {{.Name}}
  </h3>

  {{if .Description}}
    <p class="mt-2 text-sm/6 text-zinc-500 dark:text-zinc-400 line-clamp-2">
      {{.Description}}
    </p>
  {{end}}

  <div class="mt-4 flex items-center gap-4 text-sm/6 text-zinc-500 dark:text-zinc-400">
    <span>{{.ProjectCount}} projects</span>
    <span>{{.MemberCount}} members</span>
  </div>

  <div class="absolute right-6 top-6 opacity-0 group-hover:opacity-100 transition-opacity">
    <svg class="size-5 text-zinc-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M9 5l7 7-7 7" />
    </svg>
  </div>
</a>
```

**Project Card:**
```html
<a href="/projects/{{.ID}}"
   class="group relative rounded-2xl bg-white p-6 ring-1 ring-zinc-950/5
          hover:bg-zinc-50 dark:bg-zinc-900 dark:ring-white/10
          dark:hover:bg-zinc-800 transition-colors">
  <div class="flex items-start justify-between">
    <div>
      <h3 class="text-base/7 font-semibold text-zinc-950 dark:text-white">
        {{.Name}}
      </h3>
      <p class="mt-1 text-sm/6 text-zinc-500 dark:text-zinc-400">
        {{.Organization.Name}}
      </p>
    </div>
    {{template "badge" (dict "Content" .Status "Color" (statusColor .Status))}}
  </div>

  {{if .Location}}
    <div class="mt-4 text-sm/6 text-zinc-500 dark:text-zinc-400">
      {{.Location.City}}, {{.Location.State}}
    </div>
  {{end}}

  <div class="mt-4 flex items-center gap-4 text-sm/6 text-zinc-500 dark:text-zinc-400">
    <span>{{.InspectionCount}} inspections</span>
    {{if .LastInspectionDate}}
      <span>Last: {{.LastInspectionDate | formatDate}}</span>
    {{end}}
  </div>
</a>
```

**Files to Update:**
- `web/templates/pages/organizations.html`
- `web/templates/pages/organization-new.html`
- `web/templates/pages/projects.html`
- `web/templates/pages/project-new.html`
- `web/templates/pages/project-detail.html`

### 5.4 Inspection Pages

**Current State**: Functional but basic styling
**Target State**: Polished inspection workflow with refined UI

**Tasks:**
- [ ] Refine inspection list table
- [ ] Polish inspection creation form
- [ ] Improve inspection detail layout
- [ ] Update photo gallery with better styling
- [ ] Add better status indicators
- [ ] Improve empty states

**Inspection Detail Layout:**
```html
<div class="mx-auto max-w-7xl">
  <!-- Header -->
  <header class="px-[--gutter] py-8">
    <div class="flex items-start justify-between">
      <div>
        <div class="flex items-center gap-3">
          <h1 class="text-2xl/8 font-semibold text-zinc-950 dark:text-white">
            Inspection #{{.ID}}
          </h1>
          {{template "badge" (dict "Content" .Status "Color" (statusColor .Status))}}
        </div>

        <div class="mt-2 space-y-1 text-sm/6 text-zinc-500 dark:text-zinc-400">
          <div>{{.Project.Name}} • {{.Project.Organization.Name}}</div>
          <div>{{.CreatedAt | formatDate}} by {{.Inspector.DisplayName}}</div>
        </div>
      </div>

      <div class="flex gap-3">
        {{template "button" (dict "Variant" "outline" "Content" "Edit")}}
        {{template "button" (dict "Variant" "solid" "Content" "Generate Report")}}
      </div>
    </div>
  </header>

  <!-- Tabs -->
  <div class="px-[--gutter]">
    {{template "tabs" (dict
      "DefaultTab" "photos"
      "Tabs" (list
        (dict "ID" "photos" "Label" "Photos" "Count" .PhotoCount)
        (dict "ID" "violations" "Label" "Violations" "Count" .ViolationCount)
        (dict "ID" "details" "Label" "Details")
      ))}}
  </div>
</div>
```

**Photo Gallery (Refined):**
```html
<div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
  {{range .Photos}}
    <div class="group relative rounded-2xl bg-white ring-1 ring-zinc-950/5
                overflow-hidden dark:bg-zinc-900 dark:ring-white/10">
      <!-- Photo -->
      <a href="/photos/{{.ID}}" class="block aspect-square">
        <img src="{{.ThumbnailURL}}"
             alt="Inspection photo"
             class="size-full object-cover group-hover:opacity-90 transition-opacity">
      </a>

      <!-- Info overlay -->
      <div class="p-4">
        <div class="flex items-center justify-between">
          <span class="text-sm/6 text-zinc-500 dark:text-zinc-400">
            {{.CreatedAt | formatDate}}
          </span>

          {{if eq .AnalysisStatus "completed"}}
            <div class="flex items-center gap-2">
              {{if gt .ViolationCount 0}}
                {{template "badge" (dict "Content" (printf "%d violations" .ViolationCount)
                                         "Color" "red")}}
              {{else}}
                {{template "badge" (dict "Content" "No violations" "Color" "green")}}
              {{end}}
            </div>
          {{else if eq .AnalysisStatus "analyzing"}}
            {{template "badge" (dict "Content" "Analyzing..." "Color" "blue")}}
          {{else}}
            {{template "button" (dict "Variant" "plain" "Size" "sm"
                                     "Content" "Analyze"
                                     "HXPost" (printf "/photos/%d/analyze" .ID))}}
          {{end}}
        </div>
      </div>
    </div>
  {{end}}
</div>
```

**Files to Update:**
- `web/templates/pages/inspections.html`
- `web/templates/pages/inspection-new.html`
- `web/templates/pages/inspection-detail.html`

### 5.5 Photo Detail & Violation Pages

**Current State**: Functional AI analysis and violation management
**Target State**: Polished photo detail with refined violation cards

**Tasks:**
- [ ] Refine photo detail page layout
- [ ] Polish violation cards with better visual hierarchy
- [ ] Improve analysis controls section
- [ ] Add better loading states during analysis
- [ ] Refine manual violation creation form
- [ ] Improve confidence score visualization

**Photo Detail Layout:**
```html
<div class="mx-auto max-w-7xl">
  <header class="px-[--gutter] py-8">
    <a href="/inspections/{{.InspectionID}}"
       class="inline-flex items-center gap-2 text-sm/6 text-zinc-500
              hover:text-zinc-950 dark:text-zinc-400 dark:hover:text-white">
      ← Back to inspection
    </a>

    <h1 class="mt-4 text-2xl/8 font-semibold text-zinc-950 dark:text-white">
      Photo Details
    </h1>
  </header>

  <div class="px-[--gutter] pb-16">
    <div class="grid gap-8 lg:grid-cols-2">
      <!-- Photo -->
      <div class="rounded-2xl bg-white p-6 ring-1 ring-zinc-950/5
                  dark:bg-zinc-900 dark:ring-white/10">
        <img src="{{.URL}}"
             alt="Inspection photo"
             class="w-full rounded-lg">

        <!-- Analysis controls -->
        {{template "analysis-controls" .}}
      </div>

      <!-- Violations -->
      <div class="space-y-6">
        <div class="flex items-center justify-between">
          <h2 class="text-base/7 font-semibold text-zinc-950 dark:text-white">
            Detected Violations
          </h2>
          <button @click="$dispatch('open-dialog')"
                  hx-get="/violations/new?photo_id={{.ID}}"
                  hx-target="#dialog-content"
                  class="...">
            Add Violation
          </button>
        </div>

        {{if .Violations}}
          <div class="space-y-4">
            {{range .Violations}}
              {{template "violation-card" .}}
            {{end}}
          </div>
        {{else}}
          {{template "empty-state" (dict
            "Title" "No violations detected"
            "Description" "Run AI analysis to detect safety violations")}}
        {{end}}
      </div>
    </div>
  </div>
</div>
```

**Refined Violation Card:**
```html
<div class="rounded-2xl bg-white p-6 ring-1 ring-zinc-950/5
            dark:bg-zinc-900 dark:ring-white/10">
  <!-- Header -->
  <div class="flex items-start justify-between">
    <div class="flex gap-3">
      {{template "badge" (dict "Content" .Severity "Color" (severityColor .Severity))}}
      {{template "badge" (dict "Content" .Status "Color" (statusColor .Status))}}
    </div>

    {{if eq .Status "pending"}}
      <div class="flex gap-2">
        {{template "button" (dict "Variant" "outline" "Size" "sm"
                                  "Content" "Confirm"
                                  "HXPost" (printf "/violations/%d/confirm" .ID))}}
        {{template "button" (dict "Variant" "plain" "Size" "sm"
                                  "Content" "Dismiss"
                                  "HXPost" (printf "/violations/%d/dismiss" .ID))}}
      </div>
    {{end}}
  </div>

  <!-- Description -->
  <p class="mt-4 text-base/7 text-zinc-950 dark:text-white">
    {{.Description}}
  </p>

  <!-- Safety code -->
  {{if .SafetyCode}}
    <div class="mt-4 rounded-lg bg-zinc-50 p-4 dark:bg-white/5">
      <div class="text-sm/6 font-medium text-zinc-950 dark:text-white">
        {{.SafetyCode.Code}}
      </div>
      <div class="mt-1 text-sm/6 text-zinc-600 dark:text-zinc-400">
        {{.SafetyCode.Description}}
      </div>
    </div>
  {{end}}

  <!-- Metadata -->
  <div class="mt-4 flex flex-wrap items-center gap-4 text-sm/6 text-zinc-500 dark:text-zinc-400">
    {{if .ConfidenceScore}}
      <span class="flex items-center gap-1">
        <svg class="size-4" fill="currentColor"><!-- Icon --></svg>
        {{.ConfidenceScore}}% confidence
      </span>
    {{end}}

    {{if .Location}}
      <span>{{.Location}}</span>
    {{end}}

    <span>{{.DetectedAt | formatDate}}</span>
  </div>
</div>
```

**Files to Update:**
- `web/templates/pages/photo-detail.html`
- `web/templates/components/violation-card.html`
- `web/templates/components/analysis-controls.html`

### 5.6 Profile & Settings Pages

**Current State**: Basic profile page
**Target State**: Polished profile with better form layout

**Tasks:**
- [ ] Refine profile page layout
- [ ] Update profile edit form
- [ ] Add avatar upload section
- [ ] Improve password change form
- [ ] Add better success/error feedback

**Profile Page Layout:**
```html
<div class="mx-auto max-w-4xl">
  <header class="px-[--gutter] py-8">
    <h1 class="text-2xl/8 font-semibold text-zinc-950 dark:text-white">
      Profile
    </h1>
  </header>

  <div class="px-[--gutter] pb-16 space-y-8">
    <!-- Profile info -->
    <section class="rounded-2xl bg-white p-8 ring-1 ring-zinc-950/5
                    dark:bg-zinc-900 dark:ring-white/10">
      <div class="flex items-center gap-6">
        {{template "avatar" (dict "Size" "xl" "ImageURL" .User.AvatarURL
                                  "Initials" .User.Initials)}}

        <div>
          <h2 class="text-base/7 font-semibold text-zinc-950 dark:text-white">
            {{.User.DisplayName}}
          </h2>
          <p class="text-sm/6 text-zinc-500 dark:text-zinc-400">
            {{.User.Email}}
          </p>
        </div>
      </div>

      <form hx-post="/profile/update"
            hx-target="this"
            class="mt-8 space-y-6">
        {{template "field" (dict
          "Label" "Display Name"
          "ID" "display_name"
          "Input" (template "input" (dict "Name" "display_name" "Value" .User.DisplayName)))}}

        {{template "field" (dict
          "Label" "Email"
          "ID" "email"
          "Input" (template "input" (dict "Type" "email" "Name" "email" "Value" .User.Email)))}}

        <div class="flex justify-end">
          {{template "button" (dict "Type" "submit" "Content" "Save Changes")}}
        </div>
      </form>
    </section>

    <!-- Change password -->
    <section class="rounded-2xl bg-white p-8 ring-1 ring-zinc-950/5
                    dark:bg-zinc-900 dark:ring-white/10">
      <h2 class="text-base/7 font-semibold text-zinc-950 dark:text-white">
        Change Password
      </h2>

      <form hx-post="/profile/change-password"
            hx-target="this"
            class="mt-6 space-y-6">
        {{template "field" (dict
          "Label" "Current Password"
          "ID" "current_password"
          "Input" (template "input" (dict "Type" "password" "Name" "current_password")))}}

        {{template "field" (dict
          "Label" "New Password"
          "ID" "new_password"
          "Input" (template "input" (dict "Type" "password" "Name" "new_password")))}}

        {{template "field" (dict
          "Label" "Confirm New Password"
          "ID" "confirm_password"
          "Input" (template "input" (dict "Type" "password" "Name" "confirm_password")))}}

        <div class="flex justify-end">
          {{template "button" (dict "Type" "submit" "Content" "Update Password")}}
        </div>
      </form>
    </section>
  </div>
</div>
```

**Files to Update:**
- `web/templates/pages/profile.html`

### 5.7 Empty States

**Current State**: No dedicated empty state handling
**Target State**: Polished empty states throughout the app

**Tasks:**
- [ ] Create empty state component
- [ ] Add to inspection list (no inspections)
- [ ] Add to photo gallery (no photos)
- [ ] Add to violation list (no violations)
- [ ] Add to organization/project lists

**Empty State Component:**
```go
{{define "empty-state"}}
<div class="flex flex-col items-center justify-center py-12 text-center">
  {{if .Icon}}
    <svg class="size-12 text-zinc-400 dark:text-zinc-600" fill="none"
         viewBox="0 0 24 24" stroke="currentColor">
      {{.Icon}}
    </svg>
  {{end}}

  <h3 class="mt-4 text-base/7 font-semibold text-zinc-950 dark:text-white">
    {{.Title}}
  </h3>

  {{if .Description}}
    <p class="mt-2 text-sm/6 text-zinc-500 dark:text-zinc-400 max-w-sm">
      {{.Description}}
    </p>
  {{end}}

  {{if .Action}}
    <div class="mt-6">
      {{template "button" .Action}}
    </div>
  {{end}}
</div>
{{end}}
```

**Files to Create:**
- `web/templates/components/empty-state.html`

### 5.8 Error Pages

**Current State**: Basic 404/500 pages
**Target State**: Polished error pages with helpful actions

**Tasks:**
- [ ] Refine 404 page
- [ ] Refine 500 page
- [ ] Add 403 (forbidden) page
- [ ] Add helpful navigation back to app

**Error Page Layout:**
```html
<div class="flex min-h-screen items-center justify-center px-4">
  <div class="text-center">
    <p class="text-base/8 font-semibold text-zinc-500 dark:text-zinc-400">
      {{.Code}}
    </p>
    <h1 class="mt-4 text-4xl font-bold text-zinc-950 dark:text-white">
      {{.Title}}
    </h1>
    <p class="mt-6 text-base/7 text-zinc-600 dark:text-zinc-400">
      {{.Description}}
    </p>
    <div class="mt-10 flex items-center justify-center gap-4">
      {{template "button" (dict "Variant" "solid" "Content" "Go back"
                                "OnClick" "history.back()")}}
      {{template "button" (dict "Variant" "outline" "Content" "Dashboard"
                                "Href" "/dashboard")}}
    </div>
  </div>
</div>
```

**Files to Update:**
- `web/templates/pages/404.html`
- `web/templates/pages/500.html`
- Create `web/templates/pages/403.html`

### Phase 5 Success Criteria

- [ ] All pages updated with new components
- [ ] Consistent visual language throughout
- [ ] All forms use new field components
- [ ] All buttons use new button component
- [ ] All badges use new badge component
- [ ] Empty states implemented where needed
- [ ] Error pages polished
- [ ] Loading states visible during async operations
- [ ] Dark mode works perfectly on all pages
- [ ] Mobile responsive on all pages
- [ ] No visual regressions
- [ ] Performance maintained or improved

**Estimated Time**: 4-5 days

---

## Testing & Validation

After completing all 5 phases, perform comprehensive testing:

### Visual Testing
- [ ] Compare side-by-side with Catalyst examples
- [ ] Test all pages in light mode
- [ ] Test all pages in dark mode
- [ ] Test on mobile devices (iOS, Android)
- [ ] Test on tablets
- [ ] Test on desktop (various screen sizes)
- [ ] Test in different browsers (Chrome, Firefox, Safari, Edge)

### Accessibility Testing
- [ ] Keyboard navigation on all interactive components
- [ ] Screen reader testing (NVDA, VoiceOver)
- [ ] Color contrast validation (WCAG AA)
- [ ] Focus indicators visible
- [ ] ARIA attributes correct
- [ ] Semantic HTML structure
- [ ] Form labels and error messages

### Functional Testing
- [ ] All HTMX interactions working
- [ ] All Alpine.js components functional
- [ ] Forms submit correctly
- [ ] Validation working
- [ ] Loading states appear appropriately
- [ ] Error handling graceful
- [ ] Navigation working
- [ ] Authentication flow intact

### Performance Testing
- [ ] Page load times acceptable
- [ ] No JavaScript errors in console
- [ ] CSS file size reasonable
- [ ] Images optimized
- [ ] HTMX requests efficient
- [ ] No layout shifts

---

## Rollback Plan

If issues arise during migration:

1. **Git branching strategy**:
   - Create branch: `feature/catalyst-migration`
   - Commit after each phase
   - Tag stable checkpoints

2. **Backup current CSS**:
   - Save `web/static/css/main.css` as `main.css.backup`

3. **Feature flags** (if needed):
   - Environment variable: `USE_CATALYST_UI=true/false`
   - Toggle new vs. old components

4. **Progressive rollout**:
   - Phase 1-2: Internal testing only
   - Phase 3-4: Beta users
   - Phase 5: Full rollout

---

## Documentation Updates

After completion, update project documentation:

- [ ] Update `STYLE_GUIDE.md` with Catalyst patterns
- [ ] Update `web/README.md` with new component usage
- [ ] Create component library documentation
- [ ] Update `CLAUDE.md` with new CSS approach
- [ ] Document Alpine.js patterns used
- [ ] Create troubleshooting guide

---

## Resources

### Reference Materials
- Catalyst UI Kit: `catalyst-ui-kit/demo/javascript/src/components/`
- Tailwind CSS v4 docs: https://tailwindcss.com/docs
- Headless UI docs: https://headlessui.com (for pattern reference)
- Alpine.js docs: https://alpinejs.dev
- HTMX docs: https://htmx.org

### Color Palette
See Catalyst: `catalyst-ui-kit/demo/javascript/src/components/button.tsx` for exact color values

### Typography
Inter font: https://fonts.google.com/specimen/Inter

### Icons
Consider: Heroicons (used by Catalyst) or inline SVG

---

## Success Metrics

### Quantitative
- [ ] CSS file size < 50KB (minified)
- [ ] First Contentful Paint < 1.5s
- [ ] Time to Interactive < 3s
- [ ] Lighthouse Accessibility score > 95
- [ ] Zero JavaScript console errors

### Qualitative
- [ ] Visually consistent with Catalyst examples
- [ ] Professional, polished appearance
- [ ] Intuitive user experience
- [ ] Accessible to all users
- [ ] Maintainable codebase

---

## Timeline Summary

| Phase | Focus | Estimated Time |
|-------|-------|----------------|
| Phase 1 | Foundation & Design Tokens | 1-2 days |
| Phase 2 | Core Components | 2-3 days |
| Phase 3 | Layout & Navigation | 2-3 days |
| Phase 4 | Interactive Components | 3-4 days |
| Phase 5 | Pages Polish | 4-5 days |
| **Total** | | **12-17 days** |

Plus 2-3 days for testing and documentation = **14-20 days total**

---

## Notes

- Maintain backward compatibility where possible
- Test incrementally - don't wait until the end
- Keep commits small and focused
- Document decisions and trade-offs
- Get feedback early and often
- Prioritize accessibility throughout
- Don't sacrifice performance for aesthetics
- Keep the codebase clean and maintainable

---

**Last Updated**: 2025-11-21
**Current Phase**: Phase 2 - Core Components
**Progress**: Phase 1 Complete (1/5 phases done - 20%)
