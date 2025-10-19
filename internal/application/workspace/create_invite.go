package workspace

import (
	"context"
	"time"

	"github.com/lllypuk/teams-up/internal/application/shared"
	"github.com/lllypuk/teams-up/internal/domain/workspace"
)

// CreateInviteUseCase - use case для создания инвайта
type CreateInviteUseCase struct {
	shared.BaseUseCase

	workspaceRepo workspace.Repository
}

// NewCreateInviteUseCase создает новый CreateInviteUseCase
func NewCreateInviteUseCase(workspaceRepo workspace.Repository) *CreateInviteUseCase {
	return &CreateInviteUseCase{
		workspaceRepo: workspaceRepo,
	}
}

// Execute выполняет создание инвайта
func (uc *CreateInviteUseCase) Execute(
	ctx context.Context,
	cmd CreateInviteCommand,
) (InviteResult, error) {
	// Валидация контекста
	if err := uc.ValidateContext(ctx); err != nil {
		return InviteResult{}, uc.WrapError("validate context", err)
	}

	// Валидация команды
	if err := uc.validate(cmd); err != nil {
		return InviteResult{}, uc.WrapError("validation failed", err)
	}

	// Поиск workspace
	ws, err := uc.workspaceRepo.FindByID(ctx, cmd.WorkspaceID)
	if err != nil {
		return InviteResult{}, uc.WrapError("find workspace", ErrWorkspaceNotFound)
	}

	// Установка значений по умолчанию
	expiresAt := uc.getExpiresAt(cmd.ExpiresAt)
	maxUses := uc.getMaxUses(cmd.MaxUses)

	// Создание инвайта
	invite, err := ws.CreateInvite(cmd.CreatedBy, expiresAt, maxUses)
	if err != nil {
		return InviteResult{}, uc.WrapError("create invite", err)
	}

	// Сохранение workspace с новым инвайтом
	if errSave := uc.workspaceRepo.Save(ctx, ws); errSave != nil {
		return InviteResult{}, uc.WrapError("save workspace", errSave)
	}

	return InviteResult{
		Result: shared.Result[*workspace.Invite]{
			Value: invite,
		},
	}, nil
}

// validate проверяет валидность команды
func (uc *CreateInviteUseCase) validate(cmd CreateInviteCommand) error {
	if err := shared.ValidateUUID("workspaceID", cmd.WorkspaceID); err != nil {
		return err
	}
	if err := shared.ValidateUUID("createdBy", cmd.CreatedBy); err != nil {
		return err
	}
	if cmd.ExpiresAt != nil {
		if err := shared.ValidateDateNotPast("expiresAt", cmd.ExpiresAt); err != nil {
			return err
		}
	}
	if cmd.MaxUses != nil {
		if err := shared.ValidateNonNegative("maxUses", *cmd.MaxUses); err != nil {
			return err
		}
	}
	return nil
}

// getExpiresAt возвращает время истечения инвайта (по умолчанию: 7 дней)
func (uc *CreateInviteUseCase) getExpiresAt(expiresAt *time.Time) time.Time {
	if expiresAt != nil {
		return *expiresAt
	}
	return time.Now().Add(7 * 24 * time.Hour)
}

// getMaxUses возвращает максимальное количество использований (по умолчанию: 0 - unlimited)
func (uc *CreateInviteUseCase) getMaxUses(maxUses *int) int {
	if maxUses != nil {
		return *maxUses
	}
	return 0
}
