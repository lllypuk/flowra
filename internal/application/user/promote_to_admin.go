package user

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/user"
)

// PromoteToAdminUseCase handles povyshenie user before administrator
type PromoteToAdminUseCase struct {
	userRepo Repository
}

// NewPromoteToAdminUseCase creates New PromoteToAdminUseCase
func NewPromoteToAdminUseCase(userRepo Repository) *PromoteToAdminUseCase {
	return &PromoteToAdminUseCase{userRepo: userRepo}
}

// Execute performs povyshenie before administrator
func (uc *PromoteToAdminUseCase) Execute(
	ctx context.Context,
	cmd PromoteToAdminCommand,
) (Result, error) {
	// validation
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// check prav vypolnyayuschego operatsiyu
	promoter, err := uc.userRepo.FindByID(ctx, cmd.PromotedBy)
	if err != nil {
		return Result{}, ErrUserNotFound
	}

	if !promoter.IsSystemAdmin() {
		return Result{}, ErrNotSystemAdmin
	}

	// Loading tselevogo user
	targetUser, targetErr := uc.userRepo.FindByID(ctx, cmd.UserID)
	if targetErr != nil {
		return Result{}, ErrUserNotFound
	}

	// setting prav administrator
	targetUser.SetAdmin(true)

	// storage
	if saveErr := uc.userRepo.Save(ctx, targetUser); saveErr != nil {
		return Result{}, fmt.Errorf("failed to save user: %w", saveErr)
	}

	return Result{
		Result: appcore.Result[*user.User]{
			Value: targetUser,
		},
	}, nil
}

func (uc *PromoteToAdminUseCase) validate(cmd PromoteToAdminCommand) error {
	if err := appcore.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("promotedBy", cmd.PromotedBy); err != nil {
		return err
	}
	return nil
}
