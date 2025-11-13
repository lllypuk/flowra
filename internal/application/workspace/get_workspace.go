package workspace

import (
	"context"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// GetWorkspaceUseCase - use case для получения workspace по ID
type GetWorkspaceUseCase struct {
	shared.BaseUseCase

	workspaceRepo Repository
}

// NewGetWorkspaceUseCase создает новый GetWorkspaceUseCase
func NewGetWorkspaceUseCase(workspaceRepo Repository) *GetWorkspaceUseCase {
	return &GetWorkspaceUseCase{
		workspaceRepo: workspaceRepo,
	}
}

// Execute выполняет получение workspace
func (uc *GetWorkspaceUseCase) Execute(
	ctx context.Context,
	query GetWorkspaceQuery,
) (Result, error) {
	// Валидация контекста
	if err := uc.ValidateContext(ctx); err != nil {
		return Result{}, uc.WrapError("validate context", err)
	}

	// Валидация запроса
	if err := uc.validate(query); err != nil {
		return Result{}, uc.WrapError("validation failed", err)
	}

	// Поиск workspace
	ws, err := uc.workspaceRepo.FindByID(ctx, query.WorkspaceID)
	if err != nil {
		return Result{}, uc.WrapError("find workspace", ErrWorkspaceNotFound)
	}

	return Result{
		Result: shared.Result[*workspace.Workspace]{
			Value: ws,
		},
	}, nil
}

// validate проверяет валидность запроса
func (uc *GetWorkspaceUseCase) validate(query GetWorkspaceQuery) error {
	if err := shared.ValidateUUID("workspaceID", query.WorkspaceID); err != nil {
		return err
	}
	return nil
}
