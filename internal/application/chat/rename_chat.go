//nolint:dupl // Use case pattern requires similar structure
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

// RenameChatUseCase handles renaming a chat
type RenameChatUseCase struct {
	eventStore appcore.EventStore
}

// NewRenameChatUseCase creates a new RenameChatUseCase
func NewRenameChatUseCase(eventStore appcore.EventStore) *RenameChatUseCase {
	return &RenameChatUseCase{eventStore: eventStore}
}

// Execute performs renaming
func (uc *RenameChatUseCase) Execute(ctx context.Context, cmd RenameChatCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := loadAggregate(ctx, uc.eventStore, cmd.ChatID)
	if err != nil {
		return Result{}, err
	}

	if renameErr := chatAggregate.Rename(cmd.NewTitle, cmd.RenamedBy); renameErr != nil {
		return Result{}, fmt.Errorf("failed to rename: %w", renameErr)
	}

	return saveAggregate(ctx, uc.eventStore, chatAggregate, cmd.ChatID.String())
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
