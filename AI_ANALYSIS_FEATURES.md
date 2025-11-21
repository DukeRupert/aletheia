# AI Photo Analysis Features - Implementation Summary

## Overview
Aletheia has a comprehensive AI-powered photo analysis system that detects construction safety violations using Claude's vision API. The system is fully integrated with asynchronous job processing, HTMX for real-time UI updates, and violation management workflows.

---

## 1. HTTP ENDPOINTS FOR TRIGGERING ANALYSIS

### Primary Analysis Endpoints

#### POST `/api/photos/analyze`
- **Purpose**: Triggers AI analysis on a photo
- **Request Body**:
  ```json
  {
    "photo_id": "uuid",
    "context": "optional inspector notes"
  }
  ```
- **Response**: 
  - HTTP 202 (Accepted) for JSON requests
  - Returns job ID for tracking
  - For HTMX requests: Returns HTML fragment with polling setup
- **Location**: `/home/dukerupert/Repos/aletheia/internal/handlers/photos.go:57`

#### GET `/api/photos/analyze/{job_id}`
- **Purpose**: Get photo analysis job status and results
- **Response**: Job status, violations (if completed), or error details
- **HTMX Integration**: Returns updated HTML with polling continuation or final results
- **Location**: `/home/dukerupert/Repos/aletheia/internal/handlers/photos.go:205`

### Related Violation Management Endpoints

#### POST `/api/violations/{id}/confirm`
- **Purpose**: Mark a violation as confirmed by inspector
- **HTMX Support**: Returns updated violation card with visual state change
- **Location**: `/home/dukerupert/Repos/aletheia/internal/handlers/violations.go:310`

#### POST `/api/violations/{id}/dismiss`
- **Purpose**: Dismiss a violation (soft delete, treated as false positive)
- **HTMX Support**: Returns updated violation card showing dismissed status
- **Location**: `/home/dukerupert/Repos/aletheia/internal/handlers/violations.go:368`

#### POST `/api/violations/manual`
- **Purpose**: Allow inspectors to manually create violations the AI missed
- **Request Fields**: photo_id, safety_code, description, severity, location (optional)
- **Auto-Confirms**: Manual violations start with "confirmed" status (100% confidence)
- **Location**: `/home/dukerupert/Repos/aletheia/internal/handlers/violations.go:606`

#### GET `/api/inspections/{inspection_id}/violations`
- **Purpose**: List all violations for an inspection
- **Query Parameters**: `status` (optional - filter by pending/confirmed/dismissed)
- **Location**: `/home/dukerupert/Repos/aletheia/internal/handlers/violations.go:54`

---

## 2. FRONTEND TEMPLATES & UI COMPONENTS

### Photo Detail Page
**File**: `/home/dukerupert/Repos/aletheia/web/templates/pages/photo-detail.html`

Features:
- **Photo Display Section**: Shows full image with upload timestamp
- **Violations Panel**: Displays all detected violations with:
  - Severity badge (critical/high/medium/low with color coding)
  - Status badge (pending/confirmed/dismissed)
  - Confidence score as percentage
  - Violation description
  - Location annotation (if provided)
  - Safety regulation citation (prominent blue box with code)
  - Action buttons (Confirm/Dismiss) for pending violations
  
- **Re-analyze Section**:
  - Textarea for inspector to add context hints
  - "Re-analyze with Context" button with loading state
  - Status indicators for analysis progress
  - Auto-reload on completion
  
- **Manual Violation Entry Form**:
  - Safety Code/Regulation field
  - Description textarea
  - Severity dropdown (critical/high/medium/low)
  - Location field (optional)
  - Form submission with HTMX
  - Success/error message display

### Inspection Detail Page
**File**: `/home/dukerupert/Repos/aletheia/web/templates/pages/inspection-detail.html`

Features:
- **Photo Grid Display**:
  - Thumbnail previews of all photos
  - Photo timestamps
  - Violation summary cards for each photo showing:
    - Number of violations
    - Status of most recent violation
    - Regulation citations
    - Severity and confidence indicators
  
- **Inline Analysis Controls**:
  - Collapsible "Add Context (optional)" section for each photo
  - Context textarea for inspector guidance
  - "Analyze" button integrated with HTMX
  - "Delete" button for photo removal
  - Context is passed to analysis job

- **Photo Upload Section**:
  - File input for adding new photos
  - Upload status indicator with spinner
  - Photos added to grid on completion

---

## 3. HTMX INTEGRATION FOR PHOTO ANALYSIS WORKFLOWS

### Analysis Polling System

**Location**: `/home/dukerupert/Repos/aletheia/internal/handlers/photos.go:160-192` (initial response)
**Continuation**: `/home/dukerupert/Repos/aletheia/internal/handlers/photos.go:224-368` (status checks)

#### HTML Fragment Pattern:
```html
<div class="card"
  hx-get="/api/photos/analyze/{job_id}"
  hx-trigger="every 2s"
  hx-swap="outerHTML">
  <!-- Loading/status indicator -->
</div>
```

**How it works**:
1. User clicks "Analyze" button
2. HTMX detects request and includes photo_id + context
3. Server enqueues job and returns HTML fragment with polling
4. Fragment polls every 2 seconds for job status
5. Polling continues until job completes (success/failure)
6. On completion, polling stops and final HTML is rendered

#### States Rendered:

**Pending/Processing**:
- Blue background box
- "⏳ Analyzing photo for safety violations..." message
- Continues polling

**Failed**:
- Red background box
- "❌ Analysis failed: [error message]" 
- "Retry" button (re-submits analysis)
- "Delete" button

**Completed Successfully**:
- Shows all detected violations
- Each violation can be confirmed or dismissed
- "Re-analyze" button to run again
- "Delete" photo button

### HTMX Attributes Used:

- `hx-post="/api/photos/analyze"` - Trigger analysis
- `hx-get="/api/photos/analyze/{job_id}"` - Poll for status
- `hx-trigger="every 2s"` - Poll every 2 seconds
- `hx-swap="outerHTML"` - Replace entire card
- `hx-target="closest .card"` - Target the photo card
- `hx-include="#context-{photoId}"` - Include context field
- `hx-vals='{"photo_id": "..."}'` - Pass photo ID
- `hx-confirm="..."` - Confirm before delete
- `hx-post="/api/violations/{id}/confirm"` - Confirm violation
- `hx-post="/api/violations/{id}/dismiss"` - Dismiss violation

### Client-Side Event Listeners:
**File**: `/home/dukerupert/Repos/aletheia/web/templates/pages/photo-detail.html:307-330`

- `htmx:beforeRequest` - Show analysis status spinner
- `htmx:afterRequest` - Hide spinner, show completion message
- Auto-reload page after 2 seconds on success
- Clears context textarea after submission

---

## 4. VIOLATION REVIEW & MANAGEMENT UI

### Violation Card Rendering
**Location**: `/home/dukerupert/Repos/aletheia/internal/handlers/violations.go:426-583`

Violation Card Features:
- **Severity Badge**:
  - Critical: Red (#dc2626)
  - High: Orange (#f97316)
  - Medium: Yellow (#fbbf24)
  - Low: Gray (#94a3b8)

- **Status Badge**:
  - Pending: Blue (#3b82f6)
  - Confirmed: Green (#059669)
  - Dismissed: Gray (#6b7280)

- **Dynamic Background**:
  - Pending: Light red (#fef2f2)
  - Confirmed: Light green (#d1fae5)
  - Dismissed: Light gray (#f3f4f6)

- **Left Border Color**: Changes by status (confirmed/dismissed) or severity (pending)

### Violation Action Buttons

**For Pending Violations**:
- "✓ Confirm Violation" - Mark as confirmed (green)
- "✗ Dismiss" - Dismiss as false positive (gray)

**For Confirmed Violations**:
- "Change to Dismissed" - Allows undoing confirmation

**For Dismissed Violations**:
- "Change to Confirmed" - Allows reversing dismissal

### HTMX Violation Actions:
```html
<button hx-post="/api/violations/{id}/confirm"
        hx-target="#violation-{id}"
        hx-swap="outerHTML">
  ✓ Confirm Violation
</button>
```

---

## 5. BACKGROUND JOB PROCESSING

### Photo Analysis Job Handler
**Location**: `/home/dukerupert/Repos/aletheia/internal/handlers/photo_analysis_job.go`

**Job Type**: `analyze_photo`
**Queue Name**: `photo_analysis`

#### Processing Steps:
1. Extract photo ID from job payload
2. Fetch photo, inspection, and project from database
3. Get location-specific safety codes (filtered by state/country)
4. Download image from storage service
5. Build rich inspection context (project info, inspector notes)
6. Call Claude AI API with image + safety codes + context
7. Delete pending/dismissed violations from previous analysis (soft delete pattern)
8. Create new DetectedViolation records with results
9. Return job result with violation count and token usage

**Key Features**:
- Handles optional inspector context in payload
- Filters safety codes by project location
- Preserves confirmed violations when re-analyzing
- Clears only pending and dismissed violations on new analysis
- Logs detailed analysis metrics
- Graceful error handling with logging

### Worker Pool Configuration
**Location**: `/home/dukerupert/Repos/aletheia/cmd/main.go:183-193`

```go
workerPool := queue.NewWorkerPool(queueService, logger, queueConfig)
photoAnalysisJobHandler := handlers.NewPhotoAnalysisJobHandler(queries, aiService, fileStorage, logger)
workerPool.RegisterHandler("analyze_photo", photoAnalysisJobHandler.Handle)
go workerPool.Start(context.Background(), []string{"photo_analysis"})
```

**Queue Configuration**:
- Provider: PostgreSQL
- Worker Count: 3 concurrent workers
- Poll Interval: 1 second
- Job Timeout: 60 seconds
- Rate Limiting: Enabled (per-organization hourly quotas)
- Max Jobs Per Hour: 100 (configurable)
- Max Concurrent Jobs: 10 (per-organization)

---

## 6. VIOLATION DATA MODEL

### Database Schema
**File**: `/home/dukerupert/Repos/aletheia/internal/migrations/20251118165603_create_detected_violations_table.sql`

```sql
CREATE TABLE detected_violations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    photo_id UUID NOT NULL REFERENCES photos(id),
    description TEXT NOT NULL,
    confidence_score DECIMAL(5,4),
    status violation_status DEFAULT 'pending' NOT NULL,
    safety_code_id UUID REFERENCES safety_codes(id),
    severity violation_severity NOT NULL DEFAULT 'medium',
    location TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TYPE violation_status AS ENUM ('pending', 'confirmed', 'dismissed');
CREATE TYPE violation_severity AS ENUM ('critical', 'high', 'medium', 'low');
```

### Violation Status Meanings:
- **pending**: AI detected, awaiting inspector review
- **confirmed**: Inspector verified the violation is real
- **dismissed**: Inspector identified as false positive (soft-deleted)

### Violation Severity:
- **critical**: Immediate danger, must stop work
- **high**: Significant risk, requires quick remediation
- **medium**: Notable issue, should be addressed
- **low**: Minor concern, can be logged

---

## 7. AI SERVICE INTEGRATION

### AI Interface
**Location**: `/home/dukerupert/Repos/aletheia/internal/ai/ai.go`

```go
type AIService interface {
    AnalyzePhoto(ctx context.Context, request AnalysisRequest) (*AnalysisResponse, error)
}

type AnalysisRequest struct {
    ImageData []byte
    SafetyCodes []SafetyCodeContext
    InspectionContext string
}

type AnalysisResponse struct {
    Violations []DetectedViolation
    AnalysisDetails string
    TokensUsed int
}
```

### Claude Implementation Details
- Provider: Claude API (claude-3-5-sonnet-20241022)
- Image Processing: Vision API support
- Token Tracking: Logs API token usage for billing
- Context Inclusion: Supports custom inspector guidance
- Safety Code Filtering: Location-aware (state/country)

---

## 8. INSPECTOR CONTEXT FEATURE

### How It Works:
1. Inspector sees "Add Context (optional)" section
2. Types guidance like: "Look for missing hard hats near scaffolding on the right side"
3. Context is included in job payload when analysis is triggered
4. PhotoAnalysisJobHandler appends context to inspection context string
5. Claude receives enhanced context and uses it for detection

**Example Context Inclusion**:
```
Construction site inspection at project: My Building Site, Location: 123 Main St, New York, NY
Project Type: Commercial Office
Inspector's Additional Context: Check for missing fall protection equipment on the upper floors
```

---

## 9. RE-ANALYSIS WORKFLOW

### Smart Violation Handling on Re-Analysis
**Location**: `/home/dukerupert/Repos/aletheia/internal/handlers/photo_analysis_job.go:149-164`

When a photo is re-analyzed:
1. **Pending violations are deleted** - Clears old unreviewed detections
2. **Dismissed violations are deleted** - Treats dismissals as temporary, gives AI another chance
3. **Confirmed violations are preserved** - Respects inspector's verification work

This allows:
- Adding context and getting fresh AI detections
- Inspector's confirmed violations remain unchanged
- Re-running analysis to catch things that were previously dismissed

---

## 10. MANUAL VIOLATION CREATION

### Inspector Workflow
**Endpoint**: `POST /api/violations/manual`
**Location**: `/home/dukerupert/Repos/aletheia/internal/handlers/violations.go:606`

Process:
1. Inspector identifies violation AI missed
2. Opens "Add Manual Violation" form
3. Fills in:
   - Safety Code (e.g., OSHA 1926.501)
   - Description of violation
   - Severity level
   - Location annotation (optional)
4. Submits via HTMX
5. Violation created with:
   - Status: **confirmed** (100% confidence)
   - Confidence: 1.0 (manually added by human)

Benefits:
- Auditable record of manual additions
- Automatically confirmed (no need for extra review)
- Preserves inspector expertise when AI misses obvious issues

---

## 11. VIOLATION CONFIDENCE & SCORES

### Confidence Display:
- Shown as percentage (e.g., "95% confidence")
- Stored as DECIMAL(5,4) for precision
- AI provides 0-1 range, multiplied by 100 for display
- Manual violations always show 100% confidence

### Confidence Use Cases:
- Helps inspectors prioritize review
- Filter pending violations by confidence
- Track AI model performance

---

## 12. REGULATION CITATIONS

### Display in Templates:
- **Prominent Blue Box**: Safety code displayed in monospace font
- **Both Photo Views**: Shows on photo cards and detailed violation cards
- **Format**: "📋 OSHA 1926.501(b)(1)"
- **Linked to Database**: Safety codes are actual database records with descriptions

**Template Code** (photo-detail.html:78-85):
```html
{{if .SafetyCodeID.Valid}}
  <div style="background: #1e3a8a; border-radius: 4px;">
    <p style="font-size: 0.75rem; color: rgba(255,255,255,0.8);">Regulation Violated</p>
    <p style="color: white; font-weight: 700; font-family: monospace;">
      📋 {{index $.SafetyCodeMap (.SafetyCodeID.String)}}
    </p>
  </div>
{{end}}
```

---

## 13. LOCATION ANNOTATIONS

### Feature:
- AI detects where in image violation occurred
- Examples: "Upper left corner near scaffolding", "Right side, between workers"
- Inspector can also add location when creating manual violations
- Shows as "📍 Location: [description]" in violation cards

### Database:
- Stored as nullable TEXT field in detected_violations table
- Included in violation responses and displays

---

## 14. UPLOAD AND INITIAL PHOTO STATE

### Photo Upload Flow
**Endpoint**: `POST /api/upload`
**Location**: `/home/dukerupert/Repos/aletheia/internal/handlers/upload.go:40`

Process:
1. File validation (JPEG/PNG/WebP, max 5MB)
2. Save to storage (local filesystem or S3)
3. Generate thumbnail
4. Create photo record in database
5. Return HTML fragment with "Analyze" button

### Initial State:
- Photos start with **no violations**
- Inspector must explicitly click "Analyze" button
- Analysis is **optional, asynchronous**
- Photos can be deleted anytime

---

## 15. QUEUE SYSTEM FOR ANALYSIS

### Architecture
**Files**: 
- `/home/dukerupert/Repos/aletheia/internal/queue/postgres.go`
- `/home/dukerupert/Repos/aletheia/internal/queue/worker.go`

### Job Lifecycle:
1. POST `/api/photos/analyze` enqueues job
2. Worker pool polls for pending jobs every 1 second
3. Worker picks up job and executes handler
4. Handler processes image with Claude AI
5. Job status updated (pending → processing → completed/failed)
6. Frontend polling retrieves results

### Retry Strategy:
- Failed jobs retry with exponential backoff
- Default 3 maximum attempts
- Backoff: 1min → 2min → 4min

### Rate Limiting:
- Per-organization hourly quotas
- Default: 100 analyses per hour
- Concurrent: Max 10 simultaneous jobs per organization
- Sliding window tracking

---

## SUMMARY OF IMPLEMENTED FEATURES

### Fully Implemented:
- [x] Photo upload with validation and thumbnail generation
- [x] Asynchronous AI analysis via job queue
- [x] Real-time polling with HTMX (every 2 seconds)
- [x] Violation detection with severity levels
- [x] Confidence scoring (AI-provided percentages)
- [x] Location annotations for violations
- [x] Violation status workflow (pending → confirmed/dismissed)
- [x] Inspector context hints for re-analysis
- [x] Manual violation creation
- [x] Soft-delete for dismissed violations (preserved on re-analysis)
- [x] Smart re-analysis (preserves confirmed, clears pending/dismissed)
- [x] Regulation citations linked to safety codes
- [x] Location-specific safety code filtering
- [x] Multiple views (inspection detail, photo detail)
- [x] HTMX forms for seamless interaction
- [x] Job status tracking and error handling
- [x] Token usage logging
- [x] Rate limiting per organization
- [x] Graceful polling termination

### Architecture Highlights:
- Clean handler/service separation
- Pluggable storage (local/S3) and AI (Claude/mock) backends
- PostgreSQL job queue with worker pool
- Template-driven HTML responses
- Progressive enhancement with HTMX
- Comprehensive error logging with slog
- Session-based authentication enforcement

