package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
)

// SetDueDateUseCase handles setting a deadline
type SetDueDateUseCase struct {
	chatRepo CommandRepository
}

// NewSetDueDateUseCase creates a new SetDueDateUseCase
func NewSetDueDateUseCase(chatRepo CommandRepository) *SetDueDateUseCase {
	return &SetDueDateUseCase{chatRepo: chatRepo}
}

// Execute performs setting a deadline
func (uc *SetDueDateUseCase) Execute(ctx context.Context, cmd SetDueDateCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := uc.chatRepo.Load(ctx, cmd.ChatID)
	if err != nil {
		return Result{}, fmt.Errorf("failed to load chat: %w", err)
	}

	if setErr := chatAggregate.SetDueDate(cmd.DueDate, cmd.SetBy); setErr != nil {
		return Result{}, fmt.Errorf("failed to set due date: %w", setErr)
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

func (uc *SetDueDateUseCase) validate(cmd SetDueDateCommand) error {
	if err := appcore.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("setBy", cmd.SetBy); err != nil {
		return err
	}
	return nil
}
