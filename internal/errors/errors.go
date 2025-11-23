package errors

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

// AppError represents a structured application error with context.
//
// Purpose:
// - Provide consistent error handling across the application
// - Preserve error context (what failed, why, related data)
// - Enable error tracking and aggregation (by error code)
// - Separate internal error details from user-facing messages
// - Support proper error wrapping and unwrapping
//
// Usage in handlers:
//   if err != nil {
//       return errors.NewInternalError("USER_CREATION_FAILED", "failed to create user", err).
//           WithField("email", req.Email)
//   }
type AppError struct {
	// Code is a unique identifier for this error type (e.g., "USER_NOT_FOUND")
	// Used for error tracking, metrics, and client-side error handling
	Code string `json:"code"`

	// Message is the user-facing error message
	// Should be safe to display to end users
	Message string `json:"message"`

	// Err is the underlying error (wrapped)
	// Contains internal details, not exposed to clients
	Err error `json:"-"`

	// StatusCode is the HTTP status code to return
	StatusCode int `json:"-"`

	// Fields contains additional context about the error
	// Used for logging and debugging, not exposed to clients
	Fields map[string]interface{} `json:"-"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	// TODO: Return formatted error string including code and message
	// TODO: Include wrapped error if present
	return ""
}

// Unwrap implements error unwrapping for errors.Is and errors.As.
func (e *AppError) Unwrap() error {
	// TODO: Return wrapped error
	return nil
}

// WithField adds a context field to the error.
//
// Purpose:
// - Add relevant data for debugging (user_id, email, etc.)
// - Chain multiple fields: err.WithField("a", 1).WithField("b", 2)
//
// Usage:
//   return err.WithField("user_id", userID).WithField("email", email)
func (e *AppError) WithField(key string, value interface{}) *AppError {
	// TODO: Add key-value pair to Fields map
	// TODO: Return self for chaining
	return e
}

// WithFields adds multiple context fields at once.
func (e *AppError) WithFields(fields map[string]interface{}) *AppError {
	// TODO: Merge fields map into Fields
	// TODO: Return self for chaining
	return e
}

// LogError logs the error with all context.
//
// Purpose:
// - Centralized error logging with consistent format
// - Include all error fields in log output
// - Use appropriate log level based on error severity
//
// Usage in handlers:
//   appErr.LogError(logger)
//   return echo.NewHTTPError(appErr.StatusCode, appErr.Message)
func (e *AppError) LogError(logger *slog.Logger) {
	// TODO: Create log attributes from Fields
	// TODO: Add error code and message
	// TODO: Add underlying error if present
	// TODO: Log at appropriate level based on status code
	//       - 5xx: Error level
	//       - 4xx: Warn or Info level
}

// ToEchoError converts AppError to Echo's HTTPError format.
//
// Purpose:
// - Integrate with Echo's error handling
// - Return appropriate HTTP response
//
// Returns error that Echo will convert to HTTP response.
func (e *AppError) ToEchoError() error {
	// TODO: Create echo.HTTPError with status code
	// TODO: Set message as response body
	// TODO: Optionally include error code in response
	return nil
}

// Common error constructors

// NewBadRequestError creates a 400 Bad Request error.
//
// Purpose:
// - Invalid input, validation failures
// - Client should modify request and retry
//
// Usage:
//   return errors.NewBadRequestError("INVALID_EMAIL", "email format is invalid", nil)
func NewBadRequestError(code, message string, err error) *AppError {
	// TODO: Create AppError with StatusCode 400
	return nil
}

// NewUnauthorizedError creates a 401 Unauthorized error.
//
// Purpose:
// - Authentication failures
// - Missing or invalid credentials
//
// Usage:
//   return errors.NewUnauthorizedError("INVALID_CREDENTIALS", "email or password is incorrect", nil)
func NewUnauthorizedError(code, message string, err error) *AppError {
	// TODO: Create AppError with StatusCode 401
	return nil
}

// NewForbiddenError creates a 403 Forbidden error.
//
// Purpose:
// - Authorization failures
// - Authenticated but lacks permission
//
// Usage:
//   return errors.NewForbiddenError("INSUFFICIENT_PERMISSIONS", "you don't have access to this organization", nil)
func NewForbiddenError(code, message string, err error) *AppError {
	// TODO: Create AppError with StatusCode 403
	return nil
}

// NewNotFoundError creates a 404 Not Found error.
//
// Purpose:
// - Resource doesn't exist
// - Should not reveal whether resource ever existed (security)
//
// Usage:
//   return errors.NewNotFoundError("INSPECTION_NOT_FOUND", "inspection not found", nil)
func NewNotFoundError(code, message string, err error) *AppError {
	// TODO: Create AppError with StatusCode 404
	return nil
}

// NewConflictError creates a 409 Conflict error.
//
// Purpose:
// - Resource already exists (unique constraint violations)
// - Concurrent modification conflicts
//
// Usage:
//   return errors.NewConflictError("EMAIL_EXISTS", "an account with this email already exists", err)
func NewConflictError(code, message string, err error) *AppError {
	// TODO: Create AppError with StatusCode 409
	return nil
}

// NewInternalError creates a 500 Internal Server Error.
//
// Purpose:
// - Unexpected errors
// - Database failures, external service failures
// - Should be logged and monitored
//
// Usage:
//   return errors.NewInternalError("DATABASE_ERROR", "failed to save inspection", err)
func NewInternalError(code, message string, err error) *AppError {
	// TODO: Create AppError with StatusCode 500
	return nil
}

// NewServiceUnavailableError creates a 503 Service Unavailable error.
//
// Purpose:
// - Temporary failures
// - Rate limit exceeded, quota exhausted
// - External service down
//
// Usage:
//   return errors.NewServiceUnavailableError("AI_SERVICE_DOWN", "AI service temporarily unavailable", err)
func NewServiceUnavailableError(code, message string, err error) *AppError {
	// TODO: Create AppError with StatusCode 503
	return nil
}

// Predefined common errors

var (
	// ErrInvalidInput represents generic input validation failure.
	ErrInvalidInput = NewBadRequestError("INVALID_INPUT", "invalid input provided", nil)

	// ErrUnauthorized represents authentication failure.
	ErrUnauthorized = NewUnauthorizedError("UNAUTHORIZED", "authentication required", nil)

	// ErrForbidden represents authorization failure.
	ErrForbidden = NewForbiddenError("FORBIDDEN", "access denied", nil)

	// ErrNotFound represents resource not found.
	ErrNotFound = NewNotFoundError("NOT_FOUND", "resource not found", nil)

	// ErrInternal represents unexpected internal error.
	ErrInternal = NewInternalError("INTERNAL_ERROR", "an internal error occurred", nil)
)

// ErrorHandlerMiddleware provides centralized error handling.
//
// Purpose:
// - Catch all errors returned by handlers
// - Log AppError instances with full context
// - Convert AppError to appropriate HTTP response
// - Handle unexpected panics
//
// Usage in main.go:
//   e.Use(errors.ErrorHandlerMiddleware(logger))
func ErrorHandlerMiddleware(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// TODO: Call next(c) and capture error
			// TODO: If error is AppError, log with context
			// TODO: If error is echo.HTTPError, log appropriately
			// TODO: If error is unexpected, log as internal error
			// TODO: Return appropriate HTTP response
			// TODO: Handle panics with defer/recover
			return nil
		}
	}
}

// WrapDatabaseError converts database errors to AppErrors.
//
// Purpose:
// - Translate pgx errors to appropriate AppError types
// - Handle common cases: not found, unique violations, etc.
// - Preserve original error for debugging
//
// Common mappings:
// - pgx.ErrNoRows -> NotFoundError
// - Unique constraint violation -> ConflictError
// - Other errors -> InternalError
//
// Usage in handlers:
//   user, err := queries.GetUserByID(ctx, userID)
//   if err != nil {
//       return errors.WrapDatabaseError(err, "USER_FETCH_FAILED", "failed to fetch user")
//   }
func WrapDatabaseError(err error, code, message string) *AppError {
	// TODO: Check if err is pgx.ErrNoRows -> return NotFoundError
	// TODO: Check if err is unique constraint violation -> return ConflictError
	// TODO: Check if err is foreign key violation -> return BadRequestError
	// TODO: Default: return InternalError
	return nil
}

// IsErrorCode checks if an error has a specific error code.
//
// Purpose:
// - Test error types in handlers and tests
// - Enable conditional logic based on error codes
//
// Usage:
//   if errors.IsErrorCode(err, "USER_NOT_FOUND") {
//       // handle specific error
//   }
func IsErrorCode(err error, code string) bool {
	// TODO: Use errors.As to extract AppError
	// TODO: Compare Code field
	return false
}
