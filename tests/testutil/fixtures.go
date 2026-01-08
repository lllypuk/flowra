package testutil

import (
	"time"

	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

const dayDuration = 24 * time.Hour

// CreateTaskCommandFixture returns valid command for creating tasks
func CreateTaskCommandFixture() taskapp.CreateTaskCommand {
	return taskapp.CreateTaskCommand{
		ChatID:     uuid.NewUUID(),
		Title:      "Test Task",
		EntityType: task.TypeTask,
		Priority:   task.PriorityMedium,
		CreatedBy:  uuid.NewUUID(),
	}
}

// WithChatID modifies ChatID
func WithChatID(chatID uuid.UUID) func(*taskapp.CreateTaskCommand) {
	return func(cmd *taskapp.CreateTaskCommand) {
		cmd.ChatID = chatID
	}
}

// WithTitle modifies title
func WithTitle(title string) func(*taskapp.CreateTaskCommand) {
	return func(cmd *taskapp.CreateTaskCommand) {
		cmd.Title = title
	}
}

// WithEntityType modifies entity type
func WithEntityType(entityType task.EntityType) func(*taskapp.CreateTaskCommand) {
	return func(cmd *taskapp.CreateTaskCommand) {
		cmd.EntityType = entityType
	}
}

// WithPriority modifies priority
func WithPriority(priority task.Priority) func(*taskapp.CreateTaskCommand) {
	return func(cmd *taskapp.CreateTaskCommand) {
		cmd.Priority = priority
	}
}

// WithAssignee adds assignee
func WithAssignee(assigneeID uuid.UUID) func(*taskapp.CreateTaskCommand) {
	return func(cmd *taskapp.CreateTaskCommand) {
		cmd.AssigneeID = &assigneeID
	}
}

// WithDueDate adds deadline
func WithDueDate(dueDate time.Time) func(*taskapp.CreateTaskCommand) {
	return func(cmd *taskapp.CreateTaskCommand) {
		cmd.DueDate = &dueDate
	}
}

// WithCreatedBy modifies created by
func WithCreatedBy(createdBy uuid.UUID) func(*taskapp.CreateTaskCommand) {
	return func(cmd *taskapp.CreateTaskCommand) {
		cmd.CreatedBy = createdBy
	}
}

// BuildCreateTaskCommand creates command with modifiers
func BuildCreateTaskCommand(modifiers ...func(*taskapp.CreateTaskCommand)) taskapp.CreateTaskCommand {
	cmd := CreateTaskCommandFixture()
	for _, modifier := range modifiers {
		modifier(&cmd)
	}
	return cmd
}

// ChangeStatusCommandFixture returns valid command for changing status
func ChangeStatusCommandFixture(taskID uuid.UUID) taskapp.ChangeStatusCommand {
	return taskapp.ChangeStatusCommand{
		TaskID:    taskID,
		NewStatus: task.StatusInProgress,
		ChangedBy: uuid.NewUUID(),
	}
}

// WithNewStatus modifies New status
func WithNewStatus(status task.Status) func(*taskapp.ChangeStatusCommand) {
	return func(cmd *taskapp.ChangeStatusCommand) {
		cmd.NewStatus = status
	}
}

// WithStatusChangedBy modifies changed by
func WithStatusChangedBy(changedBy uuid.UUID) func(*taskapp.ChangeStatusCommand) {
	return func(cmd *taskapp.ChangeStatusCommand) {
		cmd.ChangedBy = changedBy
	}
}

// BuildChangeStatusCommand creates command with modifiers
func BuildChangeStatusCommand(
	taskID uuid.UUID,
	modifiers ...func(*taskapp.ChangeStatusCommand),
) taskapp.ChangeStatusCommand {
	cmd := ChangeStatusCommandFixture(taskID)
	for _, modifier := range modifiers {
		modifier(&cmd)
	}
	return cmd
}

// AssignTaskCommandFixture returns valid command for assigning
func AssignTaskCommandFixture(taskID uuid.UUID, assigneeID uuid.UUID) taskapp.AssignTaskCommand {
	return taskapp.AssignTaskCommand{
		TaskID:     taskID,
		AssigneeID: &assigneeID,
		AssignedBy: uuid.NewUUID(),
	}
}

// WithAssigneeID modifies assignee ID
func WithAssigneeID(assigneeID *uuid.UUID) func(*taskapp.AssignTaskCommand) {
	return func(cmd *taskapp.AssignTaskCommand) {
		cmd.AssigneeID = assigneeID
	}
}

// WithAssignedBy modifies assigned by
func WithAssignedBy(assignedBy uuid.UUID) func(*taskapp.AssignTaskCommand) {
	return func(cmd *taskapp.AssignTaskCommand) {
		cmd.AssignedBy = assignedBy
	}
}

// BuildAssignTaskCommand creates command with modifiers
func BuildAssignTaskCommand(
	taskID uuid.UUID,
	assigneeID uuid.UUID,
	modifiers ...func(*taskapp.AssignTaskCommand),
) taskapp.AssignTaskCommand {
	cmd := AssignTaskCommandFixture(taskID, assigneeID)
	for _, modifier := range modifiers {
		modifier(&cmd)
	}
	return cmd
}

// ChangePriorityCommandFixture returns valid command for changing priority
func ChangePriorityCommandFixture(taskID uuid.UUID) taskapp.ChangePriorityCommand {
	return taskapp.ChangePriorityCommand{
		TaskID:    taskID,
		Priority:  task.PriorityHigh,
		ChangedBy: uuid.NewUUID(),
	}
}

// WithPriorityValue modifies priority
func WithPriorityValue(priority task.Priority) func(*taskapp.ChangePriorityCommand) {
	return func(cmd *taskapp.ChangePriorityCommand) {
		cmd.Priority = priority
	}
}

// WithPriorityChangedBy modifies changed by
func WithPriorityChangedBy(changedBy uuid.UUID) func(*taskapp.ChangePriorityCommand) {
	return func(cmd *taskapp.ChangePriorityCommand) {
		cmd.ChangedBy = changedBy
	}
}

// BuildChangePriorityCommand creates command with modifiers
func BuildChangePriorityCommand(
	taskID uuid.UUID,
	modifiers ...func(*taskapp.ChangePriorityCommand),
) taskapp.ChangePriorityCommand {
	cmd := ChangePriorityCommandFixture(taskID)
	for _, modifier := range modifiers {
		modifier(&cmd)
	}
	return cmd
}

// SetDueDateCommandFixture returns valid command for setting deadline
func SetDueDateCommandFixture(taskID uuid.UUID) taskapp.SetDueDateCommand {
	tomorrow := time.Now().Add(dayDuration)
	return taskapp.SetDueDateCommand{
		TaskID:    taskID,
		DueDate:   &tomorrow,
		ChangedBy: uuid.NewUUID(),
	}
}

// WithDueDateValue modifies due date
func WithDueDateValue(dueDate *time.Time) func(*taskapp.SetDueDateCommand) {
	return func(cmd *taskapp.SetDueDateCommand) {
		cmd.DueDate = dueDate
	}
}

// WithDueDateChangedBy modifies changed by
func WithDueDateChangedBy(changedBy uuid.UUID) func(*taskapp.SetDueDateCommand) {
	return func(cmd *taskapp.SetDueDateCommand) {
		cmd.ChangedBy = changedBy
	}
}

// BuildSetDueDateCommand creates command with modifiers
func BuildSetDueDateCommand(
	taskID uuid.UUID,
	modifiers ...func(*taskapp.SetDueDateCommand),
) taskapp.SetDueDateCommand {
	cmd := SetDueDateCommandFixture(taskID)
	for _, modifier := range modifiers {
		modifier(&cmd)
	}
	return cmd
}

// Example usage:
// cmd := testutil.BuildCreateTaskCommand(
//     testutil.WithTitle("Custom Title"),
//     testutil.WithPriority(task.PriorityHigh),
//     testutil.WithAssignee(userID),
// )
