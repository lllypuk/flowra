package workspace_test

import (
	"context"
	"errors"
	"testing"

	"github.com/flowra/flowra/internal/application/workspace"
	"github.com/flowra/flowra/internal/domain/uuid"
	domainworkspace "github.com/flowra/flowra/internal/domain/workspace"
)

// mockWorkspaceRepository - мок репозитория для тестирования
type mockWorkspaceRepository struct {
	workspaces           map[uuid.UUID]*domainworkspace.Workspace
	workspacesByKeycloak map[string]*domainworkspace.Workspace
	invitesByToken       map[string]*domainworkspace.Invite
	saveError            error
	findError            error
}

func newMockWorkspaceRepository() *mockWorkspaceRepository {
	return &mockWorkspaceRepository{
		workspaces:           make(map[uuid.UUID]*domainworkspace.Workspace),
		workspacesByKeycloak: make(map[string]*domainworkspace.Workspace),
		invitesByToken:       make(map[string]*domainworkspace.Invite),
	}
}

func (m *mockWorkspaceRepository) FindByID(_ context.Context, id uuid.UUID) (*domainworkspace.Workspace, error) {
	if m.findError != nil {
		return nil, m.findError
	}
	if ws, ok := m.workspaces[id]; ok {
		return ws, nil
	}
	return nil, errors.New("not found")
}

func (m *mockWorkspaceRepository) FindByKeycloakGroup(
	_ context.Context,
	keycloakGroupID string,
) (*domainworkspace.Workspace, error) {
	if m.findError != nil {
		return nil, m.findError
	}
	if ws, ok := m.workspacesByKeycloak[keycloakGroupID]; ok {
		return ws, nil
	}
	return nil, errors.New("not found")
}

func (m *mockWorkspaceRepository) Save(_ context.Context, ws *domainworkspace.Workspace) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.workspaces[ws.ID()] = ws
	m.workspacesByKeycloak[ws.KeycloakGroupID()] = ws

	// Сохраняем инвайты
	for _, invite := range ws.Invites() {
		m.invitesByToken[invite.Token()] = invite
	}

	return nil
}

func (m *mockWorkspaceRepository) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.workspaces, id)
	return nil
}

func (m *mockWorkspaceRepository) List(_ context.Context, offset, limit int) ([]*domainworkspace.Workspace, error) {
	var allWorkspaces []*domainworkspace.Workspace
	for _, ws := range m.workspaces {
		allWorkspaces = append(allWorkspaces, ws)
	}

	if offset >= len(allWorkspaces) {
		return []*domainworkspace.Workspace{}, nil
	}

	end := offset + limit
	if end > len(allWorkspaces) {
		end = len(allWorkspaces)
	}

	return allWorkspaces[offset:end], nil
}

func (m *mockWorkspaceRepository) Count(_ context.Context) (int, error) {
	return len(m.workspaces), nil
}

func (m *mockWorkspaceRepository) FindInviteByToken(_ context.Context, token string) (*domainworkspace.Invite, error) {
	if invite, ok := m.invitesByToken[token]; ok {
		return invite, nil
	}
	return nil, errors.New("not found")
}

// mockKeycloakClient - мок клиента Keycloak для тестирования
type mockKeycloakClient struct {
	groups           map[string]string   // groupID -> name
	groupUsers       map[string][]string // groupID -> []userID
	createGroupError error
	deleteGroupError error
	addUserError     error
	removeUserError  error
	nextGroupID      int
}

func newMockKeycloakClient() *mockKeycloakClient {
	return &mockKeycloakClient{
		groups:      make(map[string]string),
		groupUsers:  make(map[string][]string),
		nextGroupID: 1,
	}
}

func (m *mockKeycloakClient) CreateGroup(_ context.Context, name string) (string, error) {
	if m.createGroupError != nil {
		return "", m.createGroupError
	}
	groupID := uuid.NewUUID().String()
	m.groups[groupID] = name
	m.groupUsers[groupID] = []string{}
	return groupID, nil
}

func (m *mockKeycloakClient) DeleteGroup(_ context.Context, groupID string) error {
	if m.deleteGroupError != nil {
		return m.deleteGroupError
	}
	delete(m.groups, groupID)
	delete(m.groupUsers, groupID)
	return nil
}

func (m *mockKeycloakClient) AddUserToGroup(_ context.Context, userID, groupID string) error {
	if m.addUserError != nil {
		return m.addUserError
	}
	if _, ok := m.groupUsers[groupID]; !ok {
		return errors.New("group not found")
	}
	m.groupUsers[groupID] = append(m.groupUsers[groupID], userID)
	return nil
}

func (m *mockKeycloakClient) RemoveUserFromGroup(_ context.Context, userID, groupID string) error {
	if m.removeUserError != nil {
		return m.removeUserError
	}
	if users, ok := m.groupUsers[groupID]; ok {
		for i, uid := range users {
			if uid == userID {
				m.groupUsers[groupID] = append(users[:i], users[i+1:]...)
				break
			}
		}
	}
	return nil
}

func TestCreateWorkspaceUseCase_Execute_Success(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	keycloakClient := newMockKeycloakClient()
	useCase := workspace.NewCreateWorkspaceUseCase(repo, keycloakClient)
	creatorID := uuid.NewUUID()

	cmd := workspace.CreateWorkspaceCommand{
		Name:      "Test Workspace",
		CreatedBy: creatorID,
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Value == nil {
		t.Fatal("expected workspace to be created")
	}

	if result.Value.Name() != cmd.Name {
		t.Errorf("expected name %s, got %s", cmd.Name, result.Value.Name())
	}

	if result.Value.CreatedBy() != creatorID {
		t.Errorf("expected createdBy %s, got %s", creatorID, result.Value.CreatedBy())
	}

	// Проверка, что workspace сохранен
	if len(repo.workspaces) != 1 {
		t.Errorf("expected 1 workspace in repository, got %d", len(repo.workspaces))
	}

	// Проверка, что группа создана в Keycloak
	if len(keycloakClient.groups) != 1 {
		t.Errorf("expected 1 Keycloak group, got %d", len(keycloakClient.groups))
	}

	// Проверка, что создатель добавлен в группу
	groupID := result.Value.KeycloakGroupID()
	if users, ok := keycloakClient.groupUsers[groupID]; ok {
		if len(users) != 1 || users[0] != creatorID.String() {
			t.Errorf("expected creator to be added to Keycloak group")
		}
	} else {
		t.Error("expected Keycloak group to exist")
	}
}

func TestCreateWorkspaceUseCase_Validate_MissingName(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	keycloakClient := newMockKeycloakClient()
	useCase := workspace.NewCreateWorkspaceUseCase(repo, keycloakClient)

	cmd := workspace.CreateWorkspaceCommand{
		Name:      "",
		CreatedBy: uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for missing name")
	}
}

func TestCreateWorkspaceUseCase_Validate_NameTooLong(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	keycloakClient := newMockKeycloakClient()
	useCase := workspace.NewCreateWorkspaceUseCase(repo, keycloakClient)

	longName := make([]byte, 101)
	for i := range longName {
		longName[i] = 'a'
	}

	cmd := workspace.CreateWorkspaceCommand{
		Name:      string(longName),
		CreatedBy: uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for name too long")
	}
}

func TestCreateWorkspaceUseCase_Validate_InvalidCreatedBy(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	keycloakClient := newMockKeycloakClient()
	useCase := workspace.NewCreateWorkspaceUseCase(repo, keycloakClient)

	cmd := workspace.CreateWorkspaceCommand{
		Name:      "Test Workspace",
		CreatedBy: uuid.UUID(""),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for invalid createdBy")
	}
}

func TestCreateWorkspaceUseCase_Execute_KeycloakCreateGroupError(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	keycloakClient := newMockKeycloakClient()
	keycloakClient.createGroupError = errors.New("Keycloak error")
	useCase := workspace.NewCreateWorkspaceUseCase(repo, keycloakClient)

	cmd := workspace.CreateWorkspaceCommand{
		Name:      "Test Workspace",
		CreatedBy: uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected error from Keycloak group creation")
	}

	if !errors.Is(err, workspace.ErrKeycloakGroupCreationFailed) {
		t.Errorf("expected ErrKeycloakGroupCreationFailed, got: %v", err)
	}

	// Проверка, что workspace не сохранен
	if len(repo.workspaces) != 0 {
		t.Error("workspace should not be saved when Keycloak group creation fails")
	}
}

func TestCreateWorkspaceUseCase_Execute_SaveError(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	repo.saveError = errors.New("database error")
	keycloakClient := newMockKeycloakClient()
	useCase := workspace.NewCreateWorkspaceUseCase(repo, keycloakClient)

	cmd := workspace.CreateWorkspaceCommand{
		Name:      "Test Workspace",
		CreatedBy: uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected error from save operation")
	}

	// Проверка, что группа Keycloak была удалена (rollback)
	if len(keycloakClient.groups) != 0 {
		t.Error("Keycloak group should be deleted when workspace save fails")
	}
}
