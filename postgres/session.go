package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/dukerupert/aletheia"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Compile-time check that SessionService implements aletheia.SessionService.
var _ aletheia.SessionService = (*SessionService)(nil)

// SessionService implements aletheia.SessionService using PostgreSQL.
type SessionService struct {
	db *DB
}

func (s *SessionService) CreateSession(ctx context.Context, userID uuid.UUID, duration time.Duration) (*aletheia.Session, error) {
	// Generate secure token
	token, err := generateSessionToken(32)
	if err != nil {
		return nil, aletheia.Internal("Failed to generate session token", err)
	}

	expiresAt := time.Now().Add(duration)

	session, err := s.db.queries.CreateSession(ctx, database.CreateSessionParams{
		UserID:    toPgUUID(userID),
		Token:     token,
		ExpiresAt: toPgTimestamp(expiresAt),
	})
	if err != nil {
		if isForeignKeyViolation(err) {
			return nil, aletheia.NotFound("User not found")
		}
		return nil, aletheia.Internal("Failed to create session", err)
	}

	return toDomainSession(session), nil
}

func (s *SessionService) FindSessionByToken(ctx context.Context, token string) (*aletheia.Session, error) {
	session, err := s.db.queries.GetSessionByToken(ctx, token)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.Unauthorized("Session not found or expired")
		}
		return nil, aletheia.Internal("Failed to fetch session", err)
	}

	domainSession := toDomainSession(session)
	if domainSession.IsExpired() {
		return nil, aletheia.Unauthorized("Session expired")
	}

	return domainSession, nil
}

func (s *SessionService) FindSessionByTokenWithUser(ctx context.Context, token string) (*aletheia.Session, error) {
	session, err := s.FindSessionByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Fetch user data
	user, err := s.db.queries.GetUser(ctx, toPgUUID(session.UserID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("User not found")
		}
		return nil, aletheia.Internal("Failed to fetch user", err)
	}

	session.User = toDomainUser(user)
	return session, nil
}

func (s *SessionService) FindUserSessions(ctx context.Context, userID uuid.UUID) ([]*aletheia.Session, error) {
	// Note: This would require a new sqlc query (ListUserSessions)
	// For now, return empty slice
	return []*aletheia.Session{}, nil
}

func (s *SessionService) DeleteSession(ctx context.Context, token string) error {
	err := s.db.queries.DeleteSession(ctx, token)
	if err != nil {
		return aletheia.Internal("Failed to delete session", err)
	}
	return nil
}

func (s *SessionService) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	err := s.db.queries.DeleteUserSessions(ctx, toPgUUID(userID))
	if err != nil {
		return aletheia.Internal("Failed to delete user sessions", err)
	}
	return nil
}

func (s *SessionService) ExtendSession(ctx context.Context, token string, duration time.Duration) (*aletheia.Session, error) {
	// Note: This would require a new sqlc query (UpdateSessionExpiry)
	// For now, just return the existing session
	return s.FindSessionByToken(ctx, token)
}

func (s *SessionService) CleanupExpiredSessions(ctx context.Context) (int, error) {
	err := s.db.queries.DeleteExpiredSessions(ctx)
	if err != nil {
		return 0, aletheia.Internal("Failed to cleanup expired sessions", err)
	}
	// Note: PostgreSQL doesn't return affected rows count with simple exec
	// Would need custom query to return count
	return 0, nil
}

// generateSessionToken generates a cryptographically secure random token.
func generateSessionToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
