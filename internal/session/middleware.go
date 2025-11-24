package session

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/dukerupert/aletheia/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

// ContextKey is the type for context keys
type ContextKey string

const (
	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "user_id"
	// SessionKey is the context key for session data
	SessionKey ContextKey = "session"
)

// sessionValidator defines interface for validating session tokens
type sessionValidator interface {
	ValidateSession(ctx context.Context, token string) (database.Session, error)
}

// dbValidator implements sessionValidator using direct database access
type dbValidator struct {
	db *pgxpool.Pool
}

func (v *dbValidator) ValidateSession(ctx context.Context, token string) (database.Session, error) {
	return GetSession(ctx, v.db, token)
}

// cacheValidator implements sessionValidator using cached access
type cacheValidator struct {
	cache *SessionCache
}

func (v *cacheValidator) ValidateSession(ctx context.Context, token string) (database.Session, error) {
	return v.cache.GetSession(ctx, token)
}

// getRequestLogger retrieves the request-scoped logger from context.
// Falls back to default logger if not found (e.g., if RequestIDMiddleware isn't installed).
func getRequestLogger(c echo.Context) *slog.Logger {
	// Try to get logger from context (set by RequestIDMiddleware)
	if logger, ok := c.Get("logger").(*slog.Logger); ok {
		return logger
	}
	// Fallback to default logger
	return slog.Default()
}

// sessionMiddleware is the common middleware implementation
// required: if true, returns 401 for missing/invalid sessions; if false, continues without auth
func sessionMiddleware(validator sessionValidator, required bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get session token from cookie
			cookie, err := c.Cookie(SessionCookieName)
			if err != nil {
				if required {
					// No cookie found - user not authenticated
					logger := getRequestLogger(c)
					logger.Debug("session authentication required but no cookie found",
						slog.String("path", c.Path()),
						slog.String("method", c.Request().Method),
						slog.String("reason", "missing_cookie"))
					return echo.NewHTTPError(http.StatusUnauthorized, "authentication required: no session cookie found")
				}
				// Optional session - continue without auth
				return next(c)
			}

			// Validate session
			session, err := validator.ValidateSession(c.Request().Context(), cookie.Value)
			if err != nil {
				logger := getRequestLogger(c)
				if required {
					if err == pgx.ErrNoRows {
						// Session not found or expired
						logger.Warn("session validation failed: session not found or expired",
							slog.String("path", c.Path()),
							slog.String("method", c.Request().Method),
							slog.String("reason", "session_not_found"))
						return echo.NewHTTPError(http.StatusUnauthorized, "session has expired or is invalid, please log in again")
					}
					// Database error
					logger.Error("session validation failed: database error",
						slog.String("path", c.Path()),
						slog.String("method", c.Request().Method),
						slog.String("error_type", "database_error"),
						slog.String("error", err.Error()))
					return echo.NewHTTPError(http.StatusInternalServerError, "unable to validate session, please try again")
				}
				// Optional session with error - log but continue without auth
				logger.Debug("optional session validation failed, continuing without auth",
					slog.String("path", c.Path()),
					slog.String("reason", "validation_error"),
					slog.String("error", err.Error()))
				return next(c)
			}

			// Attach user ID and session to context
			c.Set(string(UserIDKey), uuid.UUID(session.UserID.Bytes))
			c.Set(string(SessionKey), session)

			return next(c)
		}
	}
}

// SessionMiddleware validates the session cookie and attaches user ID to context
// Uses database directly (no caching) - for backward compatibility
func SessionMiddleware(db *pgxpool.Pool) echo.MiddlewareFunc {
	return sessionMiddleware(&dbValidator{db: db}, true)
}

// CachedSessionMiddleware validates the session cookie using cache, falling back to DB
// Provides significant performance improvement by avoiding DB query on every request
func CachedSessionMiddleware(cache *SessionCache) echo.MiddlewareFunc {
	return sessionMiddleware(&cacheValidator{cache: cache}, true)
}

// OptionalSessionMiddleware checks for a session but doesn't require it
// Uses database directly (no caching) - for backward compatibility
func OptionalSessionMiddleware(db *pgxpool.Pool) echo.MiddlewareFunc {
	return sessionMiddleware(&dbValidator{db: db}, false)
}

// OptionalCachedSessionMiddleware checks for a session using cache but doesn't require it
func OptionalCachedSessionMiddleware(cache *SessionCache) echo.MiddlewareFunc {
	return sessionMiddleware(&cacheValidator{cache: cache}, false)
}

// GetUserID retrieves the user ID from the Echo context
func GetUserID(c echo.Context) (uuid.UUID, bool) {
	val := c.Get(string(UserIDKey))
	if val == nil {
		return uuid.UUID{}, false
	}

	// The value is stored as uuid.UUID
	userID, ok := val.(uuid.UUID)
	return userID, ok
}

// GetSessionData retrieves the session data from the Echo context
func GetSessionData(c echo.Context) (database.Session, bool) {
	session, ok := c.Get(string(SessionKey)).(database.Session)
	return session, ok
}
