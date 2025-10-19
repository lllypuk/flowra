package user

import (
	"context"
	"fmt"

	"github.com/lllypuk/teams-up/internal/application/shared"
	"github.com/lllypuk/teams-up/internal/domain/user"
)

// PromoteToAdminUseCase обрабатывает повышение пользователя до администратора
type PromoteToAdminUseCase struct {
	userRepo user.Repository
}

// NewPromoteToAdminUseCase создает новый PromoteToAdminUseCase
func NewPromoteToAdminUseCase(userRepo user.Repository) *PromoteToAdminUseCase {
	return &PromoteToAdminUseCase{userRepo: userRepo}
}

// Execute выполняет повышение до администратора
func (uc *PromoteToAdminUseCase) Execute(
	ctx context.Context,
	cmd PromoteToAdminCommand,
) (Result, error) {
	// Валидация
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// Проверка прав выполняющего операцию
	promoter, err := uc.userRepo.FindByID(ctx, cmd.PromotedBy)
	if err != nil {
		return Result{}, ErrUserNotFound
	}

	if !promoter.IsSystemAdmin() {
		return Result{}, ErrNotSystemAdmin
	}

	// Загрузка целевого пользователя
	targetUser, targetErr := uc.userRepo.FindByID(ctx, cmd.UserID)
	if targetErr != nil {
		return Result{}, ErrUserNotFound
	}

	// Установка прав администратора
	targetUser.SetAdmin(true)

	// Сохранение
	if saveErr := uc.userRepo.Save(ctx, targetUser); saveErr != nil {
		return Result{}, fmt.Errorf("failed to save user: %w", saveErr)
	}

	return Result{
		Result: shared.Result[*user.User]{
			Value: targetUser,
		},
	}, nil
}

func (uc *PromoteToAdminUseCase) validate(cmd PromoteToAdminCommand) error {
	if err := shared.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	if err := shared.ValidateUUID("promotedBy", cmd.PromotedBy); err != nil {
		return err
	}
	return nil
}
