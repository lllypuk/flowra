package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/flowra/flowra/internal/application/shared"
	"github.com/flowra/flowra/internal/domain/user"
)

// UpdateProfileUseCase обрабатывает обновление профиля пользователя
type UpdateProfileUseCase struct {
	userRepo user.Repository
}

// NewUpdateProfileUseCase создает новый UpdateProfileUseCase
func NewUpdateProfileUseCase(userRepo user.Repository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{userRepo: userRepo}
}

// Execute выполняет обновление профиля
func (uc *UpdateProfileUseCase) Execute(
	ctx context.Context,
	cmd UpdateProfileCommand,
) (Result, error) {
	// Валидация
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// Загрузка пользователя
	usr, err := uc.userRepo.FindByID(ctx, cmd.UserID)
	if err != nil {
		return Result{}, ErrUserNotFound
	}

	// Проверка уникальности email если он меняется
	if cmd.Email != nil {
		existingByEmail, emailErr := uc.userRepo.FindByEmail(ctx, *cmd.Email)
		if emailErr == nil && existingByEmail != nil && existingByEmail.ID() != usr.ID() {
			return Result{}, ErrEmailAlreadyExists
		}
	}

	// Обновление профиля
	if updateErr := usr.UpdateProfile(cmd.DisplayName, cmd.Email); updateErr != nil {
		return Result{}, fmt.Errorf("failed to update profile: %w", updateErr)
	}

	// Сохранение
	if saveErr := uc.userRepo.Save(ctx, usr); saveErr != nil {
		return Result{}, fmt.Errorf("failed to save user: %w", saveErr)
	}

	return Result{
		Result: shared.Result[*user.User]{
			Value: usr,
		},
	}, nil
}

func (uc *UpdateProfileUseCase) validate(cmd UpdateProfileCommand) error {
	if err := shared.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}

	// Проверяем, что хотя бы одно поле для обновления указано
	if cmd.DisplayName == nil && cmd.Email == nil {
		return errors.New("at least one field (displayName or email) must be provided")
	}

	// Валидация email если он предоставлен
	if cmd.Email != nil {
		if err := shared.ValidateEmail("email", *cmd.Email); err != nil {
			return err
		}
	}

	// Валидация displayName если он предоставлен
	if cmd.DisplayName != nil && *cmd.DisplayName == "" {
		return shared.NewValidationError("displayName", "cannot be empty")
	}

	return nil
}
