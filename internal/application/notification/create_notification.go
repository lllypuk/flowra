package notification

import (
	"context"
	"fmt"

	"github.com/flowra/flowra/internal/application/shared"
	"github.com/flowra/flowra/internal/domain/notification"
)

// CreateNotificationUseCase обрабатывает создание notification
type CreateNotificationUseCase struct {
	notificationRepo notification.Repository
}

// NewCreateNotificationUseCase создает новый use case для создания notification
func NewCreateNotificationUseCase(
	notificationRepo notification.Repository,
) *CreateNotificationUseCase {
	return &CreateNotificationUseCase{
		notificationRepo: notificationRepo,
	}
}

// Execute выполняет создание notification
func (uc *CreateNotificationUseCase) Execute(
	ctx context.Context,
	cmd CreateNotificationCommand,
) (Result, error) {
	// Валидация
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// Создание notification
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
func (uc *CreateNotificationUseCase) validate(cmd CreateNotificationCommand) error {
	if err := shared.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	if err := shared.ValidateEnum("type", string(cmd.Type), []string{
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
	if err := shared.ValidateRequired("title", cmd.Title); err != nil {
		return err
	}
	if err := shared.ValidateRequired("message", cmd.Message); err != nil {
		return err
	}
	return nil
}
