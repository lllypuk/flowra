//nolint:dupl // Use case pattern requires similar structure
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

// ConvertToBugUseCase handles converting a chat to Bug
type ConvertToBugUseCase struct {
	eventStore appcore.EventStore
}

// NewConvertToBugUseCase creates a new ConvertToBugUseCase
func NewConvertToBugUseCase(eventStore appcore.EventStore) *ConvertToBugUseCase {
	return &ConvertToBugUseCase{
		eventStore: eventStore,
	}
}

// Execute performs converting to Bug
func (uc *ConvertToBugUseCase) Execute(ctx context.Context, cmd ConvertToBugCommand) (Result, error) {
	// Validation
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := loadAggregate(ctx, uc.eventStore, cmd.ChatID)
	if err != nil {
		return Result{}, err
	}

	if convertErr := chatAggregate.ConvertToBug(cmd.Title, cmd.ConvertedBy); convertErr != nil {
		return Result{}, fmt.Errorf("failed to convert to bug: %w", convertErr)
	}

	return saveAggregate(ctx, uc.eventStore, chatAggregate, cmd.ChatID.String())
}

func (uc *ConvertToBugUseCase) validate(cmd ConvertToBugCommand) error {
	if err := appcore.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := appcore.ValidateRequired("title", cmd.Title); err != nil {
		return err
	}
	if err := appcore.ValidateMaxLength("title", cmd.Title, appcore.MaxTitleLength); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("convertedBy", cmd.ConvertedBy); err != nil {
		return err
	}
	return nil
}
