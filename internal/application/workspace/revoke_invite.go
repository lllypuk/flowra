package workspace

import (
	"context"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// RevokeInviteUseCase - use case for отзыва инвайта
type RevokeInviteUseCase struct {
	appcore.BaseUseCase

	workspaceRepo Repository
}

// NewRevokeInviteUseCase creates New RevokeInviteUseCase
func NewRevokeInviteUseCase(workspaceRepo Repository) *RevokeInviteUseCase {
	return &RevokeInviteUseCase{
		workspaceRepo: workspaceRepo,
	}
}

// Execute performs отзыв инвайта
func (uc *RevokeInviteUseCase) Execute(
	ctx context.Context,
	cmd RevokeInviteCommand,
) (InviteResult, error) {
	// context validation
	if err := uc.ValidateContext(ctx); err != nil {
		return InviteResult{}, uc.WrapError("validate context", err)
	}

	// validation commands
	if err := uc.validate(cmd); err != nil {
		return InviteResult{}, uc.WrapError("validation failed", err)
	}

	// search инвайта по ID
	// Сначала нужно find workspace с этим инвайтом
	// for it isго нужно расширить Repository - add FindWorkspaceByInviteID
	// or можно исuserь FindInviteByToken, но у нас only ID
	// Упрощение: предполагаем that InviteID уникален and можем find via перебор
	// in реальном проекте лучше add method FindWorkspaceByInviteID in Repository

	// Временное решение: будем искать workspace via all workspaces
	// Это not оптимально, но for примера подойдет
	// TODO: add method FindWorkspaceByInviteID in Repository

	// for упрощения, используем прямой подход:
	// Предполагаем, that in команде также есть WorkspaceID or ищем по allм workspaces
	// Поскольку in задании такого метода no, реализуем search via приватный method

	invite, ws, err := uc.findInviteByID(ctx, cmd.InviteID)
	if err != nil {
		return InviteResult{}, uc.WrapError("find invite", ErrInviteNotFound)
	}

	// Отзыв инвайта
	if errRevoke := invite.Revoke(); errRevoke != nil {
		return InviteResult{}, uc.WrapError("revoke invite", errRevoke)
	}

	// storage workspace с отозванным инвайтом
	if errSave := uc.workspaceRepo.Save(ctx, ws); errSave != nil {
		return InviteResult{}, uc.WrapError("save workspace", errSave)
	}

	return InviteResult{
		Result: appcore.Result[*workspace.Invite]{
			Value: invite,
		},
	}, nil
}

// validate validates commands
func (uc *RevokeInviteUseCase) validate(cmd RevokeInviteCommand) error {
	if err := appcore.ValidateUUID("inviteID", cmd.InviteID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("revokedBy", cmd.RevokedBy); err != nil {
		return err
	}
	return nil
}

// findInviteByID finds инвайт по ID
// Это вспомогательный method, который ищет invite во all workspaces
// in реальном проекте лучше add индекс or прямой method searching
func (uc *RevokeInviteUseCase) findInviteByID(
	ctx context.Context,
	inviteID uuid.UUID,
) (*workspace.Invite, *workspace.Workspace, error) {
	// Получаем all workspaces (not оптимально, но for примера)
	// in реальном проекте нужен индекс inviteID -> workspaceID
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
