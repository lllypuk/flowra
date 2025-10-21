package message

import (
	"context"
	"fmt"

	"github.com/flowra/flowra/internal/application/shared"
	"github.com/flowra/flowra/internal/domain/event"
	"github.com/flowra/flowra/internal/domain/message"
)

// AddReactionUseCase обрабатывает добавление реакции к сообщению
type AddReactionUseCase struct {
	messageRepo message.Repository
	eventBus    event.Bus
}

// NewAddReactionUseCase создает новый AddReactionUseCase
func NewAddReactionUseCase(
	messageRepo message.Repository,
	eventBus event.Bus,
) *AddReactionUseCase {
	return &AddReactionUseCase{
		messageRepo: messageRepo,
		eventBus:    eventBus,
	}
}

// Execute выполняет добавление реакции
func (uc *AddReactionUseCase) Execute(
	ctx context.Context,
	cmd AddReactionCommand,
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

	// Добавление реакции
	if addErr := msg.AddReaction(cmd.UserID, cmd.Emoji); addErr != nil {
		return Result{}, addErr
	}

	// Сохранение
	if saveErr := uc.messageRepo.Save(ctx, msg); saveErr != nil {
		return Result{}, fmt.Errorf("failed to save message: %w", saveErr)
	}

	// Публикация события
	evt := message.NewReactionAdded(msg.ID(), cmd.UserID, cmd.Emoji, 1, event.Metadata{
		UserID: cmd.UserID.String(),
	})
	_ = uc.eventBus.Publish(ctx, evt)

	return Result{
		Value: msg,
	}, nil
}

func (uc *AddReactionUseCase) validate(cmd AddReactionCommand) error {
	if err := shared.ValidateUUID("messageID", cmd.MessageID); err != nil {
		return err
	}
	if err := shared.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	if err := shared.ValidateRequired("emoji", cmd.Emoji); err != nil {
		return ErrInvalidEmoji
	}
	return nil
}
