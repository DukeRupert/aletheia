package http

import (
	"log/slog"
	"net/http"

	"github.com/dukerupert/aletheia"
	"github.com/labstack/echo/v4"
)

// CreateOrganizationRequest is the request payload for creating an organization.
type CreateOrganizationRequest struct {
	Name string `json:"name" form:"name" validate:"required,min=2,max=100"`
}

func (s *Server) handleCreateOrganization(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	userID, err := requireUserID(c)
	if err != nil {
		return err
	}

	var req CreateOrganizationRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	org := &aletheia.Organization{
		Name: req.Name,
	}

	if err := s.organizationService.CreateOrganization(ctx, org, userID); err != nil {
		return err
	}

	s.log(c).Info("organization created",
		slog.String("org_id", org.ID.String()),
		slog.String("name", org.Name),
	)

	return RespondCreated(c, org)
}

func (s *Server) handleListOrganizations(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	userID, err := requireUserID(c)
	if err != nil {
		return err
	}

	orgs, err := s.organizationService.FindUserOrganizations(ctx, userID)
	if err != nil {
		return err
	}

	return RespondOK(c, orgs)
}

func (s *Server) handleGetOrganization(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	orgID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	org, err := s.organizationService.FindOrganizationByID(ctx, orgID)
	if err != nil {
		return err
	}

	return RespondOK(c, org)
}

// UpdateOrganizationRequest is the request payload for updating an organization.
type UpdateOrganizationRequest struct {
	Name *string `json:"name" form:"name" validate:"omitempty,min=2,max=100"`
}

func (s *Server) handleUpdateOrganization(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	orgID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	var req UpdateOrganizationRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	org, err := s.organizationService.UpdateOrganization(ctx, orgID, aletheia.OrganizationUpdate{
		Name: req.Name,
	})
	if err != nil {
		return err
	}

	s.log(c).Info("organization updated", slog.String("org_id", org.ID.String()))

	return RespondOK(c, org)
}

func (s *Server) handleDeleteOrganization(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	orgID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	if err := s.organizationService.DeleteOrganization(ctx, orgID); err != nil {
		return err
	}

	s.log(c).Info("organization deleted", slog.String("org_id", orgID.String()))

	return c.NoContent(http.StatusNoContent)
}

// Organization member handlers

func (s *Server) handleListOrganizationMembers(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	orgID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	members, err := s.organizationService.ListMembers(ctx, orgID)
	if err != nil {
		return err
	}

	return RespondOK(c, members)
}

// AddMemberRequest is the request payload for adding a member to an organization.
type AddMemberRequest struct {
	UserID string `json:"user_id" form:"user_id" validate:"required,uuid"`
	Role   string `json:"role" form:"role" validate:"required,oneof=admin member"`
}

func (s *Server) handleAddOrganizationMember(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	orgID, err := requireUUIDParam(c, "id")
	if err != nil {
		return err
	}

	var req AddMemberRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	userID, err := parseUUID(req.UserID)
	if err != nil {
		return err
	}

	role := aletheia.OrganizationRole(req.Role)

	member, err := s.organizationService.AddMember(ctx, orgID, userID, role)
	if err != nil {
		return err
	}

	s.log(c).Info("member added to organization",
		slog.String("org_id", orgID.String()),
		slog.String("user_id", userID.String()),
		slog.String("role", string(role)),
	)

	return RespondCreated(c, member)
}

// UpdateMemberRequest is the request payload for updating a member's role.
type UpdateMemberRequest struct {
	Role string `json:"role" form:"role" validate:"required,oneof=owner admin member"`
}

func (s *Server) handleUpdateOrganizationMember(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	memberID, err := requireUUIDParam(c, "memberId")
	if err != nil {
		return err
	}

	var req UpdateMemberRequest
	if err := bind(c, &req); err != nil {
		return err
	}

	role := aletheia.OrganizationRole(req.Role)

	member, err := s.organizationService.UpdateMemberRole(ctx, memberID, role)
	if err != nil {
		return err
	}

	s.log(c).Info("member role updated",
		slog.String("member_id", memberID.String()),
		slog.String("role", string(role)),
	)

	return RespondOK(c, member)
}

func (s *Server) handleRemoveOrganizationMember(c echo.Context) error {
	ctx, cancel := withTimeout(c)
	defer cancel()

	memberID, err := requireUUIDParam(c, "memberId")
	if err != nil {
		return err
	}

	if err := s.organizationService.RemoveMember(ctx, memberID); err != nil {
		return err
	}

	s.log(c).Info("member removed from organization", slog.String("member_id", memberID.String()))

	return c.NoContent(http.StatusNoContent)
}
