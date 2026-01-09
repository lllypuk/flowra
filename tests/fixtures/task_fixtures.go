package fixtures

import (
	"time"

	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/domain/task"
	domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
)

const hoursInDay = 24

// CreateTaskCommandBuilder creates builder for CreateTaskCommand
type CreateTaskCommandBuilder struct {
	cmd taskapp.CreateTaskCommand
}

// NewCreateTaskCommandBuilder creates new builder with default values
func NewCreateTaskCommandBuilder() *CreateTaskCommandBuilder {
	return &CreateTaskCommandBuilder{
		cmd: taskapp.CreateTaskCommand{
			ChatID:     domainUUID.NewUUID(),
			Title:      "Test Task",
			EntityType: task.TypeTask,
			Priority:   task.PriorityMedium,
			CreatedBy:  domainUUID.NewUUID(),
		},
	}
}

// WithChatID sets chat ID
func (b *CreateTaskCommandBuilder) WithChatID(chatID domainUUID.UUID) *CreateTaskCommandBuilder {
	b.cmd.ChatID = chatID
	return b
}

// WithTitle sets title
func (b *CreateTaskCommandBuilder) WithTitle(title string) *CreateTaskCommandBuilder {
	b.cmd.Title = title
	return b
}

// WithEntityType sets entity type
func (b *CreateTaskCommandBuilder) WithEntityType(entityType task.EntityType) *CreateTaskCommandBuilder {
	b.cmd.EntityType = entityType
	return b
}

// AsBug sets entity type as Bug
func (b *CreateTaskCommandBuilder) AsBug() *CreateTaskCommandBuilder {
	b.cmd.EntityType = task.TypeBug
	return b
}

// AsEpic sets entity type as Epic
func (b *CreateTaskCommandBuilder) AsEpic() *CreateTaskCommandBuilder {
	b.cmd.EntityType = task.TypeEpic
	return b
}

// WithPriority sets priority
func (b *CreateTaskCommandBuilder) WithPriority(priority task.Priority) *CreateTaskCommandBuilder {
	b.cmd.Priority = priority
	return b
}

// WithHighPriority sets high priority
func (b *CreateTaskCommandBuilder) WithHighPriority() *CreateTaskCommandBuilder {
	b.cmd.Priority = task.PriorityHigh
	return b
}

// WithLowPriority sets low priority
func (b *CreateTaskCommandBuilder) WithLowPriority() *CreateTaskCommandBuilder {
	b.cmd.Priority = task.PriorityLow
	return b
}

// WithAssignee sets assignee
func (b *CreateTaskCommandBuilder) WithAssignee(assigneeID domainUUID.UUID) *CreateTaskCommandBuilder {
	b.cmd.AssigneeID = &assigneeID
	return b
}

// WithDueDate sets due date
func (b *CreateTaskCommandBuilder) WithDueDate(dueDate time.Time) *CreateTaskCommandBuilder {
	b.cmd.DueDate = &dueDate
	return b
}

// CreatedBy sets creator ID
func (b *CreateTaskCommandBuilder) CreatedBy(userID domainUUID.UUID) *CreateTaskCommandBuilder {
	b.cmd.CreatedBy = userID
	return b
}

// Build returns prepared command
func (b *CreateTaskCommandBuilder) Build() taskapp.CreateTaskCommand {
	return b.cmd
}

// ChangeTaskStatusCommandBuilder creates builder for task ChangeStatusCommand
type ChangeTaskStatusCommandBuilder struct {
	cmd taskapp.ChangeStatusCommand
}

// NewChangeTaskStatusCommandBuilder creates new builder
func NewChangeTaskStatusCommandBuilder(taskID domainUUID.UUID) *ChangeTaskStatusCommandBuilder {
	return &ChangeTaskStatusCommandBuilder{
		cmd: taskapp.ChangeStatusCommand{
			TaskID:    taskID,
			NewStatus: task.StatusInProgress,
			ChangedBy: domainUUID.NewUUID(),
		},
	}
}

// WithStatus sets new status
func (b *ChangeTaskStatusCommandBuilder) WithStatus(status task.Status) *ChangeTaskStatusCommandBuilder {
	b.cmd.NewStatus = status
	return b
}

// ToDone sets status to Done
func (b *ChangeTaskStatusCommandBuilder) ToDone() *ChangeTaskStatusCommandBuilder {
	b.cmd.NewStatus = task.StatusDone
	return b
}

// ToInProgress sets status to InProgress
func (b *ChangeTaskStatusCommandBuilder) ToInProgress() *ChangeTaskStatusCommandBuilder {
	b.cmd.NewStatus = task.StatusInProgress
	return b
}

// ChangedBy sets user who changed status
func (b *ChangeTaskStatusCommandBuilder) ChangedBy(userID domainUUID.UUID) *ChangeTaskStatusCommandBuilder {
	b.cmd.ChangedBy = userID
	return b
}

// Build returns prepared command
func (b *ChangeTaskStatusCommandBuilder) Build() taskapp.ChangeStatusCommand {
	return b.cmd
}

// AssignTaskCommandBuilder creates builder for AssignTaskCommand
type AssignTaskCommandBuilder struct {
	cmd taskapp.AssignTaskCommand
}

// NewAssignTaskCommandBuilder creates new builder
func NewAssignTaskCommandBuilder(taskID domainUUID.UUID) *AssignTaskCommandBuilder {
	return &AssignTaskCommandBuilder{
		cmd: taskapp.AssignTaskCommand{
			TaskID:     taskID,
			AssigneeID: nil,
			AssignedBy: domainUUID.NewUUID(),
		},
	}
}

// AssignTo sets assignee
func (b *AssignTaskCommandBuilder) AssignTo(assigneeID domainUUID.UUID) *AssignTaskCommandBuilder {
	b.cmd.AssigneeID = &assigneeID
	return b
}

// Unassign removes assignee
func (b *AssignTaskCommandBuilder) Unassign() *AssignTaskCommandBuilder {
	b.cmd.AssigneeID = nil
	return b
}

// AssignedBy sets user who assigned
func (b *AssignTaskCommandBuilder) AssignedBy(userID domainUUID.UUID) *AssignTaskCommandBuilder {
	b.cmd.AssignedBy = userID
	return b
}

// Build returns prepared command
func (b *AssignTaskCommandBuilder) Build() taskapp.AssignTaskCommand {
	return b.cmd
}

// ChangePriorityCommandBuilder creates builder for ChangePriorityCommand
type ChangePriorityCommandBuilder struct {
	cmd taskapp.ChangePriorityCommand
}

// NewChangePriorityCommandBuilder creates new builder
func NewChangePriorityCommandBuilder(taskID domainUUID.UUID) *ChangePriorityCommandBuilder {
	return &ChangePriorityCommandBuilder{
		cmd: taskapp.ChangePriorityCommand{
			TaskID:    taskID,
			Priority:  task.PriorityHigh,
			ChangedBy: domainUUID.NewUUID(),
		},
	}
}

// WithPriority sets priority
func (b *ChangePriorityCommandBuilder) WithPriority(priority task.Priority) *ChangePriorityCommandBuilder {
	b.cmd.Priority = priority
	return b
}

// ToHigh sets high priority
func (b *ChangePriorityCommandBuilder) ToHigh() *ChangePriorityCommandBuilder {
	b.cmd.Priority = task.PriorityHigh
	return b
}

// ToLow sets low priority
func (b *ChangePriorityCommandBuilder) ToLow() *ChangePriorityCommandBuilder {
	b.cmd.Priority = task.PriorityLow
	return b
}

// ChangedBy sets user who changed priority
func (b *ChangePriorityCommandBuilder) ChangedBy(userID domainUUID.UUID) *ChangePriorityCommandBuilder {
	b.cmd.ChangedBy = userID
	return b
}

// Build returns prepared command
func (b *ChangePriorityCommandBuilder) Build() taskapp.ChangePriorityCommand {
	return b.cmd
}

// SetDueDateCommandBuilder creates builder for SetDueDateCommand
type SetDueDateCommandBuilder struct {
	cmd taskapp.SetDueDateCommand
}

// NewSetDueDateCommandBuilder creates new builder
func NewSetDueDateCommandBuilder(taskID domainUUID.UUID) *SetDueDateCommandBuilder {
	tomorrow := time.Now().Add(time.Duration(hoursInDay) * time.Hour)
	return &SetDueDateCommandBuilder{
		cmd: taskapp.SetDueDateCommand{
			TaskID:    taskID,
			DueDate:   &tomorrow,
			ChangedBy: domainUUID.NewUUID(),
		},
	}
}

// WithDueDate sets due date
func (b *SetDueDateCommandBuilder) WithDueDate(dueDate time.Time) *SetDueDateCommandBuilder {
	b.cmd.DueDate = &dueDate
	return b
}

// ClearDueDate removes due date
func (b *SetDueDateCommandBuilder) ClearDueDate() *SetDueDateCommandBuilder {
	b.cmd.DueDate = nil
	return b
}

// ChangedBy sets user who changed due date
func (b *SetDueDateCommandBuilder) ChangedBy(userID domainUUID.UUID) *SetDueDateCommandBuilder {
	b.cmd.ChangedBy = userID
	return b
}

// Build returns prepared command
func (b *SetDueDateCommandBuilder) Build() taskapp.SetDueDateCommand {
	return b.cmd
}
