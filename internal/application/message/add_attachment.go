package message

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/message"
)

// AddAttachmentUseCase обрабатывает добавление вложения к сообщению
type AddAttachmentUseCase struct {
	messageRepo message.Repository
	eventBus    event.Bus
}

// NewAddAttachmentUseCase создает новый AddAttachmentUseCase
func NewAddAttachmentUseCase(
	messageRepo message.Repository,
	eventBus event.Bus,
) *AddAttachmentUseCase {
	return &AddAttachmentUseCase{
		messageRepo: messageRepo,
		eventBus:    eventBus,
	}
}

// Execute выполняет добавление вложения
func (uc *AddAttachmentUseCase) Execute(
	ctx context.Context,
	cmd AddAttachmentCommand,
) (Result, error) {
	// Валидация
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// Загрузка сообщения
	msg, err := uc.messageRepo.FindByID(ctx, cmd.MessageID)
	if err != nil {
		return Result{}, ErrMessageNotFound
	}

	// Проверка, что сообщение не удалено
	if msg.IsDeleted() {
		return Result{}, ErrMessageDeleted
	}

	// Авторизация: только автор может добавлять вложения
	if !msg.CanBeEditedBy(cmd.UserID) {
		return Result{}, ErrNotAuthor
	}

	// Добавление вложения
	if addErr := msg.AddAttachment(cmd.FileID, cmd.FileName, cmd.FileSize, cmd.MimeType); addErr != nil {
		return Result{}, addErr
	}

	// Сохранение
	if saveErr := uc.messageRepo.Save(ctx, msg); saveErr != nil {
		return Result{}, fmt.Errorf("failed to save message: %w", saveErr)
	}

	// Публикация события
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
	if err := shared.ValidateUUID("messageID", cmd.MessageID); err != nil {
		return err
	}
	if err := shared.ValidateUUID("fileID", cmd.FileID); err != nil {
		return err
	}
	if err := shared.ValidateRequired("fileName", cmd.FileName); err != nil {
		return ErrInvalidFileName
	}
	if err := shared.ValidateRequired("mimeType", cmd.MimeType); err != nil {
		return ErrInvalidMimeType
	}
	if cmd.FileSize <= 0 {
		return ErrInvalidFileSize
	}
	if cmd.FileSize > MaxFileSize {
		return ErrInvalidFileSize
	}
	if err := shared.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	return nil
}
