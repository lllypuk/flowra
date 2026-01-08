package workspace

import (
	"context"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// RevokeInviteUseCase - use case for otzyva invayta
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

// Execute performs otzyv invayta
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

	// search invayta po ID
	// snachala nuzhno find workspace s etim invaytom
	// for it is nuzhno rasshirit Repository - add FindWorkspaceByInviteID
	// or mozhno user FindInviteByToken, no u nas only ID
	// uproschenie: predpolagaem that InviteID unikalen and mozhem find via perebor
	// in realnom proekte luchshe add method FindWorkspaceByInviteID in Repository

	// vremennoe reshenie: budem iskat workspace via all workspaces
	// eto not optimalno, no for primera podoydet
	// TODO: add method FindWorkspaceByInviteID in Repository

	// for uproscheniya, ispolzuem pryamoy podhod:
	// predpolagaem, that in komande takzhe est WorkspaceID or ischem po all workspaces
	// poskolku in zadanii takogo metoda no, realizuem search via privatnyy method

	invite, ws, err := uc.findInviteByID(ctx, cmd.InviteID)
	if err != nil {
		return InviteResult{}, uc.WrapError("find invite", ErrInviteNotFound)
	}

	// otzyv invayta
	if errRevoke := invite.Revoke(); errRevoke != nil {
		return InviteResult{}, uc.WrapError("revoke invite", errRevoke)
	}

	// save workspace s otozvannym invaytom
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

// findInviteByID finds invayt po ID
// eto vspomogatelnyy method, kotoryy ischet invite vo all workspaces
// in realnom proekte luchshe add indeks or pryamoy method searching
func (uc *RevokeInviteUseCase) findInviteByID(
	ctx context.Context,
	inviteID uuid.UUID,
) (*workspace.Invite, *workspace.Workspace, error) {
	// poluchaem all workspaces (not optimalno, no for primera)
	// in realnom proekte nuzhen indeks inviteID -> workspaceID
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
