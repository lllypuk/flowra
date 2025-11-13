package workspace

import (
	"context"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// RevokeInviteUseCase - use case для отзыва инвайта
type RevokeInviteUseCase struct {
	shared.BaseUseCase

	workspaceRepo Repository
}

// NewRevokeInviteUseCase создает новый RevokeInviteUseCase
func NewRevokeInviteUseCase(workspaceRepo Repository) *RevokeInviteUseCase {
	return &RevokeInviteUseCase{
		workspaceRepo: workspaceRepo,
	}
}

// Execute выполняет отзыв инвайта
func (uc *RevokeInviteUseCase) Execute(
	ctx context.Context,
	cmd RevokeInviteCommand,
) (InviteResult, error) {
	// Валидация контекста
	if err := uc.ValidateContext(ctx); err != nil {
		return InviteResult{}, uc.WrapError("validate context", err)
	}

	// Валидация команды
	if err := uc.validate(cmd); err != nil {
		return InviteResult{}, uc.WrapError("validation failed", err)
	}

	// Поиск инвайта по ID
	// Сначала нужно найти workspace с этим инвайтом
	// Для этого нужно расширить Repository - добавить FindWorkspaceByInviteID
	// Или можно использовать FindInviteByToken, но у нас только ID
	// Упрощение: предполагаем что InviteID уникален и можем найти через перебор
	// В реальном проекте лучше добавить метод FindWorkspaceByInviteID в Repository

	// Временное решение: будем искать workspace через все workspaces
	// Это не оптимально, но для примера подойдет
	// TODO: добавить метод FindWorkspaceByInviteID в Repository

	// Для упрощения, используем прямой подход:
	// Предполагаем, что в команде также есть WorkspaceID или ищем по всем workspaces
	// Поскольку в задании такого метода нет, реализуем поиск через приватный метод

	invite, ws, err := uc.findInviteByID(ctx, cmd.InviteID)
	if err != nil {
		return InviteResult{}, uc.WrapError("find invite", ErrInviteNotFound)
	}

	// Отзыв инвайта
	if errRevoke := invite.Revoke(); errRevoke != nil {
		return InviteResult{}, uc.WrapError("revoke invite", errRevoke)
	}

	// Сохранение workspace с отозванным инвайтом
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
func (uc *RevokeInviteUseCase) validate(cmd RevokeInviteCommand) error {
	if err := shared.ValidateUUID("inviteID", cmd.InviteID); err != nil {
		return err
	}
	if err := shared.ValidateUUID("revokedBy", cmd.RevokedBy); err != nil {
		return err
	}
	return nil
}

// findInviteByID находит инвайт по ID
// Это вспомогательный метод, который ищет invite во всех workspaces
// В реальном проекте лучше добавить индекс или прямой метод поиска
func (uc *RevokeInviteUseCase) findInviteByID(
	ctx context.Context,
	inviteID uuid.UUID,
) (*workspace.Invite, *workspace.Workspace, error) {
	// Получаем все workspaces (не оптимально, но для примера)
	// В реальном проекте нужен индекс inviteID -> workspaceID
	const maxWorkspaces = 1000
	workspaces, err := uc.workspaceRepo.List(ctx, 0, maxWorkspaces)
	if err != nil {
		return nil, nil, err
	}

	for _, ws := range workspaces {
		for _, invite := range ws.Invites() {
			if invite.ID() == inviteID {
				return invite, ws, nil
			}
		}
	}

	return nil, nil, ErrInviteNotFound
}
