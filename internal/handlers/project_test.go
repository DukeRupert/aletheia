package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// Helper function to create an organization with a user as owner
func createTestOrganization(t *testing.T, pool *pgxpool.Pool, userID pgtype.UUID, name string) database.Organization {
	queries := database.New(pool)

	org, err := queries.CreateOrganization(context.Background(), name)
	assert.NoError(t, err)

	_, err = queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         userID,
		Role:           database.OrganizationRoleOwner,
	})
	assert.NoError(t, err)

	return org
}

func TestCreateProject(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "testproject1@example.com")
	org := createTestOrganization(t, pool, userID, "Test Org")

	handler := NewProjectHandler(pool, logger)

	e := echo.New()
	reqBody := fmt.Sprintf(`{"organization_id":"%s","name":"Test Project"}`, org.ID.String())
	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewBufferString(reqBody))
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
		return handler.CreateProject(c)
	})

	err := h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp CreateProjectResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Test Project", resp.Name)
	assert.Equal(t, org.ID.String(), resp.OrganizationID)
	assert.NotEmpty(t, resp.ID)
}

func TestCreateProjectForbiddenForMember(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	ownerID, _ := createTestUserWithSession(t, pool, "testproject2owner@example.com")
	memberID, sessionID := createTestUserWithSession(t, pool, "testproject2member@example.com")

	org := createTestOrganization(t, pool, ownerID, "Test Org")

	// Add second user as member (not admin)
	queries := database.New(pool)
	_, err := queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         memberID,
		Role:           database.OrganizationRoleMember,
	})
	assert.NoError(t, err)

	handler := NewProjectHandler(pool, logger)

	e := echo.New()
	reqBody := fmt.Sprintf(`{"organization_id":"%s","name":"Test Project"}`, org.ID.String())
	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewBufferString(reqBody))
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
		return handler.CreateProject(c)
	})

	err = h(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, httpErr.Code)
}

func TestGetProject(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "testproject3@example.com")
	org := createTestOrganization(t, pool, userID, "Test Org")

	// Create project
	queries := database.New(pool)
	project, err := queries.CreateProject(context.Background(), database.CreateProjectParams{
		OrganizationID: org.ID,
		Name:           "Test Project",
	})
	assert.NoError(t, err)

	handler := NewProjectHandler(pool, logger)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID.String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/projects/:id")
	c.SetParamNames("id")
	c.SetParamValues(project.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.GetProject(c)
	})

	err = h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp GetProjectResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "Test Project", resp.Name)
	assert.Equal(t, project.ID.String(), resp.ID)
}

func TestGetProjectUnauthorized(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	owner1ID, _ := createTestUserWithSession(t, pool, "testproject4owner1@example.com")
	user2ID, sessionID := createTestUserWithSession(t, pool, "testproject4user2@example.com")

	org := createTestOrganization(t, pool, owner1ID, "Test Org")

	// Create project in org1
	queries := database.New(pool)
	project, err := queries.CreateProject(context.Background(), database.CreateProjectParams{
		OrganizationID: org.ID,
		Name:           "Test Project",
	})
	assert.NoError(t, err)

	// user2 is not a member of org1
	handler := NewProjectHandler(pool, logger)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID.String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/projects/:id")
	c.SetParamNames("id")
	c.SetParamValues(project.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.GetProject(c)
	})

	err = h(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, httpErr.Code)

	// Verify user2 can't access the project
	_ = user2ID
}

func TestListProjects(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "testproject5@example.com")
	org := createTestOrganization(t, pool, userID, "Test Org")

	// Create multiple projects
	queries := database.New(pool)
	_, err := queries.CreateProject(context.Background(), database.CreateProjectParams{
		OrganizationID: org.ID,
		Name:           "Project 1",
	})
	assert.NoError(t, err)

	_, err = queries.CreateProject(context.Background(), database.CreateProjectParams{
		OrganizationID: org.ID,
		Name:           "Project 2",
	})
	assert.NoError(t, err)

	handler := NewProjectHandler(pool, logger)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/organizations/"+org.ID.String()+"/projects", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/organizations/:orgId/projects")
	c.SetParamNames("orgId")
	c.SetParamValues(org.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.ListProjects(c)
	})

	err = h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp ListProjectsResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Projects, 2)
}

func TestListProjectsUnauthorized(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	owner1ID, _ := createTestUserWithSession(t, pool, "testproject6owner1@example.com")
	user2ID, sessionID := createTestUserWithSession(t, pool, "testproject6user2@example.com")

	org := createTestOrganization(t, pool, owner1ID, "Test Org")

	// user2 is not a member of org
	handler := NewProjectHandler(pool, logger)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/organizations/"+org.ID.String()+"/projects", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/organizations/:orgId/projects")
	c.SetParamNames("orgId")
	c.SetParamValues(org.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.ListProjects(c)
	})

	err := h(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, httpErr.Code)

	_ = user2ID
}

func TestUpdateProject(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "testproject7@example.com")
	org := createTestOrganization(t, pool, userID, "Test Org")

	// Create project
	queries := database.New(pool)
	project, err := queries.CreateProject(context.Background(), database.CreateProjectParams{
		OrganizationID: org.ID,
		Name:           "Old Name",
	})
	assert.NoError(t, err)

	handler := NewProjectHandler(pool, logger)

	e := echo.New()
	newName := "New Name"
	reqBody := fmt.Sprintf(`{"name":"%s"}`, newName)
	req := httptest.NewRequest(http.MethodPut, "/api/projects/"+project.ID.String(), bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/projects/:id")
	c.SetParamNames("id")
	c.SetParamValues(project.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.UpdateProject(c)
	})

	err = h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp UpdateProjectResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, newName, resp.Name)
}

func TestUpdateProjectForbiddenForMember(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	ownerID, _ := createTestUserWithSession(t, pool, "testproject8owner@example.com")
	memberID, sessionID := createTestUserWithSession(t, pool, "testproject8member@example.com")

	org := createTestOrganization(t, pool, ownerID, "Test Org")

	// Add memberID as member
	queries := database.New(pool)
	_, err := queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         memberID,
		Role:           database.OrganizationRoleMember,
	})
	assert.NoError(t, err)

	// Create project
	project, err := queries.CreateProject(context.Background(), database.CreateProjectParams{
		OrganizationID: org.ID,
		Name:           "Test Project",
	})
	assert.NoError(t, err)

	handler := NewProjectHandler(pool, logger)

	e := echo.New()
	reqBody := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPut, "/api/projects/"+project.ID.String(), bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/projects/:id")
	c.SetParamNames("id")
	c.SetParamValues(project.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.UpdateProject(c)
	})

	err = h(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, httpErr.Code)
}

func TestDeleteProject(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "testproject9@example.com")
	org := createTestOrganization(t, pool, userID, "Test Org")

	// Create project
	queries := database.New(pool)
	project, err := queries.CreateProject(context.Background(), database.CreateProjectParams{
		OrganizationID: org.ID,
		Name:           "To Delete",
	})
	assert.NoError(t, err)

	handler := NewProjectHandler(pool, logger)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/projects/"+project.ID.String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/projects/:id")
	c.SetParamNames("id")
	c.SetParamValues(project.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.DeleteProject(c)
	})

	err = h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Verify project is deleted
	_, err = queries.GetProject(context.Background(), project.ID)
	assert.Error(t, err)
}

func TestDeleteProjectForbiddenForMember(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	ownerID, _ := createTestUserWithSession(t, pool, "testproject10owner@example.com")
	memberID, sessionID := createTestUserWithSession(t, pool, "testproject10member@example.com")

	org := createTestOrganization(t, pool, ownerID, "Test Org")

	// Add memberID as member
	queries := database.New(pool)
	_, err := queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         memberID,
		Role:           database.OrganizationRoleMember,
	})
	assert.NoError(t, err)

	// Create project
	project, err := queries.CreateProject(context.Background(), database.CreateProjectParams{
		OrganizationID: org.ID,
		Name:           "Test Project",
	})
	assert.NoError(t, err)

	handler := NewProjectHandler(pool, logger)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/projects/"+project.ID.String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/projects/:id")
	c.SetParamNames("id")
	c.SetParamValues(project.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.DeleteProject(c)
	})

	err = h(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, httpErr.Code)
}
