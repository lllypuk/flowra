package notification

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

// DeleteNotificationUseCase handles deletion notification
type DeleteNotificationUseCase struct {
	notificationRepo Repository
}

// NewDeleteNotificationUseCase creates New use case for removing notification
func NewDeleteNotificationUseCase(
	notificationRepo Repository,
) *DeleteNotificationUseCase {
	return &DeleteNotificationUseCase{
		notificationRepo: notificationRepo,
	}
}

// Execute performs deletion notification
func (uc *DeleteNotificationUseCase) Execute(
	ctx context.Context,
	cmd DeleteNotificationCommand,
) error {
	// validation
	if err := uc.validate(cmd); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// retrieval notification for проверки принадлежности
	notif, err := uc.notificationRepo.FindByID(ctx, cmd.NotificationID)
	if err != nil {
		return fmt.Errorf("failed to find notification: %w", ErrNotificationNotFound)
	}

	// check принадлежности
	if notif.UserID() != cmd.UserID {
		return ErrNotificationAccessDenied
	}

	// deletion
	if deleteErr := uc.notificationRepo.Delete(ctx, cmd.NotificationID); deleteErr != nil {
		return fmt.Errorf("failed to delete notification: %w", deleteErr)
	}

	return nil
}

// validate validates commands
func (uc *DeleteNotificationUseCase) validate(cmd DeleteNotificationCommand) error {
	if err := appcore.ValidateUUID("notificationID", cmd.NotificationID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	return nil
}
