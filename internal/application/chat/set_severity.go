//nolint:dupl // Use case pattern requires similar structure
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
)

// SetSeverityUseCase handles setting severity (only for Bug)
type SetSeverityUseCase struct {
	chatRepo CommandRepository
}

// NewSetSeverityUseCase creates a new SetSeverityUseCase
func NewSetSeverityUseCase(chatRepo CommandRepository) *SetSeverityUseCase {
	return &SetSeverityUseCase{chatRepo: chatRepo}
}

// Execute performs setting severity
func (uc *SetSeverityUseCase) Execute(ctx context.Context, cmd SetSeverityCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := uc.chatRepo.Load(ctx, cmd.ChatID)
	if err != nil {
		return Result{}, fmt.Errorf("failed to load chat: %w", err)
	}

	if setErr := chatAggregate.SetSeverity(cmd.Severity, cmd.SetBy); setErr != nil {
		return Result{}, fmt.Errorf("failed to set severity: %w", setErr)
	}

	// Save via repository (updates both event store and read model)
	if err = uc.chatRepo.Save(ctx, chatAggregate); err != nil {
		return Result{}, fmt.Errorf("failed to save chat: %w", err)
	}

	return Result{
		Result: appcore.Result[*chat.Chat]{
			Value:   chatAggregate,
			Version: chatAggregate.Version(),
		},
	}, nil
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
