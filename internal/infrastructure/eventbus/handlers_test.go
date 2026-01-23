package eventbus_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/application/notification"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/message"
	domainNotif "github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/eventbus"
	"github.com/lllypuk/flowra/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// syncBuffer is a thread-safe wrapper around bytes.Buffer for testing.
type syncBuffer struct {
	buf bytes.Buffer
	mu  sync.RWMutex
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buf.Len()
}

func (b *syncBuffer) String() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buf.String()
}

// mockNotificationRepository implements notification.Repository for testing.
type mockNotificationRepository struct {
	mu            sync.RWMutex
	notifications []*domainNotif.Notification
	saveErr       error
}

func newMockNotificationRepository() *mockNotificationRepository {
	return &mockNotificationRepository{
		notifications: make([]*domainNotif.Notification, 0),
	}
}

// CommandRepository methods

func (r *mockNotificationRepository) Save(_ context.Context, n *domainNotif.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.saveErr != nil {
		return r.saveErr
	}
	r.notifications = append(r.notifications, n)
	return nil
}

func (r *mockNotificationRepository) SaveBatch(_ context.Context, notifications []*domainNotif.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.saveErr != nil {
		return r.saveErr
	}
	r.notifications = append(r.notifications, notifications...)
	return nil
}

func (r *mockNotificationRepository) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (r *mockNotificationRepository) DeleteByUserID(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (r *mockNotificationRepository) DeleteOlderThan(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}

func (r *mockNotificationRepository) DeleteReadOlderThan(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}

func (r *mockNotificationRepository) MarkAsRead(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (r *mockNotificationRepository) MarkAllAsRead(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (r *mockNotificationRepository) MarkManyAsRead(_ context.Context, _ []uuid.UUID) error {
	return nil
}

// QueryRepository methods

func (r *mockNotificationRepository) FindByID(_ context.Context, _ uuid.UUID) (*domainNotif.Notification, error) {
	return nil, nil //nolint:nilnil // test mock returns nil for not found
}

func (r *mockNotificationRepository) FindByUserID(
	_ context.Context, _ uuid.UUID, _, _ int,
) ([]*domainNotif.Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.notifications, nil
}

func (r *mockNotificationRepository) FindUnreadByUserID(
	_ context.Context, _ uuid.UUID, _ int,
) ([]*domainNotif.Notification, error) {
	return nil, nil
}

func (r *mockNotificationRepository) FindByType(
	_ context.Context,
	_ uuid.UUID,
	_ domainNotif.Type,
	_, _ int,
) ([]*domainNotif.Notification, error) {
	return nil, nil
}

func (r *mockNotificationRepository) FindByResourceID(
	_ context.Context, _ string,
) ([]*domainNotif.Notification, error) {
	return nil, nil
}

func (r *mockNotificationRepository) CountUnreadByUserID(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (r *mockNotificationRepository) CountByType(_ context.Context, _ uuid.UUID) (map[domainNotif.Type]int, error) {
	return nil, nil //nolint:nilnil // test mock returns nil for empty counts
}

func (r *mockNotificationRepository) GetNotifications() []*domainNotif.Notification {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*domainNotif.Notification, len(r.notifications))
	copy(result, r.notifications)
	return result
}

func (r *mockNotificationRepository) SetSaveError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saveErr = err
}

// mockUserResolver implements eventbus.UserResolver for testing.
type mockUserResolver struct {
	users map[string]uuid.UUID
}

func newMockUserResolver() *mockUserResolver {
	return &mockUserResolver{
		users: make(map[string]uuid.UUID),
	}
}

func (r *mockUserResolver) AddUser(username string, id uuid.UUID) {
	r.users[username] = id
}

func (r *mockUserResolver) ResolveUsername(_ context.Context, username string) (uuid.UUID, error) {
	if id, ok := r.users[username]; ok {
		return id, nil
	}
	return "", nil
}

// testPayloadEvent wraps an event with a JSON payload for testing handlers.
type testPayloadEvent struct {
	event.BaseEvent

	payload json.RawMessage
}

func (e *testPayloadEvent) Payload() json.RawMessage {
	return e.payload
}

func newTestPayloadEvent(eventType, aggregateID string, payload any) *testPayloadEvent {
	data, _ := json.Marshal(payload)
	return &testPayloadEvent{
		BaseEvent: event.NewBaseEvent(
			eventType,
			aggregateID,
			"Test",
			1,
			event.NewMetadata("user-123", "corr-1", "cause-1"),
		),
		payload: data,
	}
}

// ========== NotificationHandler Tests ==========

func TestNotificationHandler_NewNotificationHandler(t *testing.T) {
	t.Run("creates handler with defaults", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		handler := eventbus.NewNotificationHandler(uc)

		assert.NotNil(t, handler)
	})

	t.Run("applies options", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		logger := slog.Default()
		resolver := newMockUserResolver()

		handler := eventbus.NewNotificationHandler(uc,
			eventbus.WithNotificationLogger(logger),
			eventbus.WithUserResolver(resolver),
		)

		assert.NotNil(t, handler)
	})
}

func TestNotificationHandler_HandleParticipantAdded(t *testing.T) {
	t.Run("creates notification for added participant", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		handler := eventbus.NewNotificationHandler(uc)

		userID := uuid.NewUUID()
		evt := newTestPayloadEvent(
			chat.EventTypeParticipantAdded,
			"chat-123",
			map[string]any{
				"UserID": userID.String(),
				"Role":   "member",
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		notifications := repo.GetNotifications()
		assert.Len(t, notifications, 1)
		assert.Equal(t, userID, notifications[0].UserID())
		assert.Equal(t, domainNotif.TypeChatMessage, notifications[0].Type())
	})

	t.Run("skips notification when user adds themselves", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		handler := eventbus.NewNotificationHandler(uc)

		userID := uuid.NewUUID()
		// Create event with same user as metadata user ID
		payload := map[string]any{
			"UserID": userID.String(),
			"Role":   "member",
		}
		data, _ := json.Marshal(payload)
		evt := &testPayloadEvent{
			BaseEvent: event.NewBaseEvent(
				chat.EventTypeParticipantAdded,
				"chat-123",
				"Chat",
				1,
				event.NewMetadata(userID.String(), "corr-1", "cause-1"),
			),
			payload: data,
		}

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		notifications := repo.GetNotifications()
		assert.Empty(t, notifications)
	})

	t.Run("handles invalid payload gracefully", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		handler := eventbus.NewNotificationHandler(uc)

		evt := &testPayloadEvent{
			BaseEvent: event.NewBaseEvent(
				chat.EventTypeParticipantAdded,
				"chat-123",
				"Chat",
				1,
				event.Metadata{},
			),
			payload: []byte("invalid json"),
		}

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err) // Should not error, just skip

		notifications := repo.GetNotifications()
		assert.Empty(t, notifications)
	})
}

func TestNotificationHandler_HandleMessageCreated(t *testing.T) {
	t.Run("creates notification for mentioned users", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		resolver := newMockUserResolver()

		mentionedUserID := uuid.NewUUID()
		resolver.AddUser("john", mentionedUserID)

		handler := eventbus.NewNotificationHandler(uc,
			eventbus.WithUserResolver(resolver),
		)

		authorID := uuid.NewUUID()
		evt := newTestPayloadEvent(
			message.EventTypeMessageCreated,
			"msg-123",
			map[string]any{
				"ChatID":   "chat-456",
				"AuthorID": authorID.String(),
				"Content":  "Hello @john, how are you?",
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		notifications := repo.GetNotifications()
		assert.Len(t, notifications, 1)
		assert.Equal(t, mentionedUserID, notifications[0].UserID())
		assert.Equal(t, domainNotif.TypeChatMention, notifications[0].Type())
	})

	t.Run("handles multiple mentions", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		resolver := newMockUserResolver()

		user1ID := uuid.NewUUID()
		user2ID := uuid.NewUUID()
		resolver.AddUser("alice", user1ID)
		resolver.AddUser("bob", user2ID)

		handler := eventbus.NewNotificationHandler(uc,
			eventbus.WithUserResolver(resolver),
		)

		authorID := uuid.NewUUID()
		evt := newTestPayloadEvent(
			message.EventTypeMessageCreated,
			"msg-123",
			map[string]any{
				"ChatID":   "chat-456",
				"AuthorID": authorID.String(),
				"Content":  "Hey @alice and @bob, check this out!",
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		notifications := repo.GetNotifications()
		assert.Len(t, notifications, 2)
	})

	t.Run("deduplicates repeated mentions", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		resolver := newMockUserResolver()

		userID := uuid.NewUUID()
		resolver.AddUser("alice", userID)

		handler := eventbus.NewNotificationHandler(uc,
			eventbus.WithUserResolver(resolver),
		)

		authorID := uuid.NewUUID()
		evt := newTestPayloadEvent(
			message.EventTypeMessageCreated,
			"msg-123",
			map[string]any{
				"ChatID":   "chat-456",
				"AuthorID": authorID.String(),
				"Content":  "@alice @alice @alice hello!",
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		notifications := repo.GetNotifications()
		assert.Len(t, notifications, 1)
	})

	t.Run("skips when author mentions themselves", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		resolver := newMockUserResolver()

		authorID := uuid.NewUUID()
		resolver.AddUser("me", authorID)

		handler := eventbus.NewNotificationHandler(uc,
			eventbus.WithUserResolver(resolver),
		)

		evt := newTestPayloadEvent(
			message.EventTypeMessageCreated,
			"msg-123",
			map[string]any{
				"ChatID":   "chat-456",
				"AuthorID": authorID.String(),
				"Content":  "Note to @me: remember this",
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		notifications := repo.GetNotifications()
		assert.Empty(t, notifications)
	})

	t.Run("skips without user resolver", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		handler := eventbus.NewNotificationHandler(uc) // No resolver

		evt := newTestPayloadEvent(
			message.EventTypeMessageCreated,
			"msg-123",
			map[string]any{
				"ChatID":   "chat-456",
				"AuthorID": "author-123",
				"Content":  "Hello @john!",
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		notifications := repo.GetNotifications()
		assert.Empty(t, notifications)
	})

	t.Run("skips message without mentions", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		resolver := newMockUserResolver()
		handler := eventbus.NewNotificationHandler(uc,
			eventbus.WithUserResolver(resolver),
		)

		evt := newTestPayloadEvent(
			message.EventTypeMessageCreated,
			"msg-123",
			map[string]any{
				"ChatID":   "chat-456",
				"AuthorID": "author-123",
				"Content":  "Hello, world!",
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		notifications := repo.GetNotifications()
		assert.Empty(t, notifications)
	})
}

func TestNotificationHandler_HandleTaskCreated(t *testing.T) {
	t.Run("creates notification for assignee", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		handler := eventbus.NewNotificationHandler(uc)

		assigneeID := uuid.NewUUID()
		creatorID := uuid.NewUUID()
		evt := newTestPayloadEvent(
			task.EventTypeTaskCreated,
			"task-123",
			map[string]any{
				"Title":      "Important Task",
				"AssigneeID": assigneeID.String(),
				"CreatedBy":  creatorID.String(),
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		notifications := repo.GetNotifications()
		assert.Len(t, notifications, 1)
		assert.Equal(t, assigneeID, notifications[0].UserID())
		assert.Equal(t, domainNotif.TypeTaskAssigned, notifications[0].Type())
		assert.Contains(t, notifications[0].Message(), "Important Task")
	})

	t.Run("skips notification when creator is assignee", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		handler := eventbus.NewNotificationHandler(uc)

		userID := uuid.NewUUID()
		evt := newTestPayloadEvent(
			task.EventTypeTaskCreated,
			"task-123",
			map[string]any{
				"Title":      "My Task",
				"AssigneeID": userID.String(),
				"CreatedBy":  userID.String(),
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		notifications := repo.GetNotifications()
		assert.Empty(t, notifications)
	})

	t.Run("skips notification when no assignee", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		handler := eventbus.NewNotificationHandler(uc)

		evt := newTestPayloadEvent(
			task.EventTypeTaskCreated,
			"task-123",
			map[string]any{
				"Title":     "Unassigned Task",
				"CreatedBy": "creator-123",
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		notifications := repo.GetNotifications()
		assert.Empty(t, notifications)
	})

	t.Run("truncates long task titles", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		handler := eventbus.NewNotificationHandler(uc)

		assigneeID := uuid.NewUUID()
		longTitle := "This is a very long task title that should be truncated to ensure it fits nicely in the notification message without taking too much space"
		evt := newTestPayloadEvent(
			task.EventTypeTaskCreated,
			"task-123",
			map[string]any{
				"Title":      longTitle,
				"AssigneeID": assigneeID.String(),
				"CreatedBy":  "other-user",
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		notifications := repo.GetNotifications()
		require.Len(t, notifications, 1)
		assert.Contains(t, notifications[0].Message(), "...")
		assert.Less(t, len(notifications[0].Message()), len(longTitle))
	})
}

func TestNotificationHandler_HandleTaskAssigneeChanged(t *testing.T) {
	t.Run("creates notification for new assignee", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		handler := eventbus.NewNotificationHandler(uc)

		newAssigneeID := uuid.NewUUID()
		changerID := uuid.NewUUID()
		evt := newTestPayloadEvent(
			task.EventTypeAssigneeChanged,
			"task-123",
			map[string]any{
				"NewAssignee": newAssigneeID.String(),
				"ChangedBy":   changerID.String(),
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		notifications := repo.GetNotifications()
		assert.Len(t, notifications, 1)
		assert.Equal(t, newAssigneeID, notifications[0].UserID())
		assert.Equal(t, domainNotif.TypeTaskAssigned, notifications[0].Type())
	})

	t.Run("skips when user assigns to themselves", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		handler := eventbus.NewNotificationHandler(uc)

		userID := uuid.NewUUID()
		evt := newTestPayloadEvent(
			task.EventTypeAssigneeChanged,
			"task-123",
			map[string]any{
				"NewAssignee": userID.String(),
				"ChangedBy":   userID.String(),
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		notifications := repo.GetNotifications()
		assert.Empty(t, notifications)
	})

	t.Run("skips when assignee removed (nil)", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		handler := eventbus.NewNotificationHandler(uc)

		evt := newTestPayloadEvent(
			task.EventTypeAssigneeChanged,
			"task-123",
			map[string]any{
				"OldAssignee": "old-assignee-id",
				"ChangedBy":   "changer-id",
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		notifications := repo.GetNotifications()
		assert.Empty(t, notifications)
	})
}

func TestNotificationHandler_HandleChatCreated(t *testing.T) {
	t.Run("logs chat created event", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		handler := eventbus.NewNotificationHandler(uc,
			eventbus.WithNotificationLogger(logger),
		)

		evt := newTestPayloadEvent(
			chat.EventTypeChatCreated,
			"chat-123",
			map[string]any{
				"WorkspaceID": uuid.NewUUID().String(),
				"Type":        "general",
				"IsPublic":    true,
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		// Chat created is logged but doesn't create notifications directly
		notifications := repo.GetNotifications()
		assert.Empty(t, notifications)
		assert.Contains(t, buf.String(), "processing chat.created event")
	})
}

func TestNotificationHandler_HandleTaskStatusChanged(t *testing.T) {
	t.Run("logs task status change", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		handler := eventbus.NewNotificationHandler(uc,
			eventbus.WithNotificationLogger(logger),
		)

		evt := newTestPayloadEvent(
			task.EventTypeStatusChanged,
			"task-123",
			map[string]any{
				"OldStatus": "todo",
				"NewStatus": "in_progress",
				"ChangedBy": uuid.NewUUID().String(),
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		// Status changed is logged
		assert.Contains(t, buf.String(), "task status changed")
	})

	t.Run("handles invalid payload gracefully", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		handler := eventbus.NewNotificationHandler(uc)

		evt := &testPayloadEvent{
			BaseEvent: event.NewBaseEvent(
				task.EventTypeStatusChanged,
				"task-123",
				"Task",
				1,
				event.Metadata{},
			),
			payload: []byte("invalid json"),
		}

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err) // Should not error, just skip
	})
}

func TestNotificationHandler_HandleUnknownEvent(t *testing.T) {
	t.Run("ignores unknown event types", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		handler := eventbus.NewNotificationHandler(uc)

		evt := newTestPayloadEvent("unknown.event", "agg-123", map[string]any{})

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		notifications := repo.GetNotifications()
		assert.Empty(t, notifications)
	})
}

func TestNotificationHandler_RepositoryError(t *testing.T) {
	t.Run("propagates repository save error", func(t *testing.T) {
		repo := newMockNotificationRepository()
		repo.SetSaveError(errors.New("database connection failed"))
		uc := notification.NewCreateNotificationUseCase(repo)
		handler := eventbus.NewNotificationHandler(uc)

		assigneeID := uuid.NewUUID()
		evt := newTestPayloadEvent(
			task.EventTypeTaskCreated,
			"task-123",
			map[string]any{
				"Title":      "Test Task",
				"AssigneeID": assigneeID.String(),
				"CreatedBy":  "other-user",
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create task assignment notification")
	})
}

func TestNotificationHandler_HandleMessageCreatedErrors(t *testing.T) {
	t.Run("handles user resolver error", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)

		// Create a resolver that returns errors
		errorResolver := &errorUserResolver{err: errors.New("resolver error")}

		handler := eventbus.NewNotificationHandler(uc,
			eventbus.WithUserResolver(errorResolver),
		)

		evt := newTestPayloadEvent(
			message.EventTypeMessageCreated,
			"msg-123",
			map[string]any{
				"ChatID":   "chat-456",
				"AuthorID": uuid.NewUUID().String(),
				"Content":  "Hello @john!",
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err) // Errors are logged but don't fail the handler

		notifications := repo.GetNotifications()
		assert.Empty(t, notifications)
	})
}

// errorUserResolver is a mock that returns errors
type errorUserResolver struct {
	err error
}

func (r *errorUserResolver) ResolveUsername(_ context.Context, _ string) (uuid.UUID, error) {
	return "", r.err
}

func TestNotificationHandler_ExtractPayloadFromNonPayloadEvent(t *testing.T) {
	t.Run("marshals non-payload events", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		handler := eventbus.NewNotificationHandler(uc)

		// Create a regular BaseEvent (without Payload method)
		evt := &event.BaseEvent{}
		*evt = event.NewBaseEvent(
			chat.EventTypeChatCreated,
			"chat-123",
			"Chat",
			1,
			event.Metadata{},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)
	})
}

func TestNotificationHandler_AsEventHandler(t *testing.T) {
	t.Run("returns compatible EventHandler function", func(t *testing.T) {
		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		handler := eventbus.NewNotificationHandler(uc)

		fn := handler.AsEventHandler()
		assert.NotNil(t, fn)

		// Should be callable
		evt := newTestPayloadEvent("unknown.event", "agg-123", map[string]any{})
		err := fn(context.Background(), evt)
		require.NoError(t, err)
	})
}

// ========== LoggingHandler Tests ==========

func TestLoggingHandler_NewLoggingHandler(t *testing.T) {
	t.Run("creates with provided logger", func(t *testing.T) {
		logger := slog.Default()
		handler := eventbus.NewLoggingHandler(logger)
		assert.NotNil(t, handler)
	})

	t.Run("creates with default logger when nil", func(t *testing.T) {
		handler := eventbus.NewLoggingHandler(nil)
		assert.NotNil(t, handler)
	})
}

func TestLoggingHandler_Handle(t *testing.T) {
	t.Run("logs event details", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		handler := eventbus.NewLoggingHandler(logger)

		evt := newTestPayloadEvent(
			message.EventTypeMessageCreated,
			"msg-123",
			map[string]any{
				"Content": "Hello, world!",
			},
		)

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		logOutput := buf.String()
		assert.Contains(t, logOutput, "domain event")
		assert.Contains(t, logOutput, message.EventTypeMessageCreated)
		assert.Contains(t, logOutput, "msg-123")
	})

	t.Run("includes metadata in logs", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		handler := eventbus.NewLoggingHandler(logger)

		payload := map[string]any{"data": "test"}
		data, _ := json.Marshal(payload)
		evt := &testPayloadEvent{
			BaseEvent: event.NewBaseEvent(
				"test.event",
				"agg-123",
				"TestAggregate",
				1,
				event.NewMetadata("user-456", "corr-789", "cause-101"),
			),
			payload: data,
		}

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		logOutput := buf.String()
		assert.Contains(t, logOutput, "user-456")
		assert.Contains(t, logOutput, "corr-789")
	})

	t.Run("truncates large payloads", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, nil))
		handler := eventbus.NewLoggingHandler(logger)

		// Create a large payload
		largeContent := make([]byte, 1000)
		for i := range largeContent {
			largeContent[i] = 'x'
		}
		payload := map[string]any{"data": string(largeContent)}
		data, _ := json.Marshal(payload)
		evt := &testPayloadEvent{
			BaseEvent: event.NewBaseEvent("test.event", "agg-123", "Test", 1, event.Metadata{}),
			payload:   data,
		}

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)

		logOutput := buf.String()
		// Should contain truncation indicator
		assert.Contains(t, logOutput, "...")
	})

	t.Run("returns no error", func(t *testing.T) {
		handler := eventbus.NewLoggingHandler(nil)
		evt := newTestPayloadEvent("test.event", "agg-123", map[string]any{})

		err := handler.Handle(context.Background(), evt)
		require.NoError(t, err)
	})
}

func TestLoggingHandler_AsEventHandler(t *testing.T) {
	t.Run("returns compatible EventHandler function", func(t *testing.T) {
		handler := eventbus.NewLoggingHandler(nil)
		fn := handler.AsEventHandler()
		assert.NotNil(t, fn)

		evt := newTestPayloadEvent("test.event", "agg-123", map[string]any{})
		err := fn(context.Background(), evt)
		require.NoError(t, err)
	})
}

// ========== DeadLetterHandler Tests ==========

func TestDeadLetterHandler_NewDeadLetterHandler(t *testing.T) {
	client := testutil.SetupTestRedis(t)

	t.Run("creates with defaults", func(t *testing.T) {
		handler := eventbus.NewDeadLetterHandler(client)
		assert.NotNil(t, handler)
	})

	t.Run("applies options", func(t *testing.T) {
		logger := slog.Default()
		handler := eventbus.NewDeadLetterHandler(client,
			eventbus.WithDeadLetterLogger(logger),
			eventbus.WithDeadLetterQueueKey("custom:dead_letter"),
			eventbus.WithMaxDeadLetters(500),
		)
		assert.NotNil(t, handler)
	})
}

func TestDeadLetterHandler_Handle(t *testing.T) {
	client := testutil.SetupTestRedis(t)
	ctx := context.Background()

	t.Run("stores failed event in queue", func(t *testing.T) {
		handler := eventbus.NewDeadLetterHandler(client,
			eventbus.WithDeadLetterQueueKey("test:dlq:store"),
		)

		evt := newTestPayloadEvent("test.event", "agg-123", map[string]any{"key": "value"})
		originalErr := errors.New("handler failed: database timeout")

		handler.Handle(ctx, evt, originalErr)

		// Verify event was stored
		entries, err := handler.GetDeadLetters(ctx, 10)
		require.NoError(t, err)
		require.Len(t, entries, 1)

		assert.Equal(t, "test.event", entries[0].EventType)
		assert.Equal(t, "agg-123", entries[0].AggregateID)
		assert.Contains(t, entries[0].Error, "database timeout")
		assert.NotEmpty(t, entries[0].Payload)
	})

	t.Run("includes event payload", func(t *testing.T) {
		handler := eventbus.NewDeadLetterHandler(client,
			eventbus.WithDeadLetterQueueKey("test:dlq:payload"),
		)

		evt := newTestPayloadEvent("test.event", "agg-456", map[string]any{
			"important": "data",
			"count":     42,
		})
		handler.Handle(ctx, evt, errors.New("processing failed"))

		entries, err := handler.GetDeadLetters(ctx, 10)
		require.NoError(t, err)
		require.Len(t, entries, 1)

		var payload map[string]any
		err = json.Unmarshal(entries[0].Payload, &payload)
		require.NoError(t, err)
		assert.Equal(t, "data", payload["important"])
	})

	t.Run("respects max queue size", func(t *testing.T) {
		handler := eventbus.NewDeadLetterHandler(client,
			eventbus.WithDeadLetterQueueKey("test:dlq:maxsize"),
			eventbus.WithMaxDeadLetters(3),
		)

		// Add 5 events
		for i := range 5 {
			evt := newTestPayloadEvent("test.event", "agg-"+string(rune('0'+i)), map[string]any{})
			handler.Handle(ctx, evt, errors.New("failed"))
		}

		// Only 3 should be kept (the most recent ones)
		length, err := handler.QueueLength(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(3), length)
	})
}

func TestDeadLetterHandler_GetDeadLetters(t *testing.T) {
	client := testutil.SetupTestRedis(t)
	ctx := context.Background()

	t.Run("returns empty list when queue is empty", func(t *testing.T) {
		handler := eventbus.NewDeadLetterHandler(client,
			eventbus.WithDeadLetterQueueKey("test:dlq:empty"),
		)

		entries, err := handler.GetDeadLetters(ctx, 10)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("returns requested count", func(t *testing.T) {
		handler := eventbus.NewDeadLetterHandler(client,
			eventbus.WithDeadLetterQueueKey("test:dlq:count"),
		)

		// Add 5 events
		for i := range 5 {
			evt := newTestPayloadEvent("test.event", "agg-"+string(rune('a'+i)), map[string]any{})
			handler.Handle(ctx, evt, errors.New("failed"))
		}

		entries, err := handler.GetDeadLetters(ctx, 2)
		require.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("uses default count when zero or negative", func(t *testing.T) {
		handler := eventbus.NewDeadLetterHandler(client,
			eventbus.WithDeadLetterQueueKey("test:dlq:default"),
		)

		for range 15 {
			evt := newTestPayloadEvent("test.event", "agg-x", map[string]any{})
			handler.Handle(ctx, evt, errors.New("failed"))
		}

		entries, err := handler.GetDeadLetters(ctx, 0)
		require.NoError(t, err)
		assert.Len(t, entries, 10) // Default is 10
	})
}

func TestDeadLetterHandler_ClearDeadLetters(t *testing.T) {
	client := testutil.SetupTestRedis(t)
	ctx := context.Background()

	t.Run("removes all entries", func(t *testing.T) {
		handler := eventbus.NewDeadLetterHandler(client,
			eventbus.WithDeadLetterQueueKey("test:dlq:clear"),
		)

		// Add some events
		for range 3 {
			evt := newTestPayloadEvent("test.event", "agg-y", map[string]any{})
			handler.Handle(ctx, evt, errors.New("failed"))
		}

		// Verify events exist
		length, err := handler.QueueLength(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(3), length)

		// Clear queue
		err = handler.ClearDeadLetters(ctx)
		require.NoError(t, err)

		// Verify queue is empty
		length, err = handler.QueueLength(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), length)
	})
}

func TestDeadLetterHandler_QueueLength(t *testing.T) {
	client := testutil.SetupTestRedis(t)
	ctx := context.Background()

	t.Run("returns correct length", func(t *testing.T) {
		handler := eventbus.NewDeadLetterHandler(client,
			eventbus.WithDeadLetterQueueKey("test:dlq:length"),
		)

		// Initially empty
		length, err := handler.QueueLength(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), length)

		// Add events
		for range 5 {
			evt := newTestPayloadEvent("test.event", "agg-z", map[string]any{})
			handler.Handle(ctx, evt, errors.New("failed"))
		}

		length, err = handler.QueueLength(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(5), length)
	})
}

func TestDeadLetterHandler_HandleNonPayloadEvent(t *testing.T) {
	client := testutil.SetupTestRedis(t)
	ctx := context.Background()

	t.Run("handles event without payload", func(t *testing.T) {
		handler := eventbus.NewDeadLetterHandler(client,
			eventbus.WithDeadLetterQueueKey("test:dlq:nopayload"),
		)

		// Create a regular BaseEvent (without Payload method)
		evt := &event.BaseEvent{}
		*evt = event.NewBaseEvent(
			"test.event",
			"agg-nopayload",
			"Test",
			1,
			event.Metadata{},
		)

		handler.Handle(ctx, evt, errors.New("test error"))

		entries, err := handler.GetDeadLetters(ctx, 10)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.Equal(t, "test.event", entries[0].EventType)
		assert.Equal(t, "agg-nopayload", entries[0].AggregateID)
		// Payload should be nil/empty for non-payload events
		assert.Empty(t, entries[0].Payload)
	})
}

func TestHandlerRegistry_SetDeadLetterHandler(t *testing.T) {
	client := testutil.SetupTestRedis(t)

	t.Run("sets dead letter handler", func(_ *testing.T) {
		bus := eventbus.NewRedisEventBus(client)
		registry := eventbus.NewHandlerRegistry(bus, slog.Default())

		dlqHandler := eventbus.NewDeadLetterHandler(client)
		registry.SetDeadLetterHandler(dlqHandler)
		// No assertion needed - just verify it doesn't panic
	})
}

// ========== HandlerRegistry Tests ==========

func TestHandlerRegistry_Register(t *testing.T) {
	client := testutil.SetupTestRedis(t)

	t.Run("registers handler for multiple event types", func(t *testing.T) {
		bus := eventbus.NewRedisEventBus(client)
		registry := eventbus.NewHandlerRegistry(bus, slog.Default())

		handler := func(_ context.Context, _ event.DomainEvent) error {
			return nil
		}

		err := registry.Register([]string{"event.a", "event.b", "event.c"}, handler)
		require.NoError(t, err)

		assert.Equal(t, 1, bus.HandlerCount("event.a"))
		assert.Equal(t, 1, bus.HandlerCount("event.b"))
		assert.Equal(t, 1, bus.HandlerCount("event.c"))
	})
}

func TestHandlerRegistry_RegisterNotificationHandler(t *testing.T) {
	client := testutil.SetupTestRedis(t)

	t.Run("registers for all notification-relevant events", func(t *testing.T) {
		bus := eventbus.NewRedisEventBus(client)
		registry := eventbus.NewHandlerRegistry(bus, slog.Default())

		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		handler := eventbus.NewNotificationHandler(uc)

		err := registry.RegisterNotificationHandler(handler)
		require.NoError(t, err)

		// Check all expected events are subscribed
		assert.Equal(t, 1, bus.HandlerCount(chat.EventTypeChatCreated))
		assert.Equal(t, 1, bus.HandlerCount(chat.EventTypeParticipantAdded))
		assert.Equal(t, 1, bus.HandlerCount(message.EventTypeMessageCreated))
		assert.Equal(t, 1, bus.HandlerCount(task.EventTypeTaskCreated))
		assert.Equal(t, 1, bus.HandlerCount(task.EventTypeStatusChanged))
		assert.Equal(t, 1, bus.HandlerCount(task.EventTypeAssigneeChanged))
	})
}

func TestHandlerRegistry_RegisterLoggingHandler(t *testing.T) {
	client := testutil.SetupTestRedis(t)

	t.Run("registers for specified event types", func(t *testing.T) {
		bus := eventbus.NewRedisEventBus(client)
		registry := eventbus.NewHandlerRegistry(bus, slog.Default())

		handler := eventbus.NewLoggingHandler(slog.Default())
		eventTypes := []string{"event.x", "event.y"}

		err := registry.RegisterLoggingHandler(handler, eventTypes)
		require.NoError(t, err)

		assert.Equal(t, 1, bus.HandlerCount("event.x"))
		assert.Equal(t, 1, bus.HandlerCount("event.y"))
		assert.Equal(t, 0, bus.HandlerCount("event.z"))
	})
}

func TestRegisterAllHandlers(t *testing.T) {
	client := testutil.SetupTestRedis(t)

	t.Run("registers both handlers", func(t *testing.T) {
		bus := eventbus.NewRedisEventBus(client)
		logger := slog.Default()

		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		notifHandler := eventbus.NewNotificationHandler(uc)
		logHandler := eventbus.NewLoggingHandler(logger)

		err := eventbus.RegisterAllHandlers(bus, notifHandler, logHandler, nil, logger)
		require.NoError(t, err)

		// Notification handler events
		assert.GreaterOrEqual(t, bus.HandlerCount(chat.EventTypeChatCreated), 1)
		assert.GreaterOrEqual(t, bus.HandlerCount(task.EventTypeTaskCreated), 1)

		// Logging handler events (should have 2 handlers - notification + logging)
		assert.GreaterOrEqual(t, bus.HandlerCount(message.EventTypeMessageCreated), 2)
	})

	t.Run("handles nil notification handler", func(t *testing.T) {
		bus := eventbus.NewRedisEventBus(client)
		logger := slog.Default()
		logHandler := eventbus.NewLoggingHandler(logger)

		err := eventbus.RegisterAllHandlers(bus, nil, logHandler, nil, logger)
		require.NoError(t, err)
	})

	t.Run("handles nil logging handler", func(t *testing.T) {
		bus := eventbus.NewRedisEventBus(client)
		logger := slog.Default()

		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		notifHandler := eventbus.NewNotificationHandler(uc)

		err := eventbus.RegisterAllHandlers(bus, notifHandler, nil, nil, logger)
		require.NoError(t, err)
	})
}

// ========== Integration Tests ==========

func TestEventHandlers_Integration(t *testing.T) {
	client := testutil.SetupTestRedis(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("full pipeline with real event bus", func(t *testing.T) {
		bus := eventbus.NewRedisEventBus(client,
			eventbus.WithChannelPrefix("test:integration:"),
		)

		repo := newMockNotificationRepository()
		uc := notification.NewCreateNotificationUseCase(repo)
		notifHandler := eventbus.NewNotificationHandler(uc)

		// Register handler
		err := bus.Subscribe(task.EventTypeAssigneeChanged, notifHandler.AsEventHandler())
		require.NoError(t, err)

		// Start bus
		go func() {
			_ = bus.Start(ctx)
		}()
		time.Sleep(100 * time.Millisecond)

		// Publish event
		taskID := uuid.NewUUID()
		assigneeID := uuid.NewUUID()
		changerID := uuid.NewUUID()
		evt := task.NewAssigneeChanged(
			taskID,
			nil,
			&assigneeID,
			changerID,
			event.NewMetadata(changerID.String(), "corr-1", "cause-1"),
		)

		err = bus.Publish(ctx, evt)
		require.NoError(t, err)

		// Wait for notification to be created
		assert.Eventually(t, func() bool {
			notifications := repo.GetNotifications()
			return len(notifications) == 1
		}, 5*time.Second, 100*time.Millisecond)

		notifications := repo.GetNotifications()
		require.Len(t, notifications, 1)
		assert.Equal(t, assigneeID, notifications[0].UserID())
		assert.Equal(t, domainNotif.TypeTaskAssigned, notifications[0].Type())

		err = bus.Shutdown()
		require.NoError(t, err)
	})

	t.Run("logging handler captures all events", func(t *testing.T) {
		bus := eventbus.NewRedisEventBus(client,
			eventbus.WithChannelPrefix("test:logging:"),
		)

		buf := &syncBuffer{}
		logger := slog.New(slog.NewJSONHandler(buf, nil))
		logHandler := eventbus.NewLoggingHandler(logger)

		// Register for specific event
		err := bus.Subscribe(message.EventTypeMessageCreated, logHandler.AsEventHandler())
		require.NoError(t, err)

		// Start bus
		go func() {
			_ = bus.Start(ctx)
		}()
		time.Sleep(100 * time.Millisecond)

		// Publish event
		evt := message.NewCreated(
			uuid.NewUUID(),
			uuid.NewUUID(),
			uuid.NewUUID(),
			"Test message content",
			uuid.UUID(""),
			event.Metadata{},
		)

		err = bus.Publish(ctx, evt)
		require.NoError(t, err)

		// Wait for log to be written
		assert.Eventually(t, func() bool {
			return buf.Len() > 0
		}, 5*time.Second, 100*time.Millisecond)

		logOutput := buf.String()
		assert.Contains(t, logOutput, message.EventTypeMessageCreated)

		err = bus.Shutdown()
		require.NoError(t, err)
	})
}
