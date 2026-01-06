package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/appcore"
	wsapp "github.com/lllypuk/flowra/internal/application/workspace"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
	"github.com/lllypuk/flowra/internal/service"
)

// mockWSCreateUseCase is a mock implementation of CreateWorkspaceUseCase
type mockWSCreateUseCase struct {
	executeFunc func(ctx context.Context, cmd wsapp.CreateWorkspaceCommand) (wsapp.Result, error)
}

func (m *mockWSCreateUseCase) Execute(ctx context.Context, cmd wsapp.CreateWorkspaceCommand) (wsapp.Result, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return wsapp.Result{}, nil
}

// mockWSGetUseCase is a mock implementation of GetWorkspaceUseCase
type mockWSGetUseCase struct {
	executeFunc func(ctx context.Context, query wsapp.GetWorkspaceQuery) (wsapp.Result, error)
}

func (m *mockWSGetUseCase) Execute(ctx context.Context, query wsapp.GetWorkspaceQuery) (wsapp.Result, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, query)
	}
	return wsapp.Result{}, nil
}

// mockWSUpdateUseCase is a mock implementation of UpdateWorkspaceUseCase
type mockWSUpdateUseCase struct {
	executeFunc func(ctx context.Context, cmd wsapp.UpdateWorkspaceCommand) (wsapp.Result, error)
}

func (m *mockWSUpdateUseCase) Execute(ctx context.Context, cmd wsapp.UpdateWorkspaceCommand) (wsapp.Result, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return wsapp.Result{}, nil
}

// mockWSServiceCommandRepo is a mock implementation of WorkspaceServiceCommandRepository
type mockWSServiceCommandRepo struct {
	saveFunc      func(ctx context.Context, ws *workspace.Workspace) error
	deleteFunc    func(ctx context.Context, id uuid.UUID) error
	addMemberFunc func(ctx context.Context, member *workspace.Member) error
}

func (m *mockWSServiceCommandRepo) Save(ctx context.Context, ws *workspace.Workspace) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, ws)
	}
	return nil
}

func (m *mockWSServiceCommandRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockWSServiceCommandRepo) AddMember(ctx context.Context, member *workspace.Member) error {
	if m.addMemberFunc != nil {
		return m.addMemberFunc(ctx, member)
	}
	return nil
}

// mockWSServiceQueryRepo is a mock implementation of WorkspaceServiceQueryRepository
type mockWSServiceQueryRepo struct {
	findByIDFunc              func(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)
	listWorkspacesByUserFunc  func(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*workspace.Workspace, error)
	countWorkspacesByUserFunc func(ctx context.Context, userID uuid.UUID) (int, error)
	countMembersFunc          func(ctx context.Context, workspaceID uuid.UUID) (int, error)
}

func (m *mockWSServiceQueryRepo) FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error) {
	if m.findByIDFunc != nil {
		return m.findByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockWSServiceQueryRepo) ListWorkspacesByUser(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*workspace.Workspace, error) {
	if m.listWorkspacesByUserFunc != nil {
		return m.listWorkspacesByUserFunc(ctx, userID, offset, limit)
	}
	return []*workspace.Workspace{}, nil
}

func (m *mockWSServiceQueryRepo) CountWorkspacesByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	if m.countWorkspacesByUserFunc != nil {
		return m.countWorkspacesByUserFunc(ctx, userID)
	}
	return 0, nil
}

func (m *mockWSServiceQueryRepo) CountMembers(ctx context.Context, workspaceID uuid.UUID) (int, error) {
	if m.countMembersFunc != nil {
		return m.countMembersFunc(ctx, workspaceID)
	}
	return 0, nil
}

// createWSServiceTestWorkspace creates a test workspace for service testing
func createWSServiceTestWorkspace(ownerID uuid.UUID, name string) *workspace.Workspace {
	ws, _ := workspace.NewWorkspace(name, "keycloak-group-id", ownerID)
	return ws
}

func TestWorkspaceService_CreateWorkspace(t *testing.T) {
	t.Run("successfully create workspace", func(t *testing.T) {
		ownerID := uuid.NewUUID()
		expectedWS := createWSServiceTestWorkspace(ownerID, "Test Workspace")

		createUC := &mockWSCreateUseCase{
			executeFunc: func(_ context.Context, cmd wsapp.CreateWorkspaceCommand) (wsapp.Result, error) {
				assert.Equal(t, "Test Workspace", cmd.Name)
				assert.Equal(t, ownerID, cmd.CreatedBy)
				return wsapp.Result{
					Result: appcore.Result[*workspace.Workspace]{Value: expectedWS},
				}, nil
			},
		}

		svc := service.NewWorkspaceService(service.WorkspaceServiceConfig{
			CreateUC:    createUC,
			GetUC:       &mockWSGetUseCase{},
			UpdateUC:    &mockWSUpdateUseCase{},
			CommandRepo: &mockWSServiceCommandRepo{},
			QueryRepo:   &mockWSServiceQueryRepo{},
		})

		ws, err := svc.CreateWorkspace(context.Background(), ownerID, "Test Workspace", "Description")

		require.NoError(t, err)
		require.NotNil(t, ws)
		assert.Equal(t, expectedWS.ID(), ws.ID())
		assert.Equal(t, expectedWS.Name(), ws.Name())
	})

	t.Run("use case returns error", func(t *testing.T) {
		ownerID := uuid.NewUUID()
		expectedErr := errors.New("validation failed")

		createUC := &mockWSCreateUseCase{
			executeFunc: func(_ context.Context, _ wsapp.CreateWorkspaceCommand) (wsapp.Result, error) {
				return wsapp.Result{}, expectedErr
			},
		}

		svc := service.NewWorkspaceService(service.WorkspaceServiceConfig{
			CreateUC:    createUC,
			GetUC:       &mockWSGetUseCase{},
			UpdateUC:    &mockWSUpdateUseCase{},
			CommandRepo: &mockWSServiceCommandRepo{},
			QueryRepo:   &mockWSServiceQueryRepo{},
		})

		ws, err := svc.CreateWorkspace(context.Background(), ownerID, "", "")

		require.Error(t, err)
		assert.Nil(t, ws)
		assert.Equal(t, expectedErr, err)
	})
}

func TestWorkspaceService_GetWorkspace(t *testing.T) {
	t.Run("workspace exists", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		ownerID := uuid.NewUUID()
		expectedWS := createWSServiceTestWorkspace(ownerID, "Test Workspace")

		getUC := &mockWSGetUseCase{
			executeFunc: func(_ context.Context, query wsapp.GetWorkspaceQuery) (wsapp.Result, error) {
				assert.Equal(t, workspaceID, query.WorkspaceID)
				return wsapp.Result{
					Result: appcore.Result[*workspace.Workspace]{Value: expectedWS},
				}, nil
			},
		}

		svc := service.NewWorkspaceService(service.WorkspaceServiceConfig{
			CreateUC:    &mockWSCreateUseCase{},
			GetUC:       getUC,
			UpdateUC:    &mockWSUpdateUseCase{},
			CommandRepo: &mockWSServiceCommandRepo{},
			QueryRepo:   &mockWSServiceQueryRepo{},
		})

		ws, err := svc.GetWorkspace(context.Background(), workspaceID)

		require.NoError(t, err)
		require.NotNil(t, ws)
		assert.Equal(t, expectedWS.ID(), ws.ID())
	})

	t.Run("workspace not found", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		expectedErr := wsapp.ErrWorkspaceNotFound

		getUC := &mockWSGetUseCase{
			executeFunc: func(_ context.Context, _ wsapp.GetWorkspaceQuery) (wsapp.Result, error) {
				return wsapp.Result{}, expectedErr
			},
		}

		svc := service.NewWorkspaceService(service.WorkspaceServiceConfig{
			CreateUC:    &mockWSCreateUseCase{},
			GetUC:       getUC,
			UpdateUC:    &mockWSUpdateUseCase{},
			CommandRepo: &mockWSServiceCommandRepo{},
			QueryRepo:   &mockWSServiceQueryRepo{},
		})

		ws, err := svc.GetWorkspace(context.Background(), workspaceID)

		require.Error(t, err)
		assert.Nil(t, ws)
		assert.Equal(t, expectedErr, err)
	})
}

func TestWorkspaceService_ListUserWorkspaces(t *testing.T) {
	t.Run("user has workspaces", func(t *testing.T) {
		userID := uuid.NewUUID()
		ws1 := createWSServiceTestWorkspace(userID, "Workspace 1")
		ws2 := createWSServiceTestWorkspace(userID, "Workspace 2")
		expectedWorkspaces := []*workspace.Workspace{ws1, ws2}

		queryRepo := &mockWSServiceQueryRepo{
			listWorkspacesByUserFunc: func(_ context.Context, uid uuid.UUID, offset, limit int) ([]*workspace.Workspace, error) {
				assert.Equal(t, userID, uid)
				assert.Equal(t, 0, offset)
				assert.Equal(t, 10, limit)
				return expectedWorkspaces, nil
			},
			countWorkspacesByUserFunc: func(_ context.Context, uid uuid.UUID) (int, error) {
				assert.Equal(t, userID, uid)
				return 2, nil
			},
		}

		svc := service.NewWorkspaceService(service.WorkspaceServiceConfig{
			CreateUC:    &mockWSCreateUseCase{},
			GetUC:       &mockWSGetUseCase{},
			UpdateUC:    &mockWSUpdateUseCase{},
			CommandRepo: &mockWSServiceCommandRepo{},
			QueryRepo:   queryRepo,
		})

		workspaces, total, err := svc.ListUserWorkspaces(context.Background(), userID, 0, 10)

		require.NoError(t, err)
		assert.Len(t, workspaces, 2)
		assert.Equal(t, 2, total)
	})

	t.Run("user has no workspaces", func(t *testing.T) {
		userID := uuid.NewUUID()

		queryRepo := &mockWSServiceQueryRepo{
			listWorkspacesByUserFunc: func(_ context.Context, _ uuid.UUID, _, _ int) ([]*workspace.Workspace, error) {
				return []*workspace.Workspace{}, nil
			},
			countWorkspacesByUserFunc: func(_ context.Context, _ uuid.UUID) (int, error) {
				return 0, nil
			},
		}

		svc := service.NewWorkspaceService(service.WorkspaceServiceConfig{
			CreateUC:    &mockWSCreateUseCase{},
			GetUC:       &mockWSGetUseCase{},
			UpdateUC:    &mockWSUpdateUseCase{},
			CommandRepo: &mockWSServiceCommandRepo{},
			QueryRepo:   queryRepo,
		})

		workspaces, total, err := svc.ListUserWorkspaces(context.Background(), userID, 0, 10)

		require.NoError(t, err)
		assert.Empty(t, workspaces)
		assert.Equal(t, 0, total)
	})

	t.Run("pagination works correctly", func(t *testing.T) {
		userID := uuid.NewUUID()
		ws := createWSServiceTestWorkspace(userID, "Workspace")

		queryRepo := &mockWSServiceQueryRepo{
			listWorkspacesByUserFunc: func(_ context.Context, _ uuid.UUID, offset, limit int) ([]*workspace.Workspace, error) {
				assert.Equal(t, 10, offset)
				assert.Equal(t, 5, limit)
				return []*workspace.Workspace{ws}, nil
			},
			countWorkspacesByUserFunc: func(_ context.Context, _ uuid.UUID) (int, error) {
				return 15, nil
			},
		}

		svc := service.NewWorkspaceService(service.WorkspaceServiceConfig{
			CreateUC:    &mockWSCreateUseCase{},
			GetUC:       &mockWSGetUseCase{},
			UpdateUC:    &mockWSUpdateUseCase{},
			CommandRepo: &mockWSServiceCommandRepo{},
			QueryRepo:   queryRepo,
		})

		workspaces, total, err := svc.ListUserWorkspaces(context.Background(), userID, 10, 5)

		require.NoError(t, err)
		assert.Len(t, workspaces, 1)
		assert.Equal(t, 15, total)
	})

	t.Run("list returns error", func(t *testing.T) {
		userID := uuid.NewUUID()
		expectedErr := errors.New("database error")

		queryRepo := &mockWSServiceQueryRepo{
			listWorkspacesByUserFunc: func(_ context.Context, _ uuid.UUID, _, _ int) ([]*workspace.Workspace, error) {
				return nil, expectedErr
			},
		}

		svc := service.NewWorkspaceService(service.WorkspaceServiceConfig{
			CreateUC:    &mockWSCreateUseCase{},
			GetUC:       &mockWSGetUseCase{},
			UpdateUC:    &mockWSUpdateUseCase{},
			CommandRepo: &mockWSServiceCommandRepo{},
			QueryRepo:   queryRepo,
		})

		workspaces, total, err := svc.ListUserWorkspaces(context.Background(), userID, 0, 10)

		require.Error(t, err)
		assert.Nil(t, workspaces)
		assert.Equal(t, 0, total)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("count returns error", func(t *testing.T) {
		userID := uuid.NewUUID()
		expectedErr := errors.New("count error")

		queryRepo := &mockWSServiceQueryRepo{
			listWorkspacesByUserFunc: func(_ context.Context, _ uuid.UUID, _, _ int) ([]*workspace.Workspace, error) {
				return []*workspace.Workspace{}, nil
			},
			countWorkspacesByUserFunc: func(_ context.Context, _ uuid.UUID) (int, error) {
				return 0, expectedErr
			},
		}

		svc := service.NewWorkspaceService(service.WorkspaceServiceConfig{
			CreateUC:    &mockWSCreateUseCase{},
			GetUC:       &mockWSGetUseCase{},
			UpdateUC:    &mockWSUpdateUseCase{},
			CommandRepo: &mockWSServiceCommandRepo{},
			QueryRepo:   queryRepo,
		})

		workspaces, total, err := svc.ListUserWorkspaces(context.Background(), userID, 0, 10)

		require.Error(t, err)
		assert.Nil(t, workspaces)
		assert.Equal(t, 0, total)
		assert.Equal(t, expectedErr, err)
	})
}

func TestWorkspaceService_UpdateWorkspace(t *testing.T) {
	t.Run("successfully update workspace", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		ownerID := uuid.NewUUID()
		expectedWS := createWSServiceTestWorkspace(ownerID, "Updated Workspace")

		updateUC := &mockWSUpdateUseCase{
			executeFunc: func(_ context.Context, cmd wsapp.UpdateWorkspaceCommand) (wsapp.Result, error) {
				assert.Equal(t, workspaceID, cmd.WorkspaceID)
				assert.Equal(t, "Updated Workspace", cmd.Name)
				return wsapp.Result{
					Result: appcore.Result[*workspace.Workspace]{Value: expectedWS},
				}, nil
			},
		}

		svc := service.NewWorkspaceService(service.WorkspaceServiceConfig{
			CreateUC:    &mockWSCreateUseCase{},
			GetUC:       &mockWSGetUseCase{},
			UpdateUC:    updateUC,
			CommandRepo: &mockWSServiceCommandRepo{},
			QueryRepo:   &mockWSServiceQueryRepo{},
		})

		ws, err := svc.UpdateWorkspace(context.Background(), workspaceID, "Updated Workspace", "Description")

		require.NoError(t, err)
		require.NotNil(t, ws)
		assert.Equal(t, expectedWS.Name(), ws.Name())
	})

	t.Run("workspace not found", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		expectedErr := wsapp.ErrWorkspaceNotFound

		updateUC := &mockWSUpdateUseCase{
			executeFunc: func(_ context.Context, _ wsapp.UpdateWorkspaceCommand) (wsapp.Result, error) {
				return wsapp.Result{}, expectedErr
			},
		}

		svc := service.NewWorkspaceService(service.WorkspaceServiceConfig{
			CreateUC:    &mockWSCreateUseCase{},
			GetUC:       &mockWSGetUseCase{},
			UpdateUC:    updateUC,
			CommandRepo: &mockWSServiceCommandRepo{},
			QueryRepo:   &mockWSServiceQueryRepo{},
		})

		ws, err := svc.UpdateWorkspace(context.Background(), workspaceID, "Updated", "")

		require.Error(t, err)
		assert.Nil(t, ws)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("validation error", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		expectedErr := errors.New("validation failed: name is required")

		updateUC := &mockWSUpdateUseCase{
			executeFunc: func(_ context.Context, _ wsapp.UpdateWorkspaceCommand) (wsapp.Result, error) {
				return wsapp.Result{}, expectedErr
			},
		}

		svc := service.NewWorkspaceService(service.WorkspaceServiceConfig{
			CreateUC:    &mockWSCreateUseCase{},
			GetUC:       &mockWSGetUseCase{},
			UpdateUC:    updateUC,
			CommandRepo: &mockWSServiceCommandRepo{},
			QueryRepo:   &mockWSServiceQueryRepo{},
		})

		ws, err := svc.UpdateWorkspace(context.Background(), workspaceID, "", "")

		require.Error(t, err)
		assert.Nil(t, ws)
	})
}

func TestWorkspaceService_DeleteWorkspace(t *testing.T) {
	t.Run("successfully delete workspace", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		deleteCalled := false

		commandRepo := &mockWSServiceCommandRepo{
			deleteFunc: func(_ context.Context, id uuid.UUID) error {
				assert.Equal(t, workspaceID, id)
				deleteCalled = true
				return nil
			},
		}

		svc := service.NewWorkspaceService(service.WorkspaceServiceConfig{
			CreateUC:    &mockWSCreateUseCase{},
			GetUC:       &mockWSGetUseCase{},
			UpdateUC:    &mockWSUpdateUseCase{},
			CommandRepo: commandRepo,
			QueryRepo:   &mockWSServiceQueryRepo{},
		})

		err := svc.DeleteWorkspace(context.Background(), workspaceID)

		require.NoError(t, err)
		assert.True(t, deleteCalled)
	})

	t.Run("workspace not found", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		expectedErr := errors.New("not found")

		commandRepo := &mockWSServiceCommandRepo{
			deleteFunc: func(_ context.Context, _ uuid.UUID) error {
				return expectedErr
			},
		}

		svc := service.NewWorkspaceService(service.WorkspaceServiceConfig{
			CreateUC:    &mockWSCreateUseCase{},
			GetUC:       &mockWSGetUseCase{},
			UpdateUC:    &mockWSUpdateUseCase{},
			CommandRepo: commandRepo,
			QueryRepo:   &mockWSServiceQueryRepo{},
		})

		err := svc.DeleteWorkspace(context.Background(), workspaceID)

		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

func TestWorkspaceService_GetMemberCount(t *testing.T) {
	t.Run("workspace has members", func(t *testing.T) {
		workspaceID := uuid.NewUUID()

		queryRepo := &mockWSServiceQueryRepo{
			countMembersFunc: func(_ context.Context, wsID uuid.UUID) (int, error) {
				assert.Equal(t, workspaceID, wsID)
				return 5, nil
			},
		}

		svc := service.NewWorkspaceService(service.WorkspaceServiceConfig{
			CreateUC:    &mockWSCreateUseCase{},
			GetUC:       &mockWSGetUseCase{},
			UpdateUC:    &mockWSUpdateUseCase{},
			CommandRepo: &mockWSServiceCommandRepo{},
			QueryRepo:   queryRepo,
		})

		count, err := svc.GetMemberCount(context.Background(), workspaceID)

		require.NoError(t, err)
		assert.Equal(t, 5, count)
	})

	t.Run("workspace has no members", func(t *testing.T) {
		workspaceID := uuid.NewUUID()

		queryRepo := &mockWSServiceQueryRepo{
			countMembersFunc: func(_ context.Context, _ uuid.UUID) (int, error) {
				return 0, nil
			},
		}

		svc := service.NewWorkspaceService(service.WorkspaceServiceConfig{
			CreateUC:    &mockWSCreateUseCase{},
			GetUC:       &mockWSGetUseCase{},
			UpdateUC:    &mockWSUpdateUseCase{},
			CommandRepo: &mockWSServiceCommandRepo{},
			QueryRepo:   queryRepo,
		})

		count, err := svc.GetMemberCount(context.Background(), workspaceID)

		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("repository returns error", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		expectedErr := errors.New("database error")

		queryRepo := &mockWSServiceQueryRepo{
			countMembersFunc: func(_ context.Context, _ uuid.UUID) (int, error) {
				return 0, expectedErr
			},
		}

		svc := service.NewWorkspaceService(service.WorkspaceServiceConfig{
			CreateUC:    &mockWSCreateUseCase{},
			GetUC:       &mockWSGetUseCase{},
			UpdateUC:    &mockWSUpdateUseCase{},
			CommandRepo: &mockWSServiceCommandRepo{},
			QueryRepo:   queryRepo,
		})

		count, err := svc.GetMemberCount(context.Background(), workspaceID)

		require.Error(t, err)
		assert.Equal(t, 0, count)
		assert.Equal(t, expectedErr, err)
	})
}
