package postgres

import (
	"context"

	"github.com/dukerupert/aletheia"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Compile-time check that InspectionService implements aletheia.InspectionService.
var _ aletheia.InspectionService = (*InspectionService)(nil)

// InspectionService implements aletheia.InspectionService using PostgreSQL.
type InspectionService struct {
	db *DB
}

func (s *InspectionService) FindInspectionByID(ctx context.Context, id uuid.UUID) (*aletheia.Inspection, error) {
	inspection, err := s.db.queries.GetInspection(ctx, toPgUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("Inspection not found")
		}
		return nil, aletheia.Internal("Failed to fetch inspection", err)
	}
	return toDomainInspection(inspection), nil
}

func (s *InspectionService) FindInspections(ctx context.Context, filter aletheia.InspectionFilter) ([]*aletheia.Inspection, int, error) {
	var inspections []database.Inspection
	var err error

	// Choose query based on filter criteria
	if filter.ProjectID != nil && filter.Status != nil {
		inspections, err = s.db.queries.ListInspectionsByStatus(ctx, database.ListInspectionsByStatusParams{
			ProjectID: toPgUUID(*filter.ProjectID),
			Status:    database.InspectionStatus(*filter.Status),
		})
	} else if filter.ProjectID != nil {
		inspections, err = s.db.queries.ListInspections(ctx, toPgUUID(*filter.ProjectID))
	} else if filter.InspectorID != nil {
		inspections, err = s.db.queries.ListInspectionsByInspector(ctx, toPgUUID(*filter.InspectorID))
	} else {
		return nil, 0, aletheia.Invalid("ProjectID or InspectorID is required")
	}

	if err != nil {
		return nil, 0, aletheia.Internal("Failed to list inspections", err)
	}

	// Apply offset/limit in memory
	total := len(inspections)
	if filter.Offset > 0 && filter.Offset < len(inspections) {
		inspections = inspections[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(inspections) {
		inspections = inspections[:filter.Limit]
	}

	return toDomainInspections(inspections), total, nil
}

func (s *InspectionService) CreateInspection(ctx context.Context, inspection *aletheia.Inspection) error {
	dbInspection, err := s.db.queries.CreateInspection(ctx, database.CreateInspectionParams{
		ProjectID:   toPgUUID(inspection.ProjectID),
		InspectorID: toPgUUID(inspection.InspectorID),
		Status:      database.InspectionStatus(inspection.Status),
	})
	if err != nil {
		if isForeignKeyViolation(err) {
			return aletheia.NotFound("Project or inspector not found")
		}
		return aletheia.Internal("Failed to create inspection", err)
	}

	// Update inspection with generated values
	inspection.ID = fromPgUUID(dbInspection.ID)
	inspection.CreatedAt = fromPgTimestamp(dbInspection.CreatedAt)
	inspection.UpdatedAt = fromPgTimestamp(dbInspection.UpdatedAt)

	return nil
}

func (s *InspectionService) UpdateInspection(ctx context.Context, id uuid.UUID, upd aletheia.InspectionUpdate) (*aletheia.Inspection, error) {
	// Currently only status can be updated
	if upd.Status != nil {
		return s.UpdateInspectionStatus(ctx, id, *upd.Status)
	}

	// No update requested, just return current inspection
	return s.FindInspectionByID(ctx, id)
}

func (s *InspectionService) UpdateInspectionStatus(ctx context.Context, id uuid.UUID, status aletheia.InspectionStatus) (*aletheia.Inspection, error) {
	// Get current inspection to validate status transition
	current, err := s.FindInspectionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate status transition
	if !current.Status.CanTransitionTo(status) {
		return nil, aletheia.Invalid("Invalid status transition from %s to %s", current.Status, status)
	}

	inspection, err := s.db.queries.UpdateInspectionStatus(ctx, database.UpdateInspectionStatusParams{
		ID:     toPgUUID(id),
		Status: database.InspectionStatus(status),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("Inspection not found")
		}
		return nil, aletheia.Internal("Failed to update inspection status", err)
	}

	return toDomainInspection(inspection), nil
}

func (s *InspectionService) DeleteInspection(ctx context.Context, id uuid.UUID) error {
	err := s.db.queries.DeleteInspection(ctx, toPgUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return aletheia.NotFound("Inspection not found")
		}
		return aletheia.Internal("Failed to delete inspection", err)
	}
	return nil
}

func (s *InspectionService) GetInspectionStats(ctx context.Context, id uuid.UUID) (*aletheia.InspectionStats, error) {
	// TODO: This would require custom queries to aggregate stats
	// For now, return empty stats
	return &aletheia.InspectionStats{
		InspectionID: id,
	}, nil
}
