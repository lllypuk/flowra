//nolint:dupl // Use case pattern requires similar structure
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/teams-up/internal/application/shared"
)

// SetSeverityUseCase обрабатывает установку severity (только для Bug)
type SetSeverityUseCase struct {
	eventStore shared.EventStore
}

// NewSetSeverityUseCase создает новый SetSeverityUseCase
func NewSetSeverityUseCase(eventStore shared.EventStore) *SetSeverityUseCase {
	return &SetSeverityUseCase{eventStore: eventStore}
}

// Execute выполняет установку severity
func (uc *SetSeverityUseCase) Execute(ctx context.Context, cmd SetSeverityCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := loadAggregate(ctx, uc.eventStore, cmd.ChatID)
	if err != nil {
		return Result{}, err
	}

	if setErr := chatAggregate.SetSeverity(cmd.Severity, cmd.SetBy); setErr != nil {
		return Result{}, fmt.Errorf("failed to set severity: %w", setErr)
	}

	return saveAggregate(ctx, uc.eventStore, chatAggregate, cmd.ChatID.String())
}

func (uc *SetSeverityUseCase) validate(cmd SetSeverityCommand) error {
	if err := shared.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := shared.ValidateRequired("severity", cmd.Severity); err != nil {
		return err
	}
	if err := shared.ValidateUUID("setBy", cmd.SetBy); err != nil {
		return err
	}
	return nil
}
