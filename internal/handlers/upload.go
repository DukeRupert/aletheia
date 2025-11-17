package handlers

import (
	"net/http"

	"github.com/dukerupert/aletheia/internal/storage"
	"github.com/labstack/echo/v4"
)

type UploadHandler struct {
	storage storage.FileStorage
}

func NewUploadHandler(storage storage.FileStorage) *UploadHandler {
	return &UploadHandler{storage: storage}
}

// UploadImage handles image upload
func (h *UploadHandler) UploadImage(c echo.Context) error {
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
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save file")
	}

	// Get public URL
	url := h.storage.GetURL(filename)

	return c.JSON(http.StatusOK, map[string]string{
		"filename": filename,
		"url":      url,
	})
}
