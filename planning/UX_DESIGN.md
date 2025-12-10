# Aletheia UX Design Reference

This document summarizes the key findings and recommendations from the UX interface specialist's comprehensive analysis of the Aletheia construction safety inspection platform.

## Executive Summary

Aletheia targets professional safety inspectors who need to efficiently document violations, leverage AI for detection, and generate compliance reports. The application must balance:

- **Speed**: Inspectors work on-site with time constraints
- **Accuracy**: Violations must cite specific regulations and be verifiable
- **Trust**: AI suggestions must be reviewable and correctable by humans
- **Compliance**: Reports must meet legal and regulatory standards

## Critical User Flow Improvements

### 1. Simplified Inspection Creation

**Original Flow**: User provides "general goal of inspection" when creating inspection.

**Improved Flow**:
- Inspection created immediately on click (no form required)
- Defaults to "draft" status
- Goal/purpose captured implicitly through project context and violation documentation

**Impact**: Reduces inspection creation from multi-step form to single click, faster time to uploading photos.

### 2. Async Job Handling Strategy

**Challenge**: Photo analysis takes 30-60 seconds, creating potential UX friction.

**Solution**:
- Show "Analysis in progress..." with auto-refresh every 5 seconds
- Allow inspectors to continue uploading more photos while analysis runs
- Display persistent status bar when jobs are running: "3 photos analyzing... (Estimated 45 seconds remaining)"
- Photo card shows different states: Uploaded → Analyzing → Analysis Complete

**Critical**: Inspector must understand analysis is happening in background and can continue working.

### 3. Smart Status Auto-Advancement

**Implementation**:
- **Draft → In Progress**: Auto-advance when first photo is uploaded
- **In Progress → Completed**: Inspector must explicitly mark complete (with confirmation)

**UX**: Prominent "Mark Inspection Complete" button appears after all photos analyzed, with confirmation dialog.

## Core Pages and Their Purpose

### 1. Dashboard (`/dashboard`)
**Purpose**: Inspector's home base - quick access to active work

**Key Elements**:
- Recent inspections (last 10) with status badges and quick stats
- Quick stats cards: inspections this week, violations detected, reports generated
- "New Project" prominent button
- Organization switcher (if member of multiple orgs)

**Empty State**: "Welcome! Create your first project to get started."

### 2. Inspection Detail (`/inspections/:id`)
**Purpose**: Central workspace for uploading photos and reviewing violations

**Key Elements**:
- Inspection metadata (ID, project name, status badge, timestamps)
- Photo grid with thumbnails showing violation counts
- Background job status indicator (if any jobs running)
- "Add Photo" button (mobile-friendly, opens camera on mobile)
- "Analyze All Photos" batch action
- "Generate Report" button (only when status = completed)

**Critical Feature**: Inline violation review without clicking into each photo (for efficiency).

### 3. Photo Detail (`/photos/:id`)
**Purpose**: Deep focus on single photo for thorough violation review

**Key Elements**:
- Full-size photo (zoomable)
- Split view: Photo on left (sticky), violations on right (scrollable)
- Each violation shows: safety code, severity badge, status badge, description, location, confidence score
- Confirm/Dismiss buttons per violation
- "Add Manual Violation" form at bottom
- Re-analyze with context option

**Keyboard Shortcuts**: C=confirm, D=dismiss, E=edit, N=next, P=previous

### 4. Project Detail (`/projects/:id`)
**Purpose**: Manage project metadata and access all inspections for a site

**Key Elements**:
- Project header (name, client, address, description)
- Project stats (total inspections, latest date, total violations)
- Inspections table with date, inspector name, status, violation count
- "New Inspection" prominent button
- "Edit Project" secondary button

### 5. Organization Management (`/organizations/:id`)
**Purpose**: Admin controls for organization settings and members

**Key Elements**:
- Members list with roles (Owner/Admin/Member)
- "Invite Member" button
- Pending invitations list
- Organization settings (name, address, logo for reports)
- Custom safety code library management

**Access Control**: Only owners/admins can access this page.

## Component Inventory

### Existing Components (Already Implemented)
- button.html, badge.html, field.html, input.html, textarea.html, select.html
- heading.html, text.html, divider.html, spinner.html, container.html
- page-header.html, grid.html, stack.html, empty-state.html
- avatar.html, nav.html, dropdown.html, dialog.html, table.html
- tabs.html, alert.html, skeleton.html

### New Components Required

#### 1. violation-card.html
Reusable violation display with confirm/dismiss actions.

**Props**:
- ID (for HTMX targeting)
- SafetyCode (e.g., "OSHA 1926.501")
- Severity (critical/high/medium/low)
- Status (pending/confirmed/dismissed)
- Description (text)
- Location (optional)
- ConfidenceScore (0-1)
- ShowActions (boolean - hide in reports)

#### 2. photo-card.html
Photo with violation summary for grid display.

**Props**:
- PhotoID
- ThumbnailURL
- Timestamp
- ViolationCount
- Violations (array - for inline display)
- ShowAnalyzeButton (boolean)
- AnalysisStatus (uploaded/analyzing/analyzed)

#### 3. job-status.html
Background job progress indicator.

**Props**:
- JobCount (number)
- JobType (e.g., "analysis", "report_generation")
- EstimatedTime (optional)

#### 4. autocomplete-input.html
Safety code search for manual violation entry.

**Props**:
- Name, ID, Placeholder
- Options (array from safety_codes table)

**Features**:
- Search by code number or description
- Show recent codes used by inspector
- Highlight matching text

#### 5. breadcrumb.html
Navigation breadcrumb trail.

**Props**:
- Items (array of {label, url})

**Example**: Organization > Project > Inspection > Photo

#### 6. status-badge.html
Inspection status indicator.

**Props**:
- Status (draft/in_progress/completed)
- Size (sm/md/lg)

**Colors**: Gray for draft, blue for in_progress, green for completed

## Key Interaction Patterns

### 1. Photo Upload (HTMX)

```html
<form
  hx-post="/api/upload"
  hx-encoding="multipart/form-data"
  hx-target="#photo-list"
  hx-swap="afterbegin"
  hx-indicator="#upload-status">

  <input type="file" name="image" accept="image/*" capture="environment" multiple>
  <button type="submit">Upload Photos</button>
</form>

<div id="upload-status" class="htmx-indicator">Uploading...</div>
<div id="photo-list" class="grid">
  <!-- Server returns photo-card.html for each upload -->
</div>
```

**UX Flow**:
1. User selects photo → Form submits automatically
2. Loading indicator appears
3. Server saves photo, returns HTML for photo card
4. HTMX injects card at beginning of grid (most recent first)
5. User can immediately see photo and click "Analyze"

### 2. AI Photo Analysis (HTMX + Job Queue)

**Step 1: Trigger Analysis**
```html
<button
  hx-post="/api/photos/analyze"
  hx-vals='{"photo_id": "{{.PhotoID}}"}'
  hx-target="#photo-{{.PhotoID}}"
  hx-swap="outerHTML">
  Analyze
</button>
```

**Step 2: Server Returns "Analyzing" State**
```html
<div id="photo-{{.PhotoID}}" class="photo-card analyzing"
     hx-get="/api/photos/{{.PhotoID}}/status"
     hx-trigger="every 3s"
     hx-swap="outerHTML">

  <img src="{{.ThumbnailURL}}">
  <div class="analyzing-indicator">
    <spinner></spinner> Analyzing... (~30 seconds remaining)
  </div>
</div>
```

**Step 3: Poll Every 3 Seconds Until Complete**

When analysis completes, server returns final photo-card with violations (no `hx-trigger` → polling stops).

**Edge Cases**:
- Job fails → Show error with "Retry" button
- Job times out → Show warning, allow retry
- User navigates away → Resume polling when they return

### 3. Violation Review (HTMX)

**Optimistic UI update with server confirmation**:

```html
<div id="violation-{{.ViolationID}}" class="violation-card pending">
  <badge severity="{{.Severity}}"></badge>
  <badge status="{{.Status}}"></badge>
  <p>{{.Description}}</p>

  <div class="actions">
    <button
      hx-post="/api/violations/{{.ViolationID}}/confirm"
      hx-target="#violation-{{.ViolationID}}"
      hx-swap="outerHTML">
      Confirm
    </button>

    <button
      hx-post="/api/violations/{{.ViolationID}}/dismiss"
      hx-target="#violation-{{.ViolationID}}"
      hx-swap="outerHTML">
      Dismiss
    </button>
  </div>
</div>
```

**UX Flow**:
1. User clicks "Confirm"
2. Server updates database status to "confirmed"
3. Server returns updated HTML with green status badge
4. HTMX swaps entire card (smooth transition)
5. User can toggle back if they change their mind

### 4. Report Generation (Alpine.js + HTMX)

**Pattern**: Async job with progress tracking

```html
<div x-data="{ generating: false }">
  <button
    @click="generating = true"
    hx-post="/api/reports/generate"
    hx-vals='{"inspection_id": "{{.InspectionID}}"}'
    hx-target="#report-status"
    :disabled="generating">
    <span x-show="!generating">Generate Report</span>
    <span x-show="generating">Generating...</span>
  </button>

  <div id="report-status"></div>
</div>
```

**Polling for completion**:
```html
<div hx-get="/api/jobs/{{.JobID}}/status" hx-trigger="every 2s" hx-swap="outerHTML">
  <spinner></spinner> Generating report... (~30 seconds)
</div>
```

**Final state** (no hx-trigger → polling stops):
```html
<div class="alert alert-success">
  Report generated successfully!
  <a href="{{.ReportURL}}" download class="btn-primary">Download Report</a>
</div>
```

## Major UX Improvements

### 1. Batch Photo Analysis
**Problem**: Tedious to analyze 20+ photos individually.

**Solution**: "Analyze All Photos" button at top of grid that enqueues all unanalyzed photos in bulk.

**UX**: Shows aggregate progress: "Analyzing 12 photos... (5 complete, 7 remaining)"

### 2. Violation Editing
**Problem**: AI-generated descriptions might be awkwardly worded, no way to edit without dismissing.

**Solution**: Click violation description to make it editable inline. Severity and safety code also editable.

**UX**: Pencil icon next to description → Click to edit → Save Changes button appears

### 3. Safety Code Autocomplete
**Problem**: Inspector doesn't always know exact OSHA code.

**Solution**: Searchable dropdown with autocomplete:
- Type "1926.501" → Shows "Fall Protection"
- Type "fall" → Shows all fall-related codes
- Show inspector's 5 most recently used codes

**Also Provide**: Template descriptions for common violations.

### 4. Keyboard Shortcuts for Power Users
**Problem**: Reviewing 50 violations with mouse is slow.

**Solution**: Keyboard shortcuts:
- `C` - Confirm current violation
- `D` - Dismiss current violation
- `E` - Edit violation
- `N` - Next violation
- `P` - Previous violation
- `→` - Next photo
- `←` - Previous photo
- `?` - Show help modal with shortcuts

**UX**: Visual indicator shows current focused violation.

### 5. Pre-flight Report Validation
**Problem**: Inspector might generate incomplete report.

**Solution**: Before generating, show summary:
```
Ready to generate report with:
- 12 confirmed violations (3 critical, 5 high, 4 medium)
- 7 dismissed violations (excluded from report)
- 23 photos attached

⚠ Warning: 2 violations missing safety codes

[Go Back] [Generate Anyway]
```

### 6. Smart Violation Grouping
**Problem**: Photo with 15 violations is overwhelming.

**Solution**: Group violations by severity with collapsible sections:
- "5 Critical Violations ▼" (expanded by default)
- "8 High Violations ▼" (expanded by default)
- "2 Medium Violations ▶" (collapsed by default)

**UX**: Color-code sections with left border (red/orange/yellow).

## Critical Architectural Decisions

### Decision 1: Re-analysis Behavior

**Question**: When user re-analyzes a photo with new context, what happens to existing violations?

**Recommendation**: **Merge with deduplication**
- Backend deduplicates based on description similarity (>80% match)
- Show inspector: "3 new violations found, 2 duplicates merged"
- Provide "View All Raw Results" option for audit trail

**Alternatives Considered**:
- Append (could create duplicates)
- Replace (loses previous findings)

### Decision 2: Photo Deletion with Confirmed Violations

**Question**: Should user be able to delete photos after violations are confirmed?

**Recommendation**: **Block for members, allow for admins with warning**
- Members cannot delete photos with confirmed violations (preserves evidence integrity)
- Owners/admins can delete with confirmation dialog warning about report impact
- Dismissed violations don't block deletion
- Soft delete option (hide but keep in storage) for future consideration

**Alternatives Considered**:
- Allow deletion anytime (could break reports)
- Prevent all deletion (user stuck with bad photos)

### Decision 3: Report Versioning

**Question**: If violations change after report is generated, what happens?

**Recommendation**: **Version reports, allow manual regeneration**
- Each report has version number and timestamp
- Show "Report out of date (3 violations changed)" warning on inspection page
- "Regenerate Report" button creates new version
- Keep all versions accessible (dropdown: "View Version 1, 2, 3")
- Old report remains unchanged (preserves historical record)

**Alternatives Considered**:
- Old report unchanged (could be outdated)
- Auto-regenerate (could regenerate after delivery to client)

### Decision 4: Safety Code Validation

**Question**: Should we enforce that all violations have valid safety codes before allowing report generation?

**Recommendation**: **Require codes only for confirmed violations**
- Pending violations can omit safety code
- Confirmed violations MUST have safety code (enforced in UI)
- Pre-flight summary warns if any confirmed violations missing codes
- Inspector must fix or dismiss those violations before proceeding

**Alternatives Considered**:
- Hard requirement for all (blocks workflow)
- Soft warning (could result in non-compliant reports)

### Decision 5: Mobile Photo Capture

**Question**: Should we support taking photos directly in the app vs uploading from gallery?

**Recommendation**: **Both (file picker with camera option)**
- Use `<input type="file" accept="image/*" capture="environment">`
- On mobile: Opens camera directly
- On desktop: Opens file picker
- Works with progressive enhancement (no JS required)

**Implementation**: HTML5 capture attribute handles both seamlessly.

### Decision 6: Concurrent Editing

**Question**: What happens when two inspectors edit the same inspection simultaneously?

**Recommendation**: **Last write wins for MVP, real-time collaboration for future**
- MVP: Last write wins (acceptable - rare that multiple inspectors work on same inspection simultaneously)
- Future: Add "Last edited by [name] at [time]" indicator
- Future: WebSocket updates for real-time collaboration

**Rationale**: Real-time collaboration is complex and overkill for MVP. Simple approach is acceptable.

## User Journey Phases

### Phase 1: Onboarding & Account Setup

**Path A: New Organization**
1. Sign up with email, password, name, organization name
2. Receive verification email
3. Click link → Redirected to dashboard (empty state)

**Path B: Join Existing Organization**
1. Receive invitation email with link
2. Click link → Signup form with email pre-filled
3. Provide name and password
4. Redirected to dashboard (sees existing projects)

**Success Criteria**: User can access dashboard within 2 minutes of signup.

### Phase 2: Project Setup

1. From dashboard, click "New Project"
2. Fill form: project name, client name (optional), address
3. Redirected to project detail page (empty state)

**Success Criteria**: User can create project in under 60 seconds.

**UX Improvement**: Add help text: "A project represents a construction site. You'll create multiple inspections for each project over time."

### Phase 3: Inspection Creation

1. From project detail page, click "New Inspection"
2. Inspection created immediately (no form)
3. Redirected to inspection detail page
4. Inspector sees empty photo grid with "Add Photo" button

**Success Criteria**: Inspection created in 1 click.

### Phase 4: Photo Upload & AI Analysis

1. Click "Add Photo" → Device camera/file picker opens
2. Select photo(s) → Uploads immediately
3. Photo appears in grid
4. Optionally add context, click "Analyze" (or use "Analyze All Photos")
5. Job enqueued → "Analyzing..." state with polling
6. Violations appear when complete

**Success Criteria**:
- Upload completes in <3 seconds
- Inspector understands analysis is happening in background
- Inspector can continue working while analysis runs

### Phase 5: Violation Review & Decision

1. Analysis completes → Violations appear in photo card
2. Inspector reviews each violation
3. Clicks "Confirm" or "Dismiss"
4. Status updates via HTMX (no page reload)
5. Optionally clicks photo to see detail view for thorough review
6. Optionally adds manual violations AI missed

**Success Criteria**: Inspector can review and decide on violation in <10 seconds.

### Phase 6: Report Generation

1. Inspector marks inspection as "Completed"
2. Clicks "Generate Report"
3. Pre-flight summary shows what will be included
4. Confirms → Job enqueued
5. Loading state: "Generating report... (~30 seconds)"
6. Download link appears
7. PDF includes all confirmed violations with photos and citations

**Success Criteria**:
- Report includes all confirmed violations
- Report is professionally formatted
- Inspector can regenerate if needed

## Accessibility Requirements

### Keyboard Navigation
- All interactive elements focusable via Tab
- Logical tab order follows visual hierarchy
- Escape closes modals/dialogs
- Enter/Space activates buttons
- Arrow keys navigate lists

### Screen Readers
- All images have alt text
- Form inputs have associated labels
- Status changes announced via `aria-live` regions
- HTMX updates announce properly

### Visual Design
- Color contrast ratio ≥ 4.5:1 for text
- Focus indicators visible (2px solid outline)
- Don't rely solely on color (use icons + text)
- Touch targets ≥ 44x44px on mobile

## Mobile-Specific Considerations

### Touch Interactions
- Minimum button size: 44x44px
- Increase spacing between clickable elements
- Use native browser controls (date pickers, file upload)

### Performance
- Lazy load images (thumbnails first)
- Minimize JavaScript bundle (HTMX + Alpine.js are lightweight)
- Compress uploaded photos on client before upload

### Layout
- Single-column layout on mobile
- Stack form fields vertically
- Use bottom sheets for modals
- Fixed header with hamburger menu
- Sticky "Add Photo" button at bottom

## Success Metrics to Track

1. **Time to first photo upload** (target: <2 minutes from inspection creation)
2. **Photos per inspection** (average: indicates thoroughness)
3. **Violation confirmation rate** (% of AI-detected violations confirmed - indicates AI accuracy)
4. **Manual violations per inspection** (indicates what AI is missing)
5. **Time to report generation** (from inspection start to PDF download)
6. **Report regeneration rate** (should be low - indicates quality of first pass)

## MVP vs. Future Enhancements

### Must Have (MVP)
- Core workflow: Upload → Analyze → Review → Report
- Manual violation entry
- Basic error handling and empty states
- Mobile-friendly photo upload
- Keyboard navigation for forms
- Batch photo analysis
- Safety code autocomplete
- Pre-flight report validation

### Nice to Have (Post-MVP)
- Photo annotation (bounding boxes)
- Offline support (PWA with service worker)
- Real-time collaboration (WebSockets)
- Advanced batch operations
- Custom report templates
- Email delivery
- Analytics dashboard (trends over time)
- Map view of projects

## Information Architecture

### Site Map
```
/
├── / (landing page - unauthenticated)
├── /login
├── /register
├── /verify (email verification)
├── /forgot-password
├── /reset-password
│
├── /dashboard (authenticated home)
│
├── /organizations
│   ├── /organizations/new
│   └── /organizations/:id (settings, members, safety codes)
│
├── /projects
│   ├── /projects/new
│   └── /projects/:id
│       └── /projects/:id/inspections/new
│
├── /inspections
│   ├── /inspections (all inspections across projects)
│   └── /inspections/:id (photos grid, primary workspace)
│
├── /photos/:id (detailed violation review)
│
└── /profile (user settings)
```

### Navigation Hierarchy

**Primary Navigation** (authenticated):
- Dashboard
- Projects
- Inspections
- Organizations (if multiple)
- Profile (dropdown)

**Breadcrumbs**: Organization > Project > Inspection > Photo

**Contextual Actions**:
- Project page: "New Inspection"
- Inspection page: "Add Photo", "Update Status", "Generate Report"
- Photo page: "Analyze", "Add Manual Violation"

## Component Design Principles

1. **Single Responsibility**: Each component does one thing well
2. **Composability**: Components can be nested (e.g., violation-card contains badge components)
3. **Prop-based**: Pass data via Go template `dict` - no global state
4. **Semantic HTML**: Use proper elements (`<button>`, `<nav>`, `<article>`)
5. **Accessibility**: ARIA labels, keyboard navigation, focus management
6. **Mobile-first**: Touch-friendly sizes (44px minimum tap targets)

## References

- `CLAUDE.md` - Project architecture and tech stack
- `planning/CATALYST_MIGRATION_GUIDE.md` - Design system and styling guidelines (Tailwind CSS v4 + Catalyst)
- `web/README.md` - Template patterns and usage
