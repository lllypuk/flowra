package notification

import (
	"context"
	"errors"
	"fmt"

	"github.com/lllypuk/teams-up/internal/application/shared"
	"github.com/lllypuk/teams-up/internal/domain/errs"
	"github.com/lllypuk/teams-up/internal/domain/notification"
)

// MarkAsReadUseCase обрабатывает пометку notification как прочитанного
type MarkAsReadUseCase struct {
	notificationRepo notification.Repository
}

// NewMarkAsReadUseCase создает новый use case для пометки notification как прочитанного
func NewMarkAsReadUseCase(
	notificationRepo notification.Repository,
) *MarkAsReadUseCase {
	return &MarkAsReadUseCase{
		notificationRepo: notificationRepo,
	}
}

// Execute выполняет пометку notification как прочитанного
func (uc *MarkAsReadUseCase) Execute(
	ctx context.Context,
	cmd MarkAsReadCommand,
) (Result, error) {
	// Валидация
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// Получение notification
	notif, err := uc.notificationRepo.FindByID(ctx, cmd.NotificationID)
	if err != nil {
		return Result{}, fmt.Errorf("failed to find notification: %w", ErrNotificationNotFound)
	}

	// Проверка принадлежности
	if notif.UserID() != cmd.UserID {
		return Result{}, ErrNotificationAccessDenied
	}

	// Пометка как прочитанного
	if markErr := notif.MarkAsRead(); markErr != nil {
		if errors.Is(markErr, errs.ErrInvalidState) {
			return Result{}, ErrNotificationAlreadyRead
		}
		return Result{}, fmt.Errorf("failed to mark as read: %w", markErr)
	}

	// Сохранение
	if saveErr := uc.notificationRepo.Save(ctx, notif); saveErr != nil {
		return Result{}, fmt.Errorf("failed to save notification: %w", saveErr)
	}

	return Result{
		Result: shared.Result[*notification.Notification]{
			Value: notif,
		},
	}, nil
}

// validate проверяет валидность команды
func (uc *MarkAsReadUseCase) validate(cmd MarkAsReadCommand) error {
	if err := shared.ValidateUUID("notificationID", cmd.NotificationID); err != nil {
		return err
	}
	if err := shared.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	return nil
}
