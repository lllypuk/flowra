package mocks

import (
	"context"
	"errors"
	"sync"

	"github.com/lllypuk/flowra/internal/domain/message"
	domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
)

var ErrMessageNotFound = errors.New("message not found")

type MessageRepository struct {
	mu       sync.RWMutex
	messages map[string]*message.Message
	calls    map[string]int
}

func NewMessageRepository() *MessageRepository {
	return &MessageRepository{
		messages: make(map[string]*message.Message),
		calls:    make(map[string]int),
	}
}

func (r *MessageRepository) Load(ctx context.Context, id domainUUID.UUID) (*message.Message, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["Load"]++

	m, ok := r.messages[id.String()]
	if !ok {
		return nil, ErrMessageNotFound
	}

	return m, nil
}

func (r *MessageRepository) Save(ctx context.Context, m *message.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls["Save"]++
	r.messages[m.ID().String()] = m

	return nil
}

func (r *MessageRepository) SaveCallCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.calls["Save"]
}

func (r *MessageRepository) LoadCallCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.calls["Load"]
}

func (r *MessageRepository) GetAll() []*message.Message {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var messages []*message.Message
	for _, m := range r.messages {
		messages = append(messages, m)
	}
	return messages
}

func (r *MessageRepository) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.messages = make(map[string]*message.Message)
	r.calls = make(map[string]int)
}
