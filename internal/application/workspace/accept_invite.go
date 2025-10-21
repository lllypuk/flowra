package workspace

import (
	"context"

	"github.com/flowra/flowra/internal/application/shared"
	"github.com/flowra/flowra/internal/domain/workspace"
)

// AcceptInviteUseCase - use case для принятия инвайта
type AcceptInviteUseCase struct {
	shared.BaseUseCase

	workspaceRepo  workspace.Repository
	keycloakClient KeycloakClient
}

// NewAcceptInviteUseCase создает новый AcceptInviteUseCase
func NewAcceptInviteUseCase(
	workspaceRepo workspace.Repository,
	keycloakClient KeycloakClient,
) *AcceptInviteUseCase {
	return &AcceptInviteUseCase{
		workspaceRepo:  workspaceRepo,
		keycloakClient: keycloakClient,
	}
}

// Execute выполняет принятие инвайта
func (uc *AcceptInviteUseCase) Execute(
	ctx context.Context,
	cmd AcceptInviteCommand,
) (Result, error) {
	// Валидация контекста
	if err := uc.ValidateContext(ctx); err != nil {
		return Result{}, uc.WrapError("validate context", err)
	}

	// Валидация команды
	if err := uc.validate(cmd); err != nil {
		return Result{}, uc.WrapError("validation failed", err)
	}

	// Поиск инвайта по токену
	invite, err := uc.workspaceRepo.FindInviteByToken(ctx, cmd.Token)
	if err != nil {
		return Result{}, uc.WrapError("find invite", ErrInviteNotFound)
	}

	// Проверка валидности инвайта
	if !invite.IsValid() {
		if invite.IsRevoked() {
			return Result{}, uc.WrapError("validate invite", ErrInviteRevoked)
		}
		// Инвайт истек или достигнут лимит использований
		return Result{}, uc.WrapError("validate invite", ErrInviteExpired)
	}

	// Поиск workspace
	ws, err := uc.workspaceRepo.FindByID(ctx, invite.WorkspaceID())
	if err != nil {
		return Result{}, uc.WrapError("find workspace", ErrWorkspaceNotFound)
	}

	// Использование инвайта (увеличение счетчика)
	if errUse := invite.Use(); errUse != nil {
		return Result{}, uc.WrapError("use invite", errUse)
	}

	// Сохранение workspace с обновленным инвайтом
	if errSave := uc.workspaceRepo.Save(ctx, ws); errSave != nil {
		return Result{}, uc.WrapError("save workspace", errSave)
	}

	// Добавление пользователя в группу Keycloak
	if errKeycloak := uc.keycloakClient.AddUserToGroup(ctx, cmd.UserID.String(), ws.KeycloakGroupID()); errKeycloak != nil {
		// Откатываем использование инвайта? Нет, т.к. уже сохранили.
		// В реальном приложении нужна транзакционность или saga pattern
		return Result{}, uc.WrapError("add user to Keycloak group", ErrKeycloakUserAddFailed)
	}

	return Result{
		Result: shared.Result[*workspace.Workspace]{
			Value: ws,
		},
	}, nil
}

// validate проверяет валидность команды
func (uc *AcceptInviteUseCase) validate(cmd AcceptInviteCommand) error {
	if err := shared.ValidateRequired("token", cmd.Token); err != nil {
		return err
	}
	if err := shared.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	return nil
}
