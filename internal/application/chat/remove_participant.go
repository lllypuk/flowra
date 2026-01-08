package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

// RemoveParticipantUseCase handles removing a participant from a chat
type RemoveParticipantUseCase struct {
	eventStore appcore.EventStore
}

// NewRemoveParticipantUseCase creates a new RemoveParticipantUseCase
func NewRemoveParticipantUseCase(eventStore appcore.EventStore) *RemoveParticipantUseCase {
	return &RemoveParticipantUseCase{
		eventStore: eventStore,
	}
}

// Execute performs removing a participant
func (uc *RemoveParticipantUseCase) Execute(ctx context.Context, cmd RemoveParticipantCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := loadAggregate(ctx, uc.eventStore, cmd.ChatID)
	if err != nil {
		return Result{}, err
	}

	if removeErr := chatAggregate.RemoveParticipant(cmd.UserID); removeErr != nil {
		return Result{}, fmt.Errorf("failed to remove participant: %w", removeErr)
	}

	return saveAggregate(ctx, uc.eventStore, chatAggregate, cmd.ChatID.String())
}

func (uc *RemoveParticipantUseCase) validate(cmd RemoveParticipantCommand) error {
	if err := appcore.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("removedBy", cmd.RemovedBy); err != nil {
		return err
	}
	return nil
}
