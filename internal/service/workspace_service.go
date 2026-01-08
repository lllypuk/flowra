package service

import (
	"context"

	wsapp "github.com/lllypuk/flowra/internal/application/workspace"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
)

// Compile-time assertion that WorkspaceService implements httphandler.WorkspaceService.
var _ httphandler.WorkspaceService = (*WorkspaceService)(nil)

// WorkspaceServiceCommandRepository defines interface for commands (change state) workspace prostranstv.
// interface declared on the consumer side according to principles Go interface design.
type WorkspaceServiceCommandRepository interface {
	// Save saves workspace space (creation or update)
	Save(ctx context.Context, ws *workspace.Workspace) error

	// Delete udalyaet workspace space
	Delete(ctx context.Context, id uuid.UUID) error

	// AddMember adds chlena in workspace
	AddMember(ctx context.Context, member *workspace.Member) error
}

// WorkspaceServiceQueryRepository defines interface for zaprosov (only reading) workspace prostranstv.
// interface declared on the consumer side according to principles Go interface design.
type WorkspaceServiceQueryRepository interface {
	// FindByID finds workspace space po ID
	FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)

	// ListWorkspacesByUser returns workspaces, in kotoryh user is chlenom
	ListWorkspacesByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*workspace.Workspace, error)

	// CountWorkspacesByUser returns count workspaces user
	CountWorkspacesByUser(ctx context.Context, userID uuid.UUID) (int, error)

	// CountMembers returns count chlenov workspace
	CountMembers(ctx context.Context, workspaceID uuid.UUID) (int, error)
}

// CreateWorkspaceUseCase defines interface for use case creating workspace.
type CreateWorkspaceUseCase interface {
	Execute(ctx context.Context, cmd wsapp.CreateWorkspaceCommand) (wsapp.Result, error)
}

// GetWorkspaceUseCase defines interface for use case receiv workspace.
type GetWorkspaceUseCase interface {
	Execute(ctx context.Context, query wsapp.GetWorkspaceQuery) (wsapp.Result, error)
}

// UpdateWorkspaceUseCase defines interface for use case updating workspace.
type UpdateWorkspaceUseCase interface {
	Execute(ctx context.Context, cmd wsapp.UpdateWorkspaceCommand) (wsapp.Result, error)
}

// WorkspaceService realizuet httphandler.WorkspaceService
type WorkspaceService struct {
	// Use cases
	createUC CreateWorkspaceUseCase
	getUC    GetWorkspaceUseCase
	updateUC UpdateWorkspaceUseCase

	// Repositories (for operatsiy bez use case)
	commandRepo WorkspaceServiceCommandRepository
	queryRepo   WorkspaceServiceQueryRepository
}

// WorkspaceServiceConfig contains zavisimosti for WorkspaceService.
type WorkspaceServiceConfig struct {
	CreateUC    CreateWorkspaceUseCase
	GetUC       GetWorkspaceUseCase
	UpdateUC    UpdateWorkspaceUseCase
	CommandRepo WorkspaceServiceCommandRepository
	QueryRepo   WorkspaceServiceQueryRepository
}

// NewWorkspaceService sozdayot New WorkspaceService.
func NewWorkspaceService(cfg WorkspaceServiceConfig) *WorkspaceService {
	return &WorkspaceService{
		createUC:    cfg.CreateUC,
		getUC:       cfg.GetUC,
		updateUC:    cfg.UpdateUC,
		commandRepo: cfg.CommandRepo,
		queryRepo:   cfg.QueryRepo,
	}
}

// CreateWorkspace sozdayot New workspace.
func (s *WorkspaceService) CreateWorkspace(
	ctx context.Context,
	ownerID uuid.UUID,
	name, description string,
) (*workspace.Workspace, error) {
	result, err := s.createUC.Execute(ctx, wsapp.CreateWorkspaceCommand{
		Name:        name,
		Description: description,
		CreatedBy:   ownerID,
	})
	if err != nil {
		return nil, err
	}

	return result.Value, nil
}

// GetWorkspace returns workspace po ID.
func (s *WorkspaceService) GetWorkspace(
	ctx context.Context,
	id uuid.UUID,
) (*workspace.Workspace, error) {
	result, err := s.getUC.Execute(ctx, wsapp.GetWorkspaceQuery{
		WorkspaceID: id,
	})
	if err != nil {
		return nil, err
	}

	return result.Value, nil
}

// ListUserWorkspaces returns list workspaces user.
// uses repository napryamuyu, tak as ListUserWorkspacesUseCase
// trebuet dopolnitelnyh methods Keycloak, kotorye poka not realizovany.
func (s *WorkspaceService) ListUserWorkspaces(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*workspace.Workspace, int, error) {
	workspaces, err := s.queryRepo.ListWorkspacesByUser(ctx, userID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.queryRepo.CountWorkspacesByUser(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	return workspaces, total, nil
}

// UpdateWorkspace obnovlyaet workspace.
func (s *WorkspaceService) UpdateWorkspace(
	ctx context.Context,
	id uuid.UUID,
	name, _ string,
) (*workspace.Workspace, error) {
	result, err := s.updateUC.Execute(ctx, wsapp.UpdateWorkspaceCommand{
		WorkspaceID: id,
		Name:        name,
		UpdatedBy:   uuid.NewUUID(), // TODO: get from context avtorizatsii
	})
	if err != nil {
		return nil, err
	}

	return result.Value, nil
}

// DeleteWorkspace udalyaet workspace.
// Use case for delete poka not realizovan, ispolzuem repository napryamuyu.
func (s *WorkspaceService) DeleteWorkspace(
	ctx context.Context,
	id uuid.UUID,
) error {
	return s.commandRepo.Delete(ctx, id)
}

// GetMemberCount returns count participants workspace.
func (s *WorkspaceService) GetMemberCount(
	ctx context.Context,
	workspaceID uuid.UUID,
) (int, error) {
	return s.queryRepo.CountMembers(ctx, workspaceID)
}
