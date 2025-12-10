package aletheia

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SafetyCode represents a configurable safety standard or regulation.
type SafetyCode struct {
	ID            uuid.UUID `json:"id"`
	Code          string    `json:"code"`
	Description   string    `json:"description"`
	Country       string    `json:"country,omitempty"`
	StateProvince string    `json:"stateProvince,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// FullCode returns the code with jurisdiction prefix if applicable.
func (s *SafetyCode) FullCode() string {
	if s.Country != "" && s.StateProvince != "" {
		return s.Country + "/" + s.StateProvince + ": " + s.Code
	}
	if s.Country != "" {
		return s.Country + ": " + s.Code
	}
	return s.Code
}

// SafetyCodeService defines operations for managing safety codes.
type SafetyCodeService interface {
	// FindSafetyCodeByID retrieves a safety code by its ID.
	// Returns ENOTFOUND if the safety code does not exist.
	FindSafetyCodeByID(ctx context.Context, id uuid.UUID) (*SafetyCode, error)

	// FindSafetyCodeByCode retrieves a safety code by its code string.
	// Returns ENOTFOUND if the safety code does not exist.
	FindSafetyCodeByCode(ctx context.Context, code string) (*SafetyCode, error)

	// FindSafetyCodes retrieves safety codes matching the filter criteria.
	// Returns the matching safety codes and total count.
	FindSafetyCodes(ctx context.Context, filter SafetyCodeFilter) ([]*SafetyCode, int, error)

	// CreateSafetyCode creates a new safety code.
	// Returns ECONFLICT if a safety code with the same code already exists.
	CreateSafetyCode(ctx context.Context, safetyCode *SafetyCode) error

	// UpdateSafetyCode updates an existing safety code.
	// Returns ENOTFOUND if the safety code does not exist.
	UpdateSafetyCode(ctx context.Context, id uuid.UUID, upd SafetyCodeUpdate) (*SafetyCode, error)

	// DeleteSafetyCode deletes a safety code.
	// Returns ENOTFOUND if the safety code does not exist.
	// Returns ECONFLICT if the safety code is referenced by violations.
	DeleteSafetyCode(ctx context.Context, id uuid.UUID) error

	// GetAllSafetyCodes retrieves all safety codes for use in AI analysis.
	GetAllSafetyCodes(ctx context.Context) ([]*SafetyCode, error)
}

// SafetyCodeFilter defines criteria for filtering safety codes.
type SafetyCodeFilter struct {
	ID            *uuid.UUID
	Code          *string
	Country       *string
	StateProvince *string
	Search        *string // Search in code and description

	// Pagination
	Offset int
	Limit  int
}

// SafetyCodeUpdate defines fields that can be updated on a safety code.
type SafetyCodeUpdate struct {
	Code          *string
	Description   *string
	Country       *string
	StateProvince *string
}
