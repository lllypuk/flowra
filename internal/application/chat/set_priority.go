//nolint:dupl // Use case pattern requires similar structure
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
)

// SetPriorityUseCase handles setting priority
type SetPriorityUseCase struct {
	chatRepo CommandRepository
}

// NewSetPriorityUseCase creates a new SetPriorityUseCase
func NewSetPriorityUseCase(chatRepo CommandRepository) *SetPriorityUseCase {
	return &SetPriorityUseCase{
		chatRepo: chatRepo,
	}
}

// Execute performs setting the priority
func (uc *SetPriorityUseCase) Execute(ctx context.Context, cmd SetPriorityCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := uc.chatRepo.Load(ctx, cmd.ChatID)
	if err != nil {
		return Result{}, fmt.Errorf("failed to load chat: %w", err)
	}

	if setErr := chatAggregate.SetPriority(cmd.Priority, cmd.SetBy); setErr != nil {
		return Result{}, fmt.Errorf("failed to set priority: %w", setErr)
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
