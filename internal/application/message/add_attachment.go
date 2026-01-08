package message

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/message"
)

// AddAttachmentUseCase handles adding вложения to сообщению
type AddAttachmentUseCase struct {
	messageRepo Repository
	eventBus    event.Bus
}

// NewAddAttachmentUseCase creates New AddAttachmentUseCase
func NewAddAttachmentUseCase(
	messageRepo Repository,
	eventBus event.Bus,
) *AddAttachmentUseCase {
	return &AddAttachmentUseCase{
		messageRepo: messageRepo,
		eventBus:    eventBus,
	}
}

// Execute performs adding вложения
func (uc *AddAttachmentUseCase) Execute(
	ctx context.Context,
	cmd AddAttachmentCommand,
) (Result, error) {
	// validation
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// Loading message
	msg, err := uc.messageRepo.FindByID(ctx, cmd.MessageID)
	if err != nil {
		return Result{}, ErrMessageNotFound
	}

	// check, that message not удалено
	if msg.IsDeleted() {
		return Result{}, ErrMessageDeleted
	}

	// authorization: only автор может добавлять вложения
	if !msg.CanBeEditedBy(cmd.UserID) {
		return Result{}, ErrNotAuthor
	}

	// Adding вложения
	if addErr := msg.AddAttachment(cmd.FileID, cmd.FileName, cmd.FileSize, cmd.MimeType); addErr != nil {
		return Result{}, addErr
	}

	// storage
	if saveErr := uc.messageRepo.Save(ctx, msg); saveErr != nil {
		return Result{}, fmt.Errorf("failed to save message: %w", saveErr)
	}

	// Publishing event
	evt := message.NewAttachmentAdded(
		msg.ID(),
		cmd.FileID,
		cmd.FileName,
		cmd.FileSize,
		cmd.MimeType,
		1,
		event.Metadata{
			UserID: cmd.UserID.String(),
		},
	)
	_ = uc.eventBus.Publish(ctx, evt)

	return Result{
		Value: msg,
	}, nil
}

func (uc *AddAttachmentUseCase) validate(cmd AddAttachmentCommand) error {
	if err := appcore.ValidateUUID("messageID", cmd.MessageID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("fileID", cmd.FileID); err != nil {
		return err
	}
	if err := appcore.ValidateRequired("fileName", cmd.FileName); err != nil {
		return ErrInvalidFileName
	}
	if err := appcore.ValidateRequired("mimeType", cmd.MimeType); err != nil {
		return ErrInvalidMimeType
	}
	if cmd.FileSize <= 0 {
		return ErrInvalidFileSize
	}
	if cmd.FileSize > MaxFileSize {
		return ErrInvalidFileSize
	}
	if err := appcore.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	return nil
}
