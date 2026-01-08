package mocks

import (
	"context"
	"sync"

	"github.com/lllypuk/flowra/internal/domain/event"
)

// MockEventBus implements event.Bus for testing
type MockEventBus struct {
	mu        sync.RWMutex
	published []event.DomainEvent
	handlers  map[string][]EventHandler
}

// EventHandler is a type for event handlers
type EventHandler func(ctx context.Context, evt event.DomainEvent) error

// NewMockEventBus creates a new mock event bus
func NewMockEventBus() *MockEventBus {
	return &MockEventBus{
		published: []event.DomainEvent{},
		handlers:  make(map[string][]EventHandler),
	}
}

// Publish publishes an event
func (b *MockEventBus) Publish(ctx context.Context, evt event.DomainEvent) error {
	b.mu.Lock()
	b.published = append(b.published, evt)
	handlers := b.handlers[evt.EventType()]
	b.mu.Unlock()

	// Synchronous call of all handlers (for tests)
	for _, handler := range handlers {
		if err := handler(ctx, evt); err != nil {
			return err
		}
	}

	return nil
}

// Subscribe subscribes a handler to events of a specific type
func (b *MockEventBus) Subscribe(eventType string, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// PublishedCount returns the number of published events
func (b *MockEventBus) PublishedCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.published)
}

// PublishedEvents returns all published events
func (b *MockEventBus) PublishedEvents() []event.DomainEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return append([]event.DomainEvent{}, b.published...)
}

// GetPublishedEventsByType returns events of a specific type
func (b *MockEventBus) GetPublishedEventsByType(eventType string) []event.DomainEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var events []event.DomainEvent
	for _, evt := range b.published {
		if evt.EventType() == eventType {
			events = append(events, evt)
		}
	}
	return events
}

// GetPublishedEventsByAggregate returns events for a specific aggregate
func (b *MockEventBus) GetPublishedEventsByAggregate(aggregateID string) []event.DomainEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var events []event.DomainEvent
	for _, evt := range b.published {
		if evt.AggregateID() == aggregateID {
			events = append(events, evt)
		}
	}
	return events
}

// Reset clears the bus
func (b *MockEventBus) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.published = []event.DomainEvent{}
	b.handlers = make(map[string][]EventHandler)
}

// HandlerCount returns the number of registered handlers for an event type
func (b *MockEventBus) HandlerCount(eventType string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.handlers[eventType])
}
