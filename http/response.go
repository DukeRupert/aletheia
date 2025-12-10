package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Common HTTP response helpers that work with both JSON and HTMX requests.

// Respond sends a JSON response with the given status code and data.
func Respond(c echo.Context, status int, data any) error {
	return c.JSON(status, data)
}

// RespondOK sends a 200 OK response with the given data.
func RespondOK(c echo.Context, data any) error {
	return c.JSON(http.StatusOK, data)
}

// RespondCreated sends a 201 Created response with the given data.
func RespondCreated(c echo.Context, data any) error {
	return c.JSON(http.StatusCreated, data)
}

// RespondNoContent sends a 204 No Content response.
func RespondNoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// Redirect sends a redirect response.
// For HTMX requests, it uses HX-Redirect header.
// For regular requests, it uses standard HTTP redirect.
func Redirect(c echo.Context, url string) error {
	if IsHTMX(c) {
		c.Response().Header().Set("HX-Redirect", url)
		return c.NoContent(http.StatusOK)
	}
	return c.Redirect(http.StatusSeeOther, url)
}

// RefreshPage triggers a full page refresh for HTMX requests.
func RefreshPage(c echo.Context) error {
	if IsHTMX(c) {
		c.Response().Header().Set("HX-Refresh", "true")
		return c.NoContent(http.StatusOK)
	}
	return c.Redirect(http.StatusSeeOther, c.Request().URL.Path)
}

// TriggerEvent sends an HTMX trigger event.
func TriggerEvent(c echo.Context, event string) {
	c.Response().Header().Set("HX-Trigger", event)
}

// Retarget changes the target element for HTMX response.
func Retarget(c echo.Context, selector string) {
	c.Response().Header().Set("HX-Retarget", selector)
}

// Reswap changes the swap method for HTMX response.
func Reswap(c echo.Context, method string) {
	c.Response().Header().Set("HX-Reswap", method)
}

// SuccessResponse represents a simple success response.
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// RespondSuccess sends a success response with an optional message.
func RespondSuccess(c echo.Context, message string) error {
	return c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: message,
	})
}

// ListResponse represents a paginated list response.
type ListResponse[T any] struct {
	Data   []T `json:"data"`
	Total  int `json:"total"`
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// RespondList sends a paginated list response.
func RespondList[T any](c echo.Context, data []T, total, offset, limit int) error {
	return c.JSON(http.StatusOK, ListResponse[T]{
		Data:   data,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	})
}
