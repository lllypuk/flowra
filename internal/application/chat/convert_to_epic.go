//nolint:dupl // Use case pattern requires similar structure
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

// ConvertToEpicUseCase обрабатывает конвертацию чата в Epic
type ConvertToEpicUseCase struct {
	eventStore appcore.EventStore
}

// NewConvertToEpicUseCase создает новый ConvertToEpicUseCase
func NewConvertToEpicUseCase(eventStore appcore.EventStore) *ConvertToEpicUseCase {
	return &ConvertToEpicUseCase{
		eventStore: eventStore,
	}
}

// Execute выполняет конвертацию в Epic
func (uc *ConvertToEpicUseCase) Execute(ctx context.Context, cmd ConvertToEpicCommand) (Result, error) {
	// Валидация
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := loadAggregate(ctx, uc.eventStore, cmd.ChatID)
	if err != nil {
		return Result{}, err
	}

	if convertErr := chatAggregate.ConvertToEpic(cmd.Title, cmd.ConvertedBy); convertErr != nil {
		return Result{}, fmt.Errorf("failed to convert to epic: %w", convertErr)
	}

	return saveAggregate(ctx, uc.eventStore, chatAggregate, cmd.ChatID.String())
}

func (uc *ConvertToEpicUseCase) validate(cmd ConvertToEpicCommand) error {
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
