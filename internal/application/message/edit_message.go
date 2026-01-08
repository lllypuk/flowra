package message

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/message"
)

// EditMessageUseCase handles редактирование messages
type EditMessageUseCase struct {
	messageRepo Repository
	eventBus    event.Bus
}

// NewEditMessageUseCase creates New EditMessageUseCase
func NewEditMessageUseCase(
	messageRepo Repository,
	eventBus event.Bus,
) *EditMessageUseCase {
	return &EditMessageUseCase{
		messageRepo: messageRepo,
		eventBus:    eventBus,
	}
}

// Execute performs редактирование messages
func (uc *EditMessageUseCase) Execute(
	ctx context.Context,
	cmd EditMessageCommand,
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

	// Редактирование (authorization inside domain метода)
	if editErr := msg.EditContent(cmd.Content, cmd.EditorID); editErr != nil {
		return Result{}, editErr
	}

	// storage
	if saveErr := uc.messageRepo.Save(ctx, msg); saveErr != nil {
		return Result{}, fmt.Errorf("failed to save message: %w", saveErr)
	}

	// Publishing event
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
	if err := appcore.ValidateUUID("messageID", cmd.MessageID); err != nil {
		return err
	}
	if err := appcore.ValidateRequired("content", cmd.Content); err != nil {
		return ErrEmptyContent
	}
	if len(cmd.Content) > MaxContentLength {
		return ErrContentTooLong
	}
	if err := appcore.ValidateUUID("editorID", cmd.EditorID); err != nil {
		return err
	}
	return nil
}
