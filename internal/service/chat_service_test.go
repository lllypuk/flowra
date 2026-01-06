package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/appcore"
	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/service"
)

// Mock use cases

type mockCreateChatUseCase struct {
	executeFunc func(ctx context.Context, cmd chatapp.CreateChatCommand) (chatapp.Result, error)
}

func (m *mockCreateChatUseCase) Execute(ctx context.Context, cmd chatapp.CreateChatCommand) (chatapp.Result, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return chatapp.Result{}, nil
}

type mockGetChatUseCase struct {
	executeFunc func(ctx context.Context, query chatapp.GetChatQuery) (*chatapp.GetChatResult, error)
}

func (m *mockGetChatUseCase) Execute(
	ctx context.Context,
	query chatapp.GetChatQuery,
) (*chatapp.GetChatResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, query)
	}
	return &chatapp.GetChatResult{}, nil
}

type mockListChatsUseCase struct {
	executeFunc func(ctx context.Context, query chatapp.ListChatsQuery) (*chatapp.ListChatsResult, error)
}

func (m *mockListChatsUseCase) Execute(
	ctx context.Context,
	query chatapp.ListChatsQuery,
) (*chatapp.ListChatsResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, query)
	}
	return &chatapp.ListChatsResult{}, nil
}

type mockRenameChatUseCase struct {
	executeFunc func(ctx context.Context, cmd chatapp.RenameChatCommand) (chatapp.Result, error)
}

func (m *mockRenameChatUseCase) Execute(ctx context.Context, cmd chatapp.RenameChatCommand) (chatapp.Result, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return chatapp.Result{}, nil
}

type mockAddParticipantUseCase struct {
	executeFunc func(ctx context.Context, cmd chatapp.AddParticipantCommand) (chatapp.Result, error)
}

func (m *mockAddParticipantUseCase) Execute(
	ctx context.Context,
	cmd chatapp.AddParticipantCommand,
) (chatapp.Result, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return chatapp.Result{}, nil
}

type mockRemoveParticipantUseCase struct {
	executeFunc func(ctx context.Context, cmd chatapp.RemoveParticipantCommand) (chatapp.Result, error)
}

func (m *mockRemoveParticipantUseCase) Execute(
	ctx context.Context,
	cmd chatapp.RemoveParticipantCommand,
) (chatapp.Result, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cmd)
	}
	return chatapp.Result{}, nil
}

// Mock event store

type mockEventStore struct {
	loadEventsFunc  func(ctx context.Context, aggregateID string) ([]event.DomainEvent, error)
	saveEventsFunc  func(ctx context.Context, aggregateID string, events []event.DomainEvent, expectedVersion int) error
	getVersionFunc  func(ctx context.Context, aggregateID string) (int, error)
	savedEvents     []event.DomainEvent
	savedAggregates map[string][]event.DomainEvent
}

func newMockEventStore() *mockEventStore {
	return &mockEventStore{
		savedAggregates: make(map[string][]event.DomainEvent),
	}
}

func (m *mockEventStore) LoadEvents(ctx context.Context, aggregateID string) ([]event.DomainEvent, error) {
	if m.loadEventsFunc != nil {
		return m.loadEventsFunc(ctx, aggregateID)
	}
	return nil, nil
}

func (m *mockEventStore) SaveEvents(
	ctx context.Context,
	aggregateID string,
	events []event.DomainEvent,
	expectedVersion int,
) error {
	if m.saveEventsFunc != nil {
		return m.saveEventsFunc(ctx, aggregateID, events, expectedVersion)
	}
	m.savedEvents = events
	m.savedAggregates[aggregateID] = append(m.savedAggregates[aggregateID], events...)
	return nil
}

func (m *mockEventStore) GetVersion(ctx context.Context, aggregateID string) (int, error) {
	if m.getVersionFunc != nil {
		return m.getVersionFunc(ctx, aggregateID)
	}
	return 0, nil
}

// Helper functions

func createTestChatEvents(chatID, workspaceID, createdBy uuid.UUID) []event.DomainEvent {
	now := time.Now()
	metadata := event.Metadata{}

	return []event.DomainEvent{
		chat.NewChatCreated(chatID, workspaceID, chat.TypeDiscussion, true, createdBy, now, metadata),
		chat.NewParticipantAdded(chatID, createdBy, chat.RoleAdmin, now, 2, metadata),
	}
}

func createDefaultServiceConfig() service.ChatServiceConfig {
	return service.ChatServiceConfig{
		CreateUC:     &mockCreateChatUseCase{},
		GetUC:        &mockGetChatUseCase{},
		ListUC:       &mockListChatsUseCase{},
		RenameUC:     &mockRenameChatUseCase{},
		AddPartUC:    &mockAddParticipantUseCase{},
		RemovePartUC: &mockRemoveParticipantUseCase{},
		EventStore:   newMockEventStore(),
	}
}

// Tests

func TestChatService_CreateChat(t *testing.T) {
	t.Run("successfully create chat", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		createdBy := uuid.NewUUID()
		chatID := uuid.NewUUID()

		expectedChat, _ := chat.NewChat(workspaceID, chat.TypeDiscussion, true, createdBy)

		createUC := &mockCreateChatUseCase{
			executeFunc: func(_ context.Context, cmd chatapp.CreateChatCommand) (chatapp.Result, error) {
				assert.Equal(t, workspaceID, cmd.WorkspaceID)
				assert.Equal(t, chat.TypeDiscussion, cmd.Type)
				assert.Equal(t, createdBy, cmd.CreatedBy)
				return chatapp.Result{
					Result: appcore.Result[*chat.Chat]{
						Value:   expectedChat,
						Version: 1,
					},
				}, nil
			},
		}

		cfg := createDefaultServiceConfig()
		cfg.CreateUC = createUC
		svc := service.NewChatService(cfg)

		result, err := svc.CreateChat(context.Background(), chatapp.CreateChatCommand{
			WorkspaceID: workspaceID,
			Type:        chat.TypeDiscussion,
			IsPublic:    true,
			CreatedBy:   createdBy,
		})

		require.NoError(t, err)
		assert.NotNil(t, result.Value)
		assert.Equal(t, workspaceID, result.Value.WorkspaceID())
		_ = chatID // suppress unused warning
	})

	t.Run("use case returns error", func(t *testing.T) {
		expectedErr := errors.New("validation failed")

		createUC := &mockCreateChatUseCase{
			executeFunc: func(_ context.Context, _ chatapp.CreateChatCommand) (chatapp.Result, error) {
				return chatapp.Result{}, expectedErr
			},
		}

		cfg := createDefaultServiceConfig()
		cfg.CreateUC = createUC
		svc := service.NewChatService(cfg)

		result, err := svc.CreateChat(context.Background(), chatapp.CreateChatCommand{
			WorkspaceID: uuid.NewUUID(),
			Type:        chat.TypeDiscussion,
			CreatedBy:   uuid.NewUUID(),
		})

		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result.Value)
	})

	t.Run("create task type chat with title", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		createdBy := uuid.NewUUID()
		title := "Test Task"

		createUC := &mockCreateChatUseCase{
			executeFunc: func(_ context.Context, cmd chatapp.CreateChatCommand) (chatapp.Result, error) {
				assert.Equal(t, chat.TypeTask, cmd.Type)
				assert.Equal(t, title, cmd.Title)
				return chatapp.Result{
					Result: appcore.Result[*chat.Chat]{Version: 1},
				}, nil
			},
		}

		cfg := createDefaultServiceConfig()
		cfg.CreateUC = createUC
		svc := service.NewChatService(cfg)

		_, err := svc.CreateChat(context.Background(), chatapp.CreateChatCommand{
			WorkspaceID: workspaceID,
			Type:        chat.TypeTask,
			Title:       title,
			CreatedBy:   createdBy,
		})

		require.NoError(t, err)
	})
}

func TestChatService_GetChat(t *testing.T) {
	t.Run("chat exists returns chat", func(t *testing.T) {
		chatID := uuid.NewUUID()
		requestedBy := uuid.NewUUID()

		expectedResult := &chatapp.GetChatResult{
			Chat: &chatapp.Chat{
				ID:        chatID,
				Type:      chat.TypeDiscussion,
				IsPublic:  true,
				CreatedBy: uuid.NewUUID(),
			},
			Permissions: chatapp.Permissions{
				CanRead:  true,
				CanWrite: true,
			},
		}

		getUC := &mockGetChatUseCase{
			executeFunc: func(_ context.Context, query chatapp.GetChatQuery) (*chatapp.GetChatResult, error) {
				assert.Equal(t, chatID, query.ChatID)
				assert.Equal(t, requestedBy, query.RequestedBy)
				return expectedResult, nil
			},
		}

		cfg := createDefaultServiceConfig()
		cfg.GetUC = getUC
		svc := service.NewChatService(cfg)

		result, err := svc.GetChat(context.Background(), chatapp.GetChatQuery{
			ChatID:      chatID,
			RequestedBy: requestedBy,
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, chatID, result.Chat.ID)
		assert.True(t, result.Permissions.CanRead)
	})

	t.Run("chat not found returns error", func(t *testing.T) {
		getUC := &mockGetChatUseCase{
			executeFunc: func(_ context.Context, _ chatapp.GetChatQuery) (*chatapp.GetChatResult, error) {
				return nil, chatapp.ErrChatNotFound
			},
		}

		cfg := createDefaultServiceConfig()
		cfg.GetUC = getUC
		svc := service.NewChatService(cfg)

		result, err := svc.GetChat(context.Background(), chatapp.GetChatQuery{
			ChatID:      uuid.NewUUID(),
			RequestedBy: uuid.NewUUID(),
		})

		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, chatapp.ErrChatNotFound)
	})
}

func TestChatService_ListChats(t *testing.T) {
	t.Run("workspace has chats returns list", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		requestedBy := uuid.NewUUID()

		expectedResult := &chatapp.ListChatsResult{
			Chats: []chatapp.Chat{
				{ID: uuid.NewUUID(), Type: chat.TypeDiscussion},
				{ID: uuid.NewUUID(), Type: chat.TypeTask},
			},
			Total:   2,
			HasMore: false,
		}

		listUC := &mockListChatsUseCase{
			executeFunc: func(_ context.Context, query chatapp.ListChatsQuery) (*chatapp.ListChatsResult, error) {
				assert.Equal(t, workspaceID, query.WorkspaceID)
				assert.Equal(t, requestedBy, query.RequestedBy)
				return expectedResult, nil
			},
		}

		cfg := createDefaultServiceConfig()
		cfg.ListUC = listUC
		svc := service.NewChatService(cfg)

		result, err := svc.ListChats(context.Background(), chatapp.ListChatsQuery{
			WorkspaceID: workspaceID,
			RequestedBy: requestedBy,
			Limit:       20,
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Len(t, result.Chats, 2)
		assert.Equal(t, 2, result.Total)
		assert.False(t, result.HasMore)
	})

	t.Run("empty workspace returns empty list", func(t *testing.T) {
		expectedResult := &chatapp.ListChatsResult{
			Chats:   []chatapp.Chat{},
			Total:   0,
			HasMore: false,
		}

		listUC := &mockListChatsUseCase{
			executeFunc: func(_ context.Context, _ chatapp.ListChatsQuery) (*chatapp.ListChatsResult, error) {
				return expectedResult, nil
			},
		}

		cfg := createDefaultServiceConfig()
		cfg.ListUC = listUC
		svc := service.NewChatService(cfg)

		result, err := svc.ListChats(context.Background(), chatapp.ListChatsQuery{
			WorkspaceID: uuid.NewUUID(),
			RequestedBy: uuid.NewUUID(),
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result.Chats)
		assert.Equal(t, 0, result.Total)
	})

	t.Run("list with type filter", func(t *testing.T) {
		taskType := chat.TypeTask

		listUC := &mockListChatsUseCase{
			executeFunc: func(_ context.Context, query chatapp.ListChatsQuery) (*chatapp.ListChatsResult, error) {
				assert.NotNil(t, query.Type)
				assert.Equal(t, chat.TypeTask, *query.Type)
				return &chatapp.ListChatsResult{
					Chats: []chatapp.Chat{
						{ID: uuid.NewUUID(), Type: chat.TypeTask},
					},
					Total:   1,
					HasMore: false,
				}, nil
			},
		}

		cfg := createDefaultServiceConfig()
		cfg.ListUC = listUC
		svc := service.NewChatService(cfg)

		result, err := svc.ListChats(context.Background(), chatapp.ListChatsQuery{
			WorkspaceID: uuid.NewUUID(),
			RequestedBy: uuid.NewUUID(),
			Type:        &taskType,
		})

		require.NoError(t, err)
		assert.Len(t, result.Chats, 1)
	})
}

func TestChatService_RenameChat(t *testing.T) {
	t.Run("successfully rename chat", func(t *testing.T) {
		chatID := uuid.NewUUID()
		renamedBy := uuid.NewUUID()
		newTitle := "New Title"

		renameUC := &mockRenameChatUseCase{
			executeFunc: func(_ context.Context, cmd chatapp.RenameChatCommand) (chatapp.Result, error) {
				assert.Equal(t, chatID, cmd.ChatID)
				assert.Equal(t, newTitle, cmd.NewTitle)
				assert.Equal(t, renamedBy, cmd.RenamedBy)
				return chatapp.Result{
					Result: appcore.Result[*chat.Chat]{Version: 2},
				}, nil
			},
		}

		cfg := createDefaultServiceConfig()
		cfg.RenameUC = renameUC
		svc := service.NewChatService(cfg)

		result, err := svc.RenameChat(context.Background(), chatapp.RenameChatCommand{
			ChatID:    chatID,
			NewTitle:  newTitle,
			RenamedBy: renamedBy,
		})

		require.NoError(t, err)
		assert.Equal(t, 2, result.Version)
	})

	t.Run("chat not found returns error", func(t *testing.T) {
		renameUC := &mockRenameChatUseCase{
			executeFunc: func(_ context.Context, _ chatapp.RenameChatCommand) (chatapp.Result, error) {
				return chatapp.Result{}, chatapp.ErrChatNotFound
			},
		}

		cfg := createDefaultServiceConfig()
		cfg.RenameUC = renameUC
		svc := service.NewChatService(cfg)

		_, err := svc.RenameChat(context.Background(), chatapp.RenameChatCommand{
			ChatID:    uuid.NewUUID(),
			NewTitle:  "New Title",
			RenamedBy: uuid.NewUUID(),
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, chatapp.ErrChatNotFound)
	})
}

func TestChatService_AddParticipant(t *testing.T) {
	t.Run("successfully add participant", func(t *testing.T) {
		chatID := uuid.NewUUID()
		userID := uuid.NewUUID()
		addedBy := uuid.NewUUID()

		addPartUC := &mockAddParticipantUseCase{
			executeFunc: func(_ context.Context, cmd chatapp.AddParticipantCommand) (chatapp.Result, error) {
				assert.Equal(t, chatID, cmd.ChatID)
				assert.Equal(t, userID, cmd.UserID)
				assert.Equal(t, chat.RoleMember, cmd.Role)
				assert.Equal(t, addedBy, cmd.AddedBy)
				return chatapp.Result{
					Result: appcore.Result[*chat.Chat]{Version: 3},
				}, nil
			},
		}

		cfg := createDefaultServiceConfig()
		cfg.AddPartUC = addPartUC
		svc := service.NewChatService(cfg)

		result, err := svc.AddParticipant(context.Background(), chatapp.AddParticipantCommand{
			ChatID:  chatID,
			UserID:  userID,
			Role:    chat.RoleMember,
			AddedBy: addedBy,
		})

		require.NoError(t, err)
		assert.Equal(t, 3, result.Version)
	})

	t.Run("already participant returns error", func(t *testing.T) {
		addPartUC := &mockAddParticipantUseCase{
			executeFunc: func(_ context.Context, _ chatapp.AddParticipantCommand) (chatapp.Result, error) {
				return chatapp.Result{}, chatapp.ErrUserAlreadyParticipant
			},
		}

		cfg := createDefaultServiceConfig()
		cfg.AddPartUC = addPartUC
		svc := service.NewChatService(cfg)

		_, err := svc.AddParticipant(context.Background(), chatapp.AddParticipantCommand{
			ChatID:  uuid.NewUUID(),
			UserID:  uuid.NewUUID(),
			Role:    chat.RoleMember,
			AddedBy: uuid.NewUUID(),
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, chatapp.ErrUserAlreadyParticipant)
	})

	t.Run("chat not found returns error", func(t *testing.T) {
		addPartUC := &mockAddParticipantUseCase{
			executeFunc: func(_ context.Context, _ chatapp.AddParticipantCommand) (chatapp.Result, error) {
				return chatapp.Result{}, chatapp.ErrChatNotFound
			},
		}

		cfg := createDefaultServiceConfig()
		cfg.AddPartUC = addPartUC
		svc := service.NewChatService(cfg)

		_, err := svc.AddParticipant(context.Background(), chatapp.AddParticipantCommand{
			ChatID:  uuid.NewUUID(),
			UserID:  uuid.NewUUID(),
			Role:    chat.RoleMember,
			AddedBy: uuid.NewUUID(),
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, chatapp.ErrChatNotFound)
	})
}

func TestChatService_RemoveParticipant(t *testing.T) {
	t.Run("successfully remove participant", func(t *testing.T) {
		chatID := uuid.NewUUID()
		userID := uuid.NewUUID()
		removedBy := uuid.NewUUID()

		removePartUC := &mockRemoveParticipantUseCase{
			executeFunc: func(_ context.Context, cmd chatapp.RemoveParticipantCommand) (chatapp.Result, error) {
				assert.Equal(t, chatID, cmd.ChatID)
				assert.Equal(t, userID, cmd.UserID)
				assert.Equal(t, removedBy, cmd.RemovedBy)
				return chatapp.Result{
					Result: appcore.Result[*chat.Chat]{Version: 3},
				}, nil
			},
		}

		cfg := createDefaultServiceConfig()
		cfg.RemovePartUC = removePartUC
		svc := service.NewChatService(cfg)

		result, err := svc.RemoveParticipant(context.Background(), chatapp.RemoveParticipantCommand{
			ChatID:    chatID,
			UserID:    userID,
			RemovedBy: removedBy,
		})

		require.NoError(t, err)
		assert.Equal(t, 3, result.Version)
	})

	t.Run("user not participant returns error", func(t *testing.T) {
		removePartUC := &mockRemoveParticipantUseCase{
			executeFunc: func(_ context.Context, _ chatapp.RemoveParticipantCommand) (chatapp.Result, error) {
				return chatapp.Result{}, chatapp.ErrUserNotParticipant
			},
		}

		cfg := createDefaultServiceConfig()
		cfg.RemovePartUC = removePartUC
		svc := service.NewChatService(cfg)

		_, err := svc.RemoveParticipant(context.Background(), chatapp.RemoveParticipantCommand{
			ChatID:    uuid.NewUUID(),
			UserID:    uuid.NewUUID(),
			RemovedBy: uuid.NewUUID(),
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, chatapp.ErrUserNotParticipant)
	})

	t.Run("cannot remove creator returns error", func(t *testing.T) {
		removePartUC := &mockRemoveParticipantUseCase{
			executeFunc: func(_ context.Context, _ chatapp.RemoveParticipantCommand) (chatapp.Result, error) {
				return chatapp.Result{}, chatapp.ErrCannotRemoveCreator
			},
		}

		cfg := createDefaultServiceConfig()
		cfg.RemovePartUC = removePartUC
		svc := service.NewChatService(cfg)

		_, err := svc.RemoveParticipant(context.Background(), chatapp.RemoveParticipantCommand{
			ChatID:    uuid.NewUUID(),
			UserID:    uuid.NewUUID(),
			RemovedBy: uuid.NewUUID(),
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, chatapp.ErrCannotRemoveCreator)
	})
}

func TestChatService_DeleteChat(t *testing.T) {
	t.Run("successfully delete chat", func(t *testing.T) {
		chatID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		createdBy := uuid.NewUUID()
		deletedBy := uuid.NewUUID()

		events := createTestChatEvents(chatID, workspaceID, createdBy)

		eventStore := newMockEventStore()
		eventStore.loadEventsFunc = func(_ context.Context, aggregateID string) ([]event.DomainEvent, error) {
			assert.Equal(t, chatID.String(), aggregateID)
			return events, nil
		}
		eventStore.getVersionFunc = func(_ context.Context, _ string) (int, error) {
			return 2, nil
		}

		var savedEvents []event.DomainEvent
		eventStore.saveEventsFunc = func(_ context.Context, aggregateID string, evts []event.DomainEvent, expectedVersion int) error {
			assert.Equal(t, chatID.String(), aggregateID)
			assert.Equal(t, 2, expectedVersion)
			savedEvents = evts
			return nil
		}

		cfg := createDefaultServiceConfig()
		cfg.EventStore = eventStore
		svc := service.NewChatService(cfg)

		err := svc.DeleteChat(context.Background(), chatID, deletedBy)

		require.NoError(t, err)
		require.Len(t, savedEvents, 1)
		deletedEvent, ok := savedEvents[0].(*chat.Deleted)
		require.True(t, ok, "expected Deleted event")
		assert.Equal(t, deletedBy, deletedEvent.DeletedBy)
	})

	t.Run("chat not found returns error", func(t *testing.T) {
		eventStore := newMockEventStore()
		eventStore.loadEventsFunc = func(_ context.Context, _ string) ([]event.DomainEvent, error) {
			return nil, appcore.ErrAggregateNotFound
		}

		cfg := createDefaultServiceConfig()
		cfg.EventStore = eventStore
		svc := service.NewChatService(cfg)

		err := svc.DeleteChat(context.Background(), uuid.NewUUID(), uuid.NewUUID())

		require.Error(t, err)
		assert.ErrorIs(t, err, chatapp.ErrChatNotFound)
	})

	t.Run("empty events returns chat not found", func(t *testing.T) {
		eventStore := newMockEventStore()
		eventStore.loadEventsFunc = func(_ context.Context, _ string) ([]event.DomainEvent, error) {
			return []event.DomainEvent{}, nil
		}

		cfg := createDefaultServiceConfig()
		cfg.EventStore = eventStore
		svc := service.NewChatService(cfg)

		err := svc.DeleteChat(context.Background(), uuid.NewUUID(), uuid.NewUUID())

		require.Error(t, err)
		assert.ErrorIs(t, err, chatapp.ErrChatNotFound)
	})

	t.Run("chat already deleted returns error", func(t *testing.T) {
		chatID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		createdBy := uuid.NewUUID()
		deletedBy := uuid.NewUUID()

		events := createTestChatEvents(chatID, workspaceID, createdBy)
		// Add a deleted event
		deletedEvent := chat.NewChatDeleted(chatID, createdBy, time.Now(), 3, event.Metadata{})
		events = append(events, deletedEvent)

		eventStore := newMockEventStore()
		eventStore.loadEventsFunc = func(_ context.Context, _ string) ([]event.DomainEvent, error) {
			return events, nil
		}

		cfg := createDefaultServiceConfig()
		cfg.EventStore = eventStore
		svc := service.NewChatService(cfg)

		err := svc.DeleteChat(context.Background(), chatID, deletedBy)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "already deleted")
	})

	t.Run("save events error returns error", func(t *testing.T) {
		chatID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		createdBy := uuid.NewUUID()
		deletedBy := uuid.NewUUID()

		events := createTestChatEvents(chatID, workspaceID, createdBy)

		eventStore := newMockEventStore()
		eventStore.loadEventsFunc = func(_ context.Context, _ string) ([]event.DomainEvent, error) {
			return events, nil
		}
		eventStore.getVersionFunc = func(_ context.Context, _ string) (int, error) {
			return 2, nil
		}
		eventStore.saveEventsFunc = func(_ context.Context, _ string, _ []event.DomainEvent, _ int) error {
			return errors.New("database error")
		}

		cfg := createDefaultServiceConfig()
		cfg.EventStore = eventStore
		svc := service.NewChatService(cfg)

		err := svc.DeleteChat(context.Background(), chatID, deletedBy)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save events")
	})

	t.Run("concurrency conflict returns concurrent update error", func(t *testing.T) {
		chatID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		createdBy := uuid.NewUUID()
		deletedBy := uuid.NewUUID()

		events := createTestChatEvents(chatID, workspaceID, createdBy)

		eventStore := newMockEventStore()
		eventStore.loadEventsFunc = func(_ context.Context, _ string) ([]event.DomainEvent, error) {
			return events, nil
		}
		eventStore.getVersionFunc = func(_ context.Context, _ string) (int, error) {
			return 2, nil
		}
		eventStore.saveEventsFunc = func(_ context.Context, _ string, _ []event.DomainEvent, _ int) error {
			return appcore.ErrConcurrencyConflict
		}

		cfg := createDefaultServiceConfig()
		cfg.EventStore = eventStore
		svc := service.NewChatService(cfg)

		err := svc.DeleteChat(context.Background(), chatID, deletedBy)

		require.Error(t, err)
		assert.ErrorIs(t, err, appcore.ErrConcurrentUpdate)
	})

	t.Run("validation error empty chatID", func(t *testing.T) {
		cfg := createDefaultServiceConfig()
		svc := service.NewChatService(cfg)

		err := svc.DeleteChat(context.Background(), "", uuid.NewUUID())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "chatID is required")
	})

	t.Run("validation error empty deletedBy", func(t *testing.T) {
		cfg := createDefaultServiceConfig()
		svc := service.NewChatService(cfg)

		err := svc.DeleteChat(context.Background(), uuid.NewUUID(), "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "deletedBy is required")
	})
}

func TestNewChatService(t *testing.T) {
	t.Run("creates service with all dependencies", func(t *testing.T) {
		cfg := createDefaultServiceConfig()
		svc := service.NewChatService(cfg)

		require.NotNil(t, svc)
	})
}
