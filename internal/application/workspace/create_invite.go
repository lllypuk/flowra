package workspace

import (
	"context"
	"time"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// CreateInviteUseCase - use case for creating invayta
type CreateInviteUseCase struct {
	appcore.BaseUseCase

	workspaceRepo Repository
}

// NewCreateInviteUseCase creates New CreateInviteUseCase
func NewCreateInviteUseCase(workspaceRepo Repository) *CreateInviteUseCase {
	return &CreateInviteUseCase{
		workspaceRepo: workspaceRepo,
	}
}

// Execute performs creation invayta
func (uc *CreateInviteUseCase) Execute(
	ctx context.Context,
	cmd CreateInviteCommand,
) (InviteResult, error) {
	// context validation
	if err := uc.ValidateContext(ctx); err != nil {
		return InviteResult{}, uc.WrapError("validate context", err)
	}

	// validation commands
	if err := uc.validate(cmd); err != nil {
		return InviteResult{}, uc.WrapError("validation failed", err)
	}

	// Searching workspace
	ws, err := uc.workspaceRepo.FindByID(ctx, cmd.WorkspaceID)
	if err != nil {
		return InviteResult{}, uc.WrapError("find workspace", ErrWorkspaceNotFound)
	}

	// setting values by default
	expiresAt := uc.getExpiresAt(cmd.ExpiresAt)
	maxUses := uc.getMaxUses(cmd.MaxUses)

	// creation invayta
	invite, err := ws.CreateInvite(cmd.CreatedBy, expiresAt, maxUses)
	if err != nil {
		return InviteResult{}, uc.WrapError("create invite", err)
	}

	// save workspace s novym invaytom
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
func (uc *CreateInviteUseCase) validate(cmd CreateInviteCommand) error {
	if err := appcore.ValidateUUID("workspaceID", cmd.WorkspaceID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("createdBy", cmd.CreatedBy); err != nil {
		return err
	}
	if cmd.ExpiresAt != nil {
		if err := appcore.ValidateDateNotPast("expiresAt", cmd.ExpiresAt); err != nil {
			return err
		}
	}
	if cmd.MaxUses != nil {
		if err := appcore.ValidateNonNegative("maxUses", *cmd.MaxUses); err != nil {
			return err
		}
	}
	return nil
}

// getExpiresAt returns time istecheniya invayta (by default: 7 dney)
func (uc *CreateInviteUseCase) getExpiresAt(expiresAt *time.Time) time.Time {
	if expiresAt != nil {
		return *expiresAt
	}
	return time.Now().Add(7 * 24 * time.Hour)
}

// getMaxUses returns maximum count ispolzovaniy (by default: 0 - unlimited)
func (uc *CreateInviteUseCase) getMaxUses(maxUses *int) int {
	if maxUses != nil {
		return *maxUses
	}
	return 0
}
