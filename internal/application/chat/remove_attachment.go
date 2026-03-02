//nolint:dupl // Use case pattern requires similar structure.
package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
)

// RemoveAttachmentUseCase handles removing attachments from typed chats.
type RemoveAttachmentUseCase struct {
	chatRepo CommandRepository
}

// NewRemoveAttachmentUseCase creates a new RemoveAttachmentUseCase.
func NewRemoveAttachmentUseCase(chatRepo CommandRepository) *RemoveAttachmentUseCase {
	return &RemoveAttachmentUseCase{chatRepo: chatRepo}
}

// Execute removes an attachment from the chat.
func (uc *RemoveAttachmentUseCase) Execute(ctx context.Context, cmd RemoveAttachmentCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := uc.chatRepo.Load(ctx, cmd.ChatID)
	if err != nil {
		return Result{}, fmt.Errorf("failed to load chat: %w", err)
	}

	if removeErr := chatAggregate.RemoveAttachment(cmd.FileID, cmd.RemovedBy); removeErr != nil {
		return Result{}, fmt.Errorf("failed to remove attachment: %w", removeErr)
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

func (uc *RemoveAttachmentUseCase) validate(cmd RemoveAttachmentCommand) error {
	if err := appcore.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("fileID", cmd.FileID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("removedBy", cmd.RemovedBy); err != nil {
		return err
	}
	return nil
}
