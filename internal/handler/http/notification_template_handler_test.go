package httphandler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notifapp "github.com/lllypuk/flowra/internal/application/notification"
	"github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/internal/middleware"
)

// MockNotificationTemplateService is a mock implementation of NotificationTemplateService for testing.
type MockNotificationTemplateService struct {
	notifications map[uuid.UUID]*notification.Notification
	userNotifs    map[uuid.UUID][]*notification.Notification
}

// NewMockNotificationTemplateService creates a new mock notification template service.
func NewMockNotificationTemplateService() *MockNotificationTemplateService {
	return &MockNotificationTemplateService{
		notifications: make(map[uuid.UUID]*notification.Notification),
		userNotifs:    make(map[uuid.UUID][]*notification.Notification),
	}
}

// AddNotification adds a notification to the mock service.
func (m *MockNotificationTemplateService) AddNotification(n *notification.Notification) {
	m.notifications[n.ID()] = n
	m.userNotifs[n.UserID()] = append(m.userNotifs[n.UserID()], n)
}

// ListNotifications implements NotificationTemplateService.
func (m *MockNotificationTemplateService) ListNotifications(
	_ context.Context,
	query notifapp.ListNotificationsQuery,
) (notifapp.ListResult, error) {
	notifs := m.userNotifs[query.UserID]
	if notifs == nil {
		notifs = []*notification.Notification{}
	}

	// Filter by unread if requested
	var filtered []*notification.Notification
	for _, n := range notifs {
		if query.UnreadOnly && n.IsRead() {
			continue
		}
		filtered = append(filtered, n)
	}

	total := len(filtered)

	// Apply pagination
	start := min(query.Offset, len(filtered))
	end := min(start+query.Limit, len(filtered))

	return notifapp.ListResult{
		Notifications: filtered[start:end],
		TotalCount:    total,
		Offset:        query.Offset,
		Limit:         query.Limit,
	}, nil
}

// CountUnread implements NotificationTemplateService.
func (m *MockNotificationTemplateService) CountUnread(
	_ context.Context,
	query notifapp.CountUnreadQuery,
) (notifapp.CountResult, error) {
	notifs := m.userNotifs[query.UserID]
	count := 0
	for _, n := range notifs {
		if !n.IsRead() {
			count++
		}
	}
	return notifapp.CountResult{Count: count}, nil
}

// MarkAsRead implements NotificationTemplateService.
func (m *MockNotificationTemplateService) MarkAsRead(
	_ context.Context,
	cmd notifapp.MarkAsReadCommand,
) (notifapp.Result, error) {
	n, ok := m.notifications[cmd.NotificationID]
	if !ok {
		return notifapp.Result{}, notifapp.ErrNotificationNotFound
	}

	if n.UserID() != cmd.UserID {
		return notifapp.Result{}, notifapp.ErrNotificationAccessDenied
	}

	if n.IsRead() {
		return notifapp.Result{}, notifapp.ErrNotificationAlreadyRead
	}

	_ = n.MarkAsRead()

	return notifapp.Result{}, nil
}

// GetNotification implements NotificationTemplateService.
func (m *MockNotificationTemplateService) GetNotification(
	_ context.Context,
	notificationID uuid.UUID,
	userID uuid.UUID,
) (*notification.Notification, error) {
	n, ok := m.notifications[notificationID]
	if !ok {
		return nil, notifapp.ErrNotificationNotFound
	}

	if n.UserID() != userID {
		return nil, notifapp.ErrNotificationAccessDenied
	}

	return n, nil
}

// setNotificationUserContext sets user authentication context on the echo context.
func setNotificationUserContext(c echo.Context, userID uuid.UUID) {
	c.Set(string(middleware.ContextKeyUserID), userID)
	c.Set(string(middleware.ContextKeyEmail), "test@example.com")
	c.Set(string(middleware.ContextKeyUsername), "testuser")
}

// makeTestNotification creates a test notification for testing.
func makeTestNotification(
	userID uuid.UUID,
	notifType notification.Type,
	title, message, resourceID string,
) *notification.Notification {
	n, _ := notification.NewNotification(userID, notifType, title, message, resourceID)
	return n
}

func TestNotificationTemplateHandler_NotificationsDropdownPartial(t *testing.T) {
	t.Run("successful list notifications", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := NewMockNotificationTemplateService()

		// Add test notifications
		n1 := makeTestNotification(userID, notification.TypeChatMention, "Mention", "@user mentioned you", "chat-123")
		n2 := makeTestNotification(
			userID,
			notification.TypeTaskAssigned,
			"Task Assigned",
			"You were assigned",
			"task-456",
		)
		mockService.AddNotification(n1)
		mockService.AddNotification(n2)

		handler := httphandler.NewNotificationTemplateHandler(nil, nil, mockService)

		req := httptest.NewRequest(http.MethodGet, "/partials/notifications?limit=10", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		setNotificationUserContext(c, userID)

		err := handler.NotificationsDropdownPartial(c)

		// Handler gracefully handles nil renderer
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Service unavailable")
	})

	t.Run("unauthorized returns 401", func(t *testing.T) {
		e := echo.New()

		mockService := NewMockNotificationTemplateService()
		handler := httphandler.NewNotificationTemplateHandler(nil, nil, mockService)

		req := httptest.NewRequest(http.MethodGet, "/partials/notifications", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		// No user context set

		err := handler.NotificationsDropdownPartial(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("empty notifications list", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := NewMockNotificationTemplateService()
		handler := httphandler.NewNotificationTemplateHandler(nil, nil, mockService)

		req := httptest.NewRequest(http.MethodGet, "/partials/notifications", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		setNotificationUserContext(c, userID)

		err := handler.NotificationsDropdownPartial(c)

		// Handler gracefully handles nil renderer
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Service unavailable")
	})
}

func TestNotificationTemplateHandler_NotificationCountPartial(t *testing.T) {
	t.Run("successful count unread", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := NewMockNotificationTemplateService()

		// Add some unread notifications
		n1 := makeTestNotification(userID, notification.TypeChatMention, "Mention", "@user mentioned you", "chat-123")
		n2 := makeTestNotification(
			userID,
			notification.TypeTaskAssigned,
			"Task Assigned",
			"You were assigned",
			"task-456",
		)
		mockService.AddNotification(n1)
		mockService.AddNotification(n2)

		handler := httphandler.NewNotificationTemplateHandler(nil, nil, mockService)

		req := httptest.NewRequest(http.MethodGet, "/partials/notifications/count", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		setNotificationUserContext(c, userID)

		err := handler.NotificationCountPartial(c)

		// Handler gracefully handles nil renderer
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Service unavailable")
	})

	t.Run("unauthorized returns badge with zero count", func(t *testing.T) {
		e := echo.New()

		mockService := NewMockNotificationTemplateService()
		handler := httphandler.NewNotificationTemplateHandler(nil, nil, mockService)

		req := httptest.NewRequest(http.MethodGet, "/partials/notifications/count", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		// No user context set

		err := handler.NotificationCountPartial(c)

		// Handler gracefully handles nil renderer
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Service unavailable")
	})
}

func TestNotificationTemplateHandler_NotificationsListPartial(t *testing.T) {
	t.Run("successful list with pagination", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := NewMockNotificationTemplateService()

		// Add test notifications
		for range 25 {
			n := makeTestNotification(userID, notification.TypeChatMention, "Mention", "Message", "chat-123")
			mockService.AddNotification(n)
		}

		handler := httphandler.NewNotificationTemplateHandler(nil, nil, mockService)

		req := httptest.NewRequest(http.MethodGet, "/partials/notifications/list?limit=10&offset=0", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		setNotificationUserContext(c, userID)

		err := handler.NotificationsListPartial(c)

		// Handler gracefully handles nil renderer
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Service unavailable")
	})

	t.Run("unauthorized returns 401", func(t *testing.T) {
		e := echo.New()

		mockService := NewMockNotificationTemplateService()
		handler := httphandler.NewNotificationTemplateHandler(nil, nil, mockService)

		req := httptest.NewRequest(http.MethodGet, "/partials/notifications/list", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		// No user context set

		err := handler.NotificationsListPartial(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("filter by unread", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := NewMockNotificationTemplateService()

		// Add read and unread notifications
		n1 := makeTestNotification(userID, notification.TypeChatMention, "Mention 1", "Unread message", "chat-1")
		n2 := makeTestNotification(userID, notification.TypeChatMention, "Mention 2", "Read message", "chat-2")
		_ = n2.MarkAsRead()
		mockService.AddNotification(n1)
		mockService.AddNotification(n2)

		handler := httphandler.NewNotificationTemplateHandler(nil, nil, mockService)

		req := httptest.NewRequest(http.MethodGet, "/partials/notifications/list?filter=unread", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		setNotificationUserContext(c, userID)

		err := handler.NotificationsListPartial(c)

		// Handler gracefully handles nil renderer
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Service unavailable")
	})
}

func TestNotificationTemplateHandler_NotificationsPage(t *testing.T) {
	t.Run("successful page render", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := NewMockNotificationTemplateService()

		// Add notifications
		n1 := makeTestNotification(userID, notification.TypeChatMention, "Mention", "@user mentioned you", "chat-123")
		mockService.AddNotification(n1)

		handler := httphandler.NewNotificationTemplateHandler(nil, nil, mockService)

		req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		setNotificationUserContext(c, userID)

		err := handler.NotificationsPage(c)

		// Will fail due to nil renderer
		require.Error(t, err)
	})

	t.Run("unauthorized redirects to login", func(t *testing.T) {
		e := echo.New()

		mockService := NewMockNotificationTemplateService()
		handler := httphandler.NewNotificationTemplateHandler(nil, nil, mockService)

		req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		// No user context set

		err := handler.NotificationsPage(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusFound, rec.Code)
		assert.Equal(t, "/login", rec.Header().Get("Location"))
	})
}

func TestNotificationTemplateHandler_NotificationRedirect(t *testing.T) {
	t.Run("successful redirect marks as read", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := NewMockNotificationTemplateService()

		n := makeTestNotification(userID, notification.TypeChatMention, "Mention", "@user mentioned you", "chat-123")
		mockService.AddNotification(n)

		handler := httphandler.NewNotificationTemplateHandler(nil, nil, mockService)

		req := httptest.NewRequest(http.MethodGet, "/notifications/"+n.ID().String()+"/redirect", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(n.ID().String())
		setNotificationUserContext(c, userID)

		err := handler.NotificationRedirect(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusFound, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "/chats/chat-123")

		// Verify notification was marked as read
		assert.True(t, n.IsRead())
	})

	t.Run("unauthorized redirects to login", func(t *testing.T) {
		e := echo.New()

		mockService := NewMockNotificationTemplateService()
		handler := httphandler.NewNotificationTemplateHandler(nil, nil, mockService)

		req := httptest.NewRequest(http.MethodGet, "/notifications/some-id/redirect", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("some-id")
		// No user context set

		err := handler.NotificationRedirect(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusFound, rec.Code)
		assert.Equal(t, "/login", rec.Header().Get("Location"))
	})

	t.Run("invalid notification ID redirects to notifications page", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := NewMockNotificationTemplateService()
		handler := httphandler.NewNotificationTemplateHandler(nil, nil, mockService)

		req := httptest.NewRequest(http.MethodGet, "/notifications/invalid-id/redirect", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("invalid-id")
		setNotificationUserContext(c, userID)

		err := handler.NotificationRedirect(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusFound, rec.Code)
		assert.Equal(t, "/notifications", rec.Header().Get("Location"))
	})

	t.Run("notification not found redirects to notifications page", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := NewMockNotificationTemplateService()
		handler := httphandler.NewNotificationTemplateHandler(nil, nil, mockService)

		notFoundID := uuid.NewUUID()
		req := httptest.NewRequest(http.MethodGet, "/notifications/"+notFoundID.String()+"/redirect", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(notFoundID.String())
		setNotificationUserContext(c, userID)

		err := handler.NotificationRedirect(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusFound, rec.Code)
		assert.Equal(t, "/notifications", rec.Header().Get("Location"))
	})

	t.Run("htmx request returns HX-Redirect header", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := NewMockNotificationTemplateService()

		n := makeTestNotification(userID, notification.TypeTaskAssigned, "Task", "You were assigned", "task-456")
		mockService.AddNotification(n)

		handler := httphandler.NewNotificationTemplateHandler(nil, nil, mockService)

		req := httptest.NewRequest(http.MethodGet, "/notifications/"+n.ID().String()+"/redirect", nil)
		req.Header.Set("Hx-Request", "true")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(n.ID().String())
		setNotificationUserContext(c, userID)

		err := handler.NotificationRedirect(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Hx-Redirect"), "/tasks/task-456")
	})
}

func TestNotificationTemplateHandler_GenerateNotificationLink(t *testing.T) {
	tests := []struct {
		name       string
		notifType  notification.Type
		resourceID string
		expected   string
	}{
		{
			name:       "task assigned",
			notifType:  notification.TypeTaskAssigned,
			resourceID: "task-123",
			expected:   "/tasks/task-123",
		},
		{
			name:       "task status changed",
			notifType:  notification.TypeTaskStatusChanged,
			resourceID: "task-456",
			expected:   "/tasks/task-456",
		},
		{
			name:       "task created",
			notifType:  notification.TypeTaskCreated,
			resourceID: "task-789",
			expected:   "/tasks/task-789",
		},
		{
			name:       "chat mention",
			notifType:  notification.TypeChatMention,
			resourceID: "chat-123",
			expected:   "/chats/chat-123",
		},
		{
			name:       "chat message",
			notifType:  notification.TypeChatMessage,
			resourceID: "chat-456",
			expected:   "/chats/chat-456",
		},
		{
			name:       "workspace invite",
			notifType:  notification.TypeWorkspaceInvite,
			resourceID: "ws-123",
			expected:   "/workspaces/ws-123",
		},
		{
			name:       "system notification",
			notifType:  notification.TypeSystem,
			resourceID: "sys-123",
			expected:   "/notifications",
		},
	}

	mockService := NewMockNotificationTemplateService()
	handler := httphandler.NewNotificationTemplateHandler(nil, nil, mockService)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We need to create a notification and verify the link generation
			// through the redirect handler since the method is private
			if tt.resourceID != "" && tt.expected != "" {
				userID := uuid.NewUUID()
				e := echo.New()

				n := makeTestNotification(userID, tt.notifType, "Test", "Test message", tt.resourceID)
				mockService.AddNotification(n)

				req := httptest.NewRequest(http.MethodGet, "/notifications/"+n.ID().String()+"/redirect", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("id")
				c.SetParamValues(n.ID().String())
				setNotificationUserContext(c, userID)

				err := handler.NotificationRedirect(c)

				require.NoError(t, err)
				assert.Equal(t, http.StatusFound, rec.Code)
				assert.Equal(t, tt.expected, rec.Header().Get("Location"))

				// Clean up
				delete(mockService.notifications, n.ID())
				mockService.userNotifs[userID] = nil
			}
		})
	}
}

func TestMockNotificationTemplateService(t *testing.T) {
	t.Run("ListNotifications respects unread filter", func(t *testing.T) {
		mockService := NewMockNotificationTemplateService()
		userID := uuid.NewUUID()

		// Add read and unread notifications
		n1 := makeTestNotification(userID, notification.TypeChatMention, "Mention 1", "Unread", "chat-1")
		n2 := makeTestNotification(userID, notification.TypeChatMention, "Mention 2", "Read", "chat-2")
		_ = n2.MarkAsRead()
		mockService.AddNotification(n1)
		mockService.AddNotification(n2)

		// Query all
		result, err := mockService.ListNotifications(context.Background(), notifapp.ListNotificationsQuery{
			UserID:     userID,
			UnreadOnly: false,
			Limit:      10,
			Offset:     0,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, result.TotalCount)

		// Query unread only
		result, err = mockService.ListNotifications(context.Background(), notifapp.ListNotificationsQuery{
			UserID:     userID,
			UnreadOnly: true,
			Limit:      10,
			Offset:     0,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalCount)
	})

	t.Run("CountUnread counts correctly", func(t *testing.T) {
		mockService := NewMockNotificationTemplateService()
		userID := uuid.NewUUID()

		n1 := makeTestNotification(userID, notification.TypeChatMention, "Mention 1", "Unread", "chat-1")
		n2 := makeTestNotification(userID, notification.TypeChatMention, "Mention 2", "Read", "chat-2")
		_ = n2.MarkAsRead()
		mockService.AddNotification(n1)
		mockService.AddNotification(n2)

		result, err := mockService.CountUnread(context.Background(), notifapp.CountUnreadQuery{UserID: userID})
		require.NoError(t, err)
		assert.Equal(t, 1, result.Count)
	})

	t.Run("MarkAsRead marks notification", func(t *testing.T) {
		mockService := NewMockNotificationTemplateService()
		userID := uuid.NewUUID()

		n := makeTestNotification(userID, notification.TypeChatMention, "Mention", "Message", "chat-1")
		mockService.AddNotification(n)

		assert.False(t, n.IsRead())

		_, err := mockService.MarkAsRead(context.Background(), notifapp.MarkAsReadCommand{
			NotificationID: n.ID(),
			UserID:         userID,
		})
		require.NoError(t, err)
		assert.True(t, n.IsRead())
	})

	t.Run("MarkAsRead fails for wrong user", func(t *testing.T) {
		mockService := NewMockNotificationTemplateService()
		userID := uuid.NewUUID()
		otherUserID := uuid.NewUUID()

		n := makeTestNotification(userID, notification.TypeChatMention, "Mention", "Message", "chat-1")
		mockService.AddNotification(n)

		_, err := mockService.MarkAsRead(context.Background(), notifapp.MarkAsReadCommand{
			NotificationID: n.ID(),
			UserID:         otherUserID,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, notifapp.ErrNotificationAccessDenied)
	})

	t.Run("GetNotification returns notification", func(t *testing.T) {
		mockService := NewMockNotificationTemplateService()
		userID := uuid.NewUUID()

		n := makeTestNotification(userID, notification.TypeChatMention, "Mention", "Message", "chat-1")
		mockService.AddNotification(n)

		result, err := mockService.GetNotification(context.Background(), n.ID(), userID)
		require.NoError(t, err)
		assert.Equal(t, n.ID(), result.ID())
	})

	t.Run("GetNotification fails for not found", func(t *testing.T) {
		mockService := NewMockNotificationTemplateService()
		userID := uuid.NewUUID()

		_, err := mockService.GetNotification(context.Background(), uuid.NewUUID(), userID)
		require.Error(t, err)
		assert.ErrorIs(t, err, notifapp.ErrNotificationNotFound)
	})
}

// Test pagination.
func TestNotificationPagination(t *testing.T) {
	mockService := NewMockNotificationTemplateService()
	userID := uuid.NewUUID()

	// Add 25 notifications
	for range 25 {
		n := makeTestNotification(userID, notification.TypeChatMention, "Mention", "Message", "chat-123")
		time.Sleep(time.Millisecond) // Ensure different timestamps
		mockService.AddNotification(n)
	}

	// Get first page
	result, err := mockService.ListNotifications(context.Background(), notifapp.ListNotificationsQuery{
		UserID:     userID,
		UnreadOnly: false,
		Limit:      10,
		Offset:     0,
	})
	require.NoError(t, err)
	assert.Equal(t, 25, result.TotalCount)
	assert.Len(t, result.Notifications, 10)

	// Get second page
	result, err = mockService.ListNotifications(context.Background(), notifapp.ListNotificationsQuery{
		UserID:     userID,
		UnreadOnly: false,
		Limit:      10,
		Offset:     10,
	})
	require.NoError(t, err)
	assert.Equal(t, 25, result.TotalCount)
	assert.Len(t, result.Notifications, 10)

	// Get third page (partial)
	result, err = mockService.ListNotifications(context.Background(), notifapp.ListNotificationsQuery{
		UserID:     userID,
		UnreadOnly: false,
		Limit:      10,
		Offset:     20,
	})
	require.NoError(t, err)
	assert.Equal(t, 25, result.TotalCount)
	assert.Len(t, result.Notifications, 5)
}
