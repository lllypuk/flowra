package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
	"github.com/lllypuk/flowra/internal/service"
)

// mockMemberCommandRepository is a mock implementation of MemberCommandRepository
type mockMemberCommandRepository struct {
	addMemberFunc    func(ctx context.Context, member *workspace.Member) error
	removeMemberFunc func(ctx context.Context, workspaceID, userID uuid.UUID) error
	updateMemberFunc func(ctx context.Context, member *workspace.Member) error
}

func (m *mockMemberCommandRepository) AddMember(ctx context.Context, member *workspace.Member) error {
	if m.addMemberFunc != nil {
		return m.addMemberFunc(ctx, member)
	}
	return nil
}

func (m *mockMemberCommandRepository) RemoveMember(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
) error {
	if m.removeMemberFunc != nil {
		return m.removeMemberFunc(ctx, workspaceID, userID)
	}
	return nil
}

func (m *mockMemberCommandRepository) UpdateMember(ctx context.Context, member *workspace.Member) error {
	if m.updateMemberFunc != nil {
		return m.updateMemberFunc(ctx, member)
	}
	return nil
}

// mockMemberQueryRepository is a mock implementation of MemberQueryRepository
type mockMemberQueryRepository struct {
	findByIDFunc     func(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)
	getMemberFunc    func(ctx context.Context, workspaceID, userID uuid.UUID) (*workspace.Member, error)
	listMembersFunc  func(ctx context.Context, workspaceID uuid.UUID, offset, limit int) ([]*workspace.Member, error)
	countMembersFunc func(ctx context.Context, workspaceID uuid.UUID) (int, error)
}

func (m *mockMemberQueryRepository) FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, errs.ErrNotFound
}

func (m *mockMemberQueryRepository) GetMember(
	ctx context.Context,
	workspaceID, userID uuid.UUID,
) (*workspace.Member, error) {
	if m.getMemberFunc != nil {
		return m.getMemberFunc(ctx, workspaceID, userID)
	}
	return nil, errs.ErrNotFound
}

func (m *mockMemberQueryRepository) ListMembers(
	ctx context.Context,
	workspaceID uuid.UUID,
	offset, limit int,
) ([]*workspace.Member, error) {
	if m.listMembersFunc != nil {
		return m.listMembersFunc(ctx, workspaceID, offset, limit)
	}
	return []*workspace.Member{}, nil
}

func (m *mockMemberQueryRepository) CountMembers(ctx context.Context, workspaceID uuid.UUID) (int, error) {
	if m.countMembersFunc != nil {
		return m.countMembersFunc(ctx, workspaceID)
	}
	return 0, nil
}

// createMemberTestWorkspace creates a test workspace for member service testing
func createMemberTestWorkspace(ownerID uuid.UUID, name string) *workspace.Workspace {
	ws, _ := workspace.NewWorkspace(name, "", "keycloak-group-id", ownerID)
	return ws
}

func TestMemberService_AddMember(t *testing.T) {
	t.Run("successfully add member", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()
		ownerID := uuid.NewUUID()

		ws := createMemberTestWorkspace(ownerID, "Test Workspace")

		queryRepo := &mockMemberQueryRepository{
			findByIDFunc: func(_ context.Context, id uuid.UUID) (*workspace.Workspace, error) {
				if id == workspaceID {
					return ws, nil
				}
				return nil, errs.ErrNotFound
			},
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				return nil, errs.ErrNotFound
			},
		}

		var savedMember *workspace.Member
		commandRepo := &mockMemberCommandRepository{
			addMemberFunc: func(_ context.Context, member *workspace.Member) error {
				savedMember = member
				return nil
			},
		}

		svc := service.NewMemberService(commandRepo, queryRepo)

		member, err := svc.AddMember(context.Background(), workspaceID, userID, workspace.RoleMember)

		require.NoError(t, err)
		require.NotNil(t, member)
		assert.Equal(t, userID, member.UserID())
		assert.Equal(t, workspaceID, member.WorkspaceID())
		assert.Equal(t, workspace.RoleMember, member.Role())
		require.NotNil(t, savedMember)
		assert.Equal(t, userID, savedMember.UserID())
	})

	t.Run("workspace not found", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		queryRepo := &mockMemberQueryRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (*workspace.Workspace, error) {
				return nil, errs.ErrNotFound
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		member, err := svc.AddMember(context.Background(), workspaceID, userID, workspace.RoleMember)

		require.Error(t, err)
		require.ErrorIs(t, err, errs.ErrNotFound)
		assert.Nil(t, member)
	})

	t.Run("workspace exists but nil returned", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		queryRepo := &mockMemberQueryRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (*workspace.Workspace, error) {
				//nolint:nilnil // Intentionally testing nil, nil case
				return nil, nil
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		member, err := svc.AddMember(context.Background(), workspaceID, userID, workspace.RoleMember)

		require.Error(t, err)
		require.ErrorIs(t, err, errs.ErrNotFound)
		assert.Nil(t, member)
	})

	t.Run("user already member", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()
		ownerID := uuid.NewUUID()

		ws := createMemberTestWorkspace(ownerID, "Test Workspace")
		existingMember := workspace.NewMember(userID, workspaceID, workspace.RoleMember)

		queryRepo := &mockMemberQueryRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (*workspace.Workspace, error) {
				return ws, nil
			},
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				return &existingMember, nil
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		member, err := svc.AddMember(context.Background(), workspaceID, userID, workspace.RoleMember)

		require.Error(t, err)
		require.ErrorIs(t, err, errs.ErrAlreadyExists)
		assert.Nil(t, member)
	})

	t.Run("add member as admin", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()
		ownerID := uuid.NewUUID()

		ws := createMemberTestWorkspace(ownerID, "Test Workspace")

		queryRepo := &mockMemberQueryRepository{
			findByIDFunc: func(_ context.Context, _ uuid.UUID) (*workspace.Workspace, error) {
				return ws, nil
			},
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				return nil, errs.ErrNotFound
			},
		}

		commandRepo := &mockMemberCommandRepository{
			addMemberFunc: func(_ context.Context, _ *workspace.Member) error {
				return nil
			},
		}

		svc := service.NewMemberService(commandRepo, queryRepo)

		member, err := svc.AddMember(context.Background(), workspaceID, userID, workspace.RoleAdmin)

		require.NoError(t, err)
		require.NotNil(t, member)
		assert.Equal(t, workspace.RoleAdmin, member.Role())
	})
}

func TestMemberService_RemoveMember(t *testing.T) {
	t.Run("successfully remove member", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		existingMember := workspace.NewMember(userID, workspaceID, workspace.RoleMember)

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				return &existingMember, nil
			},
		}

		var removedWorkspaceID, removedUserID uuid.UUID
		commandRepo := &mockMemberCommandRepository{
			removeMemberFunc: func(_ context.Context, wsID, uID uuid.UUID) error {
				removedWorkspaceID = wsID
				removedUserID = uID
				return nil
			},
		}

		svc := service.NewMemberService(commandRepo, queryRepo)

		err := svc.RemoveMember(context.Background(), workspaceID, userID)

		require.NoError(t, err)
		assert.Equal(t, workspaceID, removedWorkspaceID)
		assert.Equal(t, userID, removedUserID)
	})

	t.Run("member not found", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				return nil, errs.ErrNotFound
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		err := svc.RemoveMember(context.Background(), workspaceID, userID)

		require.Error(t, err)
		require.ErrorIs(t, err, errs.ErrNotFound)
	})

	t.Run("member exists but nil returned", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				//nolint:nilnil // Intentionally testing nil, nil case
				return nil, nil
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		err := svc.RemoveMember(context.Background(), workspaceID, userID)

		require.Error(t, err)
		require.ErrorIs(t, err, errs.ErrNotFound)
	})

	t.Run("try to remove owner", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		ownerID := uuid.NewUUID()

		ownerMember := workspace.NewMember(ownerID, workspaceID, workspace.RoleOwner)

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				return &ownerMember, nil
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		err := svc.RemoveMember(context.Background(), workspaceID, ownerID)

		require.Error(t, err)
		require.ErrorIs(t, err, errs.ErrForbidden)
	})

	t.Run("successfully remove admin", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		adminID := uuid.NewUUID()

		adminMember := workspace.NewMember(adminID, workspaceID, workspace.RoleAdmin)

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				return &adminMember, nil
			},
		}

		commandRepo := &mockMemberCommandRepository{
			removeMemberFunc: func(_ context.Context, _, _ uuid.UUID) error {
				return nil
			},
		}

		svc := service.NewMemberService(commandRepo, queryRepo)

		err := svc.RemoveMember(context.Background(), workspaceID, adminID)

		require.NoError(t, err)
	})
}

func TestMemberService_UpdateMemberRole(t *testing.T) {
	t.Run("successfully update role", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		existingMember := workspace.NewMember(userID, workspaceID, workspace.RoleMember)

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				return &existingMember, nil
			},
		}

		var updatedMember *workspace.Member
		commandRepo := &mockMemberCommandRepository{
			updateMemberFunc: func(_ context.Context, member *workspace.Member) error {
				updatedMember = member
				return nil
			},
		}

		svc := service.NewMemberService(commandRepo, queryRepo)

		member, err := svc.UpdateMemberRole(context.Background(), workspaceID, userID, workspace.RoleAdmin)

		require.NoError(t, err)
		require.NotNil(t, member)
		assert.Equal(t, workspace.RoleAdmin, member.Role())
		require.NotNil(t, updatedMember)
		assert.Equal(t, workspace.RoleAdmin, updatedMember.Role())
	})

	t.Run("member not found", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				return nil, errs.ErrNotFound
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		member, err := svc.UpdateMemberRole(context.Background(), workspaceID, userID, workspace.RoleAdmin)

		require.Error(t, err)
		require.ErrorIs(t, err, errs.ErrNotFound)
		assert.Nil(t, member)
	})

	t.Run("member exists but nil returned", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				//nolint:nilnil // Intentionally testing nil, nil case
				return nil, nil
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		member, err := svc.UpdateMemberRole(context.Background(), workspaceID, userID, workspace.RoleAdmin)

		require.Error(t, err)
		require.ErrorIs(t, err, errs.ErrNotFound)
		assert.Nil(t, member)
	})

	t.Run("try to update owner role", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		ownerID := uuid.NewUUID()

		ownerMember := workspace.NewMember(ownerID, workspaceID, workspace.RoleOwner)

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				return &ownerMember, nil
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		member, err := svc.UpdateMemberRole(context.Background(), workspaceID, ownerID, workspace.RoleAdmin)

		require.Error(t, err)
		require.ErrorIs(t, err, errs.ErrForbidden)
		assert.Nil(t, member)
	})

	t.Run("try to set role to owner", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		existingMember := workspace.NewMember(userID, workspaceID, workspace.RoleMember)

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				return &existingMember, nil
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		member, err := svc.UpdateMemberRole(context.Background(), workspaceID, userID, workspace.RoleOwner)

		require.Error(t, err)
		require.ErrorIs(t, err, errs.ErrForbidden)
		assert.Nil(t, member)
	})

	t.Run("demote admin to member", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		adminID := uuid.NewUUID()

		adminMember := workspace.NewMember(adminID, workspaceID, workspace.RoleAdmin)

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				return &adminMember, nil
			},
		}

		commandRepo := &mockMemberCommandRepository{
			updateMemberFunc: func(_ context.Context, _ *workspace.Member) error {
				return nil
			},
		}

		svc := service.NewMemberService(commandRepo, queryRepo)

		member, err := svc.UpdateMemberRole(context.Background(), workspaceID, adminID, workspace.RoleMember)

		require.NoError(t, err)
		require.NotNil(t, member)
		assert.Equal(t, workspace.RoleMember, member.Role())
	})
}

func TestMemberService_GetMember(t *testing.T) {
	t.Run("member found", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		existingMember := workspace.NewMember(userID, workspaceID, workspace.RoleMember)

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, wsID, uID uuid.UUID) (*workspace.Member, error) {
				if wsID == workspaceID && uID == userID {
					return &existingMember, nil
				}
				return nil, errs.ErrNotFound
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		member, err := svc.GetMember(context.Background(), workspaceID, userID)

		require.NoError(t, err)
		require.NotNil(t, member)
		assert.Equal(t, userID, member.UserID())
		assert.Equal(t, workspaceID, member.WorkspaceID())
		assert.Equal(t, workspace.RoleMember, member.Role())
	})

	t.Run("member not found", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				return nil, errs.ErrNotFound
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		member, err := svc.GetMember(context.Background(), workspaceID, userID)

		require.Error(t, err)
		require.ErrorIs(t, err, errs.ErrNotFound)
		assert.Nil(t, member)
	})
}

func TestMemberService_ListMembers(t *testing.T) {
	t.Run("list members with pagination", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		user1ID := uuid.NewUUID()
		user2ID := uuid.NewUUID()

		member1 := workspace.NewMember(user1ID, workspaceID, workspace.RoleOwner)
		member2 := workspace.NewMember(user2ID, workspaceID, workspace.RoleMember)

		queryRepo := &mockMemberQueryRepository{
			listMembersFunc: func(
				_ context.Context,
				wsID uuid.UUID,
				_, _ int,
			) ([]*workspace.Member, error) {
				if wsID == workspaceID {
					return []*workspace.Member{&member1, &member2}, nil
				}
				return []*workspace.Member{}, nil
			},
			countMembersFunc: func(_ context.Context, wsID uuid.UUID) (int, error) {
				if wsID == workspaceID {
					return 5, nil // Total count is 5, but only 2 returned due to pagination
				}
				return 0, nil
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		members, total, err := svc.ListMembers(context.Background(), workspaceID, 0, 2)

		require.NoError(t, err)
		assert.Len(t, members, 2)
		assert.Equal(t, 5, total)
		assert.Equal(t, user1ID, members[0].UserID())
		assert.Equal(t, user2ID, members[1].UserID())
	})

	t.Run("empty list", func(t *testing.T) {
		workspaceID := uuid.NewUUID()

		queryRepo := &mockMemberQueryRepository{
			listMembersFunc: func(_ context.Context, _ uuid.UUID, _, _ int) ([]*workspace.Member, error) {
				return []*workspace.Member{}, nil
			},
			countMembersFunc: func(_ context.Context, _ uuid.UUID) (int, error) {
				return 0, nil
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		members, total, err := svc.ListMembers(context.Background(), workspaceID, 0, 10)

		require.NoError(t, err)
		assert.Empty(t, members)
		assert.Equal(t, 0, total)
	})

	t.Run("list members error", func(t *testing.T) {
		workspaceID := uuid.NewUUID()

		queryRepo := &mockMemberQueryRepository{
			listMembersFunc: func(_ context.Context, _ uuid.UUID, _, _ int) ([]*workspace.Member, error) {
				return nil, errs.ErrNotFound
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		members, total, err := svc.ListMembers(context.Background(), workspaceID, 0, 10)

		require.Error(t, err)
		assert.Nil(t, members)
		assert.Equal(t, 0, total)
	})

	t.Run("count members error", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		user1ID := uuid.NewUUID()

		member1 := workspace.NewMember(user1ID, workspaceID, workspace.RoleOwner)

		queryRepo := &mockMemberQueryRepository{
			listMembersFunc: func(_ context.Context, _ uuid.UUID, _, _ int) ([]*workspace.Member, error) {
				return []*workspace.Member{&member1}, nil
			},
			countMembersFunc: func(_ context.Context, _ uuid.UUID) (int, error) {
				return 0, errs.ErrNotFound
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		members, total, err := svc.ListMembers(context.Background(), workspaceID, 0, 10)

		require.Error(t, err)
		assert.Nil(t, members)
		assert.Equal(t, 0, total)
	})
}

func TestMemberService_IsOwner(t *testing.T) {
	t.Run("user is owner", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		ownerID := uuid.NewUUID()

		ownerMember := workspace.NewMember(ownerID, workspaceID, workspace.RoleOwner)

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, wsID, uID uuid.UUID) (*workspace.Member, error) {
				if wsID == workspaceID && uID == ownerID {
					return &ownerMember, nil
				}
				return nil, errs.ErrNotFound
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		isOwner, err := svc.IsOwner(context.Background(), workspaceID, ownerID)

		require.NoError(t, err)
		assert.True(t, isOwner)
	})

	t.Run("user is member but not owner", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		memberID := uuid.NewUUID()

		regularMember := workspace.NewMember(memberID, workspaceID, workspace.RoleMember)

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				return &regularMember, nil
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		isOwner, err := svc.IsOwner(context.Background(), workspaceID, memberID)

		require.NoError(t, err)
		assert.False(t, isOwner)
	})

	t.Run("user is admin but not owner", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		adminID := uuid.NewUUID()

		adminMember := workspace.NewMember(adminID, workspaceID, workspace.RoleAdmin)

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				return &adminMember, nil
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		isOwner, err := svc.IsOwner(context.Background(), workspaceID, adminID)

		require.NoError(t, err)
		assert.False(t, isOwner)
	})

	t.Run("user is not member", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				return nil, errs.ErrNotFound
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		isOwner, err := svc.IsOwner(context.Background(), workspaceID, userID)

		require.NoError(t, err)
		assert.False(t, isOwner)
	})

	t.Run("member exists but nil returned", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				//nolint:nilnil // Intentionally testing nil, nil case
				return nil, nil
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		isOwner, err := svc.IsOwner(context.Background(), workspaceID, userID)

		require.NoError(t, err)
		assert.False(t, isOwner)
	})

	t.Run("repository error", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		userID := uuid.NewUUID()

		queryRepo := &mockMemberQueryRepository{
			getMemberFunc: func(_ context.Context, _, _ uuid.UUID) (*workspace.Member, error) {
				return nil, errs.ErrInvalidInput
			},
		}

		commandRepo := &mockMemberCommandRepository{}

		svc := service.NewMemberService(commandRepo, queryRepo)

		isOwner, err := svc.IsOwner(context.Background(), workspaceID, userID)

		require.Error(t, err)
		require.ErrorIs(t, err, errs.ErrInvalidInput)
		assert.False(t, isOwner)
	})
}
