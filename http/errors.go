package http

import (
	"log/slog"
	"net/http"

	"github.com/dukerupert/aletheia"
	"github.com/labstack/echo/v4"
)

// errorStatusCode maps domain error codes to HTTP status codes.
func errorStatusCode(code string) int {
	switch code {
	case aletheia.ENOTFOUND:
		return http.StatusNotFound
	case aletheia.EINVALID:
		return http.StatusBadRequest
	case aletheia.EUNAUTHORIZED:
		return http.StatusUnauthorized
	case aletheia.EFORBIDDEN:
		return http.StatusForbidden
	case aletheia.ECONFLICT:
		return http.StatusConflict
	case aletheia.ERATELIMIT:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

// ErrorResponse represents the JSON error response format.
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// HandleError converts domain errors to appropriate HTTP responses.
// It logs internal errors and returns user-safe messages.
func HandleError(c echo.Context, logger *slog.Logger, err error) error {
	code := aletheia.ErrorCode(err)
	message := aletheia.ErrorMessage(err)
	fields := aletheia.ErrorFields(err)
	status := errorStatusCode(code)

	// Log internal errors with full details
	if code == aletheia.EINTERNAL {
		logger.Error("internal error",
			slog.String("error", err.Error()),
			slog.String("path", c.Path()),
			slog.String("method", c.Request().Method),
		)
		// Don't expose internal error details to clients
		message = "An internal error occurred."
	}

	// For HTMX requests, we might want to render an error partial
	if IsHTMX(c) {
		return handleHTMXError(c, status, code, message, fields)
	}

	// For API requests, return JSON
	return c.JSON(status, ErrorResponse{
		Error:   code,
		Message: message,
		Fields:  fields,
	})
}

// handleHTMXError handles errors for HTMX requests.
// It can either return a partial or use HX-Retarget to show errors.
func handleHTMXError(c echo.Context, status int, code, message string, fields map[string]string) error {
	// For validation errors with fields, we might want to re-render the form
	// For now, just return JSON that HTMX can handle
	return c.JSON(status, ErrorResponse{
		Error:   code,
		Message: message,
		Fields:  fields,
	})
}

// IsHTMX returns true if the request is an HTMX request.
func IsHTMX(c echo.Context) bool {
	return c.Request().Header.Get("HX-Request") == "true"
}

// WantsJSON returns true if the client prefers JSON responses.
func WantsJSON(c echo.Context) bool {
	accept := c.Request().Header.Get("Accept")
	return accept == "application/json"
}

// ErrorHandlerMiddleware provides centralized error handling.
// It converts domain errors to appropriate HTTP responses.
func ErrorHandlerMiddleware(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err == nil {
				return nil
			}

			// Check if it's already an Echo HTTP error
			if he, ok := err.(*echo.HTTPError); ok {
				// Log and pass through Echo errors
				if he.Code >= 500 {
					logger.Error("http error",
						slog.Int("status", he.Code),
						slog.Any("message", he.Message),
						slog.String("path", c.Path()),
					)
				}
				return err
			}

			// Check if it's a domain error
			if aletheia.ErrorCode(err) != aletheia.EINTERNAL || isAletheiaError(err) {
				return HandleError(c, logger, err)
			}

			// Wrap unexpected errors as internal errors
			wrapped := aletheia.Internal("An unexpected error occurred", err)
			return HandleError(c, logger, wrapped)
		}
	}
}

// isAletheiaError checks if the error is an aletheia.Error type.
func isAletheiaError(err error) bool {
	_, ok := err.(*aletheia.Error)
	return ok
}
