package mocks

import (
	"context"
	"errors"
	"sync"

	"github.com/lllypuk/flowra/internal/domain/chat"
	domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
)

var ErrChatNotFound = errors.New("chat not found")

type MockChatRepository struct {
	mu    sync.RWMutex
	chats map[string]*chat.Chat
	calls map[string]int
}

func NewMockChatRepository() *MockChatRepository {
	return &MockChatRepository{
		chats: make(map[string]*chat.Chat),
		calls: make(map[string]int),
	}
}

func (r *MockChatRepository) Load(ctx context.Context, id domainUUID.UUID) (*chat.Chat, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["Load"]++

	c, ok := r.chats[id.String()]
	if !ok {
		return nil, ErrChatNotFound
	}

	return c, nil
}

func (r *MockChatRepository) Save(ctx context.Context, c *chat.Chat) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls["Save"]++
	r.chats[c.ID().String()] = c

	return nil
}

func (r *MockChatRepository) SaveCallCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.calls["Save"]
}

func (r *MockChatRepository) LoadCallCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.calls["Load"]
}

func (r *MockChatRepository) GetAll() []*chat.Chat {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var chats []*chat.Chat
	for _, c := range r.chats {
		chats = append(chats, c)
	}
	return chats
}

func (r *MockChatRepository) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.chats = make(map[string]*chat.Chat)
	r.calls = make(map[string]int)
}
