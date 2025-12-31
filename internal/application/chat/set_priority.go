//nolint:dupl // Use case pattern requires similar structure
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

// SetPriorityUseCase обрабатывает установку приоритета
type SetPriorityUseCase struct {
	eventStore appcore.EventStore
}

// NewSetPriorityUseCase создает новый SetPriorityUseCase
func NewSetPriorityUseCase(eventStore appcore.EventStore) *SetPriorityUseCase {
	return &SetPriorityUseCase{
		eventStore: eventStore,
	}
}

// Execute выполняет установку приоритета
func (uc *SetPriorityUseCase) Execute(ctx context.Context, cmd SetPriorityCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := loadAggregate(ctx, uc.eventStore, cmd.ChatID)
	if err != nil {
		return Result{}, err
	}

	if setErr := chatAggregate.SetPriority(cmd.Priority, cmd.SetBy); setErr != nil {
		return Result{}, fmt.Errorf("failed to set priority: %w", setErr)
	}

	return saveAggregate(ctx, uc.eventStore, chatAggregate, cmd.ChatID.String())
}

func (uc *SetPriorityUseCase) validate(cmd SetPriorityCommand) error {
	if err := appcore.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := appcore.ValidateRequired("priority", cmd.Priority); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("setBy", cmd.SetBy); err != nil {
		return err
	}
	return nil
}
