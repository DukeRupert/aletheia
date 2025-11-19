package storage

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"mime/multipart"
)

// S3ClientInterface defines the S3 operations we need for storage
type S3ClientInterface interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

// S3Storage implements FileStorage for AWS S3
type S3Storage struct {
	client  S3ClientInterface
	bucket  string
	region  string
	baseURL string // CloudFront or S3 URL
}

// NewS3Storage creates a new S3 storage instance
func NewS3Storage(client S3ClientInterface, bucket, region, baseURL string) *S3Storage {
	return &S3Storage{
		client:  client,
		bucket:  bucket,
		region:  region,
		baseURL: baseURL,
	}
}

// Save uploads a file to S3
func (s *S3Storage) Save(ctx context.Context, fileHeader *multipart.FileHeader) (string, error) {
	src, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Generate unique filename
	ext := filepath.Ext(fileHeader.Filename)
	filename := fmt.Sprintf("%d_%s%s", time.Now().Unix(), uuid.New().String(), ext)

	// Upload to S3
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(filename),
		Body:        src,
		ContentType: aws.String(fileHeader.Header.Get("Content-Type")),
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	return filename, nil
}

// Delete removes a file from S3
func (s *S3Storage) Delete(ctx context.Context, filename string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filename),
	})

	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}
	return nil
}

// GetURL returns the URL to access the file
func (s *S3Storage) GetURL(filename string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, filename)
}

// GenerateThumbnail creates a thumbnail from the original image stored in S3
func (s *S3Storage) GenerateThumbnail(ctx context.Context, originalFilename string) (string, error) {
	// Download original image from S3
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(originalFilename),
	})
	if err != nil {
		return "", fmt.Errorf("failed to download original from S3: %w", err)
	}
	defer result.Body.Close()

	// Decode image
	img, format, err := image.Decode(result.Body)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize to thumbnail (300x300 max, maintaining aspect ratio)
	thumbnail := resizeImage(img, 300, 300)

	// Generate thumbnail filename
	ext := filepath.Ext(originalFilename)
	nameWithoutExt := strings.TrimSuffix(originalFilename, ext)
	thumbnailFilename := fmt.Sprintf("%s_thumb%s", nameWithoutExt, ext)

	// Encode thumbnail to buffer
	var buf bytes.Buffer
	var contentType string

	switch format {
	case "jpeg":
		contentType = "image/jpeg"
		err = jpeg.Encode(&buf, thumbnail, &jpeg.Options{Quality: 85})
	case "png":
		contentType = "image/png"
		err = png.Encode(&buf, thumbnail)
	default:
		// Default to JPEG for unknown formats
		contentType = "image/jpeg"
		err = jpeg.Encode(&buf, thumbnail, &jpeg.Options{Quality: 85})
	}

	if err != nil {
		return "", fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	// Upload thumbnail to S3
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(thumbnailFilename),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload thumbnail to S3: %w", err)
	}

	return thumbnailFilename, nil
}
