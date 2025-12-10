# Go HTTP Service Patterns — Hybrid Style Guide

> A synthesis of Mat Ryer's HTTP patterns and Ben Johnson's Standard Package Layout
> Designed for Go + htmx + Alpine.js + Tailwind CSS applications

---

## Table of Contents

1. [Philosophy](#1-philosophy)
2. [Project Structure](#2-project-structure)
3. [Domain Layer (Root Package)](#3-domain-layer-root-package)
4. [Application Entry Point](#4-application-entry-point)
5. [HTTP Layer](#5-http-layer)
6. [Storage Layer](#6-storage-layer)
7. [Testing Strategy](#7-testing-strategy)
8. [Error Handling](#8-error-handling)
9. [Validation](#9-validation)
10. [Middleware](#10-middleware)
11. [Configuration](#11-configuration)
12. [htmx Integration](#12-htmx-integration)
13. [Quick Reference](#13-quick-reference)
14. [Checklists](#14-checklists)

---

## 1. Philosophy

### Core Principles

1. **Packages are layers, not groups.** Organize by dependency, not by type.
2. **Domain first.** Define what your app does before how it does it.
3. **Explicit dependencies.** Handlers ask for what they need as arguments.
4. **Test at boundaries.** Unit test layers in isolation, integration test the whole.
5. **Errors are domain concepts.** Define error codes in domain, map at transport.

### Design Flow

```
1. Design domain language (root package)
2. Define service interfaces (root package)
3. Implement storage layer (sqlite/, postgres/)
4. Implement transport layer (http/)
5. Wire together in binary (cmd/)
```

---

## 2. Project Structure

```
myapp/                          # Root: domain types & service interfaces
├── user.go                     # User, UserService, UserFilter, UserUpdate
├── order.go                    # Order, OrderService, etc.
├── error.go                    # Error type and codes (EINVALID, ENOTFOUND, etc.)
├── context.go                  # Context helpers (UserFromContext, etc.)
├── validate.go                 # Validator interface
├── go.mod
│
├── cmd/
│   └── myappd/                 # Server binary
│       └── main.go             # main() calls run(), wires everything
│
├── sqlite/                     # Storage implementation
│   ├── sqlite.go               # DB struct, Open/Close, migrations
│   ├── user.go                 # UserService implementation
│   └── order.go
│
├── http/                       # HTTP transport layer
│   ├── server.go               # Server struct, Open/Close
│   ├── routes.go               # addRoutes() — all routes in one place
│   ├── handlers.go             # Common handler helpers
│   ├── user.go                 # User handlers
│   ├── order.go                # Order handlers
│   ├── middleware.go           # Middleware functions
│   └── templates/              # HTML templates (for htmx)
│       ├── base.html
│       ├── users/
│       │   ├── index.html
│       │   └── _list.html      # Partial for htmx
│       └── components/
│           └── _flash.html
│
├── mock/                       # Mock implementations
│   ├── user.go
│   └── order.go
│
└── assets/                     # Static files (CSS, JS)
    ├── css/
    └── js/
```

### File Naming Conventions

| File | Contents |
|------|----------|
| `entity.go` | Type, Service interface, Filter, Update |
| `sqlite/entity.go` | Service implementation for that entity |
| `http/entity.go` | HTTP handlers for that entity |
| `mock/entity.go` | Mock service for that entity |

---

## 3. Domain Layer (Root Package)

The root package defines your domain language. It has **zero external dependencies**.

### Entity Type

```go
// user.go
package myapp

import "time"

// User represents a user in the system.
type User struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    APIKey    string    `json:"-"` // Hidden from JSON
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}

// Validate returns an error if the user has invalid fields.
func (u *User) Validate() error {
    if u.Name == "" {
        return Errorf(EINVALID, "User name is required.")
    }
    if u.Email == "" {
        return Errorf(EINVALID, "User email is required.")
    }
    return nil
}
```

### Service Interface

```go
// UserService represents a service for managing users.
type UserService interface {
    // FindUserByID retrieves a user by ID.
    // Returns ENOTFOUND if user does not exist.
    FindUserByID(ctx context.Context, id int) (*User, error)

    // FindUsers retrieves users matching the filter.
    // Returns total count which may differ from len(users) if Limit is set.
    FindUsers(ctx context.Context, filter UserFilter) ([]*User, int, error)

    // CreateUser creates a new user.
    // On success, user.ID is set to the new ID.
    CreateUser(ctx context.Context, user *User) error

    // UpdateUser updates an existing user.
    // Returns ENOTFOUND if user doesn't exist.
    // Returns EUNAUTHORIZED if current user cannot update this user.
    UpdateUser(ctx context.Context, id int, upd UserUpdate) (*User, error)

    // DeleteUser permanently deletes a user.
    // Returns ENOTFOUND if user doesn't exist.
    // Returns EUNAUTHORIZED if current user cannot delete this user.
    DeleteUser(ctx context.Context, id int) error
}
```

### Filter Type (for queries)

```go
// UserFilter represents a filter for FindUsers().
type UserFilter struct {
    // Filtering fields (nil means "don't filter by this")
    ID     *int    `json:"id"`
    Email  *string `json:"email"`
    APIKey *string `json:"apiKey"`

    // Pagination
    Offset int `json:"offset"`
    Limit  int `json:"limit"`
}
```

### Update Type (for partial updates)

```go
// UserUpdate represents fields to update via UpdateUser().
// Pointer fields: nil = don't update, non-nil = update to this value.
type UserUpdate struct {
    Name  *string `json:"name"`
    Email *string `json:"email"`
}
```

### Context Helpers

```go
// context.go
package myapp

import "context"

type contextKey int

const (
    userContextKey contextKey = iota + 1
    flashContextKey
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

// UserIDFromContext returns the authenticated user's ID, or 0.
func UserIDFromContext(ctx context.Context) int {
    if user := UserFromContext(ctx); user != nil {
        return user.ID
    }
    return 0
}

// NewContextWithFlash attaches a flash message to context.
func NewContextWithFlash(ctx context.Context, msg string) context.Context {
    return context.WithValue(ctx, flashContextKey, msg)
}

// FlashFromContext returns the flash message, or empty string.
func FlashFromContext(ctx context.Context) string {
    msg, _ := ctx.Value(flashContextKey).(string)
    return msg
}
```

---

## 4. Application Entry Point

Use Mat Ryer's `run()` pattern for testability.

```go
// cmd/myappd/main.go
package main

import (
    "context"
    "fmt"
    "io"
    "os"
    "os/signal"
    "sync"
    "time"

    "myapp/http"
    "myapp/sqlite"
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

    // Initialize database
    db := sqlite.NewDB(cfg.DatabaseURL)
    if err := db.Open(); err != nil {
        return fmt.Errorf("database: %w", err)
    }
    defer db.Close()

    // Initialize HTTP server with all dependencies
    srv := http.NewServer(
        db.UserService,
        db.OrderService,
        cfg.HTTP,
    )
    if err := srv.Open(); err != nil {
        return fmt.Errorf("http server: %w", err)
    }

    fmt.Fprintf(stdout, "listening on %s\n", cfg.HTTP.Addr)

    // Wait for shutdown signal, then gracefully stop
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
    }()

    wg.Wait()
    return nil
}
```

---

## 5. HTTP Layer

### Server Structure

Combine Ben's server struct with Mat's handler pattern.

```go
// http/server.go
package http

import (
    "context"
    "net"
    "net/http"
    "time"

    "myapp"
)

// Server holds dependencies and manages the HTTP lifecycle.
type Server struct {
    // TCP listener and underlying server
    ln     net.Listener
    server *http.Server

    // Configuration
    Addr   string
    Domain string

    // Services (set by NewServer)
    userService  myapp.UserService
    orderService myapp.OrderService
}

// NewServer creates a server with all dependencies.
func NewServer(
    userService myapp.UserService,
    orderService myapp.OrderService,
    cfg HTTPConfig,
) *Server {
    s := &Server{
        Addr:         cfg.Addr,
        Domain:       cfg.Domain,
        userService:  userService,
        orderService: orderService,
    }

    // Build handler with routes and middleware
    handler := s.buildHandler()

    s.server = &http.Server{
        Handler:      handler,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    return s
}

func (s *Server) buildHandler() http.Handler {
    mux := http.NewServeMux()

    // Register all routes (see routes.go)
    s.addRoutes(mux)

    // Apply global middleware (outermost first)
    var handler http.Handler = mux
    handler = s.recoverPanic(handler)
    handler = s.logRequest(handler)
    handler = s.authenticate(handler)

    return handler
}

// Open starts the server.
func (s *Server) Open() error {
    var err error
    s.ln, err = net.Listen("tcp", s.Addr)
    if err != nil {
        return err
    }

    go s.server.Serve(s.ln)
    return nil
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
    return s.server.Shutdown(ctx)
}

// URL returns the server's base URL (useful for tests).
func (s *Server) URL() string {
    if s.ln == nil {
        return ""
    }
    return "http://" + s.ln.Addr().String()
}
```

### Routes File

**All routes in one place.** This is crucial for understanding your API surface.

```go
// http/routes.go
package http

import "net/http"

func (s *Server) addRoutes(mux *http.ServeMux) {
    // Static files
    mux.Handle("GET /static/", http.StripPrefix("/static/", 
        http.FileServer(http.Dir("assets"))))

    // Health check
    mux.HandleFunc("GET /healthz", s.handleHealthz)

    // Pages (return full HTML)
    mux.Handle("GET /", s.handleIndex())
    mux.Handle("GET /users", s.requireAuth(s.handleUserIndex()))
    mux.Handle("GET /users/{id}", s.requireAuth(s.handleUserShow()))
    mux.Handle("GET /users/new", s.requireAuth(s.handleUserNew()))
    mux.Handle("GET /users/{id}/edit", s.requireAuth(s.handleUserEdit()))

    // Actions (htmx endpoints, return partials or redirect)
    mux.Handle("POST /users", s.requireAuth(s.handleUserCreate()))
    mux.Handle("PUT /users/{id}", s.requireAuth(s.handleUserUpdate()))
    mux.Handle("DELETE /users/{id}", s.requireAuth(s.handleUserDelete()))

    // API (JSON endpoints)
    mux.Handle("GET /api/v1/users", s.requireAuth(s.handleAPIUserIndex()))
    mux.Handle("GET /api/v1/users/{id}", s.requireAuth(s.handleAPIUserShow()))
    mux.Handle("POST /api/v1/users", s.requireAuth(s.handleAPIUserCreate()))

    // Catch-all
    mux.Handle("/", http.NotFoundHandler())
}
```

### Handler Pattern

**Handlers are functions that return `http.Handler`.** Dependencies are explicit arguments.

```go
// http/user.go
package http

import (
    "net/http"

    "myapp"
)

// handleUserIndex returns a handler for GET /users.
// Dependencies are explicit — easy to test, easy to understand.
func (s *Server) handleUserIndex() http.Handler {
    // One-time setup (runs when routes are registered)
    type templateData struct {
        Users []*myapp.User
        Flash string
    }

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Fetch data using service
        users, _, err := s.userService.FindUsers(r.Context(), myapp.UserFilter{
            Limit: 100,
        })
        if err != nil {
            s.error(w, r, err)
            return
        }

        // Render template
        s.render(w, r, http.StatusOK, "users/index.html", templateData{
            Users: users,
            Flash: myapp.FlashFromContext(r.Context()),
        })
    })
}

// handleUserShow returns a handler for GET /users/{id}.
func (s *Server) handleUserShow() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id, err := parseIDParam(r, "id")
        if err != nil {
            s.error(w, r, err)
            return
        }

        user, err := s.userService.FindUserByID(r.Context(), id)
        if err != nil {
            s.error(w, r, err)
            return
        }

        s.render(w, r, http.StatusOK, "users/show.html", user)
    })
}

// handleUserCreate returns a handler for POST /users.
func (s *Server) handleUserCreate() http.Handler {
    type request struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    }

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Decode request (form or JSON based on Content-Type)
        var req request
        if err := s.decode(r, &req); err != nil {
            s.error(w, r, err)
            return
        }

        // Create domain object
        user := &myapp.User{
            Name:  req.Name,
            Email: req.Email,
        }

        // Call service
        if err := s.userService.CreateUser(r.Context(), user); err != nil {
            s.error(w, r, err)
            return
        }

        // Respond based on request type
        if isHTMX(r) {
            // htmx: return partial or trigger redirect
            w.Header().Set("HX-Redirect", "/users")
            w.WriteHeader(http.StatusCreated)
        } else if wantsJSON(r) {
            // API: return JSON
            s.json(w, http.StatusCreated, user)
        } else {
            // Browser: redirect
            http.Redirect(w, r, "/users", http.StatusSeeOther)
        }
    })
}
```

### Handler Helpers

Centralized encoding/decoding and response helpers.

```go
// http/handlers.go
package http

import (
    "encoding/json"
    "html/template"
    "net/http"
    "path/filepath"
    "strconv"

    "myapp"
)

// Templates (loaded once, cached)
var templates *template.Template

func init() {
    templates = template.Must(template.ParseGlob(
        filepath.Join("http", "templates", "**", "*.html"),
    ))
}

// decode reads the request body into v.
// Handles both JSON and form data based on Content-Type.
func (s *Server) decode(r *http.Request, v any) error {
    contentType := r.Header.Get("Content-Type")

    switch {
    case contentType == "application/json":
        if err := json.NewDecoder(r.Body).Decode(v); err != nil {
            return myapp.Errorf(myapp.EINVALID, "Invalid JSON: %v", err)
        }
    default:
        if err := r.ParseForm(); err != nil {
            return myapp.Errorf(myapp.EINVALID, "Invalid form data: %v", err)
        }
        if err := decodeForm(r.Form, v); err != nil {
            return myapp.Errorf(myapp.EINVALID, "Invalid form data: %v", err)
        }
    }
    return nil
}

// decodeValid decodes and validates in one step.
func decodeValid[T myapp.Validator](s *Server, r *http.Request) (T, error) {
    var v T
    if err := s.decode(r, &v); err != nil {
        return v, err
    }
    if problems := v.Valid(r.Context()); len(problems) > 0 {
        return v, myapp.Errorf(myapp.EINVALID, "Validation failed: %v", problems)
    }
    return v, nil
}

// json writes a JSON response.
func (s *Server) json(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    if err := json.NewEncoder(w).Encode(v); err != nil {
        s.logError(err)
    }
}

// render executes a template.
func (s *Server) render(w http.ResponseWriter, r *http.Request, status int, tmpl string, data any) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(status)
    if err := templates.ExecuteTemplate(w, tmpl, data); err != nil {
        s.logError(err)
    }
}

// error handles errors by mapping domain codes to HTTP status.
func (s *Server) error(w http.ResponseWriter, r *http.Request, err error) {
    code := myapp.ErrorCode(err)
    message := myapp.ErrorMessage(err)

    // Log internal errors
    if code == myapp.EINTERNAL {
        s.logError(err)
        message = "An internal error occurred."
    }

    // Map domain error to HTTP status
    status := errorStatusCode(code)

    // Respond based on request type
    if wantsJSON(r) {
        s.json(w, status, map[string]string{"error": message})
    } else if isHTMX(r) {
        // htmx: return error partial
        w.WriteHeader(status)
        s.render(w, r, status, "components/_error.html", message)
    } else {
        http.Error(w, message, status)
    }
}

// errorStatusCode maps domain error codes to HTTP status codes.
func errorStatusCode(code string) int {
    switch code {
    case myapp.ECONFLICT:
        return http.StatusConflict
    case myapp.EINVALID:
        return http.StatusBadRequest
    case myapp.ENOTFOUND:
        return http.StatusNotFound
    case myapp.EUNAUTHORIZED:
        return http.StatusUnauthorized
    default:
        return http.StatusInternalServerError
    }
}

// parseIDParam extracts an int path parameter.
func parseIDParam(r *http.Request, name string) (int, error) {
    idStr := r.PathValue(name)
    id, err := strconv.Atoi(idStr)
    if err != nil {
        return 0, myapp.Errorf(myapp.EINVALID, "Invalid %s: must be an integer", name)
    }
    return id, nil
}

// isHTMX returns true if this is an htmx request.
func isHTMX(r *http.Request) bool {
    return r.Header.Get("HX-Request") == "true"
}

// wantsJSON returns true if client prefers JSON.
func wantsJSON(r *http.Request) bool {
    accept := r.Header.Get("Accept")
    return accept == "application/json"
}

func (s *Server) logError(err error) {
    // Replace with your logger
    println("ERROR:", err.Error())
}
```

---

## 6. Storage Layer

### Database Wrapper

```go
// sqlite/sqlite.go
package sqlite

import (
    "context"
    "database/sql"
    "embed"
    "fmt"

    _ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps the SQL database and exposes services.
type DB struct {
    db  *sql.DB
    dsn string

    // Services (initialized in NewDB)
    UserService  *UserService
    OrderService *OrderService
}

// NewDB creates a new database wrapper.
func NewDB(dsn string) *DB {
    db := &DB{dsn: dsn}
    db.UserService = &UserService{db: db}
    db.OrderService = &OrderService{db: db}
    return db
}

// Open connects to the database and runs migrations.
func (db *DB) Open() error {
    var err error
    db.db, err = sql.Open("sqlite3", db.dsn)
    if err != nil {
        return fmt.Errorf("open database: %w", err)
    }

    if err := db.migrate(); err != nil {
        return fmt.Errorf("migrate: %w", err)
    }

    return nil
}

// Close closes the database connection.
func (db *DB) Close() error {
    if db.db != nil {
        return db.db.Close()
    }
    return nil
}

// BeginTx starts a transaction.
func (db *DB) BeginTx(ctx context.Context) (*sql.Tx, error) {
    return db.db.BeginTx(ctx, nil)
}
```

### Service Implementation

```go
// sqlite/user.go
package sqlite

import (
    "context"
    "database/sql"
    "strings"

    "myapp"
)

// Compile-time check: UserService implements myapp.UserService.
var _ myapp.UserService = (*UserService)(nil)

// UserService implements myapp.UserService using SQLite.
type UserService struct {
    db *DB
}

func (s *UserService) FindUserByID(ctx context.Context, id int) (*myapp.User, error) {
    users, _, err := s.FindUsers(ctx, myapp.UserFilter{ID: &id})
    if err != nil {
        return nil, err
    }
    if len(users) == 0 {
        return nil, myapp.Errorf(myapp.ENOTFOUND, "User not found.")
    }
    return users[0], nil
}

func (s *UserService) FindUsers(ctx context.Context, filter myapp.UserFilter) ([]*myapp.User, int, error) {
    // Build query
    where, args := []string{"1=1"}, []any{}

    if filter.ID != nil {
        where = append(where, "id = ?")
        args = append(args, *filter.ID)
    }
    if filter.Email != nil {
        where = append(where, "email = ?")
        args = append(args, *filter.Email)
    }
    if filter.APIKey != nil {
        where = append(where, "api_key = ?")
        args = append(args, *filter.APIKey)
    }

    // Count total
    var total int
    countQuery := `SELECT COUNT(*) FROM users WHERE ` + strings.Join(where, " AND ")
    if err := s.db.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
        return nil, 0, err
    }

    // Fetch rows
    query := `SELECT id, name, email, api_key, created_at, updated_at 
              FROM users 
              WHERE ` + strings.Join(where, " AND ") + `
              ORDER BY id
              LIMIT ? OFFSET ?`
    args = append(args, filter.Limit, filter.Offset)

    rows, err := s.db.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, 0, err
    }
    defer rows.Close()

    var users []*myapp.User
    for rows.Next() {
        var u myapp.User
        if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.APIKey, &u.CreatedAt, &u.UpdatedAt); err != nil {
            return nil, 0, err
        }
        users = append(users, &u)
    }

    return users, total, rows.Err()
}

func (s *UserService) CreateUser(ctx context.Context, user *myapp.User) error {
    // Validate
    if err := user.Validate(); err != nil {
        return err
    }

    // Insert
    result, err := s.db.db.ExecContext(ctx,
        `INSERT INTO users (name, email, api_key, created_at, updated_at)
         VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
        user.Name, user.Email, user.APIKey,
    )
    if err != nil {
        return err
    }

    id, err := result.LastInsertId()
    if err != nil {
        return err
    }
    user.ID = int(id)

    return nil
}

func (s *UserService) UpdateUser(ctx context.Context, id int, upd myapp.UserUpdate) (*myapp.User, error) {
    // Fetch existing
    user, err := s.FindUserByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // Authorization check
    if currentUserID := myapp.UserIDFromContext(ctx); currentUserID != user.ID {
        return nil, myapp.Errorf(myapp.EUNAUTHORIZED, "You cannot update this user.")
    }

    // Apply updates
    if upd.Name != nil {
        user.Name = *upd.Name
    }
    if upd.Email != nil {
        user.Email = *upd.Email
    }

    // Validate updated user
    if err := user.Validate(); err != nil {
        return user, err
    }

    // Save
    _, err = s.db.db.ExecContext(ctx,
        `UPDATE users SET name = ?, email = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
        user.Name, user.Email, user.ID,
    )
    if err != nil {
        return user, err
    }

    return user, nil
}

func (s *UserService) DeleteUser(ctx context.Context, id int) error {
    // Fetch to verify exists + auth
    user, err := s.FindUserByID(ctx, id)
    if err != nil {
        return err
    }

    if currentUserID := myapp.UserIDFromContext(ctx); currentUserID != user.ID {
        return myapp.Errorf(myapp.EUNAUTHORIZED, "You cannot delete this user.")
    }

    _, err = s.db.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
    return err
}
```

---

## 7. Testing Strategy

**Dual approach:** Mock-based unit tests for layers + integration tests via `run()`.

### Mock Package

```go
// mock/user.go
package mock

import (
    "context"

    "myapp"
)

var _ myapp.UserService = (*UserService)(nil)

// UserService is a mock. Set function fields to control behavior.
type UserService struct {
    FindUserByIDFn func(ctx context.Context, id int) (*myapp.User, error)
    FindUsersFn    func(ctx context.Context, filter myapp.UserFilter) ([]*myapp.User, int, error)
    CreateUserFn   func(ctx context.Context, user *myapp.User) error
    UpdateUserFn   func(ctx context.Context, id int, upd myapp.UserUpdate) (*myapp.User, error)
    DeleteUserFn   func(ctx context.Context, id int) error
}

func (s *UserService) FindUserByID(ctx context.Context, id int) (*myapp.User, error) {
    if s.FindUserByIDFn != nil {
        return s.FindUserByIDFn(ctx, id)
    }
    return nil, myapp.Errorf(myapp.ENOTFOUND, "User not found.")
}

func (s *UserService) FindUsers(ctx context.Context, filter myapp.UserFilter) ([]*myapp.User, int, error) {
    if s.FindUsersFn != nil {
        return s.FindUsersFn(ctx, filter)
    }
    return nil, 0, nil
}

func (s *UserService) CreateUser(ctx context.Context, user *myapp.User) error {
    if s.CreateUserFn != nil {
        return s.CreateUserFn(ctx, user)
    }
    user.ID = 1
    return nil
}

func (s *UserService) UpdateUser(ctx context.Context, id int, upd myapp.UserUpdate) (*myapp.User, error) {
    if s.UpdateUserFn != nil {
        return s.UpdateUserFn(ctx, id, upd)
    }
    return &myapp.User{ID: id}, nil
}

func (s *UserService) DeleteUser(ctx context.Context, id int) error {
    if s.DeleteUserFn != nil {
        return s.DeleteUserFn(ctx, id)
    }
    return nil
}
```

### HTTP Handler Tests (Unit)

```go
// http/user_test.go
package http_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "myapp"
    apphttp "myapp/http"
    "myapp/mock"
)

func TestHandleUserShow(t *testing.T) {
    t.Parallel()

    // Setup mock
    userService := &mock.UserService{
        FindUserByIDFn: func(ctx context.Context, id int) (*myapp.User, error) {
            if id == 1 {
                return &myapp.User{ID: 1, Name: "Alice", Email: "alice@test.com"}, nil
            }
            return nil, myapp.Errorf(myapp.ENOTFOUND, "User not found.")
        },
    }

    // Create server with mock
    srv := apphttp.NewServer(userService, nil, apphttp.HTTPConfig{})

    t.Run("existing user", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/users/1", nil)
        rec := httptest.NewRecorder()

        srv.ServeHTTP(rec, req)

        if rec.Code != http.StatusOK {
            t.Errorf("expected 200, got %d", rec.Code)
        }
    })

    t.Run("missing user", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/users/999", nil)
        rec := httptest.NewRecorder()

        srv.ServeHTTP(rec, req)

        if rec.Code != http.StatusNotFound {
            t.Errorf("expected 404, got %d", rec.Code)
        }
    })
}
```

### Integration Tests (End-to-End)

```go
// cmd/myappd/main_test.go
package main

import (
    "context"
    "io"
    "net/http"
    "testing"
    "time"
)

func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx, cancel := context.WithCancel(context.Background())
    t.Cleanup(cancel)

    // Use test database
    getenv := func(key string) string {
        if key == "DATABASE_URL" {
            return ":memory:"
        }
        if key == "HTTP_ADDR" {
            return ":0" // Random port
        }
        return ""
    }

    // Start server in background
    errCh := make(chan error, 1)
    go func() {
        errCh <- run(ctx, io.Discard, io.Discard, nil, getenv)
    }()

    // Wait for server to be ready
    if err := waitForReady(ctx, "http://localhost:8080/healthz", 5*time.Second); err != nil {
        t.Fatalf("server not ready: %v", err)
    }

    // Run tests against live server
    t.Run("health check", func(t *testing.T) {
        resp, err := http.Get("http://localhost:8080/healthz")
        if err != nil {
            t.Fatal(err)
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            t.Errorf("expected 200, got %d", resp.StatusCode)
        }
    })
}

func waitForReady(ctx context.Context, url string, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
        resp, err := http.DefaultClient.Do(req)
        if err == nil && resp.StatusCode == http.StatusOK {
            resp.Body.Close()
            return nil
        }
        if resp != nil {
            resp.Body.Close()
        }
        time.Sleep(100 * time.Millisecond)
    }
    return context.DeadlineExceeded
}
```

---

## 8. Error Handling

### Domain Errors

```go
// error.go
package myapp

import (
    "errors"
    "fmt"
)

// Application error codes.
// These map to HTTP status codes at the transport layer.
const (
    ECONFLICT       = "conflict"        // 409
    EINTERNAL       = "internal"        // 500
    EINVALID        = "invalid"         // 400
    ENOTFOUND       = "not_found"       // 404
    EUNAUTHORIZED   = "unauthorized"    // 401
    EFORBIDDEN      = "forbidden"       // 403
)

// Error represents an application-specific error.
type Error struct {
    Code    string            // Machine-readable code
    Message string            // Human-readable message
    Fields  map[string]string // Field-specific errors (for validation)
}

func (e *Error) Error() string {
    return e.Message
}

// Errorf creates a new application error.
func Errorf(code string, format string, args ...any) *Error {
    return &Error{
        Code:    code,
        Message: fmt.Sprintf(format, args...),
    }
}

// ErrorWithFields creates a validation error with field-specific messages.
func ErrorWithFields(fields map[string]string) *Error {
    return &Error{
        Code:    EINVALID,
        Message: "Validation failed.",
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

// ErrorMessage extracts the message from an error.
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

---

## 9. Validation

### Validator Interface

```go
// validate.go
package myapp

import "context"

// Validator can validate itself.
type Validator interface {
    // Valid returns a map of field names to error messages.
    // Empty map (or nil) means valid.
    Valid(ctx context.Context) map[string]string
}
```

### Request Validation

```go
// Example request type with validation
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (r CreateUserRequest) Valid(ctx context.Context) map[string]string {
    problems := make(map[string]string)

    if r.Name == "" {
        problems["name"] = "Name is required."
    } else if len(r.Name) > 100 {
        problems["name"] = "Name must be 100 characters or less."
    }

    if r.Email == "" {
        problems["email"] = "Email is required."
    } else if !strings.Contains(r.Email, "@") {
        problems["email"] = "Email must be valid."
    }

    return problems
}
```

### Using Validation in Handlers

```go
func (s *Server) handleUserCreate() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Decode and validate in one step
        req, err := decodeValid[CreateUserRequest](s, r)
        if err != nil {
            s.error(w, r, err)
            return
        }

        // req is now guaranteed valid
        user := &myapp.User{
            Name:  req.Name,
            Email: req.Email,
        }

        // ...
    })
}
```

---

## 10. Middleware

### Pattern

Middleware wraps handlers. Use the adapter pattern for middleware with dependencies.

```go
// http/middleware.go
package http

import (
    "log"
    "net/http"
    "time"

    "myapp"
)

// Simple middleware (no dependencies)
func (s *Server) recoverPanic(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("panic: %v", err)
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}

// Middleware with logging
func (s *Server) logRequest(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // Wrap response writer to capture status
        wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

        next.ServeHTTP(wrapped, r)

        log.Printf("%s %s %d %v",
            r.Method, r.URL.Path, wrapped.status, time.Since(start))
    })
}

type responseWriter struct {
    http.ResponseWriter
    status int
}

func (w *responseWriter) WriteHeader(status int) {
    w.status = status
    w.ResponseWriter.WriteHeader(status)
}

// Authentication middleware
func (s *Server) authenticate(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract token from cookie or header
        token := extractToken(r)
        if token == "" {
            next.ServeHTTP(w, r)
            return
        }

        // Look up user by API key
        users, _, err := s.userService.FindUsers(r.Context(), myapp.UserFilter{
            APIKey: &token,
            Limit:  1,
        })
        if err != nil || len(users) == 0 {
            next.ServeHTTP(w, r)
            return
        }

        // Attach user to context
        ctx := myapp.NewContextWithUser(r.Context(), users[0])
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Authorization middleware (requires authenticated user)
func (s *Server) requireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if myapp.UserFromContext(r.Context()) == nil {
            if wantsJSON(r) {
                s.json(w, http.StatusUnauthorized, map[string]string{
                    "error": "Authentication required.",
                })
            } else {
                http.Redirect(w, r, "/login", http.StatusSeeOther)
            }
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

---

## 11. Configuration

Config types live in `cmd/` since they're operator concerns.

```go
// cmd/myappd/config.go
package main

import (
    "fmt"
    "os"
    "strconv"
    "time"
)

type Config struct {
    DatabaseURL string
    HTTP        HTTPConfig
}

type HTTPConfig struct {
    Addr         string
    Domain       string
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
}

func loadConfig(getenv func(string) string) (*Config, error) {
    cfg := &Config{
        DatabaseURL: getenv("DATABASE_URL"),
        HTTP: HTTPConfig{
            Addr:         getenv("HTTP_ADDR"),
            Domain:       getenv("HTTP_DOMAIN"),
            ReadTimeout:  parseDuration(getenv("HTTP_READ_TIMEOUT"), 15*time.Second),
            WriteTimeout: parseDuration(getenv("HTTP_WRITE_TIMEOUT"), 15*time.Second),
        },
    }

    // Defaults
    if cfg.DatabaseURL == "" {
        cfg.DatabaseURL = "myapp.db"
    }
    if cfg.HTTP.Addr == "" {
        cfg.HTTP.Addr = ":8080"
    }

    // Validation
    if cfg.HTTP.Domain == "" && os.Getenv("GO_ENV") == "production" {
        return nil, fmt.Errorf("HTTP_DOMAIN required in production")
    }

    return cfg, nil
}

func parseDuration(s string, fallback time.Duration) time.Duration {
    if s == "" {
        return fallback
    }
    d, err := time.ParseDuration(s)
    if err != nil {
        return fallback
    }
    return d
}
```

---

## 12. htmx Integration

### Response Patterns

```go
// http/htmx.go
package http

import "net/http"

// htmx response headers
const (
    HXRedirect  = "HX-Redirect"   // Client-side redirect
    HXRefresh   = "HX-Refresh"    // Refresh current page
    HXTrigger   = "HX-Trigger"    // Trigger client event
    HXRetarget  = "HX-Retarget"   // Change target element
    HXReswap    = "HX-Reswap"     // Change swap method
)

// redirect sends an htmx redirect or standard redirect.
func (s *Server) redirect(w http.ResponseWriter, r *http.Request, url string) {
    if isHTMX(r) {
        w.Header().Set(HXRedirect, url)
        w.WriteHeader(http.StatusOK)
    } else {
        http.Redirect(w, r, url, http.StatusSeeOther)
    }
}

// trigger sends an htmx trigger event.
func (s *Server) trigger(w http.ResponseWriter, event string) {
    w.Header().Set(HXTrigger, event)
}

// renderPartial renders just the partial for htmx, or full page for browsers.
func (s *Server) renderPartial(w http.ResponseWriter, r *http.Request, status int, partial, full string, data any) {
    if isHTMX(r) {
        s.render(w, r, status, partial, data)
    } else {
        s.render(w, r, status, full, data)
    }
}
```

### Template Structure

```html
<!-- templates/base.html -->
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{block "title" .}}MyApp{{end}}</title>
    <script src="https://unpkg.com/htmx.org@2.0.0"></script>
    <script defer src="https://unpkg.com/alpinejs@3.x.x/dist/cdn.min.js"></script>
    <link href="/static/css/output.css" rel="stylesheet">
</head>
<body hx-boost="true">
    {{template "content" .}}
</body>
</html>

<!-- templates/users/index.html -->
{{define "title"}}Users{{end}}
{{define "content"}}
<div class="container mx-auto p-4">
    <h1 class="text-2xl font-bold mb-4">Users</h1>

    <div id="user-list">
        {{template "users/_list.html" .Users}}
    </div>
</div>
{{end}}

<!-- templates/users/_list.html (partial for htmx) -->
<ul class="space-y-2">
    {{range .}}
    <li class="p-2 bg-white rounded shadow">
        <a href="/users/{{.ID}}" class="text-blue-600 hover:underline">
            {{.Name}}
        </a>
        <span class="text-gray-500">{{.Email}}</span>
    </li>
    {{else}}
    <li class="text-gray-500">No users found.</li>
    {{end}}
</ul>
```

### Form Handling with htmx

```html
<!-- templates/users/new.html -->
{{define "content"}}
<form hx-post="/users" 
      hx-target="#form-errors"
      hx-swap="innerHTML"
      class="max-w-md mx-auto p-4">
    
    <div id="form-errors"></div>

    <div class="mb-4">
        <label for="name" class="block text-sm font-medium">Name</label>
        <input type="text" 
               name="name" 
               id="name"
               class="mt-1 block w-full rounded border-gray-300"
               required>
    </div>

    <div class="mb-4">
        <label for="email" class="block text-sm font-medium">Email</label>
        <input type="email" 
               name="email" 
               id="email"
               class="mt-1 block w-full rounded border-gray-300"
               required>
    </div>

    <button type="submit" 
            class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700">
        Create User
    </button>
</form>
{{end}}
```

---

## 13. Quick Reference

### Package Responsibilities

| Package | Contains | Imports |
|---------|----------|---------|
| `myapp/` (root) | Domain types, interfaces, errors | stdlib only |
| `sqlite/` | Storage implementation | root, database driver |
| `http/` | Handlers, middleware, templates | root, stdlib http |
| `mock/` | Mock services for testing | root |
| `cmd/myappd/` | main, config, wiring | all packages |

### Handler Checklist

```go
func (s *Server) handleThing() http.Handler {
    // 1. Define request/response types (if handler-specific)
    type request struct { ... }

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 2. Parse path parameters
        // 3. Decode request body
        // 4. Validate input
        // 5. Call service
        // 6. Handle errors
        // 7. Respond (JSON, HTML, or redirect)
    })
}
```

### Error Flow

```
Handler → Service → Domain Error
                         ↓
              HTTP Layer maps to status code
                         ↓
              JSON or HTML error response
```

---

## 14. Checklists

### New Entity Checklist

- [ ] `myapp/entity.go` — Type, Validate(), Service interface, Filter, Update
- [ ] `myapp/error.go` — Add any new error codes if needed
- [ ] `sqlite/entity.go` — Service implementation
- [ ] `sqlite/migrations/` — Add migration file
- [ ] `mock/entity.go` — Mock service
- [ ] `http/entity.go` — Handlers
- [ ] `http/routes.go` — Register routes
- [ ] `http/templates/entities/` — Templates
- [ ] `cmd/myappd/main.go` — Wire service if needed
- [ ] Tests for each layer

### New Endpoint Checklist

- [ ] Add route to `http/routes.go`
- [ ] Create handler function returning `http.Handler`
- [ ] Define request type (if POST/PUT)
- [ ] Implement validation if needed
- [ ] Handle htmx vs JSON vs HTML response
- [ ] Add template if HTML endpoint
- [ ] Add tests

### Pre-Deploy Checklist

- [ ] All tests pass (`go test ./...`)
- [ ] No race conditions (`go test -race ./...`)
- [ ] Config validated for production
- [ ] Database migrations applied
- [ ] Health check endpoint works
- [ ] Graceful shutdown tested

---

## Appendix: File Templates

### New Entity Template

```go
// myapp/thing.go
package myapp

import (
    "context"
    "time"
)

// Thing represents a thing in the system.
type Thing struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    UserID    int       `json:"userId"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}

// Validate returns an error if the thing has invalid fields.
func (t *Thing) Validate() error {
    if t.Name == "" {
        return Errorf(EINVALID, "Thing name is required.")
    }
    return nil
}

// ThingService represents a service for managing things.
type ThingService interface {
    FindThingByID(ctx context.Context, id int) (*Thing, error)
    FindThings(ctx context.Context, filter ThingFilter) ([]*Thing, int, error)
    CreateThing(ctx context.Context, thing *Thing) error
    UpdateThing(ctx context.Context, id int, upd ThingUpdate) (*Thing, error)
    DeleteThing(ctx context.Context, id int) error
}

// ThingFilter represents a filter for FindThings().
type ThingFilter struct {
    ID     *int `json:"id"`
    UserID *int `json:"userId"`
    Offset int  `json:"offset"`
    Limit  int  `json:"limit"`
}

// ThingUpdate represents fields to update.
type ThingUpdate struct {
    Name *string `json:"name"`
}
```