//nolint:dupl // Use case pattern requires similar structure
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
)

// ChangeStatusUseCase handles changing chat status
type ChangeStatusUseCase struct {
	chatRepo CommandRepository
}

// NewChangeStatusUseCase creates a new ChangeStatusUseCase
func NewChangeStatusUseCase(chatRepo CommandRepository) *ChangeStatusUseCase {
	return &ChangeStatusUseCase{
		chatRepo: chatRepo,
	}
}

// Execute performs changing status
func (uc *ChangeStatusUseCase) Execute(ctx context.Context, cmd ChangeStatusCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := uc.chatRepo.Load(ctx, cmd.ChatID)
	if err != nil {
		return Result{}, fmt.Errorf("failed to load chat: %w", err)
	}

	if statusErr := chatAggregate.ChangeStatus(cmd.Status, cmd.ChangedBy); statusErr != nil {
		return Result{}, fmt.Errorf("failed to change status: %w", statusErr)
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

func (uc *ChangeStatusUseCase) validate(cmd ChangeStatusCommand) error {
	if err := appcore.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := appcore.ValidateRequired("status", cmd.Status); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("changedBy", cmd.ChangedBy); err != nil {
		return err
	}
	return nil
}
