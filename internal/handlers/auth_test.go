package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/dukerupert/aletheia/internal/config"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
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

// cleanupTestUser removes a test user by email
func cleanupTestUser(t *testing.T, pool *pgxpool.Pool, email string) {
	queries := database.New(pool)
	user, err := queries.GetUserByEmail(context.Background(), email)
	if err == nil {
		// User exists, delete them
		_ = queries.DeleteUser(context.Background(), user.ID)
	}
}

func TestRegister(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewAuthHandler(pool, logger)

	e := echo.New()

	testEmail := "testuser@example.com"
	testUsername := "testuser"

	// Clean up before test
	cleanupTestUser(t, pool, testEmail)

	// Test successful registration
	reqBody := RegisterRequest{
		Email:     testEmail,
		Username:  testUsername,
		Password:  "securepassword123",
		FirstName: "Test",
		LastName:  "User",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Call handler
	if err := handler.Register(c); err != nil {
		if he, ok := err.(*echo.HTTPError); ok {
			t.Fatalf("Expected success, got error with code %d: %v", he.Code, he.Message)
		}
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	var resp RegisterResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Email != testEmail {
		t.Errorf("Expected email %s, got %s", testEmail, resp.Email)
	}

	if resp.Username != testUsername {
		t.Errorf("Expected username %s, got %s", testUsername, resp.Username)
	}

	// Clean up after test
	cleanupTestUser(t, pool, testEmail)
}

func TestRegisterDuplicateEmail(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewAuthHandler(pool, logger)

	e := echo.New()

	testEmail := "duplicate@example.com"

	// Clean up before test
	cleanupTestUser(t, pool, testEmail)

	// Create first user
	reqBody1 := RegisterRequest{
		Email:    testEmail,
		Username: "user1",
		Password: "password123",
	}

	body1, _ := json.Marshal(reqBody1)
	req1 := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body1))
	req1.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, rec1)

	if err := handler.Register(c1); err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	// Try to create second user with same email
	reqBody2 := RegisterRequest{
		Email:    testEmail,
		Username: "user2",
		Password: "password456",
	}

	body2, _ := json.Marshal(reqBody2)
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body2))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)

	err := handler.Register(c2)
	if err == nil {
		t.Fatal("Expected conflict error for duplicate email")
	}

	if he, ok := err.(*echo.HTTPError); ok {
		if he.Code != http.StatusConflict {
			t.Errorf("Expected status %d, got %d", http.StatusConflict, he.Code)
		}
	}

	// Clean up after test
	cleanupTestUser(t, pool, testEmail)
}

func TestRegisterValidation(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewAuthHandler(pool, logger)

	e := echo.New()

	tests := []struct {
		name           string
		request        RegisterRequest
		expectedStatus int
	}{
		{
			name: "missing email",
			request: RegisterRequest{
				Username: "testuser",
				Password: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing username",
			request: RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing password",
			request: RegisterRequest{
				Email:    "test@example.com",
				Username: "testuser",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "password too short",
			request: RegisterRequest{
				Email:    "test@example.com",
				Username: "testuser",
				Password: "short",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler.Register(c)
			if err != nil {
				if he, ok := err.(*echo.HTTPError); ok {
					if he.Code != tt.expectedStatus {
						t.Errorf("Expected status %d, got %d", tt.expectedStatus, he.Code)
					}
				}
			} else if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}
