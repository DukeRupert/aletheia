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
	"github.com/dukerupert/aletheia/internal/queue"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/dukerupert/aletheia/internal/validation"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
)

func setupPhotoTest(t *testing.T) (*pgxpool.Pool, *slog.Logger, func()) {
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

func createTestPhoto(t *testing.T, pool *pgxpool.Pool, inspectionID pgtype.UUID) database.Photo {
	queries := database.New(pool)

	photo, err := queries.CreatePhoto(context.Background(), database.CreatePhotoParams{
		InspectionID: inspectionID,
		StorageUrl:   "https://example.com/test-photo.jpg",
	})
	require.NoError(t, err)

	return photo
}

func TestAnalyzePhoto(t *testing.T) {
	pool, logger, cleanup := setupPhotoTest(t)
	defer cleanup()

	// Create test data
	userID, sessionID := createTestUserWithSession(t, pool, "photo1@example.com")
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
		Status:      database.InspectionStatusDraft,
	})
	require.NoError(t, err)

	photo := createTestPhoto(t, pool, inspection.ID)

	// Create handler with mock queue
	mockQueue := queue.NewMockQueue()
	handler := NewPhotoHandler(pool, queries, mockQueue, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	reqBody := fmt.Sprintf(`{"photo_id":"%s"}`, uuid.UUID(photo.ID.Bytes).String())
	req := httptest.NewRequest(http.MethodPost, "/api/photos/analyze", bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Call handler
	err = handler.AnalyzePhoto(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, rec.Code)

	var resp AnalyzePhotoResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, uuid.UUID(photo.ID.Bytes).String(), resp.PhotoID)
	assert.Equal(t, "queued", resp.Status)
	assert.NotEmpty(t, resp.JobID)
	assert.NotEmpty(t, resp.Message)

	// Verify job was enqueued
	jobID, err := uuid.Parse(resp.JobID)
	require.NoError(t, err)

	job, err := mockQueue.GetJob(context.Background(), jobID)
	require.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, "photo_analysis", job.QueueName)
	assert.Equal(t, "analyze_photo", job.JobType)
	assert.Equal(t, queue.JobStatusPending, job.Status)
}

func TestAnalyzePhoto_InvalidPhotoID(t *testing.T) {
	pool, logger, cleanup := setupPhotoTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "photo2@example.com")
	_ = createTestOrganization(t, pool, userID, "Test Org")

	queries := database.New(pool)
	mockQueue := queue.NewMockQueue()
	handler := NewPhotoHandler(pool, queries, mockQueue, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	reqBody := `{"photo_id":"invalid-uuid"}`
	req := httptest.NewRequest(http.MethodPost, "/api/photos/analyze", bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.AnalyzePhoto(c)
	assert.Error(t, err)

	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, httpErr.Code)
}

func TestAnalyzePhoto_PhotoNotFound(t *testing.T) {
	pool, logger, cleanup := setupPhotoTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "photo3@example.com")
	_ = createTestOrganization(t, pool, userID, "Test Org")

	queries := database.New(pool)
	mockQueue := queue.NewMockQueue()
	handler := NewPhotoHandler(pool, queries, mockQueue, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	nonExistentID := uuid.New()
	reqBody := fmt.Sprintf(`{"photo_id":"%s"}`, nonExistentID.String())
	req := httptest.NewRequest(http.MethodPost, "/api/photos/analyze", bytes.NewBufferString(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.AnalyzePhoto(c)
	assert.Error(t, err)

	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, httpErr.Code)
}

func TestGetPhotoAnalysisStatus(t *testing.T) {
	pool, logger, cleanup := setupPhotoTest(t)
	defer cleanup()

	// Create test data
	userID, sessionID := createTestUserWithSession(t, pool, "photo4@example.com")
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
		Status:      database.InspectionStatusDraft,
	})
	require.NoError(t, err)

	photo := createTestPhoto(t, pool, inspection.ID)

	// Create mock queue and enqueue a job
	mockQueue := queue.NewMockQueue()
	job, err := mockQueue.Enqueue(
		context.Background(),
		"photo_analysis",
		"analyze_photo",
		uuid.UUID(org.ID.Bytes),
		map[string]interface{}{
			"photo_id": uuid.UUID(photo.ID.Bytes).String(),
		},
		nil,
	)
	require.NoError(t, err)

	handler := NewPhotoHandler(pool, queries, mockQueue, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	req := httptest.NewRequest(http.MethodGet, "/api/photos/analyze/"+job.ID.String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("job_id")
	c.SetParamValues(job.ID.String())

	err = handler.GetPhotoAnalysisStatus(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var respJob queue.Job
	err = json.Unmarshal(rec.Body.Bytes(), &respJob)
	require.NoError(t, err)
	assert.Equal(t, job.ID, respJob.ID)
	assert.Equal(t, "photo_analysis", respJob.QueueName)
	assert.Equal(t, queue.JobStatusPending, respJob.Status)
}

func TestGetPhotoAnalysisStatus_InvalidJobID(t *testing.T) {
	pool, logger, cleanup := setupPhotoTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "photo5@example.com")
	_ = createTestOrganization(t, pool, userID, "Test Org")

	queries := database.New(pool)
	mockQueue := queue.NewMockQueue()
	handler := NewPhotoHandler(pool, queries, mockQueue, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	req := httptest.NewRequest(http.MethodGet, "/api/photos/analyze/invalid-uuid", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("job_id")
	c.SetParamValues("invalid-uuid")

	err := handler.GetPhotoAnalysisStatus(c)
	assert.Error(t, err)

	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, httpErr.Code)
}

func TestGetPhotoAnalysisStatus_JobNotFound(t *testing.T) {
	pool, logger, cleanup := setupPhotoTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "photo6@example.com")
	_ = createTestOrganization(t, pool, userID, "Test Org")

	queries := database.New(pool)
	mockQueue := queue.NewMockQueue()
	handler := NewPhotoHandler(pool, queries, mockQueue, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	nonExistentJobID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/photos/analyze/"+nonExistentJobID.String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("job_id")
	c.SetParamValues(nonExistentJobID.String())

	err := handler.GetPhotoAnalysisStatus(c)
	assert.Error(t, err)

	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, httpErr.Code)
}
