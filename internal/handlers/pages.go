package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// PageHandler handles template rendering for HTML pages
type PageHandler struct{}

// NewPageHandler creates a new page handler
func NewPageHandler() *PageHandler {
	return &PageHandler{}
}

// HomePage renders the home page
func (h *PageHandler) HomePage(c echo.Context) error {
	data := map[string]interface{}{
		"IsAuthenticated": false, // TODO: Check session
	}
	return c.Render(http.StatusOK, "home.html", data)
}

// NotFoundPage renders the 404 error page
func (h *PageHandler) NotFoundPage(c echo.Context) error {
	data := map[string]interface{}{
		"IsAuthenticated": false,
	}
	return c.Render(http.StatusNotFound, "404.html", data)
}

// ErrorPage renders the 500 error page
func (h *PageHandler) ErrorPage(c echo.Context) error {
	data := map[string]interface{}{
		"IsAuthenticated": false,
	}
	return c.Render(http.StatusInternalServerError, "500.html", data)
}

// LoginPage renders the login page
func (h *PageHandler) LoginPage(c echo.Context) error {
	data := map[string]interface{}{
		"IsAuthenticated": false,
	}
	return c.Render(http.StatusOK, "login.html", data)
}

// DashboardPage renders the dashboard page
func (h *PageHandler) DashboardPage(c echo.Context) error {
	// TODO: Get user from session
	data := map[string]interface{}{
		"IsAuthenticated": true,
		"User": map[string]interface{}{
			"Name": "User", // TODO: Get from session
		},
	}
	return c.Render(http.StatusOK, "dashboard.html", data)
}

// RegisterPage renders the registration page
func (h *PageHandler) RegisterPage(c echo.Context) error {
	data := map[string]interface{}{
		"IsAuthenticated": false,
	}
	return c.Render(http.StatusOK, "register.html", data)
}

// VerifyEmailPage renders the email verification page
func (h *PageHandler) VerifyEmailPage(c echo.Context) error {
	// Check if token is provided in query string
	token := c.QueryParam("token")

	data := map[string]interface{}{
		"IsAuthenticated": false,
		"Token":           token,
	}
	return c.Render(http.StatusOK, "verify.html", data)
}

// ForgotPasswordPage renders the forgot password page
func (h *PageHandler) ForgotPasswordPage(c echo.Context) error {
	data := map[string]interface{}{
		"IsAuthenticated": false,
	}
	return c.Render(http.StatusOK, "forgot-password.html", data)
}
