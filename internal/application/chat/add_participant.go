package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/chat"
)

// AddParticipantUseCase обрабатывает добавление участника в чат
type AddParticipantUseCase struct {
	eventStore shared.EventStore
}

// NewAddParticipantUseCase создает новый AddParticipantUseCase
func NewAddParticipantUseCase(eventStore shared.EventStore) *AddParticipantUseCase {
	return &AddParticipantUseCase{
		eventStore: eventStore,
	}
}

// Execute выполняет добавление участника
func (uc *AddParticipantUseCase) Execute(ctx context.Context, cmd AddParticipantCommand) (Result, error) {
	if err := uc.validate(cmd); err != nil {
		return Result{}, fmt.Errorf("validation failed: %w", err)
	}

	chatAggregate, err := loadAggregate(ctx, uc.eventStore, cmd.ChatID)
	if err != nil {
		return Result{}, err
	}

	if addErr := chatAggregate.AddParticipant(cmd.UserID, cmd.Role); addErr != nil {
		return Result{}, fmt.Errorf("failed to add participant: %w", addErr)
	}

	return saveAggregate(ctx, uc.eventStore, chatAggregate, cmd.ChatID.String())
}

func (uc *AddParticipantUseCase) validate(cmd AddParticipantCommand) error {
	if err := shared.ValidateUUID("chatID", cmd.ChatID); err != nil {
		return err
	}
	if err := shared.ValidateUUID("userID", cmd.UserID); err != nil {
		return err
	}
	if err := shared.ValidateUUID("addedBy", cmd.AddedBy); err != nil {
		return err
	}
	if err := shared.ValidateEnum("role", string(cmd.Role), []string{
		string(chat.RoleAdmin),
		string(chat.RoleMember),
	}); err != nil {
		return err
	}
	return nil
}
