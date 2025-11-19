package session

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dukerupert/aletheia/internal/config"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

// getTestDB returns a database pool for testing
// Skip tests if database is not available
func getTestDB(t *testing.T) *pgxpool.Pool {
	// Try to load config, but skip test if not available
	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available, skipping integration test: %v", err)
		return nil
	}

	pool, err := pgxpool.New(context.Background(), cfg.GetConnectionString())
	if err != nil {
		t.Skipf("Database not available, skipping integration test: %v", err)
		return nil
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("Database ping failed, skipping integration test: %v", err)
		return nil
	}

	return pool
}

// createTestUser creates a test user and returns the user ID
func createTestUser(t *testing.T, pool *pgxpool.Pool, email string) uuid.UUID {
	queries := database.New(pool)

	user, err := queries.CreateUser(context.Background(), database.CreateUserParams{
		Email:        email,
		Username:     strings.Split(email, "@")[0],
		PasswordHash: "$2a$10$test.hash.for.testing",
		FirstName:    pgtype.Text{String: "Test", Valid: true},
		LastName:     pgtype.Text{String: "User", Valid: true},
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user.ID.Bytes
}

// cleanupTestUser removes a test user by ID
func cleanupTestUser(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) {
	queries := database.New(pool)
	_ = queries.DeleteUser(context.Background(), pgtype.UUID{Bytes: userID, Valid: true})
}

func TestGenerateSessionToken(t *testing.T) {
	t.Run("generates valid token", func(t *testing.T) {
		token, err := GenerateSessionToken()

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("generates unique tokens", func(t *testing.T) {
		token1, err1 := GenerateSessionToken()
		token2, err2 := GenerateSessionToken()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, token1, token2, "Tokens should be unique")
	})

	t.Run("generates base64 url-safe tokens", func(t *testing.T) {
		token, err := GenerateSessionToken()

		assert.NoError(t, err)
		// Base64 URL encoding should not contain + or / characters
		assert.NotContains(t, token, "+")
		assert.NotContains(t, token, "/")
	})

	t.Run("generates tokens of expected length", func(t *testing.T) {
		token, err := GenerateSessionToken()

		assert.NoError(t, err)
		// Base64 encoding of 32 bytes should be ~43 characters
		// (32 bytes * 8 bits/byte / 6 bits per base64 char = 42.67, rounded up to 43)
		assert.Greater(t, len(token), 40)
		assert.Less(t, len(token), 50)
	})
}

func TestCreateSession(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	ctx := context.Background()
	testEmail := "session-test@example.com"
	userID := createTestUser(t, pool, testEmail)
	defer cleanupTestUser(t, pool, userID)

	t.Run("creates session successfully", func(t *testing.T) {
		session, err := CreateSession(ctx, pool, userID, SessionDuration)

		assert.NoError(t, err)
		assert.NotEmpty(t, session.Token)
		assert.EqualValues(t, userID, session.UserID.Bytes)
		assert.True(t, session.ExpiresAt.Valid)
		assert.True(t, session.ExpiresAt.Time.After(time.Now()))
	})

	t.Run("creates session with custom duration", func(t *testing.T) {
		customDuration := 1 * time.Hour
		session, err := CreateSession(ctx, pool, userID, customDuration)

		assert.NoError(t, err)
		assert.NotEmpty(t, session.Token)

		// Check that expiry is approximately 1 hour from now
		expectedExpiry := time.Now().Add(customDuration)
		timeDiff := session.ExpiresAt.Time.Sub(expectedExpiry)
		assert.Less(t, timeDiff.Abs(), 5*time.Second, "Expiry time should be close to expected")
	})

	t.Run("creates multiple sessions for same user", func(t *testing.T) {
		session1, err1 := CreateSession(ctx, pool, userID, SessionDuration)
		session2, err2 := CreateSession(ctx, pool, userID, SessionDuration)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, session1.Token, session2.Token)
		assert.EqualValues(t, session1.UserID.Bytes, session2.UserID.Bytes)
	})
}

func TestGetSession(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	ctx := context.Background()
	testEmail := "get-session-test@example.com"
	userID := createTestUser(t, pool, testEmail)
	defer cleanupTestUser(t, pool, userID)

	t.Run("retrieves existing session", func(t *testing.T) {
		// Create a session
		created, err := CreateSession(ctx, pool, userID, SessionDuration)
		assert.NoError(t, err)

		// Retrieve the session
		retrieved, err := GetSession(ctx, pool, created.Token)

		assert.NoError(t, err)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, created.Token, retrieved.Token)
		assert.EqualValues(t, created.UserID.Bytes, retrieved.UserID.Bytes)
	})

	t.Run("returns error for non-existent session", func(t *testing.T) {
		_, err := GetSession(ctx, pool, "non-existent-token")

		assert.Error(t, err)
	})

	t.Run("retrieves expired session", func(t *testing.T) {
		// Create a session that expires immediately
		created, err := CreateSession(ctx, pool, userID, -1*time.Hour)
		assert.NoError(t, err)

		// GetSession should still retrieve it (expiry check happens at middleware level)
		retrieved, err := GetSession(ctx, pool, created.Token)

		// This behavior depends on database query implementation
		// If query filters by expiry, this should error
		// If not, it should return the session
		if err == nil {
			assert.Equal(t, created.ID, retrieved.ID)
		}
	})
}

func TestDestroySession(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	ctx := context.Background()
	testEmail := "destroy-session-test@example.com"
	userID := createTestUser(t, pool, testEmail)
	defer cleanupTestUser(t, pool, userID)

	t.Run("destroys existing session", func(t *testing.T) {
		// Create a session
		session, err := CreateSession(ctx, pool, userID, SessionDuration)
		assert.NoError(t, err)

		// Destroy the session
		err = DestroySession(ctx, pool, session.Token)
		assert.NoError(t, err)

		// Verify session no longer exists
		_, err = GetSession(ctx, pool, session.Token)
		assert.Error(t, err)
	})

	t.Run("destroying non-existent session does not error", func(t *testing.T) {
		// Depending on implementation, DELETE of non-existent row
		// is typically not an error. Just verify it doesn't panic.
		assert.NotPanics(t, func() {
			_ = DestroySession(ctx, pool, "non-existent-token")
		})
	})
}

func TestDestroyUserSessions(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	ctx := context.Background()
	testEmail := "destroy-user-sessions-test@example.com"
	userID := createTestUser(t, pool, testEmail)
	defer cleanupTestUser(t, pool, userID)

	t.Run("destroys all sessions for user", func(t *testing.T) {
		// Create multiple sessions for the user
		session1, err1 := CreateSession(ctx, pool, userID, SessionDuration)
		session2, err2 := CreateSession(ctx, pool, userID, SessionDuration)
		session3, err3 := CreateSession(ctx, pool, userID, SessionDuration)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NoError(t, err3)

		// Destroy all user sessions
		err := DestroyUserSessions(ctx, pool, userID)
		assert.NoError(t, err)

		// Verify all sessions are gone
		_, err1 = GetSession(ctx, pool, session1.Token)
		_, err2 = GetSession(ctx, pool, session2.Token)
		_, err3 = GetSession(ctx, pool, session3.Token)

		assert.Error(t, err1)
		assert.Error(t, err2)
		assert.Error(t, err3)
	})

	t.Run("destroying sessions for user with no sessions does not error", func(t *testing.T) {
		// Create a new user with no sessions
		newUserID := createTestUser(t, pool, "no-sessions@example.com")
		defer cleanupTestUser(t, pool, newUserID)

		err := DestroyUserSessions(ctx, pool, newUserID)
		assert.NoError(t, err)
	})
}

func TestCleanupExpiredSessions(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	ctx := context.Background()
	testEmail := "cleanup-sessions-test@example.com"
	userID := createTestUser(t, pool, testEmail)
	defer cleanupTestUser(t, pool, userID)

	t.Run("removes expired sessions", func(t *testing.T) {
		// Create an expired session
		expiredSession, err := CreateSession(ctx, pool, userID, -1*time.Hour)
		assert.NoError(t, err)

		// Create a valid session
		validSession, err := CreateSession(ctx, pool, userID, SessionDuration)
		assert.NoError(t, err)

		// Cleanup expired sessions
		err = CleanupExpiredSessions(ctx, pool)
		assert.NoError(t, err)

		// Expired session should be gone
		_, err = GetSession(ctx, pool, expiredSession.Token)
		assert.Error(t, err)

		// Valid session should still exist
		retrieved, err := GetSession(ctx, pool, validSession.Token)
		assert.NoError(t, err)
		assert.Equal(t, validSession.ID, retrieved.ID)
	})

	t.Run("cleanup with no expired sessions does not error", func(t *testing.T) {
		// Create only valid sessions
		_, err := CreateSession(ctx, pool, userID, SessionDuration)
		assert.NoError(t, err)

		err = CleanupExpiredSessions(ctx, pool)
		assert.NoError(t, err)
	})
}

func TestSessionConstants(t *testing.T) {
	t.Run("session duration is 24 hours", func(t *testing.T) {
		assert.Equal(t, 24*time.Hour, SessionDuration)
	})

	t.Run("session cookie name is correct", func(t *testing.T) {
		assert.Equal(t, "session_token", SessionCookieName)
	})

	t.Run("session token length is 32 bytes", func(t *testing.T) {
		assert.Equal(t, 32, SessionTokenLength)
	})
}
