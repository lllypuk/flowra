package notification

import (
	"context"
	"fmt"

	"github.com/flowra/flowra/internal/application/shared"
	"github.com/flowra/flowra/internal/domain/notification"
)

// CountUnreadUseCase обрабатывает подсчет непрочитанных notifications пользователя
type CountUnreadUseCase struct {
	notificationRepo notification.Repository
}

// NewCountUnreadUseCase создает новый use case для подсчета непрочитанных notifications
func NewCountUnreadUseCase(
	notificationRepo notification.Repository,
) *CountUnreadUseCase {
	return &CountUnreadUseCase{
		notificationRepo: notificationRepo,
	}
}

// Execute выполняет подсчет непрочитанных notifications пользователя
func (uc *CountUnreadUseCase) Execute(
	ctx context.Context,
	query CountUnreadQuery,
) (CountResult, error) {
	// Валидация
	if err := uc.validate(query); err != nil {
		return CountResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// Подсчет непрочитанных
	count, err := uc.notificationRepo.CountUnreadByUserID(ctx, query.UserID)
	if err != nil {
		return CountResult{}, fmt.Errorf("failed to count unread notifications: %w", err)
	}

	return CountResult{
		Count: count,
	}, nil
}

// validate проверяет валидность запроса
func (uc *CountUnreadUseCase) validate(query CountUnreadQuery) error {
	if err := shared.ValidateUUID("userID", query.UserID); err != nil {
		return err
	}
	return nil
}
