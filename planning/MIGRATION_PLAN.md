# Aletheia Migration Plan: CODE_STYLE_GUIDE.md Alignment

**Created:** December 10, 2024
**Status:** Planning
**Estimated Effort:** 6-8 weeks (incremental)

---

## Executive Summary

This document provides a comprehensive migration plan to align Aletheia with the patterns described in `CODE_STYLE_GUIDE.md`. The migration transforms the current Echo-based, `internal/`-structured codebase to a domain-first architecture with types and interfaces in the root package.

### Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Web Framework** | Keep Echo (Hybrid Approach) | Minimize risk for solo developer; adopt all other patterns |
| **sqlc Location** | Keep in `internal/database/` | No churn, works as implementation detail for `postgres/` |
| **Template Location** | Keep in `web/templates/` | Clean separation of Go code from web assets |
| **Validation** | Hybrid approach | Keep `go-playground/validator` + add `Valid()` for complex rules |
| **Module Path** | Keep full GitHub path | Standard Go convention |
| **Authorization** | Keep in HTTP layer | Sufficient for HTTP-only application |

---

## Current State vs Target State

### Architecture Overview

| Aspect | Current State | Target State |
|--------|--------------|--------------|
| **Domain Types** | `internal/database/models.go` (sqlc-generated) | Root package (`/user.go`, `/organization.go`, etc.) |
| **Service Interfaces** | Per-package (`storage.FileStorage`, `queue.Queue`) | Root package interfaces + implementation packages |
| **Handlers** | Struct methods in `internal/handlers/` | Methods on Server struct in `http/` |
| **Web Framework** | Echo v4 | Echo v4 (kept) |
| **Error Handling** | `internal/errors/AppError` with HTTP coupling | Domain error codes mapped at transport layer |
| **Routes** | Scattered across `cmd/main.go` | Single `http/routes.go` file |
| **Entry Point** | Monolithic `main()` | `run()` pattern for testability |

### Directory Structure Comparison

**Current:**
```
aletheia/
├── cmd/main.go
├── internal/
│   ├── ai/
│   ├── audit/
│   ├── auth/
│   ├── config/
│   ├── database/        # sqlc-generated
│   ├── email/
│   ├── errors/
│   ├── handlers/        # 18 handler files
│   ├── middleware/
│   ├── migrations/
│   ├── queue/
│   ├── session/
│   ├── storage/
│   ├── templates/
│   └── validation/
├── web/
│   ├── templates/
│   └── static/
└── uploads/
```

**Target:**
```
aletheia/                    # Root: domain types & service interfaces
├── user.go
├── organization.go
├── project.go
├── inspection.go
├── photo.go
├── violation.go
├── safety_code.go
├── session.go
├── error.go
├── context.go
├── storage.go              # Interface only
├── email.go                # Interface only
├── ai.go                   # Interface only
├── queue.go                # Interface only
│
├── cmd/aletheiad/
│   ├── main.go             # run() pattern
│   └── config.go
│
├── postgres/               # Database service implementations
│   ├── postgres.go
│   ├── user.go
│   ├── organization.go
│   ├── project.go
│   ├── inspection.go
│   ├── photo.go
│   ├── violation.go
│   ├── safety_code.go
│   ├── session.go
│   └── migrations/         # Embedded migrations
│
├── http/                   # HTTP transport layer
│   ├── server.go
│   ├── routes.go           # ALL routes in one file
│   ├── handlers.go         # Common helpers
│   ├── middleware.go
│   ├── auth.go
│   ├── organization.go
│   ├── project.go
│   ├── inspection.go
│   ├── photo.go
│   ├── violation.go
│   └── health.go
│
├── mock/                   # Test mocks
│   ├── user.go
│   ├── organization.go
│   └── ...
│
├── internal/
│   ├── database/           # sqlc-generated (kept here)
│   ├── auth/               # Password hashing utilities
│   ├── ai/                 # AI service implementations
│   ├── email/              # Email service implementations
│   ├── storage/            # Storage implementations
│   └── queue/              # Queue implementations
│
├── web/
│   ├── templates/          # HTML templates (kept here)
│   └── static/
│
└── uploads/
```

---

## Phase 1: Domain Layer

**Estimated Complexity:** Medium
**Estimated Duration:** 2-3 weeks

### Goal

Extract domain types and service interfaces to the root package, establishing clear contracts between layers.

### Files to Create

#### 1.1 Core Domain Types

**`/user.go`**
```go
package aletheia

import (
    "context"
    "time"

    "github.com/google/uuid"
)

// User represents a user in the system.
type User struct {
    ID           uuid.UUID  `json:"id"`
    Email        string     `json:"email"`
    Username     string     `json:"username"`
    FirstName    string     `json:"firstName,omitempty"`
    LastName     string     `json:"lastName,omitempty"`
    Status       UserStatus `json:"status"`
    StatusReason string     `json:"statusReason,omitempty"`
    CreatedAt    time.Time  `json:"createdAt"`
    UpdatedAt    time.Time  `json:"updatedAt"`
    LastLoginAt  *time.Time `json:"lastLoginAt,omitempty"`
    VerifiedAt   *time.Time `json:"verifiedAt,omitempty"`
}

type UserStatus string

const (
    UserStatusActive    UserStatus = "active"
    UserStatusSuspended UserStatus = "suspended"
    UserStatusDeleted   UserStatus = "deleted"
)

// UserService defines operations for managing users.
type UserService interface {
    FindUserByID(ctx context.Context, id uuid.UUID) (*User, error)
    FindUserByEmail(ctx context.Context, email string) (*User, error)
    FindUsers(ctx context.Context, filter UserFilter) ([]*User, int, error)
    CreateUser(ctx context.Context, user *User, password string) error
    UpdateUser(ctx context.Context, id uuid.UUID, upd UserUpdate) (*User, error)
    DeleteUser(ctx context.Context, id uuid.UUID) error

    // Authentication
    VerifyPassword(ctx context.Context, email, password string) (*User, error)
    SetVerificationToken(ctx context.Context, id uuid.UUID, token string) error
    VerifyEmail(ctx context.Context, token string) (*User, error)
    RequestPasswordReset(ctx context.Context, email string) (token string, err error)
    ResetPassword(ctx context.Context, token, newPassword string) error
}

// UserFilter for querying users.
type UserFilter struct {
    ID     *uuid.UUID
    Email  *string
    Status *UserStatus
    Offset int
    Limit  int
}

// UserUpdate for partial updates.
type UserUpdate struct {
    FirstName *string
    LastName  *string
    Status    *UserStatus
}
```

**`/organization.go`**
```go
package aletheia

import (
    "context"
    "time"

    "github.com/google/uuid"
)

type Organization struct {
    ID        uuid.UUID `json:"id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}

type OrganizationRole string

const (
    RoleOwner  OrganizationRole = "owner"
    RoleAdmin  OrganizationRole = "admin"
    RoleMember OrganizationRole = "member"
)

type OrganizationMember struct {
    ID             uuid.UUID        `json:"id"`
    OrganizationID uuid.UUID        `json:"organizationId"`
    UserID         uuid.UUID        `json:"userId"`
    Role           OrganizationRole `json:"role"`
    CreatedAt      time.Time        `json:"createdAt"`
}

type OrganizationService interface {
    FindOrganizationByID(ctx context.Context, id uuid.UUID) (*Organization, error)
    FindUserOrganizations(ctx context.Context, userID uuid.UUID) ([]*OrganizationWithRole, error)
    CreateOrganization(ctx context.Context, org *Organization, ownerID uuid.UUID) error
    UpdateOrganization(ctx context.Context, id uuid.UUID, upd OrganizationUpdate) (*Organization, error)
    DeleteOrganization(ctx context.Context, id uuid.UUID) error

    // Membership
    GetMembership(ctx context.Context, orgID, userID uuid.UUID) (*OrganizationMember, error)
    AddMember(ctx context.Context, orgID, userID uuid.UUID, role OrganizationRole) (*OrganizationMember, error)
    UpdateMemberRole(ctx context.Context, memberID uuid.UUID, role OrganizationRole) (*OrganizationMember, error)
    RemoveMember(ctx context.Context, memberID uuid.UUID) error
    ListMembers(ctx context.Context, orgID uuid.UUID) ([]*OrganizationMember, error)
}

type OrganizationWithRole struct {
    Organization
    Role OrganizationRole `json:"role"`
}

type OrganizationUpdate struct {
    Name *string
}
```

**Additional domain files to create:**
- `/project.go` - Project, ProjectService, ProjectFilter, ProjectUpdate
- `/inspection.go` - Inspection, InspectionService, InspectionStatus
- `/photo.go` - Photo, PhotoService
- `/violation.go` - Violation, ViolationService, ViolationStatus, Severity
- `/safety_code.go` - SafetyCode, SafetyCodeService
- `/session.go` - Session, SessionService

#### 1.2 Error Handling

**`/error.go`**
```go
package aletheia

import (
    "errors"
    "fmt"
)

// Domain error codes - transport layer maps these to HTTP status codes.
const (
    ECONFLICT     = "conflict"      // 409
    EINTERNAL     = "internal"      // 500
    EINVALID      = "invalid"       // 400
    ENOTFOUND     = "not_found"     // 404
    EUNAUTHORIZED = "unauthorized"  // 401
    EFORBIDDEN    = "forbidden"     // 403
    ERATELIMIT    = "rate_limit"    // 429
)

// Error represents an application-specific error.
type Error struct {
    Code    string            `json:"code"`
    Message string            `json:"message"`
    Fields  map[string]string `json:"fields,omitempty"`
    Err     error             `json:"-"`
}

func (e *Error) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
    }
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
    return e.Err
}

// Errorf creates a new application error.
func Errorf(code string, format string, args ...any) *Error {
    return &Error{
        Code:    code,
        Message: fmt.Sprintf(format, args...),
    }
}

// WrapError wraps an underlying error with application context.
func WrapError(code string, message string, err error) *Error {
    return &Error{
        Code:    code,
        Message: message,
        Err:     err,
    }
}

// ErrorWithFields creates a validation error with field-specific messages.
func ErrorWithFields(fields map[string]string) *Error {
    return &Error{
        Code:    EINVALID,
        Message: "Validation failed",
        Fields:  fields,
    }
}

// ErrorCode extracts the code from an error.
func ErrorCode(err error) string {
    if err == nil {
        return ""
    }
    var e *Error
    if errors.As(err, &e) {
        return e.Code
    }
    return EINTERNAL
}

// ErrorMessage extracts the user-safe message from an error.
func ErrorMessage(err error) string {
    if err == nil {
        return ""
    }
    var e *Error
    if errors.As(err, &e) {
        return e.Message
    }
    return "An internal error occurred."
}

// ErrorFields extracts field errors from a validation error.
func ErrorFields(err error) map[string]string {
    var e *Error
    if errors.As(err, &e) {
        return e.Fields
    }
    return nil
}
```

#### 1.3 Context Helpers

**`/context.go`**
```go
package aletheia

import (
    "context"

    "github.com/google/uuid"
)

type contextKey int

const (
    userContextKey contextKey = iota + 1
    sessionContextKey
    organizationContextKey
)

// NewContextWithUser attaches user to context.
func NewContextWithUser(ctx context.Context, user *User) context.Context {
    return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext returns the authenticated user, or nil.
func UserFromContext(ctx context.Context) *User {
    user, _ := ctx.Value(userContextKey).(*User)
    return user
}

// UserIDFromContext returns the authenticated user's ID, or zero UUID.
func UserIDFromContext(ctx context.Context) uuid.UUID {
    if user := UserFromContext(ctx); user != nil {
        return user.ID
    }
    return uuid.UUID{}
}

// NewContextWithSession attaches session to context.
func NewContextWithSession(ctx context.Context, session *Session) context.Context {
    return context.WithValue(ctx, sessionContextKey, session)
}

// SessionFromContext returns the current session, or nil.
func SessionFromContext(ctx context.Context) *Session {
    session, _ := ctx.Value(sessionContextKey).(*Session)
    return session
}

// NewContextWithOrganization attaches organization to context.
func NewContextWithOrganization(ctx context.Context, org *Organization) context.Context {
    return context.WithValue(ctx, organizationContextKey, org)
}

// OrganizationFromContext returns the current organization, or nil.
func OrganizationFromContext(ctx context.Context) *Organization {
    org, _ := ctx.Value(organizationContextKey).(*Organization)
    return org
}
```

#### 1.4 External Service Interfaces

**`/storage.go`**
```go
package aletheia

import (
    "context"
    "io"
)

// FileStorage defines file storage operations.
type FileStorage interface {
    Upload(ctx context.Context, key string, reader io.Reader, contentType string) (url string, err error)
    Delete(ctx context.Context, key string) error
    GetURL(key string) string
}
```

**`/email.go`**
```go
package aletheia

import "context"

// EmailService defines email sending operations.
type EmailService interface {
    SendVerificationEmail(ctx context.Context, to, token string) error
    SendPasswordResetEmail(ctx context.Context, to, token string) error
    SendWelcomeEmail(ctx context.Context, to, name string) error
}
```

**`/ai.go`**
```go
package aletheia

import "context"

// AIService defines AI analysis operations.
type AIService interface {
    AnalyzePhoto(ctx context.Context, photoURL string, safetyCodes []SafetyCode) (*AnalysisResult, error)
}

type AnalysisResult struct {
    Violations []DetectedViolation `json:"violations"`
    Summary    string              `json:"summary"`
}

type DetectedViolation struct {
    SafetyCodeID uuid.UUID `json:"safetyCodeId"`
    Description  string    `json:"description"`
    Severity     Severity  `json:"severity"`
    Confidence   float64   `json:"confidence"`
    BoundingBox  *BoundingBox `json:"boundingBox,omitempty"`
}

type BoundingBox struct {
    X      float64 `json:"x"`
    Y      float64 `json:"y"`
    Width  float64 `json:"width"`
    Height float64 `json:"height"`
}
```

**`/queue.go`**
```go
package aletheia

import (
    "context"
    "time"

    "github.com/google/uuid"
)

// Queue defines job queue operations.
type Queue interface {
    Enqueue(ctx context.Context, job *Job) error
    Dequeue(ctx context.Context, queueName string) (*Job, error)
    Complete(ctx context.Context, jobID uuid.UUID, result []byte) error
    Fail(ctx context.Context, jobID uuid.UUID, errMsg string) error
    GetJob(ctx context.Context, jobID uuid.UUID) (*Job, error)
}

type Job struct {
    ID             uuid.UUID  `json:"id"`
    QueueName      string     `json:"queueName"`
    JobType        string     `json:"jobType"`
    OrganizationID uuid.UUID  `json:"organizationId"`
    Payload        []byte     `json:"payload"`
    Status         JobStatus  `json:"status"`
    Result         []byte     `json:"result,omitempty"`
    Error          string     `json:"error,omitempty"`
    Attempts       int        `json:"attempts"`
    MaxAttempts    int        `json:"maxAttempts"`
    ScheduledAt    time.Time  `json:"scheduledAt"`
    CreatedAt      time.Time  `json:"createdAt"`
    UpdatedAt      time.Time  `json:"updatedAt"`
}

type JobStatus string

const (
    JobStatusPending   JobStatus = "pending"
    JobStatusRunning   JobStatus = "running"
    JobStatusCompleted JobStatus = "completed"
    JobStatusFailed    JobStatus = "failed"
)
```

### Phase 1 Checklist

- [ ] Create `/user.go`
- [ ] Create `/organization.go`
- [ ] Create `/project.go`
- [ ] Create `/inspection.go`
- [ ] Create `/photo.go`
- [ ] Create `/violation.go`
- [ ] Create `/safety_code.go`
- [ ] Create `/session.go`
- [ ] Create `/error.go`
- [ ] Create `/context.go`
- [ ] Create `/storage.go`
- [ ] Create `/email.go`
- [ ] Create `/ai.go`
- [ ] Create `/queue.go`
- [ ] Verify `go build` succeeds
- [ ] Update imports in existing code to use new types (gradual)

---

## Phase 2: Error Handling Migration

**Estimated Complexity:** Low-Medium
**Estimated Duration:** 1 week

### Goal

Replace `internal/errors/AppError` with domain error codes, mapping to HTTP status at the transport layer.

### Changes

#### 2.1 Create HTTP Error Mapper

**`http/errors.go`** (new file)
```go
package http

import (
    "net/http"

    "github.com/dukerupert/aletheia"
    "github.com/labstack/echo/v4"
)

// errorStatusCode maps domain error codes to HTTP status codes.
func errorStatusCode(code string) int {
    switch code {
    case aletheia.ENOTFOUND:
        return http.StatusNotFound
    case aletheia.EINVALID:
        return http.StatusBadRequest
    case aletheia.EUNAUTHORIZED:
        return http.StatusUnauthorized
    case aletheia.EFORBIDDEN:
        return http.StatusForbidden
    case aletheia.ECONFLICT:
        return http.StatusConflict
    case aletheia.ERATELIMIT:
        return http.StatusTooManyRequests
    default:
        return http.StatusInternalServerError
    }
}

// handleError converts domain errors to HTTP responses.
func (s *Server) handleError(c echo.Context, err error) error {
    code := aletheia.ErrorCode(err)
    message := aletheia.ErrorMessage(err)
    fields := aletheia.ErrorFields(err)
    status := errorStatusCode(code)

    // Log internal errors
    if code == aletheia.EINTERNAL {
        s.logger.Error("internal error", "error", err)
        message = "An internal error occurred."
    }

    // Return appropriate response format
    if isHTMX(c) {
        return s.renderError(c, status, message, fields)
    }

    return c.JSON(status, map[string]any{
        "error":   code,
        "message": message,
        "fields":  fields,
    })
}

func isHTMX(c echo.Context) bool {
    return c.Request().Header.Get("HX-Request") == "true"
}
```

#### 2.2 Update Handlers

**Before:**
```go
func (h *OrganizationHandler) GetOrganization(c echo.Context) error {
    // ...
    if err != nil {
        return errors.NewNotFoundError("ORG_NOT_FOUND", "Organization not found", err).ToEchoError()
    }
}
```

**After:**
```go
func (s *Server) handleGetOrganization(c echo.Context) error {
    // ...
    if err != nil {
        return s.handleError(c, aletheia.Errorf(aletheia.ENOTFOUND, "Organization not found"))
    }
}
```

### Phase 2 Checklist

- [ ] Create `http/errors.go` with error mapper
- [ ] Update handlers one-by-one to return domain errors
- [ ] Update middleware to use new error pattern
- [ ] Remove `internal/errors/errors.go`
- [ ] Verify all error responses work correctly

---

## Phase 3: Storage Layer (postgres/)

**Estimated Complexity:** Medium-High
**Estimated Duration:** 2-3 weeks

### Goal

Create `postgres/` package that implements domain service interfaces, using sqlc-generated code internally.

### Structure

```
postgres/
├── postgres.go         # DB struct, connection, service initialization
├── user.go             # UserService implementation
├── organization.go     # OrganizationService implementation
├── project.go          # ProjectService implementation
├── inspection.go       # InspectionService implementation
├── photo.go            # PhotoService implementation
├── violation.go        # ViolationService implementation
├── safety_code.go      # SafetyCodeService implementation
├── session.go          # SessionService implementation
├── convert.go          # Type conversion helpers (sqlc <-> domain)
└── migrations/         # Embedded SQL migrations
```

### Key Implementation

**`postgres/postgres.go`**
```go
package postgres

import (
    "context"
    "embed"

    "github.com/dukerupert/aletheia"
    "github.com/dukerupert/aletheia/internal/database"
    "github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps the database connection and exposes domain services.
type DB struct {
    pool    *pgxpool.Pool
    queries *database.Queries

    // Services
    UserService         aletheia.UserService
    OrganizationService aletheia.OrganizationService
    ProjectService      aletheia.ProjectService
    InspectionService   aletheia.InspectionService
    PhotoService        aletheia.PhotoService
    ViolationService    aletheia.ViolationService
    SafetyCodeService   aletheia.SafetyCodeService
    SessionService      aletheia.SessionService
}

// NewDB creates a new database wrapper with all services initialized.
func NewDB(pool *pgxpool.Pool) *DB {
    queries := database.New(pool)
    db := &DB{
        pool:    pool,
        queries: queries,
    }

    // Initialize services
    db.UserService = &UserService{db: db}
    db.OrganizationService = &OrganizationService{db: db}
    db.ProjectService = &ProjectService{db: db}
    db.InspectionService = &InspectionService{db: db}
    db.PhotoService = &PhotoService{db: db}
    db.ViolationService = &ViolationService{db: db}
    db.SafetyCodeService = &SafetyCodeService{db: db}
    db.SessionService = &SessionService{db: db}

    return db
}

// Close closes the database connection pool.
func (db *DB) Close() {
    db.pool.Close()
}

// MigrationsFS returns the embedded migrations filesystem.
func MigrationsFS() embed.FS {
    return migrationsFS
}
```

**`postgres/user.go`** (example implementation)
```go
package postgres

import (
    "context"

    "github.com/dukerupert/aletheia"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
)

// Compile-time interface check
var _ aletheia.UserService = (*UserService)(nil)

type UserService struct {
    db *DB
}

func (s *UserService) FindUserByID(ctx context.Context, id uuid.UUID) (*aletheia.User, error) {
    user, err := s.db.queries.GetUser(ctx, toPgUUID(id))
    if err != nil {
        if err == pgx.ErrNoRows {
            return nil, aletheia.Errorf(aletheia.ENOTFOUND, "User not found")
        }
        return nil, aletheia.WrapError(aletheia.EINTERNAL, "Failed to fetch user", err)
    }
    return toDomainUser(user), nil
}

// ... implement remaining methods
```

**`postgres/convert.go`** (type conversions)
```go
package postgres

import (
    "github.com/dukerupert/aletheia"
    "github.com/dukerupert/aletheia/internal/database"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5/pgtype"
)

func toPgUUID(id uuid.UUID) pgtype.UUID {
    return pgtype.UUID{Bytes: id, Valid: true}
}

func fromPgUUID(id pgtype.UUID) uuid.UUID {
    if !id.Valid {
        return uuid.UUID{}
    }
    return uuid.UUID(id.Bytes)
}

func toDomainUser(u database.User) *aletheia.User {
    user := &aletheia.User{
        ID:        fromPgUUID(u.ID),
        Email:     u.Email,
        Username:  u.Username,
        Status:    aletheia.UserStatus(u.Status),
        CreatedAt: u.CreatedAt.Time,
        UpdatedAt: u.UpdatedAt.Time,
    }
    if u.FirstName.Valid {
        user.FirstName = u.FirstName.String
    }
    if u.LastName.Valid {
        user.LastName = u.LastName.String
    }
    if u.LastLoginAt.Valid {
        t := u.LastLoginAt.Time
        user.LastLoginAt = &t
    }
    if u.VerifiedAt.Valid {
        t := u.VerifiedAt.Time
        user.VerifiedAt = &t
    }
    return user
}

// ... additional conversion functions
```

### Phase 3 Checklist

- [ ] Create `postgres/postgres.go`
- [ ] Create `postgres/convert.go`
- [ ] Create `postgres/user.go`
- [ ] Create `postgres/organization.go`
- [ ] Create `postgres/project.go`
- [ ] Create `postgres/inspection.go`
- [ ] Create `postgres/photo.go`
- [ ] Create `postgres/violation.go`
- [ ] Create `postgres/safety_code.go`
- [ ] Create `postgres/session.go`
- [ ] Move migrations to `postgres/migrations/`
- [ ] Add compile-time interface checks to all services
- [ ] Verify all database operations work correctly

---

## Phase 4: HTTP Layer

**Estimated Complexity:** High
**Estimated Duration:** 2-3 weeks

### Goal

Create `http/` package with Server struct, centralized routes, and handler methods.

### Structure

```
http/
├── server.go           # Server struct, lifecycle
├── routes.go           # ALL routes in one file
├── handlers.go         # Common helpers (decode, render, respond)
├── errors.go           # Error handling (from Phase 2)
├── middleware.go       # Middleware functions
├── auth.go             # Auth handlers
├── organization.go     # Organization handlers
├── project.go          # Project handlers
├── inspection.go       # Inspection handlers
├── photo.go            # Photo handlers
├── violation.go        # Violation handlers
├── safety_code.go      # Safety code handlers
└── health.go           # Health check handlers
```

### Key Implementation

**`http/server.go`**
```go
package http

import (
    "context"
    "log/slog"
    "net"

    "github.com/dukerupert/aletheia"
    "github.com/labstack/echo/v4"
)

type Server struct {
    echo   *echo.Echo
    ln     net.Listener
    logger *slog.Logger

    // Configuration
    Addr   string
    Domain string

    // Domain services
    userService         aletheia.UserService
    organizationService aletheia.OrganizationService
    projectService      aletheia.ProjectService
    inspectionService   aletheia.InspectionService
    photoService        aletheia.PhotoService
    violationService    aletheia.ViolationService
    safetyCodeService   aletheia.SafetyCodeService
    sessionService      aletheia.SessionService

    // External services
    fileStorage  aletheia.FileStorage
    emailService aletheia.EmailService
    aiService    aletheia.AIService
    queue        aletheia.Queue
}

type Config struct {
    Addr   string
    Domain string
    Logger *slog.Logger

    // Domain services
    UserService         aletheia.UserService
    OrganizationService aletheia.OrganizationService
    ProjectService      aletheia.ProjectService
    InspectionService   aletheia.InspectionService
    PhotoService        aletheia.PhotoService
    ViolationService    aletheia.ViolationService
    SafetyCodeService   aletheia.SafetyCodeService
    SessionService      aletheia.SessionService

    // External services
    FileStorage  aletheia.FileStorage
    EmailService aletheia.EmailService
    AIService    aletheia.AIService
    Queue        aletheia.Queue
}

func NewServer(cfg Config) *Server {
    s := &Server{
        Addr:                cfg.Addr,
        Domain:              cfg.Domain,
        logger:              cfg.Logger,
        userService:         cfg.UserService,
        organizationService: cfg.OrganizationService,
        projectService:      cfg.ProjectService,
        inspectionService:   cfg.InspectionService,
        photoService:        cfg.PhotoService,
        violationService:    cfg.ViolationService,
        safetyCodeService:   cfg.SafetyCodeService,
        sessionService:      cfg.SessionService,
        fileStorage:         cfg.FileStorage,
        emailService:        cfg.EmailService,
        aiService:           cfg.AIService,
        queue:               cfg.Queue,
    }

    s.echo = echo.New()
    s.echo.HideBanner = true
    s.echo.Renderer = newTemplateRenderer()

    s.registerMiddleware()
    s.registerRoutes()

    return s
}

func (s *Server) Open() error {
    var err error
    s.ln, err = net.Listen("tcp", s.Addr)
    if err != nil {
        return err
    }
    go s.echo.Server.Serve(s.ln)
    return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
    return s.echo.Shutdown(ctx)
}

func (s *Server) URL() string {
    if s.ln == nil {
        return ""
    }
    return "http://" + s.ln.Addr().String()
}
```

**`http/routes.go`**
```go
package http

func (s *Server) registerRoutes() {
    e := s.echo

    // Static files
    e.Static("/static", "web/static")
    e.Static("/uploads", "./uploads")

    // Health checks
    e.GET("/health", s.handleHealthCheck)
    e.GET("/health/live", s.handleLivenessCheck)
    e.GET("/health/ready", s.handleReadinessCheck)

    // Public pages
    e.GET("/", s.handleHome)
    e.GET("/login", s.handleLoginPage)
    e.GET("/register", s.handleRegisterPage)
    e.GET("/verify", s.handleVerifyEmailPage)
    e.GET("/forgot-password", s.handleForgotPasswordPage)
    e.GET("/reset-password", s.handleResetPasswordPage)

    // Auth API (public, rate-limited)
    auth := e.Group("/api/auth")
    auth.Use(s.strictRateLimiter())
    auth.POST("/register", s.handleRegister)
    auth.POST("/login", s.handleLogin)
    auth.POST("/verify-email", s.handleVerifyEmail)
    auth.POST("/resend-verification", s.handleResendVerification)
    auth.POST("/request-password-reset", s.handleRequestPasswordReset)
    auth.POST("/reset-password", s.handleResetPassword)

    // Protected pages
    pages := e.Group("")
    pages.Use(s.requireAuth())
    pages.GET("/dashboard", s.handleDashboard)
    pages.GET("/profile", s.handleProfilePage)
    pages.GET("/organizations", s.handleOrganizationsPage)
    pages.GET("/organizations/new", s.handleNewOrganizationPage)
    pages.GET("/projects", s.handleProjectsPage)
    pages.GET("/projects/new", s.handleNewProjectPage)
    pages.GET("/projects/:id", s.handleProjectDetailPage)
    pages.GET("/inspections", s.handleInspectionsPage)
    pages.GET("/inspections/:id", s.handleInspectionDetailPage)
    pages.GET("/photos/:id", s.handlePhotoDetailPage)

    // Protected API
    api := e.Group("/api")
    api.Use(s.requireAuth())
    api.Use(s.userRateLimiter())

    // Auth actions
    api.POST("/auth/logout", s.handleLogout)
    api.GET("/auth/me", s.handleMe)
    api.PUT("/auth/profile", s.handleUpdateProfile)

    // Organizations
    api.POST("/organizations", s.handleCreateOrganization)
    api.GET("/organizations", s.handleListOrganizations)
    api.GET("/organizations/:id", s.handleGetOrganization)
    api.PUT("/organizations/:id", s.handleUpdateOrganization)
    api.DELETE("/organizations/:id", s.handleDeleteOrganization)
    api.GET("/organizations/:id/members", s.handleListMembers)
    api.POST("/organizations/:id/members", s.handleAddMember)
    api.PUT("/organizations/:id/members/:memberId", s.handleUpdateMemberRole)
    api.DELETE("/organizations/:id/members/:memberId", s.handleRemoveMember)

    // Projects
    api.POST("/projects", s.handleCreateProject)
    api.GET("/projects/:id", s.handleGetProject)
    api.GET("/organizations/:orgId/projects", s.handleListProjects)
    api.PUT("/projects/:id", s.handleUpdateProject)
    api.DELETE("/projects/:id", s.handleDeleteProject)

    // Inspections
    api.POST("/inspections", s.handleCreateInspection)
    api.GET("/inspections/:id", s.handleGetInspection)
    api.GET("/projects/:projectId/inspections", s.handleListInspections)
    api.PUT("/inspections/:id/status", s.handleUpdateInspectionStatus)

    // Photos
    api.POST("/upload", s.handleUploadPhoto)
    api.GET("/inspections/:inspectionId/photos", s.handleListPhotos)
    api.GET("/photos/:id", s.handleGetPhoto)
    api.DELETE("/photos/:id", s.handleDeletePhoto)
    api.POST("/photos/analyze", s.handleAnalyzePhoto)

    // Violations
    api.GET("/inspections/:inspectionId/violations", s.handleListViolations)
    api.POST("/violations/:id/confirm", s.handleConfirmViolation)
    api.POST("/violations/:id/dismiss", s.handleDismissViolation)
    api.POST("/violations/:id/pending", s.handleSetViolationPending)
    api.PATCH("/violations/:id", s.handleUpdateViolation)
    api.POST("/violations/manual", s.handleCreateManualViolation)

    // Safety codes
    api.POST("/safety-codes", s.handleCreateSafetyCode)
    api.GET("/safety-codes", s.handleListSafetyCodes)
    api.GET("/safety-codes/:id", s.handleGetSafetyCode)
    api.PUT("/safety-codes/:id", s.handleUpdateSafetyCode)
    api.DELETE("/safety-codes/:id", s.handleDeleteSafetyCode)

    // Jobs
    api.GET("/jobs/status", s.handleGetJobStatus)
    api.POST("/jobs/:id/cancel", s.handleCancelJob)
}
```

**`http/handlers.go`** (common helpers)
```go
package http

import (
    "github.com/dukerupert/aletheia"
    "github.com/labstack/echo/v4"
)

// decode reads the request body into v (JSON or form).
func (s *Server) decode(c echo.Context, v any) error {
    if err := c.Bind(v); err != nil {
        return aletheia.Errorf(aletheia.EINVALID, "Invalid request body")
    }
    return nil
}

// respond sends a JSON response.
func (s *Server) respond(c echo.Context, status int, data any) error {
    return c.JSON(status, data)
}

// render renders an HTML template.
func (s *Server) render(c echo.Context, status int, template string, data any) error {
    return c.Render(status, template, data)
}

// userFromContext extracts the authenticated user from Echo context.
func (s *Server) userFromContext(c echo.Context) *aletheia.User {
    return aletheia.UserFromContext(c.Request().Context())
}
```

### Phase 4 Checklist

- [ ] Create `http/server.go`
- [ ] Create `http/routes.go`
- [ ] Create `http/handlers.go`
- [ ] Create `http/errors.go`
- [ ] Create `http/middleware.go`
- [ ] Migrate auth handlers to `http/auth.go`
- [ ] Migrate organization handlers to `http/organization.go`
- [ ] Migrate project handlers to `http/project.go`
- [ ] Migrate inspection handlers to `http/inspection.go`
- [ ] Migrate photo handlers to `http/photo.go`
- [ ] Migrate violation handlers to `http/violation.go`
- [ ] Migrate safety code handlers to `http/safety_code.go`
- [ ] Create `http/health.go`
- [ ] Remove `internal/handlers/`
- [ ] Verify all routes work correctly

---

## Phase 5: Testing (mock/)

**Estimated Complexity:** Low-Medium
**Estimated Duration:** 1 week

### Goal

Create mock implementations for isolated handler testing.

### Structure

```
mock/
├── user.go
├── organization.go
├── project.go
├── inspection.go
├── photo.go
├── violation.go
├── safety_code.go
├── session.go
├── storage.go
├── email.go
├── ai.go
└── queue.go
```

### Implementation Pattern

**`mock/user.go`**
```go
package mock

import (
    "context"

    "github.com/dukerupert/aletheia"
    "github.com/google/uuid"
)

var _ aletheia.UserService = (*UserService)(nil)

type UserService struct {
    FindUserByIDFn    func(ctx context.Context, id uuid.UUID) (*aletheia.User, error)
    FindUserByEmailFn func(ctx context.Context, email string) (*aletheia.User, error)
    FindUsersFn       func(ctx context.Context, filter aletheia.UserFilter) ([]*aletheia.User, int, error)
    CreateUserFn      func(ctx context.Context, user *aletheia.User, password string) error
    UpdateUserFn      func(ctx context.Context, id uuid.UUID, upd aletheia.UserUpdate) (*aletheia.User, error)
    DeleteUserFn      func(ctx context.Context, id uuid.UUID) error
    VerifyPasswordFn  func(ctx context.Context, email, password string) (*aletheia.User, error)
    // ... other function fields
}

func (s *UserService) FindUserByID(ctx context.Context, id uuid.UUID) (*aletheia.User, error) {
    if s.FindUserByIDFn != nil {
        return s.FindUserByIDFn(ctx, id)
    }
    return nil, aletheia.Errorf(aletheia.ENOTFOUND, "User not found")
}

func (s *UserService) FindUserByEmail(ctx context.Context, email string) (*aletheia.User, error) {
    if s.FindUserByEmailFn != nil {
        return s.FindUserByEmailFn(ctx, email)
    }
    return nil, aletheia.Errorf(aletheia.ENOTFOUND, "User not found")
}

// ... implement remaining methods
```

### Test Pattern

**`http/organization_test.go`**
```go
package http_test

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/dukerupert/aletheia"
    apphttp "github.com/dukerupert/aletheia/http"
    "github.com/dukerupert/aletheia/mock"
    "github.com/google/uuid"
)

func TestHandleCreateOrganization(t *testing.T) {
    t.Parallel()

    orgService := &mock.OrganizationService{
        CreateOrganizationFn: func(ctx context.Context, org *aletheia.Organization, ownerID uuid.UUID) error {
            org.ID = uuid.New()
            return nil
        },
    }

    srv := apphttp.NewServer(apphttp.Config{
        OrganizationService: orgService,
        // ... minimal config
    })

    t.Run("creates organization successfully", func(t *testing.T) {
        req := httptest.NewRequest("POST", "/api/organizations",
            strings.NewReader(`{"name":"Test Org"}`))
        req.Header.Set("Content-Type", "application/json")
        // Add auth cookie/header

        rec := httptest.NewRecorder()
        srv.ServeHTTP(rec, req)

        if rec.Code != http.StatusCreated {
            t.Errorf("expected 201, got %d", rec.Code)
        }
    })
}
```

### Phase 5 Checklist

- [ ] Create `mock/user.go`
- [ ] Create `mock/organization.go`
- [ ] Create `mock/project.go`
- [ ] Create `mock/inspection.go`
- [ ] Create `mock/photo.go`
- [ ] Create `mock/violation.go`
- [ ] Create `mock/safety_code.go`
- [ ] Create `mock/session.go`
- [ ] Create `mock/storage.go`
- [ ] Create `mock/email.go`
- [ ] Create `mock/ai.go`
- [ ] Create `mock/queue.go`
- [ ] Write handler tests using mocks
- [ ] Verify test coverage is adequate

---

## Phase 6: Entry Point (run() Pattern)

**Estimated Complexity:** Low
**Estimated Duration:** 0.5 weeks

### Goal

Restructure entry point to use the testable `run()` pattern.

### Structure

```
cmd/aletheiad/
├── main.go         # main() calls run()
└── config.go       # Configuration types and loading
```

### Implementation

**`cmd/aletheiad/main.go`**
```go
package main

import (
    "context"
    "fmt"
    "io"
    "os"
    "os/signal"
    "sync"
    "time"

    "github.com/dukerupert/aletheia/http"
    "github.com/dukerupert/aletheia/postgres"
    // ... other imports
)

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
    // Handle shutdown signals
    ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
    defer cancel()

    // Load configuration
    cfg, err := loadConfig(getenv)
    if err != nil {
        return fmt.Errorf("config: %w", err)
    }

    // Configure logger
    logger := cfg.Logger()

    // Initialize database
    pool, err := connectDatabase(ctx, cfg.Database)
    if err != nil {
        return fmt.Errorf("database: %w", err)
    }
    defer pool.Close()

    // Run migrations
    if err := runMigrations(pool); err != nil {
        return fmt.Errorf("migrations: %w", err)
    }

    // Create database services
    db := postgres.NewDB(pool)

    // Initialize external services
    fileStorage, err := initStorage(ctx, logger, cfg.Storage)
    if err != nil {
        return fmt.Errorf("storage: %w", err)
    }

    emailService := initEmail(logger, cfg.Email)
    aiService := initAI(logger, cfg.AI)
    queue := initQueue(pool, logger, cfg.Queue)

    // Create HTTP server
    srv := http.NewServer(http.Config{
        Addr:                cfg.HTTP.Addr,
        Domain:              cfg.HTTP.Domain,
        Logger:              logger,
        UserService:         db.UserService,
        OrganizationService: db.OrganizationService,
        ProjectService:      db.ProjectService,
        InspectionService:   db.InspectionService,
        PhotoService:        db.PhotoService,
        ViolationService:    db.ViolationService,
        SafetyCodeService:   db.SafetyCodeService,
        SessionService:      db.SessionService,
        FileStorage:         fileStorage,
        EmailService:        emailService,
        AIService:           aiService,
        Queue:               queue,
    })

    if err := srv.Open(); err != nil {
        return fmt.Errorf("http server: %w", err)
    }

    fmt.Fprintf(stdout, "server listening on %s\n", cfg.HTTP.Addr)

    // Start background workers
    workerPool := startWorkerPool(ctx, queue, logger, db)

    // Wait for shutdown signal
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        <-ctx.Done()

        shutdownCtx, shutdownCancel := context.WithTimeout(
            context.Background(),
            10*time.Second,
        )
        defer shutdownCancel()

        if err := srv.Shutdown(shutdownCtx); err != nil {
            fmt.Fprintf(stderr, "shutdown error: %v\n", err)
        }
        workerPool.Stop()
    }()

    wg.Wait()
    fmt.Fprintf(stdout, "server stopped\n")
    return nil
}
```

**`cmd/aletheiad/config.go`**
```go
package main

import (
    "fmt"
    "log/slog"
    "os"
    "time"
)

type Config struct {
    App      AppConfig
    HTTP     HTTPConfig
    Database DatabaseConfig
    Storage  StorageConfig
    Email    EmailConfig
    AI       AIConfig
    Queue    QueueConfig
}

type AppConfig struct {
    Env      string
    LogLevel slog.Level
}

type HTTPConfig struct {
    Addr         string
    Domain       string
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
}

type DatabaseConfig struct {
    Host     string
    Port     string
    User     string
    Password string
    Name     string
}

// ... other config types

func loadConfig(getenv func(string) string) (*Config, error) {
    cfg := &Config{
        App: AppConfig{
            Env:      envOrDefault(getenv, "ENVIRONMENT", "dev"),
            LogLevel: parseLogLevel(getenv("LOG_LEVEL")),
        },
        HTTP: HTTPConfig{
            Addr:         fmt.Sprintf("%s:%s", envOrDefault(getenv, "SERVER_HOST", ""), envOrDefault(getenv, "SERVER_PORT", "1323")),
            Domain:       getenv("DOMAIN"),
            ReadTimeout:  parseDuration(getenv("HTTP_READ_TIMEOUT"), 15*time.Second),
            WriteTimeout: parseDuration(getenv("HTTP_WRITE_TIMEOUT"), 15*time.Second),
        },
        Database: DatabaseConfig{
            Host:     envOrDefault(getenv, "DB_HOSTNAME", "localhost"),
            Port:     envOrDefault(getenv, "DB_PORT", "5432"),
            User:     envOrDefault(getenv, "DB_USER", "postgres"),
            Password: getenv("DB_PASSWORD"),
            Name:     envOrDefault(getenv, "DB_NAME", "aletheia"),
        },
        // ... load other configs
    }

    // Validate production requirements
    if cfg.App.Env == "prod" {
        if getenv("JWT_SECRET") == "" {
            return nil, fmt.Errorf("JWT_SECRET is required in production")
        }
    }

    return cfg, nil
}

func (c *Config) Logger() *slog.Logger {
    var handler slog.Handler
    if c.App.Env == "prod" {
        handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: c.App.LogLevel})
    } else {
        handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: c.App.LogLevel})
    }
    return slog.New(handler)
}

func envOrDefault(getenv func(string) string, key, defaultVal string) string {
    if v := getenv(key); v != "" {
        return v
    }
    return defaultVal
}
```

### Phase 6 Checklist

- [ ] Create `cmd/aletheiad/main.go` with `run()` pattern
- [ ] Create `cmd/aletheiad/config.go`
- [ ] Move configuration logic from `internal/config/`
- [ ] Update Makefile for new binary location
- [ ] Update Docker/deployment configs
- [ ] Remove old `cmd/main.go`
- [ ] Verify application starts correctly
- [ ] Verify graceful shutdown works

---

## Post-Migration Cleanup

After all phases are complete:

- [ ] Remove `internal/errors/` (replaced by `/error.go`)
- [ ] Remove `internal/handlers/` (replaced by `http/`)
- [ ] Remove `internal/config/` (replaced by `cmd/aletheiad/config.go`)
- [ ] Remove `internal/templates/` (kept in `web/templates/`)
- [ ] Remove `internal/validation/` (hybrid approach, kept for field validation)
- [ ] Update `internal/` to only contain:
  - `database/` (sqlc-generated)
  - `auth/` (password hashing utilities)
  - `ai/` (AI service implementations)
  - `email/` (email service implementations)
  - `storage/` (storage implementations)
  - `queue/` (queue implementations)
- [ ] Update all import paths
- [ ] Run full test suite
- [ ] Update CI/CD pipeline
- [ ] Update README.md with new project structure
- [ ] Update CLAUDE.md with new patterns

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Breaking existing functionality | Migrate incrementally, run tests after each phase |
| sqlc types incompatible | Create conversion functions in `postgres/convert.go` |
| Handler behavior changes | Keep Echo framework to minimize HTTP layer changes |
| Database queries break | sqlc stays unchanged, only add wrapper layer |
| Deployment issues | Update configs before final phase, test in staging |

---

## Success Criteria

- [ ] All existing tests pass
- [ ] All routes function correctly
- [ ] Domain types are the source of truth
- [ ] Service interfaces enable mocking
- [ ] Error codes separate from HTTP status codes
- [ ] Single `routes.go` provides API visibility
- [ ] `run()` pattern enables integration testing
- [ ] Code follows CODE_STYLE_GUIDE.md patterns
