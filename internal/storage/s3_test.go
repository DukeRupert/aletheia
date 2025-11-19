package storage

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockS3Client is a mock implementation of S3 client for testing
type MockS3Client struct {
	mock.Mock
}

// PutObject mocks the S3 PutObject operation
func (m *MockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

// GetObject mocks the S3 GetObject operation
func (m *MockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.GetObjectOutput), args.Error(1)
}

// DeleteObject mocks the S3 DeleteObject operation
func (m *MockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.DeleteObjectOutput), args.Error(1)
}

// mockReadCloser wraps a bytes.Reader to implement io.ReadCloser
type mockReadCloser struct {
	*bytes.Reader
}

func (m mockReadCloser) Close() error {
	return nil
}

func TestS3Storage_Save(t *testing.T) {
	tests := []struct {
		name       string
		bucket     string
		filename   string
		setupMock  func(*MockS3Client)
		wantErr    bool
		errMessage string
	}{
		{
			name:     "successful upload",
			bucket:   "test-bucket",
			filename: "test.jpg",
			setupMock: func(m *MockS3Client) {
				m.On("PutObject",
					mock.Anything, // context
					mock.Anything, // *s3.PutObjectInput
				).Return(&s3.PutObjectOutput{}, nil).Maybe()
			},
			wantErr: false,
		},
		{
			name:     "upload failure - bucket not found",
			bucket:   "nonexistent-bucket",
			filename: "test.jpg",
			setupMock: func(m *MockS3Client) {
				m.On("PutObject", mock.Anything, mock.Anything).
					Return(nil, errors.New("NoSuchBucket: The specified bucket does not exist"))
			},
			wantErr:    true,
			errMessage: "failed to upload to S3",
		},
		{
			name:     "upload failure - access denied",
			bucket:   "restricted-bucket",
			filename: "test.jpg",
			setupMock: func(m *MockS3Client) {
				m.On("PutObject", mock.Anything, mock.Anything).
					Return(nil, errors.New("AccessDenied: Access Denied"))
			},
			wantErr:    true,
			errMessage: "failed to upload to S3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := new(MockS3Client)
			tt.setupMock(mockClient)

			// Create S3Storage with mock
			storage := &S3Storage{
				client:  mockClient,
				bucket:  tt.bucket,
				region:  "us-east-1",
				baseURL: "https://test-bucket.s3.amazonaws.com",
			}

			// Create test file
			imageData, err := createTestImage(100, 100, "jpeg")
			assert.NoError(t, err)

			fileHeader, err := createFileHeader(tt.filename, imageData)
			assert.NoError(t, err)

			// Test Save
			ctx := context.Background()
			filename, err := storage.Save(ctx, fileHeader)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, filename)
				assert.Contains(t, filename, ".jpg")
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestS3Storage_Delete(t *testing.T) {
	tests := []struct {
		name       string
		bucket     string
		filename   string
		setupMock  func(*MockS3Client)
		wantErr    bool
		errMessage string
	}{
		{
			name:     "successful deletion",
			bucket:   "test-bucket",
			filename: "test.jpg",
			setupMock: func(m *MockS3Client) {
				m.On("DeleteObject", mock.Anything, mock.MatchedBy(func(input *s3.DeleteObjectInput) bool {
					return *input.Bucket == "test-bucket" && *input.Key == "test.jpg"
				})).Return(&s3.DeleteObjectOutput{}, nil)
			},
			wantErr: false,
		},
		{
			name:     "deletion failure - object not found",
			bucket:   "test-bucket",
			filename: "nonexistent.jpg",
			setupMock: func(m *MockS3Client) {
				m.On("DeleteObject", mock.Anything, mock.Anything).
					Return(nil, errors.New("NoSuchKey: The specified key does not exist"))
			},
			wantErr:    true,
			errMessage: "failed to delete from S3",
		},
		{
			name:     "deletion failure - access denied",
			bucket:   "restricted-bucket",
			filename: "test.jpg",
			setupMock: func(m *MockS3Client) {
				m.On("DeleteObject", mock.Anything, mock.Anything).
					Return(nil, errors.New("AccessDenied: Access Denied"))
			},
			wantErr:    true,
			errMessage: "failed to delete from S3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockS3Client)
			tt.setupMock(mockClient)

			storage := &S3Storage{
				client:  mockClient,
				bucket:  tt.bucket,
				region:  "us-east-1",
				baseURL: "https://test-bucket.s3.amazonaws.com",
			}

			ctx := context.Background()
			err := storage.Delete(ctx, tt.filename)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestS3Storage_GetURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		filename string
		want     string
	}{
		{
			name:     "simple filename",
			baseURL:  "https://test-bucket.s3.amazonaws.com",
			filename: "test.jpg",
			want:     "https://test-bucket.s3.amazonaws.com/test.jpg",
		},
		{
			name:     "cloudfront url",
			baseURL:  "https://d1234567890.cloudfront.net",
			filename: "image.png",
			want:     "https://d1234567890.cloudfront.net/image.png",
		},
		{
			name:     "filename with timestamp",
			baseURL:  "https://test-bucket.s3.amazonaws.com",
			filename: "1234567890_uuid.jpg",
			want:     "https://test-bucket.s3.amazonaws.com/1234567890_uuid.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &S3Storage{
				bucket:  "test-bucket",
				region:  "us-east-1",
				baseURL: tt.baseURL,
			}

			got := storage.GetURL(tt.filename)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestS3Storage_GenerateThumbnail(t *testing.T) {
	tests := []struct {
		name           string
		bucket         string
		originalFile   string
		imageFormat    string
		imageWidth     int
		imageHeight    int
		setupMock      func(*MockS3Client, []byte)
		wantErr        bool
		errMessage     string
		verifyThumbnail bool
	}{
		{
			name:         "successful thumbnail generation - jpeg",
			bucket:       "test-bucket",
			originalFile: "test.jpg",
			imageFormat:  "jpeg",
			imageWidth:   800,
			imageHeight:  600,
			setupMock: func(m *MockS3Client, imageData []byte) {
				// Mock GetObject for original image
				m.On("GetObject", mock.Anything, mock.MatchedBy(func(input *s3.GetObjectInput) bool {
					return *input.Bucket == "test-bucket" && *input.Key == "test.jpg"
				})).Return(&s3.GetObjectOutput{
					Body: mockReadCloser{bytes.NewReader(imageData)},
				}, nil)

				// Mock PutObject for thumbnail
				m.On("PutObject", mock.Anything, mock.MatchedBy(func(input *s3.PutObjectInput) bool {
					return *input.Bucket == "test-bucket" &&
						*input.Key == "test_thumb.jpg" &&
						*input.ContentType == "image/jpeg"
				})).Return(&s3.PutObjectOutput{}, nil)
			},
			wantErr:         false,
			verifyThumbnail: true,
		},
		{
			name:         "successful thumbnail generation - png",
			bucket:       "test-bucket",
			originalFile: "test.png",
			imageFormat:  "png",
			imageWidth:   1024,
			imageHeight:  768,
			setupMock: func(m *MockS3Client, imageData []byte) {
				m.On("GetObject", mock.Anything, mock.Anything).Return(&s3.GetObjectOutput{
					Body: mockReadCloser{bytes.NewReader(imageData)},
				}, nil)

				m.On("PutObject", mock.Anything, mock.MatchedBy(func(input *s3.PutObjectInput) bool {
					return *input.ContentType == "image/png"
				})).Return(&s3.PutObjectOutput{}, nil)
			},
			wantErr:         false,
			verifyThumbnail: true,
		},
		{
			name:         "download failure - object not found",
			bucket:       "test-bucket",
			originalFile: "nonexistent.jpg",
			imageFormat:  "jpeg",
			imageWidth:   100,
			imageHeight:  100,
			setupMock: func(m *MockS3Client, imageData []byte) {
				m.On("GetObject", mock.Anything, mock.Anything).
					Return(nil, errors.New("NoSuchKey: The specified key does not exist"))
			},
			wantErr:    true,
			errMessage: "failed to download original from S3",
		},
		{
			name:         "upload failure - thumbnail upload fails",
			bucket:       "test-bucket",
			originalFile: "test.jpg",
			imageFormat:  "jpeg",
			imageWidth:   800,
			imageHeight:  600,
			setupMock: func(m *MockS3Client, imageData []byte) {
				m.On("GetObject", mock.Anything, mock.Anything).Return(&s3.GetObjectOutput{
					Body: mockReadCloser{bytes.NewReader(imageData)},
				}, nil)

				m.On("PutObject", mock.Anything, mock.Anything).
					Return(nil, errors.New("InternalError: We encountered an internal error"))
			},
			wantErr:    true,
			errMessage: "failed to upload thumbnail to S3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test image
			imageData, err := createTestImage(tt.imageWidth, tt.imageHeight, tt.imageFormat)
			assert.NoError(t, err)

			mockClient := new(MockS3Client)
			tt.setupMock(mockClient, imageData)

			storage := &S3Storage{
				client:  mockClient,
				bucket:  tt.bucket,
				region:  "us-east-1",
				baseURL: "https://test-bucket.s3.amazonaws.com",
			}

			ctx := context.Background()
			thumbnailFilename, err := storage.GenerateThumbnail(ctx, tt.originalFile)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, thumbnailFilename)

				if tt.verifyThumbnail {
					// Verify thumbnail filename has _thumb suffix
					assert.Contains(t, thumbnailFilename, "_thumb")

					// Verify extension is preserved
					originalExt := tt.originalFile[len(tt.originalFile)-4:]
					assert.Contains(t, thumbnailFilename, originalExt)
				}
			}

			mockClient.AssertExpectations(t)
		})
	}
}

func TestS3Storage_GenerateThumbnail_InvalidImage(t *testing.T) {
	mockClient := new(MockS3Client)

	// Mock returning invalid image data
	invalidData := []byte("not an image")
	mockClient.On("GetObject", mock.Anything, mock.Anything).Return(&s3.GetObjectOutput{
		Body: mockReadCloser{bytes.NewReader(invalidData)},
	}, nil)

	storage := &S3Storage{
		client:  mockClient,
		bucket:  "test-bucket",
		region:  "us-east-1",
		baseURL: "https://test-bucket.s3.amazonaws.com",
	}

	ctx := context.Background()
	_, err := storage.GenerateThumbnail(ctx, "invalid.jpg")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode image")
	mockClient.AssertExpectations(t)
}

func TestNewS3Storage(t *testing.T) {
	tests := []struct {
		name    string
		bucket  string
		region  string
		baseURL string
	}{
		{
			name:    "creates s3 storage with s3 url",
			bucket:  "test-bucket",
			region:  "us-east-1",
			baseURL: "https://test-bucket.s3.amazonaws.com",
		},
		{
			name:    "creates s3 storage with cloudfront url",
			bucket:  "prod-bucket",
			region:  "us-west-2",
			baseURL: "https://d1234567890.cloudfront.net",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockS3Client)

			storage := NewS3Storage(mockClient, tt.bucket, tt.region, tt.baseURL)

			assert.NotNil(t, storage)
			assert.Equal(t, tt.bucket, storage.bucket)
			assert.Equal(t, tt.region, storage.region)
			assert.Equal(t, tt.baseURL, storage.baseURL)
		})
	}
}

// TestS3Storage_Interface verifies S3Storage implements FileStorage interface
func TestS3Storage_Interface(t *testing.T) {
	var _ FileStorage = (*S3Storage)(nil)
}
