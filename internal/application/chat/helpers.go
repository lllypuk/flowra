package chat

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// loadAggregate loads Chat aggregate from event store
func loadAggregate(ctx context.Context, eventStore appcore.EventStore, chatID uuid.UUID) (*chat.Chat, error) {
	events, err := eventStore.LoadEvents(ctx, chatID.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrChatNotFound, err)
	}

	if len(events) == 0 {
		return nil, ErrChatNotFound
	}

	chatAggregate := &chat.Chat{}
	for _, evt := range events {
		if applyErr := chatAggregate.Apply(evt); applyErr != nil {
			return nil, fmt.Errorf("failed to apply event: %w", applyErr)
		}
	}

	return chatAggregate, nil
}

// convertToInterfaceSlice converts []event.DomainEvent to []interface{}
func convertToInterfaceSlice(events []event.DomainEvent) []any {
	result := make([]any, len(events))
	for i, evt := range events {
		result[i] = evt
	}
	return result
}
