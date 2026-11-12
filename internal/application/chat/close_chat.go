//nolint:dupl // Similar structure to ReopenChatUseCase but different domain logic
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
)

// CloseChatUseCase handles closing/archiving a chat
type CloseChatUseCase struct {
	chatRepo CommandRepository
}

// NewCloseChatUseCase creates a new CloseChatUseCase
func NewCloseChatUseCase(chatRepo CommandRepository) *CloseChatUseCase {
	return &CloseChatUseCase{chatRepo: chatRepo}
}

// Execute performs closing a chat
func (uc *CloseChatUseCase) Execute(ctx context.Context, cmd CloseChatCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := uc.chatRepo.Load(ctx, cmd.ChatID)
	if err != nil {
		return Result{}, fmt.Errorf("failed to load chat: %w", err)
	}

	if closeErr := chatAggregate.Close(cmd.ClosedBy); closeErr != nil {
		return Result{}, fmt.Errorf("failed to close chat: %w", closeErr)
	}

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

func (uc *CloseChatUseCase) validate(cmd CloseChatCommand) error {
	if err := appcore.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("closedBy", cmd.ClosedBy); err != nil {
		return err
	}
	return nil
}
