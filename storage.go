package aletheia

import (
	"context"
	"io"
)

// FileStorage defines operations for file storage.
type FileStorage interface {
	// Upload uploads a file and returns its URL.
	// The key is the storage path/identifier for the file.
	// The contentType should be a valid MIME type (e.g., "image/jpeg").
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) (url string, err error)

	// Delete removes a file from storage.
	// Returns nil if the file doesn't exist.
	Delete(ctx context.Context, key string) error

	// GetURL returns the public URL for a stored file.
	GetURL(key string) string

	// Exists checks if a file exists in storage.
	Exists(ctx context.Context, key string) (bool, error)
}

// StorageConfig holds configuration for file storage.
type StorageConfig struct {
	// Provider is the storage provider ("local" or "s3").
	Provider string

	// Local storage configuration
	LocalPath string
	LocalURL  string

	// S3 storage configuration
	S3Bucket  string
	S3Region  string
	S3BaseURL string
}

// Accepted content types for uploads.
var AcceptedImageTypes = []string{
	"image/jpeg",
	"image/png",
	"image/webp",
}

// MaxUploadSize is the maximum allowed file size (5MB).
const MaxUploadSize = 5 * 1024 * 1024

// IsAcceptedImageType checks if a content type is accepted.
func IsAcceptedImageType(contentType string) bool {
	for _, t := range AcceptedImageTypes {
		if t == contentType {
			return true
		}
	}
	return false
}
