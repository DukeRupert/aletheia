package mock

import (
	"context"
	"time"

	"github.com/dukerupert/aletheia"
	"github.com/google/uuid"
)

// Compile-time interface check
var _ aletheia.OrganizationService = (*OrganizationService)(nil)

// OrganizationService is a mock implementation of aletheia.OrganizationService.
type OrganizationService struct {
	FindOrganizationByIDFn   func(ctx context.Context, id uuid.UUID) (*aletheia.Organization, error)
	FindUserOrganizationsFn  func(ctx context.Context, userID uuid.UUID) ([]*aletheia.OrganizationWithRole, error)
	CreateOrganizationFn     func(ctx context.Context, org *aletheia.Organization, ownerID uuid.UUID) error
	UpdateOrganizationFn     func(ctx context.Context, id uuid.UUID, upd aletheia.OrganizationUpdate) (*aletheia.Organization, error)
	DeleteOrganizationFn     func(ctx context.Context, id uuid.UUID) error
	GetMembershipFn          func(ctx context.Context, orgID, userID uuid.UUID) (*aletheia.OrganizationMember, error)
	AddMemberFn              func(ctx context.Context, orgID, userID uuid.UUID, role aletheia.OrganizationRole) (*aletheia.OrganizationMember, error)
	UpdateMemberRoleFn       func(ctx context.Context, memberID uuid.UUID, role aletheia.OrganizationRole) (*aletheia.OrganizationMember, error)
	RemoveMemberFn           func(ctx context.Context, memberID uuid.UUID) error
	ListMembersFn            func(ctx context.Context, orgID uuid.UUID) ([]*aletheia.OrganizationMember, error)
	RequireMembershipFn      func(ctx context.Context, orgID, userID uuid.UUID, allowedRoles ...aletheia.OrganizationRole) (*aletheia.OrganizationMember, error)
}

func (s *OrganizationService) FindOrganizationByID(ctx context.Context, id uuid.UUID) (*aletheia.Organization, error) {
	if s.FindOrganizationByIDFn != nil {
		return s.FindOrganizationByIDFn(ctx, id)
	}
	return nil, aletheia.NotFound("Organization not found")
}

func (s *OrganizationService) FindUserOrganizations(ctx context.Context, userID uuid.UUID) ([]*aletheia.OrganizationWithRole, error) {
	if s.FindUserOrganizationsFn != nil {
		return s.FindUserOrganizationsFn(ctx, userID)
	}
	return []*aletheia.OrganizationWithRole{}, nil
}

func (s *OrganizationService) CreateOrganization(ctx context.Context, org *aletheia.Organization, ownerID uuid.UUID) error {
	if s.CreateOrganizationFn != nil {
		return s.CreateOrganizationFn(ctx, org, ownerID)
	}
	org.ID = uuid.New()
	org.CreatedAt = time.Now()
	org.UpdatedAt = time.Now()
	return nil
}

func (s *OrganizationService) UpdateOrganization(ctx context.Context, id uuid.UUID, upd aletheia.OrganizationUpdate) (*aletheia.Organization, error) {
	if s.UpdateOrganizationFn != nil {
		return s.UpdateOrganizationFn(ctx, id, upd)
	}
	return nil, aletheia.NotFound("Organization not found")
}

func (s *OrganizationService) DeleteOrganization(ctx context.Context, id uuid.UUID) error {
	if s.DeleteOrganizationFn != nil {
		return s.DeleteOrganizationFn(ctx, id)
	}
	return nil
}

func (s *OrganizationService) GetMembership(ctx context.Context, orgID, userID uuid.UUID) (*aletheia.OrganizationMember, error) {
	if s.GetMembershipFn != nil {
		return s.GetMembershipFn(ctx, orgID, userID)
	}
	return nil, aletheia.NotFound("Membership not found")
}

func (s *OrganizationService) AddMember(ctx context.Context, orgID, userID uuid.UUID, role aletheia.OrganizationRole) (*aletheia.OrganizationMember, error) {
	if s.AddMemberFn != nil {
		return s.AddMemberFn(ctx, orgID, userID, role)
	}
	return &aletheia.OrganizationMember{
		ID:             uuid.New(),
		OrganizationID: orgID,
		UserID:         userID,
		Role:           role,
		CreatedAt:      time.Now(),
	}, nil
}

func (s *OrganizationService) UpdateMemberRole(ctx context.Context, memberID uuid.UUID, role aletheia.OrganizationRole) (*aletheia.OrganizationMember, error) {
	if s.UpdateMemberRoleFn != nil {
		return s.UpdateMemberRoleFn(ctx, memberID, role)
	}
	return nil, aletheia.NotFound("Member not found")
}

func (s *OrganizationService) RemoveMember(ctx context.Context, memberID uuid.UUID) error {
	if s.RemoveMemberFn != nil {
		return s.RemoveMemberFn(ctx, memberID)
	}
	return nil
}

func (s *OrganizationService) ListMembers(ctx context.Context, orgID uuid.UUID) ([]*aletheia.OrganizationMember, error) {
	if s.ListMembersFn != nil {
		return s.ListMembersFn(ctx, orgID)
	}
	return []*aletheia.OrganizationMember{}, nil
}

func (s *OrganizationService) RequireMembership(ctx context.Context, orgID, userID uuid.UUID, allowedRoles ...aletheia.OrganizationRole) (*aletheia.OrganizationMember, error) {
	if s.RequireMembershipFn != nil {
		return s.RequireMembershipFn(ctx, orgID, userID, allowedRoles...)
	}
	return &aletheia.OrganizationMember{
		ID:             uuid.New(),
		OrganizationID: orgID,
		UserID:         userID,
		Role:           aletheia.RoleMember,
		CreatedAt:      time.Now(),
	}, nil
}
