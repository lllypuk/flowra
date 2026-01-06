package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/lllypuk/flowra/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockWorkspaceQueryRepository is a test mock for WorkspaceQueryRepository
type mockWorkspaceQueryRepository struct {
	workspaces map[uuid.UUID]*workspace.Workspace
	members    map[string]*workspace.Member // key: "workspaceID:userID"
	findError  error
	getMemErr  error
}

func newMockWorkspaceQueryRepository() *mockWorkspaceQueryRepository {
	return &mockWorkspaceQueryRepository{
		workspaces: make(map[uuid.UUID]*workspace.Workspace),
		members:    make(map[string]*workspace.Member),
	}
}

func (m *mockWorkspaceQueryRepository) FindByID(
	_ context.Context,
	id uuid.UUID,
) (*workspace.Workspace, error) {
	if m.findError != nil {
		return nil, m.findError
	}

	ws, ok := m.workspaces[id]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return ws, nil
}

func (m *mockWorkspaceQueryRepository) GetMember(
	_ context.Context,
	workspaceID, userID uuid.UUID,
) (*workspace.Member, error) {
	if m.getMemErr != nil {
		return nil, m.getMemErr
	}

	key := workspaceID.String() + ":" + userID.String()
	member, ok := m.members[key]
	if !ok {
		return nil, errs.ErrNotFound
	}
	return member, nil
}

func (m *mockWorkspaceQueryRepository) addWorkspace(ws *workspace.Workspace) {
	m.workspaces[ws.ID()] = ws
}

func (m *mockWorkspaceQueryRepository) addMember(member *workspace.Member) {
	key := member.WorkspaceID().String() + ":" + member.UserID().String()
	m.members[key] = member
}

// Helper to create test workspace
func createTestWorkspace(t *testing.T, name string, createdBy uuid.UUID) *workspace.Workspace {
	t.Helper()
	ws, err := workspace.NewWorkspace(name, "keycloak-group-"+name, createdBy)
	require.NoError(t, err)
	return ws
}

// Helper to create test member
func createTestMember(workspaceID, userID uuid.UUID, role workspace.Role) *workspace.Member {
	member := workspace.ReconstructMember(userID, workspaceID, role, time.Now())
	return &member
}

func TestRealWorkspaceAccessChecker_GetMembership(t *testing.T) {
	t.Run("user is member - returns membership", func(t *testing.T) {
		repo := newMockWorkspaceQueryRepository()
		checker := service.NewRealWorkspaceAccessChecker(repo)

		creatorID := uuid.NewUUID()
		memberID := uuid.NewUUID()
		ws := createTestWorkspace(t, "Test Workspace", creatorID)
		member := createTestMember(ws.ID(), memberID, workspace.RoleMember)

		repo.addWorkspace(ws)
		repo.addMember(member)

		membership, err := checker.GetMembership(context.Background(), ws.ID(), memberID)

		require.NoError(t, err)
		require.NotNil(t, membership)
		assert.Equal(t, ws.ID(), membership.WorkspaceID)
		assert.Equal(t, memberID, membership.UserID)
		assert.Equal(t, "member", membership.Role)
		assert.Equal(t, "Test Workspace", membership.WorkspaceName)
	})

	t.Run("user is owner - returns membership with owner role", func(t *testing.T) {
		repo := newMockWorkspaceQueryRepository()
		checker := service.NewRealWorkspaceAccessChecker(repo)

		ownerID := uuid.NewUUID()
		ws := createTestWorkspace(t, "Owner Workspace", ownerID)
		member := createTestMember(ws.ID(), ownerID, workspace.RoleOwner)

		repo.addWorkspace(ws)
		repo.addMember(member)

		membership, err := checker.GetMembership(context.Background(), ws.ID(), ownerID)

		require.NoError(t, err)
		require.NotNil(t, membership)
		assert.Equal(t, "owner", membership.Role)
	})

	t.Run("user is admin - returns membership with admin role", func(t *testing.T) {
		repo := newMockWorkspaceQueryRepository()
		checker := service.NewRealWorkspaceAccessChecker(repo)

		creatorID := uuid.NewUUID()
		adminID := uuid.NewUUID()
		ws := createTestWorkspace(t, "Admin Workspace", creatorID)
		member := createTestMember(ws.ID(), adminID, workspace.RoleAdmin)

		repo.addWorkspace(ws)
		repo.addMember(member)

		membership, err := checker.GetMembership(context.Background(), ws.ID(), adminID)

		require.NoError(t, err)
		require.NotNil(t, membership)
		assert.Equal(t, "admin", membership.Role)
	})

	t.Run("user is not member - returns nil without error", func(t *testing.T) {
		repo := newMockWorkspaceQueryRepository()
		checker := service.NewRealWorkspaceAccessChecker(repo)

		creatorID := uuid.NewUUID()
		nonMemberID := uuid.NewUUID()
		ws := createTestWorkspace(t, "Test Workspace", creatorID)

		repo.addWorkspace(ws)
		// No member added for nonMemberID

		membership, err := checker.GetMembership(context.Background(), ws.ID(), nonMemberID)

		require.NoError(t, err)
		assert.Nil(t, membership)
	})

	t.Run("workspace not found - returns ErrWorkspaceNotFound", func(t *testing.T) {
		repo := newMockWorkspaceQueryRepository()
		checker := service.NewRealWorkspaceAccessChecker(repo)

		nonExistentWorkspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		membership, err := checker.GetMembership(context.Background(), nonExistentWorkspaceID, userID)

		require.Error(t, err)
		require.ErrorIs(t, err, middleware.ErrWorkspaceNotFound)
		assert.Nil(t, membership)
	})

	t.Run("repository FindByID error - returns error", func(t *testing.T) {
		repo := newMockWorkspaceQueryRepository()
		checker := service.NewRealWorkspaceAccessChecker(repo)

		repoError := errors.New("database connection failed")
		repo.findError = repoError

		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		membership, err := checker.GetMembership(context.Background(), workspaceID, userID)

		require.Error(t, err)
		require.ErrorIs(t, err, repoError)
		assert.Nil(t, membership)
	})

	t.Run("repository GetMember error - returns error", func(t *testing.T) {
		repo := newMockWorkspaceQueryRepository()
		checker := service.NewRealWorkspaceAccessChecker(repo)

		creatorID := uuid.NewUUID()
		userID := uuid.NewUUID()
		ws := createTestWorkspace(t, "Test Workspace", creatorID)

		repo.addWorkspace(ws)
		repoError := errors.New("member lookup failed")
		repo.getMemErr = repoError

		membership, err := checker.GetMembership(context.Background(), ws.ID(), userID)

		require.Error(t, err)
		require.ErrorIs(t, err, repoError)
		assert.Nil(t, membership)
	})
}

func TestRealWorkspaceAccessChecker_WorkspaceExists(t *testing.T) {
	t.Run("workspace exists - returns true", func(t *testing.T) {
		repo := newMockWorkspaceQueryRepository()
		checker := service.NewRealWorkspaceAccessChecker(repo)

		creatorID := uuid.NewUUID()
		ws := createTestWorkspace(t, "Existing Workspace", creatorID)

		repo.addWorkspace(ws)

		exists, err := checker.WorkspaceExists(context.Background(), ws.ID())

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("workspace not found - returns false without error", func(t *testing.T) {
		repo := newMockWorkspaceQueryRepository()
		checker := service.NewRealWorkspaceAccessChecker(repo)

		nonExistentWorkspaceID := uuid.NewUUID()

		exists, err := checker.WorkspaceExists(context.Background(), nonExistentWorkspaceID)

		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("repository error - returns error", func(t *testing.T) {
		repo := newMockWorkspaceQueryRepository()
		checker := service.NewRealWorkspaceAccessChecker(repo)

		repoError := errors.New("database connection failed")
		repo.findError = repoError

		workspaceID := uuid.NewUUID()

		exists, err := checker.WorkspaceExists(context.Background(), workspaceID)

		require.Error(t, err)
		require.ErrorIs(t, err, repoError)
		assert.False(t, exists)
	})
}

func TestRealWorkspaceAccessChecker_ImplementsInterface(t *testing.T) {
	t.Helper()
	// Compile-time check that RealWorkspaceAccessChecker implements WorkspaceAccessChecker
	var _ middleware.WorkspaceAccessChecker = (*service.RealWorkspaceAccessChecker)(nil)
}

func TestNewRealWorkspaceAccessChecker(t *testing.T) {
	repo := newMockWorkspaceQueryRepository()
	checker := service.NewRealWorkspaceAccessChecker(repo)

	require.NotNil(t, checker)
}
