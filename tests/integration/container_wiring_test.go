//go:build integration

package integration_test

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

// TestContainerWiring_RealAccessChecker verifies that the real access checker
// works correctly with MongoDB repository.
func TestContainerWiring_RealAccessChecker(t *testing.T) {
	db := testutil.SetupTestMongoDB(t)

	workspaceRepo := mongodb.NewMongoWorkspaceRepository(
		db.Collection("workspaces"),
		db.Collection("workspace_members"),
	)

	// This is the key wiring: RealWorkspaceAccessChecker with real repo
	accessChecker := service.NewRealWorkspaceAccessChecker(workspaceRepo)

	// Verify it implements the interface
	var _ middleware.WorkspaceAccessChecker = accessChecker

	// Verify it's NOT a mock
	_, isMock := interface{}(accessChecker).(*middleware.MockWorkspaceAccessChecker)
	assert.False(t, isMock, "should not be a mock implementation")
}

// TestContainerWiring_MemberService verifies that MemberService works with
// real MongoDB repository.
func TestContainerWiring_MemberService(t *testing.T) {
	db := testutil.SetupTestMongoDB(t)
	ctx := context.Background()

	workspaceRepo := mongodb.NewMongoWorkspaceRepository(
		db.Collection("workspaces"),
		db.Collection("workspace_members"),
	)

	// This is the key wiring: MemberService with real repo (used as both command and query)
	memberService := service.NewMemberService(workspaceRepo, workspaceRepo)

	// Create a workspace first
	creatorID := uuid.NewUUID()
	ws, err := workspace.NewWorkspace("Test Workspace", "keycloak-group-test", creatorID)
	require.NoError(t, err)
	require.NoError(t, workspaceRepo.Save(ctx, ws))

	// Test adding a member
	newMemberID := uuid.NewUUID()
	member, err := memberService.AddMember(ctx, ws.ID(), newMemberID, workspace.RoleMember)
	require.NoError(t, err)
	require.NotNil(t, member)
	assert.Equal(t, newMemberID, member.UserID())
	assert.Equal(t, workspace.RoleMember, member.Role())

	// Test listing members
	members, total, err := memberService.ListMembers(ctx, ws.ID(), 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, members, 1)

	// Test getting a member
	fetchedMember, err := memberService.GetMember(ctx, ws.ID(), newMemberID)
	require.NoError(t, err)
	require.NotNil(t, fetchedMember)
	assert.Equal(t, newMemberID, fetchedMember.UserID())

	// Test updating member role
	updatedMember, err := memberService.UpdateMemberRole(ctx, ws.ID(), newMemberID, workspace.RoleAdmin)
	require.NoError(t, err)
	require.NotNil(t, updatedMember)
	assert.Equal(t, workspace.RoleAdmin, updatedMember.Role())

	// Test removing a member
	err = memberService.RemoveMember(ctx, ws.ID(), newMemberID)
	require.NoError(t, err)

	// Verify member is removed
	members, total, err = memberService.ListMembers(ctx, ws.ID(), 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, members)
}

// TestContainerWiring_MemberService_OwnerProtection verifies that owner cannot be removed.
func TestContainerWiring_MemberService_OwnerProtection(t *testing.T) {
	db := testutil.SetupTestMongoDB(t)
	ctx := context.Background()

	workspaceRepo := mongodb.NewMongoWorkspaceRepository(
		db.Collection("workspaces"),
		db.Collection("workspace_members"),
	)

	memberService := service.NewMemberService(workspaceRepo, workspaceRepo)

	// Create workspace and add owner
	creatorID := uuid.NewUUID()
	ws, err := workspace.NewWorkspace("Owner Protected Workspace", "keycloak-group-owner", creatorID)
	require.NoError(t, err)
	require.NoError(t, workspaceRepo.Save(ctx, ws))

	ownerMember := workspace.ReconstructMember(creatorID, ws.ID(), workspace.RoleOwner, time.Now())
	require.NoError(t, workspaceRepo.AddMember(ctx, &ownerMember))

	// Try to remove owner - should fail
	err = memberService.RemoveMember(ctx, ws.ID(), creatorID)
	require.Error(t, err)

	// Try to change owner role - should fail
	_, err = memberService.UpdateMemberRole(ctx, ws.ID(), creatorID, workspace.RoleAdmin)
	require.Error(t, err)
}

// TestContainerWiring_NoOpKeycloakClient verifies NoOpKeycloakClient works correctly.
func TestContainerWiring_NoOpKeycloakClient(t *testing.T) {
	ctx := context.Background()
	client := service.NewNoOpKeycloakClient()

	// Test complete workflow
	groupID, err := client.CreateGroup(ctx, "test-workspace")
	require.NoError(t, err)
	require.NotEmpty(t, groupID)

	err = client.AddUserToGroup(ctx, "user-123", groupID)
	require.NoError(t, err)

	err = client.RemoveUserFromGroup(ctx, "user-123", groupID)
	require.NoError(t, err)

	err = client.DeleteGroup(ctx, groupID)
	require.NoError(t, err)
}

// TestContainerWiring_AccessChecker_WorkspaceExists verifies workspace existence check.
func TestContainerWiring_AccessChecker_WorkspaceExists(t *testing.T) {
	db := testutil.SetupTestMongoDB(t)
	ctx := context.Background()

	workspaceRepo := mongodb.NewMongoWorkspaceRepository(
		db.Collection("workspaces"),
		db.Collection("workspace_members"),
	)

	accessChecker := service.NewRealWorkspaceAccessChecker(workspaceRepo)

	// Non-existent workspace
	exists, err := accessChecker.WorkspaceExists(ctx, uuid.NewUUID())
	require.NoError(t, err)
	assert.False(t, exists)

	// Create workspace
	creatorID := uuid.NewUUID()
	ws, err := workspace.NewWorkspace("Existing Workspace", "keycloak-exists", creatorID)
	require.NoError(t, err)
	require.NoError(t, workspaceRepo.Save(ctx, ws))

	// Check existence
	exists, err = accessChecker.WorkspaceExists(ctx, ws.ID())
	require.NoError(t, err)
	assert.True(t, exists)
}

// TestContainerWiring_FullMembershipFlow tests the complete membership check flow.
func TestContainerWiring_FullMembershipFlow(t *testing.T) {
	db := testutil.SetupTestMongoDB(t)
	ctx := context.Background()

	workspaceRepo := mongodb.NewMongoWorkspaceRepository(
		db.Collection("workspaces"),
		db.Collection("workspace_members"),
	)

	accessChecker := service.NewRealWorkspaceAccessChecker(workspaceRepo)
	memberService := service.NewMemberService(workspaceRepo, workspaceRepo)

	// Create workspace
	creatorID := uuid.NewUUID()
	ws, err := workspace.NewWorkspace("Membership Flow Workspace", "keycloak-flow", creatorID)
	require.NoError(t, err)
	require.NoError(t, workspaceRepo.Save(ctx, ws))

	// Initially user is not a member
	userID := uuid.NewUUID()
	membership, err := accessChecker.GetMembership(ctx, ws.ID(), userID)
	require.NoError(t, err)
	assert.Nil(t, membership)

	// Add user as member
	_, err = memberService.AddMember(ctx, ws.ID(), userID, workspace.RoleMember)
	require.NoError(t, err)

	// Now user is a member
	membership, err = accessChecker.GetMembership(ctx, ws.ID(), userID)
	require.NoError(t, err)
	require.NotNil(t, membership)
	assert.Equal(t, userID, membership.UserID)
	assert.Equal(t, "member", membership.Role)

	// Update to admin
	_, err = memberService.UpdateMemberRole(ctx, ws.ID(), userID, workspace.RoleAdmin)
	require.NoError(t, err)

	// Membership reflects new role
	membership, err = accessChecker.GetMembership(ctx, ws.ID(), userID)
	require.NoError(t, err)
	require.NotNil(t, membership)
	assert.Equal(t, "admin", membership.Role)

	// Remove member
	err = memberService.RemoveMember(ctx, ws.ID(), userID)
	require.NoError(t, err)

	// User is no longer a member
	membership, err = accessChecker.GetMembership(ctx, ws.ID(), userID)
	require.NoError(t, err)
	assert.Nil(t, membership)
}
