package user

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/user"
)

// RegisterUserUseCase handles registratsiyu novogo user
type RegisterUserUseCase struct {
	userRepo Repository
}

// NewRegisterUserUseCase creates New RegisterUserUseCase
func NewRegisterUserUseCase(userRepo Repository) *RegisterUserUseCase {
	return &RegisterUserUseCase{userRepo: userRepo}
}

// Execute performs registratsiyu user
func (uc *RegisterUserUseCase) Execute(
	ctx context.Context,
	cmd RegisterUserCommand,
) (Result, error) {
	// validation
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// check unique username
	existing, err := uc.userRepo.FindByUsername(ctx, cmd.Username)
	if err == nil && existing != nil {
		return Result{}, ErrUsernameAlreadyExists
	}

	// check unique email
	existingByEmail, err := uc.userRepo.FindByEmail(ctx, cmd.Email)
	if err == nil && existingByEmail != nil {
		return Result{}, ErrEmailAlreadyExists
	}

	// creation user
	usr, err := user.NewUser(
		cmd.ExternalID,
		cmd.Username,
		cmd.Email,
		cmd.DisplayName,
	)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create user: %w", err)
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

func (uc *RegisterUserUseCase) validate(cmd RegisterUserCommand) error {
	if err := appcore.ValidateRequired("externalID", cmd.ExternalID); err != nil {
		return err
	}
	if err := appcore.ValidateRequired("username", cmd.Username); err != nil {
		return err
	}
	if err := appcore.ValidateEmail("email", cmd.Email); err != nil {
		return err
	}
	if err := appcore.ValidateRequired("displayName", cmd.DisplayName); err != nil {
		return err
	}
	return nil
}
