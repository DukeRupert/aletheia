package handlers

import (
	"log/slog"
	"net/http"

	"github.com/dukerupert/aletheia/internal/auth"
	"github.com/dukerupert/aletheia/internal/database"
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
