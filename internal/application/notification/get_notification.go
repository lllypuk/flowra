package notification

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/notification"
)

// GetNotificationUseCase обрабатывает получение notification по ID
type GetNotificationUseCase struct {
	notificationRepo Repository
}

// NewGetNotificationUseCase создает новый use case для получения notification
func NewGetNotificationUseCase(
	notificationRepo Repository,
) *GetNotificationUseCase {
	return &GetNotificationUseCase{
		notificationRepo: notificationRepo,
	}
}

// Execute выполняет получение notification по ID
func (uc *GetNotificationUseCase) Execute(
	ctx context.Context,
	query GetNotificationQuery,
) (Result, error) {
	// Валидация
	if err := uc.validate(query); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// Получение notification
	notif, err := uc.notificationRepo.FindByID(ctx, query.NotificationID)
	if err != nil {
		return Result{}, fmt.Errorf("failed to find notification: %w", ErrNotificationNotFound)
	}

	// Проверка принадлежности
	if notif.UserID() != query.UserID {
		return Result{}, ErrNotificationAccessDenied
	}

	return Result{
		Result: shared.Result[*notification.Notification]{
			Value: notif,
		},
	}, nil
}

// validate проверяет валидность запроса
func (uc *GetNotificationUseCase) validate(query GetNotificationQuery) error {
	if err := shared.ValidateUUID("notificationID", query.NotificationID); err != nil {
		return err
	}
	if err := shared.ValidateUUID("userID", query.UserID); err != nil {
		return err
	}
	return nil
}
