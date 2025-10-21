package chat

import (
	"context"
	"fmt"
	"time"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
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

	// Проверка перед добавлением
	if chatAggregate.HasParticipant(cmd.UserID) {
		return Result{}, fmt.Errorf("failed to add participant: %w", fmt.Errorf("user already a participant"))
	}

	// Создание события ParticipantAdded для сохранения в event sourcing
	participantAddedEvent := chat.NewParticipantAdded(
		cmd.ChatID,
		cmd.UserID,
		cmd.Role,
		time.Now(),
		event.Metadata{
			CorrelationID: domainUUID.NewUUID().String(),
			CausationID:   domainUUID.NewUUID().String(),
			UserID:        cmd.AddedBy.String(),
		},
	)
	if applyErr := chatAggregate.ApplyAndTrack(participantAddedEvent); applyErr != nil {
		return Result{}, fmt.Errorf("failed to apply ParticipantAdded event: %w", applyErr)
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
