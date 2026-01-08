package notification

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

const (
	// maxNotificationsToMarkAsRead - максимальное count notifications for пометки as прочитанных за one раз
	maxNotificationsToMarkAsRead = 1000
)

// MarkAllAsReadUseCase handles пометку all notifications user as прочитанных
type MarkAllAsReadUseCase struct {
	notificationRepo Repository
}

// NewMarkAllAsReadUseCase creates New use case for пометки all notifications as прочитанных
func NewMarkAllAsReadUseCase(
	notificationRepo Repository,
) *MarkAllAsReadUseCase {
	return &MarkAllAsReadUseCase{
		notificationRepo: notificationRepo,
	}
}

// Execute performs пометку all notifications user as прочитанных
func (uc *MarkAllAsReadUseCase) Execute(
	ctx context.Context,
	cmd MarkAllAsReadCommand,
) (CountResult, error) {
	// validation
	if err := uc.validate(cmd); err != nil {
		return CountResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// retrieval all unread notifications
	notifications, err := uc.notificationRepo.FindUnreadByUserID(ctx, cmd.UserID, maxNotificationsToMarkAsRead)
	if err != nil {
		return CountResult{}, fmt.Errorf("failed to find unread notifications: %w", err)
	}

	// Пометка all as прочитанных
	markedCount := 0
	for _, notif := range notifications {
		if markErr := notif.MarkAsRead(); markErr != nil {
			// Пропускаем уже прочитанные (not должно быть, но on всякий случай)
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

// validate validates commands
func (uc *MarkAllAsReadUseCase) validate(cmd MarkAllAsReadCommand) error {
	if err := appcore.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	return nil
}
