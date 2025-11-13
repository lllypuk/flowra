package user

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/shared"
)

const (
	// MaxListLimit максимальное количество пользователей в одном запросе
	MaxListLimit = 100
)

// ListUsersUseCase обрабатывает получение списка пользователей
type ListUsersUseCase struct {
	userRepo Repository
}

// NewListUsersUseCase создает новый ListUsersUseCase
func NewListUsersUseCase(userRepo Repository) *ListUsersUseCase {
	return &ListUsersUseCase{userRepo: userRepo}
}

// Execute выполняет получение списка пользователей
func (uc *ListUsersUseCase) Execute(
	ctx context.Context,
	query ListUsersQuery,
) (UsersListResult, error) {
	// Валидация
	if err := uc.validate(query); err != nil {
		return UsersListResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// Получение общего количества
	totalCount, err := uc.userRepo.Count(ctx)
	if err != nil {
		return UsersListResult{}, fmt.Errorf("failed to get users count: %w", err)
	}

	// Получение списка пользователей
	users, err := uc.userRepo.List(ctx, query.Offset, query.Limit)
	if err != nil {
		return UsersListResult{}, fmt.Errorf("failed to list users: %w", err)
	}

	return UsersListResult{
		Users:      users,
		TotalCount: totalCount,
		Offset:     query.Offset,
		Limit:      query.Limit,
	}, nil
}

func (uc *ListUsersUseCase) validate(query ListUsersQuery) error {
	if err := shared.ValidateNonNegative("offset", query.Offset); err != nil {
		return err
	}
	if err := shared.ValidatePositive("limit", query.Limit); err != nil {
		return err
	}
	if err := shared.ValidateRange("limit", query.Limit, 1, MaxListLimit); err != nil {
		return err
	}
	return nil
}
