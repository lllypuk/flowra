package service

import (
	"context"
	"errors"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
	"github.com/lllypuk/flowra/internal/middleware"
)

// WorkspaceQueryRepository defines interface repozitoriya, neobhodimyy for access checker.
// obyavlen on storone potrebitelya according to principles Go interface design.
type WorkspaceQueryRepository interface {
	// FindByID finds workspace space po ID
	FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)

	// GetMember returns chlena workspace po userID
	GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*workspace.Member, error)
}

// RealWorkspaceAccessChecker realizuet middleware.WorkspaceAccessChecker
// ispolzuya realnyy repozitoriy workspace.
type RealWorkspaceAccessChecker struct {
	repo WorkspaceQueryRepository
}

// NewRealWorkspaceAccessChecker sozdayot New access checker.
func NewRealWorkspaceAccessChecker(repo WorkspaceQueryRepository) *RealWorkspaceAccessChecker {
	return &RealWorkspaceAccessChecker{repo: repo}
}

// GetMembership returns informatsiyu o chlenstve user in workspace.
// returns (nil, nil) if user not is chlenom workspace.
// returns middleware.ErrWorkspaceNotFound if workspace not suschestvuet.
//
//nolint:nilnil // nil, nil is a valid return to indicate "not a member" without error
func (c *RealWorkspaceAccessChecker) GetMembership(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
) (*middleware.WorkspaceMembership, error) {
	// snachala checking, that workspace suschestvuet and receiv ego data
	ws, err := c.repo.FindByID(ctx, workspaceID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, middleware.ErrWorkspaceNotFound
		}
		return nil, err
	}

	// poluchaem informatsiyu o chlenstve
	member, err := c.repo.GetMember(ctx, workspaceID, userID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			// user not member of workspace — return nil without error
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

// WorkspaceExists checks suschestvovanie workspace.
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
