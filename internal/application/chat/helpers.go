package chat

import (
	"context"
	"errors"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// loadAggregate загружает Chat агрегат из event store
func loadAggregate(ctx context.Context, eventStore appcore.EventStore, chatID uuid.UUID) (*chat.Chat, error) {
	events, err := eventStore.LoadEvents(ctx, chatID.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrChatNotFound, err)
	}

	if len(events) == 0 {
		return nil, ErrChatNotFound
	}

	chatAggregate := &chat.Chat{}
	for i, evt := range events {
		fmt.Printf("[DEBUG] loadAggregate: applying event %d, type=%T, eventType=%s\n", i, evt, evt.EventType())
		if applyErr := chatAggregate.Apply(evt); applyErr != nil {
			return nil, fmt.Errorf("failed to apply event: %w", applyErr)
		}
	}

	fmt.Printf("[DEBUG] loadAggregate: chat loaded, id=%s, version=%d, participants=%d\n",
		chatAggregate.ID().String(), chatAggregate.Version(), len(chatAggregate.Participants()))

	return chatAggregate, nil
}

// saveAggregate сохраняет новые события агрегата
func saveAggregate(
	ctx context.Context,
	eventStore appcore.EventStore,
	chatAggregate *chat.Chat,
	aggregateID string,
) (Result, error) {
	newEvents := chatAggregate.GetUncommittedEvents()
	if len(newEvents) == 0 {
		// Нет изменений - возвращаем текущее состояние
		return Result{
			Result: appcore.Result[*chat.Chat]{
				Value:   chatAggregate,
				Version: chatAggregate.Version(),
			},
			Events: nil,
		}, nil
	}

	currentVersion, _ := eventStore.GetVersion(ctx, aggregateID)

	// Конвертируем []event.DomainEvent в []event.DomainEvent (уже правильный тип)
	if err := eventStore.SaveEvents(ctx, aggregateID, newEvents, currentVersion); err != nil {
		if errors.Is(err, appcore.ErrConcurrencyConflict) {
			return Result{}, appcore.ErrConcurrentUpdate
		}
		return Result{}, fmt.Errorf("failed to save events: %w", err)
	}

	chatAggregate.MarkEventsAsCommitted()

	return Result{
		Result: appcore.Result[*chat.Chat]{
			Value:   chatAggregate,
			Version: chatAggregate.Version(),
		},
		Events: convertToInterfaceSlice(newEvents),
	}, nil
}

// convertToInterfaceSlice конвертирует []event.DomainEvent в []interface{}
func convertToInterfaceSlice(events []event.DomainEvent) []any {
	result := make([]any, len(events))
	for i, evt := range events {
		result[i] = evt
	}
	return result
}
