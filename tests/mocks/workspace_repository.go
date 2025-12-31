package mocks

import (
	"context"
	"sync"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// MockWorkspaceRepository реализует репозиторий workspaces для тестирования
type MockWorkspaceRepository struct {
	mu                   sync.RWMutex
	workspaces           map[uuid.UUID]*workspace.Workspace
	workspacesByKeycloak map[string]*workspace.Workspace
	invitesByToken       map[string]*workspace.Invite
	members              map[string]*workspace.Member // key: "workspaceID:userID"
	calls                map[string]int
	SaveError            error
	FindError            error
}

// NewMockWorkspaceRepository создает новый mock репозиторий
func NewMockWorkspaceRepository() *MockWorkspaceRepository {
	return &MockWorkspaceRepository{
		workspaces:           make(map[uuid.UUID]*workspace.Workspace),
		workspacesByKeycloak: make(map[string]*workspace.Workspace),
		invitesByToken:       make(map[string]*workspace.Invite),
		members:              make(map[string]*workspace.Member),
		calls:                make(map[string]int),
	}
}

// FindByID находит workspace по ID
func (r *MockWorkspaceRepository) FindByID(
	_ context.Context,
	id uuid.UUID,
) (*workspace.Workspace, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["FindByID"]++

	if r.FindError != nil {
		return nil, r.FindError
	}

	ws, ok := r.workspaces[id]
	if !ok {
		return nil, errs.ErrNotFound
	}

	return ws, nil
}

// FindByKeycloakGroup находит workspace по Keycloak group ID
func (r *MockWorkspaceRepository) FindByKeycloakGroup(
	_ context.Context,
	keycloakGroupID string,
) (*workspace.Workspace, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["FindByKeycloakGroup"]++

	if r.FindError != nil {
		return nil, r.FindError
	}

	ws, ok := r.workspacesByKeycloak[keycloakGroupID]
	if !ok {
		return nil, errs.ErrNotFound
	}

	return ws, nil
}

// Save сохраняет workspace
func (r *MockWorkspaceRepository) Save(_ context.Context, ws *workspace.Workspace) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls["Save"]++

	if r.SaveError != nil {
		return r.SaveError
	}

	r.workspaces[ws.ID()] = ws
	r.workspacesByKeycloak[ws.KeycloakGroupID()] = ws

	// Сохраняем инвайты
	for _, invite := range ws.Invites() {
		r.invitesByToken[invite.Token()] = invite
	}

	return nil
}

// Delete удаляет workspace
func (r *MockWorkspaceRepository) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls["Delete"]++

	ws, ok := r.workspaces[id]
	if !ok {
		return errs.ErrNotFound
	}

	delete(r.workspacesByKeycloak, ws.KeycloakGroupID())
	delete(r.workspaces, id)

	// Удаляем членов workspace
	for key, member := range r.members {
		if member.WorkspaceID() == id {
			delete(r.members, key)
		}
	}

	return nil
}

// List возвращает список workspaces
func (r *MockWorkspaceRepository) List(
	_ context.Context,
	offset, limit int,
) ([]*workspace.Workspace, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["List"]++

	var allWorkspaces []*workspace.Workspace
	for _, ws := range r.workspaces {
		allWorkspaces = append(allWorkspaces, ws)
	}

	if offset >= len(allWorkspaces) {
		return []*workspace.Workspace{}, nil
	}

	end := min(offset+limit, len(allWorkspaces))
	return allWorkspaces[offset:end], nil
}

// Count возвращает количество workspaces
func (r *MockWorkspaceRepository) Count(_ context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["Count"]++

	return len(r.workspaces), nil
}

// FindInviteByToken находит invite по токену
func (r *MockWorkspaceRepository) FindInviteByToken(
	_ context.Context,
	token string,
) (*workspace.Invite, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["FindInviteByToken"]++

	invite, ok := r.invitesByToken[token]
	if !ok {
		return nil, errs.ErrNotFound
	}

	return invite, nil
}

// GetMember возвращает члена workspace
func (r *MockWorkspaceRepository) GetMember(
	_ context.Context,
	workspaceID, userID uuid.UUID,
) (*workspace.Member, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["GetMember"]++

	key := workspaceID.String() + ":" + userID.String()
	member, ok := r.members[key]
	if !ok {
		return nil, errs.ErrNotFound
	}

	return member, nil
}

// IsMember проверяет, является ли пользователь членом workspace
func (r *MockWorkspaceRepository) IsMember(
	_ context.Context,
	workspaceID, userID uuid.UUID,
) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["IsMember"]++

	key := workspaceID.String() + ":" + userID.String()
	_, ok := r.members[key]
	return ok, nil
}

// ListWorkspacesByUser возвращает workspaces пользователя
func (r *MockWorkspaceRepository) ListWorkspacesByUser(
	_ context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*workspace.Workspace, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["ListWorkspacesByUser"]++

	var result []*workspace.Workspace
	for key, member := range r.members {
		if member.UserID() == userID {
			// Extract workspaceID from key
			wsIDStr := key[:len(key)-len(userID.String())-1]
			wsID := uuid.UUID(wsIDStr)
			if ws, ok := r.workspaces[wsID]; ok {
				result = append(result, ws)
			}
		}
	}

	if offset >= len(result) {
		return []*workspace.Workspace{}, nil
	}

	end := min(offset+limit, len(result))
	return result[offset:end], nil
}

// CountWorkspacesByUser возвращает количество workspaces пользователя
func (r *MockWorkspaceRepository) CountWorkspacesByUser(
	_ context.Context,
	userID uuid.UUID,
) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["CountWorkspacesByUser"]++

	count := 0
	for _, member := range r.members {
		if member.UserID() == userID {
			count++
		}
	}
	return count, nil
}

// ListMembers возвращает членов workspace
func (r *MockWorkspaceRepository) ListMembers(
	_ context.Context,
	workspaceID uuid.UUID,
	offset, limit int,
) ([]*workspace.Member, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["ListMembers"]++

	var result []*workspace.Member
	for _, member := range r.members {
		if member.WorkspaceID() == workspaceID {
			result = append(result, member)
		}
	}

	if offset >= len(result) {
		return []*workspace.Member{}, nil
	}

	end := min(offset+limit, len(result))
	return result[offset:end], nil
}

// CountMembers возвращает количество членов workspace
func (r *MockWorkspaceRepository) CountMembers(
	_ context.Context,
	workspaceID uuid.UUID,
) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.calls["CountMembers"]++

	count := 0
	for _, member := range r.members {
		if member.WorkspaceID() == workspaceID {
			count++
		}
	}
	return count, nil
}

// AddMember добавляет члена в workspace
func (r *MockWorkspaceRepository) AddMember(
	_ context.Context,
	member *workspace.Member,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls["AddMember"]++

	if r.SaveError != nil {
		return r.SaveError
	}

	key := member.WorkspaceID().String() + ":" + member.UserID().String()
	r.members[key] = member
	return nil
}

// RemoveMember удаляет члена из workspace
func (r *MockWorkspaceRepository) RemoveMember(
	_ context.Context,
	workspaceID, userID uuid.UUID,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls["RemoveMember"]++

	key := workspaceID.String() + ":" + userID.String()
	if _, ok := r.members[key]; !ok {
		return errs.ErrNotFound
	}
	delete(r.members, key)
	return nil
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

	r.workspaces = make(map[uuid.UUID]*workspace.Workspace)
	r.workspacesByKeycloak = make(map[string]*workspace.Workspace)
	r.invitesByToken = make(map[string]*workspace.Invite)
	r.members = make(map[string]*workspace.Member)
	r.calls = make(map[string]int)
	r.SaveError = nil
	r.FindError = nil
}

// GetAll возвращает все workspaces
func (r *MockWorkspaceRepository) GetAll() []*workspace.Workspace {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var workspaces []*workspace.Workspace
	for _, ws := range r.workspaces {
		workspaces = append(workspaces, ws)
	}
	return workspaces
}

// GetAllMembers возвращает всех членов
func (r *MockWorkspaceRepository) GetAllMembers() []*workspace.Member {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var members []*workspace.Member
	for _, m := range r.members {
		members = append(members, m)
	}
	return members
}
