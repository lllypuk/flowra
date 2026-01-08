package message

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/message"
)

// AddReactionUseCase handles adding реакции to сообщению
type AddReactionUseCase struct {
	messageRepo Repository
	eventBus    event.Bus
}

// NewAddReactionUseCase creates New AddReactionUseCase
func NewAddReactionUseCase(
	messageRepo Repository,
	eventBus event.Bus,
) *AddReactionUseCase {
	return &AddReactionUseCase{
		messageRepo: messageRepo,
		eventBus:    eventBus,
	}
}

// Execute performs adding реакции
func (uc *AddReactionUseCase) Execute(
	ctx context.Context,
	cmd AddReactionCommand,
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

	// Adding реакции
	if addErr := msg.AddReaction(cmd.UserID, cmd.Emoji); addErr != nil {
		return Result{}, addErr
	}

	// storage
	if saveErr := uc.messageRepo.Save(ctx, msg); saveErr != nil {
		return Result{}, fmt.Errorf("failed to save message: %w", saveErr)
	}

	// Publishing event
	evt := message.NewReactionAdded(msg.ID(), cmd.UserID, cmd.Emoji, 1, event.Metadata{
		UserID: cmd.UserID.String(),
	})
	_ = uc.eventBus.Publish(ctx, evt)

	return Result{
		Value: msg,
	}, nil
}

func (uc *AddReactionUseCase) validate(cmd AddReactionCommand) error {
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
