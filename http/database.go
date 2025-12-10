package http

import (
	"errors"

	"github.com/dukerupert/aletheia"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// WrapDatabaseError converts database errors to domain errors.
//
// Common mappings:
//   - pgx.ErrNoRows -> ENOTFOUND
//   - unique_violation (23505) -> ECONFLICT
//   - foreign_key_violation (23503) -> EINVALID
//   - not_null_violation (23502) -> EINVALID
//   - check_violation (23514) -> EINVALID
//   - Other errors -> EINTERNAL
//
// Usage:
//
//	user, err := queries.GetUser(ctx, id)
//	if err != nil {
//	    return WrapDatabaseError(err, "User not found", "Failed to fetch user")
//	}
func WrapDatabaseError(err error, notFoundMsg, internalMsg string) error {
	if err == nil {
		return nil
	}

	// Check for not found
	if errors.Is(err, pgx.ErrNoRows) {
		return aletheia.NotFound("%s", notFoundMsg)
	}

	// Check for PostgreSQL-specific errors
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return aletheia.Conflict("Resource already exists")
		case "23503": // foreign_key_violation
			return aletheia.Invalid("Referenced resource does not exist")
		case "23502": // not_null_violation
			return aletheia.Invalid("Required field is missing")
		case "23514": // check_violation
			return aletheia.Invalid("Value violates constraint")
		}
	}

	// Default: internal error
	return aletheia.Internal(internalMsg, err)
}

// IsNotFound checks if an error is a not found error.
func IsNotFound(err error) bool {
	return aletheia.IsErrorCode(err, aletheia.ENOTFOUND)
}

// IsConflict checks if an error is a conflict error.
func IsConflict(err error) bool {
	return aletheia.IsErrorCode(err, aletheia.ECONFLICT)
}

// IsInvalid checks if an error is a validation error.
func IsInvalid(err error) bool {
	return aletheia.IsErrorCode(err, aletheia.EINVALID)
}

// IsUnauthorized checks if an error is an unauthorized error.
func IsUnauthorized(err error) bool {
	return aletheia.IsErrorCode(err, aletheia.EUNAUTHORIZED)
}

// IsForbidden checks if an error is a forbidden error.
func IsForbidden(err error) bool {
	return aletheia.IsErrorCode(err, aletheia.EFORBIDDEN)
}
