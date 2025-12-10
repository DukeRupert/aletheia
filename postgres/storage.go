package postgres

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/dukerupert/aletheia"
	"github.com/google/uuid"
)

// Compile-time interface check
var _ aletheia.FileStorage = (*LocalStorage)(nil)
var _ aletheia.FileStorage = (*S3Storage)(nil)

// NewFileStorage creates a file storage instance based on the provider configuration.
func NewFileStorage(ctx context.Context, logger *slog.Logger, cfg aletheia.StorageConfig) (aletheia.FileStorage, error) {
	switch cfg.Provider {
	case "s3":
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(cfg.S3Region))
		if err != nil {
			return nil, fmt.Errorf("loading AWS config: %w", err)
		}
		client := s3.NewFromConfig(awsCfg)
		logger.Info("initialized S3 storage",
			slog.String("bucket", cfg.S3Bucket),
			slog.String("region", cfg.S3Region))
		return &S3Storage{
			client:  client,
			bucket:  cfg.S3Bucket,
			region:  cfg.S3Region,
			baseURL: cfg.S3BaseURL,
		}, nil
	default:
		if err := os.MkdirAll(cfg.LocalPath, 0755); err != nil {
			return nil, fmt.Errorf("creating storage directory: %w", err)
		}
		logger.Info("initialized local storage",
			slog.String("path", cfg.LocalPath),
			slog.String("url", cfg.LocalURL))
		return &LocalStorage{
			basePath: cfg.LocalPath,
			baseURL:  cfg.LocalURL,
		}, nil
	}
}

// LocalStorage implements aletheia.FileStorage for local disk storage.
type LocalStorage struct {
	basePath string
	baseURL  string
}

// Upload saves a file to local disk.
func (s *LocalStorage) Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	// Generate unique filename if key is empty
	if key == "" {
		key = fmt.Sprintf("%d_%s", time.Now().Unix(), uuid.New().String())
	}

	filePath := filepath.Join(s.basePath, key)

	// Create parent directories if needed
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", fmt.Errorf("creating directories: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}

	return s.GetURL(key), nil
}

// Delete removes a file from local disk.
func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	filePath := filepath.Join(s.basePath, key)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting file: %w", err)
	}
	return nil
}

// GetURL returns the URL to access the file.
func (s *LocalStorage) GetURL(key string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, key)
}

// Exists checks if a file exists in local storage.
func (s *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	filePath := filepath.Join(s.basePath, key)
	_, err := os.Stat(filePath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("checking file: %w", err)
}

// S3Storage implements aletheia.FileStorage for AWS S3.
type S3Storage struct {
	client  *s3.Client
	bucket  string
	region  string
	baseURL string
}

// Upload uploads a file to S3.
func (s *S3Storage) Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	// Generate unique filename if key is empty
	if key == "" {
		key = fmt.Sprintf("%d_%s", time.Now().Unix(), uuid.New().String())
	}

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("uploading to S3: %w", err)
	}

	return s.GetURL(key), nil
}

// Delete removes a file from S3.
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("deleting from S3: %w", err)
	}
	return nil
}

// GetURL returns the URL to access the file.
func (s *S3Storage) GetURL(key string) string {
	if s.baseURL != "" {
		return fmt.Sprintf("%s/%s", s.baseURL, key)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
}

// Exists checks if a file exists in S3.
func (s *S3Storage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if it's a "not found" error
		return false, nil
	}
	return true, nil
}
