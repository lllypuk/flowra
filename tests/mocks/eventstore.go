package mocks

import (
	"context"
	"sync"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/event"
)

// MockEventStore implements appcore.EventStore for testing
type MockEventStore struct {
	mu        sync.RWMutex
	events    map[string][]event.DomainEvent
	versions  map[string]int
	calls     map[string]int
	failNext  bool
	failError error
}

// NewMockEventStore creates a new mock event store
func NewMockEventStore() *MockEventStore {
	return &MockEventStore{
		events:   make(map[string][]event.DomainEvent),
		versions: make(map[string]int),
		calls:    make(map[string]int),
	}
}

// SaveEvents saves events for an aggregate
func (s *MockEventStore) SaveEvents(
	ctx context.Context,
	aggregateID string,
	events []event.DomainEvent,
	expectedVersion int,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls["SaveEvents"]++

	// Check for error (if set)
	if s.failNext {
		s.failNext = false
		return s.failError
	}

	// Version check (optimistic locking)
	currentVersion, exists := s.versions[aggregateID]
	if exists && currentVersion != expectedVersion {
		return appcore.ErrConcurrencyConflict
	}

	// Save events
	s.events[aggregateID] = append(s.events[aggregateID], events...)

	// Update version
	newVersion := expectedVersion + len(events)
	s.versions[aggregateID] = newVersion

	return nil
}

// LoadEvents loads all events for an aggregate
func (s *MockEventStore) LoadEvents(ctx context.Context, aggregateID string) ([]event.DomainEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.calls["LoadEvents"]++

	events, ok := s.events[aggregateID]
	if !ok || len(events) == 0 {
		return nil, appcore.ErrAggregateNotFound
	}

	// Return a copy
	return append([]event.DomainEvent{}, events...), nil
}

// GetVersion returns the current version of an aggregate
func (s *MockEventStore) GetVersion(ctx context.Context, aggregateID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.calls["GetVersion"]++

	version, ok := s.versions[aggregateID]
	if !ok {
		return 0, nil
	}

	return version, nil
}

// GetCallCount returns the number of method calls
func (s *MockEventStore) GetCallCount(method string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.calls[method]
}

// AllEvents returns all events (for tests)
func (s *MockEventStore) AllEvents() map[string][]event.DomainEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy
	result := make(map[string][]event.DomainEvent)
	for k, v := range s.events {
		result[k] = append([]event.DomainEvent{}, v...)
	}
	return result
}

// SetFailureNext sets an error for the next call
func (s *MockEventStore) SetFailureNext(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failNext = true
	s.failError = err
}

// Reset clears the store
func (s *MockEventStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = make(map[string][]event.DomainEvent)
	s.versions = make(map[string]int)
	s.calls = make(map[string]int)
	s.failNext = false
	s.failError = nil
}
