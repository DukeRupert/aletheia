package aletheia

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Photo represents an image captured during an inspection.
type Photo struct {
	ID           uuid.UUID `json:"id"`
	InspectionID uuid.UUID `json:"inspectionId"`
	StorageURL   string    `json:"storageUrl"`
	ThumbnailURL string    `json:"thumbnailUrl,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`

	// Joined fields (populated by some queries)
	Inspection *Inspection  `json:"inspection,omitempty"`
	Violations []*Violation `json:"violations,omitempty"`
}

// PhotoService defines operations for managing photos.
type PhotoService interface {
	// FindPhotoByID retrieves a photo by its ID.
	// Returns ENOTFOUND if the photo does not exist.
	FindPhotoByID(ctx context.Context, id uuid.UUID) (*Photo, error)

	// FindPhotos retrieves photos matching the filter criteria.
	// Returns the matching photos and total count.
	FindPhotos(ctx context.Context, filter PhotoFilter) ([]*Photo, int, error)

	// CreatePhoto creates a new photo record.
	// Note: Actual file upload is handled by FileStorage.
	CreatePhoto(ctx context.Context, photo *Photo) error

	// UpdatePhoto updates an existing photo.
	// Returns ENOTFOUND if the photo does not exist.
	UpdatePhoto(ctx context.Context, id uuid.UUID, upd PhotoUpdate) (*Photo, error)

	// DeletePhoto deletes a photo and its associated violations.
	// Note: Actual file deletion should be handled by FileStorage.
	// Returns ENOTFOUND if the photo does not exist.
	DeletePhoto(ctx context.Context, id uuid.UUID) error

	// FindPhotoWithViolations retrieves a photo with its associated violations.
	// Returns ENOTFOUND if the photo does not exist.
	FindPhotoWithViolations(ctx context.Context, id uuid.UUID) (*Photo, error)
}

// PhotoFilter defines criteria for filtering photos.
type PhotoFilter struct {
	ID           *uuid.UUID
	InspectionID *uuid.UUID

	// Pagination
	Offset int
	Limit  int
}

// PhotoUpdate defines fields that can be updated on a photo.
type PhotoUpdate struct {
	ThumbnailURL *string
}
