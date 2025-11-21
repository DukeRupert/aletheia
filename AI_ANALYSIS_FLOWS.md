# AI Photo Analysis - Flow Diagrams

## 1. Complete Photo Analysis Flow

```
INSPECTOR WORKFLOW
==================

Upload Photo
    ↓
[Photo Card Appears]
    ├─ Photo thumbnail
    ├─ Upload timestamp
    ├─ "Add Context (optional)" button
    ├─ "Analyze" button
    └─ "Delete" button
    ↓
[Optional] Add Context
    ├─ Click to expand "Add Context"
    ├─ Type inspector notes (e.g., "Check for missing PPE")
    ↓
Click "Analyze"
    ↓
[Frontend] HTMX POST /api/photos/analyze
    ├─ Sends: photo_id, context (if provided)
    ├─ Sets header: HX-Request=true
    ↓
[Backend] PhotoHandler.AnalyzePhoto()
    ├─ Parse photo_id
    ├─ Verify photo exists
    ├─ Fetch inspection + project
    ├─ Enqueue job to PostgreSQL
    ├─ Return HTML fragment with polling setup
    ↓
[Frontend] Polling Begins (every 2s)
    ├─ HTMX GET /api/photos/analyze/{job_id}
    ├─ Server returns current job status
    ├─ If pending/processing: update spinner, continue polling
    ├─ If failed: show error, offer retry
    ├─ If completed: show violations, stop polling
    ↓
[Backend Worker] Process Job (async)
    ├─ PhotoAnalysisJobHandler.Handle()
    ├─ Fetch photo, inspection, project
    ├─ Get location-specific safety codes
    ├─ Download image from storage
    ├─ Build inspection context + inspector notes
    ├─ Call Claude API: AnalyzePhoto()
    ├─ Claude detects violations
    ├─ Delete pending/dismissed violations (preserve confirmed)
    ├─ Create DetectedViolation records
    ├─ Update job status = completed
    ↓
[Frontend] Results Appear (polling returns violations)
    ├─ Hide loading spinner
    ├─ Show detected violations in cards
    ├─ Each violation shows:
    │  ├─ Severity badge (critical/high/medium/low)
    │  ├─ Status badge (pending)
    │  ├─ Confidence percentage
    │  ├─ Regulation citation (📋 CODE)
    │  ├─ Description
    │  ├─ Location (if detected)
    │  └─ Action buttons (Confirm / Dismiss)
    ├─ Show "Re-analyze" and "Delete" buttons
    ↓
[Inspector Reviews Violations]
    ├─ Reads each violation
    ├─ Makes decision: Confirm or Dismiss
    ↓
Inspector Confirms Violation
    ├─ Clicks "✓ Confirm Violation"
    ├─ HTMX POST /api/violations/{id}/confirm
    ├─ ViolationHandler.ConfirmViolation()
    ├─ Update status = confirmed in database
    ├─ Render violation card with updated state
    │  ├─ Background: light green
    │  ├─ Status badge: confirmed (green)
    │  ├─ Show: "✓ Confirmed by inspector"
    │  └─ Show: "Change to Dismissed" button
    ├─ HTMX swaps out old card with new card
    ↓
Inspector Dismisses Violation
    ├─ Clicks "✗ Dismiss"
    ├─ HTMX POST /api/violations/{id}/dismiss
    ├─ ViolationHandler.DismissViolation()
    ├─ Update status = dismissed in database
    ├─ Render violation card with updated state
    │  ├─ Background: light gray
    │  ├─ Status badge: dismissed (gray)
    │  ├─ Show: "✗ Dismissed by inspector"
    │  └─ Show: "Change to Confirmed" button
    ├─ HTMX swaps out old card with new card
    ↓
[Optional] Add Manual Violation
    ├─ Inspector finds violation AI missed
    ├─ Clicks "Add Manual Violation"
    ├─ Fills form:
    │  ├─ Safety Code/Regulation (required)
    │  ├─ Description (required)
    │  ├─ Severity (required)
    │  └─ Location (optional)
    ├─ Submits via HTMX POST /api/violations/manual
    ├─ ViolationHandler.CreateManualViolation()
    ├─ Create violation with status=confirmed (100% confidence)
    ├─ Page auto-reloads (2s delay)
    ↓
[Optional] Re-analyze Photo
    ├─ Inspector adds new context or wants fresh analysis
    ├─ Scrolls to "Re-analyze Photo" section
    ├─ Types context: "Look for fall protection equipment"
    ├─ Clicks "Re-analyze with Context"
    ├─ Same flow as initial analysis, but:
    │  ├─ Old pending violations are deleted
    │  ├─ Old dismissed violations are deleted
    │  ├─ Confirmed violations are PRESERVED
    │  └─ Fresh Claude analysis runs
    ↓
Inspection Complete
    └─ All violations reviewed and actioned
```

## 2. Job Queue Processing

```
QUEUE LIFECYCLE
===============

REST API Request
    ↓
PhotoHandler.AnalyzePhoto()
    ├─ Validate photo exists
    ├─ Enqueue job:
    │  ├─ Queue name: "photo_analysis"
    │  ├─ Job type: "analyze_photo"
    │  ├─ Organization ID: project.organization_id
    │  ├─ Payload:
    │  │  ├─ photo_id: UUID
    │  │  ├─ inspection_id: UUID
    │  │  └─ context: string (optional)
    │  ├─ Priority: 5 (medium)
    │  └─ Max attempts: 3
    ├─ Job inserted into database:
    │  INSERT INTO jobs (
    │    id, queue_name, job_type, organization_id,
    │    payload, status, priority, max_attempts, created_at
    │  )
    │  VALUES (...)
    ├─ Return job ID to client
    ↓
Worker Pool
    ├─ Continuously polls (every 1 second):
    │  SELECT * FROM jobs
    │  WHERE queue_name = 'photo_analysis'
    │    AND status = 'pending'
    │  ORDER BY priority DESC, created_at ASC
    │  LIMIT 3
    │  FOR UPDATE SKIP LOCKED
    ├─ Picks up 3 pending jobs (worker count = 3)
    ├─ Updates status = 'processing'
    ↓
Job Processing (PhotoAnalysisJobHandler)
    ├─ Extract photo_id from payload
    ├─ Fetch entities from database
    ├─ Download image from storage
    ├─ Call Claude API
    ├─ Process results
    ├─ Create violation records
    ├─ Update job:
    │  ├─ status = 'completed'
    │  ├─ result = { violations_detected, tokens_used, ... }
    │  └─ completed_at = NOW()
    ↓
Success Path
    ├─ Client polling sees completed status
    ├─ Fetches violations from database
    ├─ Renders violation cards
    ├─ Polling stops
    ↓
Failure Path
    ├─ PhotoAnalysisJobHandler throws error
    ├─ Worker catches exception
    ├─ Checks attempt count
    ├─ If attempts < max_attempts:
    │  ├─ Update status = 'pending'
    │  ├─ Increment attempt_count
    │  ├─ Set next run time:
    │  │  ├─ Attempt 1 → retry in 1 minute
    │  │  ├─ Attempt 2 → retry in 2 minutes
    │  │  └─ Attempt 3 → retry in 4 minutes
    │  └─ Job returns to queue
    ├─ If attempts >= max_attempts:
    │  ├─ Update status = 'failed'
    │  ├─ Set error_message
    │  ├─ Client polling sees failed status
    │  └─ Offers "Retry" button
    ↓
Rate Limiting Check
    ├─ Before enqueueing:
    │  ├─ Check organization_rate_limits table
    │  ├─ Count jobs in last hour for org
    │  ├─ If >= limit (default 100):
    │  │  ├─ Return HTTP 429 (Too Many Requests)
    │  │  └─ Client sees rate limit error
    │  ├─ Check concurrent jobs for org
    │  ├─ If >= limit (default 10):
    │  │  ├─ Queue job but mark as deferred
    │  │  └─ Job waits for worker availability
    ↓
Cleanup (Automatic)
    ├─ After 7 days, completed jobs are deleted
    ├─ Failed jobs kept for 7 days (audit trail)
    └─ Manual jobs can be archived on demand
```

## 3. Violation State Machine

```
VIOLATION LIFECYCLE
===================

[Created]
    ↓
AI DETECTION
    ├─ Claude detects violation in photo
    ├─ Create record:
    │  ├─ status = pending
    │  ├─ confidence = 0.75 (AI confidence)
    │  ├─ description = AI description
    │  ├─ severity = AI severity
    │  └─ location = AI location annotation
    ↓
[PENDING] (waiting for inspector review)
    │
    ├─ Inspector Reviews
    │  ├─ Reads description
    │  ├─ Checks confidence score
    │  ├─ Examines photo for accuracy
    │  ├─ Considers severity
    │  ↓
    │  YES (violation is real)
    │  ├─ Click "✓ Confirm Violation"
    │  ├─ HTMX POST /api/violations/{id}/confirm
    │  ├─ Update: status = confirmed
    │  ↓
    │  NO (false positive)
    │  ├─ Click "✗ Dismiss"
    │  ├─ HTMX POST /api/violations/{id}/dismiss
    │  ├─ Update: status = dismissed
    │  ↓
    ├─ Not Yet (needs more context)
    │  └─ Inspector can:
    │     ├─ Re-analyze photo with context
    │     ├─ (Pending violation persists during re-analysis)
    │     ├─ Dismiss it as pending
    │     ├─ Or come back later
    │
    ├─ Photo Re-analyzed
    │  ├─ IF status = pending:
    │  │  └─ DELETE violation (clear for new detection)
    │  └─ IF status = confirmed:
    │     └─ PRESERVE (don't lose inspector's work)
    │
    ├─ Photo Deleted
    │  └─ CASCADE DELETE violation (via foreign key)

[CONFIRMED] (violation is real)
    │
    ├─ Inspector verified the violation exists
    ├─ Used in reports and compliance tracking
    ├─ Counted in violation metrics
    │
    ├─ Inspector Changes Mind
    │  ├─ Click "Change to Dismissed"
    │  ├─ HTMX POST /api/violations/{id}/dismiss
    │  ├─ Update: status = dismissed
    │  ↓

[DISMISSED] (false positive)
    │
    ├─ Inspector determined it's not a real violation
    ├─ Soft-deleted (not hard-deleted)
    ├─ Hidden from most views
    ├─ Preserved in database for audit
    │
    ├─ Photo Re-analyzed
    │  └─ DELETE dismissed violation (give AI fresh chance)
    │
    ├─ Inspector Changes Mind
    │  ├─ Click "Change to Confirmed"
    │  ├─ HTMX POST /api/violations/{id}/confirm
    │  ├─ Update: status = confirmed
    │  ↓

MANUAL CREATION
    ├─ Inspector finds violation AI missed
    ├─ Fills form in "Add Manual Violation"
    ├─ HTMX POST /api/violations/manual
    ├─ ViolationHandler.CreateManualViolation()
    ├─ Create record:
    │  ├─ status = confirmed (auto-confirmed by human)
    │  ├─ confidence = 1.0 (100%, manually added)
    │  ├─ created_by = inspector (audit trail)
    │  └─ severity = inspector-selected
    ├─ Goes directly to [CONFIRMED]
    └─ No review needed (human verified it)
```

## 4. HTMX Polling Diagram

```
FRONTEND POLLING
================

User clicks "Analyze" button
    ↓
Browser detects HTMX:
    ├─ Attribute: hx-post="/api/photos/analyze"
    ├─ Attribute: hx-include="#context-{photoId}"
    ├─ Attribute: hx-vals='{"photo_id": "..."}'
    ├─ Attribute: hx-target="closest .card"
    ├─ Attribute: hx-swap="outerHTML"
    ├─ Adds header: HX-Request: true
    ↓
Server Response (202 Accepted)
    ├─ Returns HTML fragment with polling setup:
    ├─ <div class="card"
    │    hx-get="/api/photos/analyze/{job_id}"
    │    hx-trigger="every 2s"
    │    hx-swap="outerHTML">
    │   ⏳ Analyzing...
    │   </div>
    ↓
HTMX Replaces Card (outerHTML)
    └─ Old card completely replaced with new HTML
    ↓
Polling Starts (every 2 seconds)
    ├─ HTMX GET /api/photos/analyze/{job_id}
    ├─ Server checks job status
    │  ├─ IF pending/processing:
    │  │  └─ Return polling HTML (same as above)
    │  ├─ IF completed:
    │  │  └─ Return violations HTML
    │  └─ IF failed:
    │     └─ Return error HTML
    │
    ├─ HTMX swaps response into card
    │  (outerHTML = entire card gets replaced)
    │
    ├─ If response still has polling attributes:
    │  ├─ Continue polling (every 2s)
    │  └─ Go back to server request
    │
    ├─ If response has NO polling attributes:
    │  └─ Stop polling (job completed)
    │
    ↓
Polling Termination
    ├─ Automatic when:
    │  ├─ Job completes (violations shown)
    │  ├─ Job fails (error shown)
    │  └─ User navigates away
    │
    ├─ HTML sent contains:
    │  ├─ hx-trigger="every 2s" (continues)
    │  └─ OR no polling attrs (stops)
    │
    └─ Prevents infinite polling

VIOLATION CONFIRMATION (after polling stops)
    ├─ User sees violation cards
    ├─ Clicks "✓ Confirm Violation"
    ├─ HTMX POST /api/violations/{id}/confirm
    ├─ Attribute: hx-target="#violation-{id}"
    ├─ Attribute: hx-swap="outerHTML"
    │
    ├─ Server returns updated card HTML:
    │  ├─ New styling (green background)
    │  ├─ New status badge (confirmed)
    │  ├─ New button ("Change to Dismissed")
    │
    ├─ HTMX swaps card innerHTML
    └─ Inspector sees updated state immediately

CONTEXT INCLUSION
    ├─ HTML includes hidden textarea:
    │  <textarea id="context-{photoId}" name="context">...</textarea>
    ├─ HTMX includes this field in request:
    │  hx-include="#context-{photoId}"
    ├─ Request body contains:
    │  ├─ photo_id
    │  └─ context (inspector notes)
    ├─ Server extracts context from form data
    └─ Passes to job payload
```

## 5. Database Relations

```
SCHEMA RELATIONSHIPS
====================

organizations
    ├── PK: id
    └── 1:N ─── organization_members
              ├── PK: id
              ├── FK: organization_id
              └── FK: user_id

    └── 1:N ─── projects
              ├── PK: id
              ├── FK: organization_id
              └── 1:N ─── inspections
                      ├── PK: id
                      ├── FK: project_id
                      └── 1:N ─── photos
                              ├── PK: id
                              ├── FK: inspection_id
                              └── 1:N ─── detected_violations
                                      ├── PK: id
                                      ├── FK: photo_id
                                      ├── FK: safety_code_id (nullable)
                                      ├── status (ENUM)
                                      ├── severity (ENUM)
                                      └── confidence_score

            └── 1:N ─── safety_codes
                        ├── PK: id
                        ├── FK: organization_id
                        ├── code (OSHA 1926.501)
                        └── ← referenced by detected_violations

jobs (global queue)
    ├── PK: id
    ├── FK: organization_id (for rate limiting)
    ├── queue_name ('photo_analysis')
    ├── job_type ('analyze_photo')
    ├── payload (JSON)
    ├── status (pending/processing/completed/failed)
    └── result (JSON)

organization_rate_limits
    ├── organization_id
    ├── queue_name
    ├── jobs_in_hour (count)
    ├── concurrent_jobs (count)
    └── window_start (timestamp)
```

## 6. Error & Retry Flow

```
FAILURE HANDLING
================

Job Starts
    ├─ PhotoAnalysisJobHandler.Handle()
    ├─ Extract photo_id
    ├─ Fetch photo
    ├─ Download image
    ├─ Call Claude API
    │
    └─ ERROR OCCURS
        ├─ Network error
        ├─ Storage service down
        ├─ Claude API error
        ├─ Database error
        └─ etc.
        ↓

Worker Catches Exception
    ├─ Check: attempts < max_attempts (3)?
    │
    ├─ YES (retry possible)
    │  ├─ Update job:
    │  │  ├─ status = 'pending'
    │  │  ├─ attempt_count += 1
    │  │  ├─ next_run_at = NOW() + backoff_time
    │  │  └─ error_message = last error
    │  │
    │  ├─ Backoff schedule:
    │  │  ├─ Attempt 1 fail → wait 1 minute
    │  │  ├─ Attempt 2 fail → wait 2 minutes
    │  │  ├─ Attempt 3 fail → wait 4 minutes
    │  │  └─ Attempt 4 fail → FINAL FAILURE
    │  │
    │  ├─ Job returns to queue
    │  ├─ Worker picks it up again after delay
    │  └─ Repeats processing
    │
    └─ NO (max retries exceeded)
        ├─ Update job:
        │  ├─ status = 'failed'
        │  ├─ error_message = detailed error
        │  ├─ failed_at = NOW()
        │  └─ attempt_count = 3
        │
        ├─ Job stops processing
        │
        └─ Frontend Polling Sees Failure
            ├─ GET /api/photos/analyze/{job_id}
            ├─ Sees status = 'failed'
            ├─ Returns error HTML:
            │  ├─ "❌ Analysis failed"
            │  ├─ "Retry" button (triggers new analysis)
            │  ├─ "Delete" button
            │  └─ Optional: error details
            │
            ├─ Inspector Options:
            │  ├─ Click "Retry" (enqueues new job)
            │  ├─ Click "Delete" (removes photo)
            │  ├─ Add more context (re-analyze)
            │  └─ Try again later
            │
            └─ Polling stops (no more hx-trigger)
```

---

These diagrams show how all components interact in the photo analysis workflow, from initial upload through violation review and the background job processing that powers it all.

