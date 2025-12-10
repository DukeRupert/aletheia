package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/dukerupert/aletheia"
	"github.com/dukerupert/aletheia/internal/auth"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Compile-time check that UserService implements aletheia.UserService.
var _ aletheia.UserService = (*UserService)(nil)

// UserService implements aletheia.UserService using PostgreSQL.
type UserService struct {
	db *DB
}

func (s *UserService) FindUserByID(ctx context.Context, id uuid.UUID) (*aletheia.User, error) {
	user, err := s.db.queries.GetUser(ctx, toPgUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("User not found")
		}
		return nil, aletheia.Internal("Failed to fetch user", err)
	}
	return toDomainUser(user), nil
}

func (s *UserService) FindUserByEmail(ctx context.Context, email string) (*aletheia.User, error) {
	user, err := s.db.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("User not found")
		}
		return nil, aletheia.Internal("Failed to fetch user", err)
	}
	return toDomainUser(user), nil
}

func (s *UserService) FindUserByUsername(ctx context.Context, username string) (*aletheia.User, error) {
	user, err := s.db.queries.GetUserByUsername(ctx, username)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("User not found")
		}
		return nil, aletheia.Internal("Failed to fetch user", err)
	}
	return toDomainUser(user), nil
}

func (s *UserService) FindUsers(ctx context.Context, filter aletheia.UserFilter) ([]*aletheia.User, int, error) {
	// Currently sqlc only supports filtering by status
	// For more complex filtering, we would need custom queries
	status := database.UserStatusActive
	if filter.Status != nil {
		status = database.UserStatus(*filter.Status)
	}

	users, err := s.db.queries.ListUsers(ctx, status)
	if err != nil {
		return nil, 0, aletheia.Internal("Failed to list users", err)
	}

	// Apply offset/limit in memory (not ideal, but matches current sqlc queries)
	total := len(users)
	if filter.Offset > 0 && filter.Offset < len(users) {
		users = users[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(users) {
		users = users[:filter.Limit]
	}

	return toDomainUsers(users), total, nil
}

func (s *UserService) CreateUser(ctx context.Context, user *aletheia.User, password string) error {
	// Hash password
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		return aletheia.Internal("Failed to hash password", err)
	}

	dbUser, err := s.db.queries.CreateUser(ctx, database.CreateUserParams{
		Email:        user.Email,
		Username:     user.Username,
		PasswordHash: hashedPassword,
		FirstName:    toPgText(user.FirstName),
		LastName:     toPgText(user.LastName),
	})
	if err != nil {
		// Check for unique constraint violation
		if isUniqueViolation(err) {
			return aletheia.Conflict("User with this email or username already exists")
		}
		return aletheia.Internal("Failed to create user", err)
	}

	// Update user with generated values
	user.ID = fromPgUUID(dbUser.ID)
	user.Status = aletheia.UserStatus(dbUser.Status)
	user.CreatedAt = fromPgTimestamp(dbUser.CreatedAt)
	user.UpdatedAt = fromPgTimestamp(dbUser.UpdatedAt)

	return nil
}

func (s *UserService) UpdateUser(ctx context.Context, id uuid.UUID, upd aletheia.UserUpdate) (*aletheia.User, error) {
	// Build update params
	params := database.UpdateUserParams{
		ID: toPgUUID(id),
	}

	if upd.FirstName != nil {
		params.FirstName = toPgText(*upd.FirstName)
	}
	if upd.LastName != nil {
		params.LastName = toPgText(*upd.LastName)
	}

	user, err := s.db.queries.UpdateUser(ctx, params)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("User not found")
		}
		return nil, aletheia.Internal("Failed to update user", err)
	}

	// If status update is requested, use separate query
	if upd.Status != nil {
		statusReason := ""
		if upd.StatusReason != nil {
			statusReason = *upd.StatusReason
		}
		user, err = s.db.queries.UpdateUserStatus(ctx, database.UpdateUserStatusParams{
			ID:           toPgUUID(id),
			Status:       database.UserStatus(*upd.Status),
			StatusReason: toPgText(statusReason),
		})
		if err != nil {
			return nil, aletheia.Internal("Failed to update user status", err)
		}
	}

	return toDomainUser(user), nil
}

func (s *UserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	// Soft delete by setting status to deleted
	_, err := s.db.queries.UpdateUserStatus(ctx, database.UpdateUserStatusParams{
		ID:           toPgUUID(id),
		Status:       database.UserStatusDeleted,
		StatusReason: toPgText("Account deleted"),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return aletheia.NotFound("User not found")
		}
		return aletheia.Internal("Failed to delete user", err)
	}
	return nil
}

func (s *UserService) VerifyPassword(ctx context.Context, email, password string) (*aletheia.User, error) {
	user, err := s.db.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.Unauthorized("Invalid email or password")
		}
		return nil, aletheia.Internal("Failed to fetch user", err)
	}

	// Check password
	if err := auth.VerifyPassword(password, user.PasswordHash); err != nil {
		return nil, aletheia.Unauthorized("Invalid email or password")
	}

	// Check account status
	if user.Status != database.UserStatusActive {
		return nil, aletheia.Forbidden("Account is not active")
	}

	// Check if email is verified
	if !user.VerifiedAt.Valid {
		return nil, aletheia.Forbidden("Email not verified")
	}

	return toDomainUser(user), nil
}

func (s *UserService) SetVerificationToken(ctx context.Context, id uuid.UUID, token string) error {
	err := s.db.queries.SetVerificationToken(ctx, database.SetVerificationTokenParams{
		ID:                toPgUUID(id),
		VerificationToken: toPgText(token),
	})
	if err != nil {
		return aletheia.Internal("Failed to set verification token", err)
	}
	return nil
}

func (s *UserService) VerifyEmail(ctx context.Context, token string) (*aletheia.User, error) {
	// Find user by verification token
	user, err := s.db.queries.GetUserByVerificationToken(ctx, toPgText(token))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.Invalid("Invalid or expired verification token")
		}
		return nil, aletheia.Internal("Failed to verify email", err)
	}

	// Mark email as verified
	user, err = s.db.queries.VerifyUserEmail(ctx, user.ID)
	if err != nil {
		return nil, aletheia.Internal("Failed to verify email", err)
	}

	return toDomainUser(user), nil
}

func (s *UserService) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	// Find user by email
	user, err := s.db.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Don't reveal that email doesn't exist
			return "", nil
		}
		return "", aletheia.Internal("Failed to request password reset", err)
	}

	// Generate reset token
	token, err := generateSecureToken(32)
	if err != nil {
		return "", aletheia.Internal("Failed to generate reset token", err)
	}

	// Set token with 1 hour expiration
	expiresAt := time.Now().Add(1 * time.Hour)
	err = s.db.queries.SetPasswordResetToken(ctx, database.SetPasswordResetTokenParams{
		ID:                  user.ID,
		ResetToken:          toPgText(token),
		ResetTokenExpiresAt: toPgTimestamp(expiresAt),
	})
	if err != nil {
		return "", aletheia.Internal("Failed to set reset token", err)
	}

	return token, nil
}

func (s *UserService) ResetPassword(ctx context.Context, token, newPassword string) error {
	// Find user by reset token
	user, err := s.db.queries.GetUserByResetToken(ctx, toPgText(token))
	if err != nil {
		if err == pgx.ErrNoRows {
			return aletheia.Invalid("Invalid or expired reset token")
		}
		return aletheia.Internal("Failed to reset password", err)
	}

	// Hash new password
	hashedPassword, err := auth.HashPassword(newPassword)
	if err != nil {
		return aletheia.Internal("Failed to hash password", err)
	}

	// Update password and clear reset token
	_, err = s.db.queries.ResetUserPassword(ctx, database.ResetUserPasswordParams{
		ID:           user.ID,
		PasswordHash: hashedPassword,
	})
	if err != nil {
		return aletheia.Internal("Failed to reset password", err)
	}

	return nil
}

func (s *UserService) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	err := s.db.queries.UpdateUserLastLogin(ctx, toPgUUID(id))
	if err != nil {
		return aletheia.Internal("Failed to update last login", err)
	}
	return nil
}

// generateSecureToken generates a cryptographically secure random token.
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
