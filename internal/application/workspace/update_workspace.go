package workspace

import (
	"context"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// UpdateWorkspaceUseCase - use case for updating workspace
type UpdateWorkspaceUseCase struct {
	appcore.BaseUseCase

	workspaceRepo Repository
}

// NewUpdateWorkspaceUseCase creates New UpdateWorkspaceUseCase
func NewUpdateWorkspaceUseCase(workspaceRepo Repository) *UpdateWorkspaceUseCase {
	return &UpdateWorkspaceUseCase{
		workspaceRepo: workspaceRepo,
	}
}

// Execute performs update workspace
func (uc *UpdateWorkspaceUseCase) Execute(
	ctx context.Context,
	cmd UpdateWorkspaceCommand,
) (Result, error) {
	// context validation
	if err := uc.ValidateContext(ctx); err != nil {
		return Result{}, uc.WrapError("validate context", err)
	}

	// validation commands
	if err := uc.validate(cmd); err != nil {
		return Result{}, uc.WrapError("validation failed", err)
	}

	// Searching workspace
	ws, err := uc.workspaceRepo.FindByID(ctx, cmd.WorkspaceID)
	if err != nil {
		return Result{}, uc.WrapError("find workspace", ErrWorkspaceNotFound)
	}

	// update названия
	if errUpdate := ws.UpdateName(cmd.Name); errUpdate != nil {
		return Result{}, uc.WrapError("update workspace name", errUpdate)
	}

	// storage
	if errSave := uc.workspaceRepo.Save(ctx, ws); errSave != nil {
		return Result{}, uc.WrapError("save workspace", errSave)
	}

	return Result{
		Result: appcore.Result[*workspace.Workspace]{
			Value: ws,
		},
	}, nil
}

// validate validates commands
func (uc *UpdateWorkspaceUseCase) validate(cmd UpdateWorkspaceCommand) error {
	if err := appcore.ValidateUUID("workspaceID", cmd.WorkspaceID); err != nil {
		return err
	}
	if err := appcore.ValidateRequired("name", cmd.Name); err != nil {
		return err
	}
	const maxNameLength = 100
	if err := appcore.ValidateMaxLength("name", cmd.Name, maxNameLength); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("updatedBy", cmd.UpdatedBy); err != nil {
		return err
	}
	return nil
}
