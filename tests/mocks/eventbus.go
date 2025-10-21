package mocks

import (
	"context"
	"sync"

	"github.com/lllypuk/teams-up/internal/domain/event"
)

// MockEventBus реализует event.Bus для тестирования
type MockEventBus struct {
	mu        sync.RWMutex
	published []event.DomainEvent
	handlers  map[string][]EventHandler
}

// EventHandler тип для обработчиков событий
type EventHandler func(ctx context.Context, evt event.DomainEvent) error

// NewMockEventBus создает новый mock event bus
func NewMockEventBus() *MockEventBus {
	return &MockEventBus{
		published: []event.DomainEvent{},
		handlers:  make(map[string][]EventHandler),
	}
}

// Publish публикует событие
func (b *MockEventBus) Publish(ctx context.Context, evt event.DomainEvent) error {
	b.mu.Lock()
	b.published = append(b.published, evt)
	handlers := b.handlers[evt.EventType()]
	b.mu.Unlock()

	// Синхронный вызов всех handlers (для тестов)
	for _, handler := range handlers {
		if err := handler(ctx, evt); err != nil {
			return err
		}
	}

	return nil
}

// Subscribe подписывает handler на событие определенного типа
func (b *MockEventBus) Subscribe(eventType string, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// PublishedCount возвращает количество опубликованных событий
func (b *MockEventBus) PublishedCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.published)
}

// PublishedEvents возвращает все опубликованные события
func (b *MockEventBus) PublishedEvents() []event.DomainEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return append([]event.DomainEvent{}, b.published...)
}

// GetPublishedEventsByType возвращает события определенного типа
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

// GetPublishedEventsByAggregate возвращает события конкретного агрегата
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

// Reset очищает bus
func (b *MockEventBus) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.published = []event.DomainEvent{}
	b.handlers = make(map[string][]EventHandler)
}

// HandlerCount возвращает количество зарегистрированных handlers для типа события
func (b *MockEventBus) HandlerCount(eventType string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.handlers[eventType])
}
