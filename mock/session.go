package mock

import (
	"context"
	"time"

	"github.com/dukerupert/aletheia"
	"github.com/google/uuid"
)

// Compile-time interface check
var _ aletheia.SessionService = (*SessionService)(nil)

// SessionService is a mock implementation of aletheia.SessionService.
type SessionService struct {
	CreateSessionFn              func(ctx context.Context, userID uuid.UUID, duration time.Duration) (*aletheia.Session, error)
	FindSessionByTokenFn         func(ctx context.Context, token string) (*aletheia.Session, error)
	FindSessionByTokenWithUserFn func(ctx context.Context, token string) (*aletheia.Session, error)
	FindUserSessionsFn           func(ctx context.Context, userID uuid.UUID) ([]*aletheia.Session, error)
	DeleteSessionFn              func(ctx context.Context, token string) error
	DeleteUserSessionsFn         func(ctx context.Context, userID uuid.UUID) error
	ExtendSessionFn              func(ctx context.Context, token string, duration time.Duration) (*aletheia.Session, error)
	CleanupExpiredSessionsFn     func(ctx context.Context) (int, error)

	// Counter for generating session IDs
	nextID int
}

func (s *SessionService) CreateSession(ctx context.Context, userID uuid.UUID, duration time.Duration) (*aletheia.Session, error) {
	if s.CreateSessionFn != nil {
		return s.CreateSessionFn(ctx, userID, duration)
	}
	s.nextID++
	return &aletheia.Session{
		ID:        s.nextID,
		UserID:    userID,
		Token:     "mock-session-token-" + uuid.New().String(),
		ExpiresAt: time.Now().Add(duration),
		CreatedAt: time.Now(),
	}, nil
}

func (s *SessionService) FindSessionByToken(ctx context.Context, token string) (*aletheia.Session, error) {
	if s.FindSessionByTokenFn != nil {
		return s.FindSessionByTokenFn(ctx, token)
	}
	return nil, aletheia.Unauthorized("Session not found or expired")
}

func (s *SessionService) FindSessionByTokenWithUser(ctx context.Context, token string) (*aletheia.Session, error) {
	if s.FindSessionByTokenWithUserFn != nil {
		return s.FindSessionByTokenWithUserFn(ctx, token)
	}
	return nil, aletheia.Unauthorized("Session not found or expired")
}

func (s *SessionService) FindUserSessions(ctx context.Context, userID uuid.UUID) ([]*aletheia.Session, error) {
	if s.FindUserSessionsFn != nil {
		return s.FindUserSessionsFn(ctx, userID)
	}
	return []*aletheia.Session{}, nil
}

func (s *SessionService) DeleteSession(ctx context.Context, token string) error {
	if s.DeleteSessionFn != nil {
		return s.DeleteSessionFn(ctx, token)
	}
	return nil
}

func (s *SessionService) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	if s.DeleteUserSessionsFn != nil {
		return s.DeleteUserSessionsFn(ctx, userID)
	}
	return nil
}

func (s *SessionService) ExtendSession(ctx context.Context, token string, duration time.Duration) (*aletheia.Session, error) {
	if s.ExtendSessionFn != nil {
		return s.ExtendSessionFn(ctx, token, duration)
	}
	return nil, aletheia.NotFound("Session not found")
}

func (s *SessionService) CleanupExpiredSessions(ctx context.Context) (int, error) {
	if s.CleanupExpiredSessionsFn != nil {
		return s.CleanupExpiredSessionsFn(ctx)
	}
	return 0, nil
}
