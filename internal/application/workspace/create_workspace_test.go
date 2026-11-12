package workspace_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lllypuk/flowra/internal/application/workspace"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	domainworkspace "github.com/lllypuk/flowra/internal/domain/workspace"
)

// mockWorkspaceRepository - mok repozitoriya for testing
type mockWorkspaceRepository struct {
	workspaces     map[uuid.UUID]*domainworkspace.Workspace
	invitesByToken map[string]*domainworkspace.Invite
	members        map[string]*domainworkspace.Member // key: "workspaceID:userID"
	saveError      error
	findError      error
}

func newMockWorkspaceRepository() *mockWorkspaceRepository {
	return &mockWorkspaceRepository{
		workspaces:     make(map[uuid.UUID]*domainworkspace.Workspace),
		invitesByToken: make(map[string]*domainworkspace.Invite),
		members:        make(map[string]*domainworkspace.Member),
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

func (m *mockWorkspaceRepository) Save(_ context.Context, ws *domainworkspace.Workspace) error {
	if m.saveError != nil {
		return m.saveError
	}
	m.workspaces[ws.ID()] = ws

	// Saving invayty
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

	end := min(offset+limit, len(allWorkspaces))

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

func (m *mockWorkspaceRepository) GetMember(
	_ context.Context,
	workspaceID, userID uuid.UUID,
) (*domainworkspace.Member, error) {
	key := workspaceID.String() + ":" + userID.String()
	if member, ok := m.members[key]; ok {
		return member, nil
	}
	return nil, errors.New("not found")
}

func (m *mockWorkspaceRepository) IsMember(
	_ context.Context,
	workspaceID, userID uuid.UUID,
) (bool, error) {
	key := workspaceID.String() + ":" + userID.String()
	_, ok := m.members[key]
	return ok, nil
}

func (m *mockWorkspaceRepository) ListWorkspacesByUser(
	_ context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*domainworkspace.Workspace, error) {
	var result []*domainworkspace.Workspace
	for key, member := range m.members {
		if member.UserID() == userID {
			// Extract workspaceID from key
			wsIDStr := key[:len(key)-len(userID.String())-1]
			wsID := uuid.UUID(wsIDStr)
			if ws, ok := m.workspaces[wsID]; ok {
				result = append(result, ws)
			}
		}
	}
	if offset >= len(result) {
		return []*domainworkspace.Workspace{}, nil
	}
	end := min(offset+limit, len(result))
	return result[offset:end], nil
}

func (m *mockWorkspaceRepository) CountWorkspacesByUser(
	_ context.Context,
	userID uuid.UUID,
) (int, error) {
	count := 0
	for _, member := range m.members {
		if member.UserID() == userID {
			count++
		}
	}
	return count, nil
}

func (m *mockWorkspaceRepository) ListMembers(
	_ context.Context,
	workspaceID uuid.UUID,
	offset, limit int,
) ([]*domainworkspace.Member, error) {
	var result []*domainworkspace.Member
	for _, member := range m.members {
		if member.WorkspaceID() == workspaceID {
			result = append(result, member)
		}
	}
	if offset >= len(result) {
		return []*domainworkspace.Member{}, nil
	}
	end := min(offset+limit, len(result))
	return result[offset:end], nil
}

func (m *mockWorkspaceRepository) CountMembers(
	_ context.Context,
	workspaceID uuid.UUID,
) (int, error) {
	count := 0
	for _, member := range m.members {
		if member.WorkspaceID() == workspaceID {
			count++
		}
	}
	return count, nil
}

func (m *mockWorkspaceRepository) AddMember(
	_ context.Context,
	member *domainworkspace.Member,
) error {
	if m.saveError != nil {
		return m.saveError
	}
	key := member.WorkspaceID().String() + ":" + member.UserID().String()
	m.members[key] = member
	return nil
}

func (m *mockWorkspaceRepository) RemoveMember(
	_ context.Context,
	workspaceID, userID uuid.UUID,
) error {
	key := workspaceID.String() + ":" + userID.String()
	if _, ok := m.members[key]; !ok {
		return errors.New("not found")
	}
	delete(m.members, key)
	return nil
}

func (m *mockWorkspaceRepository) UpdateMember(
	_ context.Context,
	member *domainworkspace.Member,
) error {
	if m.saveError != nil {
		return m.saveError
	}
	key := member.WorkspaceID().String() + ":" + member.UserID().String()
	if _, ok := m.members[key]; !ok {
		return errors.New("not found")
	}
	m.members[key] = member
	return nil
}

func TestCreateWorkspaceUseCase_Execute_Success(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewCreateWorkspaceUseCase(repo)
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

	// check that workspace sav
	if len(repo.workspaces) != 1 {
		t.Errorf("expected 1 workspace in repository, got %d", len(repo.workspaces))
	}

	owner, ok := repo.members[result.Value.ID().String()+":"+creatorID.String()]
	if !ok || owner.Role() != domainworkspace.RoleOwner {
		t.Error("expected creator to be the workspace owner")
	}
}

func TestCreateWorkspaceUseCase_Validate_MissingName(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewCreateWorkspaceUseCase(repo)

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
	useCase := workspace.NewCreateWorkspaceUseCase(repo)

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
	useCase := workspace.NewCreateWorkspaceUseCase(repo)

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

func TestCreateWorkspaceUseCase_Execute_SaveError(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	repo.saveError = errors.New("database error")
	useCase := workspace.NewCreateWorkspaceUseCase(repo)

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
}
