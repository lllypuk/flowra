package mocks

import (
	"context"
	"sync"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/task"
	domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
)

// MockTaskRepository implements taskapp.CommandRepository for testing.
type MockTaskRepository struct {
	mu       sync.RWMutex
	tasks    map[string]*task.Aggregate
	calls    map[string]int
	failNext error
}

// NewMockTaskRepository creates a new mock task repository.
func NewMockTaskRepository() *MockTaskRepository {
	return &MockTaskRepository{
		tasks: make(map[string]*task.Aggregate),
		calls: make(map[string]int),
	}
}

// Load loads a task aggregate by ID.
func (r *MockTaskRepository) Load(ctx context.Context, id domainUUID.UUID) (*task.Aggregate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls["Load"]++

	t, ok := r.tasks[id.String()]
	if !ok {
		return nil, errs.ErrNotFound
	}

	return t, nil
}

// Save saves a task aggregate.
func (r *MockTaskRepository) Save(ctx context.Context, t *task.Aggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls["Save"]++

	if r.failNext != nil {
		err := r.failNext
		r.failNext = nil
		return err
	}

	r.tasks[t.ID().String()] = t
	// Clear uncommitted events after save, simulating real event store behavior
	t.MarkEventsAsCommitted()
	return nil
}

// GetEvents returns all events for a task (mock returns empty slice).
func (r *MockTaskRepository) GetEvents(ctx context.Context, taskID domainUUID.UUID) ([]event.DomainEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["GetEvents"]++

	// Mock returns empty events - actual event handling is done internally
	return []event.DomainEvent{}, nil
}

// SetFailureNext sets an error to be returned on the next Save call.
func (r *MockTaskRepository) SetFailureNext(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failNext = err
}

// SaveCallCount returns the number of times Save was called.
func (r *MockTaskRepository) SaveCallCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.calls["Save"]
}

// LoadCallCount returns the number of times Load was called.
func (r *MockTaskRepository) LoadCallCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.calls["Load"]
}

// GetAll returns all stored tasks.
func (r *MockTaskRepository) GetAll() []*task.Aggregate {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tasks []*task.Aggregate
	for _, t := range r.tasks {
		tasks = append(tasks, t)
	}
	return tasks
}

// Reset clears all stored tasks and call counts.
func (r *MockTaskRepository) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tasks = make(map[string]*task.Aggregate)
	r.calls = make(map[string]int)
}
