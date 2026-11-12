//nolint:dupl // Similar structure to CloseChatUseCase but different domain logic
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
)

// ReopenChatUseCase handles reopening a closed chat
type ReopenChatUseCase struct {
	chatRepo CommandRepository
}

// NewReopenChatUseCase creates a new ReopenChatUseCase
func NewReopenChatUseCase(chatRepo CommandRepository) *ReopenChatUseCase {
	return &ReopenChatUseCase{chatRepo: chatRepo}
}

// Execute performs reopening a chat
func (uc *ReopenChatUseCase) Execute(ctx context.Context, cmd ReopenChatCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := uc.chatRepo.Load(ctx, cmd.ChatID)
	if err != nil {
		return Result{}, fmt.Errorf("failed to load chat: %w", err)
	}

	if reopenErr := chatAggregate.Reopen(cmd.ReopenedBy); reopenErr != nil {
		return Result{}, fmt.Errorf("failed to reopen chat: %w", reopenErr)
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

func (uc *ReopenChatUseCase) validate(cmd ReopenChatCommand) error {
	if err := appcore.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("reopenedBy", cmd.ReopenedBy); err != nil {
		return err
	}
	return nil
}
