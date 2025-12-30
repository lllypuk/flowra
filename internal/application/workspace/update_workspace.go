package workspace

import (
	"context"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// UpdateWorkspaceUseCase - use case для обновления workspace
type UpdateWorkspaceUseCase struct {
	shared.BaseUseCase

	workspaceRepo Repository
}

// NewUpdateWorkspaceUseCase создает новый UpdateWorkspaceUseCase
func NewUpdateWorkspaceUseCase(workspaceRepo Repository) *UpdateWorkspaceUseCase {
	return &UpdateWorkspaceUseCase{
		workspaceRepo: workspaceRepo,
	}
}

// Execute выполняет обновление workspace
func (uc *UpdateWorkspaceUseCase) Execute(
	ctx context.Context,
	cmd UpdateWorkspaceCommand,
) (Result, error) {
	// Валидация контекста
	if err := uc.ValidateContext(ctx); err != nil {
		return Result{}, uc.WrapError("validate context", err)
	}

	// Валидация команды
	if err := uc.validate(cmd); err != nil {
		return Result{}, uc.WrapError("validation failed", err)
	}

	// Поиск workspace
	ws, err := uc.workspaceRepo.FindByID(ctx, cmd.WorkspaceID)
	if err != nil {
		return Result{}, uc.WrapError("find workspace", ErrWorkspaceNotFound)
	}

	// Обновление названия
	if errUpdate := ws.UpdateName(cmd.Name); errUpdate != nil {
		return Result{}, uc.WrapError("update workspace name", errUpdate)
	}

	// Сохранение
	if errSave := uc.workspaceRepo.Save(ctx, ws); errSave != nil {
		return Result{}, uc.WrapError("save workspace", errSave)
	}

	return Result{
		Result: shared.Result[*workspace.Workspace]{
			Value: ws,
		},
	}, nil
}

// validate проверяет валидность команды
func (uc *UpdateWorkspaceUseCase) validate(cmd UpdateWorkspaceCommand) error {
	if err := shared.ValidateUUID("workspaceID", cmd.WorkspaceID); err != nil {
		return err
	}
	if err := shared.ValidateRequired("name", cmd.Name); err != nil {
		return err
	}
	const maxNameLength = 100
	if err := shared.ValidateMaxLength("name", cmd.Name, maxNameLength); err != nil {
		return err
	}
	if err := shared.ValidateUUID("updatedBy", cmd.UpdatedBy); err != nil {
		return err
	}
	return nil
}
