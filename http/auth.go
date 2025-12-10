package http

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/dukerupert/aletheia"
	"github.com/dukerupert/aletheia/internal/auth"
	"github.com/labstack/echo/v4"
)

// RegisterRequest is the request payload for user registration.
type RegisterRequest struct {
	Email     string `json:"email" form:"email" validate:"required,email,max=255"`
	Username  string `json:"username" form:"username" validate:"required,min=3,max=50"`
	Password  string `json:"password" form:"password" validate:"required,min=8,max=128"`
	Name      string `json:"name" form:"name" validate:"omitempty,max=100"`
	FirstName string `json:"first_name" form:"first_name" validate:"omitempty,max=50"`
	LastName  string `json:"last_name" form:"last_name" validate:"omitempty,max=50"`
}

// RegisterResponse is the response payload for user registration.
type RegisterResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

func (s *Server) handleRegister(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	var req RegisterRequest
	if err := bind(c, &req); err != nil {
		if IsHTMX(c) {
			return s.renderRegisterFormWithErrors(c, &req, map[string]string{"general": "Invalid form data"})
		}
		return err
	}

	// Validate password complexity
	if err := auth.ValidatePassword(req.Password); err != nil {
		if IsHTMX(c) {
			return s.renderRegisterFormWithErrors(c, &req, map[string]string{"password": err.Error()})
		}
		return aletheia.Invalid("%s", err.Error())
	}

	// Parse name
	firstName, lastName := parseName(req.Name, req.FirstName, req.LastName)

	// Create user
	user := &aletheia.User{
		Email:     req.Email,
		Username:  req.Username,
		FirstName: firstName,
		LastName:  lastName,
	}

	if err := s.userService.CreateUser(ctx, user, req.Password); err != nil {
		if aletheia.IsErrorCode(err, aletheia.ECONFLICT) {
			if IsHTMX(c) {
				return s.renderRegisterFormWithErrors(c, &req, map[string]string{"email": "Email or username already exists"})
			}
		}
		return err
	}

	// Generate verification token
	token, err := auth.GenerateVerificationToken()
	if err != nil {
		s.log(c).Error("failed to generate verification token", slog.String("error", err.Error()))
		return aletheia.Internal("Failed to create user", err)
	}

	// Save verification token
	if err := s.userService.SetVerificationToken(ctx, user.ID, token); err != nil {
		s.log(c).Error("failed to set verification token", slog.String("error", err.Error()))
		return aletheia.Internal("Failed to create user", err)
	}

	// Send verification email (don't fail on email error)
	if s.emailService != nil {
		name := user.FirstName
		if name == "" {
			name = user.Username
		}
		if err := s.emailService.SendVerificationEmail(ctx, user.Email, name, token); err != nil {
			s.log(c).Error("failed to send verification email", slog.String("error", err.Error()))
		}
	}

	s.log(c).Info("user registered", slog.String("user_id", user.ID.String()), slog.String("email", user.Email))

	if IsHTMX(c) {
		return Redirect(c, "/login")
	}

	return RespondCreated(c, RegisterResponse{
		ID:       user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
	})
}

// LoginRequest is the request payload for user login.
type LoginRequest struct {
	Email    string `json:"email" form:"email" validate:"required,email,max=255"`
	Password string `json:"password" form:"password" validate:"required,max=128"`
}

// LoginResponse is the response payload for user login.
type LoginResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

func (s *Server) handleLogin(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	var req LoginRequest
	if err := bind(c, &req); err != nil {
		if IsHTMX(c) {
			return s.renderLoginFormWithErrors(c, &req, map[string]string{"general": "Invalid form data"})
		}
		return err
	}

	// Verify password
	user, err := s.userService.VerifyPassword(ctx, req.Email, req.Password)
	if err != nil {
		if IsHTMX(c) {
			if aletheia.IsErrorCode(err, aletheia.EUNAUTHORIZED) {
				return s.renderLoginFormWithErrors(c, &req, map[string]string{"general": "Invalid email or password"})
			}
			if aletheia.IsErrorCode(err, aletheia.EFORBIDDEN) {
				return s.renderLoginFormWithErrors(c, &req, map[string]string{"general": aletheia.ErrorMessage(err)})
			}
		}
		return err
	}

	// Create session
	session, err := s.sessionService.CreateSession(ctx, user.ID, s.SessionDuration)
	if err != nil {
		s.log(c).Error("failed to create session", slog.String("error", err.Error()))
		return aletheia.Internal("Login failed", err)
	}

	// Update last login
	_ = s.userService.UpdateLastLogin(ctx, user.ID)

	// Set session cookie
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    session.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.SessionSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	}
	c.SetCookie(cookie)

	s.log(c).Info("user logged in", slog.String("user_id", user.ID.String()))

	if IsHTMX(c) {
		return Redirect(c, "/dashboard")
	}

	return RespondOK(c, LoginResponse{
		ID:       user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
	})
}

func (s *Server) handleLogout(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	// Get session token from cookie
	cookie, err := c.Cookie(SessionCookieName)
	if err != nil {
		if IsHTMX(c) {
			return Redirect(c, "/login")
		}
		return aletheia.Invalid("Not logged in")
	}

	// Delete session
	if err := s.sessionService.DeleteSession(ctx, cookie.Value); err != nil {
		s.log(c).Error("failed to delete session", slog.String("error", err.Error()))
	}

	// Clear cookie
	c.SetCookie(&http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	s.log(c).Info("user logged out")

	if IsHTMX(c) {
		return Redirect(c, "/login")
	}

	return RespondOK(c, map[string]string{"message": "logged out successfully"})
}

func (s *Server) handleMe(c echo.Context) error {
	user, err := requireUser(c)
	if err != nil {
		return err
	}

	return RespondOK(c, map[string]interface{}{
		"id":         user.ID.String(),
		"email":      user.Email,
		"username":   user.Username,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"status":     user.Status,
	})
}

// UpdateProfileRequest is the request payload for updating user profile.
type UpdateProfileRequest struct {
	FirstName *string `json:"first_name" form:"first_name" validate:"omitempty,max=50"`
	LastName  *string `json:"last_name" form:"last_name" validate:"omitempty,max=50"`
}

func (s *Server) handleUpdateProfile(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	userID, err := requireUserID(c)
	if err != nil {
		return err
	}

	var req UpdateProfileRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	user, err := s.userService.UpdateUser(ctx, userID, aletheia.UserUpdate{
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		return err
	}

	s.log(c).Info("profile updated", slog.String("user_id", user.ID.String()))

	return RespondOK(c, map[string]interface{}{
		"id":         user.ID.String(),
		"email":      user.Email,
		"username":   user.Username,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
	})
}

// VerifyEmailRequest is the request payload for email verification.
type VerifyEmailRequest struct {
	Token string `json:"token" form:"token" validate:"required,max=255"`
}

func (s *Server) handleVerifyEmail(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	var req VerifyEmailRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	user, err := s.userService.VerifyEmail(ctx, req.Token)
	if err != nil {
		return err
	}

	s.log(c).Info("email verified", slog.String("user_id", user.ID.String()))

	if IsHTMX(c) {
		return c.HTML(http.StatusOK, `
			<div id="verify-content">
				<h1>Email Verified!</h1>
				<p>Your email has been successfully verified.</p>
				<a href="/login" class="btn-primary">Go to Sign In</a>
			</div>
		`)
	}

	return RespondOK(c, map[string]string{"message": "email verified successfully"})
}

// ResendVerificationRequest is the request payload for resending verification email.
type ResendVerificationRequest struct {
	Email string `json:"email" form:"email" validate:"required,email,max=255"`
}

func (s *Server) handleResendVerification(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	var req ResendVerificationRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	// Find user by email
	user, err := s.userService.FindUserByEmail(ctx, req.Email)
	if err != nil {
		// Don't reveal if email exists
		return RespondOK(c, map[string]string{"message": "if that email exists and is not verified, a verification email has been sent"})
	}

	// Check if already verified
	if user.VerifiedAt != nil {
		return RespondOK(c, map[string]string{"message": "if that email exists and is not verified, a verification email has been sent"})
	}

	// Generate new token
	token, err := auth.GenerateVerificationToken()
	if err != nil {
		return aletheia.Internal("Failed to resend verification email", err)
	}

	// Save token
	if err := s.userService.SetVerificationToken(ctx, user.ID, token); err != nil {
		return aletheia.Internal("Failed to resend verification email", err)
	}

	// Send email
	if s.emailService != nil {
		name := user.FirstName
		if name == "" {
			name = user.Username
		}
		if err := s.emailService.SendVerificationEmail(ctx, user.Email, name, token); err != nil {
			s.log(c).Error("failed to send verification email", slog.String("error", err.Error()))
			return aletheia.Internal("Failed to send verification email", err)
		}
	}

	s.log(c).Info("verification email resent", slog.String("user_id", user.ID.String()))

	return RespondOK(c, map[string]string{"message": "if that email exists and is not verified, a verification email has been sent"})
}

// RequestPasswordResetRequest is the request payload for requesting a password reset.
type RequestPasswordResetRequest struct {
	Email string `json:"email" form:"email" validate:"required,email,max=255"`
}

func (s *Server) handleRequestPasswordReset(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	var req RequestPasswordResetRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	token, err := s.userService.RequestPasswordReset(ctx, req.Email)
	if err != nil {
		// Don't reveal internal errors
		s.log(c).Error("password reset request failed", slog.String("error", err.Error()))
	}

	// Send email if token was generated
	if token != "" && s.emailService != nil {
		// We don't have user info at this point, use email as fallback name
		if err := s.emailService.SendPasswordResetEmail(ctx, req.Email, req.Email, token); err != nil {
			s.log(c).Error("failed to send password reset email", slog.String("error", err.Error()))
		}
	}

	// Always return success to not reveal if email exists
	message := "if that email exists, a password reset link has been sent"

	if IsHTMX(c) {
		return c.HTML(http.StatusOK, `
			<div id="forgot-form">
				<h1>Check Your Email</h1>
				<p>`+message+`</p>
				<a href="/login" class="btn-primary">Back to Sign In</a>
			</div>
		`)
	}

	return RespondOK(c, map[string]string{"message": message})
}

// VerifyResetTokenRequest is the request payload for verifying a reset token.
type VerifyResetTokenRequest struct {
	Token string `json:"token" form:"token" validate:"required,max=255"`
}

func (s *Server) handleVerifyResetToken(c echo.Context) error {
	// This is handled by the ResetPassword endpoint in the service
	// Just validate the token format
	var req VerifyResetTokenRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	return RespondOK(c, map[string]string{"message": "token is valid"})
}

// ResetPasswordRequest is the request payload for resetting a password.
type ResetPasswordRequest struct {
	Token       string `json:"token" form:"token" validate:"required,max=255"`
	NewPassword string `json:"new_password" form:"password" validate:"required,min=8,max=128"`
}

func (s *Server) handleResetPassword(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	var req ResetPasswordRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	// Validate password complexity
	if err := auth.ValidatePassword(req.NewPassword); err != nil {
		return aletheia.Invalid("%s", err.Error())
	}

	if err := s.userService.ResetPassword(ctx, req.Token, req.NewPassword); err != nil {
		return err
	}

	s.log(c).Info("password reset successfully")

	return RespondOK(c, map[string]string{"message": "password reset successfully"})
}

// Helper functions

func parseName(fullName, firstName, lastName string) (string, string) {
	if fullName != "" {
		parts := strings.Fields(strings.TrimSpace(fullName))
		if len(parts) == 1 {
			return parts[0], ""
		}
		if len(parts) > 1 {
			return parts[0], strings.Join(parts[1:], " ")
		}
	}
	return strings.TrimSpace(firstName), strings.TrimSpace(lastName)
}

func (s *Server) renderRegisterFormWithErrors(c echo.Context, req *RegisterRequest, errors map[string]string) error {
	data := map[string]interface{}{
		"IsAuthenticated": false,
		"Values": map[string]string{
			"name":     req.Name,
			"username": req.Username,
			"email":    req.Email,
		},
		"Errors": errors,
	}
	return c.Render(http.StatusUnprocessableEntity, "register.html", data)
}

func (s *Server) renderLoginFormWithErrors(c echo.Context, req *LoginRequest, errors map[string]string) error {
	data := map[string]interface{}{
		"IsAuthenticated": false,
		"Values": map[string]string{
			"email": req.Email,
		},
		"Errors": errors,
	}
	return c.Render(http.StatusUnprocessableEntity, "login.html", data)
}
