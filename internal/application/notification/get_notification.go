package notification

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/notification"
)

// GetNotificationUseCase handles retrieval notification по ID
type GetNotificationUseCase struct {
	notificationRepo Repository
}

// NewGetNotificationUseCase creates New use case for receivения notification
func NewGetNotificationUseCase(
	notificationRepo Repository,
) *GetNotificationUseCase {
	return &GetNotificationUseCase{
		notificationRepo: notificationRepo,
	}
}

// Execute performs retrieval notification по ID
func (uc *GetNotificationUseCase) Execute(
	ctx context.Context,
	query GetNotificationQuery,
) (Result, error) {
	// validation
	if err := uc.validate(query); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// retrieval notification
	notif, err := uc.notificationRepo.FindByID(ctx, query.NotificationID)
	if err != nil {
		return Result{}, fmt.Errorf("failed to find notification: %w", ErrNotificationNotFound)
	}

	// check принадлежности
	if notif.UserID() != query.UserID {
		return Result{}, ErrNotificationAccessDenied
	}

	return Result{
		Result: appcore.Result[*notification.Notification]{
			Value: notif,
		},
	}, nil
}

// validate validates request
func (uc *GetNotificationUseCase) validate(query GetNotificationQuery) error {
	if err := appcore.ValidateUUID("notificationID", query.NotificationID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("userID", query.UserID); err != nil {
		return err
	}
	return nil
}
