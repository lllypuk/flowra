package user

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/user"
)

// RegisterUserUseCase обрабатывает регистрацию нового пользователя
type RegisterUserUseCase struct {
	userRepo Repository
}

// NewRegisterUserUseCase создает новый RegisterUserUseCase
func NewRegisterUserUseCase(userRepo Repository) *RegisterUserUseCase {
	return &RegisterUserUseCase{userRepo: userRepo}
}

// Execute выполняет регистрацию пользователя
func (uc *RegisterUserUseCase) Execute(
	ctx context.Context,
	cmd RegisterUserCommand,
) (Result, error) {
	// Валидация
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// Проверка уникальности username
	existing, err := uc.userRepo.FindByUsername(ctx, cmd.Username)
	if err == nil && existing != nil {
		return Result{}, ErrUsernameAlreadyExists
	}

	// Проверка уникальности email
	existingByEmail, err := uc.userRepo.FindByEmail(ctx, cmd.Email)
	if err == nil && existingByEmail != nil {
		return Result{}, ErrEmailAlreadyExists
	}

	// Создание пользователя
	usr, err := user.NewUser(
		cmd.ExternalID,
		cmd.Username,
		cmd.Email,
		cmd.DisplayName,
	)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create user: %w", err)
	}

	// Сохранение
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
