package mock

import (
	"context"
	"time"

	"github.com/dukerupert/aletheia"
	"github.com/google/uuid"
)

// Compile-time interface check
var _ aletheia.InspectionService = (*InspectionService)(nil)

// InspectionService is a mock implementation of aletheia.InspectionService.
type InspectionService struct {
	FindInspectionByIDFn     func(ctx context.Context, id uuid.UUID) (*aletheia.Inspection, error)
	FindInspectionsFn        func(ctx context.Context, filter aletheia.InspectionFilter) ([]*aletheia.Inspection, int, error)
	CreateInspectionFn       func(ctx context.Context, inspection *aletheia.Inspection) error
	UpdateInspectionFn       func(ctx context.Context, id uuid.UUID, upd aletheia.InspectionUpdate) (*aletheia.Inspection, error)
	UpdateInspectionStatusFn func(ctx context.Context, id uuid.UUID, status aletheia.InspectionStatus) (*aletheia.Inspection, error)
	DeleteInspectionFn       func(ctx context.Context, id uuid.UUID) error
	GetInspectionStatsFn     func(ctx context.Context, id uuid.UUID) (*aletheia.InspectionStats, error)
}

func (s *InspectionService) FindInspectionByID(ctx context.Context, id uuid.UUID) (*aletheia.Inspection, error) {
	if s.FindInspectionByIDFn != nil {
		return s.FindInspectionByIDFn(ctx, id)
	}
	return nil, aletheia.NotFound("Inspection not found")
}

func (s *InspectionService) FindInspections(ctx context.Context, filter aletheia.InspectionFilter) ([]*aletheia.Inspection, int, error) {
	if s.FindInspectionsFn != nil {
		return s.FindInspectionsFn(ctx, filter)
	}
	return []*aletheia.Inspection{}, 0, nil
}

func (s *InspectionService) CreateInspection(ctx context.Context, inspection *aletheia.Inspection) error {
	if s.CreateInspectionFn != nil {
		return s.CreateInspectionFn(ctx, inspection)
	}
	inspection.ID = uuid.New()
	inspection.CreatedAt = time.Now()
	inspection.UpdatedAt = time.Now()
	return nil
}

func (s *InspectionService) UpdateInspection(ctx context.Context, id uuid.UUID, upd aletheia.InspectionUpdate) (*aletheia.Inspection, error) {
	if s.UpdateInspectionFn != nil {
		return s.UpdateInspectionFn(ctx, id, upd)
	}
	return nil, aletheia.NotFound("Inspection not found")
}

func (s *InspectionService) UpdateInspectionStatus(ctx context.Context, id uuid.UUID, status aletheia.InspectionStatus) (*aletheia.Inspection, error) {
	if s.UpdateInspectionStatusFn != nil {
		return s.UpdateInspectionStatusFn(ctx, id, status)
	}
	return nil, aletheia.NotFound("Inspection not found")
}

func (s *InspectionService) DeleteInspection(ctx context.Context, id uuid.UUID) error {
	if s.DeleteInspectionFn != nil {
		return s.DeleteInspectionFn(ctx, id)
	}
	return nil
}

func (s *InspectionService) GetInspectionStats(ctx context.Context, id uuid.UUID) (*aletheia.InspectionStats, error) {
	if s.GetInspectionStatsFn != nil {
		return s.GetInspectionStatsFn(ctx, id)
	}
	return &aletheia.InspectionStats{
		InspectionID: id,
	}, nil
}
