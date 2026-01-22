//nolint:dupl // Use case pattern requires similar structure
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
)

// RenameChatUseCase handles renaming a chat
type RenameChatUseCase struct {
	chatRepo CommandRepository
}

// NewRenameChatUseCase creates a new RenameChatUseCase
func NewRenameChatUseCase(chatRepo CommandRepository) *RenameChatUseCase {
	return &RenameChatUseCase{chatRepo: chatRepo}
}

// Execute performs renaming
func (uc *RenameChatUseCase) Execute(ctx context.Context, cmd RenameChatCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := uc.chatRepo.Load(ctx, cmd.ChatID)
	if err != nil {
		return Result{}, fmt.Errorf("failed to load chat: %w", err)
	}

	if renameErr := chatAggregate.Rename(cmd.NewTitle, cmd.RenamedBy); renameErr != nil {
		return Result{}, fmt.Errorf("failed to rename: %w", renameErr)
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

func (uc *RenameChatUseCase) validate(cmd RenameChatCommand) error {
	if err := appcore.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := appcore.ValidateRequired("newTitle", cmd.NewTitle); err != nil {
		return err
	}
	if err := appcore.ValidateMaxLength("newTitle", cmd.NewTitle, appcore.MaxTitleLength); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("renamedBy", cmd.RenamedBy); err != nil {
		return err
	}
	return nil
}
