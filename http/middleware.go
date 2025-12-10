package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/dukerupert/aletheia"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	// SessionCookieName is the name of the session cookie.
	SessionCookieName = "session"

	// Default timeout for database operations.
	DefaultTimeout = 5 * time.Second
)

// registerMiddleware sets up all middleware for the server.
func (s *Server) registerMiddleware() {
	// Recovery middleware
	s.echo.Use(middleware.Recover())

	// Request ID middleware
	s.echo.Use(middleware.RequestID())

	// Logger middleware with request ID
	s.echo.Use(s.requestLoggerMiddleware())

	// CORS middleware (configure as needed)
	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "HX-Request", "HX-Target", "HX-Trigger"},
	}))

	// Custom error handler
	s.echo.HTTPErrorHandler = s.httpErrorHandler
}

// requestLoggerMiddleware creates a middleware that logs requests with context.
func (s *Server) requestLoggerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			requestID := c.Response().Header().Get(echo.HeaderXRequestID)

			// Create request-scoped logger
			logger := s.logger.With(
				slog.String("request_id", requestID),
				slog.String("method", c.Request().Method),
				slog.String("path", c.Path()),
			)
			c.Set("logger", logger)

			err := next(c)

			// Log request completion
			duration := time.Since(start)
			status := c.Response().Status

			logAttrs := []any{
				slog.Int("status", status),
				slog.Duration("duration", duration),
			}

			if err != nil {
				logAttrs = append(logAttrs, slog.String("error", err.Error()))
				logger.Error("request failed", logAttrs...)
			} else if status >= 500 {
				logger.Error("request completed with server error", logAttrs...)
			} else if status >= 400 {
				logger.Warn("request completed with client error", logAttrs...)
			} else {
				logger.Info("request completed", logAttrs...)
			}

			return err
		}
	}
}

// httpErrorHandler handles errors and returns appropriate responses.
func (s *Server) httpErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	// Check if it's an Echo HTTP error
	if he, ok := err.(*echo.HTTPError); ok {
		msg := he.Message
		if m, ok := msg.(string); ok {
			_ = HandleError(c, s.logger, echo.NewHTTPError(he.Code, m))
		} else {
			_ = c.JSON(he.Code, map[string]any{"error": msg})
		}
		return
	}

	// Handle domain errors
	_ = HandleError(c, s.logger, err)
}

// SessionMiddleware validates session and attaches user to context.
// If required is true, returns 401 for missing/invalid sessions.
func (s *Server) SessionMiddleware(required bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			logger := s.getRequestLogger(c)

			// Get session token from cookie
			cookie, err := c.Cookie(SessionCookieName)
			if err != nil {
				if required {
					logger.Debug("session required but no cookie found")
					return aletheia.Unauthorized("Authentication required")
				}
				return next(c)
			}

			// Validate session
			session, err := s.sessionService.FindSessionByTokenWithUser(c.Request().Context(), cookie.Value)
			if err != nil {
				if required {
					if aletheia.IsErrorCode(err, aletheia.EUNAUTHORIZED) {
						logger.Debug("session expired or invalid")
						return err
					}
					logger.Error("session validation failed", slog.String("error", err.Error()))
					return aletheia.Internal("Failed to validate session", err)
				}
				// Optional session - continue without auth
				return next(c)
			}

			// Attach user to context
			ctx := aletheia.NewContextWithUser(c.Request().Context(), session.User)
			ctx = aletheia.NewContextWithSession(ctx, session)
			c.SetRequest(c.Request().WithContext(ctx))

			// Also set in Echo context for backward compatibility
			c.Set("user_id", session.UserID)
			c.Set("user", session.User)
			c.Set("session", session)

			return next(c)
		}
	}
}

// RequireAuth is a middleware that requires authentication.
func (s *Server) RequireAuth() echo.MiddlewareFunc {
	return s.SessionMiddleware(true)
}

// OptionalAuth is a middleware that checks for authentication but doesn't require it.
func (s *Server) OptionalAuth() echo.MiddlewareFunc {
	return s.SessionMiddleware(false)
}

// RequireOrgMembership is a middleware that requires the user to be a member of the organization.
// The organization ID is extracted from the route parameter specified by paramName.
func (s *Server) RequireOrgMembership(paramName string, allowedRoles ...aletheia.OrganizationRole) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()

			// Get user from context
			user := aletheia.UserFromContext(ctx)
			if user == nil {
				return aletheia.Unauthorized("Authentication required")
			}

			// Get organization ID from route parameter
			orgIDStr := c.Param(paramName)
			if orgIDStr == "" {
				return aletheia.Invalid("Organization ID is required")
			}

			orgID, err := uuid.Parse(orgIDStr)
			if err != nil {
				return aletheia.Invalid("Invalid organization ID format")
			}

			// Check membership
			member, err := s.organizationService.RequireMembership(ctx, orgID, user.ID, allowedRoles...)
			if err != nil {
				return err
			}

			// Get the organization for context
			org, err := s.organizationService.FindOrganizationByID(ctx, orgID)
			if err != nil {
				return err
			}

			// Attach organization context
			ctx = aletheia.NewContextWithOrganization(ctx, org)
			c.SetRequest(c.Request().WithContext(ctx))
			c.Set("org_member", member)

			return next(c)
		}
	}
}

// getRequestLogger retrieves the request-scoped logger from context.
func (s *Server) getRequestLogger(c echo.Context) *slog.Logger {
	if logger, ok := c.Get("logger").(*slog.Logger); ok {
		return logger
	}
	return s.logger
}

// getUserID retrieves the user ID from the Echo context.
func getUserID(c echo.Context) (uuid.UUID, bool) {
	if user := aletheia.UserFromContext(c.Request().Context()); user != nil {
		return user.ID, true
	}
	if val := c.Get("user_id"); val != nil {
		if id, ok := val.(uuid.UUID); ok {
			return id, true
		}
	}
	return uuid.UUID{}, false
}
