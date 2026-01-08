package user

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/user"
)

// GetUserUseCase handles retrieval user по ID
type GetUserUseCase struct {
	userRepo Repository
}

// NewGetUserUseCase creates New GetUserUseCase
func NewGetUserUseCase(userRepo Repository) *GetUserUseCase {
	return &GetUserUseCase{userRepo: userRepo}
}

// Execute performs retrieval user
func (uc *GetUserUseCase) Execute(
	ctx context.Context,
	query GetUserQuery,
) (Result, error) {
	// validation
	if err := uc.validate(query); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// search user
	usr, err := uc.userRepo.FindByID(ctx, query.UserID)
	if err != nil {
		return Result{}, ErrUserNotFound
	}

	return Result{
		Result: appcore.Result[*user.User]{
			Value: usr,
		},
	}, nil
}

func (uc *GetUserUseCase) validate(query GetUserQuery) error {
	if err := appcore.ValidateUUID("userID", query.UserID); err != nil {
		return err
	}
	return nil
}
