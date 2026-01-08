package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/user"
)

// UpdateProfileUseCase handles update profilya user
type UpdateProfileUseCase struct {
	userRepo Repository
}

// NewUpdateProfileUseCase creates New UpdateProfileUseCase
func NewUpdateProfileUseCase(userRepo Repository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{userRepo: userRepo}
}

// Execute performs update profilya
func (uc *UpdateProfileUseCase) Execute(
	ctx context.Context,
	cmd UpdateProfileCommand,
) (Result, error) {
	// validation
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// Loading user
	usr, err := uc.userRepo.FindByID(ctx, cmd.UserID)
	if err != nil {
		return Result{}, ErrUserNotFound
	}

	// check unique email if on menyaetsya
	if cmd.Email != nil {
		existingByEmail, emailErr := uc.userRepo.FindByEmail(ctx, *cmd.Email)
		if emailErr == nil && existingByEmail != nil && existingByEmail.ID() != usr.ID() {
			return Result{}, ErrEmailAlreadyExists
		}
	}

	// update profilya
	if updateErr := usr.UpdateProfile(cmd.DisplayName, cmd.Email); updateErr != nil {
		return Result{}, fmt.Errorf("failed to update profile: %w", updateErr)
	}

	// storage
	if saveErr := uc.userRepo.Save(ctx, usr); saveErr != nil {
		return Result{}, fmt.Errorf("failed to save user: %w", saveErr)
	}

	return Result{
		Result: appcore.Result[*user.User]{
			Value: usr,
		},
	}, nil
}

func (uc *UpdateProfileUseCase) validate(cmd UpdateProfileCommand) error {
	if err := appcore.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}

	// Checking, that hotya by odno field for updating ukazano
	if cmd.DisplayName == nil && cmd.Email == nil {
		return errors.New("at least one field (displayName or email) must be provided")
	}

	// validation email if on predostavlen
	if cmd.Email != nil {
		if err := appcore.ValidateEmail("email", *cmd.Email); err != nil {
			return err
		}
	}

	// validation displayName if on predostavlen
	if cmd.DisplayName != nil && *cmd.DisplayName == "" {
		return appcore.NewValidationError("displayName", "cannot be empty")
	}

	return nil
}
