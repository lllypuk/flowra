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

// WorkspaceServiceCommandRepository определяет интерфейс для команд (изменение состояния) рабочих пространств.
// Интерфейс объявлен на стороне потребителя согласно принципам Go interface design.
type WorkspaceServiceCommandRepository interface {
	// Save сохраняет рабочее пространство (создание или обновление)
	Save(ctx context.Context, ws *workspace.Workspace) error

	// Delete удаляет рабочее пространство
	Delete(ctx context.Context, id uuid.UUID) error

	// AddMember добавляет члена в workspace
	AddMember(ctx context.Context, member *workspace.Member) error
}

// WorkspaceServiceQueryRepository определяет интерфейс для запросов (только чтение) рабочих пространств.
// Интерфейс объявлен на стороне потребителя согласно принципам Go interface design.
type WorkspaceServiceQueryRepository interface {
	// FindByID находит рабочее пространство по ID
	FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)

	// ListWorkspacesByUser возвращает workspaces, в которых пользователь является членом
	ListWorkspacesByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*workspace.Workspace, error)

	// CountWorkspacesByUser возвращает количество workspaces пользователя
	CountWorkspacesByUser(ctx context.Context, userID uuid.UUID) (int, error)

	// CountMembers возвращает количество членов workspace
	CountMembers(ctx context.Context, workspaceID uuid.UUID) (int, error)
}

// CreateWorkspaceUseCase определяет интерфейс для use case создания workspace.
type CreateWorkspaceUseCase interface {
	Execute(ctx context.Context, cmd wsapp.CreateWorkspaceCommand) (wsapp.Result, error)
}

// GetWorkspaceUseCase определяет интерфейс для use case получения workspace.
type GetWorkspaceUseCase interface {
	Execute(ctx context.Context, query wsapp.GetWorkspaceQuery) (wsapp.Result, error)
}

// UpdateWorkspaceUseCase определяет интерфейс для use case обновления workspace.
type UpdateWorkspaceUseCase interface {
	Execute(ctx context.Context, cmd wsapp.UpdateWorkspaceCommand) (wsapp.Result, error)
}

// WorkspaceService реализует httphandler.WorkspaceService
type WorkspaceService struct {
	// Use cases
	createUC CreateWorkspaceUseCase
	getUC    GetWorkspaceUseCase
	updateUC UpdateWorkspaceUseCase

	// Repositories (для операций без use case)
	commandRepo WorkspaceServiceCommandRepository
	queryRepo   WorkspaceServiceQueryRepository
}

// WorkspaceServiceConfig содержит зависимости для WorkspaceService.
type WorkspaceServiceConfig struct {
	CreateUC    CreateWorkspaceUseCase
	GetUC       GetWorkspaceUseCase
	UpdateUC    UpdateWorkspaceUseCase
	CommandRepo WorkspaceServiceCommandRepository
	QueryRepo   WorkspaceServiceQueryRepository
}

// NewWorkspaceService создаёт новый WorkspaceService.
func NewWorkspaceService(cfg WorkspaceServiceConfig) *WorkspaceService {
	return &WorkspaceService{
		createUC:    cfg.CreateUC,
		getUC:       cfg.GetUC,
		updateUC:    cfg.UpdateUC,
		commandRepo: cfg.CommandRepo,
		queryRepo:   cfg.QueryRepo,
	}
}

// CreateWorkspace создаёт новый workspace.
func (s *WorkspaceService) CreateWorkspace(
	ctx context.Context,
	ownerID uuid.UUID,
	name, _ string,
) (*workspace.Workspace, error) {
	result, err := s.createUC.Execute(ctx, wsapp.CreateWorkspaceCommand{
		Name:      name,
		CreatedBy: ownerID,
	})
	if err != nil {
		return nil, err
	}

	return result.Value, nil
}

// GetWorkspace возвращает workspace по ID.
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

// ListUserWorkspaces возвращает список workspaces пользователя.
// Использует repository напрямую, так как ListUserWorkspacesUseCase
// требует дополнительных методов Keycloak, которые пока не реализованы.
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
		UpdatedBy:   uuid.NewUUID(), // TODO: получить из контекста авторизации
	})
	if err != nil {
		return nil, err
	}

	return result.Value, nil
}

// DeleteWorkspace удаляет workspace.
// Use case для delete пока не реализован, используем repository напрямую.
func (s *WorkspaceService) DeleteWorkspace(
	ctx context.Context,
	id uuid.UUID,
) error {
	return s.commandRepo.Delete(ctx, id)
}

// GetMemberCount возвращает количество участников workspace.
func (s *WorkspaceService) GetMemberCount(
	ctx context.Context,
	workspaceID uuid.UUID,
) (int, error) {
	return s.queryRepo.CountMembers(ctx, workspaceID)
}
