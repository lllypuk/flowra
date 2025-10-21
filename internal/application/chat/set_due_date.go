package chat

import (
	"context"
	"fmt"

	"github.com/flowra/flowra/internal/application/shared"
)

// SetDueDateUseCase обрабатывает установку дедлайна
type SetDueDateUseCase struct {
	eventStore shared.EventStore
}

// NewSetDueDateUseCase создает новый SetDueDateUseCase
func NewSetDueDateUseCase(eventStore shared.EventStore) *SetDueDateUseCase {
	return &SetDueDateUseCase{eventStore: eventStore}
}

// Execute выполняет установку дедлайна
func (uc *SetDueDateUseCase) Execute(ctx context.Context, cmd SetDueDateCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := loadAggregate(ctx, uc.eventStore, cmd.ChatID)
	if err != nil {
		return Result{}, err
	}

	if setErr := chatAggregate.SetDueDate(cmd.DueDate, cmd.SetBy); setErr != nil {
		return Result{}, fmt.Errorf("failed to set due date: %w", setErr)
	}

	return saveAggregate(ctx, uc.eventStore, chatAggregate, cmd.ChatID.String())
}

func (uc *SetDueDateUseCase) validate(cmd SetDueDateCommand) error {
	if err := shared.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := shared.ValidateUUID("setBy", cmd.SetBy); err != nil {
		return err
	}
	return nil
}
