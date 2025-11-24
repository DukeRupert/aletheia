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
	"github.com/dukerupert/aletheia/internal/validation"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// Helper function to create a project with a user as member
func createTestProject(t *testing.T, pool *pgxpool.Pool, userID pgtype.UUID, orgName, projName string) database.Project {
	org := createTestOrganization(t, pool, userID, orgName)

	queries := database.New(pool)
	project, err := queries.CreateProject(context.Background(), database.CreateProjectParams{
		OrganizationID: org.ID,
		Name:           projName,
	})
	assert.NoError(t, err)

	return project
}

func TestCreateInspection(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "testinspection1@example.com")
	project := createTestProject(t, pool, userID, "Test Org", "Test Project")

	handler := NewInspectionHandler(pool, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	reqBody := fmt.Sprintf(`{"project_id":"%s"}`, project.ID.String())
	req := httptest.NewRequest(http.MethodPost, "/api/inspections", bytes.NewBufferString(reqBody))
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
		return handler.CreateInspection(c)
	})

	err := h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp CreateInspectionResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, project.ID.String(), resp.ProjectID)
	assert.Equal(t, "draft", resp.Status)
	assert.NotEmpty(t, resp.ID)
}

func TestCreateInspectionUnauthorized(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	owner1ID, _ := createTestUserWithSession(t, pool, "testinspection2owner@example.com")
	user2ID, sessionID := createTestUserWithSession(t, pool, "testinspection2user@example.com")

	project := createTestProject(t, pool, owner1ID, "Test Org", "Test Project")

	// user2 is not a member of the organization
	handler := NewInspectionHandler(pool, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	reqBody := fmt.Sprintf(`{"project_id":"%s"}`, project.ID.String())
	req := httptest.NewRequest(http.MethodPost, "/api/inspections", bytes.NewBufferString(reqBody))
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
		return handler.CreateInspection(c)
	})

	err := h(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, httpErr.Code)

	_ = user2ID
}

func TestGetInspection(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "testinspection3@example.com")
	project := createTestProject(t, pool, userID, "Test Org", "Test Project")

	// Create inspection
	queries := database.New(pool)
	inspection, err := queries.CreateInspection(context.Background(), database.CreateInspectionParams{
		ProjectID:   project.ID,
		InspectorID: userID,
		Status:      database.InspectionStatusDraft,
	})
	assert.NoError(t, err)

	handler := NewInspectionHandler(pool, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	req := httptest.NewRequest(http.MethodGet, "/api/inspections/"+inspection.ID.String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/inspections/:id")
	c.SetParamNames("id")
	c.SetParamValues(inspection.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.GetInspection(c)
	})

	err = h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp GetInspectionResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, inspection.ID.String(), resp.ID)
	assert.Equal(t, "draft", resp.Status)
}

func TestListInspections(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "testinspection4@example.com")
	project := createTestProject(t, pool, userID, "Test Org", "Test Project")

	// Create multiple inspections
	queries := database.New(pool)
	_, err := queries.CreateInspection(context.Background(), database.CreateInspectionParams{
		ProjectID:   project.ID,
		InspectorID: userID,
		Status:      database.InspectionStatusDraft,
	})
	assert.NoError(t, err)

	_, err = queries.CreateInspection(context.Background(), database.CreateInspectionParams{
		ProjectID:   project.ID,
		InspectorID: userID,
		Status:      database.InspectionStatusInProgress,
	})
	assert.NoError(t, err)

	handler := NewInspectionHandler(pool, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	req := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID.String()+"/inspections", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/projects/:projectId/inspections")
	c.SetParamNames("projectId")
	c.SetParamValues(project.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.ListInspections(c)
	})

	err = h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp ListInspectionsResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Inspections, 2)
}

func TestUpdateInspectionStatus(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "testinspection5@example.com")
	project := createTestProject(t, pool, userID, "Test Org", "Test Project")

	// Create inspection
	queries := database.New(pool)
	inspection, err := queries.CreateInspection(context.Background(), database.CreateInspectionParams{
		ProjectID:   project.ID,
		InspectorID: userID,
		Status:      database.InspectionStatusDraft,
	})
	assert.NoError(t, err)

	handler := NewInspectionHandler(pool, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	reqBody := `{"status":"in_progress"}`
	req := httptest.NewRequest(http.MethodPut, "/api/inspections/"+inspection.ID.String()+"/status", bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/inspections/:id/status")
	c.SetParamNames("id")
	c.SetParamValues(inspection.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.UpdateInspectionStatus(c)
	})

	err = h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp UpdateInspectionStatusResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "in_progress", resp.Status)
}

func TestUpdateInspectionStatusForbiddenForNonInspector(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	inspector1ID, _ := createTestUserWithSession(t, pool, "testinspection6inspector@example.com")
	member2ID, sessionID := createTestUserWithSession(t, pool, "testinspection6member@example.com")

	// Create project and add both users to the org
	org := createTestOrganization(t, pool, inspector1ID, "Test Org")

	queries := database.New(pool)
	_, err := queries.AddOrganizationMember(context.Background(), database.AddOrganizationMemberParams{
		OrganizationID: org.ID,
		UserID:         member2ID,
		Role:           database.OrganizationRoleMember,
	})
	assert.NoError(t, err)

	project, err := queries.CreateProject(context.Background(), database.CreateProjectParams{
		OrganizationID: org.ID,
		Name:           "Test Project",
	})
	assert.NoError(t, err)

	// inspector1 creates the inspection
	inspection, err := queries.CreateInspection(context.Background(), database.CreateInspectionParams{
		ProjectID:   project.ID,
		InspectorID: inspector1ID,
		Status:      database.InspectionStatusDraft,
	})
	assert.NoError(t, err)

	// member2 (not the inspector, not owner/admin) tries to update
	handler := NewInspectionHandler(pool, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	reqBody := `{"status":"in_progress"}`
	req := httptest.NewRequest(http.MethodPut, "/api/inspections/"+inspection.ID.String()+"/status", bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/inspections/:id/status")
	c.SetParamNames("id")
	c.SetParamValues(inspection.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.UpdateInspectionStatus(c)
	})

	err = h(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, httpErr.Code)
}
