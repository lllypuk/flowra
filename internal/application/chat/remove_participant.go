package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/teams-up/internal/application/shared"
)

// RemoveParticipantUseCase обрабатывает удаление участника из чата
type RemoveParticipantUseCase struct {
	eventStore shared.EventStore
}

// NewRemoveParticipantUseCase создает новый RemoveParticipantUseCase
func NewRemoveParticipantUseCase(eventStore shared.EventStore) *RemoveParticipantUseCase {
	return &RemoveParticipantUseCase{
		eventStore: eventStore,
	}
}

// Execute выполняет удаление участника
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
	if err := shared.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := shared.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	if err := shared.ValidateUUID("removedBy", cmd.RemovedBy); err != nil {
		return err
	}
	return nil
}
