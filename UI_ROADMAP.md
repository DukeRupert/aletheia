# Aletheia UI Roadmap

Frontend implementation using Go templates + HTMX + Alpine.js

## Tech Stack

- **Templates**: Go `html/template`
- **Interactivity**: HTMX for server interactions
- **Client State**: Alpine.js for lightweight reactive UI
- **Styling**: Minimal CSS, semantic HTML-first
- **Icons**: Simple SVG icons (inline)

## Foundation (Phase 1) ✓ COMPLETED

### Template Structure Setup
- [x] Create base layout template (`layouts/base.html`)
- [x] Create navigation component (`components/nav.html`)
- [x] Set up template rendering in Echo
- [x] Configure static file serving (CSS, images)
- [x] Create error page templates (404, 500)
- [x] Create home page template (`pages/home.html`)
- [x] Create PageHandler for rendering templates

### HTMX Integration
- [x] Add HTMX to base template
- [x] Configure HTMX headers in Echo (HX-Request detection middleware)
- [ ] Create HTMX response helpers (deferred until needed)
- [ ] Test basic HTMX interactions (will test as we build features)

### Styling Foundation
- [x] Create minimal CSS file (`static/css/main.css`)
- [x] Define color palette (see STYLE_GUIDE.md)
- [x] Create utility classes for spacing, layout
- [x] Set up responsive containers
- [x] Mobile-first breakpoints
- [x] Component styles (buttons, forms, cards, tables, badges, alerts, modal)
- [x] Navigation styles
- [x] Accessibility features (skip link, focus states)

## Authentication UI (Phase 2) ✓ COMPLETED

### Public Pages
- [x] Login page (`/login`) - HTMX form with redirect to dashboard on success
- [x] Dashboard page (`/dashboard`) - Protected page for post-login
- [x] Register page (`/register`) - HTMX form with redirect to login on success
- [x] Email verification page (`/verify`) - Auto-verify with token in URL, manual entry fallback
- [x] Forgot password page (`/forgot-password`)
- [x] Reset password page (`/reset-password`)
- [x] User profile page (`/profile`) - Edit name with HTMX form

### Components
- [ ] Form component (validation errors, success messages)
- [ ] Input field component (text, email, password)
- [x] Button component (primary, secondary, danger) - in CSS
- [x] Flash message component (success, error, info) - in base layout

### HTMX Integration
- [x] Login handler detects `HX-Request` header
- [x] Returns `HX-Redirect` header for HTMX requests
- [x] Maintains JSON response for API compatibility
- [x] Registration handler with HTMX redirect
- [x] Organization/project creation with HTMX redirect
- [x] Inspection creation with HTMX redirect

## Dashboard & Navigation (Phase 3) - In Progress

### Main Dashboard
- [x] Dashboard layout (`/dashboard`) - Basic page with navigation
- [ ] Organization selector/switcher
- [ ] Recent inspections widget
- [ ] Pending violations count
- [ ] Quick actions menu

### Navigation
- [x] Top navigation bar (with user display name)
- [x] Logo link to dashboard
- [x] Profile link
- [x] Logout functionality
- [ ] Mobile hamburger menu
- [ ] User profile dropdown
- [ ] Breadcrumb navigation

## Organization & Project Management (Phase 4) ✓ COMPLETED

### Organization Pages
- [x] Organization list page (`/organizations`) - Card grid layout with creation dates
- [x] Create organization form (`/organizations/new`) - HTMX form with redirect
- [ ] Organization detail page (`/organizations/:id`)
- [ ] Organization settings page
- [ ] Member management interface
  - [ ] Member list
  - [ ] Invite member form
  - [ ] Change member role
  - [ ] Remove member

### Project Pages
- [x] Project list page (`/projects`) - Shows all projects across user's organizations
- [x] Create project form (`/projects/new`) - HTMX form with organization selector
- [x] Project detail page (`/projects/:id`) - Comprehensive location info collection with full US state dropdown
- [x] Edit project form - Inline editing with location fields (address, city, state, zip, country)
- [ ] Archive/delete project

### Components
- [x] Card component (for projects, organizations) - in CSS
- [ ] Table component (for lists)
- [ ] Modal component (HTMX-powered)
- [ ] Dropdown menu component

## Inspection Workflow (Phase 5) - In Progress

### Inspection Pages
- [x] Inspection list page (`/inspections`) - Global view across all projects with context
- [x] Project-specific inspection list (`/projects/:projectId/inspections`)
- [x] Create inspection page (`/projects/:projectId/inspections/new`) - HTMX form with project context
- [ ] Filter by project (global view)
- [ ] Filter by status
- [ ] Sort by date
- [ ] Inspection detail page (`/inspections/:id`)
  - [ ] Inspection metadata
  - [ ] Photo gallery
  - [ ] Violation summary
  - [ ] Status workflow

### Photo Upload & Management
- [ ] Photo upload interface
  - [ ] Drag & drop zone
  - [ ] File input fallback
  - [ ] Upload progress bar
  - [ ] Multiple file support
- [ ] Photo gallery component
  - [ ] Thumbnail grid
  - [ ] Lightbox viewer
  - [ ] Photo metadata
- [ ] Photo detail view
  - [ ] Full-size image
  - [ ] EXIF data display
  - [ ] Trigger AI analysis button
  - [ ] Violations detected on this photo

### Components
- [ ] File upload component (HTMX)
- [ ] Progress bar component
- [ ] Image gallery component
- [ ] Status badge component (draft, in_progress, completed)
- [ ] Tab component (for inspection sections)

## AI Analysis & Violations (Phase 6)

### AI Analysis Interface
- [ ] "Analyze Photo" button (HTMX trigger)
- [ ] Analysis status indicator
  - [ ] Queued state
  - [ ] Processing state (polling)
  - [ ] Completed state
  - [ ] Failed state
- [ ] Analysis results display
  - [ ] Violations found count
  - [ ] Confidence scores
  - [ ] Severity indicators

### Violation Review Interface
- [ ] Violation list page (`/inspections/:id/violations`)
  - [ ] Group by photo
  - [ ] Filter by severity
  - [ ] Filter by status
- [ ] Violation card component
  - [ ] Photo thumbnail
  - [ ] Description
  - [ ] Severity badge
  - [ ] Confidence score
  - [ ] Safety code reference
  - [ ] Location in image
- [ ] Violation detail modal
  - [ ] Full photo with violation highlighted
  - [ ] Complete details
  - [ ] Action buttons (confirm, dismiss, edit)
- [ ] Violation actions
  - [ ] Confirm violation
  - [ ] Dismiss violation (with reason)
  - [ ] Add notes/comments
  - [ ] Change severity (if needed)

### Components
- [ ] Severity badge (critical, high, medium, low)
- [ ] Confidence indicator (visual bar or percentage)
- [ ] Action buttons group
- [ ] Notes/comments textarea
- [ ] Polling component (for job status)

## Safety Code Management (Phase 7)

### Safety Code Pages
- [ ] Safety code list page (`/safety-codes`)
  - [ ] Search/filter by code or description
  - [ ] Filter by country
  - [ ] Filter by state/province
- [ ] Create safety code form (modal)
- [ ] Edit safety code form
- [ ] Delete safety code confirmation

### Components
- [ ] Search input component
- [ ] Filter dropdown component
- [ ] Confirmation dialog component

## Reports (Phase 8)

### Report Pages
- [ ] Report list page (`/reports`)
- [ ] Generate report page
  - [ ] Select inspection
  - [ ] Preview report
- [ ] Report preview page (HTML view)
  - [ ] Print-friendly layout
  - [ ] Export to PDF button
- [ ] PDF generation (server-side)

### Report Template
- [ ] Report header (logo, project info)
- [ ] Inspection metadata section
- [ ] Photos with violations section
- [ ] Violation summary table
- [ ] Safety code reference section
- [ ] Inspector signature section

### Components
- [ ] Print layout stylesheet
- [ ] Export button
- [ ] Report preview component

## User Settings & Profile (Phase 9)

### Settings Pages
- [ ] User profile page (`/settings/profile`)
  - [ ] Edit name, email
  - [ ] Change password
  - [ ] Profile photo upload
- [ ] Notification preferences
- [ ] Account settings

### Components
- [ ] Avatar component
- [ ] Settings form sections

## Polish & Refinements (Phase 10)

### UX Improvements
- [ ] Loading states for all async actions
- [ ] Empty states for lists
- [ ] Error states with retry buttons
- [ ] Success confirmations
- [ ] Keyboard shortcuts (navigation)
- [ ] Focus management
- [ ] Form validation (client-side + server-side)

### Accessibility
- [ ] Semantic HTML throughout
- [ ] ARIA labels where needed
- [ ] Keyboard navigation
- [ ] Focus indicators
- [ ] Screen reader testing
- [ ] Color contrast validation

### Mobile Optimization
- [ ] Touch-friendly buttons (min 44px)
- [ ] Responsive tables
- [ ] Mobile navigation
- [ ] Photo upload on mobile
- [ ] Swipe gestures for galleries

### Performance
- [ ] Lazy load images
- [ ] Optimize image sizes
- [ ] Minimize CSS
- [ ] Cache static assets
- [ ] Reduce HTMX payload sizes

## Future Enhancements (Post-MVP)

- [ ] Offline support (service worker)
- [ ] Real-time updates (WebSockets)
- [ ] Advanced filtering and search
- [ ] Bulk operations
- [ ] Data export (CSV, Excel)
- [ ] Analytics dashboard
- [ ] Custom report templates
- [ ] Multi-language support

---

## Implementation Notes

### HTMX Patterns

**Inline Edit:**
```html
<div hx-get="/violations/123/edit" hx-target="this" hx-swap="outerHTML">
  <span>{{.Description}}</span>
  <button>Edit</button>
</div>
```

**Form Submission:**
```html
<form hx-post="/inspections" hx-target="#inspection-list" hx-swap="afterbegin">
  <!-- form fields -->
  <button type="submit">Create Inspection</button>
</form>
```

**Polling for Job Status:**
```html
<div hx-get="/jobs/{{.JobID}}/status"
     hx-trigger="every 2s"
     hx-swap="outerHTML">
  <span>Processing... {{.Progress}}%</span>
</div>
```

### Alpine.js Patterns

**Dropdown Menu:**
```html
<div x-data="{ open: false }">
  <button @click="open = !open">Menu</button>
  <div x-show="open" @click.away="open = false">
    <!-- menu items -->
  </div>
</div>
```

**Tabs:**
```html
<div x-data="{ tab: 'details' }">
  <button @click="tab = 'details'">Details</button>
  <button @click="tab = 'photos'">Photos</button>
  <div x-show="tab === 'details'"><!-- content --></div>
  <div x-show="tab === 'photos'"><!-- content --></div>
</div>
```

---

## Development Workflow

1. **Start with HTML** - Build semantic HTML first
2. **Add HTMX** - Make it interactive with server requests
3. **Add Alpine.js** - Add client-side reactivity if needed
4. **Add minimal CSS** - Style for clarity, not decoration
5. **Test on mobile** - Ensure responsive throughout
6. **Test accessibility** - Keyboard navigation, screen readers

---

## Success Metrics

- [ ] All core workflows functional
- [ ] Works on mobile devices (iOS, Android)
- [ ] Loads fast on 3G connection
- [ ] Accessible (WCAG 2.1 AA)
- [ ] No JavaScript errors in console
- [ ] Forms have proper validation
- [ ] Error states are clear and actionable
