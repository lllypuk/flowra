//nolint:dupl // Similar structure to ReopenChatUseCase but different domain logic
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

// CloseChatUseCase handles closing/archiving a chat
type CloseChatUseCase struct {
	eventStore appcore.EventStore
}

// NewCloseChatUseCase creates a new CloseChatUseCase
func NewCloseChatUseCase(eventStore appcore.EventStore) *CloseChatUseCase {
	return &CloseChatUseCase{
		eventStore: eventStore,
	}
}

// Execute performs closing a chat
func (uc *CloseChatUseCase) Execute(ctx context.Context, cmd CloseChatCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := loadAggregate(ctx, uc.eventStore, cmd.ChatID)
	if err != nil {
		return Result{}, err
	}

	if closeErr := chatAggregate.Close(cmd.ClosedBy); closeErr != nil {
		return Result{}, fmt.Errorf("failed to close chat: %w", closeErr)
	}

	return saveAggregate(ctx, uc.eventStore, chatAggregate, cmd.ChatID.String())
}

func (uc *CloseChatUseCase) validate(cmd CloseChatCommand) error {
	if err := appcore.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("closedBy", cmd.ClosedBy); err != nil {
		return err
	}
	return nil
}
