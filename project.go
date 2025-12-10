package aletheia

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Project represents a construction site or building being inspected.
type Project struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organizationId"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	ProjectType    string    `json:"projectType,omitempty"`
	Status         string    `json:"status,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`

	// Address fields
	Address string `json:"address,omitempty"`
	City    string `json:"city,omitempty"`
	State   string `json:"state,omitempty"`
	ZipCode string `json:"zipCode,omitempty"`
	Country string `json:"country,omitempty"`

	// Joined fields (populated by some queries)
	Organization *Organization `json:"organization,omitempty"`
}

// FullAddress returns the formatted full address.
func (p *Project) FullAddress() string {
	if p.Address == "" {
		return ""
	}

	addr := p.Address
	if p.City != "" {
		addr += ", " + p.City
	}
	if p.State != "" {
		addr += ", " + p.State
	}
	if p.ZipCode != "" {
		addr += " " + p.ZipCode
	}
	if p.Country != "" {
		addr += ", " + p.Country
	}
	return addr
}

// ProjectService defines operations for managing projects.
type ProjectService interface {
	// FindProjectByID retrieves a project by its ID.
	// Returns ENOTFOUND if the project does not exist.
	FindProjectByID(ctx context.Context, id uuid.UUID) (*Project, error)

	// FindProjects retrieves projects matching the filter criteria.
	// Returns the matching projects and total count.
	FindProjects(ctx context.Context, filter ProjectFilter) ([]*Project, int, error)

	// CreateProject creates a new project.
	// Returns EFORBIDDEN if the user lacks permission in the organization.
	CreateProject(ctx context.Context, project *Project) error

	// UpdateProject updates an existing project.
	// Returns ENOTFOUND if the project does not exist.
	// Returns EFORBIDDEN if the user lacks permission.
	UpdateProject(ctx context.Context, id uuid.UUID, upd ProjectUpdate) (*Project, error)

	// DeleteProject deletes a project and all associated data.
	// Returns ENOTFOUND if the project does not exist.
	// Returns EFORBIDDEN if the user lacks permission.
	DeleteProject(ctx context.Context, id uuid.UUID) error

	// GetProjectStats retrieves statistics for a project.
	GetProjectStats(ctx context.Context, id uuid.UUID) (*ProjectStats, error)
}

// ProjectFilter defines criteria for filtering projects.
type ProjectFilter struct {
	ID             *uuid.UUID
	OrganizationID *uuid.UUID
	Status         *string
	Search         *string // Search in name and description

	// Pagination
	Offset int
	Limit  int
}

// ProjectUpdate defines fields that can be updated on a project.
type ProjectUpdate struct {
	Name        *string
	Description *string
	ProjectType *string
	Status      *string
	Address     *string
	City        *string
	State       *string
	ZipCode     *string
	Country     *string
}

// ProjectStats contains aggregated statistics for a project.
type ProjectStats struct {
	ProjectID        uuid.UUID `json:"projectId"`
	InspectionCount  int       `json:"inspectionCount"`
	PhotoCount       int       `json:"photoCount"`
	ViolationCount   int       `json:"violationCount"`
	OpenViolations   int       `json:"openViolations"`
	ClosedViolations int       `json:"closedViolations"`
}
