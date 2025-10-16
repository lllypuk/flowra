package chat_test

import (
	"testing"
	"time"

	"github.com/lllypuk/teams-up/internal/domain/chat"
	"github.com/lllypuk/teams-up/internal/domain/errs"
	"github.com/lllypuk/teams-up/internal/domain/event"
	"github.com/lllypuk/teams-up/internal/domain/task"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChat(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		createdBy := uuid.NewUUID()

		c, err := chat.NewChat(workspaceID, chat.TypeDiscussion, true, createdBy)

		require.NoError(t, err)
		assert.False(t, c.ID().IsZero())
		assert.Equal(t, workspaceID, c.WorkspaceID())
		assert.Equal(t, chat.TypeDiscussion, c.Type())
		assert.True(t, c.IsPublic())
		assert.Equal(t, createdBy, c.CreatedBy())
		assert.False(t, c.CreatedAt().IsZero())
		assert.Equal(t, 0, c.Version())
		assert.Len(t, c.Participants(), 1)
		assert.True(t, c.IsParticipantAdmin(createdBy))
	})

	t.Run("empty workspace ID", func(t *testing.T) {
		_, err := chat.NewChat("", chat.TypeDiscussion, true, uuid.NewUUID())
		require.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("empty created by", func(t *testing.T) {
		_, err := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, "")
		require.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("invalid chat type", func(t *testing.T) {
		_, err := chat.NewChat(uuid.NewUUID(), "invalid", true, uuid.NewUUID())
		require.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("create task chat", func(t *testing.T) {
		c, err := chat.NewChat(uuid.NewUUID(), chat.TypeTask, false, uuid.NewUUID())
		require.NoError(t, err)
		assert.Equal(t, chat.TypeTask, c.Type())
		assert.False(t, c.IsPublic())
		assert.True(t, c.IsTyped())
	})
}

func TestChat_AddParticipant(t *testing.T) {
	t.Run("successful add", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		userID := uuid.NewUUID()

		err := c.AddParticipant(userID, chat.RoleMember)

		require.NoError(t, err)
		assert.True(t, c.HasParticipant(userID))
		assert.False(t, c.IsParticipantAdmin(userID))
		assert.Len(t, c.Participants(), 2)
	})

	t.Run("add admin", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		userID := uuid.NewUUID()

		err := c.AddParticipant(userID, chat.RoleAdmin)

		require.NoError(t, err)
		assert.True(t, c.IsParticipantAdmin(userID))
	})

	t.Run("empty user ID", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		err := c.AddParticipant("", chat.RoleMember)
		require.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("already participant", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		userID := uuid.NewUUID()
		c.AddParticipant(userID, chat.RoleMember)

		err := c.AddParticipant(userID, chat.RoleMember)
		require.ErrorIs(t, err, errs.ErrAlreadyExists)
	})
}

func TestChat_RemoveParticipant(t *testing.T) {
	t.Run("successful remove", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		userID := uuid.NewUUID()
		c.AddParticipant(userID, chat.RoleMember)

		err := c.RemoveParticipant(userID)

		require.NoError(t, err)
		assert.False(t, c.HasParticipant(userID))
		assert.Len(t, c.Participants(), 1)
	})

	t.Run("cannot remove creator", func(t *testing.T) {
		createdBy := uuid.NewUUID()
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, createdBy)

		err := c.RemoveParticipant(createdBy)
		require.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("not found", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		err := c.RemoveParticipant(uuid.NewUUID())
		require.ErrorIs(t, err, errs.ErrNotFound)
	})
}

func TestChat_ConvertToTask(t *testing.T) {
	t.Run("successful conversion to task", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())

		err := c.ConvertToTask(chat.TypeTask, "Implement feature")

		require.NoError(t, err)
		assert.Equal(t, chat.TypeTask, c.Type())
		assert.True(t, c.IsTyped())
	})

	t.Run("successful conversion to bug", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())

		err := c.ConvertToTask(chat.TypeBug, "Fix bug")

		require.NoError(t, err)
		assert.Equal(t, chat.TypeBug, c.Type())
	})

	t.Run("already typed", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeTask, true, uuid.NewUUID())

		err := c.ConvertToTask(chat.TypeBug, "Title")
		require.ErrorIs(t, err, errs.ErrInvalidState)
	})

	t.Run("invalid new type", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())

		err := c.ConvertToTask(chat.TypeDiscussion, "Title")
		require.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("empty title", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())

		err := c.ConvertToTask(chat.TypeTask, "")
		require.ErrorIs(t, err, errs.ErrInvalidInput)
	})
}

func TestChat_GetTaskEntityType(t *testing.T) {
	t.Run("task type", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeTask, true, uuid.NewUUID())
		entityType, err := c.GetTaskEntityType()
		require.NoError(t, err)
		assert.Equal(t, task.TypeTask, entityType)
	})

	t.Run("bug type", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeBug, true, uuid.NewUUID())
		entityType, err := c.GetTaskEntityType()
		require.NoError(t, err)
		assert.Equal(t, task.TypeBug, entityType)
	})

	t.Run("epic type", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeEpic, true, uuid.NewUUID())
		entityType, err := c.GetTaskEntityType()
		require.NoError(t, err)
		assert.Equal(t, task.TypeEpic, entityType)
	})

	t.Run("discussion has no task type", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		_, err := c.GetTaskEntityType()
		require.ErrorIs(t, err, errs.ErrInvalidState)
	})
}

func TestChat_FindParticipant(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		createdBy := uuid.NewUUID()
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, createdBy)

		p := c.FindParticipant(createdBy)

		require.NotNil(t, p)
		assert.Equal(t, createdBy, p.UserID())
		assert.True(t, p.IsAdmin())
	})

	t.Run("not found", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		p := c.FindParticipant(uuid.NewUUID())
		assert.Nil(t, p)
	})
}

// Event Sourcing Tests

func TestChat_EventSourcing_Apply(t *testing.T) {
	t.Run("apply ChatCreated", func(t *testing.T) {
		c := &chat.Chat{}
		chatID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		createdBy := uuid.NewUUID()

		evt := chat.NewChatCreated(
			chatID,
			workspaceID,
			chat.TypeDiscussion,
			true,
			createdBy,
			time.Now(),
			event.NewMetadata("", "", ""),
		)

		err := c.Apply(evt)

		require.NoError(t, err)
		assert.Equal(t, chatID, c.ID())
		assert.Equal(t, workspaceID, c.WorkspaceID())
		assert.Equal(t, chat.TypeDiscussion, c.Type())
		assert.True(t, c.IsPublic())
		assert.Equal(t, createdBy, c.CreatedBy())
	})

	t.Run("apply ParticipantAdded", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		userID := uuid.NewUUID()

		evt := chat.NewParticipantAdded(
			c.ID(),
			userID,
			chat.RoleMember,
			time.Now(),
			event.NewMetadata("", "", ""),
		)

		err := c.Apply(evt)

		require.NoError(t, err)
		assert.True(t, c.HasParticipant(userID))
	})

	t.Run("apply ChatTypeChanged", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())

		evt := chat.NewChatTypeChanged(
			c.ID(),
			chat.TypeDiscussion,
			chat.TypeTask,
			"Title",
			event.NewMetadata("", "", ""),
		)

		err := c.Apply(evt)

		require.NoError(t, err)
		assert.Equal(t, chat.TypeTask, c.Type())
	})
}

func TestChat_EventSourcing_UncommittedEvents(t *testing.T) {
	t.Run("no uncommitted events initially", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		events := c.GetUncommittedEvents()
		assert.Empty(t, events)
	})

	t.Run("mark events as committed", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		c.MarkEventsAsCommitted()
		events := c.GetUncommittedEvents()
		assert.Empty(t, events)
	})
}

func TestParticipant(t *testing.T) {
	t.Run("create participant", func(t *testing.T) {
		userID := uuid.NewUUID()
		p := chat.NewParticipant(userID, chat.RoleMember)

		assert.Equal(t, userID, p.UserID())
		assert.Equal(t, chat.RoleMember, p.Role())
		assert.False(t, p.JoinedAt().IsZero())
		assert.False(t, p.IsAdmin())
	})

	t.Run("admin participant", func(t *testing.T) {
		p := chat.NewParticipant(uuid.NewUUID(), chat.RoleAdmin)
		assert.True(t, p.IsAdmin())
	})
}
