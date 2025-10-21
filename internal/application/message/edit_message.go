package message

import (
	"context"
	"fmt"

	"github.com/flowra/flowra/internal/application/shared"
	"github.com/flowra/flowra/internal/domain/event"
	"github.com/flowra/flowra/internal/domain/message"
)

// EditMessageUseCase обрабатывает редактирование сообщения
type EditMessageUseCase struct {
	messageRepo message.Repository
	eventBus    event.Bus
}

// NewEditMessageUseCase создает новый EditMessageUseCase
func NewEditMessageUseCase(
	messageRepo message.Repository,
	eventBus event.Bus,
) *EditMessageUseCase {
	return &EditMessageUseCase{
		messageRepo: messageRepo,
		eventBus:    eventBus,
	}
}

// Execute выполняет редактирование сообщения
func (uc *EditMessageUseCase) Execute(
	ctx context.Context,
	cmd EditMessageCommand,
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

	// Редактирование (авторизация внутри domain метода)
	if editErr := msg.EditContent(cmd.Content, cmd.EditorID); editErr != nil {
		return Result{}, editErr
	}

	// Сохранение
	if saveErr := uc.messageRepo.Save(ctx, msg); saveErr != nil {
		return Result{}, fmt.Errorf("failed to save message: %w", saveErr)
	}

	// Публикация события
	evt := message.NewEdited(msg.ID(), cmd.Content, 1, event.Metadata{
		UserID:    cmd.EditorID.String(),
		Timestamp: *msg.EditedAt(),
	})
	_ = uc.eventBus.Publish(ctx, evt)

	return Result{
		Value: msg,
	}, nil
}

func (uc *EditMessageUseCase) validate(cmd EditMessageCommand) error {
	if err := shared.ValidateUUID("messageID", cmd.MessageID); err != nil {
		return err
	}
	if err := shared.ValidateRequired("content", cmd.Content); err != nil {
		return ErrEmptyContent
	}
	if len(cmd.Content) > MaxContentLength {
		return ErrContentTooLong
	}
	if err := shared.ValidateUUID("editorID", cmd.EditorID); err != nil {
		return err
	}
	return nil
}
