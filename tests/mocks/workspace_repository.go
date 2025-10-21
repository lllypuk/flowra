package mocks

import (
	"context"
	"sync"

	"github.com/flowra/flowra/internal/domain/errs"
	"github.com/google/uuid"
)

// WorkspaceData представляет данные workspace для тестов
type WorkspaceData struct {
	ID   uuid.UUID
	Name string
}

// MockWorkspaceRepository реализует репозиторий workspaces для тестирования
type MockWorkspaceRepository struct {
	mu         sync.RWMutex
	workspaces map[uuid.UUID]*WorkspaceData
	calls      map[string]int
}

// NewMockWorkspaceRepository создает новый mock репозиторий
func NewMockWorkspaceRepository() *MockWorkspaceRepository {
	return &MockWorkspaceRepository{
		workspaces: make(map[uuid.UUID]*WorkspaceData),
		calls:      make(map[string]int),
	}
}

// Load загружает workspace по ID
func (r *MockWorkspaceRepository) Load(ctx context.Context, id uuid.UUID) (*WorkspaceData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["Load"]++

	ws, ok := r.workspaces[id]
	if !ok {
		return nil, errs.ErrNotFound
	}

	return ws, nil
}

// Save сохраняет workspace
func (r *MockWorkspaceRepository) Save(ctx context.Context, ws *WorkspaceData) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls["Save"]++
	r.workspaces[ws.ID] = ws

	return nil
}

// Exists проверяет существование workspace
func (r *MockWorkspaceRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["Exists"]++

	_, ok := r.workspaces[id]
	return ok, nil
}

// GetAll возвращает все workspaces
func (r *MockWorkspaceRepository) GetAll() []*WorkspaceData {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var workspaces []*WorkspaceData
	for _, ws := range r.workspaces {
		workspaces = append(workspaces, ws)
	}
	return workspaces
}

// GetCallCount возвращает количество вызовов метода
func (r *MockWorkspaceRepository) GetCallCount(method string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.calls[method]
}

// Reset очищает все данные
func (r *MockWorkspaceRepository) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.workspaces = make(map[uuid.UUID]*WorkspaceData)
	r.calls = make(map[string]int)
}
