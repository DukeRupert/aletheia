package mock

import (
	"context"
	"time"

	"github.com/dukerupert/aletheia"
	"github.com/google/uuid"
)

// Compile-time interface check
var _ aletheia.UserService = (*UserService)(nil)

// UserService is a mock implementation of aletheia.UserService.
type UserService struct {
	FindUserByIDFn         func(ctx context.Context, id uuid.UUID) (*aletheia.User, error)
	FindUserByEmailFn      func(ctx context.Context, email string) (*aletheia.User, error)
	FindUserByUsernameFn   func(ctx context.Context, username string) (*aletheia.User, error)
	FindUsersFn            func(ctx context.Context, filter aletheia.UserFilter) ([]*aletheia.User, int, error)
	CreateUserFn           func(ctx context.Context, user *aletheia.User, password string) error
	UpdateUserFn           func(ctx context.Context, id uuid.UUID, upd aletheia.UserUpdate) (*aletheia.User, error)
	DeleteUserFn           func(ctx context.Context, id uuid.UUID) error
	UpdateLastLoginFn      func(ctx context.Context, id uuid.UUID) error
	VerifyPasswordFn       func(ctx context.Context, email, password string) (*aletheia.User, error)
	SetVerificationTokenFn func(ctx context.Context, id uuid.UUID, token string) error
	VerifyEmailFn          func(ctx context.Context, token string) (*aletheia.User, error)
	RequestPasswordResetFn func(ctx context.Context, email string) (string, error)
	ResetPasswordFn        func(ctx context.Context, token, newPassword string) error
}

func (s *UserService) FindUserByID(ctx context.Context, id uuid.UUID) (*aletheia.User, error) {
	if s.FindUserByIDFn != nil {
		return s.FindUserByIDFn(ctx, id)
	}
	return nil, aletheia.NotFound("User not found")
}

func (s *UserService) FindUserByEmail(ctx context.Context, email string) (*aletheia.User, error) {
	if s.FindUserByEmailFn != nil {
		return s.FindUserByEmailFn(ctx, email)
	}
	return nil, aletheia.NotFound("User not found")
}

func (s *UserService) FindUserByUsername(ctx context.Context, username string) (*aletheia.User, error) {
	if s.FindUserByUsernameFn != nil {
		return s.FindUserByUsernameFn(ctx, username)
	}
	return nil, aletheia.NotFound("User not found")
}

func (s *UserService) FindUsers(ctx context.Context, filter aletheia.UserFilter) ([]*aletheia.User, int, error) {
	if s.FindUsersFn != nil {
		return s.FindUsersFn(ctx, filter)
	}
	return []*aletheia.User{}, 0, nil
}

func (s *UserService) CreateUser(ctx context.Context, user *aletheia.User, password string) error {
	if s.CreateUserFn != nil {
		return s.CreateUserFn(ctx, user, password)
	}
	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	return nil
}

func (s *UserService) UpdateUser(ctx context.Context, id uuid.UUID, upd aletheia.UserUpdate) (*aletheia.User, error) {
	if s.UpdateUserFn != nil {
		return s.UpdateUserFn(ctx, id, upd)
	}
	return nil, aletheia.NotFound("User not found")
}

func (s *UserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if s.DeleteUserFn != nil {
		return s.DeleteUserFn(ctx, id)
	}
	return nil
}

func (s *UserService) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	if s.UpdateLastLoginFn != nil {
		return s.UpdateLastLoginFn(ctx, id)
	}
	return nil
}

func (s *UserService) VerifyPassword(ctx context.Context, email, password string) (*aletheia.User, error) {
	if s.VerifyPasswordFn != nil {
		return s.VerifyPasswordFn(ctx, email, password)
	}
	return nil, aletheia.Unauthorized("Invalid credentials")
}

func (s *UserService) SetVerificationToken(ctx context.Context, id uuid.UUID, token string) error {
	if s.SetVerificationTokenFn != nil {
		return s.SetVerificationTokenFn(ctx, id, token)
	}
	return nil
}

func (s *UserService) VerifyEmail(ctx context.Context, token string) (*aletheia.User, error) {
	if s.VerifyEmailFn != nil {
		return s.VerifyEmailFn(ctx, token)
	}
	return nil, aletheia.NotFound("Invalid token")
}

func (s *UserService) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	if s.RequestPasswordResetFn != nil {
		return s.RequestPasswordResetFn(ctx, email)
	}
	return "mock-reset-token", nil
}

func (s *UserService) ResetPassword(ctx context.Context, token, newPassword string) error {
	if s.ResetPasswordFn != nil {
		return s.ResetPasswordFn(ctx, token, newPassword)
	}
	return nil
}
