package message

import (
	"context"
	"fmt"

	"github.com/lllypuk/teams-up/internal/application/shared"
	"github.com/lllypuk/teams-up/internal/domain/event"
	"github.com/lllypuk/teams-up/internal/domain/message"
)

// RemoveReactionUseCase обрабатывает удаление реакции с сообщения
type RemoveReactionUseCase struct {
	messageRepo message.Repository
	eventBus    event.Bus
}

// NewRemoveReactionUseCase создает новый RemoveReactionUseCase
func NewRemoveReactionUseCase(
	messageRepo message.Repository,
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
