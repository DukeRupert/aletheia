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

## Authentication UI (Phase 2)

### Public Pages
- [ ] Login page (`/login`)
- [ ] Register page (`/register`)
- [ ] Email verification page (`/verify`)
- [ ] Forgot password page (`/forgot-password`)
- [ ] Reset password page (`/reset-password`)

### Components
- [ ] Form component (validation errors, success messages)
- [ ] Input field component (text, email, password)
- [x] Button component (primary, secondary, danger) - in CSS
- [x] Flash message component (success, error, info) - in base layout

## Dashboard & Navigation (Phase 3)

### Main Dashboard
- [ ] Dashboard layout (`/dashboard`)
- [ ] Organization selector/switcher
- [ ] Recent inspections widget
- [ ] Pending violations count
- [ ] Quick actions menu

### Navigation
- [ ] Top navigation bar
- [ ] Mobile hamburger menu
- [ ] User profile dropdown
- [ ] Logout functionality
- [ ] Breadcrumb navigation

## Organization & Project Management (Phase 4)

### Organization Pages
- [ ] Organization list page (`/organizations`)
- [ ] Create organization form (modal or page)
- [ ] Organization detail page (`/organizations/:id`)
- [ ] Organization settings page
- [ ] Member management interface
  - [ ] Member list
  - [ ] Invite member form
  - [ ] Change member role
  - [ ] Remove member

### Project Pages
- [ ] Project list page (`/projects`)
- [ ] Create project form (HTMX modal)
- [ ] Project detail page (`/projects/:id`)
- [ ] Edit project form
- [ ] Archive/delete project

### Components
- [ ] Card component (for projects, organizations)
- [ ] Table component (for lists)
- [ ] Modal component (HTMX-powered)
- [ ] Dropdown menu component

## Inspection Workflow (Phase 5)

### Inspection Pages
- [ ] Inspection list page (`/inspections`)
  - [ ] Filter by project
  - [ ] Filter by status
  - [ ] Sort by date
- [ ] Create inspection page (`/inspections/new`)
  - [ ] Select project
  - [ ] Select inspector
  - [ ] Set status
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
