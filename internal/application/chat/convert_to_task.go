//nolint:dupl // Use case pattern requires similar structure
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

// ConvertToTaskUseCase handles converting a chat to Task
type ConvertToTaskUseCase struct {
	eventStore appcore.EventStore
}

// NewConvertToTaskUseCase creates a new ConvertToTaskUseCase
func NewConvertToTaskUseCase(eventStore appcore.EventStore) *ConvertToTaskUseCase {
	return &ConvertToTaskUseCase{
		eventStore: eventStore,
	}
}

// Execute performs converting to Task
func (uc *ConvertToTaskUseCase) Execute(ctx context.Context, cmd ConvertToTaskCommand) (Result, error) {
	// Validation
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := loadAggregate(ctx, uc.eventStore, cmd.ChatID)
	if err != nil {
		return Result{}, err
	}

	if convertErr := chatAggregate.ConvertToTask(cmd.Title, cmd.ConvertedBy); convertErr != nil {
		return Result{}, fmt.Errorf("failed to convert to task: %w", convertErr)
	}

	return saveAggregate(ctx, uc.eventStore, chatAggregate, cmd.ChatID.String())
}

func (uc *ConvertToTaskUseCase) validate(cmd ConvertToTaskCommand) error {
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
