package aletheia

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Session represents an active user session.
type Session struct {
	ID        int       `json:"id"`
	UserID    uuid.UUID `json:"userId"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`

	// Joined fields (populated by some queries)
	User *User `json:"user,omitempty"`
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsValid returns true if the session is valid (not expired).
func (s *Session) IsValid() bool {
	return !s.IsExpired()
}

// TimeUntilExpiry returns the duration until the session expires.
func (s *Session) TimeUntilExpiry() time.Duration {
	return time.Until(s.ExpiresAt)
}

// SessionService defines operations for managing user sessions.
type SessionService interface {
	// CreateSession creates a new session for a user.
	// Returns the session with a generated token.
	CreateSession(ctx context.Context, userID uuid.UUID, duration time.Duration) (*Session, error)

	// FindSessionByToken retrieves a session by its token.
	// Returns ENOTFOUND if the session does not exist.
	// Returns EUNAUTHORIZED if the session has expired.
	FindSessionByToken(ctx context.Context, token string) (*Session, error)

	// FindSessionByTokenWithUser retrieves a session with its associated user.
	// Returns ENOTFOUND if the session does not exist.
	// Returns EUNAUTHORIZED if the session has expired.
	FindSessionByTokenWithUser(ctx context.Context, token string) (*Session, error)

	// FindUserSessions retrieves all active sessions for a user.
	FindUserSessions(ctx context.Context, userID uuid.UUID) ([]*Session, error)

	// DeleteSession deletes a session (logout).
	// Returns ENOTFOUND if the session does not exist.
	DeleteSession(ctx context.Context, token string) error

	// DeleteUserSessions deletes all sessions for a user (logout everywhere).
	DeleteUserSessions(ctx context.Context, userID uuid.UUID) error

	// ExtendSession extends the expiration time of a session.
	// Returns ENOTFOUND if the session does not exist.
	ExtendSession(ctx context.Context, token string, duration time.Duration) (*Session, error)

	// CleanupExpiredSessions removes all expired sessions from the database.
	// Returns the number of sessions deleted.
	CleanupExpiredSessions(ctx context.Context) (int, error)
}

// SessionConfig holds session configuration options.
type SessionConfig struct {
	// DefaultDuration is the default session duration.
	DefaultDuration time.Duration

	// MaxDuration is the maximum session duration.
	MaxDuration time.Duration

	// CleanupInterval is how often to clean up expired sessions.
	CleanupInterval time.Duration
}

// DefaultSessionConfig returns the default session configuration.
func DefaultSessionConfig() SessionConfig {
	return SessionConfig{
		DefaultDuration: 24 * time.Hour,
		MaxDuration:     7 * 24 * time.Hour,
		CleanupInterval: 1 * time.Hour,
	}
}
