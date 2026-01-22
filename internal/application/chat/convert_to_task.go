//nolint:dupl // Use case pattern requires similar structure
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
)

// ConvertToTaskUseCase handles converting a chat to Task
type ConvertToTaskUseCase struct {
	chatRepo CommandRepository
}

// NewConvertToTaskUseCase creates a new ConvertToTaskUseCase
func NewConvertToTaskUseCase(chatRepo CommandRepository) *ConvertToTaskUseCase {
	return &ConvertToTaskUseCase{
		chatRepo: chatRepo,
	}
}

// Execute performs converting to Task
func (uc *ConvertToTaskUseCase) Execute(ctx context.Context, cmd ConvertToTaskCommand) (Result, error) {
	// Validation
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := uc.chatRepo.Load(ctx, cmd.ChatID)
	if err != nil {
		return Result{}, fmt.Errorf("failed to load chat: %w", err)
	}

	if convertErr := chatAggregate.ConvertToTask(cmd.Title, cmd.ConvertedBy); convertErr != nil {
		return Result{}, fmt.Errorf("failed to convert to task: %w", convertErr)
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

func (uc *ConvertToTaskUseCase) validate(cmd ConvertToTaskCommand) error {
	if err := appcore.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := appcore.ValidateRequired("title", cmd.Title); err != nil {
		return err
	}
	if err := appcore.ValidateMaxLength("title", cmd.Title, appcore.MaxTitleLength); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("convertedBy", cmd.ConvertedBy); err != nil {
		return err
	}
	return nil
}
