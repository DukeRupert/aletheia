package mock

import (
	"context"
	"io"

	"github.com/dukerupert/aletheia"
)

// Compile-time interface check
var _ aletheia.FileStorage = (*FileStorage)(nil)

// FileStorage is a mock implementation of aletheia.FileStorage.
type FileStorage struct {
	UploadFn  func(ctx context.Context, key string, reader io.Reader, contentType string) (string, error)
	DeleteFn  func(ctx context.Context, key string) error
	GetURLFn  func(key string) string
	ExistsFn  func(ctx context.Context, key string) (bool, error)
}

func (s *FileStorage) Upload(ctx context.Context, key string, reader io.Reader, contentType string) (string, error) {
	if s.UploadFn != nil {
		return s.UploadFn(ctx, key, reader, contentType)
	}
	return "https://mock-storage.example.com/" + key, nil
}

func (s *FileStorage) Delete(ctx context.Context, key string) error {
	if s.DeleteFn != nil {
		return s.DeleteFn(ctx, key)
	}
	return nil
}

func (s *FileStorage) GetURL(key string) string {
	if s.GetURLFn != nil {
		return s.GetURLFn(key)
	}
	return "https://mock-storage.example.com/" + key
}

func (s *FileStorage) Exists(ctx context.Context, key string) (bool, error) {
	if s.ExistsFn != nil {
		return s.ExistsFn(ctx, key)
	}
	return false, nil
}
