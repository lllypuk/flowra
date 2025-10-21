//nolint:dupl // Use case pattern requires similar structure
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/shared"
)

// ConvertToTaskUseCase обрабатывает конвертацию чата в Task
type ConvertToTaskUseCase struct {
	eventStore shared.EventStore
}

// NewConvertToTaskUseCase создает новый ConvertToTaskUseCase
func NewConvertToTaskUseCase(eventStore shared.EventStore) *ConvertToTaskUseCase {
	return &ConvertToTaskUseCase{
		eventStore: eventStore,
	}
}

// Execute выполняет конвертацию в Task
func (uc *ConvertToTaskUseCase) Execute(ctx context.Context, cmd ConvertToTaskCommand) (Result, error) {
	// Валидация
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
