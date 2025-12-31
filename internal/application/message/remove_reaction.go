package message

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/message"
)

// RemoveReactionUseCase обрабатывает удаление реакции с сообщения
type RemoveReactionUseCase struct {
	messageRepo Repository
	eventBus    event.Bus
}

// NewRemoveReactionUseCase создает новый RemoveReactionUseCase
func NewRemoveReactionUseCase(
	messageRepo Repository,
	eventBus event.Bus,
) *RemoveReactionUseCase {
	return &RemoveReactionUseCase{
		messageRepo: messageRepo,
		eventBus:    eventBus,
	}
}

// Execute выполняет удаление реакции
func (uc *RemoveReactionUseCase) Execute(
	ctx context.Context,
	cmd RemoveReactionCommand,
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

	// Удаление реакции
	if removeErr := msg.RemoveReaction(cmd.UserID, cmd.Emoji); removeErr != nil {
		return Result{}, removeErr
	}

	// Сохранение
	if saveErr := uc.messageRepo.Save(ctx, msg); saveErr != nil {
		return Result{}, fmt.Errorf("failed to save message: %w", saveErr)
	}

	// Публикация события
	evt := message.NewReactionRemoved(msg.ID(), cmd.UserID, cmd.Emoji, 1, event.Metadata{
		UserID: cmd.UserID.String(),
	})
	_ = uc.eventBus.Publish(ctx, evt)

	return Result{
		Value: msg,
	}, nil
}

func (uc *RemoveReactionUseCase) validate(cmd RemoveReactionCommand) error {
	if err := appcore.ValidateUUID("messageID", cmd.MessageID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	if err := appcore.ValidateRequired("emoji", cmd.Emoji); err != nil {
		return ErrInvalidEmoji
	}
	return nil
}
