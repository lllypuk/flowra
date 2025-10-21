//nolint:dupl // Use case pattern requires similar structure
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/shared"
)

// ConvertToBugUseCase обрабатывает конвертацию чата в Bug
type ConvertToBugUseCase struct {
	eventStore shared.EventStore
}

// NewConvertToBugUseCase создает новый ConvertToBugUseCase
func NewConvertToBugUseCase(eventStore shared.EventStore) *ConvertToBugUseCase {
	return &ConvertToBugUseCase{
		eventStore: eventStore,
	}
}

// Execute выполняет конвертацию в Bug
func (uc *ConvertToBugUseCase) Execute(ctx context.Context, cmd ConvertToBugCommand) (Result, error) {
	// Валидация
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
	if err := shared.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := shared.ValidateRequired("title", cmd.Title); err != nil {
		return err
	}
	if err := shared.ValidateMaxLength("title", cmd.Title, shared.MaxTitleLength); err != nil {
		return err
	}
	if err := shared.ValidateUUID("convertedBy", cmd.ConvertedBy); err != nil {
		return err
	}
	return nil
}
