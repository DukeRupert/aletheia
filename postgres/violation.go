package postgres

import (
	"context"

	"github.com/dukerupert/aletheia"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Compile-time check that ViolationService implements aletheia.ViolationService.
var _ aletheia.ViolationService = (*ViolationService)(nil)

// ViolationService implements aletheia.ViolationService using PostgreSQL.
type ViolationService struct {
	db *DB
}

func (s *ViolationService) FindViolationByID(ctx context.Context, id uuid.UUID) (*aletheia.Violation, error) {
	violation, err := s.db.queries.GetDetectedViolation(ctx, toPgUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("Violation not found")
		}
		return nil, aletheia.Internal("Failed to fetch violation", err)
	}
	return toDomainViolation(violation), nil
}

func (s *ViolationService) FindViolations(ctx context.Context, filter aletheia.ViolationFilter) ([]*aletheia.Violation, int, error) {
	var violations []database.DetectedViolation
	var err error

	// Choose query based on filter criteria
	if filter.PhotoID != nil && filter.Status != nil {
		violations, err = s.db.queries.ListDetectedViolationsByStatus(ctx, database.ListDetectedViolationsByStatusParams{
			PhotoID: toPgUUID(*filter.PhotoID),
			Status:  database.ViolationStatus(*filter.Status),
		})
	} else if filter.PhotoID != nil {
		violations, err = s.db.queries.ListDetectedViolations(ctx, toPgUUID(*filter.PhotoID))
	} else if filter.InspectionID != nil && filter.Status != nil {
		violations, err = s.db.queries.ListDetectedViolationsByInspectionAndStatus(ctx, database.ListDetectedViolationsByInspectionAndStatusParams{
			InspectionID: toPgUUID(*filter.InspectionID),
			Status:       database.ViolationStatus(*filter.Status),
		})
	} else if filter.InspectionID != nil {
		violations, err = s.db.queries.ListDetectedViolationsByInspection(ctx, toPgUUID(*filter.InspectionID))
	} else {
		return nil, 0, aletheia.Invalid("PhotoID or InspectionID is required")
	}

	if err != nil {
		return nil, 0, aletheia.Internal("Failed to list violations", err)
	}

	// Apply offset/limit in memory
	total := len(violations)
	if filter.Offset > 0 && filter.Offset < len(violations) {
		violations = violations[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(violations) {
		violations = violations[:filter.Limit]
	}

	return toDomainViolations(violations), total, nil
}

func (s *ViolationService) CreateViolation(ctx context.Context, violation *aletheia.Violation) error {
	dbViolation, err := s.db.queries.CreateDetectedViolation(ctx, database.CreateDetectedViolationParams{
		PhotoID:         toPgUUID(violation.PhotoID),
		Description:     violation.Description,
		ConfidenceScore: toPgNumeric(violation.ConfidenceScore),
		SafetyCodeID:    toPgUUID(violation.SafetyCodeID),
		Status:          database.ViolationStatus(violation.Status),
		Severity:        database.ViolationSeverity(violation.Severity),
		Location:        toPgText(violation.Location),
	})
	if err != nil {
		if isForeignKeyViolation(err) {
			return aletheia.NotFound("Photo or safety code not found")
		}
		return aletheia.Internal("Failed to create violation", err)
	}

	// Update violation with generated values
	violation.ID = fromPgUUID(dbViolation.ID)
	violation.CreatedAt = fromPgTimestamp(dbViolation.CreatedAt)

	return nil
}

func (s *ViolationService) CreateViolations(ctx context.Context, violations []*aletheia.Violation) error {
	for _, v := range violations {
		if err := s.CreateViolation(ctx, v); err != nil {
			return err
		}
	}
	return nil
}

func (s *ViolationService) UpdateViolation(ctx context.Context, id uuid.UUID, upd aletheia.ViolationUpdate) (*aletheia.Violation, error) {
	// Handle status update
	if upd.Status != nil {
		violation, err := s.db.queries.UpdateDetectedViolationStatus(ctx, database.UpdateDetectedViolationStatusParams{
			ID:     toPgUUID(id),
			Status: database.ViolationStatus(*upd.Status),
		})
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, aletheia.NotFound("Violation not found")
			}
			return nil, aletheia.Internal("Failed to update violation status", err)
		}
		return toDomainViolation(violation), nil
	}

	// Handle safety code update
	if upd.SafetyCodeID != nil {
		violation, err := s.db.queries.UpdateDetectedViolationSafetyCode(ctx, database.UpdateDetectedViolationSafetyCodeParams{
			ID:           toPgUUID(id),
			SafetyCodeID: toPgUUID(*upd.SafetyCodeID),
		})
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, aletheia.NotFound("Violation not found")
			}
			return nil, aletheia.Internal("Failed to update violation safety code", err)
		}
		return toDomainViolation(violation), nil
	}

	// Handle description update
	if upd.Description != nil {
		violation, err := s.db.queries.UpdateDetectedViolationNotes(ctx, database.UpdateDetectedViolationNotesParams{
			ID:          toPgUUID(id),
			Description: toPgText(*upd.Description),
		})
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, aletheia.NotFound("Violation not found")
			}
			return nil, aletheia.Internal("Failed to update violation description", err)
		}
		return toDomainViolation(violation), nil
	}

	// No update requested, just return current violation
	return s.FindViolationByID(ctx, id)
}

func (s *ViolationService) ConfirmViolation(ctx context.Context, id uuid.UUID) (*aletheia.Violation, error) {
	violation, err := s.db.queries.UpdateDetectedViolationStatus(ctx, database.UpdateDetectedViolationStatusParams{
		ID:     toPgUUID(id),
		Status: database.ViolationStatusConfirmed,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("Violation not found")
		}
		return nil, aletheia.Internal("Failed to confirm violation", err)
	}
	return toDomainViolation(violation), nil
}

func (s *ViolationService) DismissViolation(ctx context.Context, id uuid.UUID) (*aletheia.Violation, error) {
	violation, err := s.db.queries.UpdateDetectedViolationStatus(ctx, database.UpdateDetectedViolationStatusParams{
		ID:     toPgUUID(id),
		Status: database.ViolationStatusDismissed,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("Violation not found")
		}
		return nil, aletheia.Internal("Failed to dismiss violation", err)
	}
	return toDomainViolation(violation), nil
}

func (s *ViolationService) SetViolationPending(ctx context.Context, id uuid.UUID) (*aletheia.Violation, error) {
	violation, err := s.db.queries.UpdateDetectedViolationStatus(ctx, database.UpdateDetectedViolationStatusParams{
		ID:     toPgUUID(id),
		Status: database.ViolationStatusPending,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("Violation not found")
		}
		return nil, aletheia.Internal("Failed to set violation to pending", err)
	}
	return toDomainViolation(violation), nil
}

func (s *ViolationService) DeleteViolation(ctx context.Context, id uuid.UUID) error {
	err := s.db.queries.DeleteDetectedViolation(ctx, toPgUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return aletheia.NotFound("Violation not found")
		}
		return aletheia.Internal("Failed to delete violation", err)
	}
	return nil
}

func (s *ViolationService) GetViolationsByInspection(ctx context.Context, inspectionID uuid.UUID) ([]*aletheia.Violation, error) {
	violations, err := s.db.queries.ListDetectedViolationsByInspection(ctx, toPgUUID(inspectionID))
	if err != nil {
		return nil, aletheia.Internal("Failed to fetch violations", err)
	}
	return toDomainViolations(violations), nil
}
