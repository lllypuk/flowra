package chat

import (
	"context"
	"errors"
	"fmt"

	"github.com/lllypuk/teams-up/internal/application/shared"
	"github.com/lllypuk/teams-up/internal/domain/chat"
	"github.com/lllypuk/teams-up/internal/domain/event"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
)

// loadAggregate загружает Chat агрегат из event store
func loadAggregate(ctx context.Context, eventStore shared.EventStore, chatID uuid.UUID) (*chat.Chat, error) {
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

// saveAggregate сохраняет новые события агрегата
func saveAggregate(
	ctx context.Context,
	eventStore shared.EventStore,
	chatAggregate *chat.Chat,
	aggregateID string,
) (Result, error) {
	newEvents := chatAggregate.GetUncommittedEvents()
	if len(newEvents) == 0 {
		// Нет изменений - возвращаем текущее состояние
		return Result{
			Result: shared.Result[*chat.Chat]{
				Value:   chatAggregate,
				Version: chatAggregate.Version(),
			},
			Events: nil,
		}, nil
	}

	currentVersion, _ := eventStore.GetVersion(ctx, aggregateID)

	// Конвертируем []event.DomainEvent в []event.DomainEvent (уже правильный тип)
	if err := eventStore.SaveEvents(ctx, aggregateID, newEvents, currentVersion); err != nil {
		if errors.Is(err, shared.ErrConcurrencyConflict) {
			return Result{}, shared.ErrConcurrentUpdate
		}
		return Result{}, fmt.Errorf("failed to save events: %w", err)
	}

	chatAggregate.MarkEventsAsCommitted()

	return Result{
		Result: shared.Result[*chat.Chat]{
			Value:   chatAggregate,
			Version: chatAggregate.Version(),
		},
		Events: convertToInterfaceSlice(newEvents),
	}, nil
}

// convertToInterfaceSlice конвертирует []event.DomainEvent в []interface{}
func convertToInterfaceSlice(events []event.DomainEvent) []interface{} {
	result := make([]interface{}, len(events))
	for i, evt := range events {
		result[i] = evt
	}
	return result
}
