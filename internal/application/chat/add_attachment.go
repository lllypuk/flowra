package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
)

// AddAttachmentUseCase handles adding attachments to typed chats.
type AddAttachmentUseCase struct {
	chatRepo CommandRepository
}

// NewAddAttachmentUseCase creates a new AddAttachmentUseCase.
func NewAddAttachmentUseCase(chatRepo CommandRepository) *AddAttachmentUseCase {
	return &AddAttachmentUseCase{chatRepo: chatRepo}
}

// Execute adds an attachment to the chat.
func (uc *AddAttachmentUseCase) Execute(ctx context.Context, cmd AddAttachmentCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := uc.chatRepo.Load(ctx, cmd.ChatID)
	if err != nil {
		return Result{}, fmt.Errorf("failed to load chat: %w", err)
	}

	if addErr := chatAggregate.AddAttachment(
		cmd.FileID,
		cmd.FileName,
		cmd.FileSize,
		cmd.MimeType,
		cmd.AddedBy,
	); addErr != nil {
		return Result{}, fmt.Errorf("failed to add attachment: %w", addErr)
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

func (uc *AddAttachmentUseCase) validate(cmd AddAttachmentCommand) error {
	if err := appcore.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("fileID", cmd.FileID); err != nil {
		return err
	}
	if err := appcore.ValidateRequired("fileName", cmd.FileName); err != nil {
		return err
	}
	if cmd.FileSize <= 0 {
		return appcore.NewValidationError("fileSize", "must be positive")
	}
	if err := appcore.ValidateRequired("mimeType", cmd.MimeType); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("addedBy", cmd.AddedBy); err != nil {
		return err
	}
	return nil
}
