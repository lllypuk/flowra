package mocks

import (
	"context"
	"errors"
	"sync"

	"github.com/lllypuk/flowra/internal/domain/message"
	domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
)

var ErrMessageNotFound = errors.New("message not found")

// MockMessageRepository is a mock implementation of message repository for testing.
type MockMessageRepository struct {
	mu       sync.RWMutex
	messages map[string]*message.Message
	calls    map[string]int
}

// NewMockMessageRepository creates a new MockMessageRepository.
func NewMockMessageRepository() *MockMessageRepository {
	return &MockMessageRepository{
		messages: make(map[string]*message.Message),
		calls:    make(map[string]int),
	}
}

func (r *MockMessageRepository) Load(ctx context.Context, id domainUUID.UUID) (*message.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["Load"]++

	m, ok := r.messages[id.String()]
	if !ok {
		return nil, ErrMessageNotFound
	}

	return m, nil
}

func (r *MockMessageRepository) Save(ctx context.Context, m *message.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls["Save"]++
	r.messages[m.ID().String()] = m

	return nil
}

func (r *MockMessageRepository) SaveCallCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.calls["Save"]
}

func (r *MockMessageRepository) LoadCallCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.calls["Load"]
}

func (r *MockMessageRepository) GetAll() []*message.Message {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var messages []*message.Message
	for _, m := range r.messages {
		messages = append(messages, m)
	}
	return messages
}

func (r *MockMessageRepository) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.messages = make(map[string]*message.Message)
	r.calls = make(map[string]int)
}
