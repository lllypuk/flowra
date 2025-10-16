package task_test

import (
	"testing"
	"time"

	"github.com/lllypuk/teams-up/internal/domain/errs"
	"github.com/lllypuk/teams-up/internal/domain/task"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTask(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		chatID := uuid.NewUUID()
		title := "Implement feature X"
		entityType := task.TypeTask

		taskEntity, err := task.NewTask(chatID, title, entityType)

		require.NoError(t, err)
		assert.Equal(t, chatID, taskEntity.ID())
		assert.Equal(t, chatID, taskEntity.ChatID())
		assert.Equal(t, title, taskEntity.Title())
		assert.Equal(t, entityType, taskEntity.Type())
		assert.Equal(t, task.StatusBacklog, taskEntity.Status())
		assert.Equal(t, task.PriorityMedium, taskEntity.Priority())
		assert.Nil(t, taskEntity.AssignedTo())
		assert.Nil(t, taskEntity.DueDate())
		assert.Empty(t, taskEntity.CustomFields())
		assert.False(t, taskEntity.CreatedAt().IsZero())
		assert.False(t, taskEntity.UpdatedAt().IsZero())
	})

	t.Run("empty chat ID", func(t *testing.T) {
		_, err := task.NewTask("", "Title", task.TypeTask)
		require.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("empty title", func(t *testing.T) {
		_, err := task.NewTask(uuid.NewUUID(), "", task.TypeTask)
		require.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("invalid entity type", func(t *testing.T) {
		_, err := task.NewTask(uuid.NewUUID(), "Title", "invalid")
		require.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("create bug", func(t *testing.T) {
		taskEntity, err := task.NewTask(uuid.NewUUID(), "Fix bug", task.TypeBug)
		require.NoError(t, err)
		assert.Equal(t, task.TypeBug, taskEntity.Type())
	})

	t.Run("create epic", func(t *testing.T) {
		taskEntity, err := task.NewTask(uuid.NewUUID(), "Epic feature", task.TypeEpic)
		require.NoError(t, err)
		assert.Equal(t, task.TypeEpic, taskEntity.Type())
	})
}

func TestTaskEntity_ChangeStatus(t *testing.T) {
	t.Run("valid transition Backlog -> To Do", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)

		err := taskEntity.ChangeStatus(task.StatusToDo)
		require.NoError(t, err)
		assert.Equal(t, task.StatusToDo, taskEntity.Status())
	})

	t.Run("valid transition To Do -> In Progress", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		taskEntity.ChangeStatus(task.StatusToDo)

		err := taskEntity.ChangeStatus(task.StatusInProgress)
		require.NoError(t, err)
		assert.Equal(t, task.StatusInProgress, taskEntity.Status())
	})

	t.Run("valid transition In Progress -> In Review", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		taskEntity.ChangeStatus(task.StatusToDo)
		taskEntity.ChangeStatus(task.StatusInProgress)

		err := taskEntity.ChangeStatus(task.StatusInReview)
		require.NoError(t, err)
		assert.Equal(t, task.StatusInReview, taskEntity.Status())
	})

	t.Run("valid transition In Review -> Done", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		taskEntity.ChangeStatus(task.StatusToDo)
		taskEntity.ChangeStatus(task.StatusInProgress)
		taskEntity.ChangeStatus(task.StatusInReview)

		err := taskEntity.ChangeStatus(task.StatusDone)
		require.NoError(t, err)
		assert.Equal(t, task.StatusDone, taskEntity.Status())
	})

	t.Run("valid backward transition In Progress -> To Do", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		taskEntity.ChangeStatus(task.StatusToDo)
		taskEntity.ChangeStatus(task.StatusInProgress)

		err := taskEntity.ChangeStatus(task.StatusToDo)
		require.NoError(t, err)
		assert.Equal(t, task.StatusToDo, taskEntity.Status())
	})

	t.Run("valid cancellation from any status", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		taskEntity.ChangeStatus(task.StatusToDo)
		taskEntity.ChangeStatus(task.StatusInProgress)

		err := taskEntity.ChangeStatus(task.StatusCancelled)
		require.NoError(t, err)
		assert.Equal(t, task.StatusCancelled, taskEntity.Status())
	})

	t.Run("invalid transition Backlog -> In Progress", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)

		err := taskEntity.ChangeStatus(task.StatusInProgress)
		require.ErrorIs(t, err, errs.ErrInvalidTransition)
		assert.Equal(t, task.StatusBacklog, taskEntity.Status())
	})

	t.Run("invalid transition Done -> To Do", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		taskEntity.ChangeStatus(task.StatusToDo)
		taskEntity.ChangeStatus(task.StatusInProgress)
		taskEntity.ChangeStatus(task.StatusInReview)
		taskEntity.ChangeStatus(task.StatusDone)

		err := taskEntity.ChangeStatus(task.StatusToDo)
		require.ErrorIs(t, err, errs.ErrInvalidTransition)
	})

	t.Run("reopening from Done to In Review", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		taskEntity.ChangeStatus(task.StatusToDo)
		taskEntity.ChangeStatus(task.StatusInProgress)
		taskEntity.ChangeStatus(task.StatusInReview)
		taskEntity.ChangeStatus(task.StatusDone)

		err := taskEntity.ChangeStatus(task.StatusInReview)
		require.NoError(t, err)
		assert.Equal(t, task.StatusInReview, taskEntity.Status())
	})

	t.Run("return from Cancelled to Backlog", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		taskEntity.ChangeStatus(task.StatusCancelled)

		err := taskEntity.ChangeStatus(task.StatusBacklog)
		require.NoError(t, err)
		assert.Equal(t, task.StatusBacklog, taskEntity.Status())
	})
}

func TestTaskEntity_Assign(t *testing.T) {
	t.Run("successful assignment", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		userID := uuid.NewUUID()

		err := taskEntity.Assign(userID)
		require.NoError(t, err)
		assert.NotNil(t, taskEntity.AssignedTo())
		assert.Equal(t, userID, *taskEntity.AssignedTo())
	})

	t.Run("empty user ID", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)

		err := taskEntity.Assign("")
		require.ErrorIs(t, err, errs.ErrInvalidInput)
		assert.Nil(t, taskEntity.AssignedTo())
	})

	t.Run("reassignment", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		user1 := uuid.NewUUID()
		user2 := uuid.NewUUID()

		taskEntity.Assign(user1)
		err := taskEntity.Assign(user2)

		require.NoError(t, err)
		assert.Equal(t, user2, *taskEntity.AssignedTo())
	})
}

func TestTaskEntity_Unassign(t *testing.T) {
	t.Run("successful unassignment", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		userID := uuid.NewUUID()
		taskEntity.Assign(userID)

		taskEntity.Unassign()
		assert.Nil(t, taskEntity.AssignedTo())
	})
}

func TestTaskEntity_SetPriority(t *testing.T) {
	t.Run("successful priority change", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)

		err := taskEntity.SetPriority(task.PriorityHigh)
		require.NoError(t, err)
		assert.Equal(t, task.PriorityHigh, taskEntity.Priority())
	})

	t.Run("invalid priority", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)

		err := taskEntity.SetPriority("invalid")
		require.ErrorIs(t, err, errs.ErrInvalidInput)
		assert.Equal(t, task.PriorityMedium, taskEntity.Priority())
	})

	t.Run("all valid priorities", func(t *testing.T) {
		priorities := []task.Priority{
			task.PriorityLow,
			task.PriorityMedium,
			task.PriorityHigh,
			task.PriorityCritical,
		}

		for _, priority := range priorities {
			taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
			err := taskEntity.SetPriority(priority)
			require.NoError(t, err)
			assert.Equal(t, priority, taskEntity.Priority())
		}
	})
}

func TestTaskEntity_SetDueDate(t *testing.T) {
	t.Run("successful due date set", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		futureDate := time.Now().Add(24 * time.Hour)

		err := taskEntity.SetDueDate(futureDate)
		require.NoError(t, err)
		assert.NotNil(t, taskEntity.DueDate())
		assert.Equal(t, futureDate.Unix(), taskEntity.DueDate().Unix())
	})

	t.Run("past date rejected", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		pastDate := time.Now().Add(-24 * time.Hour)

		err := taskEntity.SetDueDate(pastDate)
		require.ErrorIs(t, err, errs.ErrInvalidInput)
		assert.Nil(t, taskEntity.DueDate())
	})
}

func TestTaskEntity_ClearDueDate(t *testing.T) {
	t.Run("successful clear", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		futureDate := time.Now().Add(24 * time.Hour)
		taskEntity.SetDueDate(futureDate)

		taskEntity.ClearDueDate()
		assert.Nil(t, taskEntity.DueDate())
	})
}

func TestTaskEntity_SetCustomField(t *testing.T) {
	t.Run("successful set", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)

		err := taskEntity.SetCustomField("sprint", "sprint-1")
		require.NoError(t, err)

		fields := taskEntity.CustomFields()
		assert.Equal(t, "sprint-1", fields["sprint"])
	})

	t.Run("empty key rejected", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)

		err := taskEntity.SetCustomField("", "value")
		require.ErrorIs(t, err, errs.ErrInvalidInput)
	})

	t.Run("empty value removes field", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		taskEntity.SetCustomField("sprint", "sprint-1")

		err := taskEntity.SetCustomField("sprint", "")
		require.NoError(t, err)

		fields := taskEntity.CustomFields()
		_, exists := fields["sprint"]
		assert.False(t, exists)
	})

	t.Run("multiple custom fields", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)

		taskEntity.SetCustomField("sprint", "sprint-1")
		taskEntity.SetCustomField("component", "backend")
		taskEntity.SetCustomField("team", "platform")

		fields := taskEntity.CustomFields()
		assert.Len(t, fields, 3)
		assert.Equal(t, "sprint-1", fields["sprint"])
		assert.Equal(t, "backend", fields["component"])
		assert.Equal(t, "platform", fields["team"])
	})
}

func TestTaskEntity_UpdateTitle(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Old Title", task.TypeTask)

		err := taskEntity.UpdateTitle("New Title")
		require.NoError(t, err)
		assert.Equal(t, "New Title", taskEntity.Title())
	})

	t.Run("empty title rejected", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Old Title", task.TypeTask)

		err := taskEntity.UpdateTitle("")
		require.ErrorIs(t, err, errs.ErrInvalidInput)
		assert.Equal(t, "Old Title", taskEntity.Title())
	})
}

func TestTaskEntity_IsOverdue(t *testing.T) {
	t.Run("no due date - not overdue", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		assert.False(t, taskEntity.IsOverdue())
	})

	t.Run("future due date - not overdue", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		futureDate := time.Now().Add(24 * time.Hour)
		taskEntity.SetDueDate(futureDate)

		assert.False(t, taskEntity.IsOverdue())
	})

	t.Run("past due date - overdue", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		// Используем через публичный API - создаем задачу и меняем dueDate через время
		taskEntity.SetDueDate(time.Now().Add(1 * time.Millisecond))
		time.Sleep(2 * time.Millisecond)

		assert.True(t, taskEntity.IsOverdue())
	})

	t.Run("done task - not overdue even with past due date", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		taskEntity.SetDueDate(time.Now().Add(1 * time.Millisecond))
		time.Sleep(2 * time.Millisecond)

		// Переводим в Done
		taskEntity.ChangeStatus(task.StatusToDo)
		taskEntity.ChangeStatus(task.StatusInProgress)
		taskEntity.ChangeStatus(task.StatusInReview)
		taskEntity.ChangeStatus(task.StatusDone)

		assert.False(t, taskEntity.IsOverdue())
	})

	t.Run("cancelled task - not overdue even with past due date", func(t *testing.T) {
		taskEntity, _ := task.NewTask(uuid.NewUUID(), "Title", task.TypeTask)
		taskEntity.SetDueDate(time.Now().Add(1 * time.Millisecond))
		time.Sleep(2 * time.Millisecond)

		taskEntity.ChangeStatus(task.StatusCancelled)

		assert.False(t, taskEntity.IsOverdue())
	})
}
