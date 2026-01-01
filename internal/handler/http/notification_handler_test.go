package httphandler_test

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	notifapp "github.com/lllypuk/flowra/internal/application/notification"
	"github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to set up notification auth context.
func setupNotificationAuthContext(c echo.Context, userID uuid.UUID) {
	c.Set(string(middleware.ContextKeyUserID), userID)
	c.Set(string(middleware.ContextKeyUsername), "testuser")
	c.Set(string(middleware.ContextKeyEmail), "test@example.com")
}

// Helper function to create a test notification.
func createTestNotification(t *testing.T, userID uuid.UUID) *notification.Notification {
	t.Helper()
	n, err := notification.NewNotification(
		userID,
		notification.TypeTaskAssigned,
		"Task Assigned",
		"You have been assigned a new task",
		uuid.NewUUID().String(),
	)
	require.NoError(t, err)
	return n
}

func TestNotificationHandler_List(t *testing.T) {
	t.Run("successful list notifications", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockNotificationService()
		notif1 := createTestNotification(t, userID)
		notif2 := createTestNotification(t, userID)
		mockService.AddNotification(notif1)
		mockService.AddNotification(notif2)

		handler := httphandler.NewNotificationHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/notifications", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupNotificationAuthContext(c, userID)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("list with unread_only filter", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockNotificationService()
		notif1 := createTestNotification(t, userID)
		notif2 := createTestNotification(t, userID)
		_ = notif2.MarkAsRead() // Mark one as read
		mockService.AddNotification(notif1)
		mockService.AddNotification(notif2)

		handler := httphandler.NewNotificationHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/notifications?unread_only=true", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupNotificationAuthContext(c, userID)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("list with pagination", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockNotificationService()
		for range 25 {
			mockService.AddNotification(createTestNotification(t, userID))
		}

		handler := httphandler.NewNotificationHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/notifications?page=2&limit=10", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupNotificationAuthContext(c, userID)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("missing auth", func(t *testing.T) {
		e := echo.New()

		mockService := httphandler.NewMockNotificationService()
		handler := httphandler.NewNotificationHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/notifications", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})

	t.Run("empty list", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockNotificationService()
		handler := httphandler.NewNotificationHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/notifications", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupNotificationAuthContext(c, userID)

		err := handler.List(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})
}

func TestNotificationHandler_UnreadCount(t *testing.T) {
	t.Run("successful unread count", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockNotificationService()
		notif1 := createTestNotification(t, userID)
		notif2 := createTestNotification(t, userID)
		notif3 := createTestNotification(t, userID)
		_ = notif3.MarkAsRead()
		mockService.AddNotification(notif1)
		mockService.AddNotification(notif2)
		mockService.AddNotification(notif3)

		handler := httphandler.NewNotificationHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/notifications/unread/count", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupNotificationAuthContext(c, userID)

		err := handler.UnreadCount(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		// Check count in data
		data, ok := resp.Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 2, int(data["count"].(float64)))
	})

	t.Run("zero unread count", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockNotificationService()
		handler := httphandler.NewNotificationHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/notifications/unread/count", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupNotificationAuthContext(c, userID)

		err := handler.UnreadCount(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		data, ok := resp.Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 0, int(data["count"].(float64)))
	})

	t.Run("missing auth", func(t *testing.T) {
		e := echo.New()

		mockService := httphandler.NewMockNotificationService()
		handler := httphandler.NewNotificationHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/notifications/unread/count", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.UnreadCount(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})
}

func TestNotificationHandler_MarkAsRead(t *testing.T) {
	t.Run("successful mark as read", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockNotificationService()
		notif := createTestNotification(t, userID)
		mockService.AddNotification(notif)

		handler := httphandler.NewNotificationHandler(mockService)

		url := "/api/v1/notifications/" + notif.ID().String() + "/read"
		req := httptest.NewRequest(stdhttp.MethodPut, url, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(notif.ID().String())

		setupNotificationAuthContext(c, userID)

		err := handler.MarkAsRead(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)
	})

	t.Run("notification not found", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		notifID := uuid.NewUUID()

		mockService := httphandler.NewMockNotificationService()
		handler := httphandler.NewNotificationHandler(mockService)

		url := "/api/v1/notifications/" + notifID.String() + "/read"
		req := httptest.NewRequest(stdhttp.MethodPut, url, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(notifID.String())

		setupNotificationAuthContext(c, userID)

		err := handler.MarkAsRead(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	})

	t.Run("access denied - different user", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		otherUserID := uuid.NewUUID()

		mockService := httphandler.NewMockNotificationService()
		notif := createTestNotification(t, otherUserID)
		mockService.AddNotification(notif)

		handler := httphandler.NewNotificationHandler(mockService)

		url := "/api/v1/notifications/" + notif.ID().String() + "/read"
		req := httptest.NewRequest(stdhttp.MethodPut, url, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(notif.ID().String())

		setupNotificationAuthContext(c, userID)

		err := handler.MarkAsRead(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusForbidden, rec.Code)
	})

	t.Run("already read notification", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockNotificationService()
		notif := createTestNotification(t, userID)
		_ = notif.MarkAsRead()
		mockService.AddNotification(notif)

		handler := httphandler.NewNotificationHandler(mockService)

		url := "/api/v1/notifications/" + notif.ID().String() + "/read"
		req := httptest.NewRequest(stdhttp.MethodPut, url, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(notif.ID().String())

		setupNotificationAuthContext(c, userID)

		err := handler.MarkAsRead(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusConflict, rec.Code)
	})

	t.Run("invalid notification ID", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockNotificationService()
		handler := httphandler.NewNotificationHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodPut, "/api/v1/notifications/invalid-id/read", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("invalid-id")

		setupNotificationAuthContext(c, userID)

		err := handler.MarkAsRead(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})
}

func TestNotificationHandler_MarkAllRead(t *testing.T) {
	t.Run("successful mark all as read", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockNotificationService()
		notif1 := createTestNotification(t, userID)
		notif2 := createTestNotification(t, userID)
		notif3 := createTestNotification(t, userID)
		mockService.AddNotification(notif1)
		mockService.AddNotification(notif2)
		mockService.AddNotification(notif3)

		handler := httphandler.NewNotificationHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodPut, "/api/v1/notifications/mark-all-read", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupNotificationAuthContext(c, userID)

		err := handler.MarkAllRead(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)

		data, ok := resp.Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 3, int(data["marked_count"].(float64)))
	})

	t.Run("no unread notifications", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockNotificationService()
		handler := httphandler.NewNotificationHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodPut, "/api/v1/notifications/mark-all-read", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		setupNotificationAuthContext(c, userID)

		err := handler.MarkAllRead(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusOK, rec.Code)

		var resp httpserver.Response
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		data, ok := resp.Data.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 0, int(data["marked_count"].(float64)))
	})

	t.Run("missing auth", func(t *testing.T) {
		e := echo.New()

		mockService := httphandler.NewMockNotificationService()
		handler := httphandler.NewNotificationHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodPut, "/api/v1/notifications/mark-all-read", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler.MarkAllRead(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusUnauthorized, rec.Code)
	})
}

func TestNotificationHandler_Delete(t *testing.T) {
	t.Run("successful delete notification", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockNotificationService()
		notif := createTestNotification(t, userID)
		mockService.AddNotification(notif)

		handler := httphandler.NewNotificationHandler(mockService)

		url := "/api/v1/notifications/" + notif.ID().String()
		req := httptest.NewRequest(stdhttp.MethodDelete, url, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(notif.ID().String())

		setupNotificationAuthContext(c, userID)

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNoContent, rec.Code)
	})

	t.Run("notification not found", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		notifID := uuid.NewUUID()

		mockService := httphandler.NewMockNotificationService()
		handler := httphandler.NewNotificationHandler(mockService)

		url := "/api/v1/notifications/" + notifID.String()
		req := httptest.NewRequest(stdhttp.MethodDelete, url, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(notifID.String())

		setupNotificationAuthContext(c, userID)

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusNotFound, rec.Code)
	})

	t.Run("access denied - different user", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()
		otherUserID := uuid.NewUUID()

		mockService := httphandler.NewMockNotificationService()
		notif := createTestNotification(t, otherUserID)
		mockService.AddNotification(notif)

		handler := httphandler.NewNotificationHandler(mockService)

		url := "/api/v1/notifications/" + notif.ID().String()
		req := httptest.NewRequest(stdhttp.MethodDelete, url, nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(notif.ID().String())

		setupNotificationAuthContext(c, userID)

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusForbidden, rec.Code)
	})

	t.Run("invalid notification ID", func(t *testing.T) {
		e := echo.New()
		userID := uuid.NewUUID()

		mockService := httphandler.NewMockNotificationService()
		handler := httphandler.NewNotificationHandler(mockService)

		req := httptest.NewRequest(stdhttp.MethodDelete, "/api/v1/notifications/invalid-id", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("invalid-id")

		setupNotificationAuthContext(c, userID)

		err := handler.Delete(c)
		require.NoError(t, err)
		assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	})
}

func TestNewNotificationHandler(t *testing.T) {
	mockService := httphandler.NewMockNotificationService()
	handler := httphandler.NewNotificationHandler(mockService)
	assert.NotNil(t, handler)
}

func TestToNotificationResponse(t *testing.T) {
	userID := uuid.NewUUID()
	resourceID := uuid.NewUUID().String()

	n, err := notification.NewNotification(
		userID,
		notification.TypeTaskAssigned,
		"Task Assigned",
		"You have been assigned to task XYZ",
		resourceID,
	)
	require.NoError(t, err)

	resp := httphandler.ToNotificationResponse(n)

	assert.Equal(t, n.ID().String(), resp.ID)
	assert.Equal(t, string(notification.TypeTaskAssigned), resp.Type)
	assert.Equal(t, "Task Assigned", resp.Title)
	assert.Equal(t, "You have been assigned to task XYZ", resp.Body)
	assert.False(t, resp.IsRead)
	assert.Equal(t, resourceID, resp.ResourceID)
	assert.NotEmpty(t, resp.CreatedAt)
	assert.Nil(t, resp.ReadAt)
	assert.Contains(t, resp.Link, "/tasks/")
}

func TestMockNotificationService(t *testing.T) {
	t.Run("list notifications", func(t *testing.T) {
		mockService := httphandler.NewMockNotificationService()
		userID := uuid.NewUUID()

		n1, _ := notification.NewNotification(userID, notification.TypeTaskCreated, "Title 1", "Body 1", "res1")
		n2, _ := notification.NewNotification(userID, notification.TypeTaskAssigned, "Title 2", "Body 2", "res2")
		mockService.AddNotification(n1)
		mockService.AddNotification(n2)

		query := notifapp.ListNotificationsQuery{
			UserID: userID,
			Limit:  10,
			Offset: 0,
		}

		result, err := mockService.ListNotifications(context.Background(), query)
		require.NoError(t, err)
		assert.Len(t, result.Notifications, 2)
	})

	t.Run("count unread", func(t *testing.T) {
		mockService := httphandler.NewMockNotificationService()
		userID := uuid.NewUUID()

		n1, _ := notification.NewNotification(userID, notification.TypeTaskCreated, "Title 1", "Body 1", "res1")
		n2, _ := notification.NewNotification(userID, notification.TypeTaskAssigned, "Title 2", "Body 2", "res2")
		_ = n2.MarkAsRead()
		mockService.AddNotification(n1)
		mockService.AddNotification(n2)

		query := notifapp.CountUnreadQuery{
			UserID: userID,
		}

		result, err := mockService.CountUnread(context.Background(), query)
		require.NoError(t, err)
		assert.Equal(t, 1, result.Count)
	})

	t.Run("mark all as read", func(t *testing.T) {
		mockService := httphandler.NewMockNotificationService()
		userID := uuid.NewUUID()

		n1, _ := notification.NewNotification(userID, notification.TypeTaskCreated, "Title 1", "Body 1", "res1")
		n2, _ := notification.NewNotification(userID, notification.TypeTaskAssigned, "Title 2", "Body 2", "res2")
		mockService.AddNotification(n1)
		mockService.AddNotification(n2)

		cmd := notifapp.MarkAllAsReadCommand{
			UserID: userID,
		}

		result, err := mockService.MarkAllAsRead(context.Background(), cmd)
		require.NoError(t, err)
		assert.Equal(t, 2, result.Count)

		// Verify all are read now
		countResult, _ := mockService.CountUnread(context.Background(), notifapp.CountUnreadQuery{UserID: userID})
		assert.Equal(t, 0, countResult.Count)
	})

	t.Run("delete notification", func(t *testing.T) {
		mockService := httphandler.NewMockNotificationService()
		userID := uuid.NewUUID()

		n, _ := notification.NewNotification(userID, notification.TypeTaskCreated, "Title", "Body", "res1")
		mockService.AddNotification(n)

		cmd := notifapp.DeleteNotificationCommand{
			NotificationID: n.ID(),
			UserID:         userID,
		}

		err := mockService.DeleteNotification(context.Background(), cmd)
		require.NoError(t, err)

		// Verify it's deleted
		_, err = mockService.GetNotification(context.Background(), n.ID(), userID)
		assert.ErrorIs(t, err, notifapp.ErrNotificationNotFound)
	})
}
