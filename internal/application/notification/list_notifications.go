package notification

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/notification"
)

// ListNotificationsUseCase обрабатывает получение списка notifications пользователя
type ListNotificationsUseCase struct {
	notificationRepo Repository
}

// NewListNotificationsUseCase создает новый use case для получения списка notifications
func NewListNotificationsUseCase(
	notificationRepo Repository,
) *ListNotificationsUseCase {
	return &ListNotificationsUseCase{
		notificationRepo: notificationRepo,
	}
}

// Execute выполняет получение списка notifications пользователя
func (uc *ListNotificationsUseCase) Execute(
	ctx context.Context,
	query ListNotificationsQuery,
) (ListResult, error) {
	// Валидация
	if err := uc.validate(query); err != nil {
		return ListResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// Дефолтные значения для пагинации
	limit := query.Limit
	if limit == 0 || limit > 100 {
		limit = 50
	}

	offset := max(query.Offset, 0)

	// Получение notifications
	var notifications []*notification.Notification
	var err error

	if query.UnreadOnly {
		notifications, err = uc.notificationRepo.FindUnreadByUserID(
			ctx,
			query.UserID,
			limit,
		)
	} else {
		notifications, err = uc.notificationRepo.FindByUserID(
			ctx,
			query.UserID,
			offset,
			limit,
		)
	}

	if err != nil {
		return ListResult{}, fmt.Errorf("failed to fetch notifications: %w", err)
	}

	// Получаем общее количество (для пагинации)
	var totalCount int
	if query.UnreadOnly {
		totalCount, err = uc.notificationRepo.CountUnreadByUserID(ctx, query.UserID)
	} else {
		// Для всех notifications мы можем использовать длину результата
		// В реальном приложении здесь должен быть отдельный метод CountByUserID
		totalCount = len(notifications)
	}

	if err != nil {
		return ListResult{}, fmt.Errorf("failed to count notifications: %w", err)
	}

	return ListResult{
		Notifications: notifications,
		TotalCount:    totalCount,
		Offset:        offset,
		Limit:         limit,
	}, nil
}

// validate проверяет валидность запроса
func (uc *ListNotificationsUseCase) validate(query ListNotificationsQuery) error {
	if err := shared.ValidateUUID("userID", query.UserID); err != nil {
		return err
	}
	if query.Limit < 0 {
		return shared.NewValidationError("limit", "must be non-negative")
	}
	if query.Offset < 0 {
		return shared.NewValidationError("offset", "must be non-negative")
	}
	return nil
}
