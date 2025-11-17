package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// FileStorage defines the interface for file storage operations
type FileStorage interface {
	Save(ctx context.Context, file *multipart.FileHeader) (string, error)
	Delete(ctx context.Context, filename string) error
	GetURL(filename string) string
}

// LocalStorage implements FileStorage for local disk storage
type LocalStorage struct {
	basePath string
	baseURL  string
}

// NewLocalStorage creates a new local storage instance
func NewLocalStorage(basePath, baseURL string) (*LocalStorage, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalStorage{
		basePath: basePath,
		baseURL:  baseURL,
	}, nil
}

// Save saves a file to local disk with a unique filename
func (s *LocalStorage) Save(ctx context.Context, fileHeader *multipart.FileHeader) (string, error) {
	// Open uploaded file
	src, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Generate unique filename
	ext := filepath.Ext(fileHeader.Filename)
	filename := fmt.Sprintf("%d_%s%s", time.Now().Unix(), uuid.New().String(), ext)

	// Create destination file
	destPath := filepath.Join(s.basePath, filename)
	dst, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// Copy file contents
	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	return filename, nil
}

// Delete removes a file from local disk
func (s *LocalStorage) Delete(ctx context.Context, filename string) error {
	filePath := filepath.Join(s.basePath, filename)
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// GetURL returns the URL to access the file
func (s *LocalStorage) GetURL(filename string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, filename)
}
