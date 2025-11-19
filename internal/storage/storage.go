package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// FileStorage defines the interface for file storage operations
type FileStorage interface {
	Save(ctx context.Context, file *multipart.FileHeader) (string, error)
	Delete(ctx context.Context, filename string) error
	GetURL(filename string) string
}

// StorageConfig holds configuration for storage services
type StorageConfig struct {
	Provider  string // "local" or "s3"
	LocalPath string // Path for local storage
	LocalURL  string // Base URL for local storage
	S3Bucket  string // S3 bucket name
	S3Region  string // S3 region
	S3BaseURL string // CloudFront or S3 base URL
}

// NewFileStorage creates a file storage instance based on the provider configuration
func NewFileStorage(ctx context.Context, logger *slog.Logger, cfg StorageConfig) (FileStorage, error) {
	switch cfg.Provider {
	case "s3":
		// Load AWS configuration
		awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.S3Region))
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}

		// Create S3 client
		s3Client := s3.NewFromConfig(awsCfg)

		logger.Info("initialized S3 storage",
			slog.String("bucket", cfg.S3Bucket),
			slog.String("region", cfg.S3Region),
		)

		return NewS3Storage(s3Client, cfg.S3Bucket, cfg.S3Region, cfg.S3BaseURL), nil

	default: // "local"
		storage, err := NewLocalStorage(cfg.LocalPath, cfg.LocalURL)
		if err != nil {
			return nil, fmt.Errorf("failed to create local storage: %w", err)
		}

		logger.Info("initialized local storage",
			slog.String("path", cfg.LocalPath),
			slog.String("url", cfg.LocalURL),
		)

		return storage, nil
	}
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
