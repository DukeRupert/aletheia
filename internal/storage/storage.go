package storage

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
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
	GenerateThumbnail(ctx context.Context, originalFilename string) (thumbnailFilename string, err error)
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

// GenerateThumbnail creates a thumbnail from the original image
func (s *LocalStorage) GenerateThumbnail(ctx context.Context, originalFilename string) (string, error) {
	// Open original file
	originalPath := filepath.Join(s.basePath, originalFilename)
	file, err := os.Open(originalPath)
	if err != nil {
		return "", fmt.Errorf("failed to open original file: %w", err)
	}
	defer file.Close()

	// Decode image
	img, format, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize to thumbnail (300x300 max, maintaining aspect ratio)
	thumbnail := resizeImage(img, 300, 300)

	// Generate thumbnail filename
	ext := filepath.Ext(originalFilename)
	nameWithoutExt := strings.TrimSuffix(originalFilename, ext)
	thumbnailFilename := fmt.Sprintf("%s_thumb%s", nameWithoutExt, ext)
	thumbnailPath := filepath.Join(s.basePath, thumbnailFilename)

	// Create thumbnail file
	thumbFile, err := os.Create(thumbnailPath)
	if err != nil {
		return "", fmt.Errorf("failed to create thumbnail file: %w", err)
	}
	defer thumbFile.Close()

	// Encode thumbnail
	switch format {
	case "jpeg":
		err = jpeg.Encode(thumbFile, thumbnail, &jpeg.Options{Quality: 85})
	case "png":
		err = png.Encode(thumbFile, thumbnail)
	default:
		// Default to JPEG for unknown formats
		err = jpeg.Encode(thumbFile, thumbnail, &jpeg.Options{Quality: 85})
	}

	if err != nil {
		return "", fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	return thumbnailFilename, nil
}

// resizeImage resizes an image to fit within maxWidth x maxHeight while maintaining aspect ratio
func resizeImage(img image.Image, maxWidth, maxHeight int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate new dimensions maintaining aspect ratio
	newWidth, newHeight := width, height
	if width > maxWidth || height > maxHeight {
		ratio := float64(width) / float64(height)
		if width > height {
			newWidth = maxWidth
			newHeight = int(float64(maxWidth) / ratio)
		} else {
			newHeight = maxHeight
			newWidth = int(float64(maxHeight) * ratio)
		}
	}

	// If image is already smaller or equal, return original
	if newWidth >= width && newHeight >= height {
		return img
	}

	// Create new image
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Simple nearest-neighbor resizing using the standard library
	xRatio := float64(width) / float64(newWidth)
	yRatio := float64(height) / float64(newHeight)

	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := int(float64(x) * xRatio)
			srcY := int(float64(y) * yRatio)
			dst.Set(x, y, img.At(srcX, srcY))
		}
	}

	return dst
}
