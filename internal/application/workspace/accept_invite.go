package workspace

import (
	"context"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// AcceptInviteUseCase - use case for prinyatiya invayta
type AcceptInviteUseCase struct {
	appcore.BaseUseCase

	workspaceRepo  Repository
	keycloakClient KeycloakClient
}

// NewAcceptInviteUseCase creates New AcceptInviteUseCase
func NewAcceptInviteUseCase(
	workspaceRepo Repository,
	keycloakClient KeycloakClient,
) *AcceptInviteUseCase {
	return &AcceptInviteUseCase{
		workspaceRepo:  workspaceRepo,
		keycloakClient: keycloakClient,
	}
}

// Execute performs prinyatie invayta
func (uc *AcceptInviteUseCase) Execute(
	ctx context.Context,
	cmd AcceptInviteCommand,
) (Result, error) {
	// context validation
	if err := uc.ValidateContext(ctx); err != nil {
		return Result{}, uc.WrapError("validate context", err)
	}

	// validation commands
	if err := uc.validate(cmd); err != nil {
		return Result{}, uc.WrapError("validation failed", err)
	}

	// search invayta po tokenu
	invite, err := uc.workspaceRepo.FindInviteByToken(ctx, cmd.Token)
	if err != nil {
		return Result{}, uc.WrapError("find invite", ErrInviteNotFound)
	}

	// check valid invayta
	if !invite.IsValid() {
		if invite.IsRevoked() {
			return Result{}, uc.WrapError("validate invite", ErrInviteRevoked)
		}
		// invayt expired or dostignut limit ispolzovaniy
		return Result{}, uc.WrapError("validate invite", ErrInviteExpired)
	}

	// Searching workspace
	ws, err := uc.workspaceRepo.FindByID(ctx, invite.WorkspaceID())
	if err != nil {
		return Result{}, uc.WrapError("find workspace", ErrWorkspaceNotFound)
	}

	// use invayta (uvelichenie schetchika)
	if errUse := invite.Use(); errUse != nil {
		return Result{}, uc.WrapError("use invite", errUse)
	}

	// save workspace s obnovlennym invaytom
	if errSave := uc.workspaceRepo.Save(ctx, ws); errSave != nil {
		return Result{}, uc.WrapError("save workspace", errSave)
	}

	// Adding user in groups Keycloak
	errKeycloak := uc.keycloakClient.AddUserToGroup(
		ctx,
		cmd.UserID.String(),
		ws.KeycloakGroupID(),
	)
	if errKeycloak != nil {
		// otkatyvaem use invayta? no, t.to. uzhe sav.
		// in realnom prilozhenii nuzhna tranzaktsionnost or saga pattern
		return Result{}, uc.WrapError("add user to Keycloak group", ErrKeycloakUserAddFailed)
	}

	return Result{
		Result: appcore.Result[*workspace.Workspace]{
			Value: ws,
		},
	}, nil
}

// validate validates commands
func (uc *AcceptInviteUseCase) validate(cmd AcceptInviteCommand) error {
	if err := appcore.ValidateRequired("token", cmd.Token); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	return nil
}
