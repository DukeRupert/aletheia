package aletheia

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Inspection represents a specific inspection event at a project.
type Inspection struct {
	ID          uuid.UUID        `json:"id"`
	ProjectID   uuid.UUID        `json:"projectId"`
	InspectorID uuid.UUID        `json:"inspectorId"`
	Status      InspectionStatus `json:"status"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`

	// Joined fields (populated by some queries)
	Project   *Project `json:"project,omitempty"`
	Inspector *User    `json:"inspector,omitempty"`
}

// InspectionStatus represents the status of an inspection.
type InspectionStatus string

const (
	InspectionStatusDraft      InspectionStatus = "draft"
	InspectionStatusInProgress InspectionStatus = "in_progress"
	InspectionStatusCompleted  InspectionStatus = "completed"
)

// IsEditable returns true if the inspection can still be modified.
func (s InspectionStatus) IsEditable() bool {
	return s == InspectionStatusDraft || s == InspectionStatusInProgress
}

// CanTransitionTo returns true if this status can transition to the target status.
func (s InspectionStatus) CanTransitionTo(target InspectionStatus) bool {
	switch s {
	case InspectionStatusDraft:
		return target == InspectionStatusInProgress || target == InspectionStatusCompleted
	case InspectionStatusInProgress:
		return target == InspectionStatusCompleted
	case InspectionStatusCompleted:
		return false // Cannot transition from completed
	default:
		return false
	}
}

// InspectionService defines operations for managing inspections.
type InspectionService interface {
	// FindInspectionByID retrieves an inspection by its ID.
	// Returns ENOTFOUND if the inspection does not exist.
	FindInspectionByID(ctx context.Context, id uuid.UUID) (*Inspection, error)

	// FindInspections retrieves inspections matching the filter criteria.
	// Returns the matching inspections and total count.
	FindInspections(ctx context.Context, filter InspectionFilter) ([]*Inspection, int, error)

	// CreateInspection creates a new inspection.
	// Returns EFORBIDDEN if the user lacks permission in the organization.
	CreateInspection(ctx context.Context, inspection *Inspection) error

	// UpdateInspection updates an existing inspection.
	// Returns ENOTFOUND if the inspection does not exist.
	// Returns EFORBIDDEN if the user lacks permission.
	UpdateInspection(ctx context.Context, id uuid.UUID, upd InspectionUpdate) (*Inspection, error)

	// UpdateInspectionStatus changes the status of an inspection.
	// Returns EINVALID if the status transition is not allowed.
	UpdateInspectionStatus(ctx context.Context, id uuid.UUID, status InspectionStatus) (*Inspection, error)

	// DeleteInspection deletes an inspection and all associated data.
	// Returns ENOTFOUND if the inspection does not exist.
	// Returns EFORBIDDEN if the user lacks permission.
	DeleteInspection(ctx context.Context, id uuid.UUID) error

	// GetInspectionStats retrieves statistics for an inspection.
	GetInspectionStats(ctx context.Context, id uuid.UUID) (*InspectionStats, error)
}

// InspectionFilter defines criteria for filtering inspections.
type InspectionFilter struct {
	ID          *uuid.UUID
	ProjectID   *uuid.UUID
	InspectorID *uuid.UUID
	Status      *InspectionStatus

	// Pagination
	Offset int
	Limit  int
}

// InspectionUpdate defines fields that can be updated on an inspection.
type InspectionUpdate struct {
	Status *InspectionStatus
}

// InspectionStats contains aggregated statistics for an inspection.
type InspectionStats struct {
	InspectionID     uuid.UUID `json:"inspectionId"`
	PhotoCount       int       `json:"photoCount"`
	ViolationCount   int       `json:"violationCount"`
	PendingCount     int       `json:"pendingCount"`
	ConfirmedCount   int       `json:"confirmedCount"`
	DismissedCount   int       `json:"dismissedCount"`
	CriticalCount    int       `json:"criticalCount"`
	HighCount        int       `json:"highCount"`
	MediumCount      int       `json:"mediumCount"`
	LowCount         int       `json:"lowCount"`
}
