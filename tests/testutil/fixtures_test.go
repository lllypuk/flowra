package testutil

import (
	"testing"
	"time"

	"github.com/lllypuk/teams-up/internal/domain/task"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateTaskCommandFixture(t *testing.T) {
	cmd := CreateTaskCommandFixture()

	assert.False(t, cmd.ChatID.IsZero())
	assert.Equal(t, "Test Task", cmd.Title)
	assert.Equal(t, task.TypeTask, cmd.EntityType)
	assert.Equal(t, task.PriorityMedium, cmd.Priority)
	assert.False(t, cmd.CreatedBy.IsZero())
	assert.Nil(t, cmd.AssigneeID)
	assert.Nil(t, cmd.DueDate)
}

func TestBuildCreateTaskCommand_WithModifiers(t *testing.T) {
	chatID := uuid.NewUUID()
	assigneeID := uuid.NewUUID()
	dueDate := time.Now().Add(24 * time.Hour)

	cmd := BuildCreateTaskCommand(
		WithChatID(chatID),
		WithTitle("Custom Task"),
		WithEntityType(task.TypeBug),
		WithPriority(task.PriorityHigh),
		WithAssignee(assigneeID),
		WithDueDate(dueDate),
	)

	assert.Equal(t, chatID, cmd.ChatID)
	assert.Equal(t, "Custom Task", cmd.Title)
	assert.Equal(t, task.TypeBug, cmd.EntityType)
	assert.Equal(t, task.PriorityHigh, cmd.Priority)
	assert.NotNil(t, cmd.AssigneeID)
	assert.Equal(t, assigneeID, *cmd.AssigneeID)
	assert.NotNil(t, cmd.DueDate)
	assert.True(t, cmd.DueDate.Equal(dueDate))
}

func TestChangeStatusCommandFixture(t *testing.T) {
	taskID := uuid.NewUUID()
	cmd := ChangeStatusCommandFixture(taskID)

	assert.Equal(t, taskID, cmd.TaskID)
	assert.Equal(t, task.StatusInProgress, cmd.NewStatus)
	assert.False(t, cmd.ChangedBy.IsZero())
}

func TestBuildChangeStatusCommand_WithModifiers(t *testing.T) {
	taskID := uuid.NewUUID()
	changedBy := uuid.NewUUID()

	cmd := BuildChangeStatusCommand(
		taskID,
		WithNewStatus(task.StatusDone),
		WithStatusChangedBy(changedBy),
	)

	assert.Equal(t, taskID, cmd.TaskID)
	assert.Equal(t, task.StatusDone, cmd.NewStatus)
	assert.Equal(t, changedBy, cmd.ChangedBy)
}

func TestAssignTaskCommandFixture(t *testing.T) {
	taskID := uuid.NewUUID()
	assigneeID := uuid.NewUUID()
	cmd := AssignTaskCommandFixture(taskID, assigneeID)

	assert.Equal(t, taskID, cmd.TaskID)
	assert.NotNil(t, cmd.AssigneeID)
	assert.Equal(t, assigneeID, *cmd.AssigneeID)
	assert.False(t, cmd.AssignedBy.IsZero())
}

func TestChangePriorityCommandFixture(t *testing.T) {
	taskID := uuid.NewUUID()
	cmd := ChangePriorityCommandFixture(taskID)

	assert.Equal(t, taskID, cmd.TaskID)
	assert.Equal(t, task.PriorityHigh, cmd.Priority)
	assert.False(t, cmd.ChangedBy.IsZero())
}

func TestSetDueDateCommandFixture(t *testing.T) {
	taskID := uuid.NewUUID()
	cmd := SetDueDateCommandFixture(taskID)

	assert.Equal(t, taskID, cmd.TaskID)
	assert.NotNil(t, cmd.DueDate)
	assert.False(t, cmd.ChangedBy.IsZero())
}
