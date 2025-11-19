# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Aletheia is a Go-based construction safety inspection platform that uses AI to detect safety violations from photos. The application handles image uploads, stores photos, and tracks safety violations against configurable safety codes.

## Tech Stack

- **Language**: Go 1.25.1
- **Web Framework**: Echo v4
- **Database**: PostgreSQL with pgx/v5 connection pool
- **Storage**: Pluggable interface supporting local filesystem and AWS S3
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

### Request Flow

1. Echo router receives HTTP request
2. Middleware: Request logger (slog-based)
3. Handler (e.g., `UploadHandler`) processes request
4. Handler uses storage interface to save files
5. Handler uses pgxpool for database operations
6. Response returned as JSON

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
}

func NewUploadHandler(storage storage.FileStorage) *UploadHandler {
    return &UploadHandler{storage: storage}
}
```

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
│   └── migrations/          # Goose SQL migrations
├── pkg/                     # (Currently empty - future shared packages)
├── uploads/                 # Local file storage directory
├── web/                     # (Currently empty - future frontend)
├── docker-compose.yaml      # PostgreSQL service
└── Makefile                 # Build and migration commands
```

## Important Notes

- Server runs on port 1323 by default (configurable)
- Graceful shutdown implemented with 10-second timeout
- Static file serving enabled at `/uploads` route
- Upload endpoint: `POST /api/upload` (accepts "image" form field)
- Accepted image types: JPEG, PNG, WebP (5MB max)
- Database pool is closed on shutdown
