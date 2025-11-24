package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/dukerupert/aletheia/internal/auth"
	"github.com/dukerupert/aletheia/internal/config"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/email"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)


type AuthHandler struct {
	db           *pgxpool.Pool
	logger       *slog.Logger
	emailService email.EmailService
	cfg          *config.Config
}

func NewAuthHandler(db *pgxpool.Pool, logger *slog.Logger, emailService email.EmailService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		db:           db,
		logger:       logger,
		emailService: emailService,
		cfg:          cfg,
	}
}

type RegisterRequest struct {
	Email     string `json:"email" form:"email" validate:"required,email,max=255"`
	Username  string `json:"username" form:"username" validate:"required,min=3,max=50"`
	Password  string `json:"password" form:"password" validate:"required,min=8,max=128"`
	Name      string `json:"name" form:"name" validate:"omitempty,max=100"` // Full name from form
	FirstName string `json:"first_name" form:"first_name" validate:"omitempty,max=50"`
	LastName  string `json:"last_name" form:"last_name" validate:"omitempty,max=50"`
}

type RegisterResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// Register handles user registration
func (h *AuthHandler) Register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("failed to bind request", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Validate request
	if req.Email == "" || req.Username == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email, username, and password are required")
	}

	// Validate password complexity
	if err := auth.ValidatePassword(req.Password); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		h.logger.Error("failed to hash password", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to process password")
	}

	// Create user in database
	queries := database.New(h.db)

	// Handle name field - split full name if provided, otherwise use first/last names
	var firstName, lastName pgtype.Text
	if req.Name != "" {
		// Validate and sanitize name
		name := strings.TrimSpace(req.Name)
		if len(name) > 100 {
			return echo.NewHTTPError(http.StatusBadRequest, "name too long (max 100 characters)")
		}

		// Split on spaces, handling multiple words better
		parts := strings.Fields(name) // Handles multiple spaces, trims whitespace
		if len(parts) > 0 {
			if len(parts) == 1 {
				// Single name - use as first name
				if len(parts[0]) > 50 {
					return echo.NewHTTPError(http.StatusBadRequest, "name too long")
				}
				firstName = pgtype.Text{String: parts[0], Valid: true}
			} else {
				// Multiple words - first word is first name, rest is last name
				if len(parts[0]) > 50 {
					return echo.NewHTTPError(http.StatusBadRequest, "first name too long (max 50 characters)")
				}
				lastNameStr := strings.Join(parts[1:], " ")
				if len(lastNameStr) > 50 {
					return echo.NewHTTPError(http.StatusBadRequest, "last name too long (max 50 characters)")
				}
				firstName = pgtype.Text{String: parts[0], Valid: true}
				lastName = pgtype.Text{String: lastNameStr, Valid: true}
			}
		}
	} else {
		// Use explicit first/last name fields with validation
		if req.FirstName != "" {
			name := strings.TrimSpace(req.FirstName)
			if len(name) > 50 {
				return echo.NewHTTPError(http.StatusBadRequest, "first name too long (max 50 characters)")
			}
			firstName = pgtype.Text{String: name, Valid: true}
		}
		if req.LastName != "" {
			name := strings.TrimSpace(req.LastName)
			if len(name) > 50 {
				return echo.NewHTTPError(http.StatusBadRequest, "last name too long (max 50 characters)")
			}
			lastName = pgtype.Text{String: name, Valid: true}
		}
	}

	// Create user with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	user, err := queries.CreateUser(ctx, database.CreateUserParams{
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: passwordHash,
		FirstName:    firstName,
		LastName:     lastName,
	})

	if err != nil {
		// Check for unique constraint violation using pgx error codes
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// 23505 = unique_violation
			h.logger.Warn("registration attempt with existing email/username",
				slog.String("email", req.Email),
				slog.String("username", req.Username),
				slog.String("constraint", pgErr.ConstraintName),
			)
			return echo.NewHTTPError(http.StatusConflict, "email or username already exists")
		}

		h.logger.Error("failed to create user", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
	}

	// Generate verification token
	verificationToken, err := auth.GenerateVerificationToken()
	if err != nil {
		h.logger.Error("failed to generate verification token", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
	}

	// Save verification token to database
	if err := queries.SetVerificationToken(ctx, database.SetVerificationTokenParams{
		ID:                user.ID,
		VerificationToken: pgtype.Text{String: verificationToken, Valid: true},
	}); err != nil {
		h.logger.Error("failed to set verification token", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
	}

	// Send verification email
	if err := h.emailService.SendVerificationEmail(user.Email, verificationToken); err != nil {
		h.logger.Error("failed to send verification email",
			slog.String("user_id", user.ID.String()),
			slog.String("err", err.Error()),
		)
		// Don't fail registration if email fails - user can request resend
	}

	h.logger.Info("user registered successfully",
		slog.String("user_id", user.ID.String()),
		slog.String("email", user.Email),
		slog.String("username", user.Username),
	)

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		// HTMX request - redirect to login page
		c.Response().Header().Set("HX-Redirect", "/login")
		return c.NoContent(http.StatusOK)
	}

	// Regular API request - return JSON
	return c.JSON(http.StatusCreated, RegisterResponse{
		ID:       user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
	})
}

type LoginRequest struct {
	Email    string `json:"email" form:"email" validate:"required,email,max=255"`
	Password string `json:"password" form:"password" validate:"required,max=128"`
}

type LoginResponse struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// Login handles user login
func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("failed to bind request", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Validate request
	if req.Email == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email and password are required")
	}

	// Get user by email with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	queries := database.New(h.db)
	user, err := queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err == pgx.ErrNoRows {
			h.logger.Warn("login attempt with non-existent email", slog.String("email", req.Email))
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid email or password")
		}
		h.logger.Error("failed to get user", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "login failed")
	}

	// Verify password
	if err := auth.VerifyPassword(req.Password, user.PasswordHash); err != nil {
		h.logger.Warn("login attempt with invalid password",
			slog.String("user_id", user.ID.String()),
			slog.String("email", user.Email),
		)
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid email or password")
	}

	// Check if email is verified
	if !user.VerifiedAt.Valid {
		h.logger.Warn("login attempt with unverified email",
			slog.String("user_id", user.ID.String()),
			slog.String("email", user.Email),
		)
		return echo.NewHTTPError(http.StatusForbidden, "please verify your email address before logging in")
	}

	// Check if user is active
	if user.Status != database.UserStatusActive {
		h.logger.Warn("login attempt for non-active user",
			slog.String("user_id", user.ID.String()),
			slog.String("status", string(user.Status)),
		)
		return echo.NewHTTPError(http.StatusForbidden, "account is not active")
	}

	// Create session (convert pgtype.UUID to uuid.UUID)
	sess, err := session.CreateSession(ctx, h.db, user.ID.Bytes, session.SessionDuration)
	if err != nil {
		h.logger.Error("failed to create session", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "login failed")
	}

	// Update last login time
	if err := queries.UpdateUserLastLogin(ctx, user.ID); err != nil {
		h.logger.Warn("failed to update last login time", slog.String("err", err.Error()))
		// Don't fail the login for this
	}

	// Set session cookie
	cookie := &http.Cookie{
		Name:     session.SessionCookieName,
		Value:    sess.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.Session.Secure, // Use config-based Secure flag (set by ENVIRONMENT variable)
		SameSite: http.SameSiteLaxMode,
		Expires:  sess.ExpiresAt.Time, // Convert pgtype.Timestamptz to time.Time
	}
	c.SetCookie(cookie)

	h.logger.Info("user logged in successfully",
		slog.String("user_id", user.ID.String()),
		slog.String("email", user.Email),
	)

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		// HTMX request - redirect to dashboard
		c.Response().Header().Set("HX-Redirect", "/dashboard")
		return c.NoContent(http.StatusOK)
	}

	// Regular API request - return JSON
	return c.JSON(http.StatusOK, LoginResponse{
		ID:       user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Try to get user ID from session context for logging
	userID, hasUserID := session.GetUserID(c)

	// Get session token from cookie
	cookie, err := c.Cookie(session.SessionCookieName)
	if err != nil {
		// No session cookie found - redirect to login anyway
		if c.Request().Header.Get("HX-Request") == "true" {
			c.Response().Header().Set("HX-Redirect", "/login")
			return c.NoContent(http.StatusOK)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "not logged in")
	}

	// Delete session from database
	if err := session.DestroySession(ctx, h.db, cookie.Value); err != nil{
		h.logger.Error("failed to destroy session", slog.String("err", err.Error()))
		// Continue to clear cookie even if database delete fails
	}

	// Clear session cookie
	cookie = &http.Cookie{
		Name:     session.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1, // Delete cookie
	}
	c.SetCookie(cookie)

	// Log logout with user ID if available
	if hasUserID {
		h.logger.Info("user logged out successfully",
			slog.String("user_id", userID.String()),
		)
	} else {
		h.logger.Info("user logged out successfully")
	}

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		// HTMX request - redirect to login page
		c.Response().Header().Set("HX-Redirect", "/login")
		return c.NoContent(http.StatusOK)
	}

	// Regular API request - return JSON
	return c.JSON(http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}

// Me returns the current authenticated user's information
func (h *AuthHandler) Me(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get user ID from context (set by session middleware)
	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}

	// Get user from database
	queries := database.New(h.db)
	user, err := queries.GetUser(ctx, pgtype.UUID{
		Bytes: userID,
		Valid: true,
	})
	if err != nil {
		h.logger.Error("failed to get user", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get user")
	}

	// Return user info
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":         user.ID.String(),
		"email":      user.Email,
		"username":   user.Username,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"status":     user.Status,
	})
}

type UpdateProfileRequest struct {
	FirstName *string `json:"first_name" form:"first_name" validate:"omitempty,max=50"`
	LastName  *string `json:"last_name" form:"last_name" validate:"omitempty,max=50"`
}

type UpdateProfileResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// UpdateProfile updates the current user's profile information
func (h *AuthHandler) UpdateProfile(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	// Get user ID from context (set by session middleware)
	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}

	var req UpdateProfileRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("failed to bind request", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Build update parameters
	queries := database.New(h.db)

	// Convert to pgtype.Text for nullable fields
	var firstName, lastName pgtype.Text
	if req.FirstName != nil {
		firstName = pgtype.Text{String: *req.FirstName, Valid: true}
	}
	if req.LastName != nil {
		lastName = pgtype.Text{String: *req.LastName, Valid: true}
	}

	// Update user
	user, err := queries.UpdateUser(ctx, database.UpdateUserParams{
		ID:        pgtype.UUID{Bytes: userID, Valid: true},
		FirstName: firstName,
		LastName:  lastName,
	})
	if err != nil {
		h.logger.Error("failed to update user", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update profile")
	}

	h.logger.Info("user profile updated",
		slog.String("user_id", user.ID.String()),
		slog.String("email", user.Email),
	)

	// Return updated user info
	resp := UpdateProfileResponse{
		ID:       user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
	}

	if user.FirstName.Valid {
		resp.FirstName = user.FirstName.String
	}
	if user.LastName.Valid {
		resp.LastName = user.LastName.String
	}

	return c.JSON(http.StatusOK, resp)
}

type VerifyEmailRequest struct {
	Token string `json:"token" form:"token" validate:"required,max=255"`
}

// VerifyEmail verifies a user's email address using the verification token
func (h *AuthHandler) VerifyEmail(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	var req VerifyEmailRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("failed to bind request", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Token == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "verification token is required")
	}

	queries := database.New(h.db)

	// Find user by verification token
	user, err := queries.GetUserByVerificationToken(ctx, pgtype.Text{
		String: req.Token,
		Valid:  true,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			h.logger.Warn("verification attempt with invalid token", slog.String("token", req.Token))
			return echo.NewHTTPError(http.StatusBadRequest, "invalid or expired verification token")
		}
		h.logger.Error("failed to get user by verification token", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "verification failed")
	}

	// Verify the user's email
	verifiedUser, err := queries.VerifyUserEmail(ctx, user.ID)
	if err != nil {
		h.logger.Error("failed to verify user email", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "verification failed")
	}

	h.logger.Info("user email verified successfully",
		slog.String("user_id", verifiedUser.ID.String()),
		slog.String("email", verifiedUser.Email),
	)

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		// HTMX request - return success HTML fragment
		successHTML := `
			<div id="verify-content">
				<h1>Email Verified!</h1>
				<p>Your email has been successfully verified.</p>
				<div class="alert alert-success">
					<strong>Success:</strong> You can now sign in to your account.
				</div>
				<a href="/login" class="btn-primary">Go to Sign In</a>
			</div>
		`
		return c.HTML(http.StatusOK, successHTML)
	}

	// Regular API request - return JSON
	return c.JSON(http.StatusOK, map[string]string{
		"message": "email verified successfully",
	})
}

type ResendVerificationRequest struct {
	Email string `json:"email" form:"email" validate:"required,email,max=255"`
}

// ResendVerification resends the verification email to a user
func (h *AuthHandler) ResendVerification(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	var req ResendVerificationRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("failed to bind request", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Email == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email is required")
	}

	queries := database.New(h.db)

	// Get user by email
	user, err := queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Don't reveal if email exists or not for security
			h.logger.Warn("resend verification attempt for non-existent email", slog.String("email", req.Email))
			return c.JSON(http.StatusOK, map[string]string{
				"message": "if that email exists and is not verified, a verification email has been sent",
			})
		}
		h.logger.Error("failed to get user", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to resend verification email")
	}

	// Check if user is already verified
	if user.VerifiedAt.Valid {
		h.logger.Info("resend verification attempt for already verified user",
			slog.String("user_id", user.ID.String()),
			slog.String("email", user.Email),
		)
		return c.JSON(http.StatusOK, map[string]string{
			"message": "if that email exists and is not verified, a verification email has been sent",
		})
	}

	// Generate new verification token
	verificationToken, err := auth.GenerateVerificationToken()
	if err != nil {
		h.logger.Error("failed to generate verification token", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to resend verification email")
	}

	// Save verification token to database
	if err := queries.SetVerificationToken(ctx, database.SetVerificationTokenParams{
		ID:                user.ID,
		VerificationToken: pgtype.Text{String: verificationToken, Valid: true},
	}); err != nil {
		h.logger.Error("failed to set verification token", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to resend verification email")
	}

	// Send verification email
	if err := h.emailService.SendVerificationEmail(user.Email, verificationToken); err != nil {
		h.logger.Error("failed to send verification email",
			slog.String("user_id", user.ID.String()),
			slog.String("err", err.Error()),
		)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to send verification email")
	}

	h.logger.Info("verification email resent",
		slog.String("user_id", user.ID.String()),
		slog.String("email", user.Email),
	)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "if that email exists and is not verified, a verification email has been sent",
	})
}

type RequestPasswordResetRequest struct {
	Email string `json:"email" form:"email" validate:"required,email,max=255"`
}

// RequestPasswordReset initiates the password reset flow
func (h *AuthHandler) RequestPasswordReset(c echo.Context) error {
	var req RequestPasswordResetRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("failed to bind request", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Email == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email is required")
	}

	queries := database.New(h.db)

	// Get user by email with timeout
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	user, err := queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Don't reveal if email exists or not for security
			h.logger.Warn("password reset attempt for non-existent email", slog.String("email", req.Email))

			// Check if this is an HTMX request
			if c.Request().Header.Get("HX-Request") == "true" {
				// HTMX request - return success HTML fragment (same message for security)
				successHTML := `
					<div id="forgot-form">
						<h1>Check Your Email</h1>
						<p>If that email exists in our system, we've sent a password reset link.</p>
						<div class="alert alert-success">
							<strong>Success:</strong> Check your email for the reset link.
						</div>
						<a href="/login" class="btn-primary">Back to Sign In</a>
					</div>
				`
				return c.HTML(http.StatusOK, successHTML)
			}

			return c.JSON(http.StatusOK, map[string]string{
				"message": "if that email exists, a password reset link has been sent",
			})
		}
		h.logger.Error("failed to get user", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to request password reset")
	}

	// Generate reset token
	resetToken, err := auth.GenerateVerificationToken()
	if err != nil {
		h.logger.Error("failed to generate reset token", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to request password reset")
	}

	// Set token expiration to 1 hour from now
	expiresAt := pgtype.Timestamptz{
		Time:  time.Now().Add(1 * time.Hour),
		Valid: true,
	}

	// Save reset token to database
	if err := queries.SetPasswordResetToken(ctx, database.SetPasswordResetTokenParams{
		ID:                    user.ID,
		ResetToken:            pgtype.Text{String: resetToken, Valid: true},
		ResetTokenExpiresAt:   expiresAt,
	}); err != nil {
		h.logger.Error("failed to set reset token", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to request password reset")
	}

	// Send password reset email
	if err := h.emailService.SendPasswordResetEmail(user.Email, resetToken); err != nil {
		h.logger.Error("failed to send password reset email",
			slog.String("user_id", user.ID.String()),
			slog.String("err", err.Error()),
		)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to send password reset email")
	}

	h.logger.Info("password reset email sent",
		slog.String("user_id", user.ID.String()),
		slog.String("email", user.Email),
	)

	// Check if this is an HTMX request
	if c.Request().Header.Get("HX-Request") == "true" {
		// HTMX request - return success HTML fragment
		successHTML := `
			<div id="forgot-form">
				<h1>Check Your Email</h1>
				<p>If that email exists in our system, we've sent a password reset link.</p>
				<div class="alert alert-success">
					<strong>Success:</strong> Check your email for the reset link.
				</div>
				<a href="/login" class="btn-primary">Back to Sign In</a>
			</div>
		`
		return c.HTML(http.StatusOK, successHTML)
	}

	// Regular API request - return JSON
	return c.JSON(http.StatusOK, map[string]string{
		"message": "if that email exists, a password reset link has been sent",
	})
}

type VerifyResetTokenRequest struct {
	Token string `json:"token" form:"token" validate:"required,max=255"`
}

// VerifyResetToken verifies that a password reset token is valid
func (h *AuthHandler) VerifyResetToken(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	var req VerifyResetTokenRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("failed to bind request", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Token == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "reset token is required")
	}

	queries := database.New(h.db)

	// Find user by reset token
	user, err := queries.GetUserByResetToken(ctx, pgtype.Text{
		String: req.Token,
		Valid:  true,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			h.logger.Warn("reset token verification failed", slog.String("token", req.Token))
			return echo.NewHTTPError(http.StatusBadRequest, "invalid or expired reset token")
		}
		h.logger.Error("failed to get user by reset token", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "token verification failed")
	}

	// Validate token hasn't expired
	if !user.ResetTokenExpiresAt.Valid || time.Now().After(user.ResetTokenExpiresAt.Time) {
		h.logger.Warn("reset token expired",
			slog.String("user_id", user.ID.String()),
			slog.String("email", user.Email),
			slog.String("token", req.Token),
		)
		return echo.NewHTTPError(http.StatusBadRequest, "reset token has expired, please request a new one")
	}

	h.logger.Info("reset token verified",
		slog.String("user_id", user.ID.String()),
		slog.String("email", user.Email),
	)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "reset token is valid",
	})
}

type ResetPasswordRequest struct {
	Token       string `json:"token" form:"token" validate:"required,max=255"`
	NewPassword string `json:"new_password" form:"password" validate:"required,min=8,max=128"`
	Password    string `json:"password" form:"confirm_password" validate:"omitempty,max=128"` // For form compatibility
}

// ResetPassword resets a user's password using a valid reset token
func (h *AuthHandler) ResetPassword(c echo.Context) error {
	// Create context with timeout for database operations
	ctx, cancel := context.WithTimeout(c.Request().Context(), DatabaseTimeout)
	defer cancel()

	var req ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("failed to bind request", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Token == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "reset token is required")
	}

	// Validate password complexity
	if err := auth.ValidatePassword(req.NewPassword); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	queries := database.New(h.db)

	// Find user by reset token
	user, err := queries.GetUserByResetToken(ctx, pgtype.Text{
		String: req.Token,
		Valid:  true,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			h.logger.Warn("password reset attempt with invalid token", slog.String("token", req.Token))
			return echo.NewHTTPError(http.StatusBadRequest, "invalid or expired reset token")
		}
		h.logger.Error("failed to get user by reset token", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "password reset failed")
	}

	// Validate token hasn't expired
	if !user.ResetTokenExpiresAt.Valid || time.Now().After(user.ResetTokenExpiresAt.Time) {
		h.logger.Warn("password reset attempt with expired token",
			slog.String("user_id", user.ID.String()),
			slog.String("email", user.Email),
			slog.String("token", req.Token),
		)
		return echo.NewHTTPError(http.StatusBadRequest, "reset token has expired, please request a new one")
	}

	// Hash new password
	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		h.logger.Error("failed to hash password", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to process password")
	}

	// Reset password and clear reset token
	updatedUser, err := queries.ResetUserPassword(ctx, database.ResetUserPasswordParams{
		ID:           user.ID,
		PasswordHash: passwordHash,
	})
	if err != nil {
		h.logger.Error("failed to reset password", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "password reset failed")
	}

	// Invalidate all existing sessions for security
	if err := queries.DeleteUserSessions(ctx, user.ID); err != nil{
		// Log but don't fail - password was already reset
		h.logger.Warn("failed to invalidate sessions after password reset",
			slog.String("user_id", user.ID.String()),
			slog.String("err", err.Error()),
		)
	}

	h.logger.Info("password reset successfully",
		slog.String("user_id", updatedUser.ID.String()),
		slog.String("email", updatedUser.Email),
	)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "password reset successfully",
	})
}
