package postgres

import (
	"context"

	"github.com/dukerupert/aletheia"
	"github.com/dukerupert/aletheia/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Compile-time check that ProjectService implements aletheia.ProjectService.
var _ aletheia.ProjectService = (*ProjectService)(nil)

// ProjectService implements aletheia.ProjectService using PostgreSQL.
type ProjectService struct {
	db *DB
}

func (s *ProjectService) FindProjectByID(ctx context.Context, id uuid.UUID) (*aletheia.Project, error) {
	project, err := s.db.queries.GetProject(ctx, toPgUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("Project not found")
		}
		return nil, aletheia.Internal("Failed to fetch project", err)
	}
	return toDomainProject(project), nil
}

func (s *ProjectService) FindProjects(ctx context.Context, filter aletheia.ProjectFilter) ([]*aletheia.Project, int, error) {
	// Currently sqlc only supports filtering by organization_id
	if filter.OrganizationID == nil {
		return nil, 0, aletheia.Invalid("Organization ID is required")
	}

	projects, err := s.db.queries.ListProjects(ctx, toPgUUID(*filter.OrganizationID))
	if err != nil {
		return nil, 0, aletheia.Internal("Failed to list projects", err)
	}

	// Apply offset/limit in memory
	total := len(projects)
	if filter.Offset > 0 && filter.Offset < len(projects) {
		projects = projects[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(projects) {
		projects = projects[:filter.Limit]
	}

	return toDomainProjects(projects), total, nil
}

func (s *ProjectService) CreateProject(ctx context.Context, project *aletheia.Project) error {
	dbProject, err := s.db.queries.CreateProject(ctx, database.CreateProjectParams{
		OrganizationID: toPgUUID(project.OrganizationID),
		Name:           project.Name,
		Description:    toPgText(project.Description),
		ProjectType:    toPgText(project.ProjectType),
		Address:        toPgText(project.Address),
		City:           toPgText(project.City),
		State:          toPgText(project.State),
		ZipCode:        toPgText(project.ZipCode),
		Country:        toPgText(project.Country),
	})
	if err != nil {
		if isForeignKeyViolation(err) {
			return aletheia.NotFound("Organization not found")
		}
		return aletheia.Internal("Failed to create project", err)
	}

	// Update project with generated values
	project.ID = fromPgUUID(dbProject.ID)
	project.Status = fromPgText(dbProject.Status)
	project.CreatedAt = fromPgTimestamp(dbProject.CreatedAt)
	project.UpdatedAt = fromPgTimestamp(dbProject.UpdatedAt)

	return nil
}

func (s *ProjectService) UpdateProject(ctx context.Context, id uuid.UUID, upd aletheia.ProjectUpdate) (*aletheia.Project, error) {
	params := database.UpdateProjectParams{
		ID: toPgUUID(id),
	}

	if upd.Name != nil {
		params.Name = toPgText(*upd.Name)
	}
	if upd.Description != nil {
		params.Description = toPgText(*upd.Description)
	}
	if upd.ProjectType != nil {
		params.ProjectType = toPgText(*upd.ProjectType)
	}
	if upd.Status != nil {
		params.Status = toPgText(*upd.Status)
	}
	if upd.Address != nil {
		params.Address = toPgText(*upd.Address)
	}
	if upd.City != nil {
		params.City = toPgText(*upd.City)
	}
	if upd.State != nil {
		params.State = toPgText(*upd.State)
	}
	if upd.ZipCode != nil {
		params.ZipCode = toPgText(*upd.ZipCode)
	}
	if upd.Country != nil {
		params.Country = toPgText(*upd.Country)
	}

	project, err := s.db.queries.UpdateProject(ctx, params)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, aletheia.NotFound("Project not found")
		}
		return nil, aletheia.Internal("Failed to update project", err)
	}

	return toDomainProject(project), nil
}

func (s *ProjectService) DeleteProject(ctx context.Context, id uuid.UUID) error {
	err := s.db.queries.DeleteProject(ctx, toPgUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return aletheia.NotFound("Project not found")
		}
		return aletheia.Internal("Failed to delete project", err)
	}
	return nil
}

func (s *ProjectService) GetProjectStats(ctx context.Context, id uuid.UUID) (*aletheia.ProjectStats, error) {
	// TODO: This would require custom queries to aggregate stats
	// For now, return empty stats
	return &aletheia.ProjectStats{
		ProjectID: id,
	}, nil
}
