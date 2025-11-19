package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/dukerupert/aletheia/internal/config"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// Test helpers

func setupOrganizationTest(t *testing.T) (*pgxpool.Pool, *slog.Logger, func()) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	logger := slog.New(cfg.GetLogger())

	connString := cfg.GetConnectionString()
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		t.Fatalf("Failed to parse connection string: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	cleanup := func() {
		queries := database.New(pool)

		// Clean up test data
		pool.Exec(context.Background(), "DELETE FROM organization_members WHERE 1=1")
		pool.Exec(context.Background(), "DELETE FROM organizations WHERE 1=1")
		pool.Exec(context.Background(), "DELETE FROM sessions WHERE 1=1")
		pool.Exec(context.Background(), "DELETE FROM users WHERE email LIKE 'test%@example.com'")

		pool.Close()

		// Also cleanup any test data
		_ = queries
	}

	return pool, logger, cleanup
}

func createTestUserWithSession(t *testing.T, pool *pgxpool.Pool, email string) (pgtype.UUID, string) {
	queries := database.New(pool)

	// Create user
	user, err := queries.CreateUser(context.Background(), database.CreateUserParams{
		Email:        email,
		Username:     email,
		PasswordHash: "$2a$10$dummy",
		FirstName:    pgtype.Text{String: "Test", Valid: true},
		LastName:     pgtype.Text{String: "User", Valid: true},
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Mark as verified
	_, err = queries.VerifyUserEmail(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("Failed to verify test user: %v", err)
	}

	// Create session
	userIDStandard, err := uuid.FromBytes(user.ID.Bytes[:])
	if err != nil {
		t.Fatalf("Failed to convert user ID: %v", err)
	}

	sess, err := session.CreateSession(context.Background(), pool, userIDStandard, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	return user.ID, sess.Token
}

// Tests

func TestCreateOrganization(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "testorg1@example.com")

	handler := NewOrganizationHandler(pool, logger)

	e := echo.New()
	reqBody := `{"name":"Test Organization"}`
	req := httptest.NewRequest(http.MethodPost, "/api/organizations", bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.CreateOrganization(c)
	})

	err := h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp CreateOrganizationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Test Organization", resp.Name)
	assert.NotEmpty(t, resp.ID)

	// Verify user was added as owner
	queries := database.New(pool)
	orgUUID, _ := parseUUID(resp.ID)
	member, err := queries.GetOrganizationMemberByUserAndOrg(context.Background(), database.GetOrganizationMemberByUserAndOrgParams{
		OrganizationID: orgUUID,
		UserID:         userID,
	})
	assert.NoError(t, err)
	assert.Equal(t, database.OrganizationRoleOwner, member.Role)
}

func TestGetOrganization(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "testorg2@example.com")
	queries := database.New(pool)

	// Create organization
	org, err := queries.CreateOrganization(context.Background(), "Test Org")
	assert.NoError(t, err)

	// Add user as member
	_, err = queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         userID,
		Role:           database.OrganizationRoleMember,
	})
	assert.NoError(t, err)

	handler := NewOrganizationHandler(pool, logger)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/organizations/"+org.ID.String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/organizations/:id")
	c.SetParamNames("id")
	c.SetParamValues(org.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.GetOrganization(c)
	})

	err = h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp GetOrganizationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Test Org", resp.Name)
}

func TestGetOrganizationUnauthorized(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	// Create user who is NOT a member
	_, sessionID := createTestUserWithSession(t, pool, "testorg3@example.com")
	queries := database.New(pool)

	// Create organization with different user
	otherUserID, _ := createTestUserWithSession(t, pool, "testorg3other@example.com")
	org, err := queries.CreateOrganization(context.Background(), "Other Org")
	assert.NoError(t, err)

	_, err = queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         otherUserID,
		Role:           database.OrganizationRoleOwner,
	})
	assert.NoError(t, err)

	handler := NewOrganizationHandler(pool, logger)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/organizations/"+org.ID.String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/organizations/:id")
	c.SetParamNames("id")
	c.SetParamValues(org.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.GetOrganization(c)
	})

	err = h(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, httpErr.Code)
}

func TestListOrganizations(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "testorg4@example.com")
	queries := database.New(pool)

	// Create multiple organizations
	org1, _ := queries.CreateOrganization(context.Background(), "Org 1")
	org2, _ := queries.CreateOrganization(context.Background(), "Org 2")

	// Add user to both organizations
	queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org1.ID,
		UserID:         userID,
		Role:           database.OrganizationRoleOwner,
	})

	queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org2.ID,
		UserID:         userID,
		Role:           database.OrganizationRoleMember,
	})

	handler := NewOrganizationHandler(pool, logger)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/organizations", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.ListOrganizations(c)
	})

	err := h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp ListOrganizationsResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Organizations, 2)
}

func TestUpdateOrganization(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "testorg5@example.com")
	queries := database.New(pool)

	// Create organization
	org, _ := queries.CreateOrganization(context.Background(), "Old Name")
	queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         userID,
		Role:           database.OrganizationRoleOwner,
	})

	handler := NewOrganizationHandler(pool, logger)

	e := echo.New()
	newName := "New Name"
	reqBody := fmt.Sprintf(`{"name":"%s"}`, newName)
	req := httptest.NewRequest(http.MethodPut, "/api/organizations/"+org.ID.String(), bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/organizations/:id")
	c.SetParamNames("id")
	c.SetParamValues(org.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.UpdateOrganization(c)
	})

	err := h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp UpdateOrganizationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, newName, resp.Name)
}

func TestUpdateOrganizationForbiddenForMember(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "testorg6@example.com")
	queries := database.New(pool)

	// Create organization
	org, _ := queries.CreateOrganization(context.Background(), "Org Name")
	queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         userID,
		Role:           database.OrganizationRoleMember, // Not owner or admin
	})

	handler := NewOrganizationHandler(pool, logger)

	e := echo.New()
	reqBody := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPut, "/api/organizations/"+org.ID.String(), bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/organizations/:id")
	c.SetParamNames("id")
	c.SetParamValues(org.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.UpdateOrganization(c)
	})

	err := h(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, httpErr.Code)
}

func TestDeleteOrganization(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "testorg7@example.com")
	queries := database.New(pool)

	// Create organization
	org, _ := queries.CreateOrganization(context.Background(), "To Delete")
	queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         userID,
		Role:           database.OrganizationRoleOwner,
	})

	handler := NewOrganizationHandler(pool, logger)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/organizations/"+org.ID.String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/organizations/:id")
	c.SetParamNames("id")
	c.SetParamValues(org.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.DeleteOrganization(c)
	})

	err := h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Verify organization is deleted
	_, err = queries.GetOrganization(context.Background(), org.ID)
	assert.Error(t, err)
}

func TestListOrganizationMembers(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	user1ID, sessionID := createTestUserWithSession(t, pool, "testorg8@example.com")
	user2ID, _ := createTestUserWithSession(t, pool, "testorg8other@example.com")
	queries := database.New(pool)

	// Create organization
	org, _ := queries.CreateOrganization(context.Background(), "Team Org")
	queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         user1ID,
		Role:           database.OrganizationRoleOwner,
	})
	queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         user2ID,
		Role:           database.OrganizationRoleMember,
	})

	handler := NewOrganizationHandler(pool, logger)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/organizations/"+org.ID.String()+"/members", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/organizations/:id/members")
	c.SetParamNames("id")
	c.SetParamValues(org.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.ListOrganizationMembers(c)
	})

	err := h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp ListOrganizationMembersResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Members, 2)
}

func TestAddOrganizationMember(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	ownerID, sessionID := createTestUserWithSession(t, pool, "testorg9@example.com")
	newMemberID, _ := createTestUserWithSession(t, pool, "testorg9new@example.com")
	queries := database.New(pool)

	// Get new member's email
	newMember, _ := queries.GetUser(context.Background(), newMemberID)

	// Create organization
	org, _ := queries.CreateOrganization(context.Background(), "Hiring Org")
	queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         ownerID,
		Role:           database.OrganizationRoleOwner,
	})

	handler := NewOrganizationHandler(pool, logger)

	e := echo.New()
	reqBody := fmt.Sprintf(`{"email":"%s","role":"member"}`, newMember.Email)
	req := httptest.NewRequest(http.MethodPost, "/api/organizations/"+org.ID.String()+"/members", bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/organizations/:id/members")
	c.SetParamNames("id")
	c.SetParamValues(org.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.AddOrganizationMember(c)
	})

	err := h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp AddOrganizationMemberResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "member", resp.Role)
}

func TestAddOrganizationMemberAlreadyExists(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	ownerID, sessionID := createTestUserWithSession(t, pool, "testorg10@example.com")
	existingMemberID, _ := createTestUserWithSession(t, pool, "testorg10existing@example.com")
	queries := database.New(pool)

	existingMember, _ := queries.GetUser(context.Background(), existingMemberID)

	// Create organization with existing member
	org, _ := queries.CreateOrganization(context.Background(), "Duplicate Test Org")
	queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         ownerID,
		Role:           database.OrganizationRoleOwner,
	})
	queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         existingMemberID,
		Role:           database.OrganizationRoleMember,
	})

	handler := NewOrganizationHandler(pool, logger)

	e := echo.New()
	reqBody := fmt.Sprintf(`{"email":"%s","role":"admin"}`, existingMember.Email)
	req := httptest.NewRequest(http.MethodPost, "/api/organizations/"+org.ID.String()+"/members", bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/organizations/:id/members")
	c.SetParamNames("id")
	c.SetParamValues(org.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.AddOrganizationMember(c)
	})

	err := h(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusConflict, httpErr.Code)
}

func TestUpdateOrganizationMember(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	ownerID, sessionID := createTestUserWithSession(t, pool, "testorg11@example.com")
	memberID, _ := createTestUserWithSession(t, pool, "testorg11member@example.com")
	queries := database.New(pool)

	// Create organization
	org, _ := queries.CreateOrganization(context.Background(), "Role Change Org")
	queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         ownerID,
		Role:           database.OrganizationRoleOwner,
	})
	member, _ := queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         memberID,
		Role:           database.OrganizationRoleMember,
	})

	handler := NewOrganizationHandler(pool, logger)

	e := echo.New()
	reqBody := `{"role":"admin"}`
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/organizations/%s/members/%s", org.ID.String(), member.ID.String()), bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/organizations/:id/members/:memberId")
	c.SetParamNames("id", "memberId")
	c.SetParamValues(org.ID.String(), member.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.UpdateOrganizationMember(c)
	})

	err := h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp UpdateOrganizationMemberResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "admin", resp.Role)
}

func TestRemoveOrganizationMember(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	ownerID, sessionID := createTestUserWithSession(t, pool, "testorg12@example.com")
	memberID, _ := createTestUserWithSession(t, pool, "testorg12member@example.com")
	queries := database.New(pool)

	// Create organization
	org, _ := queries.CreateOrganization(context.Background(), "Remove Member Org")
	queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         ownerID,
		Role:           database.OrganizationRoleOwner,
	})
	member, _ := queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         memberID,
		Role:           database.OrganizationRoleMember,
	})

	handler := NewOrganizationHandler(pool, logger)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/organizations/%s/members/%s", org.ID.String(), member.ID.String()), nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/organizations/:id/members/:memberId")
	c.SetParamNames("id", "memberId")
	c.SetParamValues(org.ID.String(), member.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.RemoveOrganizationMember(c)
	})

	err := h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Verify member is removed
	_, err = queries.GetOrganizationMember(context.Background(), member.ID)
	assert.Error(t, err)
}

func TestMain(m *testing.M) {
	// Ensure we have a clean test environment
	os.Exit(m.Run())
}
