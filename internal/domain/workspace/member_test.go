package workspace_test

import (
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
	"github.com/stretchr/testify/assert"
)

func TestNewMember(t *testing.T) {
	t.Run("creates member with correct fields", func(t *testing.T) {
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		role := workspace.RoleMember

		member := workspace.NewMember(userID, workspaceID, role)

		assert.Equal(t, userID, member.UserID())
		assert.Equal(t, workspaceID, member.WorkspaceID())
		assert.Equal(t, role, member.Role())
		assert.False(t, member.JoinedAt().IsZero())
		assert.WithinDuration(t, time.Now(), member.JoinedAt(), time.Second)
	})

	t.Run("creates owner", func(t *testing.T) {
		member := workspace.NewMember(uuid.NewUUID(), uuid.NewUUID(), workspace.RoleOwner)

		assert.Equal(t, workspace.RoleOwner, member.Role())
		assert.True(t, member.IsOwner())
		assert.True(t, member.IsAdmin())
	})

	t.Run("creates admin", func(t *testing.T) {
		member := workspace.NewMember(uuid.NewUUID(), uuid.NewUUID(), workspace.RoleAdmin)

		assert.Equal(t, workspace.RoleAdmin, member.Role())
		assert.False(t, member.IsOwner())
		assert.True(t, member.IsAdmin())
	})

	t.Run("creates regular member", func(t *testing.T) {
		member := workspace.NewMember(uuid.NewUUID(), uuid.NewUUID(), workspace.RoleMember)

		assert.Equal(t, workspace.RoleMember, member.Role())
		assert.False(t, member.IsOwner())
		assert.False(t, member.IsAdmin())
	})
}

func TestReconstructMember(t *testing.T) {
	t.Run("reconstructs member with all fields", func(t *testing.T) {
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		role := workspace.RoleAdmin
		joinedAt := time.Now().Add(-24 * time.Hour)

		member := workspace.ReconstructMember(userID, workspaceID, role, joinedAt)

		assert.Equal(t, userID, member.UserID())
		assert.Equal(t, workspaceID, member.WorkspaceID())
		assert.Equal(t, role, member.Role())
		assert.Equal(t, joinedAt, member.JoinedAt())
	})
}

func TestRole_IsValid(t *testing.T) {
	t.Run("owner is valid", func(t *testing.T) {
		assert.True(t, workspace.RoleOwner.IsValid())
	})

	t.Run("admin is valid", func(t *testing.T) {
		assert.True(t, workspace.RoleAdmin.IsValid())
	})

	t.Run("member is valid", func(t *testing.T) {
		assert.True(t, workspace.RoleMember.IsValid())
	})

	t.Run("unknown role is invalid", func(t *testing.T) {
		invalidRole := workspace.Role("unknown")
		assert.False(t, invalidRole.IsValid())
	})

	t.Run("empty role is invalid", func(t *testing.T) {
		emptyRole := workspace.Role("")
		assert.False(t, emptyRole.IsValid())
	})
}

func TestRole_String(t *testing.T) {
	t.Run("owner string", func(t *testing.T) {
		assert.Equal(t, "owner", workspace.RoleOwner.String())
	})

	t.Run("admin string", func(t *testing.T) {
		assert.Equal(t, "admin", workspace.RoleAdmin.String())
	})

	t.Run("member string", func(t *testing.T) {
		assert.Equal(t, "member", workspace.RoleMember.String())
	})
}

func TestMember_IsOwner(t *testing.T) {
	t.Run("owner returns true", func(t *testing.T) {
		member := workspace.NewMember(uuid.NewUUID(), uuid.NewUUID(), workspace.RoleOwner)
		assert.True(t, member.IsOwner())
	})

	t.Run("admin returns false", func(t *testing.T) {
		member := workspace.NewMember(uuid.NewUUID(), uuid.NewUUID(), workspace.RoleAdmin)
		assert.False(t, member.IsOwner())
	})

	t.Run("member returns false", func(t *testing.T) {
		member := workspace.NewMember(uuid.NewUUID(), uuid.NewUUID(), workspace.RoleMember)
		assert.False(t, member.IsOwner())
	})
}

func TestMember_IsAdmin(t *testing.T) {
	t.Run("owner is admin", func(t *testing.T) {
		member := workspace.NewMember(uuid.NewUUID(), uuid.NewUUID(), workspace.RoleOwner)
		assert.True(t, member.IsAdmin())
	})

	t.Run("admin is admin", func(t *testing.T) {
		member := workspace.NewMember(uuid.NewUUID(), uuid.NewUUID(), workspace.RoleAdmin)
		assert.True(t, member.IsAdmin())
	})

	t.Run("member is not admin", func(t *testing.T) {
		member := workspace.NewMember(uuid.NewUUID(), uuid.NewUUID(), workspace.RoleMember)
		assert.False(t, member.IsAdmin())
	})
}

func TestMember_CanManageMembers(t *testing.T) {
	t.Run("owner can manage members", func(t *testing.T) {
		member := workspace.NewMember(uuid.NewUUID(), uuid.NewUUID(), workspace.RoleOwner)
		assert.True(t, member.CanManageMembers())
	})

	t.Run("admin can manage members", func(t *testing.T) {
		member := workspace.NewMember(uuid.NewUUID(), uuid.NewUUID(), workspace.RoleAdmin)
		assert.True(t, member.CanManageMembers())
	})

	t.Run("member cannot manage members", func(t *testing.T) {
		member := workspace.NewMember(uuid.NewUUID(), uuid.NewUUID(), workspace.RoleMember)
		assert.False(t, member.CanManageMembers())
	})
}

func TestMember_CanInvite(t *testing.T) {
	t.Run("owner can invite", func(t *testing.T) {
		member := workspace.NewMember(uuid.NewUUID(), uuid.NewUUID(), workspace.RoleOwner)
		assert.True(t, member.CanInvite())
	})

	t.Run("admin can invite", func(t *testing.T) {
		member := workspace.NewMember(uuid.NewUUID(), uuid.NewUUID(), workspace.RoleAdmin)
		assert.True(t, member.CanInvite())
	})

	t.Run("member cannot invite", func(t *testing.T) {
		member := workspace.NewMember(uuid.NewUUID(), uuid.NewUUID(), workspace.RoleMember)
		assert.False(t, member.CanInvite())
	})
}
