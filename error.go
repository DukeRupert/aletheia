package aletheia

import (
	"errors"
	"fmt"
)

// Domain error codes - transport layer maps these to HTTP status codes.
const (
	ECONFLICT     = "conflict"     // 409 - Resource already exists
	EINTERNAL     = "internal"     // 500 - Internal server error
	EINVALID      = "invalid"      // 400 - Invalid input
	ENOTFOUND     = "not_found"    // 404 - Resource not found
	EUNAUTHORIZED = "unauthorized" // 401 - Authentication required
	EFORBIDDEN    = "forbidden"    // 403 - Permission denied
	ERATELIMIT    = "rate_limit"   // 429 - Too many requests
)

// Error represents an application-specific error.
type Error struct {
	// Code is a machine-readable error code.
	Code string `json:"code"`

	// Message is a human-readable error message.
	Message string `json:"message"`

	// Fields contains field-specific validation errors.
	Fields map[string]string `json:"fields,omitempty"`

	// Err is the underlying error (not exposed to clients).
	Err error `json:"-"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *Error) Unwrap() error {
	return e.Err
}

// Errorf creates a new application error with a formatted message.
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

// ErrorCode extracts the error code from an error.
// Returns EINTERNAL if the error is not an *Error.
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
// Returns a generic message if the error is not an *Error.
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

// ErrorFields extracts field-specific errors from a validation error.
// Returns nil if the error has no field errors.
func ErrorFields(err error) map[string]string {
	var e *Error
	if errors.As(err, &e) {
		return e.Fields
	}
	return nil
}

// IsErrorCode checks if an error has the specified error code.
func IsErrorCode(err error, code string) bool {
	return ErrorCode(err) == code
}

// NotFound creates a not found error.
func NotFound(format string, args ...any) *Error {
	return Errorf(ENOTFOUND, format, args...)
}

// Invalid creates a validation error.
func Invalid(format string, args ...any) *Error {
	return Errorf(EINVALID, format, args...)
}

// Unauthorized creates an authentication error.
func Unauthorized(format string, args ...any) *Error {
	return Errorf(EUNAUTHORIZED, format, args...)
}

// Forbidden creates a permission error.
func Forbidden(format string, args ...any) *Error {
	return Errorf(EFORBIDDEN, format, args...)
}

// Conflict creates a conflict error.
func Conflict(format string, args ...any) *Error {
	return Errorf(ECONFLICT, format, args...)
}

// Internal creates an internal error, wrapping the underlying cause.
func Internal(message string, err error) *Error {
	return WrapError(EINTERNAL, message, err)
}
