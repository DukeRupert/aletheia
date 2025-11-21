# AI Photo Analysis - File Guide

This document maps all relevant files for the AI photo analysis feature.

## Core Handler Files

### /internal/handlers/photos.go
**Purpose**: HTTP endpoints for triggering and polling photo analysis

**Key Functions**:
- `AnalyzePhoto()` - POST /api/photos/analyze
  - Triggers AI analysis on a photo
  - Enqueues job and returns HTML fragment for HTMX polling
  - Handles optional inspector context
  - 57 lines of endpoint logic

- `GetPhotoAnalysisStatus()` - GET /api/photos/analyze/{job_id}
  - Polls for job completion
  - Returns HTML fragment based on job status (pending/failed/completed)
  - Shows violations when analysis completes
  - Implements HTMX-friendly polling UI

**Lines**: 369 total, with detailed HTMX HTML rendering inline

### /internal/handlers/photo_analysis_job.go
**Purpose**: Background job handler for photo analysis processing

**Key Class**: `PhotoAnalysisJobHandler`
- `Handle()` - Processes analysis jobs from queue
  - Fetches photo, inspection, project from database
  - Gets location-specific safety codes
  - Downloads image from storage
  - Builds inspection context with inspector notes
  - Calls Claude AI API
  - Stores violations in database
  - Handles re-analysis smart violation deletion
  
**Key Feature**: Smart violation handling
- Deletes pending violations on re-analysis (clear for new detection)
- Deletes dismissed violations on re-analysis (give AI another chance)
- Preserves confirmed violations (respects inspector's work)

**Lines**: 223 total, comprehensive error handling and logging

### /internal/handlers/violations.go
**Purpose**: HTTP endpoints for violation review and management

**Key Functions**:
- `ListViolationsByInspection()` - GET /api/inspections/{id}/violations
  - Lists all violations for an inspection
  - Optional status filter (pending/confirmed/dismissed)

- `GetViolation()` - GET /api/violations/{id}
  - Gets single violation details

- `UpdateViolation()` - PATCH /api/violations/{id}
  - Updates violation status or description

- `DeleteViolation()` - DELETE /api/violations/{id}
  - Deletes (soft-delete) a violation

- `ConfirmViolation()` - POST /api/violations/{id}/confirm
  - Marks violation as confirmed by inspector
  - HTMX support: renders updated violation card
  - Status change: pending → confirmed

- `DismissViolation()` - POST /api/violations/{id}/dismiss
  - Marks violation as dismissed (false positive)
  - HTMX support: renders updated violation card
  - Status change: pending → dismissed OR confirmed → dismissed

- `CreateManualViolation()` - POST /api/violations/manual
  - Allows inspectors to manually add violations
  - Auto-confirms (status = confirmed, confidence = 1.0)
  - HTMX support with form submission

- `renderViolationCard()` - Helper function
  - Renders individual violation card with styling
  - Color-coded by severity and status
  - Generates action buttons based on current status
  - Used by confirm/dismiss endpoints

**Lines**: 732 total, extensive UI rendering

### /internal/handlers/upload.go
**Purpose**: Photo upload and listing

**Key Functions**:
- `UploadImage()` - POST /api/upload
  - Validates file (JPEG/PNG/WebP, max 5MB)
  - Saves to storage (local or S3)
  - Generates thumbnail
  - Creates photo record in database
  - Returns HTML fragment with "Analyze" button for HTMX

- `ListPhotos()` - GET /api/inspections/{id}/photos
  - Lists all photos for an inspection
  - Returns JSON array with photo details

- `GetPhoto()` - GET /photos/{id}
  - Retrieves single photo

- `DeletePhoto()` - DELETE /photos/{id}
  - Deletes photo (cascades to violations)

**Lines**: 260+ total

## Template Files

### /web/templates/pages/photo-detail.html
**Purpose**: Full photo review and violation management UI

**Sections**:
1. **Photo Display**
   - Full image with link to open full size
   - Upload timestamp
   - Photo metadata

2. **Detected Violations**
   - List of all violations for photo
   - Violation cards with:
     - Severity badge (critical/high/medium/low)
     - Status badge (pending/confirmed/dismissed)
     - Confidence percentage
     - Safety regulation citation (prominent blue box)
     - Description
     - Location annotation
     - Action buttons (Confirm/Dismiss for pending)

3. **Re-analyze Section**
   - Textarea for additional context
   - "Re-analyze with Context" button
   - Loading spinner animation
   - Completion message
   - Auto-reload on success

4. **Manual Violation Entry**
   - Form with:
     - Safety Code field
     - Description textarea
     - Severity dropdown
     - Location field
   - HTMX form submission
   - Success/error feedback

5. **Styling & Animations**
   - CSS for loading spinner
   - Color coding for severity/status
   - Responsive design

**Lines**: 335 total

### /web/templates/pages/inspection-detail.html
**Purpose**: Inspection overview with photo grid and quick analysis

**Sections**:
1. **Header**
   - Inspection ID
   - Project name and location
   - Inspection status badge

2. **Inspection Details Card**
   - Inspector ID
   - Created date
   - Last updated

3. **Photo Upload Section**
   - File input button
   - Upload status indicator
   - Photo count display
   - File type validation

4. **Photo Grid**
   - Thumbnail preview of each photo
   - Upload timestamp
   - Violation summary for each photo:
     - Number of violations
     - Status of violations
     - Severity and confidence indicators
     - Regulation citations

5. **Inline Analysis Controls** (per photo)
   - Collapsible "Add Context (optional)" section
   - Context textarea
   - "Analyze" button with HTMX
   - "Delete" button

6. **JavaScript Event Handlers**
   - Upload form submit handling
   - Upload status display/hide
   - Form reset on completion

**Lines**: 250 total

## Backend Service Files

### /internal/ai/ai.go
**Purpose**: AI service interface and types

**Types**:
- `AIService` interface
  - `AnalyzePhoto(ctx, request) (*response, error)`
  
- `AnalysisRequest`
  - ImageData: raw image bytes
  - ImageURL: alternative to ImageData
  - SafetyCodes: list of safety code contexts
  - InspectionContext: project/location info
  
- `AnalysisResponse`
  - Violations: detected violation list
  - AnalysisDetails: additional analysis info
  - TokensUsed: API token count

- `SafetyCodeContext`
  - Code: OSHA code (e.g., "1926.501")
  - Description: regulation description
  - Country: jurisdiction

- `DetectedViolation` (from AI)
  - SafetyCode: code string
  - Description: AI-generated description
  - Severity: critical/high/medium/low
  - Confidence: 0.0-1.0
  - Location: where in image

- `ViolationSeverity` enum
  - Critical
  - High
  - Medium
  - Low

- `AIConfig`
  - Provider: "claude", "mock"
  - ClaudeAPIKey, ClaudeModel
  - MaxTokens, Temperature

**Factory**: `NewAIService()` creates provider-specific implementation

### /internal/ai/claude.go
**Purpose**: Claude API integration for photo analysis

**Implementation Details**:
- Uses Claude 3.5 Sonnet model for vision
- Sends image as base64 or URL
- Includes safety codes in system prompt
- Includes inspection context for relevance
- Parses JSON response with violations
- Tracks token usage
- Error handling and logging

**Key**: This is where Claude's vision capabilities are leveraged

### /internal/queue/postgres.go
**Purpose**: PostgreSQL-backed job queue

**Key Methods**:
- `Enqueue()` - Add job to queue
- `Dequeue()` - Get pending jobs
- `UpdateJobStatus()` - Change job state
- `GetJob()` - Fetch job details
- `DeleteJob()` - Remove completed job

**Features**:
- SELECT FOR UPDATE SKIP LOCKED for safe concurrency
- Job polling by status
- Rate limiting checks
- Automatic cleanup

### /internal/queue/worker.go
**Purpose**: Worker pool for processing jobs

**Key Methods**:
- `RegisterHandler()` - Register job type handler
- `Start()` - Start worker pool
- `processJob()` - Execute job with handler
- Retry logic with exponential backoff

**Features**:
- 3 concurrent workers (configurable)
- 1 second poll interval
- 60 second job timeout
- 3 maximum attempts with backoff

## Database Migration Files

### /internal/migrations/20251118165603_create_detected_violations_table.sql
**Purpose**: Initial violations table schema

**Creates**:
- `detected_violations` table
  - id (UUID PK)
  - photo_id (FK to photos)
  - description (TEXT)
  - confidence_score (DECIMAL 5,4)
  - status (violation_status enum)
  - created_at (timestamp)

- `violation_status` enum
  - pending
  - confirmed
  - dismissed

**Indexes**:
- photo_id (for querying by photo)
- status (for filtering)

### /internal/migrations/20251118174247_add_safety_code_to_detected_violations.sql
**Purpose**: Link violations to safety codes

**Adds**:
- safety_code_id (UUID FK to safety_codes)

**Allows**:
- Match violations to regulation codes
- Display code in UI
- Filter by regulation

### /internal/migrations/20251120024523_add_severity_location_to_detected_violations.sql
**Purpose**: Add severity and location fields

**Creates**:
- `violation_severity` enum
  - critical
  - high
  - medium
  - low

**Adds Columns**:
- severity (enum, default: medium)
- location (TEXT, nullable)

**Indexes**:
- severity (for filtering by severity)

**Purpose**: Enables color-coded UI and location annotations

## Configuration Files

### /internal/config/config.go
**Purpose**: Configuration loading and defaults

**Key Config**:
- AI_PROVIDER (claude, mock)
- QUEUE_PROVIDER (postgres)
- QUEUE_WORKER_COUNT (3)
- QUEUE_POLL_INTERVAL (1s)
- QUEUE_JOB_TIMEOUT (60s)
- QUEUE_ENABLE_RATE_LIMITING (true)
- STORAGE_PROVIDER (local, s3)

## Main Application File

### /cmd/main.go
**Purpose**: Application initialization and setup

**Key Sections**:
- Database pool configuration
- Service initialization
  - Storage (local/S3)
  - Email service
  - AI service (Claude/mock)
  - Queue (PostgreSQL)
  - Template renderer

- Worker pool setup
  - Register photo analysis job handler
  - Start 3 concurrent workers
  - Poll "photo_analysis" queue

- Route registration
  - POST /api/photos/analyze
  - GET /api/photos/analyze/{job_id}
  - POST /api/violations/{id}/confirm
  - POST /api/violations/{id}/dismiss
  - POST /api/violations/manual

- Server startup and graceful shutdown

**Lines**: 334 total

## Database Query Files (Auto-generated)

### /internal/database/detected_violations.sql.go
**Auto-generated** from SQL queries

**Functions**:
- `CreateDetectedViolation()` - Insert new violation
- `GetDetectedViolation()` - Fetch by ID
- `ListDetectedViolations()` - List by photo
- `ListDetectedViolationsByInspection()` - List by inspection
- `ListDetectedViolationsByInspectionAndStatus()` - Filter by status
- `UpdateDetectedViolationStatus()` - Change status
- `UpdateDetectedViolationNotes()` - Update description
- `DeleteDetectedViolation()` - Soft-delete
- `DeletePendingAndDismissedViolationsByPhoto()` - Re-analysis cleanup

### /internal/database/safety_codes.sql.go
**Auto-generated** from SQL queries

**Functions**:
- `CreateSafetyCode()`
- `GetSafetyCode()` - Fetch by ID
- `GetSafetyCodeByCode()` - Fetch by code string
- `ListSafetyCodes()` - All codes
- `ListSafetyCodesByLocation()` - Filtered by state/country
- `UpdateSafetyCode()`
- `DeleteSafetyCode()`

## Testing Files

### /internal/handlers/photos_test.go
**Purpose**: Unit tests for photo analysis endpoints

**Test Functions**:
- `TestAnalyzePhoto()` - Test job enqueueing
- `TestAnalyzePhoto_InvalidPhotoID()` - Validation
- `TestAnalyzePhoto_PhotoNotFound()` - Error handling
- `TestGetPhotoAnalysisStatus()` - Polling
- Multiple status code tests

### /internal/handlers/violations_test.go
**Purpose**: Unit tests for violation endpoints

**Test Functions**:
- Tests for confirm, dismiss, manual creation
- Status validation
- Database updates

## Summary File Tree

```
Aletheia (AI Photo Analysis Implementation)
├── cmd/
│   └── main.go                                    (App initialization, routes, worker setup)
│
├── internal/
│   ├── handlers/
│   │   ├── photos.go                             (Analyze & polling endpoints)
│   │   ├── photo_analysis_job.go                 (Background job processor)
│   │   ├── violations.go                         (Violation CRUD & UI rendering)
│   │   ├── upload.go                             (Photo upload)
│   │   ├── photos_test.go                        (Photo endpoint tests)
│   │   └── violations_test.go                    (Violation endpoint tests)
│   │
│   ├── ai/
│   │   ├── ai.go                                 (AI interface & types)
│   │   ├── claude.go                             (Claude API implementation)
│   │   └── mock.go                               (Mock AI for testing)
│   │
│   ├── queue/
│   │   ├── queue.go                              (Queue interface)
│   │   ├── postgres.go                           (PostgreSQL queue implementation)
│   │   ├── worker.go                             (Worker pool)
│   │   ├── postgres_test.go                      (Queue tests)
│   │   └── worker_test.go                        (Worker tests)
│   │
│   ├── database/
│   │   ├── detected_violations.sql.go            (Violation queries)
│   │   ├── safety_codes.sql.go                   (Safety code queries)
│   │   └── [other auto-generated query files]
│   │
│   ├── migrations/
│   │   ├── 20251118165603_create_detected_violations_table.sql
│   │   ├── 20251118174247_add_safety_code_to_detected_violations.sql
│   │   ├── 20251120024523_add_severity_location_to_detected_violations.sql
│   │   └── [other migrations]
│   │
│   ├── config/
│   │   └── config.go                             (Configuration loading)
│   │
│   ├── storage/
│   │   ├── storage.go                            (Storage interface)
│   │   ├── local.go                              (Local filesystem storage)
│   │   └── s3.go                                 (S3 storage)
│   │
│   └── session/
│       └── session.go                            (User session management)
│
└── web/
    └── templates/
        └── pages/
            ├── photo-detail.html                 (Full photo review UI)
            ├── inspection-detail.html            (Inspection overview)
            ├── [other page templates]
            └── components/
                └── [reusable components]
```

## Statistics

- **Main Analysis Handlers**: 3 files (photos.go, violations.go, photo_analysis_job.go)
- **Total Handler Code**: ~1,300+ lines
- **Templates**: 2 main files (photo-detail, inspection-detail)
- **Total Template Code**: ~585 lines
- **Database Migrations**: 3 violation-specific migrations
- **AI/Queue Services**: 5+ core files
- **Configuration**: Environment-driven

## Entry Points

1. **Frontend**: `/photos/{id}` - Photo detail page or `/inspections/{id}` - Inspection overview
2. **REST API**: `POST /api/photos/analyze` - Trigger analysis
3. **Background**: Worker pool starts on app boot, continuously polls job queue
4. **Manual**: `POST /api/violations/manual` - Inspector-created violations

