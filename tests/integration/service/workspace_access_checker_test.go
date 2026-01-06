//go:build integration

package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
	"github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/lllypuk/flowra/internal/service"
	"github.com/lllypuk/flowra/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealWorkspaceAccessChecker_Integration_GetMembership(t *testing.T) {
	t.Run("user is member - returns membership from MongoDB", func(t *testing.T) {
		db := testutil.SetupTestMongoDB(t)
		repo := mongodb.NewMongoWorkspaceRepository(
			db.Collection("workspaces"),
			db.Collection("workspace_members"),
		)
		checker := service.NewRealWorkspaceAccessChecker(repo)
		ctx := context.Background()

		// Create workspace
		creatorID := uuid.NewUUID()
		ws, err := workspace.NewWorkspace("Integration Test Workspace", "keycloak-group-1", creatorID)
		require.NoError(t, err)

		err = repo.Save(ctx, ws)
		require.NoError(t, err)

		// Add member
		memberID := uuid.NewUUID()
		member := workspace.ReconstructMember(memberID, ws.ID(), workspace.RoleMember, time.Now())
		err = repo.AddMember(ctx, &member)
		require.NoError(t, err)

		// Test GetMembership
		membership, err := checker.GetMembership(ctx, ws.ID(), memberID)

		require.NoError(t, err)
		require.NotNil(t, membership)
		assert.Equal(t, ws.ID(), membership.WorkspaceID)
		assert.Equal(t, memberID, membership.UserID)
		assert.Equal(t, "member", membership.Role)
		assert.Equal(t, "Integration Test Workspace", membership.WorkspaceName)
	})

	t.Run("user is owner - returns membership with owner role", func(t *testing.T) {
		db := testutil.SetupTestMongoDB(t)
		repo := mongodb.NewMongoWorkspaceRepository(
			db.Collection("workspaces"),
			db.Collection("workspace_members"),
		)
		checker := service.NewRealWorkspaceAccessChecker(repo)
		ctx := context.Background()

		// Create workspace
		ownerID := uuid.NewUUID()
		ws, err := workspace.NewWorkspace("Owner Test Workspace", "keycloak-group-2", ownerID)
		require.NoError(t, err)

		err = repo.Save(ctx, ws)
		require.NoError(t, err)

		// Add owner as member
		member := workspace.ReconstructMember(ownerID, ws.ID(), workspace.RoleOwner, time.Now())
		err = repo.AddMember(ctx, &member)
		require.NoError(t, err)

		// Test GetMembership
		membership, err := checker.GetMembership(ctx, ws.ID(), ownerID)

		require.NoError(t, err)
		require.NotNil(t, membership)
		assert.Equal(t, "owner", membership.Role)
	})

	t.Run("user is admin - returns membership with admin role", func(t *testing.T) {
		db := testutil.SetupTestMongoDB(t)
		repo := mongodb.NewMongoWorkspaceRepository(
			db.Collection("workspaces"),
			db.Collection("workspace_members"),
		)
		checker := service.NewRealWorkspaceAccessChecker(repo)
		ctx := context.Background()

		// Create workspace
		creatorID := uuid.NewUUID()
		ws, err := workspace.NewWorkspace("Admin Test Workspace", "keycloak-group-3", creatorID)
		require.NoError(t, err)

		err = repo.Save(ctx, ws)
		require.NoError(t, err)

		// Add admin as member
		adminID := uuid.NewUUID()
		member := workspace.ReconstructMember(adminID, ws.ID(), workspace.RoleAdmin, time.Now())
		err = repo.AddMember(ctx, &member)
		require.NoError(t, err)

		// Test GetMembership
		membership, err := checker.GetMembership(ctx, ws.ID(), adminID)

		require.NoError(t, err)
		require.NotNil(t, membership)
		assert.Equal(t, "admin", membership.Role)
	})

	t.Run("user is not member - returns nil without error", func(t *testing.T) {
		db := testutil.SetupTestMongoDB(t)
		repo := mongodb.NewMongoWorkspaceRepository(
			db.Collection("workspaces"),
			db.Collection("workspace_members"),
		)
		checker := service.NewRealWorkspaceAccessChecker(repo)
		ctx := context.Background()

		// Create workspace
		creatorID := uuid.NewUUID()
		ws, err := workspace.NewWorkspace("No Member Workspace", "keycloak-group-4", creatorID)
		require.NoError(t, err)

		err = repo.Save(ctx, ws)
		require.NoError(t, err)

		// Test GetMembership for non-member
		nonMemberID := uuid.NewUUID()
		membership, err := checker.GetMembership(ctx, ws.ID(), nonMemberID)

		require.NoError(t, err)
		assert.Nil(t, membership)
	})

	t.Run("workspace not found - returns ErrWorkspaceNotFound", func(t *testing.T) {
		db := testutil.SetupTestMongoDB(t)
		repo := mongodb.NewMongoWorkspaceRepository(
			db.Collection("workspaces"),
			db.Collection("workspace_members"),
		)
		checker := service.NewRealWorkspaceAccessChecker(repo)
		ctx := context.Background()

		nonExistentWorkspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		membership, err := checker.GetMembership(ctx, nonExistentWorkspaceID, userID)

		require.Error(t, err)
		require.ErrorIs(t, err, middleware.ErrWorkspaceNotFound)
		assert.Nil(t, membership)
	})
}

func TestRealWorkspaceAccessChecker_Integration_WorkspaceExists(t *testing.T) {
	t.Run("workspace exists - returns true", func(t *testing.T) {
		db := testutil.SetupTestMongoDB(t)
		repo := mongodb.NewMongoWorkspaceRepository(
			db.Collection("workspaces"),
			db.Collection("workspace_members"),
		)
		checker := service.NewRealWorkspaceAccessChecker(repo)
		ctx := context.Background()

		// Create workspace
		creatorID := uuid.NewUUID()
		ws, err := workspace.NewWorkspace("Existing Workspace", "keycloak-group-5", creatorID)
		require.NoError(t, err)

		err = repo.Save(ctx, ws)
		require.NoError(t, err)

		// Test WorkspaceExists
		exists, err := checker.WorkspaceExists(ctx, ws.ID())

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("workspace not found - returns false without error", func(t *testing.T) {
		db := testutil.SetupTestMongoDB(t)
		repo := mongodb.NewMongoWorkspaceRepository(
			db.Collection("workspaces"),
			db.Collection("workspace_members"),
		)
		checker := service.NewRealWorkspaceAccessChecker(repo)
		ctx := context.Background()

		nonExistentWorkspaceID := uuid.NewUUID()

		exists, err := checker.WorkspaceExists(ctx, nonExistentWorkspaceID)

		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestRealWorkspaceAccessChecker_Integration_MultipleMembers(t *testing.T) {
	t.Run("multiple members in same workspace", func(t *testing.T) {
		db := testutil.SetupTestMongoDB(t)
		repo := mongodb.NewMongoWorkspaceRepository(
			db.Collection("workspaces"),
			db.Collection("workspace_members"),
		)
		checker := service.NewRealWorkspaceAccessChecker(repo)
		ctx := context.Background()

		// Create workspace
		creatorID := uuid.NewUUID()
		ws, err := workspace.NewWorkspace("Multi Member Workspace", "keycloak-group-6", creatorID)
		require.NoError(t, err)

		err = repo.Save(ctx, ws)
		require.NoError(t, err)

		// Add multiple members with different roles
		ownerID := uuid.NewUUID()
		adminID := uuid.NewUUID()
		memberID := uuid.NewUUID()

		ownerMember := workspace.ReconstructMember(ownerID, ws.ID(), workspace.RoleOwner, time.Now())
		adminMember := workspace.ReconstructMember(adminID, ws.ID(), workspace.RoleAdmin, time.Now())
		regularMember := workspace.ReconstructMember(memberID, ws.ID(), workspace.RoleMember, time.Now())

		require.NoError(t, repo.AddMember(ctx, &ownerMember))
		require.NoError(t, repo.AddMember(ctx, &adminMember))
		require.NoError(t, repo.AddMember(ctx, &regularMember))

		// Verify each member has correct role
		ownerMembership, err := checker.GetMembership(ctx, ws.ID(), ownerID)
		require.NoError(t, err)
		require.NotNil(t, ownerMembership)
		assert.Equal(t, "owner", ownerMembership.Role)

		adminMembership, err := checker.GetMembership(ctx, ws.ID(), adminID)
		require.NoError(t, err)
		require.NotNil(t, adminMembership)
		assert.Equal(t, "admin", adminMembership.Role)

		memberMembership, err := checker.GetMembership(ctx, ws.ID(), memberID)
		require.NoError(t, err)
		require.NotNil(t, memberMembership)
		assert.Equal(t, "member", memberMembership.Role)

		// Non-member still returns nil
		nonMemberID := uuid.NewUUID()
		nonMembership, err := checker.GetMembership(ctx, ws.ID(), nonMemberID)
		require.NoError(t, err)
		assert.Nil(t, nonMembership)
	})
}
