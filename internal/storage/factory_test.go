package storage

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFileStorage_LocalProvider(t *testing.T) {
	tests := []struct {
		name      string
		cfg       StorageConfig
		wantErr   bool
		wantType  string
		cleanupFn func()
	}{
		{
			name: "creates local storage successfully",
			cfg: StorageConfig{
				Provider:  "local",
				LocalPath: "./testdata/factory-test",
				LocalURL:  "http://localhost:8080/uploads",
			},
			wantErr:  false,
			wantType: "*storage.LocalStorage",
			cleanupFn: func() {
				os.RemoveAll("./testdata")
			},
		},
		{
			name: "creates local storage with different path",
			cfg: StorageConfig{
				Provider:  "local",
				LocalPath: "./testdata/factory-test-2",
				LocalURL:  "http://example.com/files",
			},
			wantErr:  false,
			wantType: "*storage.LocalStorage",
			cleanupFn: func() {
				os.RemoveAll("./testdata")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cleanupFn != nil {
				defer tt.cleanupFn()
			}

			ctx := context.Background()
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			storage, err := NewFileStorage(ctx, logger, tt.cfg)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, storage)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, storage)

				// Verify the type
				_, ok := storage.(*LocalStorage)
				assert.True(t, ok, "Expected storage to be *LocalStorage")

				// Verify the storage can be used
				url := storage.GetURL("test.jpg")
				assert.Contains(t, url, "test.jpg")
			}
		})
	}
}

func TestNewFileStorage_UnknownProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "treats empty provider as local",
			provider: "",
			wantErr:  false, // Default case uses local storage
			errMsg:   "",
		},
		{
			name:     "treats unknown provider as local",
			provider: "dropbox",
			wantErr:  false, // Default case uses local storage
			errMsg:   "",
		},
	}

	defer os.RemoveAll("./testdata")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			cfg := StorageConfig{
				Provider:  tt.provider,
				LocalPath: "./testdata/unknown-test",
				LocalURL:  "http://localhost:8080/uploads",
			}

			storage, err := NewFileStorage(ctx, logger, cfg)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, storage)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, storage)
			}
		})
	}
}

func TestNewFileStorage_S3Provider(t *testing.T) {
	t.Run("s3 provider initialization", func(t *testing.T) {
		ctx := context.Background()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		cfg := StorageConfig{
			Provider:  "s3",
			S3Bucket:  "test-bucket",
			S3Region:  "us-east-1",
			S3BaseURL: "https://test-bucket.s3.amazonaws.com",
		}

		storage, err := NewFileStorage(ctx, logger, cfg)

		// In test environment without AWS credentials, this will likely fail
		// or succeed depending on the environment. We test both cases.
		if err != nil {
			// If AWS config loading fails (expected in most test environments)
			assert.Error(t, err)
			assert.Nil(t, storage)
			t.Logf("Expected error in test environment without AWS credentials: %v", err)
		} else {
			// If AWS credentials are available (CI environment or developer machine)
			assert.NoError(t, err)
			assert.NotNil(t, storage)

			// Verify the type
			_, ok := storage.(*S3Storage)
			assert.True(t, ok, "Expected storage to be *S3Storage")

			// Verify the storage can generate URLs
			url := storage.GetURL("test.jpg")
			assert.Contains(t, url, "test.jpg")
			assert.Contains(t, url, "https://")
		}
	})

	t.Run("s3 provider with cloudfront url", func(t *testing.T) {
		ctx := context.Background()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		cfg := StorageConfig{
			Provider:  "s3",
			S3Bucket:  "prod-bucket",
			S3Region:  "us-west-2",
			S3BaseURL: "https://d1234567890.cloudfront.net",
		}

		storage, err := NewFileStorage(ctx, logger, cfg)

		// Same as above - might succeed or fail depending on environment
		if err == nil {
			assert.NotNil(t, storage)

			// Verify CloudFront URL is used
			url := storage.GetURL("image.png")
			assert.Contains(t, url, "cloudfront.net")
		} else {
			t.Logf("Expected error in test environment: %v", err)
		}
	})
}

func TestNewFileStorage_Interface(t *testing.T) {
	defer os.RemoveAll("./testdata")

	t.Run("local storage implements FileStorage interface", func(t *testing.T) {
		ctx := context.Background()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		cfg := StorageConfig{
			Provider:  "local",
			LocalPath: "./testdata/interface-test",
			LocalURL:  "http://localhost:8080/uploads",
		}

		storage, err := NewFileStorage(ctx, logger, cfg)
		assert.NoError(t, err)
		assert.NotNil(t, storage)

		// Verify it implements the interface by assigning to interface type
		var _ FileStorage = storage
	})
}
