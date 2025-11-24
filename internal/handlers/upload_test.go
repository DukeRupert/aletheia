package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/dukerupert/aletheia/internal/storage"
	"github.com/dukerupert/aletheia/internal/validation"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestUploadPhoto(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "testphoto1@example.com")
	project := createTestProject(t, pool, userID, "Test Org", "Test Project")

	// Create inspection
	queries := database.New(pool)
	inspection, err := queries.CreateInspection(context.Background(), database.CreateInspectionParams{
		ProjectID:   project.ID,
		InspectorID: userID,
		Status:      database.InspectionStatusDraft,
	})
	assert.NoError(t, err)

	// Create temporary uploads directory
	tmpDir := t.TempDir()
	fileStorage, err := storage.NewLocalStorage(tmpDir, "http://localhost:1323/uploads")
	assert.NoError(t, err)

	handler := NewUploadHandler(fileStorage, pool, logger)

	// Create a test image file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add inspection_id field
	err = writer.WriteField("inspection_id", inspection.ID.String())
	assert.NoError(t, err)

	// Add image file with proper Content-Type header
	part, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="image"; filename="test.jpg"`},
		"Content-Type":        {"image/jpeg"},
	})
	assert.NoError(t, err)

	// Write a minimal JPEG header to make it a valid image
	jpegHeader := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
	_, err = part.Write(jpegHeader)
	assert.NoError(t, err)

	err = writer.Close()
	assert.NoError(t, err)

	e := echo.New()
	e.Validator = validation.NewValidator()
	req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.UploadImage(c)
	})

	err = h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp UploadPhotoResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, inspection.ID.String(), resp.InspectionID)
	assert.NotEmpty(t, resp.ID)
	assert.NotEmpty(t, resp.StorageURL)
}

func TestUploadPhotoUnauthorized(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	owner1ID, _ := createTestUserWithSession(t, pool, "testphoto2owner@example.com")
	user2ID, sessionID := createTestUserWithSession(t, pool, "testphoto2user@example.com")

	project := createTestProject(t, pool, owner1ID, "Test Org", "Test Project")

	// Create inspection
	queries := database.New(pool)
	inspection, err := queries.CreateInspection(context.Background(), database.CreateInspectionParams{
		ProjectID:   project.ID,
		InspectorID: owner1ID,
		Status:      database.InspectionStatusDraft,
	})
	assert.NoError(t, err)

	// Create temporary uploads directory
	tmpDir := t.TempDir()
	fileStorage, err := storage.NewLocalStorage(tmpDir, "http://localhost:1323/uploads")
	assert.NoError(t, err)

	handler := NewUploadHandler(fileStorage, pool, logger)

	// Create a test image file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add inspection_id field
	err = writer.WriteField("inspection_id", inspection.ID.String())
	assert.NoError(t, err)

	// Add image file with proper Content-Type header
	part, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": {`form-data; name="image"; filename="test.jpg"`},
		"Content-Type":        {"image/jpeg"},
	})
	assert.NoError(t, err)

	// Write a minimal JPEG header
	jpegHeader := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
	_, err = part.Write(jpegHeader)
	assert.NoError(t, err)

	err = writer.Close()
	assert.NoError(t, err)

	e := echo.New()
	e.Validator = validation.NewValidator()
	req := httptest.NewRequest(http.MethodPost, "/api/upload", body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.UploadImage(c)
	})

	err = h(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, httpErr.Code)

	_ = user2ID
}

func TestListPhotos(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "testphoto3@example.com")
	project := createTestProject(t, pool, userID, "Test Org", "Test Project")

	// Create inspection
	queries := database.New(pool)
	inspection, err := queries.CreateInspection(context.Background(), database.CreateInspectionParams{
		ProjectID:   project.ID,
		InspectorID: userID,
		Status:      database.InspectionStatusDraft,
	})
	assert.NoError(t, err)

	// Create some photos
	_, err = queries.CreatePhoto(context.Background(), database.CreatePhotoParams{
		InspectionID: inspection.ID,
		StorageUrl:   "http://example.com/photo1.jpg",
	})
	assert.NoError(t, err)

	_, err = queries.CreatePhoto(context.Background(), database.CreatePhotoParams{
		InspectionID: inspection.ID,
		StorageUrl:   "http://example.com/photo2.jpg",
	})
	assert.NoError(t, err)

	// Create temporary uploads directory
	tmpDir := t.TempDir()
	fileStorage, err := storage.NewLocalStorage(tmpDir, "http://localhost:1323/uploads")
	assert.NoError(t, err)

	handler := NewUploadHandler(fileStorage, pool, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	req := httptest.NewRequest(http.MethodGet, "/api/inspections/"+inspection.ID.String()+"/photos", nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/inspections/:inspectionId/photos")
	c.SetParamNames("inspectionId")
	c.SetParamValues(inspection.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.ListPhotos(c)
	})

	err = h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp ListPhotosResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Photos, 2)
}

func TestGetPhoto(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	userID, sessionID := createTestUserWithSession(t, pool, "testphoto4@example.com")
	project := createTestProject(t, pool, userID, "Test Org", "Test Project")

	// Create inspection
	queries := database.New(pool)
	inspection, err := queries.CreateInspection(context.Background(), database.CreateInspectionParams{
		ProjectID:   project.ID,
		InspectorID: userID,
		Status:      database.InspectionStatusDraft,
	})
	assert.NoError(t, err)

	// Create a photo
	photo, err := queries.CreatePhoto(context.Background(), database.CreatePhotoParams{
		InspectionID: inspection.ID,
		StorageUrl:   "http://example.com/photo1.jpg",
	})
	assert.NoError(t, err)

	// Create temporary uploads directory
	tmpDir := t.TempDir()
	fileStorage, err := storage.NewLocalStorage(tmpDir, "http://localhost:1323/uploads")
	assert.NoError(t, err)

	handler := NewUploadHandler(fileStorage, pool, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	req := httptest.NewRequest(http.MethodGet, "/api/photos/"+photo.ID.String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/photos/:id")
	c.SetParamNames("id")
	c.SetParamValues(photo.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.GetPhoto(c)
	})

	err = h(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp GetPhotoResponse
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, photo.ID.String(), resp.ID)
	assert.Equal(t, inspection.ID.String(), resp.InspectionID)
	assert.Equal(t, "http://example.com/photo1.jpg", resp.StorageURL)
}

func TestGetPhotoUnauthorized(t *testing.T) {
	pool, logger, cleanup := setupOrganizationTest(t)
	defer cleanup()

	owner1ID, _ := createTestUserWithSession(t, pool, "testphoto5owner@example.com")
	user2ID, sessionID := createTestUserWithSession(t, pool, "testphoto5user@example.com")

	project := createTestProject(t, pool, owner1ID, "Test Org", "Test Project")

	// Create inspection
	queries := database.New(pool)
	inspection, err := queries.CreateInspection(context.Background(), database.CreateInspectionParams{
		ProjectID:   project.ID,
		InspectorID: owner1ID,
		Status:      database.InspectionStatusDraft,
	})
	assert.NoError(t, err)

	// Create a photo
	photo, err := queries.CreatePhoto(context.Background(), database.CreatePhotoParams{
		InspectionID: inspection.ID,
		StorageUrl:   "http://example.com/photo1.jpg",
	})
	assert.NoError(t, err)

	// Create temporary uploads directory
	tmpDir := t.TempDir()
	fileStorage, err := storage.NewLocalStorage(tmpDir, "http://localhost:1323/uploads")
	assert.NoError(t, err)

	handler := NewUploadHandler(fileStorage, pool, logger)

	e := echo.New()
	e.Validator = validation.NewValidator()
	req := httptest.NewRequest(http.MethodGet, "/api/photos/"+photo.ID.String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  session.SessionCookieName,
		Value: sessionID,
	})

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/photos/:id")
	c.SetParamNames("id")
	c.SetParamValues(photo.ID.String())

	// Add session middleware
	middleware := session.SessionMiddleware(pool)
	h := middleware(func(c echo.Context) error {
		return handler.GetPhoto(c)
	})

	err = h(c)
	assert.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusForbidden, httpErr.Code)

	_ = user2ID
}

func init() {
	// Suppress logger output during tests
	os.Setenv("LOG_LEVEL", "error")
}
