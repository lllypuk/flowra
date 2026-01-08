package service

import (
	"context"
	"errors"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
	"github.com/lllypuk/flowra/internal/middleware"
)

// WorkspaceQueryRepository defines interface репозитория, необходимый for access checker.
// Объявлен on стороне потребителя according to principles Go interface design.
type WorkspaceQueryRepository interface {
	// FindByID finds workspaceее пространство по ID
	FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)

	// GetMember returns члена workspace по userID
	GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*workspace.Member, error)
}

// RealWorkspaceAccessChecker реализует middleware.WorkspaceAccessChecker
// используя реальный репозиторий workspace.
type RealWorkspaceAccessChecker struct {
	repo WorkspaceQueryRepository
}

// NewRealWorkspaceAccessChecker создаёт New access checker.
func NewRealWorkspaceAccessChecker(repo WorkspaceQueryRepository) *RealWorkspaceAccessChecker {
	return &RealWorkspaceAccessChecker{repo: repo}
}

// GetMembership returns информацию о членстве user in workspace.
// returns (nil, nil) if userель not is членом workspace.
// returns middleware.ErrWorkspaceNotFound if workspace not существует.
//
//nolint:nilnil // nil, nil is a valid return to indicate "not a member" without error
func (c *RealWorkspaceAccessChecker) GetMembership(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
) (*middleware.WorkspaceMembership, error) {
	// Сначала checking, that workspace существует and receivаем его data
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
			// Пользователь not член workspace — возвращаем nil без error
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

// WorkspaceExists checks существование workspace.
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
