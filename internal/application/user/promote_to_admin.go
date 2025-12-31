package user

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/user"
)

// PromoteToAdminUseCase обрабатывает повышение пользователя до администратора
type PromoteToAdminUseCase struct {
	userRepo Repository
}

// NewPromoteToAdminUseCase создает новый PromoteToAdminUseCase
func NewPromoteToAdminUseCase(userRepo Repository) *PromoteToAdminUseCase {
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
