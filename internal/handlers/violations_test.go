package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dukerupert/aletheia/internal/config"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/dukerupert/aletheia/internal/validation"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"math/big"
)

func setupViolationTest(t *testing.T) (*pgxpool.Pool, *slog.Logger, func()) {
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
		// Cleanup test data - cascading deletes will handle related records
		pool.Exec(context.Background(), "DELETE FROM detected_violations WHERE 1=1")
		pool.Exec(context.Background(), "DELETE FROM photos WHERE 1=1")
		pool.Exec(context.Background(), "DELETE FROM inspections WHERE 1=1")
		pool.Exec(context.Background(), "DELETE FROM projects WHERE 1=1")
		pool.Exec(context.Background(), "DELETE FROM organization_members WHERE 1=1")
		pool.Exec(context.Background(), "DELETE FROM organizations WHERE 1=1")
		pool.Exec(context.Background(), "DELETE FROM sessions WHERE 1=1")
		pool.Exec(context.Background(), "DELETE FROM users WHERE 1=1")
		pool.Close()
	}

	return pool, logger, cleanup
}

func createTestViolation(t *testing.T, pool *pgxpool.Pool, photoID pgtype.UUID, severity database.ViolationSeverity, status database.ViolationStatus) database.DetectedViolation {
	queries := database.New(pool)

	confidenceInt := new(big.Int).SetInt64(8500) // 0.85 confidence
	violation, err := queries.CreateDetectedViolation(context.Background(), database.CreateDetectedViolationParams{
		PhotoID:         photoID,
		Description:     "Test violation - worker without hard hat",
		ConfidenceScore: pgtype.Numeric{Int: confidenceInt, Exp: -4, Valid: true},
		Status:          status,
		Severity:        severity,
		Location:        pgtype.Text{String: "center of image", Valid: true},
	})
	require.NoError(t, err)

	return violation
}

func TestListViolationsByInspection(t *testing.T) {
	pool, logger, cleanup := setupViolationTest(t)
	defer cleanup()

	// Create test data
	userID, sessionID := createTestUserWithSession(t, pool, "violation1@example.com")
	org := createTestOrganization(t, pool, userID, "Test Org")

	queries := database.New(pool)
	project, err := queries.CreateProject(context.Background(), database.CreateProjectParams{
		OrganizationID: org.ID,
		Name:           "Test Project",
	})
	require.NoError(t, err)

	inspection, err := queries.CreateInspection(context.Background(), database.CreateInspectionParams{
		ProjectID:   project.ID,
		InspectorID: userID,
		Status:      database.InspectionStatusInProgress,
	})
	require.NoError(t, err)

	photo := createTestPhoto(t, pool, inspection.ID)

	// Create multiple violations
	violation1 := createTestViolation(t, pool, photo.ID, database.ViolationSeverityCritical, database.ViolationStatusPending)
	_ = createTestViolation(t, pool, photo.ID, database.ViolationSeverityHigh, database.ViolationStatusConfirmed)
	_ = createTestViolation(t, pool, photo.ID, database.ViolationSeverityMedium, database.ViolationStatusPending)

	handler := NewViolationHandler(pool, queries, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	req := httptest.NewRequest(http.MethodGet, "/api/inspections/"+uuid.UUID(inspection.ID.Bytes).String()+"/violations", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("inspection_id")
	c.SetParamValues(uuid.UUID(inspection.ID.Bytes).String())

	err = handler.ListViolationsByInspection(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var violations []ViolationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &violations)
	require.NoError(t, err)
	assert.Len(t, violations, 3)

	// Verify violations (ordered by created_at DESC, so most recent first)
	// Last created was medium, then high, then critical
	assert.Equal(t, "medium", violations[0].Severity)
	assert.Equal(t, "high", violations[1].Severity)
	assert.Equal(t, uuid.UUID(violation1.ID.Bytes).String(), violations[2].ID)
	assert.Equal(t, "critical", violations[2].Severity)
	assert.Equal(t, "pending", violations[2].Status)
	assert.Equal(t, 0.85, violations[2].ConfidenceScore)
	assert.NotNil(t, violations[2].Location)
	assert.Equal(t, "center of image", *violations[2].Location)
}

func TestListViolationsByInspection_WithStatusFilter(t *testing.T) {
	pool, logger, cleanup := setupViolationTest(t)
	defer cleanup()

	// Create test data
	userID, sessionID := createTestUserWithSession(t, pool, "violation2@example.com")
	org := createTestOrganization(t, pool, userID, "Test Org")

	queries := database.New(pool)
	project, err := queries.CreateProject(context.Background(), database.CreateProjectParams{
		OrganizationID: org.ID,
		Name:           "Test Project",
	})
	require.NoError(t, err)

	inspection, err := queries.CreateInspection(context.Background(), database.CreateInspectionParams{
		ProjectID:   project.ID,
		InspectorID: userID,
		Status:      database.InspectionStatusInProgress,
	})
	require.NoError(t, err)

	photo := createTestPhoto(t, pool, inspection.ID)

	// Create violations with different statuses
	_ = createTestViolation(t, pool, photo.ID, database.ViolationSeverityCritical, database.ViolationStatusPending)
	confirmedViolation := createTestViolation(t, pool, photo.ID, database.ViolationSeverityHigh, database.ViolationStatusConfirmed)

	handler := NewViolationHandler(pool, queries, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	req := httptest.NewRequest(http.MethodGet, "/api/inspections/"+uuid.UUID(inspection.ID.Bytes).String()+"/violations?status=confirmed", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("inspection_id")
	c.SetParamValues(uuid.UUID(inspection.ID.Bytes).String())

	err = handler.ListViolationsByInspection(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var violations []ViolationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &violations)
	require.NoError(t, err)
	assert.Len(t, violations, 1)
	assert.Equal(t, uuid.UUID(confirmedViolation.ID.Bytes).String(), violations[0].ID)
	assert.Equal(t, "confirmed", violations[0].Status)
}

func TestGetViolation(t *testing.T) {
	pool, logger, cleanup := setupViolationTest(t)
	defer cleanup()

	// Create test data
	userID, sessionID := createTestUserWithSession(t, pool, "violation3@example.com")
	org := createTestOrganization(t, pool, userID, "Test Org")

	queries := database.New(pool)
	project, err := queries.CreateProject(context.Background(), database.CreateProjectParams{
		OrganizationID: org.ID,
		Name:           "Test Project",
	})
	require.NoError(t, err)

	inspection, err := queries.CreateInspection(context.Background(), database.CreateInspectionParams{
		ProjectID:   project.ID,
		InspectorID: userID,
		Status:      database.InspectionStatusInProgress,
	})
	require.NoError(t, err)

	photo := createTestPhoto(t, pool, inspection.ID)
	violation := createTestViolation(t, pool, photo.ID, database.ViolationSeverityHigh, database.ViolationStatusPending)

	handler := NewViolationHandler(pool, queries, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	req := httptest.NewRequest(http.MethodGet, "/api/violations/"+uuid.UUID(violation.ID.Bytes).String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("violation_id")
	c.SetParamValues(uuid.UUID(violation.ID.Bytes).String())

	err = handler.GetViolation(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp ViolationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, uuid.UUID(violation.ID.Bytes).String(), resp.ID)
	assert.Equal(t, "high", resp.Severity)
	assert.Equal(t, "pending", resp.Status)
	assert.Equal(t, "Test violation - worker without hard hat", resp.Description)
}

func TestGetViolation_NotFound(t *testing.T) {
	pool, logger, cleanup := setupViolationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "violation4@example.com")
	_ = createTestOrganization(t, pool, userID, "Test Org")

	queries := database.New(pool)
	handler := NewViolationHandler(pool, queries, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	nonExistentID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/violations/"+nonExistentID.String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("violation_id")
	c.SetParamValues(nonExistentID.String())

	err := handler.GetViolation(c)
	assert.Error(t, err)

	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, httpErr.Code)
}

func TestUpdateViolation(t *testing.T) {
	pool, logger, cleanup := setupViolationTest(t)
	defer cleanup()

	// Create test data
	userID, sessionID := createTestUserWithSession(t, pool, "violation5@example.com")
	org := createTestOrganization(t, pool, userID, "Test Org")

	queries := database.New(pool)
	project, err := queries.CreateProject(context.Background(), database.CreateProjectParams{
		OrganizationID: org.ID,
		Name:           "Test Project",
	})
	require.NoError(t, err)

	inspection, err := queries.CreateInspection(context.Background(), database.CreateInspectionParams{
		ProjectID:   project.ID,
		InspectorID: userID,
		Status:      database.InspectionStatusInProgress,
	})
	require.NoError(t, err)

	photo := createTestPhoto(t, pool, inspection.ID)
	violation := createTestViolation(t, pool, photo.ID, database.ViolationSeverityHigh, database.ViolationStatusPending)

	handler := NewViolationHandler(pool, queries, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	newStatus := "confirmed"
	newDescription := "Confirmed violation - inspector verified missing hard hat"
	reqBody := fmt.Sprintf(`{"status":"%s","description":"%s"}`, newStatus, newDescription)
	req := httptest.NewRequest(http.MethodPatch, "/api/violations/"+uuid.UUID(violation.ID.Bytes).String(), bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("violation_id")
	c.SetParamValues(uuid.UUID(violation.ID.Bytes).String())

	err = handler.UpdateViolation(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp ViolationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "confirmed", resp.Status)
	assert.Equal(t, newDescription, resp.Description)
}

func TestUpdateViolation_StatusOnly(t *testing.T) {
	pool, logger, cleanup := setupViolationTest(t)
	defer cleanup()

	// Create test data
	userID, sessionID := createTestUserWithSession(t, pool, "violation6@example.com")
	org := createTestOrganization(t, pool, userID, "Test Org")

	queries := database.New(pool)
	project, err := queries.CreateProject(context.Background(), database.CreateProjectParams{
		OrganizationID: org.ID,
		Name:           "Test Project",
	})
	require.NoError(t, err)

	inspection, err := queries.CreateInspection(context.Background(), database.CreateInspectionParams{
		ProjectID:   project.ID,
		InspectorID: userID,
		Status:      database.InspectionStatusInProgress,
	})
	require.NoError(t, err)

	photo := createTestPhoto(t, pool, inspection.ID)
	violation := createTestViolation(t, pool, photo.ID, database.ViolationSeverityHigh, database.ViolationStatusPending)
	originalDescription := violation.Description

	handler := NewViolationHandler(pool, queries, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	reqBody := `{"status":"dismissed"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/violations/"+uuid.UUID(violation.ID.Bytes).String(), bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("violation_id")
	c.SetParamValues(uuid.UUID(violation.ID.Bytes).String())

	err = handler.UpdateViolation(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp ViolationResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "dismissed", resp.Status)
	assert.Equal(t, originalDescription, resp.Description) // Description should remain unchanged
}

func TestDeleteViolation(t *testing.T) {
	pool, logger, cleanup := setupViolationTest(t)
	defer cleanup()

	// Create test data
	userID, sessionID := createTestUserWithSession(t, pool, "violation7@example.com")
	org := createTestOrganization(t, pool, userID, "Test Org")

	queries := database.New(pool)
	project, err := queries.CreateProject(context.Background(), database.CreateProjectParams{
		OrganizationID: org.ID,
		Name:           "Test Project",
	})
	require.NoError(t, err)

	inspection, err := queries.CreateInspection(context.Background(), database.CreateInspectionParams{
		ProjectID:   project.ID,
		InspectorID: userID,
		Status:      database.InspectionStatusInProgress,
	})
	require.NoError(t, err)

	photo := createTestPhoto(t, pool, inspection.ID)
	violation := createTestViolation(t, pool, photo.ID, database.ViolationSeverityLow, database.ViolationStatusPending)

	handler := NewViolationHandler(pool, queries, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	req := httptest.NewRequest(http.MethodDelete, "/api/violations/"+uuid.UUID(violation.ID.Bytes).String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("violation_id")
	c.SetParamValues(uuid.UUID(violation.ID.Bytes).String())

	err = handler.DeleteViolation(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Verify violation was deleted
	_, err = queries.GetDetectedViolation(context.Background(), violation.ID)
	assert.Error(t, err) // Should not be found
}

func TestDeleteViolation_NotFound(t *testing.T) {
	pool, logger, cleanup := setupViolationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "violation8@example.com")
	_ = createTestOrganization(t, pool, userID, "Test Org")

	queries := database.New(pool)
	handler := NewViolationHandler(pool, queries, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	nonExistentID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/violations/"+nonExistentID.String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("violation_id")
	c.SetParamValues(nonExistentID.String())

	err := handler.DeleteViolation(c)
	// DELETE is idempotent - succeeds even if violation doesn't exist
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}
