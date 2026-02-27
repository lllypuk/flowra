package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	domchat "github.com/lllypuk/flowra/internal/domain/chat"
)

// RemoveParticipantUseCase handles removing a participant from a chat
type RemoveParticipantUseCase struct {
	chatRepo CommandRepository
}

// NewRemoveParticipantUseCase creates a new RemoveParticipantUseCase
func NewRemoveParticipantUseCase(chatRepo CommandRepository) *RemoveParticipantUseCase {
	return &RemoveParticipantUseCase{
		chatRepo: chatRepo,
	}
}

// Execute performs removing a participant
func (uc *RemoveParticipantUseCase) Execute(ctx context.Context, cmd RemoveParticipantCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := uc.chatRepo.Load(ctx, cmd.ChatID)
	if err != nil {
		return Result{}, err
	}

	if removeErr := chatAggregate.RemoveParticipant(cmd.UserID); removeErr != nil {
		return Result{}, fmt.Errorf("failed to remove participant: %w", removeErr)
	}

	// Capture events before save (Save marks them as committed)
	newEvents := chatAggregate.GetUncommittedEvents()

	// Save via repository (updates both event store and read model)
	if saveErr := uc.chatRepo.Save(ctx, chatAggregate); saveErr != nil {
		return Result{}, fmt.Errorf("failed to save chat: %w", saveErr)
	}

	return Result{
		Result: appcore.Result[*domchat.Chat]{
			Value:   chatAggregate,
			Version: chatAggregate.Version(),
		},
		Events: convertToInterfaceSlice(newEvents),
	}, nil
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
