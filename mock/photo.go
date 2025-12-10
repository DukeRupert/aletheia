package mock

import (
	"context"
	"time"

	"github.com/dukerupert/aletheia"
	"github.com/google/uuid"
)

// Compile-time interface check
var _ aletheia.PhotoService = (*PhotoService)(nil)

// PhotoService is a mock implementation of aletheia.PhotoService.
type PhotoService struct {
	FindPhotoByIDFn          func(ctx context.Context, id uuid.UUID) (*aletheia.Photo, error)
	FindPhotosFn             func(ctx context.Context, filter aletheia.PhotoFilter) ([]*aletheia.Photo, int, error)
	CreatePhotoFn            func(ctx context.Context, photo *aletheia.Photo) error
	UpdatePhotoFn            func(ctx context.Context, id uuid.UUID, upd aletheia.PhotoUpdate) (*aletheia.Photo, error)
	DeletePhotoFn            func(ctx context.Context, id uuid.UUID) error
	FindPhotoWithViolationsFn func(ctx context.Context, id uuid.UUID) (*aletheia.Photo, error)
}

func (s *PhotoService) FindPhotoByID(ctx context.Context, id uuid.UUID) (*aletheia.Photo, error) {
	if s.FindPhotoByIDFn != nil {
		return s.FindPhotoByIDFn(ctx, id)
	}
	return nil, aletheia.NotFound("Photo not found")
}

func (s *PhotoService) FindPhotos(ctx context.Context, filter aletheia.PhotoFilter) ([]*aletheia.Photo, int, error) {
	if s.FindPhotosFn != nil {
		return s.FindPhotosFn(ctx, filter)
	}
	return []*aletheia.Photo{}, 0, nil
}

func (s *PhotoService) CreatePhoto(ctx context.Context, photo *aletheia.Photo) error {
	if s.CreatePhotoFn != nil {
		return s.CreatePhotoFn(ctx, photo)
	}
	if photo.ID == uuid.Nil {
		photo.ID = uuid.New()
	}
	photo.CreatedAt = time.Now()
	return nil
}

func (s *PhotoService) UpdatePhoto(ctx context.Context, id uuid.UUID, upd aletheia.PhotoUpdate) (*aletheia.Photo, error) {
	if s.UpdatePhotoFn != nil {
		return s.UpdatePhotoFn(ctx, id, upd)
	}
	return nil, aletheia.NotFound("Photo not found")
}

func (s *PhotoService) DeletePhoto(ctx context.Context, id uuid.UUID) error {
	if s.DeletePhotoFn != nil {
		return s.DeletePhotoFn(ctx, id)
	}
	return nil
}

func (s *PhotoService) FindPhotoWithViolations(ctx context.Context, id uuid.UUID) (*aletheia.Photo, error) {
	if s.FindPhotoWithViolationsFn != nil {
		return s.FindPhotoWithViolationsFn(ctx, id)
	}
	return nil, aletheia.NotFound("Photo not found")
}
