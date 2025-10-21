package notification

import (
	"context"
	"fmt"

	"github.com/lllypuk/teams-up/internal/application/shared"
	"github.com/lllypuk/teams-up/internal/domain/notification"
)

const (
	// maxNotificationsToMarkAsRead - максимальное количество notifications для пометки как прочитанных за один раз
	maxNotificationsToMarkAsRead = 1000
)

// MarkAllAsReadUseCase обрабатывает пометку всех notifications пользователя как прочитанных
type MarkAllAsReadUseCase struct {
	notificationRepo notification.Repository
}

// NewMarkAllAsReadUseCase создает новый use case для пометки всех notifications как прочитанных
func NewMarkAllAsReadUseCase(
	notificationRepo notification.Repository,
) *MarkAllAsReadUseCase {
	return &MarkAllAsReadUseCase{
		notificationRepo: notificationRepo,
	}
}

// Execute выполняет пометку всех notifications пользователя как прочитанных
func (uc *MarkAllAsReadUseCase) Execute(
	ctx context.Context,
	cmd MarkAllAsReadCommand,
) (CountResult, error) {
	// Валидация
	if err := uc.validate(cmd); err != nil {
		return CountResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// Получение всех непрочитанных notifications
	notifications, err := uc.notificationRepo.FindUnreadByUserID(ctx, cmd.UserID, maxNotificationsToMarkAsRead)
	if err != nil {
		return CountResult{}, fmt.Errorf("failed to find unread notifications: %w", err)
	}

	// Пометка всех как прочитанных
	markedCount := 0
	for _, notif := range notifications {
		if markErr := notif.MarkAsRead(); markErr != nil {
			// Пропускаем уже прочитанные (не должно быть, но на всякий случай)
			continue
		}

		if saveErr := uc.notificationRepo.Save(ctx, notif); saveErr != nil {
			return CountResult{}, fmt.Errorf("failed to save notification: %w", saveErr)
		}
		markedCount++
	}

	return CountResult{
		Count: markedCount,
	}, nil
}

// validate проверяет валидность команды
func (uc *MarkAllAsReadUseCase) validate(cmd MarkAllAsReadCommand) error {
	if err := shared.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	return nil
}
