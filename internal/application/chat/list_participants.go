package chat

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

// ListParticipantsUseCase - use case для получения списка участников
type ListParticipantsUseCase struct {
	eventStore appcore.EventStore
}

// NewListParticipantsUseCase - конструктор
func NewListParticipantsUseCase(eventStore appcore.EventStore) *ListParticipantsUseCase {
	return &ListParticipantsUseCase{
		eventStore: eventStore,
	}
}

// Execute - выполнить запрос
func (uc *ListParticipantsUseCase) Execute(
	ctx context.Context,
	query ListParticipantsQuery,
) (*ListParticipantsResult, error) {
	// 1. Validate input
	if err := uc.validate(query); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 2. Load chat
	chatAggregate, err := loadAggregate(ctx, uc.eventStore, query.ChatID)
	if err != nil {
		return nil, fmt.Errorf("failed to load chat: %w", err)
	}

	// 3. Check access (must be participant or public chat)
	if !chatAggregate.IsPublic() && !chatAggregate.HasParticipant(query.RequestedBy) {
		return nil, errors.New("access denied: user is not a participant")
	}

	// 4. Get participants
	participants := chatAggregate.Participants()

	// 5. Sort by join date (ascending)
	sort.Slice(participants, func(i, j int) bool {
		return participants[i].JoinedAt().Before(participants[j].JoinedAt())
	})

	// 6. Map to DTOs
	participantDTOs := make([]Participant, len(participants))
	for i, p := range participants {
		participantDTOs[i] = Participant{
			UserID:   p.UserID(),
			Role:     p.Role(),
			JoinedAt: p.JoinedAt(),
		}
	}

	return &ListParticipantsResult{
		Participants: participantDTOs,
	}, nil
}

func (uc *ListParticipantsUseCase) validate(query ListParticipantsQuery) error {
	if err := appcore.ValidateUUID("chatID", query.ChatID); err != nil {
		return err
	}
	if err := appcore.ValidateUUID("requestedBy", query.RequestedBy); err != nil {
		return err
	}
	return nil
}
