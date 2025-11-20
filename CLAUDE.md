# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Aletheia is a Go-based construction safety inspection platform that uses AI to detect safety violations from photos. The application handles image uploads, stores photos, and tracks safety violations against configurable safety codes.

## Tech Stack

- **Language**: Go 1.25.1
- **Web Framework**: Echo v4
- **Database**: PostgreSQL with pgx/v5 connection pool
- **Templates**: Go `html/template` with HTMX and Alpine.js
- **Storage**: Pluggable interface supporting local filesystem and AWS S3
- **Queue**: Pluggable job queue supporting PostgreSQL (Redis planned for future)
- **Migrations**: Goose (SQL migrations in `internal/migrations/`)
- **Configuration**: Environment variables via godotenv + command-line flags
- **Logging**: slog (JSON in production, text in development)

## Development Commands

### Running the Application

```bash
# Run in development mode
make dev

# Build binary
make build

# The binary will be in bin/myapp
```

### Database Management

```bash
# Start PostgreSQL via Docker Compose
docker-compose up -d

# Run all pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Goose expects GOOSE_DRIVER and GOOSE_DBSTRING environment variables
# These should be set in .env file (see config/config.go for details)
```

### Environment Configuration

Configuration is loaded from `.env` file (required) and can be overridden with command-line flags. Key variables:

- `SERVER_HOST`, `SERVER_PORT` - Server configuration
- `ENVIRONMENT` - "prod" or "dev" (affects logging format and session cookies)
- `LOG_LEVEL` - "debug", "info", "warn", "error"
- `DB_USER`, `DB_PASSWORD`, `DB_HOSTNAME`, `DB_PORT`, `DB_NAME` - PostgreSQL connection
- `JWT_SECRET` - Must be set in production (validation enforced)
- `STORAGE_PROVIDER` - "local" or "s3" (default: "local")
- `STORAGE_LOCAL_PATH` - Path for local storage (default: "./uploads")
- `STORAGE_LOCAL_URL` - Base URL for local storage (default: "http://localhost:1323/uploads")
- `STORAGE_S3_BUCKET` - S3 bucket name (required when using S3)
- `STORAGE_S3_REGION` - S3 region (default: "us-east-1")
- `STORAGE_S3_BASE_URL` - CloudFront or S3 base URL (required when using S3)
- `EMAIL_PROVIDER` - "mock" or "postmark" (default: "mock")
- `EMAIL_FROM_ADDRESS`, `EMAIL_FROM_NAME` - Email sender configuration
- `EMAIL_VERIFY_BASE_URL` - Base URL for verification links
- `QUEUE_PROVIDER` - "postgres" or "redis" (default: "postgres")
- `QUEUE_WORKER_COUNT` - Number of concurrent workers (default: 3)
- `QUEUE_POLL_INTERVAL` - How often to poll for jobs (default: "1s")
- `QUEUE_JOB_TIMEOUT` - Job processing timeout (default: "60s")
- `QUEUE_ENABLE_RATE_LIMITING` - Enable per-organization rate limits (default: true)

## Architecture

### Core Domain Model

The application models construction safety inspections with the following hierarchy:

1. **Organizations** - Companies/entities conducting inspections
   - Has many **Organization Members** (users)
2. **Projects** - Construction sites or buildings being inspected
3. **Inspections** - Specific inspection events at a project
4. **Photos** - Images captured during inspections
5. **Detected Violations** - AI-identified safety issues in photos
   - References **Safety Codes** - Configurable safety standards
6. **Reports** - Generated inspection reports

See migrations in `internal/migrations/` for exact schema.

### Storage Abstraction

The `storage.FileStorage` interface (`internal/storage/storage.go`) provides pluggable storage:

- **LocalStorage**: Development/testing - stores files in `./uploads` directory
- **S3Storage**: Production - uploads to AWS S3 with CloudFront support

Storage is configured via the `STORAGE_PROVIDER` environment variable. The `storage.NewFileStorage()` factory function automatically initializes the appropriate storage implementation based on configuration (similar to the email service pattern). To switch between local and S3 storage, update the `STORAGE_PROVIDER` variable in your `.env` file.

### Queue System

The `queue.Queue` interface (`internal/queue/queue.go`) provides a pluggable job queue for async background processing:

- **PostgresQueue**: Production-ready PostgreSQL implementation using `SELECT FOR UPDATE SKIP LOCKED`
- **RedisQueue**: Planned for high-performance scenarios (not yet implemented)
- **MockQueue**: In-memory implementation for testing

**Key Features:**
- Job priorities and delayed scheduling
- Automatic retry with exponential backoff (1min → 2min → 4min)
- Per-organization rate limiting (hourly quotas + concurrent job limits)
- Worker pool with configurable concurrency
- Job handlers registered by type
- Graceful shutdown support

**Database Tables:**
- `jobs` - Main job queue with status tracking, retry counts, and results
- `organization_rate_limits` - Per-organization rate limiting with sliding window tracking

**Usage Pattern:**
1. Register job handlers with `WorkerPool.RegisterHandler(jobType, handlerFunc)`
2. Start worker pool with `WorkerPool.Start(ctx, queueNames)`
3. Enqueue jobs with `Queue.Enqueue(ctx, queueName, jobType, organizationID, payload, opts)`
4. Workers automatically dequeue, process, and update job status

The queue is configured via `QUEUE_PROVIDER` environment variable. The `queue.NewQueue()` factory function initializes the appropriate implementation. Workers poll for jobs at `QUEUE_POLL_INTERVAL` and process them with registered handlers.

**Common Job Types:**
- `photo_analysis` - Analyze photos for safety violations using AI
- `report_generation` - Generate PDF inspection reports
- `notification_email` - Send email notifications

### Request Flow

**Synchronous Requests:**
1. Echo router receives HTTP request
2. Middleware: Request logger (slog-based)
3. Handler processes request
4. Handler uses storage interface to save files
5. Handler uses pgxpool for database operations
6. Response returned as JSON

**Asynchronous Background Jobs:**
1. Handler enqueues job via `Queue.Enqueue()`
2. Returns immediately with job ID
3. Worker pool continuously polls for pending jobs
4. Worker dequeues job and invokes registered handler
5. Handler processes job (e.g., AI photo analysis)
6. Worker updates job status (completed/failed) with results

### Database Connection

- Uses `pgxpool` for connection pooling (configured in `cmd/main.go:44-49`)
- Pool settings: 25 max connections, 5 min idle, 1 hour max lifetime
- Connection string built via `config.GetConnectionString()`

### Configuration System

Configuration loading priority (lowest to highest):
1. Default values in `config.go`
2. `.env` file variables
3. Command-line flags

The `Config.GetLogger()` method returns appropriate slog handler based on environment (JSON for prod, text for dev).

## Key Implementation Patterns

### Handler Pattern

Handlers are structs with dependencies injected via constructor:

```go
type UploadHandler struct {
    storage storage.FileStorage
    queue   queue.Queue
}

func NewUploadHandler(storage storage.FileStorage, q queue.Queue) *UploadHandler {
    return &UploadHandler{storage: storage, queue: q}
}
```

### Pluggable Services Pattern

Both storage and queue follow a factory pattern for swappable implementations:

```go
// Storage factory (local vs S3)
storage := storage.NewFileStorage(cfg)

// Queue factory (postgres vs redis)
queue := queue.NewQueue(ctx, logger, queueCfg)
```

This pattern allows switching implementations via environment variables without code changes.

### Migration Pattern

Migrations use Goose format with `-- +goose Up` and `-- +goose Down` sections. All migrations are in `internal/migrations/`. CREATE INDEX statements are included in the same migration as table creation.

### Error Handling

- Use `echo.NewHTTPError()` for client-facing errors
- Log internal errors with slog before returning generic error to client
- Validation happens at handler level before calling storage/database

## Project Structure

```
/
├── cmd/main.go              # Application entry point, server setup
├── internal/
│   ├── config/              # Configuration loading and validation
│   ├── handlers/            # HTTP request handlers
│   ├── storage/             # File storage implementations (local, S3)
│   ├── queue/               # Job queue system (postgres, redis, mock)
│   ├── templates/           # Template renderer for Go html/template
│   └── migrations/          # Goose SQL migrations
├── pkg/                     # (Currently empty - future shared packages)
├── uploads/                 # Local file storage directory
├── web/
│   ├── templates/           # HTML templates (layouts, components, pages)
│   └── static/              # CSS, JavaScript, images
├── docker-compose.yaml      # PostgreSQL service
└── Makefile                 # Build and migration commands
```

## Frontend Architecture

The frontend uses a server-side rendering approach with progressive enhancement:

**Template System:**
- Go `html/template` for server-side rendering
- Base layout (`layouts/base.html`) with reusable components
- HTMX for dynamic server interactions without full page reloads
- Alpine.js for lightweight client-side reactivity
- Minimal CSS following semantic HTML-first approach

**Template Structure:**
- `web/templates/layouts/` - Base page layouts
- `web/templates/components/` - Reusable components (nav, forms, etc.)
- `web/templates/pages/` - Individual page templates

**Static Assets:**
- Served at `/static/` route
- CSS in `web/static/css/main.css`
- Images in `web/static/images/`

**Rendering Pattern:**
```go
func (h *Handler) MyPage(c echo.Context) error {
    data := map[string]interface{}{
        "IsAuthenticated": true,
        "User": user,
    }
    return c.Render(http.StatusOK, "mypage.html", data)
}
```

See `web/README.md` for template usage patterns and `STYLE_GUIDE.md` for styling guidelines.

## Important Notes

- Server runs on port 1323 by default (configurable)
- Graceful shutdown implemented with 10-second timeout
- Static files served at `/static/` (CSS, JS, images) and `/uploads/` (user uploads)
- Upload endpoint: `POST /api/upload` (accepts "image" form field)
- Accepted image types: JPEG, PNG, WebP (5MB max)
- Database pool is closed on shutdown

### Queue System Notes

- PostgreSQL queue uses `SELECT FOR UPDATE SKIP LOCKED` for safe concurrent job dequeuing
- Failed jobs automatically retry with exponential backoff: 1min, 2min, 4min, etc.
- Rate limits are per-organization and per-queue with sliding window tracking
- Worker pool gracefully shuts down, waiting for in-flight jobs to complete
- Job handlers should be idempotent (safe to retry) since jobs may be retried on failure
- Use `MockQueue` for testing to avoid database dependencies
- Maximum retry attempts default to 3 but can be configured per job
- Jobs can be scheduled for future execution via `EnqueueOptions.ScheduledAt` or `Delay`
