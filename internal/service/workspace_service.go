package service

import (
	"context"

	wsapp "github.com/lllypuk/flowra/internal/application/workspace"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
	httphandler "github.com/lllypuk/flowra/internal/handler/HTTP"
)

// Compile-time assertion that WorkspaceService implements httphandler.WorkspaceService.
var _ httphandler.WorkspaceService = (*WorkspaceService)(nil)

// WorkspaceServiceCommandRepository defines interface for commands (change state) workspaceих пространств.
// interface declared on the consumer side according to principles Go interface design.
type WorkspaceServiceCommandRepository interface {
	// Save saves workspaceее пространство (creation or update)
	Save(ctx context.Context, ws *workspace.Workspace) error

	// Delete удаляет workspaceее пространство
	Delete(ctx context.Context, id uuid.UUID) error

	// AddMember добавляет члена in workspace
	AddMember(ctx context.Context, member *workspace.Member) error
}

// WorkspaceServiceQueryRepository defines interface for запросов (only reading) workspaceих пространств.
// interface declared on the consumer side according to principles Go interface design.
type WorkspaceServiceQueryRepository interface {
	// FindByID finds workspaceее пространство по ID
	FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)

	// ListWorkspacesByUser returns workspaces, in которых userель is членом
	ListWorkspacesByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*workspace.Workspace, error)

	// CountWorkspacesByUser returns count workspaces user
	CountWorkspacesByUser(ctx context.Context, userID uuid.UUID) (int, error)

	// CountMembers returns count членов workspace
	CountMembers(ctx context.Context, workspaceID uuid.UUID) (int, error)
}

// CreateWorkspaceUseCase defines interface for use case creating workspace.
type CreateWorkspaceUseCase interface {
	Execute(ctx context.Context, cmd wsapp.CreateWorkspaceCommand) (wsapp.Result, error)
}

// GetWorkspaceUseCase defines interface for use case receivения workspace.
type GetWorkspaceUseCase interface {
	Execute(ctx context.Context, query wsapp.GetWorkspaceQuery) (wsapp.Result, error)
}

// UpdateWorkspaceUseCase defines interface for use case updating workspace.
type UpdateWorkspaceUseCase interface {
	Execute(ctx context.Context, cmd wsapp.UpdateWorkspaceCommand) (wsapp.Result, error)
}

// WorkspaceService реализует httphandler.WorkspaceService
type WorkspaceService struct {
	// Use cases
	createUC CreateWorkspaceUseCase
	getUC    GetWorkspaceUseCase
	updateUC UpdateWorkspaceUseCase

	// Repositories (for операций без use case)
	commandRepo WorkspaceServiceCommandRepository
	queryRepo   WorkspaceServiceQueryRepository
}

// WorkspaceServiceConfig contains зависимости for WorkspaceService.
type WorkspaceServiceConfig struct {
	CreateUC    CreateWorkspaceUseCase
	GetUC       GetWorkspaceUseCase
	UpdateUC    UpdateWorkspaceUseCase
	CommandRepo WorkspaceServiceCommandRepository
	QueryRepo   WorkspaceServiceQueryRepository
}

// NewWorkspaceService создаёт New WorkspaceService.
func NewWorkspaceService(cfg WorkspaceServiceConfig) *WorkspaceService {
	return &WorkspaceService{
		createUC:    cfg.CreateUC,
		getUC:       cfg.GetUC,
		updateUC:    cfg.UpdateUC,
		commandRepo: cfg.CommandRepo,
		queryRepo:   cfg.QueryRepo,
	}
}

// CreateWorkspace создаёт New workspace.
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

// GetWorkspace returns workspace по ID.
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
// uses repository напрямую, так as ListUserWorkspacesUseCase
// требует дополнительных methods Keycloak, которые пока not реализованы.
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

// UpdateWorkspace обновляет workspace.
func (s *WorkspaceService) UpdateWorkspace(
	ctx context.Context,
	id uuid.UUID,
	name, _ string,
) (*workspace.Workspace, error) {
	result, err := s.updateUC.Execute(ctx, wsapp.UpdateWorkspaceCommand{
		WorkspaceID: id,
		Name:        name,
		UpdatedBy:   uuid.NewUUID(), // TODO: get from context авторизации
	})
	if err != nil {
		return nil, err
	}

	return result.Value, nil
}

// DeleteWorkspace удаляет workspace.
// Use case for delete пока not реализован, используем repository напрямую.
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
