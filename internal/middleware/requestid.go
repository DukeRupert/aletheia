package middleware

import (
	"log/slog"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// RequestIDMiddleware adds a unique request ID to each incoming request for distributed tracing.
//
// Purpose:
// - Generate a unique UUID for each request
// - Add the request ID to the response header (X-Request-ID)
// - Store the request ID in the Echo context for use by handlers
// - Include the request ID in all log entries for this request
//
// This enables:
// - Correlating logs across the entire request lifecycle
// - Tracing requests through async operations (queue jobs, external API calls)
// - Debugging production issues by following a single request ID
//
// Usage in main.go:
//   e.Use(middleware.RequestIDMiddleware(logger))
func RequestIDMiddleware(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Generate unique request ID (UUID v4)
			requestID := uuid.New().String()

			// Add to response header: X-Request-ID
			c.Response().Header().Set(echo.HeaderXRequestID, requestID)

			// Store in Echo context for handlers to access
			c.Set("request_id", requestID)

			// Create child logger with request_id field for consistent logging
			requestLogger := logger.With(slog.String("request_id", requestID))

			// Store child logger in context for handlers to use
			c.Set("logger", requestLogger)

			// Continue with request processing
			return next(c)
		}
	}
}

// GetRequestID retrieves the request ID from the Echo context.
//
// Purpose:
// - Provide a helper function for handlers to get the current request ID
// - Use when enqueuing jobs to maintain correlation across async operations
//
// Usage in handlers:
//   requestID := middleware.GetRequestID(c)
//   job.CorrelationID = requestID
func GetRequestID(c echo.Context) string {
	// Retrieve request_id from context
	requestID, ok := c.Get("request_id").(string)
	if !ok {
		// Return empty string if not found or wrong type
		return ""
	}
	return requestID
}

// GetRequestLogger retrieves the request-scoped logger from the Echo context.
//
// Purpose:
// - Get a logger that automatically includes the request ID in all log entries
// - Ensures consistent logging across the request lifecycle
//
// Usage in handlers:
//   logger := middleware.GetRequestLogger(c)
//   logger.Info("processing request", slog.String("user_id", userID))
func GetRequestLogger(c echo.Context) *slog.Logger {
	// Retrieve logger from context
	logger, ok := c.Get("logger").(*slog.Logger)
	if !ok {
		// Fall back to default logger if not found
		return slog.Default()
	}
	return logger
}
