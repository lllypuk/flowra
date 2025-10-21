package mocks

import (
	"context"
	"sync"

	"github.com/lllypuk/teams-up/internal/application/shared"
	"github.com/lllypuk/teams-up/internal/domain/event"
)

// MockEventStore реализует shared.EventStore для тестирования
type MockEventStore struct {
	mu        sync.RWMutex
	events    map[string][]event.DomainEvent
	versions  map[string]int
	calls     map[string]int
	failNext  bool
	failError error
}

// NewMockEventStore создает новый mock event store
func NewMockEventStore() *MockEventStore {
	return &MockEventStore{
		events:   make(map[string][]event.DomainEvent),
		versions: make(map[string]int),
		calls:    make(map[string]int),
	}
}

// SaveEvents сохраняет события для агрегата
func (s *MockEventStore) SaveEvents(
	ctx context.Context,
	aggregateID string,
	events []event.DomainEvent,
	expectedVersion int,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.calls["SaveEvents"]++

	// Проверка на ошибку (если она установлена)
	if s.failNext {
		s.failNext = false
		return s.failError
	}

	// Проверка версии (optimistic locking)
	currentVersion, exists := s.versions[aggregateID]
	if exists && currentVersion != expectedVersion {
		return shared.ErrConcurrencyConflict
	}

	// Сохранение событий
	s.events[aggregateID] = append(s.events[aggregateID], events...)

	// Обновление версии
	newVersion := expectedVersion + len(events)
	s.versions[aggregateID] = newVersion

	return nil
}

// LoadEvents загружает все события для агрегата
func (s *MockEventStore) LoadEvents(ctx context.Context, aggregateID string) ([]event.DomainEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.calls["LoadEvents"]++

	events, ok := s.events[aggregateID]
	if !ok {
		return []event.DomainEvent{}, nil
	}

	// Возвращаем копию
	return append([]event.DomainEvent{}, events...), nil
}

// GetVersion возвращает текущую версию агрегата
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

// GetCallCount возвращает количество вызовов метода
func (s *MockEventStore) GetCallCount(method string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.calls[method]
}

// AllEvents возвращает все события (для тестов)
func (s *MockEventStore) AllEvents() map[string][]event.DomainEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Возвращаем копию
	result := make(map[string][]event.DomainEvent)
	for k, v := range s.events {
		result[k] = append([]event.DomainEvent{}, v...)
	}
	return result
}

// SetFailureNext установить ошибку для следующего вызова
func (s *MockEventStore) SetFailureNext(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failNext = true
	s.failError = err
}

// Reset очищает store
func (s *MockEventStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = make(map[string][]event.DomainEvent)
	s.versions = make(map[string]int)
	s.calls = make(map[string]int)
	s.failNext = false
	s.failError = nil
}
