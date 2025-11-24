package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/dukerupert/aletheia/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

const (
	// RFC3339Format is the date/time format used for API responses
	RFC3339Format = "2006-01-02T15:04:05Z07:00"
	// DatabaseTimeout is the default timeout for database operations
	DatabaseTimeout = 5 * time.Second
)

// parseUUID converts a string UUID to pgtype.UUID
func parseUUID(s string) (pgtype.UUID, error) {
	var pguuid pgtype.UUID
	err := pguuid.Scan(s)
	return pguuid, err
}

// uuidToPgUUID converts a uuid.UUID to pgtype.UUID
func uuidToPgUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: u,
		Valid: true,
	}
}

// requireOrganizationMembership checks if a user is a member of an organization
// and optionally verifies they have one of the required roles.
// Returns the membership if authorized, error otherwise.
func requireOrganizationMembership(
	ctx context.Context,
	pool *pgxpool.Pool,
	logger *slog.Logger,
	userID uuid.UUID,
	orgID pgtype.UUID,
	allowedRoles ...database.OrganizationRole,
) (*database.OrganizationMember, error) {
	queries := database.New(pool)

	membership, err := queries.GetOrganizationMemberByUserAndOrg(ctx, database.GetOrganizationMemberByUserAndOrgParams{
		OrganizationID: orgID,
		UserID:         uuidToPgUUID(userID),
	})
	if err != nil {
		logger.Warn("authorization check failed",
			slog.String("user_id", userID.String()),
			slog.String("org_id", orgID.String()),
			slog.String("error", err.Error()))
		return nil, echo.NewHTTPError(http.StatusForbidden, "you are not a member of this organization")
	}

	// Check role if specified
	if len(allowedRoles) > 0 {
		hasRole := false
		for _, role := range allowedRoles {
			if membership.Role == role {
				hasRole = true
				break
			}
		}
		if !hasRole {
			// Build string representation of allowed roles
			allowedRolesStr := make([]string, len(allowedRoles))
			for i, role := range allowedRoles {
				allowedRolesStr[i] = string(role)
			}
			logger.Warn("user does not have required role",
				slog.String("user_id", userID.String()),
				slog.String("org_id", orgID.String()),
				slog.String("user_role", string(membership.Role)),
				slog.Any("required_roles", allowedRolesStr))
			return nil, echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
		}
	}

	return &membership, nil
}
