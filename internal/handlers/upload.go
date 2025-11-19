package handlers

import (
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/dukerupert/aletheia/internal/database"
	"github.com/dukerupert/aletheia/internal/session"
	"github.com/dukerupert/aletheia/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type UploadHandler struct {
	storage storage.FileStorage
	pool    *pgxpool.Pool
	logger  *slog.Logger
}

func NewUploadHandler(storage storage.FileStorage, pool *pgxpool.Pool, logger *slog.Logger) *UploadHandler {
	return &UploadHandler{
		storage: storage,
		pool:    pool,
		logger:  logger,
	}
}

// UploadPhotoResponse is the response payload for photo upload
type UploadPhotoResponse struct {
	ID           string `json:"id"`
	InspectionID string `json:"inspection_id"`
	StorageURL   string `json:"storage_url"`
	CreatedAt    string `json:"created_at"`
}

// UploadImage handles image upload and associates it with an inspection
func (h *UploadHandler) UploadImage(c echo.Context) error {
	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	// Get inspection_id from form
	inspectionID := c.FormValue("inspection_id")
	if inspectionID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "inspection_id is required")
	}

	queries := database.New(h.pool)

	// Parse inspection ID
	inspectionUUID, err := parseUUID(inspectionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid inspection_id")
	}

	// Get inspection to verify access
	inspection, err := queries.GetInspection(c.Request().Context(), inspectionUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "inspection not found")
	}

	// Get project to find its organization
	project, err := queries.GetProject(c.Request().Context(), inspection.ProjectID)
	if err != nil {
		h.logger.Error("failed to get project for inspection", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to verify access")
	}

	// Verify user is a member of the organization that owns this inspection
	_, err = queries.GetOrganizationMemberByUserAndOrg(c.Request().Context(), database.GetOrganizationMemberByUserAndOrgParams{
		OrganizationID: project.OrganizationID,
		UserID:         uuidToPgUUID(userID),
	})
	if err != nil {
		h.logger.Warn("user not authorized to upload photo to inspection",
			slog.String("user_id", userID.String()),
			slog.String("inspection_id", inspectionID))
		return echo.NewHTTPError(http.StatusForbidden, "you are not a member of this inspection's organization")
	}

	// Get file from form
	file, err := c.FormFile("image")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "no file uploaded")
	}

	// Validate file size (e.g., 5MB max)
	if file.Size > 5*1024*1024 {
		return echo.NewHTTPError(http.StatusBadRequest, "file too large (max 5MB)")
	}

	// Validate file type
	contentType := file.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file type (only JPEG, PNG, WebP allowed)")
	}

	// Save file using storage interface
	filename, err := h.storage.Save(c.Request().Context(), file)
	if err != nil {
		h.logger.Error("failed to save file", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save file")
	}

	// Get public URL
	url := h.storage.GetURL(filename)

	// Create photo record in database
	photo, err := queries.CreatePhoto(c.Request().Context(), database.CreatePhotoParams{
		InspectionID: inspectionUUID,
		StorageUrl:   url,
	})
	if err != nil {
		h.logger.Error("failed to create photo record", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create photo record")
	}

	h.logger.Info("photo uploaded",
		slog.String("photo_id", photo.ID.String()),
		slog.String("inspection_id", inspectionID),
		slog.String("user_id", userID.String()))

	return c.JSON(http.StatusCreated, UploadPhotoResponse{
		ID:           photo.ID.String(),
		InspectionID: photo.InspectionID.String(),
		StorageURL:   photo.StorageUrl,
		CreatedAt:    photo.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// PhotoSummary represents a photo in the list
type PhotoSummary struct {
	ID           string `json:"id"`
	InspectionID string `json:"inspection_id"`
	StorageURL   string `json:"storage_url"`
	CreatedAt    string `json:"created_at"`
}

// ListPhotosResponse is the response payload for listing photos
type ListPhotosResponse struct {
	Photos []PhotoSummary `json:"photos"`
}

// ListPhotos lists all photos for an inspection
func (h *UploadHandler) ListPhotos(c echo.Context) error {
	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	inspectionID := c.Param("inspectionId")
	if inspectionID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "inspection id is required")
	}

	queries := database.New(h.pool)

	// Parse inspection ID
	inspectionUUID, err := parseUUID(inspectionID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid inspection id")
	}

	// Get inspection to verify access
	inspection, err := queries.GetInspection(c.Request().Context(), inspectionUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "inspection not found")
	}

	// Get project to find its organization
	project, err := queries.GetProject(c.Request().Context(), inspection.ProjectID)
	if err != nil {
		h.logger.Error("failed to get project for inspection", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to verify access")
	}

	// Verify user is a member of the organization
	_, err = queries.GetOrganizationMemberByUserAndOrg(c.Request().Context(), database.GetOrganizationMemberByUserAndOrgParams{
		OrganizationID: project.OrganizationID,
		UserID:         uuidToPgUUID(userID),
	})
	if err != nil {
		h.logger.Warn("user not authorized to list photos for inspection",
			slog.String("user_id", userID.String()),
			slog.String("inspection_id", inspectionID))
		return echo.NewHTTPError(http.StatusForbidden, "you are not a member of this inspection's organization")
	}

	// Get all photos for the inspection
	photos, err := queries.ListPhotos(c.Request().Context(), inspectionUUID)
	if err != nil {
		h.logger.Error("failed to list photos", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list photos")
	}

	photoSummaries := make([]PhotoSummary, len(photos))
	for i, photo := range photos {
		photoSummaries[i] = PhotoSummary{
			ID:           photo.ID.String(),
			InspectionID: photo.InspectionID.String(),
			StorageURL:   photo.StorageUrl,
			CreatedAt:    photo.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return c.JSON(http.StatusOK, ListPhotosResponse{
		Photos: photoSummaries,
	})
}

// GetPhotoResponse is the response payload for photo retrieval
type GetPhotoResponse struct {
	ID           string `json:"id"`
	InspectionID string `json:"inspection_id"`
	StorageURL   string `json:"storage_url"`
	CreatedAt    string `json:"created_at"`
}

// GetPhoto retrieves a photo by ID
func (h *UploadHandler) GetPhoto(c echo.Context) error {
	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	photoID := c.Param("id")
	if photoID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "photo id is required")
	}

	queries := database.New(h.pool)

	// Parse photo ID
	photoUUID, err := parseUUID(photoID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid photo id")
	}

	// Get photo
	photo, err := queries.GetPhoto(c.Request().Context(), photoUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "photo not found")
	}

	// Get inspection to verify access
	inspection, err := queries.GetInspection(c.Request().Context(), photo.InspectionID)
	if err != nil {
		h.logger.Error("failed to get inspection for photo", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to verify access")
	}

	// Get project to find its organization
	project, err := queries.GetProject(c.Request().Context(), inspection.ProjectID)
	if err != nil {
		h.logger.Error("failed to get project for inspection", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to verify access")
	}

	// Verify user is a member of the organization
	_, err = queries.GetOrganizationMemberByUserAndOrg(c.Request().Context(), database.GetOrganizationMemberByUserAndOrgParams{
		OrganizationID: project.OrganizationID,
		UserID:         uuidToPgUUID(userID),
	})
	if err != nil {
		h.logger.Warn("user not authorized to access photo",
			slog.String("user_id", userID.String()),
			slog.String("photo_id", photoID))
		return echo.NewHTTPError(http.StatusForbidden, "you are not a member of this photo's organization")
	}

	return c.JSON(http.StatusOK, GetPhotoResponse{
		ID:           photo.ID.String(),
		InspectionID: photo.InspectionID.String(),
		StorageURL:   photo.StorageUrl,
		CreatedAt:    photo.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// DeletePhoto deletes a photo by ID
func (h *UploadHandler) DeletePhoto(c echo.Context) error {
	// Get authenticated user from session
	userID, ok := session.GetUserID(c)
	if !ok {
		h.logger.Error("failed to get user from session")
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	photoID := c.Param("id")
	if photoID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "photo id is required")
	}

	queries := database.New(h.pool)

	// Parse photo ID
	photoUUID, err := parseUUID(photoID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid photo id")
	}

	// Get photo to verify access and get storage URL
	photo, err := queries.GetPhoto(c.Request().Context(), photoUUID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "photo not found")
	}

	// Get inspection to verify access
	inspection, err := queries.GetInspection(c.Request().Context(), photo.InspectionID)
	if err != nil {
		h.logger.Error("failed to get inspection for photo", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to verify access")
	}

	// Get project to find its organization
	project, err := queries.GetProject(c.Request().Context(), inspection.ProjectID)
	if err != nil {
		h.logger.Error("failed to get project for inspection", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to verify access")
	}

	// Verify user is a member of the organization
	_, err = queries.GetOrganizationMemberByUserAndOrg(c.Request().Context(), database.GetOrganizationMemberByUserAndOrgParams{
		OrganizationID: project.OrganizationID,
		UserID:         uuidToPgUUID(userID),
	})
	if err != nil {
		h.logger.Warn("user not authorized to delete photo",
			slog.String("user_id", userID.String()),
			slog.String("photo_id", photoID))
		return echo.NewHTTPError(http.StatusForbidden, "you are not a member of this photo's organization")
	}

	// Extract filename from storage URL
	// The URL is like "http://localhost:1323/uploads/filename.jpg" or "https://cdn.example.com/filename.jpg"
	filename := filepath.Base(photo.StorageUrl)

	// Delete file from storage
	if err := h.storage.Delete(c.Request().Context(), filename); err != nil {
		h.logger.Error("failed to delete file from storage",
			slog.String("err", err.Error()),
			slog.String("photo_id", photoID),
			slog.String("filename", filename))
		// Continue with database deletion even if storage deletion fails
		// This prevents orphaned database records
	}

	// Delete photo record from database
	if err := queries.DeletePhoto(c.Request().Context(), photoUUID); err != nil {
		h.logger.Error("failed to delete photo from database", slog.String("err", err.Error()))
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete photo")
	}

	h.logger.Info("photo deleted",
		slog.String("photo_id", photoID),
		slog.String("user_id", userID.String()))

	return c.NoContent(http.StatusNoContent)
}
