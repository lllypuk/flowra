package notification

import (
	"context"
	"errors"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/notification"
)

// MarkAsReadUseCase handles пометку notification as read
type MarkAsReadUseCase struct {
	notificationRepo Repository
}

// NewMarkAsReadUseCase creates New use case for пометки notification as read
func NewMarkAsReadUseCase(
	notificationRepo Repository,
) *MarkAsReadUseCase {
	return &MarkAsReadUseCase{
		notificationRepo: notificationRepo,
	}
}

// Execute performs пометку notification as read
func (uc *MarkAsReadUseCase) Execute(
	ctx context.Context,
	cmd MarkAsReadCommand,
) (Result, error) {
	// validation
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// retrieval notification
	notif, err := uc.notificationRepo.FindByID(ctx, cmd.NotificationID)
	if err != nil {
		return Result{}, fmt.Errorf("failed to find notification: %w", ErrNotificationNotFound)
	}

	// check принадлежности
	if notif.UserID() != cmd.UserID {
		return Result{}, ErrNotificationAccessDenied
	}

	// Пометка as read
	if markErr := notif.MarkAsRead(); markErr != nil {
		if errors.Is(markErr, errs.ErrInvalidState) {
			return Result{}, ErrNotificationAlreadyRead
		}
		return Result{}, fmt.Errorf("failed to mark as read: %w", markErr)
	}

	// storage
	if saveErr := uc.notificationRepo.Save(ctx, notif); saveErr != nil {
		return Result{}, fmt.Errorf("failed to save notification: %w", saveErr)
	}

	return Result{
		Result: appcore.Result[*notification.Notification]{
			Value: notif,
		},
	}, nil
}

// validate validates commands
func (uc *MarkAsReadUseCase) validate(cmd MarkAsReadCommand) error {
	if err := appcore.ValidateUUID("notificationID", cmd.NotificationID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	return nil
}
