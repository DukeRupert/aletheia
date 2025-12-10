package mock

import (
	"context"
	"time"

	"github.com/dukerupert/aletheia"
	"github.com/google/uuid"
)

// Compile-time interface check
var _ aletheia.SafetyCodeService = (*SafetyCodeService)(nil)

// SafetyCodeService is a mock implementation of aletheia.SafetyCodeService.
type SafetyCodeService struct {
	FindSafetyCodeByIDFn   func(ctx context.Context, id uuid.UUID) (*aletheia.SafetyCode, error)
	FindSafetyCodeByCodeFn func(ctx context.Context, code string) (*aletheia.SafetyCode, error)
	FindSafetyCodesFn      func(ctx context.Context, filter aletheia.SafetyCodeFilter) ([]*aletheia.SafetyCode, int, error)
	CreateSafetyCodeFn     func(ctx context.Context, safetyCode *aletheia.SafetyCode) error
	UpdateSafetyCodeFn     func(ctx context.Context, id uuid.UUID, upd aletheia.SafetyCodeUpdate) (*aletheia.SafetyCode, error)
	DeleteSafetyCodeFn     func(ctx context.Context, id uuid.UUID) error
	GetAllSafetyCodesFn    func(ctx context.Context) ([]*aletheia.SafetyCode, error)
}

func (s *SafetyCodeService) FindSafetyCodeByID(ctx context.Context, id uuid.UUID) (*aletheia.SafetyCode, error) {
	if s.FindSafetyCodeByIDFn != nil {
		return s.FindSafetyCodeByIDFn(ctx, id)
	}
	return nil, aletheia.NotFound("Safety code not found")
}

func (s *SafetyCodeService) FindSafetyCodeByCode(ctx context.Context, code string) (*aletheia.SafetyCode, error) {
	if s.FindSafetyCodeByCodeFn != nil {
		return s.FindSafetyCodeByCodeFn(ctx, code)
	}
	return nil, aletheia.NotFound("Safety code not found")
}

func (s *SafetyCodeService) FindSafetyCodes(ctx context.Context, filter aletheia.SafetyCodeFilter) ([]*aletheia.SafetyCode, int, error) {
	if s.FindSafetyCodesFn != nil {
		return s.FindSafetyCodesFn(ctx, filter)
	}
	return []*aletheia.SafetyCode{}, 0, nil
}

func (s *SafetyCodeService) CreateSafetyCode(ctx context.Context, safetyCode *aletheia.SafetyCode) error {
	if s.CreateSafetyCodeFn != nil {
		return s.CreateSafetyCodeFn(ctx, safetyCode)
	}
	safetyCode.ID = uuid.New()
	safetyCode.CreatedAt = time.Now()
	safetyCode.UpdatedAt = time.Now()
	return nil
}

func (s *SafetyCodeService) UpdateSafetyCode(ctx context.Context, id uuid.UUID, upd aletheia.SafetyCodeUpdate) (*aletheia.SafetyCode, error) {
	if s.UpdateSafetyCodeFn != nil {
		return s.UpdateSafetyCodeFn(ctx, id, upd)
	}
	return nil, aletheia.NotFound("Safety code not found")
}

func (s *SafetyCodeService) DeleteSafetyCode(ctx context.Context, id uuid.UUID) error {
	if s.DeleteSafetyCodeFn != nil {
		return s.DeleteSafetyCodeFn(ctx, id)
	}
	return nil
}

func (s *SafetyCodeService) GetAllSafetyCodes(ctx context.Context) ([]*aletheia.SafetyCode, error) {
	if s.GetAllSafetyCodesFn != nil {
		return s.GetAllSafetyCodesFn(ctx)
	}
	return []*aletheia.SafetyCode{}, nil
}
