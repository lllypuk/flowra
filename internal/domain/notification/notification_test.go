package notification_test

import (
	"testing"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNotification(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		userID := uuid.NewUUID()
		typ := notification.TypeTaskAssigned
		title := "Task assigned to you"
		message := "You have been assigned task #123"
		resourceID := "task-123"

		notif, err := notification.NewNotification(userID, typ, title, message, resourceID)

		require.NoError(t, err)
		assert.False(t, notif.ID().IsZero())
		assert.Equal(t, userID, notif.UserID())
		assert.Equal(t, typ, notif.Type())
		assert.Equal(t, title, notif.Title())
		assert.Equal(t, message, notif.Message())
		assert.Equal(t, resourceID, notif.ResourceID())
		assert.Nil(t, notif.ReadAt())
		assert.False(t, notif.CreatedAt().IsZero())
		assert.False(t, notif.IsRead())
	})

	t.Run("empty user ID", func(t *testing.T) {
		_, err := notification.NewNotification(
			"",
			notification.TypeTaskAssigned,
			"Title",
			"Message",
			"resource-123",
		)
		require.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("empty type", func(t *testing.T) {
		_, err := notification.NewNotification(
			uuid.NewUUID(),
			"",
			"Title",
			"Message",
			"resource-123",
		)
		require.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("empty title", func(t *testing.T) {
		_, err := notification.NewNotification(
			uuid.NewUUID(),
			notification.TypeTaskAssigned,
			"",
			"Message",
			"resource-123",
		)
		require.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("empty message", func(t *testing.T) {
		_, err := notification.NewNotification(
			uuid.NewUUID(),
			notification.TypeTaskAssigned,
			"Title",
			"",
			"resource-123",
		)
		require.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("empty resource ID is allowed", func(t *testing.T) {
		notif, err := notification.NewNotification(
			uuid.NewUUID(),
			notification.TypeSystem,
			"System notification",
			"System maintenance scheduled",
			"",
		)
		require.NoError(t, err)
		assert.Empty(t, notif.ResourceID())
	})
}

func TestNotification_MarkAsRead(t *testing.T) {
	t.Run("successful mark as read", func(t *testing.T) {
		notif, _ := notification.NewNotification(
			uuid.NewUUID(),
			notification.TypeTaskAssigned,
			"Title",
			"Message",
			"resource-123",
		)

		assert.False(t, notif.IsRead())
		assert.Nil(t, notif.ReadAt())

		err := notif.MarkAsRead()
		require.NoError(t, err)
		assert.True(t, notif.IsRead())
		assert.NotNil(t, notif.ReadAt())
		assert.False(t, notif.ReadAt().IsZero())
	})

	t.Run("already read", func(t *testing.T) {
		notif, _ := notification.NewNotification(
			uuid.NewUUID(),
			notification.TypeTaskAssigned,
			"Title",
			"Message",
			"resource-123",
		)

		notif.MarkAsRead()
		err := notif.MarkAsRead()
		require.ErrorIs(t, err, errs.ErrInvalidState)
	})
}

func TestNotification_IsRead(t *testing.T) {
	t.Run("unread notification", func(t *testing.T) {
		notif, _ := notification.NewNotification(
			uuid.NewUUID(),
			notification.TypeTaskAssigned,
			"Title",
			"Message",
			"resource-123",
		)
		assert.False(t, notif.IsRead())
	})

	t.Run("read notification", func(t *testing.T) {
		notif, _ := notification.NewNotification(
			uuid.NewUUID(),
			notification.TypeTaskAssigned,
			"Title",
			"Message",
			"resource-123",
		)
		notif.MarkAsRead()
		assert.True(t, notif.IsRead())
	})
}

func TestNotificationTypes(t *testing.T) {
	tests := []struct {
		name string
		typ  notification.Type
	}{
		{"task status changed", notification.TypeTaskStatusChanged},
		{"task assigned", notification.TypeTaskAssigned},
		{"task created", notification.TypeTaskCreated},
		{"chat mention", notification.TypeChatMention},
		{"chat message", notification.TypeChatMessage},
		{"workspace invite", notification.TypeWorkspaceInvite},
		{"system", notification.TypeSystem},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notif, err := notification.NewNotification(
				uuid.NewUUID(),
				tt.typ,
				"Title",
				"Message",
				"resource-123",
			)
			require.NoError(t, err)
			assert.Equal(t, tt.typ, notif.Type())
		})
	}
}

func TestNotification_Getters(t *testing.T) {
	userID := uuid.NewUUID()
	typ := notification.TypeChatMention
	title := "You were mentioned"
	message := "@user mentioned you in chat"
	resourceID := "message-456"

	notif, _ := notification.NewNotification(userID, typ, title, message, resourceID)

	t.Run("ID returns non-zero UUID", func(t *testing.T) {
		assert.False(t, notif.ID().IsZero())
	})

	t.Run("UserID returns correct value", func(t *testing.T) {
		assert.Equal(t, userID, notif.UserID())
	})

	t.Run("Type returns correct value", func(t *testing.T) {
		assert.Equal(t, typ, notif.Type())
	})

	t.Run("Title returns correct value", func(t *testing.T) {
		assert.Equal(t, title, notif.Title())
	})

	t.Run("Message returns correct value", func(t *testing.T) {
		assert.Equal(t, message, notif.Message())
	})

	t.Run("ResourceID returns correct value", func(t *testing.T) {
		assert.Equal(t, resourceID, notif.ResourceID())
	})

	t.Run("CreatedAt returns non-zero time", func(t *testing.T) {
		assert.False(t, notif.CreatedAt().IsZero())
	})

	t.Run("ReadAt initially nil", func(t *testing.T) {
		assert.Nil(t, notif.ReadAt())
	})

	t.Run("ReadAt set after MarkAsRead", func(t *testing.T) {
		notif.MarkAsRead()
		assert.NotNil(t, notif.ReadAt())
	})
}
