package handlers

import (
	"log/slog"
	"net/http"

	"github.com/dukerupert/aletheia/internal/auth"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/email"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	db           *pgxpool.Pool
	logger       *slog.Logger
	emailService email.EmailService
}

func NewAuthHandler(db *pgxpool.Pool, logger *slog.Logger, emailService email.EmailService) *AuthHandler {
	return &AuthHandler{
		db:           db,
		logger:       logger,
		emailService: emailService,
	}
}

type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Username  string `json:"username" validate:"required,min=3,max=50"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
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

	if len(req.Password) < 8 {
		return echo.NewHTTPError(http.StatusBadRequest, "password must be at least 8 characters")
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		h.logger.Error("failed to hash password", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to process password")
	}

	// Create user in database
	queries := database.New(h.db)

	// Convert optional fields to pgtype.Text
	var firstName, lastName pgtype.Text
	if req.FirstName != "" {
		firstName = pgtype.Text{String: req.FirstName, Valid: true}
	}
	if req.LastName != "" {
		lastName = pgtype.Text{String: req.LastName, Valid: true}
	}

	user, err := queries.CreateUser(c.Request().Context(), database.CreateUserParams{
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: passwordHash,
		FirstName:    firstName,
		LastName:     lastName,
	})

	if err != nil {
		// Check for unique constraint violation
		if err.Error() == "ERROR: duplicate key value violates unique constraint \"users_email_key\" (SQLSTATE 23505)" ||
			err.Error() == "ERROR: duplicate key value violates unique constraint \"users_username_key\" (SQLSTATE 23505)" {
			h.logger.Warn("registration attempt with existing email/username",
				slog.String("email", req.Email),
				slog.String("username", req.Username),
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
	if err := queries.SetVerificationToken(c.Request().Context(), database.SetVerificationTokenParams{
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

	// Return user info (without password hash)
	return c.JSON(http.StatusCreated, RegisterResponse{
		ID:       user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
	})
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
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

	// Get user by email
	queries := database.New(h.db)
	user, err := queries.GetUserByEmail(c.Request().Context(), req.Email)
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

	// Check if user is active
	if user.Status != "active" {
		h.logger.Warn("login attempt for non-active user",
			slog.String("user_id", user.ID.String()),
			slog.String("status", string(user.Status)),
		)
		return echo.NewHTTPError(http.StatusForbidden, "account is not active")
	}

	// Create session (convert pgtype.UUID to uuid.UUID)
	sess, err := session.CreateSession(c.Request().Context(), h.db, user.ID.Bytes, session.SessionDuration)
	if err != nil {
		h.logger.Error("failed to create session", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "login failed")
	}

	// Update last login time
	if err := queries.UpdateUserLastLogin(c.Request().Context(), user.ID); err != nil {
		h.logger.Warn("failed to update last login time", slog.String("err", err.Error()))
		// Don't fail the login for this
	}

	// Set session cookie
	cookie := &http.Cookie{
		Name:     session.SessionCookieName,
		Value:    sess.Token,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request().URL.Scheme == "https", // Only send over HTTPS in production
		SameSite: http.SameSiteLaxMode,
		Expires:  sess.ExpiresAt.Time, // Convert pgtype.Timestamptz to time.Time
	}
	c.SetCookie(cookie)

	h.logger.Info("user logged in successfully",
		slog.String("user_id", user.ID.String()),
		slog.String("email", user.Email),
	)

	// Return user info
	return c.JSON(http.StatusOK, LoginResponse{
		ID:       user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c echo.Context) error {
	// Get session token from cookie
	cookie, err := c.Cookie(session.SessionCookieName)
	if err != nil {
		// No session cookie found
		return echo.NewHTTPError(http.StatusBadRequest, "not logged in")
	}

	// Delete session from database
	if err := session.DestroySession(c.Request().Context(), h.db, cookie.Value); err != nil {
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

	h.logger.Info("user logged out successfully")

	return c.JSON(http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}

// Me returns the current authenticated user's information
func (h *AuthHandler) Me(c echo.Context) error {
	// Get user ID from context (set by session middleware)
	userID, ok := session.GetUserID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
	}

	// Get user from database
	queries := database.New(h.db)
	user, err := queries.GetUser(c.Request().Context(), pgtype.UUID{
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
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
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
	user, err := queries.UpdateUser(c.Request().Context(), database.UpdateUserParams{
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
	Token string `json:"token" validate:"required"`
}

// VerifyEmail verifies a user's email address using the verification token
func (h *AuthHandler) VerifyEmail(c echo.Context) error {
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
	user, err := queries.GetUserByVerificationToken(c.Request().Context(), pgtype.Text{
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
	verifiedUser, err := queries.VerifyUserEmail(c.Request().Context(), user.ID)
	if err != nil {
		h.logger.Error("failed to verify user email", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "verification failed")
	}

	h.logger.Info("user email verified successfully",
		slog.String("user_id", verifiedUser.ID.String()),
		slog.String("email", verifiedUser.Email),
	)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "email verified successfully",
	})
}

type ResendVerificationRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResendVerification resends the verification email to a user
func (h *AuthHandler) ResendVerification(c echo.Context) error {
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
	user, err := queries.GetUserByEmail(c.Request().Context(), req.Email)
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
	if err := queries.SetVerificationToken(c.Request().Context(), database.SetVerificationTokenParams{
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
