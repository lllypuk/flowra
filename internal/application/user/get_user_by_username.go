package user

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/user"
)

// GetUserByUsernameUseCase handles search user po username
type GetUserByUsernameUseCase struct {
	userRepo Repository
}

// NewGetUserByUsernameUseCase creates New GetUserByUsernameUseCase
func NewGetUserByUsernameUseCase(userRepo Repository) *GetUserByUsernameUseCase {
	return &GetUserByUsernameUseCase{userRepo: userRepo}
}

// Execute performs search user po username
func (uc *GetUserByUsernameUseCase) Execute(
	ctx context.Context,
	query GetUserByUsernameQuery,
) (Result, error) {
	// validation
	if err := uc.validate(query); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// search user
	usr, err := uc.userRepo.FindByUsername(ctx, query.Username)
	if err != nil {
		return Result{}, ErrUserNotFound
	}

	return Result{
		Result: appcore.Result[*user.User]{
			Value: usr,
		},
	}, nil
}

func (uc *GetUserByUsernameUseCase) validate(query GetUserByUsernameQuery) error {
	if err := appcore.ValidateRequired("username", query.Username); err != nil {
		return err
	}
	return nil
}
