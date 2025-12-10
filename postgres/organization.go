package postgres

import (
	"context"

	"github.com/dukerupert/aletheia"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Compile-time check that OrganizationService implements aletheia.OrganizationService.
var _ aletheia.OrganizationService = (*OrganizationService)(nil)

// OrganizationService implements aletheia.OrganizationService using PostgreSQL.
type OrganizationService struct {
	db *DB
}

func (s *OrganizationService) FindOrganizationByID(ctx context.Context, id uuid.UUID) (*aletheia.Organization, error) {
	org, err := s.db.queries.GetOrganization(ctx, toPgUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("Organization not found")
		}
		return nil, aletheia.Internal("Failed to fetch organization", err)
	}
	return toDomainOrganization(org), nil
}

func (s *OrganizationService) FindUserOrganizations(ctx context.Context, userID uuid.UUID) ([]*aletheia.OrganizationWithRole, error) {
	rows, err := s.db.queries.ListUserOrganizationsWithDetails(ctx, toPgUUID(userID))
	if err != nil {
		return nil, aletheia.Internal("Failed to list user organizations", err)
	}

	result := make([]*aletheia.OrganizationWithRole, len(rows))
	for i, row := range rows {
		result[i] = &aletheia.OrganizationWithRole{
			Organization: aletheia.Organization{
				ID:        fromPgUUID(row.ID),
				Name:      row.Name,
				CreatedAt: fromPgTimestamp(row.CreatedAt),
				UpdatedAt: fromPgTimestamp(row.UpdatedAt),
			},
			Role: aletheia.OrganizationRole(row.Role),
		}
	}
	return result, nil
}

func (s *OrganizationService) CreateOrganization(ctx context.Context, org *aletheia.Organization, ownerID uuid.UUID) error {
	// Create organization
	dbOrg, err := s.db.queries.CreateOrganization(ctx, org.Name)
	if err != nil {
		if isUniqueViolation(err) {
			return aletheia.Conflict("Organization with this name already exists")
		}
		return aletheia.Internal("Failed to create organization", err)
	}

	// Add owner as member
	_, err = s.db.queries.AddOrganizationMember(ctx, database.AddOrganizationMemberParams{
		OrganizationID: dbOrg.ID,
		UserID:         toPgUUID(ownerID),
		Role:           database.OrganizationRoleOwner,
	})
	if err != nil {
		return aletheia.Internal("Failed to add owner to organization", err)
	}

	// Update org with generated values
	org.ID = fromPgUUID(dbOrg.ID)
	org.CreatedAt = fromPgTimestamp(dbOrg.CreatedAt)
	org.UpdatedAt = fromPgTimestamp(dbOrg.UpdatedAt)

	return nil
}

func (s *OrganizationService) UpdateOrganization(ctx context.Context, id uuid.UUID, upd aletheia.OrganizationUpdate) (*aletheia.Organization, error) {
	params := database.UpdateOrganizationParams{
		ID: toPgUUID(id),
	}

	if upd.Name != nil {
		params.Name = *upd.Name
	}

	org, err := s.db.queries.UpdateOrganization(ctx, params)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("Organization not found")
		}
		if isUniqueViolation(err) {
			return nil, aletheia.Conflict("Organization with this name already exists")
		}
		return nil, aletheia.Internal("Failed to update organization", err)
	}

	return toDomainOrganization(org), nil
}

func (s *OrganizationService) DeleteOrganization(ctx context.Context, id uuid.UUID) error {
	err := s.db.queries.DeleteOrganization(ctx, toPgUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return aletheia.NotFound("Organization not found")
		}
		return aletheia.Internal("Failed to delete organization", err)
	}
	return nil
}

func (s *OrganizationService) GetMembership(ctx context.Context, orgID, userID uuid.UUID) (*aletheia.OrganizationMember, error) {
	member, err := s.db.queries.GetOrganizationMemberByUserAndOrg(ctx, database.GetOrganizationMemberByUserAndOrgParams{
		OrganizationID: toPgUUID(orgID),
		UserID:         toPgUUID(userID),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("Membership not found")
		}
		return nil, aletheia.Internal("Failed to fetch membership", err)
	}
	return toDomainOrganizationMember(member), nil
}

func (s *OrganizationService) AddMember(ctx context.Context, orgID, userID uuid.UUID, role aletheia.OrganizationRole) (*aletheia.OrganizationMember, error) {
	member, err := s.db.queries.AddOrganizationMember(ctx, database.AddOrganizationMemberParams{
		OrganizationID: toPgUUID(orgID),
		UserID:         toPgUUID(userID),
		Role:           database.OrganizationRole(role),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, aletheia.Conflict("User is already a member of this organization")
		}
		return nil, aletheia.Internal("Failed to add member", err)
	}
	return toDomainOrganizationMember(member), nil
}

func (s *OrganizationService) UpdateMemberRole(ctx context.Context, memberID uuid.UUID, role aletheia.OrganizationRole) (*aletheia.OrganizationMember, error) {
	member, err := s.db.queries.UpdateOrganizationMemberRole(ctx, database.UpdateOrganizationMemberRoleParams{
		ID:   toPgUUID(memberID),
		Role: database.OrganizationRole(role),
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("Membership not found")
		}
		return nil, aletheia.Internal("Failed to update member role", err)
	}
	return toDomainOrganizationMember(member), nil
}

func (s *OrganizationService) RemoveMember(ctx context.Context, memberID uuid.UUID) error {
	err := s.db.queries.RemoveOrganizationMember(ctx, toPgUUID(memberID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return aletheia.NotFound("Membership not found")
		}
		return aletheia.Internal("Failed to remove member", err)
	}
	return nil
}

func (s *OrganizationService) ListMembers(ctx context.Context, orgID uuid.UUID) ([]*aletheia.OrganizationMember, error) {
	members, err := s.db.queries.ListOrganizationMembers(ctx, toPgUUID(orgID))
	if err != nil {
		return nil, aletheia.Internal("Failed to list members", err)
	}
	return toDomainOrganizationMembers(members), nil
}

func (s *OrganizationService) RequireMembership(ctx context.Context, orgID, userID uuid.UUID, allowedRoles ...aletheia.OrganizationRole) (*aletheia.OrganizationMember, error) {
	member, err := s.GetMembership(ctx, orgID, userID)
	if err != nil {
		if aletheia.IsErrorCode(err, aletheia.ENOTFOUND) {
			return nil, aletheia.Forbidden("Access denied to this organization")
		}
		return nil, err
	}

	// If no roles specified, any membership is sufficient
	if len(allowedRoles) == 0 {
		return member, nil
	}

	// Check if member's role is in allowed roles
	for _, role := range allowedRoles {
		if member.Role == role {
			return member, nil
		}
	}

	return nil, aletheia.Forbidden("Insufficient permissions for this operation")
}
