//nolint:dupl // Separate use case with different query parameters
package user

import (
	"context"
	"fmt"

	"github.com/lllypuk/teams-up/internal/application/shared"
	"github.com/lllypuk/teams-up/internal/domain/user"
)

// GetUserByUsernameUseCase обрабатывает поиск пользователя по username
type GetUserByUsernameUseCase struct {
	userRepo user.Repository
}

// NewGetUserByUsernameUseCase создает новый GetUserByUsernameUseCase
func NewGetUserByUsernameUseCase(userRepo user.Repository) *GetUserByUsernameUseCase {
	return &GetUserByUsernameUseCase{userRepo: userRepo}
}

// Execute выполняет поиск пользователя по username
func (uc *GetUserByUsernameUseCase) Execute(
	ctx context.Context,
	query GetUserByUsernameQuery,
) (Result, error) {
	// Валидация
	if err := uc.validate(query); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// Поиск пользователя
	usr, err := uc.userRepo.FindByUsername(ctx, query.Username)
	if err != nil {
		return Result{}, ErrUserNotFound
	}

	return Result{
		Result: shared.Result[*user.User]{
			Value: usr,
		},
	}, nil
}

func (uc *GetUserByUsernameUseCase) validate(query GetUserByUsernameQuery) error {
	if err := shared.ValidateRequired("username", query.Username); err != nil {
		return err
	}
	return nil
}
