package aletheia

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Violation represents a detected safety violation in a photo.
type Violation struct {
	ID              uuid.UUID       `json:"id"`
	PhotoID         uuid.UUID       `json:"photoId"`
	SafetyCodeID    uuid.UUID       `json:"safetyCodeId,omitempty"`
	Description     string          `json:"description"`
	Severity        Severity        `json:"severity"`
	Status          ViolationStatus `json:"status"`
	ConfidenceScore float64         `json:"confidenceScore,omitempty"`
	Location        string          `json:"location,omitempty"`
	CreatedAt       time.Time       `json:"createdAt"`

	// Joined fields (populated by some queries)
	Photo      *Photo      `json:"photo,omitempty"`
	SafetyCode *SafetyCode `json:"safetyCode,omitempty"`
}

// Severity represents the severity level of a violation.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// Weight returns a numeric weight for sorting by severity.
func (s Severity) Weight() int {
	switch s {
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	default:
		return 0
	}
}

// ViolationStatus represents the status of a detected violation.
type ViolationStatus string

const (
	ViolationStatusPending   ViolationStatus = "pending"
	ViolationStatusConfirmed ViolationStatus = "confirmed"
	ViolationStatusDismissed ViolationStatus = "dismissed"
)

// IsResolved returns true if the violation has been reviewed.
func (s ViolationStatus) IsResolved() bool {
	return s == ViolationStatusConfirmed || s == ViolationStatusDismissed
}

// ViolationService defines operations for managing violations.
type ViolationService interface {
	// FindViolationByID retrieves a violation by its ID.
	// Returns ENOTFOUND if the violation does not exist.
	FindViolationByID(ctx context.Context, id uuid.UUID) (*Violation, error)

	// FindViolations retrieves violations matching the filter criteria.
	// Returns the matching violations and total count.
	FindViolations(ctx context.Context, filter ViolationFilter) ([]*Violation, int, error)

	// CreateViolation creates a new violation (for manual entry).
	CreateViolation(ctx context.Context, violation *Violation) error

	// CreateViolations creates multiple violations (for AI detection results).
	CreateViolations(ctx context.Context, violations []*Violation) error

	// UpdateViolation updates an existing violation.
	// Returns ENOTFOUND if the violation does not exist.
	UpdateViolation(ctx context.Context, id uuid.UUID, upd ViolationUpdate) (*Violation, error)

	// ConfirmViolation marks a violation as confirmed.
	// Returns ENOTFOUND if the violation does not exist.
	ConfirmViolation(ctx context.Context, id uuid.UUID) (*Violation, error)

	// DismissViolation marks a violation as dismissed (false positive).
	// Returns ENOTFOUND if the violation does not exist.
	DismissViolation(ctx context.Context, id uuid.UUID) (*Violation, error)

	// SetViolationPending resets a violation status to pending.
	// Returns ENOTFOUND if the violation does not exist.
	SetViolationPending(ctx context.Context, id uuid.UUID) (*Violation, error)

	// DeleteViolation deletes a violation.
	// Returns ENOTFOUND if the violation does not exist.
	DeleteViolation(ctx context.Context, id uuid.UUID) error

	// GetViolationsByInspection retrieves all violations for an inspection.
	GetViolationsByInspection(ctx context.Context, inspectionID uuid.UUID) ([]*Violation, error)
}

// ViolationFilter defines criteria for filtering violations.
type ViolationFilter struct {
	ID           *uuid.UUID
	PhotoID      *uuid.UUID
	InspectionID *uuid.UUID
	SafetyCodeID *uuid.UUID
	Status       *ViolationStatus
	Severity     *Severity

	// Pagination
	Offset int
	Limit  int
}

// ViolationUpdate defines fields that can be updated on a violation.
type ViolationUpdate struct {
	Description  *string
	Severity     *Severity
	Status       *ViolationStatus
	SafetyCodeID *uuid.UUID
	Location     *string
}
