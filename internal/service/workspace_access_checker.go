package service

import (
	"context"
	"errors"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
	"github.com/lllypuk/flowra/internal/middleware"
)

// WorkspaceQueryRepository определяет интерфейс репозитория, необходимый для access checker.
// Объявлен на стороне потребителя согласно принципам Go interface design.
type WorkspaceQueryRepository interface {
	// FindByID находит рабочее пространство по ID
	FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)

	// GetMember возвращает члена workspace по userID
	GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*workspace.Member, error)
}

// RealWorkspaceAccessChecker реализует middleware.WorkspaceAccessChecker
// используя реальный репозиторий workspace.
type RealWorkspaceAccessChecker struct {
	repo WorkspaceQueryRepository
}

// NewRealWorkspaceAccessChecker создаёт новый access checker.
func NewRealWorkspaceAccessChecker(repo WorkspaceQueryRepository) *RealWorkspaceAccessChecker {
	return &RealWorkspaceAccessChecker{repo: repo}
}

// GetMembership возвращает информацию о членстве пользователя в workspace.
// Возвращает (nil, nil) если пользователь не является членом workspace.
// Возвращает middleware.ErrWorkspaceNotFound если workspace не существует.
//
//nolint:nilnil // nil, nil is a valid return to indicate "not a member" without error
func (c *RealWorkspaceAccessChecker) GetMembership(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
) (*middleware.WorkspaceMembership, error) {
	// Сначала проверяем, что workspace существует и получаем его данные
	ws, err := c.repo.FindByID(ctx, workspaceID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, middleware.ErrWorkspaceNotFound
		}
		return nil, err
	}

	// Получаем информацию о членстве
	member, err := c.repo.GetMember(ctx, workspaceID, userID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			// Пользователь не член workspace — возвращаем nil без ошибки
			return nil, nil
		}
		return nil, err
	}

	return &middleware.WorkspaceMembership{
		WorkspaceID:   workspaceID,
		WorkspaceName: ws.Name(),
		UserID:        userID,
		Role:          member.Role().String(),
	}, nil
}

// WorkspaceExists проверяет существование workspace.
func (c *RealWorkspaceAccessChecker) WorkspaceExists(
	ctx context.Context,
	workspaceID uuid.UUID,
) (bool, error) {
	ws, err := c.repo.FindByID(ctx, workspaceID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return ws != nil, nil
}
