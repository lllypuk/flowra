//nolint:dupl // Use case pattern requires similar structure
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
)

// ConvertToBugUseCase handles converting a chat to Bug
type ConvertToBugUseCase struct {
	chatRepo CommandRepository
}

// NewConvertToBugUseCase creates a new ConvertToBugUseCase
func NewConvertToBugUseCase(chatRepo CommandRepository) *ConvertToBugUseCase {
	return &ConvertToBugUseCase{
		chatRepo: chatRepo,
	}
}

// Execute performs converting to Bug
func (uc *ConvertToBugUseCase) Execute(ctx context.Context, cmd ConvertToBugCommand) (Result, error) {
	// Validation
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := uc.chatRepo.Load(ctx, cmd.ChatID)
	if err != nil {
		return Result{}, fmt.Errorf("failed to load chat: %w", err)
	}

	if convertErr := chatAggregate.ConvertToBug(cmd.Title, cmd.ConvertedBy); convertErr != nil {
		return Result{}, fmt.Errorf("failed to convert to bug: %w", convertErr)
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

func (uc *ConvertToBugUseCase) validate(cmd ConvertToBugCommand) error {
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
