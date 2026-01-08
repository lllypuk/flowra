package notification

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/notification"
)

// ListNotificationsUseCase handles retrieval list notifications user
type ListNotificationsUseCase struct {
	notificationRepo Repository
}

// NewListNotificationsUseCase creates New use case for receiv list notifications
func NewListNotificationsUseCase(
	notificationRepo Repository,
) *ListNotificationsUseCase {
	return &ListNotificationsUseCase{
		notificationRepo: notificationRepo,
	}
}

// Execute performs retrieval list notifications user
func (uc *ListNotificationsUseCase) Execute(
	ctx context.Context,
	query ListNotificationsQuery,
) (ListResult, error) {
	// validation
	if err := uc.validate(query); err != nil {
		return ListResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// defoltnye values for paginatsii
	limit := query.Limit
	if limit == 0 || limit > 100 {
		limit = 50
	}

	offset := max(query.Offset, 0)

	// retrieval notifications
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

	// poluchaem obschee count (for paginatsii)
	var totalCount int
	if query.UnreadOnly {
		totalCount, err = uc.notificationRepo.CountUnreadByUserID(ctx, query.UserID)
	} else {
		// for all notifications my mozhem user length result
		// in realnom prilozhenii zdes dolzhen byt otdelnyy method CountByUserID
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

// validate validates request
func (uc *ListNotificationsUseCase) validate(query ListNotificationsQuery) error {
	if err := appcore.ValidateUUID("userID", query.UserID); err != nil {
		return err
	}
	if query.Limit < 0 {
		return appcore.NewValidationError("limit", "must be non-negative")
	}
	if query.Offset < 0 {
		return appcore.NewValidationError("offset", "must be non-negative")
	}
	return nil
}
