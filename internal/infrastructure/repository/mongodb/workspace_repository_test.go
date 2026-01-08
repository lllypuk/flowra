package mongodb_test

import (
	"context"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
	"github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	"github.com/lllypuk/flowra/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestWorkspaceRepository creates test репозиторий workspace
func setupTestWorkspaceRepository(t *testing.T) *mongodb.MongoWorkspaceRepository {
	t.Helper()

	db := testutil.SetupTestMongoDB(t)
	coll := db.Collection("workspaces")
	membersColl := db.Collection("workspace_members")

	return mongodb.NewMongoWorkspaceRepository(coll, membersColl)
}

// createTestWorkspace creates test workspace с uniqueыми данными
func createTestWorkspace(t *testing.T, suffix string) *workspace.Workspace {
	t.Helper()

	createdBy := uuid.NewUUID()
	ws, err := workspace.NewWorkspace(
		"Test Workspace "+suffix,
		"Test description "+suffix,
		"keycloak-group-"+suffix,
		createdBy,
	)
	require.NoError(t, err)
	return ws
}

// TestMongoWorkspaceRepository_Save_And_FindByID checks storage and searching workspace по ID
func TestMongoWorkspaceRepository_Save_And_FindByID(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create and save workspace
	ws := createTestWorkspace(t, "1")
	err := repo.Save(ctx, ws)
	require.NoError(t, err)

	// Find by ID
	loaded, err := repo.FindByID(ctx, ws.ID())
	require.NoError(t, err)
	require.NotNil(t, loaded)

	// Verify all fields
	assert.Equal(t, ws.ID(), loaded.ID())
	assert.Equal(t, ws.Name(), loaded.Name())
	assert.Equal(t, ws.KeycloakGroupID(), loaded.KeycloakGroupID())
	assert.Equal(t, ws.CreatedBy(), loaded.CreatedBy())
}

// TestMongoWorkspaceRepository_FindByID_NotFound checks search неexistingего workspace
func TestMongoWorkspaceRepository_FindByID_NotFound(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Try to find non-existent workspace
	_, err := repo.FindByID(ctx, uuid.NewUUID())
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoWorkspaceRepository_FindByKeycloakGroup checks searching workspace по Keycloak group ID
func TestMongoWorkspaceRepository_FindByKeycloakGroup(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create and save workspace
	ws := createTestWorkspace(t, "keycloak")
	err := repo.Save(ctx, ws)
	require.NoError(t, err)

	// Find by Keycloak group ID
	loaded, err := repo.FindByKeycloakGroup(ctx, ws.KeycloakGroupID())
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, ws.ID(), loaded.ID())
	assert.Equal(t, ws.KeycloakGroupID(), loaded.KeycloakGroupID())

	// Find by non-existent Keycloak group ID
	_, err = repo.FindByKeycloakGroup(ctx, "non-existent-group")
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoWorkspaceRepository_List checks retrieval list workspaces
func TestMongoWorkspaceRepository_List(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create and save multiple workspaces
	for i := range 5 {
		ws := createTestWorkspace(t, string(rune('a'+i)))
		err := repo.Save(ctx, ws)
		require.NoError(t, err)
	}

	// List all workspaces
	workspaces, err := repo.List(ctx, 0, 10)
	require.NoError(t, err)
	assert.Len(t, workspaces, 5)

	// List with pagination
	workspaces, err = repo.List(ctx, 0, 2)
	require.NoError(t, err)
	assert.Len(t, workspaces, 2)

	// List with offset
	workspaces, err = repo.List(ctx, 2, 10)
	require.NoError(t, err)
	assert.Len(t, workspaces, 3)

	// List with offset beyond count
	workspaces, err = repo.List(ctx, 10, 10)
	require.NoError(t, err)
	assert.Empty(t, workspaces)
}

// TestMongoWorkspaceRepository_Count checks подсчет workspaces
func TestMongoWorkspaceRepository_Count(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Initial count should be 0
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Add workspaces and verify count
	for i := range 3 {
		ws := createTestWorkspace(t, string(rune('x'+i)))
		saveErr := repo.Save(ctx, ws)
		require.NoError(t, saveErr)
	}

	count, err = repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

// TestMongoWorkspaceRepository_Delete checks deletion workspace
func TestMongoWorkspaceRepository_Delete(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create and save workspace
	ws := createTestWorkspace(t, "delete")
	err := repo.Save(ctx, ws)
	require.NoError(t, err)

	// Verify workspace exists
	_, err = repo.FindByID(ctx, ws.ID())
	require.NoError(t, err)

	// Delete workspace
	err = repo.Delete(ctx, ws.ID())
	require.NoError(t, err)

	// Verify workspace no longer exists
	_, err = repo.FindByID(ctx, ws.ID())
	require.ErrorIs(t, err, errs.ErrNotFound)

	// Delete non-existent workspace should return error
	err = repo.Delete(ctx, uuid.NewUUID())
	require.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoWorkspaceRepository_Update checks update workspace
func TestMongoWorkspaceRepository_Update(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create and save workspace
	ws := createTestWorkspace(t, "update")
	err := repo.Save(ctx, ws)
	require.NoError(t, err)

	// Update workspace name
	err = ws.UpdateName("Updated Workspace Name")
	require.NoError(t, err)

	// Save updated workspace
	err = repo.Save(ctx, ws)
	require.NoError(t, err)

	// Load and verify
	loaded, err := repo.FindByID(ctx, ws.ID())
	require.NoError(t, err)
	assert.Equal(t, "Updated Workspace Name", loaded.Name())
}

// TestMongoWorkspaceRepository_FindInviteByToken checks search приглашения по токену
func TestMongoWorkspaceRepository_FindInviteByToken(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create workspace with invite
	ws := createTestWorkspace(t, "invite")
	expiresAt := time.Now().Add(24 * time.Hour)
	invite, err := ws.CreateInvite(uuid.NewUUID(), expiresAt, 5)
	require.NoError(t, err)

	// Save workspace
	err = repo.Save(ctx, ws)
	require.NoError(t, err)

	// Find invite by token
	found, err := repo.FindInviteByToken(ctx, invite.Token())
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, invite.Token(), found.Token())
	assert.Equal(t, invite.WorkspaceID(), found.WorkspaceID())
	assert.Equal(t, invite.MaxUses(), found.MaxUses())

	// Find non-existent invite
	_, err = repo.FindInviteByToken(ctx, "non-existent-token")
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoWorkspaceRepository_MultiInvites checks workspace с несколькими приглашениями
func TestMongoWorkspaceRepository_MultiInvites(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create workspace with multiple invites
	ws := createTestWorkspace(t, "multi-invite")
	expiresAt := time.Now().Add(24 * time.Hour)

	invite1, err := ws.CreateInvite(uuid.NewUUID(), expiresAt, 5)
	require.NoError(t, err)

	invite2, err := ws.CreateInvite(uuid.NewUUID(), expiresAt, 10)
	require.NoError(t, err)

	invite3, err := ws.CreateInvite(uuid.NewUUID(), expiresAt, 0) // unlimited
	require.NoError(t, err)

	// Save workspace
	err = repo.Save(ctx, ws)
	require.NoError(t, err)

	// Load and verify invites
	loaded, err := repo.FindByID(ctx, ws.ID())
	require.NoError(t, err)
	assert.Len(t, loaded.Invites(), 3)

	// Find each invite by token
	found1, err := repo.FindInviteByToken(ctx, invite1.Token())
	require.NoError(t, err)
	assert.Equal(t, 5, found1.MaxUses())

	found2, err := repo.FindInviteByToken(ctx, invite2.Token())
	require.NoError(t, err)
	assert.Equal(t, 10, found2.MaxUses())

	found3, err := repo.FindInviteByToken(ctx, invite3.Token())
	require.NoError(t, err)
	assert.Equal(t, 0, found3.MaxUses())
}

// TestMongoWorkspaceRepository_InputValidation checks validацию входных данных
func TestMongoWorkspaceRepository_InputValidation(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	t.Run("FindByID with zero UUID", func(t *testing.T) {
		_, err := repo.FindByID(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("FindByKeycloakGroup with empty group ID", func(t *testing.T) {
		_, err := repo.FindByKeycloakGroup(ctx, "")
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("Save with nil workspace", func(t *testing.T) {
		err := repo.Save(ctx, nil)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("Delete with zero UUID", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("FindInviteByToken with empty token", func(t *testing.T) {
		_, err := repo.FindInviteByToken(ctx, "")
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})
}

// TestMongoWorkspaceRepository_DocToWorkspace checks converting документа in Workspace
func TestMongoWorkspaceRepository_DocToWorkspace(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create workspace with invite
	ws := createTestWorkspace(t, "doctoworkspace")
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err := ws.CreateInvite(uuid.NewUUID(), expiresAt, 5)
	require.NoError(t, err)

	// Save workspace
	err = repo.Save(ctx, ws)
	require.NoError(t, err)

	// Load workspace
	loaded, err := repo.FindByID(ctx, ws.ID())
	require.NoError(t, err)

	// Verify all fields are correctly restored
	assert.Equal(t, ws.ID(), loaded.ID())
	assert.Equal(t, ws.Name(), loaded.Name())
	assert.Equal(t, ws.KeycloakGroupID(), loaded.KeycloakGroupID())
	assert.Equal(t, ws.CreatedBy(), loaded.CreatedBy())
	assert.Len(t, loaded.Invites(), 1)

	// Times should be close (allow for millisecond precision loss due to MongoDB serialization)
	assert.WithinDuration(t, ws.CreatedAt(), loaded.CreatedAt(), time.Millisecond)
	assert.WithinDuration(t, ws.UpdatedAt(), loaded.UpdatedAt(), time.Millisecond)
}

// ============== Member Tests ==============

// TestMongoWorkspaceRepository_AddMember checks adding члена in workspace
func TestMongoWorkspaceRepository_AddMember(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create workspace
	ws := createTestWorkspace(t, "addmember")
	err := repo.Save(ctx, ws)
	require.NoError(t, err)

	// Create and add member
	member := workspace.NewMember(uuid.NewUUID(), ws.ID(), workspace.RoleMember)
	err = repo.AddMember(ctx, &member)
	require.NoError(t, err)

	// Verify member exists
	loaded, err := repo.GetMember(ctx, ws.ID(), member.UserID())
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, member.UserID(), loaded.UserID())
	assert.Equal(t, member.WorkspaceID(), loaded.WorkspaceID())
	assert.Equal(t, member.Role(), loaded.Role())
}

// TestMongoWorkspaceRepository_AddMember_Owner checks adding владельца
func TestMongoWorkspaceRepository_AddMember_Owner(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create workspace
	ws := createTestWorkspace(t, "addowner")
	err := repo.Save(ctx, ws)
	require.NoError(t, err)

	// Add owner
	owner := workspace.NewMember(ws.CreatedBy(), ws.ID(), workspace.RoleOwner)
	err = repo.AddMember(ctx, &owner)
	require.NoError(t, err)

	// Verify owner
	loaded, err := repo.GetMember(ctx, ws.ID(), owner.UserID())
	require.NoError(t, err)
	assert.True(t, loaded.IsOwner())
	assert.True(t, loaded.IsAdmin())
	assert.True(t, loaded.CanManageMembers())
}

// TestMongoWorkspaceRepository_AddMember_Admin checks adding administratorа
func TestMongoWorkspaceRepository_AddMember_Admin(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create workspace
	ws := createTestWorkspace(t, "addadmin")
	err := repo.Save(ctx, ws)
	require.NoError(t, err)

	// Add admin
	admin := workspace.NewMember(uuid.NewUUID(), ws.ID(), workspace.RoleAdmin)
	err = repo.AddMember(ctx, &admin)
	require.NoError(t, err)

	// Verify admin
	loaded, err := repo.GetMember(ctx, ws.ID(), admin.UserID())
	require.NoError(t, err)
	assert.False(t, loaded.IsOwner())
	assert.True(t, loaded.IsAdmin())
	assert.True(t, loaded.CanInvite())
}

// TestMongoWorkspaceRepository_RemoveMember checks deletion члена from workspace
func TestMongoWorkspaceRepository_RemoveMember(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create workspace
	ws := createTestWorkspace(t, "removemember")
	err := repo.Save(ctx, ws)
	require.NoError(t, err)

	// Add member
	member := workspace.NewMember(uuid.NewUUID(), ws.ID(), workspace.RoleMember)
	err = repo.AddMember(ctx, &member)
	require.NoError(t, err)

	// Verify member exists
	_, err = repo.GetMember(ctx, ws.ID(), member.UserID())
	require.NoError(t, err)

	// Remove member
	err = repo.RemoveMember(ctx, ws.ID(), member.UserID())
	require.NoError(t, err)

	// Verify member no longer exists
	_, err = repo.GetMember(ctx, ws.ID(), member.UserID())
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoWorkspaceRepository_RemoveMemberNotFound checks deletion неexistingего члена
func TestMongoWorkspaceRepository_RemoveMemberNotFound(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create workspace
	ws := createTestWorkspace(t, "removemembernotfound")
	err := repo.Save(ctx, ws)
	require.NoError(t, err)

	// Try to remove non-existent member
	err = repo.RemoveMember(ctx, ws.ID(), uuid.NewUUID())
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoWorkspaceRepository_GetMember checks retrieval члена workspace
func TestMongoWorkspaceRepository_GetMember(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create workspace
	ws := createTestWorkspace(t, "getmember")
	err := repo.Save(ctx, ws)
	require.NoError(t, err)

	// Add member
	userID := uuid.NewUUID()
	member := workspace.NewMember(userID, ws.ID(), workspace.RoleAdmin)
	err = repo.AddMember(ctx, &member)
	require.NoError(t, err)

	// Get member
	loaded, err := repo.GetMember(ctx, ws.ID(), userID)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, userID, loaded.UserID())
	assert.Equal(t, ws.ID(), loaded.WorkspaceID())
	assert.Equal(t, workspace.RoleAdmin, loaded.Role())
	assert.WithinDuration(t, member.JoinedAt(), loaded.JoinedAt(), time.Millisecond)
}

// TestMongoWorkspaceRepository_GetMemberNotFound checks retrieval неexistingего члена
func TestMongoWorkspaceRepository_GetMemberNotFound(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create workspace
	ws := createTestWorkspace(t, "getmembernotfound")
	err := repo.Save(ctx, ws)
	require.NoError(t, err)

	// Try to get non-existent member
	_, err = repo.GetMember(ctx, ws.ID(), uuid.NewUUID())
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

// TestMongoWorkspaceRepository_IsMember checks проверку членства
func TestMongoWorkspaceRepository_IsMember(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create workspace
	ws := createTestWorkspace(t, "ismember")
	err := repo.Save(ctx, ws)
	require.NoError(t, err)

	// Add member
	memberUserID := uuid.NewUUID()
	member := workspace.NewMember(memberUserID, ws.ID(), workspace.RoleMember)
	err = repo.AddMember(ctx, &member)
	require.NoError(t, err)

	// Check member exists
	isMember, err := repo.IsMember(ctx, ws.ID(), memberUserID)
	require.NoError(t, err)
	assert.True(t, isMember)

	// Check non-member
	nonMemberID := uuid.NewUUID()
	isMember, err = repo.IsMember(ctx, ws.ID(), nonMemberID)
	require.NoError(t, err)
	assert.False(t, isMember)
}

// TestMongoWorkspaceRepository_ListMembers checks retrieval list членов workspace
func TestMongoWorkspaceRepository_ListMembers(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create workspace
	ws := createTestWorkspace(t, "listmembers")
	err := repo.Save(ctx, ws)
	require.NoError(t, err)

	// Add multiple members
	for i := range 5 {
		var role workspace.Role
		switch i {
		case 0:
			role = workspace.RoleOwner
		case 1:
			role = workspace.RoleAdmin
		default:
			role = workspace.RoleMember
		}
		member := workspace.NewMember(uuid.NewUUID(), ws.ID(), role)
		addErr := repo.AddMember(ctx, &member)
		require.NoError(t, addErr)
	}

	// List all members
	members, err := repo.ListMembers(ctx, ws.ID(), 0, 10)
	require.NoError(t, err)
	assert.Len(t, members, 5)

	// List with pagination
	members, err = repo.ListMembers(ctx, ws.ID(), 0, 2)
	require.NoError(t, err)
	assert.Len(t, members, 2)

	// List with offset
	members, err = repo.ListMembers(ctx, ws.ID(), 2, 10)
	require.NoError(t, err)
	assert.Len(t, members, 3)
}

// TestMongoWorkspaceRepository_CountMembers checks подсчет членов workspace
func TestMongoWorkspaceRepository_CountMembers(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create workspace
	ws := createTestWorkspace(t, "countmembers")
	err := repo.Save(ctx, ws)
	require.NoError(t, err)

	// Initial count should be 0
	count, err := repo.CountMembers(ctx, ws.ID())
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Add members
	for range 4 {
		member := workspace.NewMember(uuid.NewUUID(), ws.ID(), workspace.RoleMember)
		addErr := repo.AddMember(ctx, &member)
		require.NoError(t, addErr)
	}

	// Count should be 4
	count, err = repo.CountMembers(ctx, ws.ID())
	require.NoError(t, err)
	assert.Equal(t, 4, count)
}

// TestMongoWorkspaceRepository_ListByUser checks retrieval workspaces user
func TestMongoWorkspaceRepository_ListByUser(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create user
	userID := uuid.NewUUID()

	// Create workspaces and add user as member
	for i := range 3 {
		ws := createTestWorkspace(t, "listbyuser"+string(rune('a'+i)))
		err := repo.Save(ctx, ws)
		require.NoError(t, err)

		member := workspace.NewMember(userID, ws.ID(), workspace.RoleMember)
		err = repo.AddMember(ctx, &member)
		require.NoError(t, err)
	}

	// Create workspace where user is NOT a member
	wsNotMember := createTestWorkspace(t, "listbyuser_notmember")
	err := repo.Save(ctx, wsNotMember)
	require.NoError(t, err)

	// List workspaces by user
	workspaces, err := repo.ListWorkspacesByUser(ctx, userID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, workspaces, 3)

	// List with pagination
	workspaces, err = repo.ListWorkspacesByUser(ctx, userID, 0, 2)
	require.NoError(t, err)
	assert.Len(t, workspaces, 2)

	// List with offset
	workspaces, err = repo.ListWorkspacesByUser(ctx, userID, 1, 10)
	require.NoError(t, err)
	assert.Len(t, workspaces, 2)
}

// TestMongoWorkspaceRepository_ListByUserEmpty checks случай без workspaces
func TestMongoWorkspaceRepository_ListByUserEmpty(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// User without any workspaces
	userID := uuid.NewUUID()

	workspaces, err := repo.ListWorkspacesByUser(ctx, userID, 0, 10)
	require.NoError(t, err)
	assert.Empty(t, workspaces)
	assert.NotNil(t, workspaces) // Should return empty slice, not nil
}

// TestMongoWorkspaceRepository_CountByUser checks подсчет workspaces user
func TestMongoWorkspaceRepository_CountByUser(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create user
	userID := uuid.NewUUID()

	// Initial count should be 0
	count, err := repo.CountWorkspacesByUser(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Create workspaces and add user as member
	for i := range 5 {
		ws := createTestWorkspace(t, "countbyuser"+string(rune('a'+i)))
		saveErr := repo.Save(ctx, ws)
		require.NoError(t, saveErr)

		member := workspace.NewMember(userID, ws.ID(), workspace.RoleMember)
		addErr := repo.AddMember(ctx, &member)
		require.NoError(t, addErr)
	}

	// Count should be 5
	count, err = repo.CountWorkspacesByUser(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

// TestMongoWorkspaceRepository_MemberValidation checks validацию входных данных for methods членов
func TestMongoWorkspaceRepository_MemberValidation(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	t.Run("GetMember with zero workspaceID", func(t *testing.T) {
		_, err := repo.GetMember(ctx, uuid.UUID(""), uuid.NewUUID())
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("GetMember with zero userID", func(t *testing.T) {
		_, err := repo.GetMember(ctx, uuid.NewUUID(), uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("IsMember with zero workspaceID", func(t *testing.T) {
		_, err := repo.IsMember(ctx, uuid.UUID(""), uuid.NewUUID())
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("IsMember with zero userID", func(t *testing.T) {
		_, err := repo.IsMember(ctx, uuid.NewUUID(), uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("AddMember with nil member", func(t *testing.T) {
		err := repo.AddMember(ctx, nil)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("RemoveMember with zero workspaceID", func(t *testing.T) {
		err := repo.RemoveMember(ctx, uuid.UUID(""), uuid.NewUUID())
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("RemoveMember with zero userID", func(t *testing.T) {
		err := repo.RemoveMember(ctx, uuid.NewUUID(), uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("ListMembers with zero workspaceID", func(t *testing.T) {
		_, err := repo.ListMembers(ctx, uuid.UUID(""), 0, 10)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("CountMembers with zero workspaceID", func(t *testing.T) {
		_, err := repo.CountMembers(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("ListWorkspacesByUser with zero userID", func(t *testing.T) {
		_, err := repo.ListWorkspacesByUser(ctx, uuid.UUID(""), 0, 10)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("CountWorkspacesByUser with zero userID", func(t *testing.T) {
		_, err := repo.CountWorkspacesByUser(ctx, uuid.UUID(""))
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})
}

// TestMongoWorkspaceRepository_DeleteRemovesMembers checks, that deletion workspace удаляет and членов
func TestMongoWorkspaceRepository_DeleteRemovesMembers(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create workspace
	ws := createTestWorkspace(t, "delwithmembers")
	err := repo.Save(ctx, ws)
	require.NoError(t, err)

	// Add members
	memberIDs := make([]uuid.UUID, 3)
	for i := range 3 {
		memberIDs[i] = uuid.NewUUID()
		member := workspace.NewMember(memberIDs[i], ws.ID(), workspace.RoleMember)
		addErr := repo.AddMember(ctx, &member)
		require.NoError(t, addErr)
	}

	// Verify members exist
	count, err := repo.CountMembers(ctx, ws.ID())
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// Delete workspace
	err = repo.Delete(ctx, ws.ID())
	require.NoError(t, err)

	// Verify members are also deleted
	count, err = repo.CountMembers(ctx, ws.ID())
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Verify each member is not found
	for _, memberID := range memberIDs {
		_, err = repo.GetMember(ctx, ws.ID(), memberID)
		assert.ErrorIs(t, err, errs.ErrNotFound)
	}
}

// TestMongoWorkspaceRepository_UpsertMember checks update члена (upsert)
func TestMongoWorkspaceRepository_UpsertMember(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create workspace
	ws := createTestWorkspace(t, "updatemember")
	err := repo.Save(ctx, ws)
	require.NoError(t, err)

	// Add member as regular member
	userID := uuid.NewUUID()
	member := workspace.NewMember(userID, ws.ID(), workspace.RoleMember)
	err = repo.AddMember(ctx, &member)
	require.NoError(t, err)

	// Verify role
	loaded, err := repo.GetMember(ctx, ws.ID(), userID)
	require.NoError(t, err)
	assert.Equal(t, workspace.RoleMember, loaded.Role())

	// Update to admin (using upsert behavior)
	adminMember := workspace.NewMember(userID, ws.ID(), workspace.RoleAdmin)
	err = repo.AddMember(ctx, &adminMember)
	require.NoError(t, err)

	// Verify updated role
	loaded, err = repo.GetMember(ctx, ws.ID(), userID)
	require.NoError(t, err)
	assert.Equal(t, workspace.RoleAdmin, loaded.Role())

	// Count should still be 1
	count, err := repo.CountMembers(ctx, ws.ID())
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// TestMongoWorkspaceRepository_Isolation checks изоляцию членов between workspaces
func TestMongoWorkspaceRepository_Isolation(t *testing.T) {
	repo := setupTestWorkspaceRepository(t)
	ctx := context.Background()

	// Create two workspaces
	ws1 := createTestWorkspace(t, "isolation1")
	err := repo.Save(ctx, ws1)
	require.NoError(t, err)

	ws2 := createTestWorkspace(t, "isolation2")
	err = repo.Save(ctx, ws2)
	require.NoError(t, err)

	// Add user to ws1
	userID := uuid.NewUUID()
	member1 := workspace.NewMember(userID, ws1.ID(), workspace.RoleMember)
	err = repo.AddMember(ctx, &member1)
	require.NoError(t, err)

	// User should be member of ws1
	isMember, err := repo.IsMember(ctx, ws1.ID(), userID)
	require.NoError(t, err)
	assert.True(t, isMember)

	// User should NOT be member of ws2
	isMember, err = repo.IsMember(ctx, ws2.ID(), userID)
	require.NoError(t, err)
	assert.False(t, isMember)

	// Count workspaces by user should be 1
	count, err := repo.CountWorkspacesByUser(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Add user to ws2
	member2 := workspace.NewMember(userID, ws2.ID(), workspace.RoleAdmin)
	err = repo.AddMember(ctx, &member2)
	require.NoError(t, err)

	// Now user should be member of both
	isMember, err = repo.IsMember(ctx, ws1.ID(), userID)
	require.NoError(t, err)
	assert.True(t, isMember)

	isMember, err = repo.IsMember(ctx, ws2.ID(), userID)
	require.NoError(t, err)
	assert.True(t, isMember)

	// Count workspaces by user should be 2
	count, err = repo.CountWorkspacesByUser(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
