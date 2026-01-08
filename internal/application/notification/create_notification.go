package notification

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/notification"
)

// CreateNotificationUseCase handles notification creation
type CreateNotificationUseCase struct {
	notificationRepo Repository
}

// NewCreateNotificationUseCase creates New use case for creating notification
func NewCreateNotificationUseCase(
	notificationRepo Repository,
) *CreateNotificationUseCase {
	return &CreateNotificationUseCase{
		notificationRepo: notificationRepo,
	}
}

// Execute performs notification creation
func (uc *CreateNotificationUseCase) Execute(
	ctx context.Context,
	cmd CreateNotificationCommand,
) (Result, error) {
	// validation
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// notification creation
	notif, err := notification.NewNotification(
		cmd.UserID,
		cmd.Type,
		cmd.Title,
		cmd.Message,
		cmd.ResourceID,
	)
	if err != nil {
		return Result{}, fmt.Errorf("failed to create notification: %w", err)
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
func (uc *CreateNotificationUseCase) validate(cmd CreateNotificationCommand) error {
	if err := appcore.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	if err := appcore.ValidateEnum("type", string(cmd.Type), []string{
		string(notification.TypeTaskStatusChanged),
		string(notification.TypeTaskAssigned),
		string(notification.TypeTaskCreated),
		string(notification.TypeChatMention),
		string(notification.TypeChatMessage),
		string(notification.TypeWorkspaceInvite),
		string(notification.TypeSystem),
	}); err != nil {
		return err
	}
	if err := appcore.ValidateRequired("title", cmd.Title); err != nil {
		return err
	}
	if err := appcore.ValidateRequired("message", cmd.Message); err != nil {
		return err
	}
	return nil
}
