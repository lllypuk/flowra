package notification

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/notification"
)

// DeleteNotificationUseCase обрабатывает удаление notification
type DeleteNotificationUseCase struct {
	notificationRepo notification.Repository
}

// NewDeleteNotificationUseCase создает новый use case для удаления notification
func NewDeleteNotificationUseCase(
	notificationRepo notification.Repository,
) *DeleteNotificationUseCase {
	return &DeleteNotificationUseCase{
		notificationRepo: notificationRepo,
	}
}

// Execute выполняет удаление notification
func (uc *DeleteNotificationUseCase) Execute(
	ctx context.Context,
	cmd DeleteNotificationCommand,
) error {
	// Валидация
	if err := uc.validate(cmd); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Получение notification для проверки принадлежности
	notif, err := uc.notificationRepo.FindByID(ctx, cmd.NotificationID)
	if err != nil {
		return fmt.Errorf("failed to find notification: %w", ErrNotificationNotFound)
	}

	// Проверка принадлежности
	if notif.UserID() != cmd.UserID {
		return ErrNotificationAccessDenied
	}

	// Удаление
	if deleteErr := uc.notificationRepo.Delete(ctx, cmd.NotificationID); deleteErr != nil {
		return fmt.Errorf("failed to delete notification: %w", deleteErr)
	}

	return nil
}

// validate проверяет валидность команды
func (uc *DeleteNotificationUseCase) validate(cmd DeleteNotificationCommand) error {
	if err := shared.ValidateUUID("notificationID", cmd.NotificationID); err != nil {
		return err
	}
	if err := shared.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	return nil
}
