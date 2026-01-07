package workspace_test

import (
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorkspace(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		name := "Test Workspace"
		keycloakGroupID := "keycloak-group-123"
		createdBy := uuid.NewUUID()

		workspace, err := workspace.NewWorkspace(name, "", keycloakGroupID, createdBy)

		require.NoError(t, err)
		assert.False(t, workspace.ID().IsZero())
		assert.Equal(t, name, workspace.Name())
		assert.Equal(t, keycloakGroupID, workspace.KeycloakGroupID())
		assert.Equal(t, createdBy, workspace.CreatedBy())
		assert.False(t, workspace.CreatedAt().IsZero())
		assert.False(t, workspace.UpdatedAt().IsZero())
		assert.Empty(t, workspace.Invites())
	})

	t.Run("empty name", func(t *testing.T) {
		_, err := workspace.NewWorkspace("", "", "keycloak-group-123", uuid.NewUUID())
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("empty keycloak group ID", func(t *testing.T) {
		_, err := workspace.NewWorkspace("Test Workspace", "", "", uuid.NewUUID())
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("empty created by", func(t *testing.T) {
		_, err := workspace.NewWorkspace("Test Workspace", "", "keycloak-group-123", "")
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})
}

func TestWorkspace_UpdateName(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		workspace, _ := workspace.NewWorkspace("Old Name", "", "keycloak-group-123", uuid.NewUUID())
		oldUpdatedAt := workspace.UpdatedAt()

		time.Sleep(1 * time.Millisecond)
		err := workspace.UpdateName("New Name")

		require.NoError(t, err)
		assert.Equal(t, "New Name", workspace.Name())
		assert.True(t, workspace.UpdatedAt().After(oldUpdatedAt))
	})

	t.Run("empty name", func(t *testing.T) {
		workspace, _ := workspace.NewWorkspace("Old Name", "", "keycloak-group-123", uuid.NewUUID())
		err := workspace.UpdateName("")
		require.ErrorIs(t, err, errs.ErrInvalidInput)
		assert.Equal(t, "Old Name", workspace.Name())
	})
}

func TestWorkspace_CreateInvite(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		workspace, _ := workspace.NewWorkspace("Test Workspace", "", "keycloak-group-123", uuid.NewUUID())
		createdBy := uuid.NewUUID()
		expiresAt := time.Now().Add(24 * time.Hour)
		maxUses := 5

		invite, err := workspace.CreateInvite(createdBy, expiresAt, maxUses)

		require.NoError(t, err)
		assert.NotNil(t, invite)
		assert.Equal(t, workspace.ID(), invite.WorkspaceID())
		assert.Equal(t, createdBy, invite.CreatedBy())
		assert.Equal(t, maxUses, invite.MaxUses())
		assert.Len(t, workspace.Invites(), 1)
	})

	t.Run("empty created by", func(t *testing.T) {
		workspace, _ := workspace.NewWorkspace("Test Workspace", "", "keycloak-group-123", uuid.NewUUID())
		expiresAt := time.Now().Add(24 * time.Hour)

		_, err := workspace.CreateInvite("", expiresAt, 5)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("expired date", func(t *testing.T) {
		workspace, _ := workspace.NewWorkspace("Test Workspace", "", "keycloak-group-123", uuid.NewUUID())
		expiresAt := time.Now().Add(-24 * time.Hour)

		_, err := workspace.CreateInvite(uuid.NewUUID(), expiresAt, 5)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("negative max uses", func(t *testing.T) {
		workspace, _ := workspace.NewWorkspace("Test Workspace", "", "keycloak-group-123", uuid.NewUUID())
		expiresAt := time.Now().Add(24 * time.Hour)

		_, err := workspace.CreateInvite(uuid.NewUUID(), expiresAt, -1)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("unlimited uses (maxUses = 0)", func(t *testing.T) {
		workspace, _ := workspace.NewWorkspace("Test Workspace", "", "keycloak-group-123", uuid.NewUUID())
		expiresAt := time.Now().Add(24 * time.Hour)

		invite, err := workspace.CreateInvite(uuid.NewUUID(), expiresAt, 0)
		require.NoError(t, err)
		assert.Equal(t, 0, invite.MaxUses())
	})
}

func TestWorkspace_FindInviteByToken(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		workspace, _ := workspace.NewWorkspace("Test Workspace", "", "keycloak-group-123", uuid.NewUUID())
		expiresAt := time.Now().Add(24 * time.Hour)
		invite, _ := workspace.CreateInvite(uuid.NewUUID(), expiresAt, 5)

		found, err := workspace.FindInviteByToken(invite.Token())

		require.NoError(t, err)
		assert.Equal(t, invite.ID(), found.ID())
	})

	t.Run("not found", func(t *testing.T) {
		workspace, _ := workspace.NewWorkspace("Test Workspace", "", "keycloak-group-123", uuid.NewUUID())

		_, err := workspace.FindInviteByToken("non-existent-token")
		assert.ErrorIs(t, err, errs.ErrNotFound)
	})
}

func TestNewInvite(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		createdBy := uuid.NewUUID()
		expiresAt := time.Now().Add(24 * time.Hour)
		maxUses := 5

		invite, err := workspace.NewInvite(workspaceID, createdBy, expiresAt, maxUses)

		require.NoError(t, err)
		assert.NotEmpty(t, invite.ID())
		assert.Equal(t, workspaceID, invite.WorkspaceID())
		assert.NotEmpty(t, invite.Token())
		assert.Equal(t, createdBy, invite.CreatedBy())
		assert.Equal(t, maxUses, invite.MaxUses())
		assert.Equal(t, 0, invite.UsedCount())
		assert.False(t, invite.IsRevoked())
	})

	t.Run("empty workspace ID", func(t *testing.T) {
		_, err := workspace.NewInvite("", uuid.NewUUID(), time.Now().Add(24*time.Hour), 5)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("empty created by", func(t *testing.T) {
		_, err := workspace.NewInvite(uuid.NewUUID(), "", time.Now().Add(24*time.Hour), 5)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("expired date", func(t *testing.T) {
		_, err := workspace.NewInvite(uuid.NewUUID(), uuid.NewUUID(), time.Now().Add(-24*time.Hour), 5)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("negative max uses", func(t *testing.T) {
		_, err := workspace.NewInvite(uuid.NewUUID(), uuid.NewUUID(), time.Now().Add(24*time.Hour), -1)
		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})
}

func TestInvite_Use(t *testing.T) {
	t.Run("successful use", func(t *testing.T) {
		invite, _ := workspace.NewInvite(uuid.NewUUID(), uuid.NewUUID(), time.Now().Add(24*time.Hour), 3)

		err := invite.Use()
		require.NoError(t, err)
		assert.Equal(t, 1, invite.UsedCount())

		err = invite.Use()
		require.NoError(t, err)
		assert.Equal(t, 2, invite.UsedCount())
	})

	t.Run("revoked invite", func(t *testing.T) {
		invite, _ := workspace.NewInvite(uuid.NewUUID(), uuid.NewUUID(), time.Now().Add(24*time.Hour), 3)
		invite.Revoke()

		err := invite.Use()
		require.ErrorIs(t, err, errs.ErrInvalidState)
		assert.Equal(t, 0, invite.UsedCount())
	})

	t.Run("expired invite", func(t *testing.T) {
		invite, _ := workspace.NewInvite(uuid.NewUUID(), uuid.NewUUID(), time.Now().Add(1*time.Millisecond), 3)
		time.Sleep(2 * time.Millisecond)

		err := invite.Use()
		assert.ErrorIs(t, err, errs.ErrInvalidState)
	})

	t.Run("max uses reached", func(t *testing.T) {
		invite, _ := workspace.NewInvite(uuid.NewUUID(), uuid.NewUUID(), time.Now().Add(24*time.Hour), 2)
		invite.Use()
		invite.Use()

		err := invite.Use()
		require.ErrorIs(t, err, errs.ErrInvalidState)
		assert.Equal(t, 2, invite.UsedCount())
	})

	t.Run("unlimited uses", func(t *testing.T) {
		invite, _ := workspace.NewInvite(uuid.NewUUID(), uuid.NewUUID(), time.Now().Add(24*time.Hour), 0)

		for range 100 {
			err := invite.Use()
			require.NoError(t, err)
		}
		assert.Equal(t, 100, invite.UsedCount())
	})
}

func TestInvite_Revoke(t *testing.T) {
	t.Run("successful revoke", func(t *testing.T) {
		invite, _ := workspace.NewInvite(uuid.NewUUID(), uuid.NewUUID(), time.Now().Add(24*time.Hour), 3)

		err := invite.Revoke()
		require.NoError(t, err)
		assert.True(t, invite.IsRevoked())
	})

	t.Run("already revoked", func(t *testing.T) {
		invite, _ := workspace.NewInvite(uuid.NewUUID(), uuid.NewUUID(), time.Now().Add(24*time.Hour), 3)
		invite.Revoke()

		err := invite.Revoke()
		assert.ErrorIs(t, err, errs.ErrInvalidState)
	})
}

func TestInvite_IsValid(t *testing.T) {
	t.Run("valid invite", func(t *testing.T) {
		invite, _ := workspace.NewInvite(uuid.NewUUID(), uuid.NewUUID(), time.Now().Add(24*time.Hour), 3)
		assert.True(t, invite.IsValid())
	})

	t.Run("revoked invite", func(t *testing.T) {
		invite, _ := workspace.NewInvite(uuid.NewUUID(), uuid.NewUUID(), time.Now().Add(24*time.Hour), 3)
		invite.Revoke()
		assert.False(t, invite.IsValid())
	})

	t.Run("expired invite", func(t *testing.T) {
		invite, _ := workspace.NewInvite(uuid.NewUUID(), uuid.NewUUID(), time.Now().Add(1*time.Millisecond), 3)
		time.Sleep(2 * time.Millisecond)
		assert.False(t, invite.IsValid())
	})

	t.Run("max uses reached", func(t *testing.T) {
		invite, _ := workspace.NewInvite(uuid.NewUUID(), uuid.NewUUID(), time.Now().Add(24*time.Hour), 2)
		invite.Use()
		assert.True(t, invite.IsValid())
		invite.Use()
		assert.False(t, invite.IsValid())
	})

	t.Run("unlimited uses always valid", func(t *testing.T) {
		invite, _ := workspace.NewInvite(uuid.NewUUID(), uuid.NewUUID(), time.Now().Add(24*time.Hour), 0)
		for range 100 {
			invite.Use()
		}
		assert.True(t, invite.IsValid())
	})
}
