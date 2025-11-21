# AI Photo Analysis - Quick Reference

## Key Endpoints

```
POST   /api/photos/analyze                    → Trigger analysis
GET    /api/photos/analyze/{job_id}          → Get job status & results
POST   /api/violations/{id}/confirm          → Mark violation as confirmed
POST   /api/violations/{id}/dismiss          → Dismiss violation
POST   /api/violations/manual                → Add manual violation
GET    /api/inspections/{id}/violations      → List inspection violations
```

## Key Templates

```
/web/templates/pages/photo-detail.html       → Full photo review UI
/web/templates/pages/inspection-detail.html  → Grid of photos + analysis
```

## Key Handlers

```
PhotoHandler.AnalyzePhoto()                  → Trigger analysis endpoint
PhotoHandler.GetPhotoAnalysisStatus()        → Polling status endpoint
PhotoAnalysisJobHandler.Handle()             → Background job processor
ViolationHandler.{Confirm,Dismiss}()         → Violation state changes
ViolationHandler.CreateManualViolation()     → Inspector-created violations
```

## Key Database Tables

```
detected_violations          → Violations found by AI or manually created
violation_status            → ENUM: pending, confirmed, dismissed
violation_severity          → ENUM: critical, high, medium, low
safety_codes                → Configurable safety regulations
jobs                        → Queue job tracking
```

## HTMX Flow

```
User clicks "Analyze"
    ↓
POST /api/photos/analyze (returns HTML fragment with polling)
    ↓
HTML includes: hx-get="/api/photos/analyze/{job_id}"
                hx-trigger="every 2s"
                hx-swap="outerHTML"
    ↓
Polling continues until job completes (success/failure)
    ↓
Final HTML rendered with violations or error
```

## Job Processing Flow

```
Enqueue (REST API)
    ↓
Wait in PostgreSQL jobs table
    ↓
Worker pool picks up (every 1s poll)
    ↓
PhotoAnalysisJobHandler processes
    ↓
Claude API analysis
    ↓
Store violations in database
    ↓
Update job status
    ↓
Frontend polling gets results
```

## Violation States

```
pending   → AI detected, awaiting inspector review
   ├─→ Confirm  → confirmed (inspector verified)
   └─→ Dismiss  → dismissed (false positive)

confirmed → Inspector verified real violation
   └─→ Dismiss  → dismissed (inspector changed mind)

dismissed → Inspector marked false positive
   └─→ Confirm → confirmed (inspector changed mind)
```

## Key Features

| Feature | Status | Details |
|---------|--------|---------|
| Photo upload | ✓ | JPEG/PNG/WebP, max 5MB, generates thumbnail |
| AI analysis | ✓ | Claude vision API, asynchronous queue |
| HTMX polling | ✓ | Every 2 seconds, auto-stops on completion |
| Violation review | ✓ | Confirm/dismiss with HTMX updates |
| Manual violations | ✓ | Inspectors add missed violations |
| Context hints | ✓ | Inspector notes included in AI prompt |
| Re-analysis | ✓ | Preserves confirmed, clears pending/dismissed |
| Regulation codes | ✓ | Location-filtered safety codes |
| Severity levels | ✓ | Critical/High/Medium/Low with colors |
| Confidence scores | ✓ | AI confidence 0-100%, displayed as % |
| Location annotations | ✓ | Where in image violation detected |
| Rate limiting | ✓ | Per-org hourly quotas + concurrent limits |

## Violation UI Colors

```
Severity Badges:
  Critical → Red (#dc2626)
  High     → Orange (#f97316)
  Medium   → Yellow (#fbbf24)
  Low      → Gray (#94a3b8)

Status Badges:
  Pending    → Blue (#3b82f6)
  Confirmed  → Green (#059669)
  Dismissed  → Gray (#6b7280)

Card Backgrounds:
  Pending    → Light red (#fef2f2)
  Confirmed  → Light green (#d1fae5)
  Dismissed  → Light gray (#f3f4f6)
```

## Configuration

```
AI_PROVIDER              → "claude" (default: "mock")
QUEUE_PROVIDER           → "postgres" (job queue backend)
QUEUE_WORKER_COUNT       → 3 (concurrent workers)
QUEUE_POLL_INTERVAL      → 1s (check for jobs)
QUEUE_JOB_TIMEOUT        → 60s (max job duration)
QUEUE_ENABLE_RATE_LIMITING → true
STORAGE_PROVIDER         → "local" or "s3"
```

## Inspector Workflow

1. **Upload Photo** → Click "Add Photo" on inspection page
2. **Analyze** → Click "Analyze" button (or add context first)
3. **Wait** → HTMX polls while Claude processes
4. **Review** → Violations appear in cards
5. **Action**:
   - ✓ **Confirm** → Mark as real violation
   - ✗ **Dismiss** → Mark as false positive
   - **Add Manual** → Inspector-identified violation
6. **Re-analyze** → Add context hints and re-run (preserves confirmed)

## Performance Notes

- Photo analysis: 30-60 seconds (depends on AI model)
- Polling overhead: Minimal (2-second intervals, auto-stops)
- Rate limits: 100 analyses/hour per organization
- Concurrent: Max 10 simultaneous jobs per organization
- Job queue: PostgreSQL with SELECT FOR UPDATE SKIP LOCKED
- Retries: 3 attempts with exponential backoff

## Testing Commands

```bash
# View photo detail page
GET /photos/{photo_id}

# Manual violation form
POST /api/violations/manual
  photo_id, safety_code, description, severity, location

# Check violation status
GET /api/inspections/{inspection_id}/violations

# Re-analyze with context
POST /api/photos/analyze
  photo_id, context

# Confirm violation
POST /api/violations/{violation_id}/confirm

# Dismiss violation
POST /api/violations/{violation_id}/dismiss
```

## Files to Review

| File | Purpose |
|------|---------|
| `internal/handlers/photos.go` | Analysis endpoints + polling |
| `internal/handlers/violations.go` | Violation actions + UI rendering |
| `internal/handlers/photo_analysis_job.go` | Background job processor |
| `web/templates/pages/photo-detail.html` | Photo review UI |
| `web/templates/pages/inspection-detail.html` | Inspection overview |
| `internal/ai/ai.go` | AI service interface |
| `internal/ai/claude.go` | Claude API implementation |
| `internal/queue/postgres.go` | Job queue implementation |

