package workspace

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// CreateWorkspaceUseCase - use case для создания workspace
type CreateWorkspaceUseCase struct {
	shared.BaseUseCase

	workspaceRepo  workspace.Repository
	keycloakClient KeycloakClient
}

// NewCreateWorkspaceUseCase создает новый CreateWorkspaceUseCase
func NewCreateWorkspaceUseCase(
	workspaceRepo workspace.Repository,
	keycloakClient KeycloakClient,
) *CreateWorkspaceUseCase {
	return &CreateWorkspaceUseCase{
		workspaceRepo:  workspaceRepo,
		keycloakClient: keycloakClient,
	}
}

// Execute выполняет создание workspace
func (uc *CreateWorkspaceUseCase) Execute(
	ctx context.Context,
	cmd CreateWorkspaceCommand,
) (Result, error) {
	// Валидация контекста
	if err := uc.ValidateContext(ctx); err != nil {
		return Result{}, uc.WrapError("validate context", err)
	}

	// Валидация команды
	if err := uc.validate(cmd); err != nil {
		return Result{}, uc.WrapError("validation failed", err)
	}

	// Создание группы в Keycloak
	keycloakGroupID, err := uc.keycloakClient.CreateGroup(ctx, cmd.Name)
	if err != nil {
		return Result{}, uc.WrapError(
			"create Keycloak group",
			fmt.Errorf("%w: %w", ErrKeycloakGroupCreationFailed, err),
		)
	}

	// Создание workspace
	ws, err := workspace.NewWorkspace(cmd.Name, keycloakGroupID, cmd.CreatedBy)
	if err != nil {
		// Rollback: удаляем группу в Keycloak
		_ = uc.keycloakClient.DeleteGroup(ctx, keycloakGroupID)
		return Result{}, uc.WrapError("create workspace entity", err)
	}

	// Сохранение workspace
	if errSave := uc.workspaceRepo.Save(ctx, ws); errSave != nil {
		// Rollback: удаляем группу в Keycloak
		_ = uc.keycloakClient.DeleteGroup(ctx, keycloakGroupID)
		return Result{}, uc.WrapError("save workspace", errSave)
	}

	// Добавление создателя в группу Keycloak
	// Не критично, можно залогировать, но не откатываем workspace
	_ = uc.keycloakClient.AddUserToGroup(ctx, cmd.CreatedBy.String(), keycloakGroupID)

	return Result{
		Result: shared.Result[*workspace.Workspace]{
			Value: ws,
		},
	}, nil
}

// validate проверяет валидность команды
func (uc *CreateWorkspaceUseCase) validate(cmd CreateWorkspaceCommand) error {
	if err := shared.ValidateRequired("name", cmd.Name); err != nil {
		return err
	}
	const maxNameLength = 100
	if err := shared.ValidateMaxLength("name", cmd.Name, maxNameLength); err != nil {
		return err
	}
	if err := shared.ValidateUUID("createdBy", cmd.CreatedBy); err != nil {
		return err
	}
	return nil
}
