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
	"github.com/dukerupert/aletheia/internal/email"
	"github.com/dukerupert/aletheia/internal/session"
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

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

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

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

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

	err = handler.Register(c2)
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

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

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

func TestLogin(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	testEmail := "logintest@example.com"
	testPassword := "securepassword123"

	// Clean up before test
	cleanupTestUser(t, pool, testEmail)

	// Register user first
	regBody := RegisterRequest{
		Email:    testEmail,
		Username: "logintest",
		Password: testPassword,
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Register(c); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Test successful login
	loginBody := LoginRequest{
		Email:    testEmail,
		Password: testPassword,
	}
	body, _ = json.Marshal(loginBody)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := handler.Login(c); err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp LoginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Email != testEmail {
		t.Errorf("Expected email %s, got %s", testEmail, resp.Email)
	}

	// Check that session cookie was set
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == session.SessionCookieName {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("Expected session cookie to be set")
	}

	if sessionCookie.Value == "" {
		t.Error("Session cookie value should not be empty")
	}

	if !sessionCookie.HttpOnly {
		t.Error("Session cookie should be HttpOnly")
	}

	// Clean up
	cleanupTestUser(t, pool, testEmail)
}

func TestLoginInvalidPassword(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	testEmail := "invalidpw@example.com"
	testPassword := "correctpassword"

	// Clean up before test
	cleanupTestUser(t, pool, testEmail)

	// Register user
	regBody := RegisterRequest{
		Email:    testEmail,
		Username: "invalidpwtest",
		Password: testPassword,
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Register(c); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Try login with wrong password
	loginBody := LoginRequest{
		Email:    testEmail,
		Password: "wrongpassword",
	}
	body, _ = json.Marshal(loginBody)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	err = handler.Login(c)
	if err == nil {
		t.Fatal("Expected login to fail with wrong password")
	}

	if he, ok := err.(*echo.HTTPError); ok {
		if he.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, he.Code)
		}
	}

	// Clean up
	cleanupTestUser(t, pool, testEmail)
}

func TestLoginNonExistentUser(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	loginBody := LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(loginBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = handler.Login(c)
	if err == nil {
		t.Fatal("Expected login to fail for non-existent user")
	}

	if he, ok := err.(*echo.HTTPError); ok {
		if he.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, he.Code)
		}
	}
}

func TestLogout(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	testEmail := "logouttest@example.com"
	testPassword := "securepassword123"

	// Clean up before test
	cleanupTestUser(t, pool, testEmail)

	// Register and login user
	regBody := RegisterRequest{
		Email:    testEmail,
		Username: "logouttest",
		Password: testPassword,
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Register(c); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Login
	loginBody := LoginRequest{
		Email:    testEmail,
		Password: testPassword,
	}
	body, _ = json.Marshal(loginBody)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := handler.Login(c); err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	// Get session cookie from login response
	var sessionCookie *http.Cookie
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == session.SessionCookieName {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("No session cookie found after login")
	}

	// Test logout
	req = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := handler.Logout(c); err != nil {
		t.Fatalf("Logout failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Check that session cookie was cleared
	var logoutCookie *http.Cookie
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == session.SessionCookieName {
			logoutCookie = cookie
			break
		}
	}

	if logoutCookie == nil {
		t.Fatal("Expected session cookie to be cleared")
	}

	if logoutCookie.MaxAge != -1 {
		t.Errorf("Expected MaxAge -1 to clear cookie, got %d", logoutCookie.MaxAge)
	}

	// Verify session is actually deleted from database
	_, verifyErr := session.GetSession(context.Background(), pool, sessionCookie.Value)
	if verifyErr == nil {
		t.Error("Expected session to be deleted from database")
	}

	// Clean up
	cleanupTestUser(t, pool, testEmail)
}

func TestLogoutWithoutSession(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = handler.Logout(c)
	if err == nil {
		t.Fatal("Expected logout to fail without session cookie")
	}

	if he, ok := err.(*echo.HTTPError); ok {
		if he.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, he.Code)
		}
	}
}

func TestMeEndpoint(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	testEmail := "metest@example.com"
	testPassword := "securepassword123"
	testUsername := "metest"
	testFirstName := "Test"
	testLastName := "User"

	// Clean up before test
	cleanupTestUser(t, pool, testEmail)

	// Register user
	regBody := RegisterRequest{
		Email:     testEmail,
		Username:  testUsername,
		Password:  testPassword,
		FirstName: testFirstName,
		LastName:  testLastName,
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Register(c); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Login
	loginBody := LoginRequest{
		Email:    testEmail,
		Password: testPassword,
	}
	body, _ = json.Marshal(loginBody)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := handler.Login(c); err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	// Get session cookie
	var sessionCookie *http.Cookie
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == session.SessionCookieName {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("No session cookie found after login")
	}

	// Test /me endpoint with session
	req = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	// Manually set user ID in context (simulating middleware)
	sess, err := session.GetSession(context.Background(), pool, sessionCookie.Value)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}
	c.Set(string(session.UserIDKey), sess.UserID.Bytes)

	if err := handler.Me(c); err != nil {
		t.Fatalf("Me endpoint failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var meResp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &meResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if meResp["email"] != testEmail {
		t.Errorf("Expected email %s, got %s", testEmail, meResp["email"])
	}

	if meResp["username"] != testUsername {
		t.Errorf("Expected username %s, got %s", testUsername, meResp["username"])
	}

	if meResp["first_name"] != testFirstName {
		t.Errorf("Expected first_name %s, got %s", testFirstName, meResp["first_name"])
	}

	if meResp["last_name"] != testLastName {
		t.Errorf("Expected last_name %s, got %s", testLastName, meResp["last_name"])
	}

	// Clean up
	cleanupTestUser(t, pool, testEmail)
}

func TestMeEndpointWithoutSession(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = handler.Me(c)
	if err == nil {
		t.Fatal("Expected /me endpoint to fail without session")
	}

	if he, ok := err.(*echo.HTTPError); ok {
		if he.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, he.Code)
		}
	}
}

func TestUpdateProfile(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	testEmail := "updateprofile@example.com"
	testPassword := "securepassword123"

	// Clean up before test
	cleanupTestUser(t, pool, testEmail)

	// Register user
	regBody := RegisterRequest{
		Email:     testEmail,
		Username:  "updateprofile",
		Password:  testPassword,
		FirstName: "Original",
		LastName:  "Name",
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Register(c); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Login to get session
	loginBody := LoginRequest{
		Email:    testEmail,
		Password: testPassword,
	}
	body, _ = json.Marshal(loginBody)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := handler.Login(c); err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	// Get session cookie
	var sessionCookie *http.Cookie
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == session.SessionCookieName {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("No session cookie found after login")
	}

	// Get session and set context
	sess, err := session.GetSession(context.Background(), pool, sessionCookie.Value)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	// Update profile
	newFirstName := "Updated"
	newLastName := "Profile"
	updateBody := UpdateProfileRequest{
		FirstName: &newFirstName,
		LastName:  &newLastName,
	}
	body, _ = json.Marshal(updateBody)
	req = httptest.NewRequest(http.MethodPut, "/api/auth/profile", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	// Set user ID in context (simulating middleware)
	c.Set(string(session.UserIDKey), sess.UserID.Bytes)

	if err := handler.UpdateProfile(c); err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var updateResp UpdateProfileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &updateResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if updateResp.FirstName != newFirstName {
		t.Errorf("Expected first_name %s, got %s", newFirstName, updateResp.FirstName)
	}

	if updateResp.LastName != newLastName {
		t.Errorf("Expected last_name %s, got %s", newLastName, updateResp.LastName)
	}

	// Verify the update persisted by calling /me
	req = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(string(session.UserIDKey), sess.UserID.Bytes)

	if err := handler.Me(c); err != nil {
		t.Fatalf("Me endpoint failed: %v", err)
	}

	var meResp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &meResp); err != nil {
		t.Fatalf("Failed to unmarshal me response: %v", err)
	}

	if meResp["first_name"] != newFirstName {
		t.Errorf("Profile update did not persist. Expected first_name %s, got %v", newFirstName, meResp["first_name"])
	}

	// Clean up
	cleanupTestUser(t, pool, testEmail)
}

func TestUpdateProfilePartial(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	testEmail := "partialupdate@example.com"
	testPassword := "securepassword123"

	// Clean up before test
	cleanupTestUser(t, pool, testEmail)

	// Register user
	regBody := RegisterRequest{
		Email:     testEmail,
		Username:  "partialupdate",
		Password:  testPassword,
		FirstName: "Original",
		LastName:  "Name",
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Register(c); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Login
	loginBody := LoginRequest{
		Email:    testEmail,
		Password: testPassword,
	}
	body, _ = json.Marshal(loginBody)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := handler.Login(c); err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	var sessionCookie *http.Cookie
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == session.SessionCookieName {
			sessionCookie = cookie
			break
		}
	}

	sess, err := session.GetSession(context.Background(), pool, sessionCookie.Value)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	// Update only first name
	newFirstName := "OnlyFirst"
	updateBody := UpdateProfileRequest{
		FirstName: &newFirstName,
	}
	body, _ = json.Marshal(updateBody)
	req = httptest.NewRequest(http.MethodPut, "/api/auth/profile", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(string(session.UserIDKey), sess.UserID.Bytes)

	if err := handler.UpdateProfile(c); err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}

	var updateResp UpdateProfileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &updateResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if updateResp.FirstName != newFirstName {
		t.Errorf("Expected first_name %s, got %s", newFirstName, updateResp.FirstName)
	}

	// Last name should remain unchanged
	if updateResp.LastName != "Name" {
		t.Errorf("Expected last_name to remain 'Name', got %s", updateResp.LastName)
	}

	// Clean up
	cleanupTestUser(t, pool, testEmail)
}

func TestUpdateProfileWithoutSession(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	newFirstName := "Should"
	newLastName := "Fail"
	updateBody := UpdateProfileRequest{
		FirstName: &newFirstName,
		LastName:  &newLastName,
	}
	body, _ := json.Marshal(updateBody)
	req := httptest.NewRequest(http.MethodPut, "/api/auth/profile", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = handler.UpdateProfile(c)
	if err == nil {
		t.Fatal("Expected UpdateProfile to fail without session")
	}

	if he, ok := err.(*echo.HTTPError); ok {
		if he.Code != http.StatusUnauthorized {
			t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, he.Code)
		}
	}
}

func TestVerifyEmail(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	testEmail := "verifytest@example.com"
	testUsername := "verifytest"

	// Clean up before test
	cleanupTestUser(t, pool, testEmail)

	// Register a user
	regBody := RegisterRequest{
		Email:    testEmail,
		Username: testUsername,
		Password: "securepassword123",
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Register(c); err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	// Get the verification token from the database
	queries := database.New(pool)
	user, err := queries.GetUserByEmail(context.Background(), testEmail)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if !user.VerificationToken.Valid {
		t.Fatal("Expected verification token to be set")
	}

	// Verify the user is not yet verified
	if user.VerifiedAt.Valid {
		t.Fatal("Expected user to not be verified yet")
	}

	// Test successful verification
	verifyBody := VerifyEmailRequest{
		Token: user.VerificationToken.String,
	}
	body, _ = json.Marshal(verifyBody)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/verify-email", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := handler.VerifyEmail(c); err != nil {
		t.Fatalf("Failed to verify email: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Verify the user is now verified
	user, err = queries.GetUserByEmail(context.Background(), testEmail)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if !user.VerifiedAt.Valid {
		t.Fatal("Expected user to be verified")
	}

	if user.VerificationToken.Valid {
		t.Fatal("Expected verification token to be cleared")
	}

	// Clean up
	cleanupTestUser(t, pool, testEmail)
}

func TestVerifyEmailInvalidToken(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	// Test with invalid token
	verifyBody := VerifyEmailRequest{
		Token: "invalid-token-12345",
	}
	body, _ := json.Marshal(verifyBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/verify-email", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = handler.VerifyEmail(c)
	if err == nil {
		t.Fatal("Expected VerifyEmail to fail with invalid token")
	}

	if he, ok := err.(*echo.HTTPError); ok {
		if he.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, he.Code)
		}
	}
}

func TestVerifyEmailAlreadyVerified(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	testEmail := "alreadyverified@example.com"
	testUsername := "alreadyverified"

	// Clean up before test
	cleanupTestUser(t, pool, testEmail)

	// Register a user
	regBody := RegisterRequest{
		Email:    testEmail,
		Username: testUsername,
		Password: "securepassword123",
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Register(c); err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	// Get the verification token
	queries := database.New(pool)
	user, err := queries.GetUserByEmail(context.Background(), testEmail)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	token := user.VerificationToken.String

	// Verify once
	verifyBody := VerifyEmailRequest{
		Token: token,
	}
	body, _ = json.Marshal(verifyBody)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/verify-email", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := handler.VerifyEmail(c); err != nil {
		t.Fatalf("Failed to verify email: %v", err)
	}

	// Try to verify again with the same token
	req = httptest.NewRequest(http.MethodPost, "/api/auth/verify-email", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	err = handler.VerifyEmail(c)
	if err == nil {
		t.Fatal("Expected VerifyEmail to fail when already verified")
	}

	if he, ok := err.(*echo.HTTPError); ok {
		if he.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, he.Code)
		}
	}

	// Clean up
	cleanupTestUser(t, pool, testEmail)
}

func TestResendVerification(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	testEmail := "resendtest@example.com"
	testUsername := "resendtest"

	// Clean up before test
	cleanupTestUser(t, pool, testEmail)

	// Register a user
	regBody := RegisterRequest{
		Email:    testEmail,
		Username: testUsername,
		Password: "securepassword123",
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Register(c); err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	// Get the original verification token
	queries := database.New(pool)
	user, err := queries.GetUserByEmail(context.Background(), testEmail)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	originalToken := user.VerificationToken.String

	// Resend verification email
	resendBody := ResendVerificationRequest{
		Email: testEmail,
	}
	body, _ = json.Marshal(resendBody)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/resend-verification", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := handler.ResendVerification(c); err != nil {
		t.Fatalf("Failed to resend verification: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Verify that a new token was generated
	user, err = queries.GetUserByEmail(context.Background(), testEmail)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if !user.VerificationToken.Valid {
		t.Fatal("Expected verification token to be set")
	}

	if user.VerificationToken.String == originalToken {
		t.Error("Expected new verification token to be different from original")
	}

	// Clean up
	cleanupTestUser(t, pool, testEmail)
}

func TestResendVerificationNonExistentEmail(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	// Test with non-existent email
	resendBody := ResendVerificationRequest{
		Email: "nonexistent@example.com",
	}
	body, _ := json.Marshal(resendBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/resend-verification", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.ResendVerification(c); err != nil {
		t.Fatalf("ResendVerification should not error for non-existent email: %v", err)
	}

	// Should return 200 to avoid leaking whether email exists
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestResendVerificationAlreadyVerified(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	testEmail := "alreadyverifiedresend@example.com"
	testUsername := "alreadyverifiedresend"

	// Clean up before test
	cleanupTestUser(t, pool, testEmail)

	// Register a user
	regBody := RegisterRequest{
		Email:    testEmail,
		Username: testUsername,
		Password: "securepassword123",
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Register(c); err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	// Get the verification token and verify
	queries := database.New(pool)
	user, err := queries.GetUserByEmail(context.Background(), testEmail)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	verifyBody := VerifyEmailRequest{
		Token: user.VerificationToken.String,
	}
	body, _ = json.Marshal(verifyBody)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/verify-email", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := handler.VerifyEmail(c); err != nil {
		t.Fatalf("Failed to verify email: %v", err)
	}

	// Try to resend verification for already verified user
	resendBody := ResendVerificationRequest{
		Email: testEmail,
	}
	body, _ = json.Marshal(resendBody)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/resend-verification", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := handler.ResendVerification(c); err != nil {
		t.Fatalf("ResendVerification should not error for verified user: %v", err)
	}

	// Should return 200 to avoid leaking verification status
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Clean up
	cleanupTestUser(t, pool, testEmail)
}

func TestRequestPasswordReset(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	testEmail := "resettest@example.com"
	testUsername := "resettest"

	// Clean up before test
	cleanupTestUser(t, pool, testEmail)

	// Register a user
	regBody := RegisterRequest{
		Email:    testEmail,
		Username: testUsername,
		Password: "securepassword123",
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Register(c); err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	// Request password reset
	resetReq := RequestPasswordResetRequest{
		Email: testEmail,
	}
	body, _ = json.Marshal(resetReq)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/request-password-reset", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := handler.RequestPasswordReset(c); err != nil {
		t.Fatalf("Failed to request password reset: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Verify reset token was set
	queries := database.New(pool)
	user, err := queries.GetUserByEmail(context.Background(), testEmail)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if !user.ResetToken.Valid {
		t.Fatal("Expected reset token to be set")
	}

	if !user.ResetTokenExpiresAt.Valid {
		t.Fatal("Expected reset token expiration to be set")
	}

	// Clean up
	cleanupTestUser(t, pool, testEmail)
}

func TestRequestPasswordResetNonExistentEmail(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	// Request password reset for non-existent email
	resetReq := RequestPasswordResetRequest{
		Email: "nonexistent@example.com",
	}
	body, _ := json.Marshal(resetReq)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/request-password-reset", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.RequestPasswordReset(c); err != nil {
		t.Fatalf("RequestPasswordReset should not error for non-existent email: %v", err)
	}

	// Should return 200 to avoid leaking whether email exists
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestVerifyResetToken(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	testEmail := "verifyreset@example.com"
	testUsername := "verifyreset"

	// Clean up before test
	cleanupTestUser(t, pool, testEmail)

	// Register a user
	regBody := RegisterRequest{
		Email:    testEmail,
		Username: testUsername,
		Password: "securepassword123",
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Register(c); err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	// Request password reset
	resetReq := RequestPasswordResetRequest{
		Email: testEmail,
	}
	body, _ = json.Marshal(resetReq)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/request-password-reset", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := handler.RequestPasswordReset(c); err != nil {
		t.Fatalf("Failed to request password reset: %v", err)
	}

	// Get the reset token from the database
	queries := database.New(pool)
	user, err := queries.GetUserByEmail(context.Background(), testEmail)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	// Verify reset token
	verifyReq := VerifyResetTokenRequest{
		Token: user.ResetToken.String,
	}
	body, _ = json.Marshal(verifyReq)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/verify-reset-token", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := handler.VerifyResetToken(c); err != nil {
		t.Fatalf("Failed to verify reset token: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Clean up
	cleanupTestUser(t, pool, testEmail)
}

func TestVerifyResetTokenInvalid(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	// Test with invalid token
	verifyReq := VerifyResetTokenRequest{
		Token: "invalid-token-12345",
	}
	body, _ := json.Marshal(verifyReq)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/verify-reset-token", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = handler.VerifyResetToken(c)
	if err == nil {
		t.Fatal("Expected VerifyResetToken to fail with invalid token")
	}

	if he, ok := err.(*echo.HTTPError); ok {
		if he.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, he.Code)
		}
	}
}

func TestResetPassword(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	testEmail := "resetpasswordtest@example.com"
	testUsername := "resetpasswordtest"
	oldPassword := "securepassword123"
	newPassword := "newsecurepassword456"

	// Clean up before test
	cleanupTestUser(t, pool, testEmail)

	// Register a user
	regBody := RegisterRequest{
		Email:    testEmail,
		Username: testUsername,
		Password: oldPassword,
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Register(c); err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	// Request password reset
	resetReq := RequestPasswordResetRequest{
		Email: testEmail,
	}
	body, _ = json.Marshal(resetReq)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/request-password-reset", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := handler.RequestPasswordReset(c); err != nil {
		t.Fatalf("Failed to request password reset: %v", err)
	}

	// Get the reset token from the database
	queries := database.New(pool)
	user, err := queries.GetUserByEmail(context.Background(), testEmail)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	resetToken := user.ResetToken.String

	// Reset password with new password
	resetPasswordReq := ResetPasswordRequest{
		Token:       resetToken,
		NewPassword: newPassword,
	}
	body, _ = json.Marshal(resetPasswordReq)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/reset-password", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := handler.ResetPassword(c); err != nil {
		t.Fatalf("Failed to reset password: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	// Verify reset token was cleared
	user, err = queries.GetUserByEmail(context.Background(), testEmail)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if user.ResetToken.Valid {
		t.Error("Expected reset token to be cleared")
	}

	if user.ResetTokenExpiresAt.Valid {
		t.Error("Expected reset token expiration to be cleared")
	}

	// Try to login with old password - should fail
	loginReq := LoginRequest{
		Email:    testEmail,
		Password: oldPassword,
	}
	body, _ = json.Marshal(loginReq)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	err = handler.Login(c)
	if err == nil {
		t.Fatal("Expected login with old password to fail")
	}

	// Try to login with new password - should succeed
	loginReq.Password = newPassword
	body, _ = json.Marshal(loginReq)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := handler.Login(c); err != nil {
		t.Fatalf("Failed to login with new password: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected login to succeed with new password. Status: %d", rec.Code)
	}

	// Clean up
	cleanupTestUser(t, pool, testEmail)
}

func TestResetPasswordInvalidToken(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Config not available: %v", err)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	emailService := email.NewEmailService(logger, email.EmailConfig{
		Provider:      "mock",
		FromAddress:   "test@example.com",
		FromName:      "Test",
		VerifyBaseURL: "http://localhost:1323",
	})
	handler := NewAuthHandler(pool, logger, emailService, cfg)

	e := echo.New()

	// Try to reset password with invalid token
	resetPasswordReq := ResetPasswordRequest{
		Token:       "invalid-token-12345",
		NewPassword: "newsecurepassword456",
	}
	body, _ := json.Marshal(resetPasswordReq)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/reset-password", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = handler.ResetPassword(c)
	if err == nil {
		t.Fatal("Expected ResetPassword to fail with invalid token")
	}

	if he, ok := err.(*echo.HTTPError); ok {
		if he.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, he.Code)
		}
	}
}
