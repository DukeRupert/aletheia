package aletheia

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Organization represents a company or entity that conducts inspections.
type Organization struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// OrganizationRole represents a user's role within an organization.
type OrganizationRole string

const (
	RoleOwner  OrganizationRole = "owner"
	RoleAdmin  OrganizationRole = "admin"
	RoleMember OrganizationRole = "member"
)

// CanManageMembers returns true if the role can add/remove members.
func (r OrganizationRole) CanManageMembers() bool {
	return r == RoleOwner || r == RoleAdmin
}

// CanDelete returns true if the role can delete the organization.
func (r OrganizationRole) CanDelete() bool {
	return r == RoleOwner
}

// OrganizationMember represents a user's membership in an organization.
type OrganizationMember struct {
	ID             uuid.UUID        `json:"id"`
	OrganizationID uuid.UUID        `json:"organizationId"`
	UserID         uuid.UUID        `json:"userId"`
	Role           OrganizationRole `json:"role"`
	CreatedAt      time.Time        `json:"createdAt"`

	// Joined fields (populated by some queries)
	User         *User         `json:"user,omitempty"`
	Organization *Organization `json:"organization,omitempty"`
}

// OrganizationWithRole represents an organization with the user's role in it.
type OrganizationWithRole struct {
	Organization
	Role OrganizationRole `json:"role"`
}

// OrganizationService defines operations for managing organizations.
type OrganizationService interface {
	// FindOrganizationByID retrieves an organization by its ID.
	// Returns ENOTFOUND if the organization does not exist.
	FindOrganizationByID(ctx context.Context, id uuid.UUID) (*Organization, error)

	// FindUserOrganizations retrieves all organizations a user belongs to.
	FindUserOrganizations(ctx context.Context, userID uuid.UUID) ([]*OrganizationWithRole, error)

	// CreateOrganization creates a new organization with the specified user as owner.
	// Returns ECONFLICT if an organization with the same name already exists.
	CreateOrganization(ctx context.Context, org *Organization, ownerID uuid.UUID) error

	// UpdateOrganization updates an existing organization.
	// Returns ENOTFOUND if the organization does not exist.
	// Returns EFORBIDDEN if the user lacks permission.
	UpdateOrganization(ctx context.Context, id uuid.UUID, upd OrganizationUpdate) (*Organization, error)

	// DeleteOrganization deletes an organization and all associated data.
	// Returns ENOTFOUND if the organization does not exist.
	// Returns EFORBIDDEN if the user is not the owner.
	DeleteOrganization(ctx context.Context, id uuid.UUID) error

	// Membership operations

	// GetMembership retrieves a user's membership in an organization.
	// Returns ENOTFOUND if the membership does not exist.
	GetMembership(ctx context.Context, orgID, userID uuid.UUID) (*OrganizationMember, error)

	// AddMember adds a user to an organization with the specified role.
	// Returns ECONFLICT if the user is already a member.
	AddMember(ctx context.Context, orgID, userID uuid.UUID, role OrganizationRole) (*OrganizationMember, error)

	// UpdateMemberRole changes a member's role in an organization.
	// Returns ENOTFOUND if the membership does not exist.
	// Returns EFORBIDDEN if trying to demote the last owner.
	UpdateMemberRole(ctx context.Context, memberID uuid.UUID, role OrganizationRole) (*OrganizationMember, error)

	// RemoveMember removes a user from an organization.
	// Returns ENOTFOUND if the membership does not exist.
	// Returns EFORBIDDEN if trying to remove the last owner.
	RemoveMember(ctx context.Context, memberID uuid.UUID) error

	// ListMembers retrieves all members of an organization.
	ListMembers(ctx context.Context, orgID uuid.UUID) ([]*OrganizationMember, error)

	// Authorization helpers

	// RequireMembership verifies a user is a member of an organization.
	// Returns EFORBIDDEN if not a member or role is insufficient.
	RequireMembership(ctx context.Context, orgID, userID uuid.UUID, allowedRoles ...OrganizationRole) (*OrganizationMember, error)
}

// OrganizationFilter defines criteria for filtering organizations.
type OrganizationFilter struct {
	ID     *uuid.UUID
	Name   *string
	UserID *uuid.UUID // Filter by organizations the user belongs to

	// Pagination
	Offset int
	Limit  int
}

// OrganizationUpdate defines fields that can be updated on an organization.
type OrganizationUpdate struct {
	Name *string
}
