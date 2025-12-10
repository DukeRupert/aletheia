# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Aletheia is a Go-based construction safety inspection platform that uses AI to detect safety violations from photos. The application handles image uploads, stores photos, and tracks safety violations against configurable safety codes.

## Tech Stack

- **Language**: Go 1.25.1
- **Web Framework**: Echo v4
- **Database**: PostgreSQL with pgx/v5 connection pool
- **ORM**: sqlc for type-safe SQL queries
- **Templates**: Go `html/template` with HTMX and Alpine.js
- **Storage**: Pluggable interface supporting local filesystem and AWS S3
- **Queue**: Pluggable job queue supporting PostgreSQL
- **Migrations**: Goose (SQL migrations in `internal/migrations/`)
- **Configuration**: Environment variables loaded in `cmd/aletheiad/config.go`
- **Logging**: slog (JSON in production, text in development)

## Development Commands

### Running the Application

```bash
# Run in development mode
make dev

# Build binary
make build

# The binary will be in bin/aletheiad
```

### Database Management

```bash
# Start PostgreSQL via Docker Compose
docker-compose up -d

# Run all pending migrations
make migrate-up

# Rollback last migration
make migrate-down
```

### Environment Configuration

Configuration is loaded from environment variables. Key variables:

- `SERVER_HOST`, `SERVER_PORT` - Server configuration
- `ENVIRONMENT` - "prod" or "dev" (affects logging format and session cookies)
- `LOG_LEVEL` - "debug", "info", "warn", "error"
- `DB_USER`, `DB_PASSWORD`, `DB_HOSTNAME`, `DB_PORT`, `DB_NAME` - PostgreSQL connection
- `JWT_SECRET` - Must be set in production (validation enforced)
- `STORAGE_PROVIDER` - "local" or "s3" (default: "local")
- `STORAGE_LOCAL_PATH` - Path for local storage (default: "./uploads")
- `STORAGE_LOCAL_URL` - Base URL for local storage
- `STORAGE_S3_BUCKET`, `STORAGE_S3_REGION`, `STORAGE_S3_BASE_URL` - S3 configuration
- `EMAIL_PROVIDER` - "mock" or "postmark" (default: "mock")
- `AI_PROVIDER` - "mock" or "claude" (default: "mock")
- `QUEUE_PROVIDER` - "postgres" (default: "postgres")

## Architecture

### Domain-First Design

The codebase follows a domain-first architecture where:
- **Domain types and interfaces** live in the root `aletheia` package
- **Implementations** live in separate packages (`postgres/`, `http/`, `mock/`)
- **Transport layer** maps domain errors to HTTP status codes

### Project Structure

```
aletheia/                    # Root: domain types & service interfaces
├── user.go                  # User type and UserService interface
├── organization.go          # Organization types and service interface
├── project.go               # Project types and service interface
├── inspection.go            # Inspection types and service interface
├── photo.go                 # Photo types and service interface
├── violation.go             # Violation types and service interface
├── safety_code.go           # SafetyCode types and service interface
├── session.go               # Session types and service interface
├── error.go                 # Domain error codes (ENOTFOUND, EINVALID, etc.)
├── context.go               # Context helpers for user/org/session
├── storage.go               # FileStorage interface
├── email.go                 # EmailService interface
├── ai.go                    # AIService interface
├── queue.go                 # Queue interface and Job type
│
├── cmd/aletheiad/           # Application entry point
│   ├── main.go              # run() pattern for testability
│   ├── config.go            # Configuration loading
│   └── services.go          # Service initialization
│
├── postgres/                # PostgreSQL service implementations
│   ├── postgres.go          # DB struct with all services
│   ├── user.go              # UserService implementation
│   ├── organization.go      # OrganizationService implementation
│   ├── project.go           # ProjectService implementation
│   ├── inspection.go        # InspectionService implementation
│   ├── photo.go             # PhotoService implementation
│   ├── violation.go         # ViolationService implementation
│   ├── safety_code.go       # SafetyCodeService implementation
│   ├── session.go           # SessionService implementation
│   ├── storage.go           # FileStorage implementations (local, S3)
│   ├── email.go             # EmailService implementations
│   ├── ai.go                # AIService implementations
│   ├── queue.go             # Queue implementation
│   └── convert.go           # sqlc <-> domain type conversions
│
├── http/                    # HTTP transport layer
│   ├── server.go            # Server struct with all dependencies
│   ├── routes.go            # ALL routes in one file
│   ├── handlers.go          # Common helpers (decode, respond)
│   ├── errors.go            # Error code -> HTTP status mapping
│   ├── middleware.go        # Authentication, rate limiting
│   ├── auth.go              # Auth handlers (login, register, etc.)
│   ├── organization.go      # Organization handlers
│   ├── project.go           # Project handlers
│   ├── inspection.go        # Inspection handlers
│   ├── photo.go             # Photo handlers
│   ├── violation.go         # Violation handlers
│   └── safety_code.go       # Safety code handlers
│
├── mock/                    # Test mocks for all services
│   ├── user.go
│   ├── organization.go
│   ├── session.go
│   ├── storage.go
│   ├── email.go
│   ├── ai.go
│   ├── queue.go
│   └── ...
│
├── internal/
│   ├── database/            # sqlc-generated code
│   ├── migrations/          # Goose SQL migrations
│   ├── templates/           # Template renderer
│   ├── auth/                # Password hashing utilities
│   └── validation/          # Input validation
│
├── web/
│   ├── templates/           # HTML templates
│   └── static/              # CSS, JavaScript, images
│
└── uploads/                 # Local file storage
```

### Core Domain Model

1. **Organizations** - Companies/entities conducting inspections
   - Has many **Organization Members** (users with roles)
2. **Projects** - Construction sites or buildings being inspected
3. **Inspections** - Specific inspection events at a project
4. **Photos** - Images captured during inspections
5. **Violations** - AI-identified or manual safety issues in photos
   - References **Safety Codes** - Configurable safety standards

### Service Interfaces

All service interfaces are defined in the root package:

```go
// Root package defines interfaces
type UserService interface {
    FindUserByID(ctx context.Context, id uuid.UUID) (*User, error)
    CreateUser(ctx context.Context, user *User, password string) error
    // ...
}

// postgres/ package implements them
type UserService struct { db *DB }
func (s *UserService) FindUserByID(ctx context.Context, id uuid.UUID) (*aletheia.User, error) {
    // Implementation using sqlc queries
}
```

### Error Handling

Domain errors use error codes that are mapped to HTTP status at the transport layer:

```go
// Domain layer returns errors with codes
return aletheia.NotFound("User not found")
return aletheia.Invalid("Email is required")
return aletheia.Unauthorized("Invalid credentials")

// HTTP layer maps codes to status
func errorStatusCode(code string) int {
    switch code {
    case aletheia.ENOTFOUND:
        return http.StatusNotFound      // 404
    case aletheia.EINVALID:
        return http.StatusBadRequest    // 400
    case aletheia.EUNAUTHORIZED:
        return http.StatusUnauthorized  // 401
    case aletheia.EFORBIDDEN:
        return http.StatusForbidden     // 403
    case aletheia.ECONFLICT:
        return http.StatusConflict      // 409
    default:
        return http.StatusInternalServerError // 500
    }
}
```

### Entry Point Pattern

The application uses a testable `run()` pattern:

```go
func main() {
    ctx := context.Background()
    if err := run(ctx, os.Stdout, os.Stderr, os.Args, os.Getenv); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}

func run(
    ctx context.Context,
    stdout, stderr io.Writer,
    args []string,
    getenv func(string) string,
) error {
    // All external dependencies passed as parameters
    cfg, err := LoadConfig(getenv)
    // ...
}
```

### HTTP Server Pattern

The Server struct holds all dependencies:

```go
type Server struct {
    // Domain services
    userService         aletheia.UserService
    organizationService aletheia.OrganizationService
    // ...

    // External services
    fileStorage  aletheia.FileStorage
    emailService aletheia.EmailService
    aiService    aletheia.AIService
    queue        aletheia.Queue
}

// Handlers are methods on Server
func (s *Server) handleCreateOrganization(c echo.Context) error {
    // Has access to all services via s.*
}
```

### Testing with Mocks

Mock implementations use function fields for flexible test setup:

```go
type UserService struct {
    FindUserByIDFn func(ctx context.Context, id uuid.UUID) (*aletheia.User, error)
    // ...
}

func (s *UserService) FindUserByID(ctx context.Context, id uuid.UUID) (*aletheia.User, error) {
    if s.FindUserByIDFn != nil {
        return s.FindUserByIDFn(ctx, id)
    }
    return nil, aletheia.NotFound("User not found")
}
```

## Key Implementation Patterns

### Type Conversion (sqlc <-> domain)

The `postgres/convert.go` file handles conversions between sqlc-generated types and domain types:

```go
func toDomainUser(u database.User) *aletheia.User {
    return &aletheia.User{
        ID:        fromPgUUID(u.ID),
        Email:     u.Email,
        // ...
    }
}
```

### Context Helpers

User/organization context is managed via the root package:

```go
// Set in middleware
ctx = aletheia.NewContextWithUser(ctx, user)

// Retrieve in handlers
user := aletheia.UserFromContext(ctx)
```

### Routes Organization

All routes are defined in a single `http/routes.go` file for easy API overview.

## Important Notes

- Server runs on port 8080 by default (configurable via SERVER_PORT)
- Graceful shutdown implemented with 10-second timeout
- Static files served at `/static/` and `/uploads/`
- Upload endpoint: `POST /api/upload`
- Accepted image types: JPEG, PNG, WebP (5MB max)
- Job handlers should be idempotent (safe to retry)
