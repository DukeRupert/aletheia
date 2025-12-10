package aletheia

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system.
type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	Username     string     `json:"username"`
	FirstName    string     `json:"firstName,omitempty"`
	LastName     string     `json:"lastName,omitempty"`
	Status       UserStatus `json:"status"`
	StatusReason string     `json:"statusReason,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	LastLoginAt  *time.Time `json:"lastLoginAt,omitempty"`
	VerifiedAt   *time.Time `json:"verifiedAt,omitempty"`
}

// UserStatus represents the status of a user account.
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusDeleted   UserStatus = "deleted"
)

// FullName returns the user's full name.
func (u *User) FullName() string {
	if u.FirstName == "" && u.LastName == "" {
		return u.Username
	}
	if u.LastName == "" {
		return u.FirstName
	}
	if u.FirstName == "" {
		return u.LastName
	}
	return u.FirstName + " " + u.LastName
}

// IsActive returns true if the user account is active.
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// IsVerified returns true if the user's email has been verified.
func (u *User) IsVerified() bool {
	return u.VerifiedAt != nil
}

// UserService defines operations for managing users.
type UserService interface {
	// FindUserByID retrieves a user by their ID.
	// Returns ENOTFOUND if the user does not exist.
	FindUserByID(ctx context.Context, id uuid.UUID) (*User, error)

	// FindUserByEmail retrieves a user by their email address.
	// Returns ENOTFOUND if the user does not exist.
	FindUserByEmail(ctx context.Context, email string) (*User, error)

	// FindUserByUsername retrieves a user by their username.
	// Returns ENOTFOUND if the user does not exist.
	FindUserByUsername(ctx context.Context, username string) (*User, error)

	// FindUsers retrieves users matching the filter criteria.
	// Returns the matching users and total count (which may differ if Limit is set).
	FindUsers(ctx context.Context, filter UserFilter) ([]*User, int, error)

	// CreateUser creates a new user with the given password.
	// Returns ECONFLICT if email or username already exists.
	CreateUser(ctx context.Context, user *User, password string) error

	// UpdateUser updates an existing user.
	// Returns ENOTFOUND if the user does not exist.
	UpdateUser(ctx context.Context, id uuid.UUID, upd UserUpdate) (*User, error)

	// DeleteUser soft-deletes a user by setting status to deleted.
	// Returns ENOTFOUND if the user does not exist.
	DeleteUser(ctx context.Context, id uuid.UUID) error

	// Authentication methods

	// VerifyPassword verifies a user's password and returns the user if valid.
	// Returns EUNAUTHORIZED if credentials are invalid.
	// Returns EFORBIDDEN if the account is not active or not verified.
	VerifyPassword(ctx context.Context, email, password string) (*User, error)

	// SetVerificationToken sets the email verification token for a user.
	SetVerificationToken(ctx context.Context, id uuid.UUID, token string) error

	// VerifyEmail verifies a user's email using the provided token.
	// Returns EINVALID if the token is invalid or expired.
	VerifyEmail(ctx context.Context, token string) (*User, error)

	// RequestPasswordReset generates a password reset token.
	// Returns the token if the email exists, or nil error if it doesn't (to prevent enumeration).
	RequestPasswordReset(ctx context.Context, email string) (token string, err error)

	// ResetPassword resets a user's password using the provided token.
	// Returns EINVALID if the token is invalid or expired.
	ResetPassword(ctx context.Context, token, newPassword string) error

	// UpdateLastLogin updates the user's last login timestamp.
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

// UserFilter defines criteria for filtering users.
type UserFilter struct {
	ID       *uuid.UUID
	Email    *string
	Username *string
	Status   *UserStatus

	// Pagination
	Offset int
	Limit  int
}

// UserUpdate defines fields that can be updated on a user.
// Pointer fields: nil = don't update, non-nil = update to this value.
type UserUpdate struct {
	FirstName    *string
	LastName     *string
	Status       *UserStatus
	StatusReason *string
}
