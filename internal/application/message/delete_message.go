package message

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/message"
)

// DeleteMessageUseCase handles deletion messages (soft delete)
type DeleteMessageUseCase struct {
	messageRepo Repository
	eventBus    event.Bus
}

// NewDeleteMessageUseCase creates New DeleteMessageUseCase
func NewDeleteMessageUseCase(
	messageRepo Repository,
	eventBus event.Bus,
) *DeleteMessageUseCase {
	return &DeleteMessageUseCase{
		messageRepo: messageRepo,
		eventBus:    eventBus,
	}
}

// Execute performs deletion messages
func (uc *DeleteMessageUseCase) Execute(
	ctx context.Context,
	cmd DeleteMessageCommand,
) (Result, error) {
	// validation
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	// load message
	msg, err := uc.messageRepo.FindByID(ctx, cmd.MessageID)
	if err != nil {
		return Result{}, ErrMessageNotFound
	}

	// delete (authorization inside domain method)
	if deleteErr := msg.Delete(cmd.DeletedBy); deleteErr != nil {
		return Result{}, deleteErr
	}

	// save
	if saveErr := uc.messageRepo.Save(ctx, msg); saveErr != nil {
		return Result{}, fmt.Errorf("failed to save message: %w", saveErr)
	}

	// publish event
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
	if err := appcore.ValidateUUID("messageID", cmd.MessageID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("deletedBy", cmd.DeletedBy); err != nil {
		return err
	}
	return nil
}
