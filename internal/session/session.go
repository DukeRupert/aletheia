package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/dukerupert/aletheia/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// SessionDuration is the default session lifetime
	SessionDuration = 24 * time.Hour
	// SessionCookieName is the name of the session cookie
	SessionCookieName = "session_token"
	// SessionTokenLength is the length of the session token in bytes
	SessionTokenLength = 32
)

// GenerateSessionToken generates a cryptographically secure random session token
func GenerateSessionToken() (string, error) {
	b := make([]byte, SessionTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// CreateSession creates a new session for the given user
func CreateSession(ctx context.Context, db *pgxpool.Pool, userID uuid.UUID, duration time.Duration) (database.Session, error) {
	queries := database.New(db)

	token, err := GenerateSessionToken()
	if err != nil {
		return database.Session{}, err
	}

	expiresAt := time.Now().Add(duration)

	session, err := queries.CreateSession(ctx, database.CreateSessionParams{
		UserID: pgtype.UUID{
			Bytes: userID,
			Valid: true,
		},
		Token: token,
		ExpiresAt: pgtype.Timestamptz{
			Time:  expiresAt,
			Valid: true,
		},
	})
	if err != nil {
		return database.Session{}, err
	}

	return session, nil
}

// GetSession retrieves a session by token if it exists and hasn't expired
func GetSession(ctx context.Context, db *pgxpool.Pool, token string) (database.Session, error) {
	queries := database.New(db)
	return queries.GetSessionByToken(ctx, token)
}

// DestroySession deletes a session by token
func DestroySession(ctx context.Context, db *pgxpool.Pool, token string) error {
	queries := database.New(db)
	return queries.DeleteSession(ctx, token)
}

// DestroyUserSessions deletes all sessions for a given user
func DestroyUserSessions(ctx context.Context, db *pgxpool.Pool, userID uuid.UUID) error {
	queries := database.New(db)
	return queries.DeleteUserSessions(ctx, pgtype.UUID{
		Bytes: userID,
		Valid: true,
	})
}

// CleanupExpiredSessions removes all expired sessions from the database
func CleanupExpiredSessions(ctx context.Context, db *pgxpool.Pool) error {
	queries := database.New(db)
	return queries.DeleteExpiredSessions(ctx)
}
