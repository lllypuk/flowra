package notification

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

// CountUnreadUseCase handles подсчет unread notifications user
type CountUnreadUseCase struct {
	notificationRepo Repository
}

// NewCountUnreadUseCase creates New use case for подсчета unread notifications
func NewCountUnreadUseCase(
	notificationRepo Repository,
) *CountUnreadUseCase {
	return &CountUnreadUseCase{
		notificationRepo: notificationRepo,
	}
}

// Execute performs подсчет unread notifications user
func (uc *CountUnreadUseCase) Execute(
	ctx context.Context,
	query CountUnreadQuery,
) (CountResult, error) {
	// validation
	if err := uc.validate(query); err != nil {
		return CountResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// Подсчет unread
	count, err := uc.notificationRepo.CountUnreadByUserID(ctx, query.UserID)
	if err != nil {
		return CountResult{}, fmt.Errorf("failed to count unread notifications: %w", err)
	}

	return CountResult{
		Count: count,
	}, nil
}

// validate validates request
func (uc *CountUnreadUseCase) validate(query CountUnreadQuery) error {
	if err := appcore.ValidateUUID("userID", query.UserID); err != nil {
		return err
	}
	return nil
}
