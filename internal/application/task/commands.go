package task

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// CreateTaskCommand contains data for creating tasks
type CreateTaskCommand struct {
	ChatID     uuid.UUID
	Title      string
	EntityType task.EntityType // "task", "bug", "epic"
	Priority   task.Priority   // optional, by default "Medium"
	AssigneeID *uuid.UUID      // optional
	DueDate    *time.Time      // optional
	CreatedBy  uuid.UUID
}

// ChangeStatusCommand contains data for changing status
type ChangeStatusCommand struct {
	TaskID    uuid.UUID
	NewStatus task.Status // "Backlog", "To Do", "In Progress", "In Review", "Done", "Cancelled"
	ChangedBy uuid.UUID
}

// AssignTaskCommand contains data for наvalueения исполнителя
type AssignTaskCommand struct {
	TaskID     uuid.UUID
	AssigneeID *uuid.UUID // nil = снять assignee
	AssignedBy uuid.UUID
}

// ChangePriorityCommand contains data for changing priority
type ChangePriorityCommand struct {
	TaskID    uuid.UUID
	Priority  task.Priority // "Low", "Medium", "High", "Critical"
	ChangedBy uuid.UUID
}

// SetDueDateCommand contains data for setting deadline
type SetDueDateCommand struct {
	TaskID    uuid.UUID
	DueDate   *time.Time // nil = снять дедлайн
	ChangedBy uuid.UUID
}

// UpdateTitleCommand contains data for updating заголовка
type UpdateTitleCommand struct {
	TaskID    uuid.UUID
	Title     string
	UpdatedBy uuid.UUID
}
