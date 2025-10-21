//nolint:dupl // Use case pattern requires similar structure
package chat

import (
	"context"
	"fmt"

	"github.com/flowra/flowra/internal/application/shared"
)

// RenameChatUseCase обрабатывает переименование чата
type RenameChatUseCase struct {
	eventStore shared.EventStore
}

// NewRenameChatUseCase создает новый RenameChatUseCase
func NewRenameChatUseCase(eventStore shared.EventStore) *RenameChatUseCase {
	return &RenameChatUseCase{eventStore: eventStore}
}

// Execute выполняет переименование
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
	if err := shared.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := shared.ValidateRequired("newTitle", cmd.NewTitle); err != nil {
		return err
	}
	if err := shared.ValidateMaxLength("newTitle", cmd.NewTitle, shared.MaxTitleLength); err != nil {
		return err
	}
	if err := shared.ValidateUUID("renamedBy", cmd.RenamedBy); err != nil {
		return err
	}
	return nil
}
