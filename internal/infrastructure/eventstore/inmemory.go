package eventstore

import (
	"context"
	"sync"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/event"
)

// InMemoryEventStore реализует EventStore в памяти для тестирования
type InMemoryEventStore struct {
	mu     sync.RWMutex
	events map[string][]event.DomainEvent
}

// NewInMemoryEventStore создает новый in-memory event store
func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{
		events: make(map[string][]event.DomainEvent),
	}
}

// SaveEvents сохраняет события для агрегата
func (s *InMemoryEventStore) SaveEvents(
	_ context.Context,
	aggregateID string,
	events []event.DomainEvent,
	expectedVersion int,
) error {
	if len(events) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверка optimistic locking
	currentVersion := len(s.events[aggregateID])
	if currentVersion != expectedVersion {
		return appcore.ErrConcurrencyConflict
	}

	// Сохраняем события
	s.events[aggregateID] = append(s.events[aggregateID], events...)

	return nil
}

// LoadEvents загружает все события для агрегата
func (s *InMemoryEventStore) LoadEvents(
	_ context.Context,
	aggregateID string,
) ([]event.DomainEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events, exists := s.events[aggregateID]
	if !exists {
		return nil, appcore.ErrAggregateNotFound
	}

	// Возвращаем копию чтобы избежать race conditions
	result := make([]event.DomainEvent, len(events))
	copy(result, events)

	return result, nil
}

// GetVersion возвращает текущую версию агрегата
func (s *InMemoryEventStore) GetVersion(
	_ context.Context,
	aggregateID string,
) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events, exists := s.events[aggregateID]
	if !exists {
		return 0, nil
	}

	return len(events), nil
}

// Clear очищает все события (для тестов)
func (s *InMemoryEventStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = make(map[string][]event.DomainEvent)
}

// GetAllAggregateIDs возвращает все ID агрегатов (для тестов)
func (s *InMemoryEventStore) GetAllAggregateIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.events))
	for id := range s.events {
		ids = append(ids, id)
	}

	return ids
}
