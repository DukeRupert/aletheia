# Component Handlers Implementation Summary

This document summarizes the new components and backend handlers created for the Aletheia inspection platform.

## New Components Created

All components are located in `/home/dukerupert/Repos/aletheia/web/templates/components/`

### 1. violation-card.html
**Purpose**: Display safety violations with interactive confirm/dismiss actions

**Features**:
- Visual severity indicator (colored left border)
- Status-aware styling (pending/confirmed/dismissed)
- Safety code prominence
- Confidence score display
- Location information (optional)
- HTMX-powered action buttons
- Inline editing support (optional)

**Usage**:
```go
data := map[string]interface{}{
    "ID":              "violation-123",
    "SafetyCode":      "OSHA 1926.501",
    "Severity":        "critical",
    "Status":          "pending",
    "Description":     "Worker without fall protection",
    "Location":        "Near scaffolding",
    "ConfidenceScore": 0.92,
    "ShowActions":     true,
}
c.Render(http.StatusOK, "violation-card", data)
```

### 2. photo-card.html
**Purpose**: Display photo thumbnails with analysis status and violation summary

**Features**:
- 4 analysis states: uploaded, analyzing, analyzed, failed
- Auto-polling when analyzing (every 3 seconds)
- Inline violation preview (first 2 violations)
- Violation count badge
- Collapsible context input for AI guidance
- Re-analyze and delete buttons

**Usage**:
```go
photo := PhotoCardData{
    PhotoID:         "photo-123",
    InspectionID:    "inspection-456",
    ThumbnailURL:    "/uploads/photo-123-thumb.jpg",
    Timestamp:       time.Now(),
    AnalysisStatus:  "analyzing",
    EstimatedTime:   30,
}
```

### 3. job-status.html
**Purpose**: Persistent background job progress indicator

**Features**:
- Fixed position top bar
- Auto-polls every 5 seconds
- Job type-specific messages
- Estimated time remaining
- Progress bar for batch jobs
- Automatically stops polling when no jobs

**Usage**:
```go
data := JobStatusData{
    JobCount:       3,
    JobType:        "photo_analysis",
    EstimatedTime:  45,
}
c.Render(http.StatusOK, "job-status", data)
```

### 4. autocomplete-input.html
**Purpose**: Searchable dropdown for safety codes

**Features**:
- Real-time search (code, description, category)
- Recently used codes section
- Keyboard navigation (arrows, enter, escape)
- Alpine.js powered
- Dual input pattern (hidden value + visible display)

**Usage**:
```html
{{template "autocomplete-input" (dict
  "Name" "safety_code"
  "ID" "safety-code-input"
  "Label" "Safety Code"
  "Required" true
  "Options" .SafetyCodes
  "RecentCodes" .RecentCodes
)}}
```

**Required Template Function**:
```go
funcMap := template.FuncMap{
    "toJSON": func(v interface{}) template.JS {
        b, _ := json.Marshal(v)
        return template.JS(b)
    },
}
```

### 5. breadcrumb.html
**Purpose**: Navigation breadcrumb trail

**Features**:
- 3 variants: standard, with-action, compact
- Responsive (hides middle items on mobile)
- Accessible with ARIA attributes
- Current page indicator

**Usage**:
```go
breadcrumbs := []BreadcrumbItem{
    {Label: "Dashboard", URL: "/dashboard"},
    {Label: "Projects", URL: "/projects"},
    {Label: "Main Street Building", Active: true},
}
```

### 6. stats-card.html
**Purpose**: Dashboard metrics display

**Features**:
- 8 built-in icons (clipboard, alert, document, photo, check, users, building, chart)
- 5 color themes (blue, red, green, orange, zinc)
- Optional trend indicators (up/down arrows)
- Optional sub-values
- Clickable cards (link to detail pages)

**Usage**:
```go
card := map[string]interface{}{
    "Label":       "Inspections This Week",
    "Value":       12,
    "Icon":        "clipboard",
    "Color":       "blue",
    "Trend":       "up",
    "TrendValue":  "+15%",
    "URL":         "/inspections",
}
```

## Backend Handlers Created/Updated

### 1. Violation Handlers (`internal/handlers/violations.go`)

**Updated**:
- `renderViolationCard()` - Now uses violation-card component template

**New**:
- `SetViolationPending()` - POST `/api/violations/:id/pending` - Undo confirm/dismiss

**Existing** (verified working):
- `ConfirmViolation()` - POST `/api/violations/:id/confirm`
- `DismissViolation()` - POST `/api/violations/:id/dismiss`
- `UpdateViolation()` - PATCH `/api/violations/:violation_id`
- `CreateManualViolation()` - POST `/api/violations/manual`
- `ListViolationsByInspection()` - GET `/api/inspections/:inspection_id/violations`

### 2. Photo Handlers (`internal/handlers/photos.go`)

**New**:
- `DeletePhoto()` - DELETE `/api/photos/:photo_id`
  - Deletes photo and associated violations
  - Checks for confirmed violations (optional protection)
  - HTMX-compatible response
  - TODO: Add storage cleanup (S3/local file deletion)

**Existing** (verified):
- `AnalyzePhoto()` - POST `/api/photos/analyze`
- `GetPhotoAnalysisStatus()` - GET `/api/photos/analyze/:job_id`

### 3. Job Handlers (`internal/handlers/jobs.go`) - NEW FILE

**Created**:
- `GetJobStatus()` - GET `/api/jobs/status`
  - Returns active jobs for user's organization
  - Auto-polled by job-status component
  - Returns HTML (HTMX) or JSON
  - Shows job count, type, estimated time

- `CancelJob()` - POST `/api/jobs/:job_id/cancel`
  - Cancel pending/processing jobs
  - Authorization check
  - Updates job status to failed

- `GetActiveJobsByOrganization()` - Helper method
  - Queries jobs table for active jobs
  - Filters by organization_id
  - Returns pending + processing jobs

## Routes Added to `cmd/main.go`

```go
// Initialize job handler
jobHandler := handlers.NewJobHandler(pool, queries, queueService, logger)

// Photo routes - updated delete route
protected.DELETE("/photos/:photo_id", photoHandler.DeletePhoto)

// Violation routes - added new endpoints
protected.POST("/violations/:id/pending", violationHandler.SetViolationPending)
protected.PATCH("/violations/:violation_id", violationHandler.UpdateViolation)
protected.GET("/inspections/:inspection_id/violations", violationHandler.ListViolationsByInspection)

// Job status routes - new
protected.GET("/jobs/status", jobHandler.GetJobStatus)
protected.POST("/jobs/:job_id/cancel", jobHandler.CancelJob)
```

## Integration Points

### HTMX Integration

All components are designed for HTMX interactions:

1. **violation-card**: Action buttons use `hx-post` with `hx-target` and `hx-swap`
2. **photo-card**: Auto-polls with `hx-trigger="every 3s"` during analysis
3. **job-status**: Self-polling with `hx-get` and `hx-trigger="every 5s"`
4. **Handlers**: Detect HTMX requests via `HX-Request` header and return HTML

### Alpine.js Integration

Components requiring client-side interactivity:

1. **autocomplete-input**: Full Alpine.js component with search, filtering, keyboard nav
2. **photo-card**: Collapsible context input
3. **violation-card**: Optional inline editing mode

### Database Integration

Handlers interact with existing database schema:

- **jobs table**: For job status queries
- **detected_violations table**: For violation CRUD
- **photos table**: For photo deletion
- **safety_codes table**: For autocomplete options

## Testing Checklist

### Component Rendering
- [ ] violation-card renders with all status states
- [ ] photo-card renders with all analysis states
- [ ] job-status shows/hides based on active jobs
- [ ] autocomplete-input loads options correctly
- [ ] breadcrumb handles various depths
- [ ] stats-card displays all icon/color combinations

### Handler Functionality
- [ ] Confirm violation updates status and returns HTML
- [ ] Dismiss violation updates status and returns HTML
- [ ] Set pending violation updates status correctly
- [ ] Delete photo removes photo and violations
- [ ] Job status returns active jobs for organization
- [ ] Cancel job updates job status

### HTMX Interactions
- [ ] Violation actions swap card without page reload
- [ ] Photo analysis polls until complete
- [ ] Job status bar auto-updates
- [ ] Delete photo removes card from DOM

### Authorization
- [ ] All handlers verify organization membership
- [ ] Job handlers check ownership before cancellation
- [ ] Photo deletion checks permissions

## Next Steps

### Immediate
1. Test all new endpoints with curl/Postman
2. Add unit tests for new handlers
3. Test HTMX interactions in browser
4. Verify Alpine.js components work correctly

### Future Enhancements
1. Add storage cleanup to DeletePhoto (async job)
2. Implement photo deletion protection (confirmed violations)
3. Add violation update endpoint for inline editing
4. Enhance job status with more detailed progress
5. Add batch operations for violations
6. Implement keyboard shortcuts for violation review

## Files Modified

1. `/home/dukerupert/Repos/aletheia/web/templates/components/violation-card.html` - Created
2. `/home/dukerupert/Repos/aletheia/web/templates/components/photo-card.html` - Created
3. `/home/dukerupert/Repos/aletheia/web/templates/components/job-status.html` - Created
4. `/home/dukerupert/Repos/aletheia/web/templates/components/autocomplete-input.html` - Created
5. `/home/dukerupert/Repos/aletheia/web/templates/components/breadcrumb.html` - Created
6. `/home/dukerupert/Repos/aletheia/web/templates/components/stats-card.html` - Created
7. `/home/dukerupert/Repos/aletheia/internal/handlers/violations.go` - Updated
8. `/home/dukerupert/Repos/aletheia/internal/handlers/photos.go` - Updated
9. `/home/dukerupert/Repos/aletheia/internal/handlers/jobs.go` - Created
10. `/home/dukerupert/Repos/aletheia/cmd/main.go` - Updated
11. `/home/dukerupert/Repos/aletheia/UX_DESIGN.md` - Created (earlier)
12. `/home/dukerupert/Repos/aletheia/COMPONENT_HANDLERS_SUMMARY.md` - This file

## API Endpoints Summary

### Violations
- `POST /api/violations/:id/confirm` - Confirm violation
- `POST /api/violations/:id/dismiss` - Dismiss violation
- `POST /api/violations/:id/pending` - Set to pending (undo)
- `PATCH /api/violations/:violation_id` - Update violation
- `POST /api/violations/manual` - Create manual violation
- `GET /api/inspections/:inspection_id/violations` - List violations

### Photos
- `DELETE /api/photos/:photo_id` - Delete photo
- `POST /api/photos/analyze` - Trigger AI analysis
- `GET /api/photos/analyze/:job_id` - Check analysis status

### Jobs
- `GET /api/jobs/status` - Get active jobs (polled by UI)
- `POST /api/jobs/:job_id/cancel` - Cancel job

All endpoints require authentication and organization membership verification.
