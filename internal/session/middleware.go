package session

import (
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

// SessionMiddleware validates the session cookie and attaches user ID to context
func SessionMiddleware(db *pgxpool.Pool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get session token from cookie
			cookie, err := c.Cookie(SessionCookieName)
			if err != nil {
				// No cookie found - user not authenticated
				return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
			}

			// Validate session
			session, err := GetSession(c.Request().Context(), db, cookie.Value)
			if err != nil {
				if err == pgx.ErrNoRows {
					// Session not found or expired
					return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired session")
				}
				// Database error
				return echo.NewHTTPError(http.StatusInternalServerError, "session validation failed")
			}

			// Attach user ID (convert from pgtype.UUID to uuid.UUID) and session to context
			c.Set(string(UserIDKey), session.UserID.Bytes)
			c.Set(string(SessionKey), session)

			return next(c)
		}
	}
}

// OptionalSessionMiddleware checks for a session but doesn't require it
func OptionalSessionMiddleware(db *pgxpool.Pool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get session token from cookie
			cookie, err := c.Cookie(SessionCookieName)
			if err != nil {
				// No cookie found - continue without auth
				return next(c)
			}

			// Try to validate session
			session, err := GetSession(c.Request().Context(), db, cookie.Value)
			if err == nil {
				// Session valid - attach to context (convert from pgtype.UUID to uuid.UUID)
				c.Set(string(UserIDKey), session.UserID.Bytes)
				c.Set(string(SessionKey), session)
			}
			// If session is invalid, continue without auth

			return next(c)
		}
	}
}

// GetUserID retrieves the user ID from the Echo context
func GetUserID(c echo.Context) (uuid.UUID, bool) {
	val := c.Get(string(UserIDKey))
	if val == nil {
		return uuid.UUID{}, false
	}

	// The value is stored as [16]byte from pgtype.UUID.Bytes
	if bytes, ok := val.([16]byte); ok {
		return uuid.UUID(bytes), true
	}

	// Fallback: try direct uuid.UUID cast
	if userID, ok := val.(uuid.UUID); ok {
		return userID, true
	}

	return uuid.UUID{}, false
}

// GetSessionData retrieves the session data from the Echo context
func GetSessionData(c echo.Context) (database.Session, bool) {
	session, ok := c.Get(string(SessionKey)).(database.Session)
	return session, ok
}
