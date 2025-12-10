package mock

import (
	"context"
	"time"

	"github.com/dukerupert/aletheia"
	"github.com/google/uuid"
)

// Compile-time interface check
var _ aletheia.ViolationService = (*ViolationService)(nil)

// ViolationService is a mock implementation of aletheia.ViolationService.
type ViolationService struct {
	FindViolationByIDFn        func(ctx context.Context, id uuid.UUID) (*aletheia.Violation, error)
	FindViolationsFn           func(ctx context.Context, filter aletheia.ViolationFilter) ([]*aletheia.Violation, int, error)
	CreateViolationFn          func(ctx context.Context, violation *aletheia.Violation) error
	CreateViolationsFn         func(ctx context.Context, violations []*aletheia.Violation) error
	UpdateViolationFn          func(ctx context.Context, id uuid.UUID, upd aletheia.ViolationUpdate) (*aletheia.Violation, error)
	ConfirmViolationFn         func(ctx context.Context, id uuid.UUID) (*aletheia.Violation, error)
	DismissViolationFn         func(ctx context.Context, id uuid.UUID) (*aletheia.Violation, error)
	SetViolationPendingFn      func(ctx context.Context, id uuid.UUID) (*aletheia.Violation, error)
	DeleteViolationFn          func(ctx context.Context, id uuid.UUID) error
	GetViolationsByInspectionFn func(ctx context.Context, inspectionID uuid.UUID) ([]*aletheia.Violation, error)
}

func (s *ViolationService) FindViolationByID(ctx context.Context, id uuid.UUID) (*aletheia.Violation, error) {
	if s.FindViolationByIDFn != nil {
		return s.FindViolationByIDFn(ctx, id)
	}
	return nil, aletheia.NotFound("Violation not found")
}

func (s *ViolationService) FindViolations(ctx context.Context, filter aletheia.ViolationFilter) ([]*aletheia.Violation, int, error) {
	if s.FindViolationsFn != nil {
		return s.FindViolationsFn(ctx, filter)
	}
	return []*aletheia.Violation{}, 0, nil
}

func (s *ViolationService) CreateViolation(ctx context.Context, violation *aletheia.Violation) error {
	if s.CreateViolationFn != nil {
		return s.CreateViolationFn(ctx, violation)
	}
	violation.ID = uuid.New()
	violation.CreatedAt = time.Now()
	return nil
}

func (s *ViolationService) CreateViolations(ctx context.Context, violations []*aletheia.Violation) error {
	if s.CreateViolationsFn != nil {
		return s.CreateViolationsFn(ctx, violations)
	}
	for _, v := range violations {
		v.ID = uuid.New()
		v.CreatedAt = time.Now()
	}
	return nil
}

func (s *ViolationService) UpdateViolation(ctx context.Context, id uuid.UUID, upd aletheia.ViolationUpdate) (*aletheia.Violation, error) {
	if s.UpdateViolationFn != nil {
		return s.UpdateViolationFn(ctx, id, upd)
	}
	return nil, aletheia.NotFound("Violation not found")
}

func (s *ViolationService) ConfirmViolation(ctx context.Context, id uuid.UUID) (*aletheia.Violation, error) {
	if s.ConfirmViolationFn != nil {
		return s.ConfirmViolationFn(ctx, id)
	}
	return &aletheia.Violation{
		ID:     id,
		Status: aletheia.ViolationStatusConfirmed,
	}, nil
}

func (s *ViolationService) DismissViolation(ctx context.Context, id uuid.UUID) (*aletheia.Violation, error) {
	if s.DismissViolationFn != nil {
		return s.DismissViolationFn(ctx, id)
	}
	return &aletheia.Violation{
		ID:     id,
		Status: aletheia.ViolationStatusDismissed,
	}, nil
}

func (s *ViolationService) SetViolationPending(ctx context.Context, id uuid.UUID) (*aletheia.Violation, error) {
	if s.SetViolationPendingFn != nil {
		return s.SetViolationPendingFn(ctx, id)
	}
	return &aletheia.Violation{
		ID:     id,
		Status: aletheia.ViolationStatusPending,
	}, nil
}

func (s *ViolationService) DeleteViolation(ctx context.Context, id uuid.UUID) error {
	if s.DeleteViolationFn != nil {
		return s.DeleteViolationFn(ctx, id)
	}
	return nil
}

func (s *ViolationService) GetViolationsByInspection(ctx context.Context, inspectionID uuid.UUID) ([]*aletheia.Violation, error) {
	if s.GetViolationsByInspectionFn != nil {
		return s.GetViolationsByInspectionFn(ctx, inspectionID)
	}
	return []*aletheia.Violation{}, nil
}
