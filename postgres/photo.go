package postgres

import (
	"context"

	"github.com/dukerupert/aletheia"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Compile-time check that PhotoService implements aletheia.PhotoService.
var _ aletheia.PhotoService = (*PhotoService)(nil)

// PhotoService implements aletheia.PhotoService using PostgreSQL.
type PhotoService struct {
	db *DB
}

func (s *PhotoService) FindPhotoByID(ctx context.Context, id uuid.UUID) (*aletheia.Photo, error) {
	photo, err := s.db.queries.GetPhoto(ctx, toPgUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("Photo not found")
		}
		return nil, aletheia.Internal("Failed to fetch photo", err)
	}
	return toDomainPhoto(photo), nil
}

func (s *PhotoService) FindPhotos(ctx context.Context, filter aletheia.PhotoFilter) ([]*aletheia.Photo, int, error) {
	if filter.InspectionID == nil {
		return nil, 0, aletheia.Invalid("Inspection ID is required")
	}

	photos, err := s.db.queries.ListPhotos(ctx, toPgUUID(*filter.InspectionID))
	if err != nil {
		return nil, 0, aletheia.Internal("Failed to list photos", err)
	}

	// Apply offset/limit in memory
	total := len(photos)
	if filter.Offset > 0 && filter.Offset < len(photos) {
		photos = photos[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(photos) {
		photos = photos[:filter.Limit]
	}

	return toDomainPhotos(photos), total, nil
}

func (s *PhotoService) CreatePhoto(ctx context.Context, photo *aletheia.Photo) error {
	dbPhoto, err := s.db.queries.CreatePhoto(ctx, database.CreatePhotoParams{
		InspectionID: toPgUUID(photo.InspectionID),
		StorageUrl:   photo.StorageURL,
		ThumbnailUrl: toPgText(photo.ThumbnailURL),
	})
	if err != nil {
		if isForeignKeyViolation(err) {
			return aletheia.NotFound("Inspection not found")
		}
		return aletheia.Internal("Failed to create photo", err)
	}

	// Update photo with generated values
	photo.ID = fromPgUUID(dbPhoto.ID)
	photo.CreatedAt = fromPgTimestamp(dbPhoto.CreatedAt)

	return nil
}

func (s *PhotoService) UpdatePhoto(ctx context.Context, id uuid.UUID, upd aletheia.PhotoUpdate) (*aletheia.Photo, error) {
	// Currently no update query exists in sqlc, just return current photo
	// TODO: Add UpdatePhoto query to sqlc
	photo, err := s.FindPhotoByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates in memory (would need sqlc query to persist)
	if upd.ThumbnailURL != nil {
		photo.ThumbnailURL = *upd.ThumbnailURL
	}

	return photo, nil
}

func (s *PhotoService) DeletePhoto(ctx context.Context, id uuid.UUID) error {
	err := s.db.queries.DeletePhoto(ctx, toPgUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return aletheia.NotFound("Photo not found")
		}
		return aletheia.Internal("Failed to delete photo", err)
	}
	return nil
}

func (s *PhotoService) FindPhotoWithViolations(ctx context.Context, id uuid.UUID) (*aletheia.Photo, error) {
	// Get photo
	photo, err := s.FindPhotoByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get violations for this photo
	violations, err := s.db.queries.ListDetectedViolations(ctx, toPgUUID(id))
	if err != nil {
		return nil, aletheia.Internal("Failed to fetch violations", err)
	}

	photo.Violations = toDomainViolations(violations)
	return photo, nil
}
