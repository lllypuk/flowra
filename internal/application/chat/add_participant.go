package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
)

// AddParticipantUseCase handles adding a participant to a chat
type AddParticipantUseCase struct {
	eventStore appcore.EventStore
}

// NewAddParticipantUseCase creates a new AddParticipantUseCase
func NewAddParticipantUseCase(eventStore appcore.EventStore) *AddParticipantUseCase {
	return &AddParticipantUseCase{
		eventStore: eventStore,
	}
}

// Execute performs adding a participant
func (uc *AddParticipantUseCase) Execute(ctx context.Context, cmd AddParticipantCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := loadAggregate(ctx, uc.eventStore, cmd.ChatID)
	if err != nil {
		return Result{}, err
	}

	// Domain layer manages events itself
	if addErr := chatAggregate.AddParticipant(cmd.UserID, cmd.Role); addErr != nil {
		return Result{}, fmt.Errorf("failed to add participant: %w", addErr)
	}

	return saveAggregate(ctx, uc.eventStore, chatAggregate, cmd.ChatID.String())
}

func (uc *AddParticipantUseCase) validate(cmd AddParticipantCommand) error {
	if err := appcore.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("addedBy", cmd.AddedBy); err != nil {
		return err
	}
	if err := appcore.ValidateEnum("role", string(cmd.Role), []string{
		string(chat.RoleAdmin),
		string(chat.RoleMember),
	}); err != nil {
		return err
	}
	return nil
}
