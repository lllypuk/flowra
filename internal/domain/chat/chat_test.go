package chat_test

import (
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
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
		// NewChat() applies 2 events: ChatCreated (v1) + ParticipantAdded (v2)
		assert.Equal(t, 2, c.Version())
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
		userID := uuid.NewUUID()

		err := c.ConvertToTask("Implement feature", userID)

		require.NoError(t, err)
		assert.Equal(t, chat.TypeTask, c.Type())
		assert.Equal(t, "Implement feature", c.Title())
		assert.Equal(t, "To Do", c.Status())
		assert.True(t, c.IsTyped())
	})

	t.Run("successful conversion to bug", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		userID := uuid.NewUUID()

		err := c.ConvertToBug("Fix bug", userID)

		require.NoError(t, err)
		assert.Equal(t, chat.TypeBug, c.Type())
		assert.Equal(t, "Fix bug", c.Title())
		assert.Equal(t, "New", c.Status())
	})

	t.Run("successful conversion to epic", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		userID := uuid.NewUUID()

		err := c.ConvertToEpic("Epic title", userID)

		require.NoError(t, err)
		assert.Equal(t, chat.TypeEpic, c.Type())
		assert.Equal(t, "Epic title", c.Title())
		assert.Equal(t, "Planned", c.Status())
	})

	t.Run("already typed", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeTask, true, uuid.NewUUID())
		userID := uuid.NewUUID()

		err := c.ConvertToTask("Title", userID)
		require.ErrorIs(t, err, errs.ErrInvalidState)
	})

	t.Run("empty title", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		userID := uuid.NewUUID()

		err := c.ConvertToTask("", userID)
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
		entityType, err := c.GetTaskEntityType()
		require.NoError(t, err)
		assert.Equal(t, task.TypeDiscussion, entityType)
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
			3,
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
			1,
			event.NewMetadata("", "", ""),
		)

		err := c.Apply(evt)

		require.NoError(t, err)
		assert.Equal(t, chat.TypeTask, c.Type())
	})
}

func TestChat_EventSourcing_UncommittedEvents(t *testing.T) {
	t.Run("has creation events after new", func(t *testing.T) {
		// NewChat() generates 2 events: ChatCreated + ParticipantAdded
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		events := c.GetUncommittedEvents()
		assert.Len(t, events, 2)
		_, isChatCreated := events[0].(*chat.Created)
		assert.True(t, isChatCreated, "First event should be ChatCreated")
		_, isParticipantAdded := events[1].(*chat.ParticipantAdded)
		assert.True(t, isParticipantAdded, "Second event should be ParticipantAdded")
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

// ====== Task 07.6: New Tests ======

func TestChat_ChangeStatus(t *testing.T) {
	t.Run("valid task status change", func(t *testing.T) {
		c := createTypedChat(t, chat.TypeTask, "Test Task")
		userID := uuid.NewUUID()

		err := c.ChangeStatus("In Progress", userID)

		require.NoError(t, err)
		assert.Equal(t, "In Progress", c.Status())

		events := c.GetUncommittedEvents()
		assert.Len(t, events, 1)
		assert.IsType(t, &chat.StatusChanged{}, events[0])
	})

	t.Run("valid bug status change", func(t *testing.T) {
		c := createTypedChat(t, chat.TypeBug, "Test Bug")
		userID := uuid.NewUUID()

		err := c.ChangeStatus("Investigating", userID)

		require.NoError(t, err)
		assert.Equal(t, "Investigating", c.Status())
	})

	t.Run("invalid status for task", func(t *testing.T) {
		c := createTypedChat(t, chat.TypeTask, "Test")
		userID := uuid.NewUUID()

		err := c.ChangeStatus("Fixed", userID) // Bug status

		assert.Error(t, err)
	})

	t.Run("cannot set status on discussion", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		userID := uuid.NewUUID()

		err := c.ChangeStatus("To Do", userID)

		assert.ErrorIs(t, err, errs.ErrInvalidState)
	})

	t.Run("no change if same status", func(t *testing.T) {
		c := createTypedChat(t, chat.TypeTask, "Test")
		userID := uuid.NewUUID()

		err := c.ChangeStatus("To Do", userID)

		require.NoError(t, err)
		assert.Empty(t, c.GetUncommittedEvents())
	})
}

func TestChat_AssignUser(t *testing.T) {
	t.Run("assign user", func(t *testing.T) {
		c := createTypedChat(t, chat.TypeTask, "Test")
		userID := uuid.NewUUID()
		assigneeID := uuid.NewUUID()

		err := c.AssignUser(&assigneeID, userID)

		require.NoError(t, err)
		assert.NotNil(t, c.AssigneeID())
		assert.Equal(t, assigneeID, *c.AssigneeID())

		events := c.GetUncommittedEvents()
		assert.Len(t, events, 1)
		assert.IsType(t, &chat.UserAssigned{}, events[0])
	})

	t.Run("remove assignee", func(t *testing.T) {
		c := createTypedChat(t, chat.TypeTask, "Test")
		userID := uuid.NewUUID()

		// First assign someone
		initialAssignee := uuid.NewUUID()
		_ = c.AssignUser(&initialAssignee, userID)
		c.MarkEventsAsCommitted()

		// Now remove
		err := c.AssignUser(nil, userID)

		require.NoError(t, err)
		assert.Nil(t, c.AssigneeID())

		events := c.GetUncommittedEvents()
		assert.Len(t, events, 1)
		assert.IsType(t, &chat.AssigneeRemoved{}, events[0])
	})

	t.Run("cannot assign on discussion", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		userID := uuid.NewUUID()
		assigneeID := uuid.NewUUID()

		err := c.AssignUser(&assigneeID, userID)

		assert.ErrorIs(t, err, errs.ErrInvalidState)
	})

	t.Run("no change if same assignee", func(t *testing.T) {
		c := createTypedChat(t, chat.TypeTask, "Test")
		userID := uuid.NewUUID()
		assigneeID := uuid.NewUUID()

		_ = c.AssignUser(&assigneeID, userID)
		c.MarkEventsAsCommitted()

		err := c.AssignUser(&assigneeID, userID)

		require.NoError(t, err)
		assert.Empty(t, c.GetUncommittedEvents())
	})
}

func TestChat_SetPriority(t *testing.T) {
	validPriorities := []string{"Low", "Medium", "High", "Critical"}

	for _, priority := range validPriorities {
		t.Run("set priority "+priority, func(t *testing.T) {
			c := createTypedChat(t, chat.TypeTask, "Test")
			userID := uuid.NewUUID()

			err := c.SetPriority(priority, userID)

			require.NoError(t, err)
			assert.Equal(t, priority, c.Priority())
		})
	}

	t.Run("invalid priority", func(t *testing.T) {
		c := createTypedChat(t, chat.TypeTask, "Test")
		userID := uuid.NewUUID()

		err := c.SetPriority("InvalidPriority", userID)

		assert.Error(t, err)
	})

	t.Run("cannot set priority on discussion", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		userID := uuid.NewUUID()

		err := c.SetPriority("High", userID)

		assert.ErrorIs(t, err, errs.ErrInvalidState)
	})
}

func TestChat_SetDueDate(t *testing.T) {
	t.Run("set due date", func(t *testing.T) {
		c := createTypedChat(t, chat.TypeTask, "Test")
		userID := uuid.NewUUID()
		dueDate := time.Now().Add(24 * time.Hour)

		err := c.SetDueDate(&dueDate, userID)

		require.NoError(t, err)
		assert.NotNil(t, c.DueDate())
		assert.True(t, c.DueDate().Equal(dueDate))

		events := c.GetUncommittedEvents()
		assert.Len(t, events, 1)
		assert.IsType(t, &chat.DueDateSet{}, events[0])
	})

	t.Run("remove due date", func(t *testing.T) {
		c := createTypedChat(t, chat.TypeTask, "Test")
		userID := uuid.NewUUID()

		// First set a due date
		dueDate := time.Now().Add(24 * time.Hour)
		_ = c.SetDueDate(&dueDate, userID)
		c.MarkEventsAsCommitted()

		// Now remove it
		err := c.SetDueDate(nil, userID)

		require.NoError(t, err)
		assert.Nil(t, c.DueDate())

		events := c.GetUncommittedEvents()
		assert.Len(t, events, 1)
		assert.IsType(t, &chat.DueDateRemoved{}, events[0])
	})

	t.Run("cannot set due date on discussion", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		userID := uuid.NewUUID()
		dueDate := time.Now().Add(24 * time.Hour)

		err := c.SetDueDate(&dueDate, userID)

		assert.ErrorIs(t, err, errs.ErrInvalidState)
	})
}

func TestChat_Rename(t *testing.T) {
	t.Run("rename chat", func(t *testing.T) {
		c := createTypedChat(t, chat.TypeTask, "Old Title")
		userID := uuid.NewUUID()

		err := c.Rename("New Title", userID)

		require.NoError(t, err)
		assert.Equal(t, "New Title", c.Title())

		events := c.GetUncommittedEvents()
		assert.Len(t, events, 1)
		assert.IsType(t, &chat.Renamed{}, events[0])
	})

	t.Run("empty title", func(t *testing.T) {
		c := createTypedChat(t, chat.TypeTask, "Title")
		userID := uuid.NewUUID()

		err := c.Rename("", userID)

		assert.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("cannot rename discussion", func(t *testing.T) {
		c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
		userID := uuid.NewUUID()

		err := c.Rename("New Title", userID)

		assert.ErrorIs(t, err, errs.ErrInvalidState)
	})

	t.Run("no change if same title", func(t *testing.T) {
		c := createTypedChat(t, chat.TypeTask, "Test Title")
		userID := uuid.NewUUID()

		err := c.Rename("Test Title", userID)

		require.NoError(t, err)
		assert.Empty(t, c.GetUncommittedEvents())
	})
}

func TestChat_SetSeverity(t *testing.T) {
	validSeverities := []string{"Minor", "Major", "Critical", "Blocker"}

	for _, severity := range validSeverities {
		t.Run("set severity "+severity, func(t *testing.T) {
			c := createTypedChat(t, chat.TypeBug, "Test")
			userID := uuid.NewUUID()

			err := c.SetSeverity(severity, userID)

			require.NoError(t, err)
			assert.Equal(t, severity, c.Severity())
		})
	}

	t.Run("cannot set severity on task", func(t *testing.T) {
		c := createTypedChat(t, chat.TypeTask, "Test")
		userID := uuid.NewUUID()

		err := c.SetSeverity("Critical", userID)

		assert.ErrorIs(t, err, errs.ErrInvalidState)
	})

	t.Run("invalid severity", func(t *testing.T) {
		c := createTypedChat(t, chat.TypeBug, "Test")
		userID := uuid.NewUUID()

		err := c.SetSeverity("InvalidSeverity", userID)

		assert.Error(t, err)
	})
}

func TestChat_EventSourcing_NewEvents(t *testing.T) {
	t.Run("replay StatusChanged event", func(t *testing.T) {
		c := createTypedChat(t, chat.TypeTask, "Test")

		evt := chat.NewStatusChanged(
			c.ID(),
			"To Do",
			"In Progress",
			uuid.NewUUID(),
			2,
			event.NewMetadata("", "", ""),
		)

		err := c.Apply(evt)

		require.NoError(t, err)
		assert.Equal(t, "In Progress", c.Status())
	})

	t.Run("replay UserAssigned event", func(t *testing.T) {
		c := createTypedChat(t, chat.TypeTask, "Test")
		assigneeID := uuid.NewUUID()

		evt := chat.NewUserAssigned(
			c.ID(),
			assigneeID,
			uuid.NewUUID(),
			2,
			event.NewMetadata("", "", ""),
		)

		err := c.Apply(evt)

		require.NoError(t, err)
		assert.NotNil(t, c.AssigneeID())
		assert.Equal(t, assigneeID, *c.AssigneeID())
	})

	t.Run("replay PrioritySet event", func(t *testing.T) {
		c := createTypedChat(t, chat.TypeTask, "Test")

		evt := chat.NewPrioritySet(
			c.ID(),
			"",
			"High",
			uuid.NewUUID(),
			2,
			event.NewMetadata("", "", ""),
		)

		err := c.Apply(evt)

		require.NoError(t, err)
		assert.Equal(t, "High", c.Priority())
	})
}

// Test helper
func createTypedChat(t *testing.T, chatType chat.Type, title string) *chat.Chat {
	t.Helper()
	c, _ := chat.NewChat(uuid.NewUUID(), chat.TypeDiscussion, true, uuid.NewUUID())
	userID := uuid.NewUUID()

	switch chatType {
	case chat.TypeTask:
		_ = c.ConvertToTask(title, userID)
	case chat.TypeBug:
		_ = c.ConvertToBug(title, userID)
	case chat.TypeEpic:
		_ = c.ConvertToEpic(title, userID)
	case chat.TypeDiscussion:
		// Already created as TypeDiscussion, no conversion needed
	}

	c.MarkEventsAsCommitted()
	return c
}
