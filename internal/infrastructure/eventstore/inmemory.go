package eventstore

import (
	"context"
	"sync"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/event"
)

// InMemoryEventStore реализует EventStore in памяти for testing
type InMemoryEventStore struct {
	mu     sync.RWMutex
	events map[string][]event.DomainEvent
}

// NewInMemoryEventStore creates New in-memory event store
func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{
		events: make(map[string][]event.DomainEvent),
	}
}

// SaveEvents saves event for aggregate
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

	// check optimistic locking
	currentVersion := len(s.events[aggregateID])
	if currentVersion != expectedVersion {
		return appcore.ErrConcurrencyConflict
	}

	// Saving event
	s.events[aggregateID] = append(s.events[aggregateID], events...)

	return nil
}

// LoadEvents loads all event for aggregate
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

// GetVersion returns current version aggregate
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

// Clear clears all event (for tests)
func (s *InMemoryEventStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = make(map[string][]event.DomainEvent)
}

// GetAllAggregateIDs returns all ID агрегатов (for tests)
func (s *InMemoryEventStore) GetAllAggregateIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.events))
	for id := range s.events {
		ids = append(ids, id)
	}

	return ids
}
