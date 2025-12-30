package message

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/message"
)

// DeleteMessageUseCase обрабатывает удаление сообщения (soft delete)
type DeleteMessageUseCase struct {
	messageRepo Repository
	eventBus    event.Bus
}

// NewDeleteMessageUseCase создает новый DeleteMessageUseCase
func NewDeleteMessageUseCase(
	messageRepo Repository,
	eventBus event.Bus,
) *DeleteMessageUseCase {
	return &DeleteMessageUseCase{
		messageRepo: messageRepo,
		eventBus:    eventBus,
	}
}

// Execute выполняет удаление сообщения
func (uc *DeleteMessageUseCase) Execute(
	ctx context.Context,
	cmd DeleteMessageCommand,
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

	// Удаление (авторизация внутри domain метода)
	if deleteErr := msg.Delete(cmd.DeletedBy); deleteErr != nil {
		return Result{}, deleteErr
	}

	// Сохранение
	if saveErr := uc.messageRepo.Save(ctx, msg); saveErr != nil {
		return Result{}, fmt.Errorf("failed to save message: %w", saveErr)
	}

	// Публикация события
	evt := message.NewDeleted(msg.ID(), cmd.DeletedBy, 1, event.Metadata{
		UserID:    cmd.DeletedBy.String(),
		Timestamp: *msg.DeletedAt(),
	})
	_ = uc.eventBus.Publish(ctx, evt)

	return Result{
		Value: msg,
	}, nil
}

func (uc *DeleteMessageUseCase) validate(cmd DeleteMessageCommand) error {
	if err := shared.ValidateUUID("messageID", cmd.MessageID); err != nil {
		return err
	}
	if err := shared.ValidateUUID("deletedBy", cmd.DeletedBy); err != nil {
		return err
	}
	return nil
}
