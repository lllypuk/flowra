package user

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/user"
)

// GetUserUseCase обрабатывает получение пользователя по ID
type GetUserUseCase struct {
	userRepo Repository
}

// NewGetUserUseCase создает новый GetUserUseCase
func NewGetUserUseCase(userRepo Repository) *GetUserUseCase {
	return &GetUserUseCase{userRepo: userRepo}
}

// Execute выполняет получение пользователя
func (uc *GetUserUseCase) Execute(
	ctx context.Context,
	query GetUserQuery,
) (Result, error) {
	// Валидация
	if err := uc.validate(query); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// Поиск пользователя
	usr, err := uc.userRepo.FindByID(ctx, query.UserID)
	if err != nil {
		return Result{}, ErrUserNotFound
	}

	return Result{
		Result: shared.Result[*user.User]{
			Value: usr,
		},
	}, nil
}

func (uc *GetUserUseCase) validate(query GetUserQuery) error {
	if err := shared.ValidateUUID("userID", query.UserID); err != nil {
		return err
	}
	return nil
}
