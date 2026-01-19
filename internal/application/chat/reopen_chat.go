//nolint:dupl // Similar structure to CloseChatUseCase but different domain logic
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

// ReopenChatUseCase handles reopening a closed chat
type ReopenChatUseCase struct {
	eventStore appcore.EventStore
}

// NewReopenChatUseCase creates a new ReopenChatUseCase
func NewReopenChatUseCase(eventStore appcore.EventStore) *ReopenChatUseCase {
	return &ReopenChatUseCase{
		eventStore: eventStore,
	}
}

// Execute performs reopening a chat
func (uc *ReopenChatUseCase) Execute(ctx context.Context, cmd ReopenChatCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := loadAggregate(ctx, uc.eventStore, cmd.ChatID)
	if err != nil {
		return Result{}, err
	}

	if reopenErr := chatAggregate.Reopen(cmd.ReopenedBy); reopenErr != nil {
		return Result{}, fmt.Errorf("failed to reopen chat: %w", reopenErr)
	}

	return saveAggregate(ctx, uc.eventStore, chatAggregate, cmd.ChatID.String())
}

func (uc *ReopenChatUseCase) validate(cmd ReopenChatCommand) error {
	if err := appcore.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("reopenedBy", cmd.ReopenedBy); err != nil {
		return err
	}
	return nil
}
