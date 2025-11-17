package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"mime/multipart"
)

// S3Storage implements FileStorage for AWS S3
type S3Storage struct {
	client  *s3.Client
	bucket  string
	region  string
	baseURL string // CloudFront or S3 URL
}

// NewS3Storage creates a new S3 storage instance
func NewS3Storage(client *s3.Client, bucket, region, baseURL string) *S3Storage {
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
