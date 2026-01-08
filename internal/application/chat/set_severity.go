//nolint:dupl // Use case pattern requires similar structure
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

// SetSeverityUseCase handles setting severity (only for Bug)
type SetSeverityUseCase struct {
	eventStore appcore.EventStore
}

// NewSetSeverityUseCase creates a new SetSeverityUseCase
func NewSetSeverityUseCase(eventStore appcore.EventStore) *SetSeverityUseCase {
	return &SetSeverityUseCase{eventStore: eventStore}
}

// Execute performs setting severity
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
	if err := appcore.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := appcore.ValidateRequired("severity", cmd.Severity); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("setBy", cmd.SetBy); err != nil {
		return err
	}
	return nil
}
