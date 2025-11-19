package handlers

import (
	"log/slog"
	"net/http"

	"github.com/dukerupert/aletheia/internal/auth"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func NewAuthHandler(db *pgxpool.Pool, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		db:     db,
		logger: logger,
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
