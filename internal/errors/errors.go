package errors

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	// Build error string with code and message
	var parts []string
	if e.Code != "" {
		parts = append(parts, fmt.Sprintf("[%s]", e.Code))
	}
	if e.Message != "" {
		parts = append(parts, e.Message)
	}

	result := strings.Join(parts, " ")

	// Include wrapped error if present
	if e.Err != nil {
		result = fmt.Sprintf("%s: %v", result, e.Err)
	}

	return result
}

// Unwrap implements error unwrapping for errors.Is and errors.As.
func (e *AppError) Unwrap() error {
	return e.Err
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
	// Initialize Fields map if nil
	if e.Fields == nil {
		e.Fields = make(map[string]interface{})
	}

	// Add key-value pair to Fields map
	e.Fields[key] = value

	// Return self for chaining
	return e
}

// WithFields adds multiple context fields at once.
func (e *AppError) WithFields(fields map[string]interface{}) *AppError {
	// Initialize Fields map if nil
	if e.Fields == nil {
		e.Fields = make(map[string]interface{})
	}

	// Merge fields map into Fields
	for k, v := range fields {
		e.Fields[k] = v
	}

	// Return self for chaining
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
	// Create log attributes from Fields
	attrs := make([]slog.Attr, 0, len(e.Fields)+3)

	// Add error code and message
	attrs = append(attrs, slog.String("error_code", e.Code))
	attrs = append(attrs, slog.String("error_message", e.Message))

	// Add underlying error if present
	if e.Err != nil {
		attrs = append(attrs, slog.String("underlying_error", e.Err.Error()))
	}

	// Add all Fields
	for k, v := range e.Fields {
		attrs = append(attrs, slog.Any(k, v))
	}

	// Log at appropriate level based on status code
	if e.StatusCode >= 500 {
		// 5xx errors are server errors - log at Error level
		logger.LogAttrs(nil, slog.LevelError, "application error", attrs...)
	} else if e.StatusCode >= 400 {
		// 4xx errors are client errors - log at Warn level
		logger.LogAttrs(nil, slog.LevelWarn, "client error", attrs...)
	} else {
		// Other errors - log at Info level
		logger.LogAttrs(nil, slog.LevelInfo, "error", attrs...)
	}
}

// ToEchoError converts AppError to Echo's HTTPError format.
//
// Purpose:
// - Integrate with Echo's error handling
// - Return appropriate HTTP response
//
// Returns error that Echo will convert to HTTP response.
func (e *AppError) ToEchoError() error {
	// Create response body with code and message
	response := map[string]interface{}{
		"code":    e.Code,
		"message": e.Message,
	}

	// Create echo.HTTPError with status code and response body
	return echo.NewHTTPError(e.StatusCode, response)
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
	return &AppError{
		Code:       code,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusBadRequest,
		Fields:     make(map[string]interface{}),
	}
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
	return &AppError{
		Code:       code,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusUnauthorized,
		Fields:     make(map[string]interface{}),
	}
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
	return &AppError{
		Code:       code,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusForbidden,
		Fields:     make(map[string]interface{}),
	}
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
	return &AppError{
		Code:       code,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusNotFound,
		Fields:     make(map[string]interface{}),
	}
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
	return &AppError{
		Code:       code,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusConflict,
		Fields:     make(map[string]interface{}),
	}
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
	return &AppError{
		Code:       code,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusInternalServerError,
		Fields:     make(map[string]interface{}),
	}
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
	return &AppError{
		Code:       code,
		Message:    message,
		Err:        err,
		StatusCode: http.StatusServiceUnavailable,
		Fields:     make(map[string]interface{}),
	}
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
			// Handle panics with defer/recover
			defer func() {
				if r := recover(); r != nil {
					// Log panic
					logger.Error("panic recovered",
						slog.Any("panic", r),
						slog.String("path", c.Path()),
						slog.String("method", c.Request().Method))

					// Return internal server error
					err := NewInternalError("PANIC", "an unexpected error occurred", fmt.Errorf("%v", r))
					c.JSON(err.StatusCode, err.ToEchoError())
				}
			}()

			// Call next handler and capture error
			err := next(c)
			if err == nil {
				return nil
			}

			// Check if error is AppError
			var appErr *AppError
			if errors.As(err, &appErr) {
				// Log with full context
				appErr.LogError(logger)
				// Return as Echo error
				return appErr.ToEchoError()
			}

			// Check if error is echo.HTTPError
			var echoErr *echo.HTTPError
			if errors.As(err, &echoErr) {
				// Log echo HTTP errors
				logger.Warn("HTTP error",
					slog.Int("status", echoErr.Code),
					slog.Any("message", echoErr.Message),
					slog.String("path", c.Path()))
				return err
			}

			// Unexpected error - wrap as internal error
			internalErr := NewInternalError("UNEXPECTED_ERROR", "an internal error occurred", err)
			internalErr.LogError(logger)
			return internalErr.ToEchoError()
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
	// Check if err is pgx.ErrNoRows -> return NotFoundError
	if errors.Is(err, pgx.ErrNoRows) {
		return NewNotFoundError(code, message, err)
	}

	// Check if err is a PostgreSQL error
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// Check error code for specific constraint violations
		switch pgErr.Code {
		case "23505": // unique_violation
			return NewConflictError(code, message, err)
		case "23503": // foreign_key_violation
			return NewBadRequestError(code, message, err)
		case "23502": // not_null_violation
			return NewBadRequestError(code, message, err)
		case "23514": // check_violation
			return NewBadRequestError(code, message, err)
		}
	}

	// Default: return InternalError for unknown database errors
	return NewInternalError(code, message, err)
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
	// Use errors.As to extract AppError
	var appErr *AppError
	if errors.As(err, &appErr) {
		// Compare Code field
		return appErr.Code == code
	}
	return false
}
