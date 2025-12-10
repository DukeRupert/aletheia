package http

import (
	"context"
	"log/slog"
	"time"

	"github.com/dukerupert/aletheia"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// requestTimeout returns the default timeout for handler operations.
func requestTimeout() time.Duration {
	return DefaultTimeout
}

// withTimeout creates a context with a timeout for handler operations.
func withTimeout(c echo.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request().Context(), requestTimeout())
}

// parseUUID parses a UUID from a string, returning a domain error if invalid.
func parseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, aletheia.Invalid("Invalid ID format")
	}
	return id, nil
}

// requireParam extracts a required route parameter, returning error if empty.
func requireParam(c echo.Context, name string) (string, error) {
	value := c.Param(name)
	if value == "" {
		return "", aletheia.Invalid("%s is required", name)
	}
	return value, nil
}

// requireUUIDParam extracts and parses a required UUID route parameter.
func requireUUIDParam(c echo.Context, name string) (uuid.UUID, error) {
	value, err := requireParam(c, name)
	if err != nil {
		return uuid.UUID{}, err
	}
	return parseUUID(value)
}

// requireUser extracts the authenticated user from context.
func requireUser(c echo.Context) (*aletheia.User, error) {
	user := aletheia.UserFromContext(c.Request().Context())
	if user == nil {
		return nil, aletheia.Unauthorized("Authentication required")
	}
	return user, nil
}

// requireUserID extracts the authenticated user's ID from context.
func requireUserID(c echo.Context) (uuid.UUID, error) {
	user, err := requireUser(c)
	if err != nil {
		return uuid.UUID{}, err
	}
	return user.ID, nil
}

// bind binds the request body to a struct and validates it.
func bind(c echo.Context, v interface{}) error {
	if err := c.Bind(v); err != nil {
		return aletheia.Invalid("Invalid request body")
	}
	if err := c.Validate(v); err != nil {
		return err
	}
	return nil
}

// log returns the request-scoped logger.
func (s *Server) log(c echo.Context) *slog.Logger {
	return s.getRequestLogger(c)
}

// Placeholder handlers - these will be implemented in separate files

// Health handlers
func (s *Server) handleHealthCheck(c echo.Context) error {
	return RespondOK(c, map[string]string{"status": "ok"})
}

func (s *Server) handleLivenessCheck(c echo.Context) error {
	return RespondOK(c, map[string]string{"status": "alive"})
}

func (s *Server) handleReadinessCheck(c echo.Context) error {
	return RespondOK(c, map[string]string{"status": "ready"})
}
