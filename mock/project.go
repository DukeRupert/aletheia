package mock

import (
	"context"
	"time"

	"github.com/dukerupert/aletheia"
	"github.com/google/uuid"
)

// Compile-time interface check
var _ aletheia.ProjectService = (*ProjectService)(nil)

// ProjectService is a mock implementation of aletheia.ProjectService.
type ProjectService struct {
	FindProjectByIDFn func(ctx context.Context, id uuid.UUID) (*aletheia.Project, error)
	FindProjectsFn    func(ctx context.Context, filter aletheia.ProjectFilter) ([]*aletheia.Project, int, error)
	CreateProjectFn   func(ctx context.Context, project *aletheia.Project) error
	UpdateProjectFn   func(ctx context.Context, id uuid.UUID, upd aletheia.ProjectUpdate) (*aletheia.Project, error)
	DeleteProjectFn   func(ctx context.Context, id uuid.UUID) error
	GetProjectStatsFn func(ctx context.Context, id uuid.UUID) (*aletheia.ProjectStats, error)
}

func (s *ProjectService) FindProjectByID(ctx context.Context, id uuid.UUID) (*aletheia.Project, error) {
	if s.FindProjectByIDFn != nil {
		return s.FindProjectByIDFn(ctx, id)
	}
	return nil, aletheia.NotFound("Project not found")
}

func (s *ProjectService) FindProjects(ctx context.Context, filter aletheia.ProjectFilter) ([]*aletheia.Project, int, error) {
	if s.FindProjectsFn != nil {
		return s.FindProjectsFn(ctx, filter)
	}
	return []*aletheia.Project{}, 0, nil
}

func (s *ProjectService) CreateProject(ctx context.Context, project *aletheia.Project) error {
	if s.CreateProjectFn != nil {
		return s.CreateProjectFn(ctx, project)
	}
	project.ID = uuid.New()
	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()
	return nil
}

func (s *ProjectService) UpdateProject(ctx context.Context, id uuid.UUID, upd aletheia.ProjectUpdate) (*aletheia.Project, error) {
	if s.UpdateProjectFn != nil {
		return s.UpdateProjectFn(ctx, id, upd)
	}
	return nil, aletheia.NotFound("Project not found")
}

func (s *ProjectService) DeleteProject(ctx context.Context, id uuid.UUID) error {
	if s.DeleteProjectFn != nil {
		return s.DeleteProjectFn(ctx, id)
	}
	return nil
}

func (s *ProjectService) GetProjectStats(ctx context.Context, id uuid.UUID) (*aletheia.ProjectStats, error) {
	if s.GetProjectStatsFn != nil {
		return s.GetProjectStatsFn(ctx, id)
	}
	return &aletheia.ProjectStats{
		ProjectID: id,
	}, nil
}
