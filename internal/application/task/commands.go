package task

import (
	"time"

	"github.com/lllypuk/teams-up/internal/domain/task"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
)

// CreateTaskCommand содержит данные для создания задачи
type CreateTaskCommand struct {
	ChatID     uuid.UUID
	Title      string
	EntityType task.EntityType // "task", "bug", "epic"
	Priority   task.Priority   // optional, по умолчанию "Medium"
	AssigneeID *uuid.UUID      // optional
	DueDate    *time.Time      // optional
	CreatedBy  uuid.UUID
}

// ChangeStatusCommand содержит данные для изменения статуса
type ChangeStatusCommand struct {
	TaskID    uuid.UUID
	NewStatus task.Status // "Backlog", "To Do", "In Progress", "In Review", "Done", "Cancelled"
	ChangedBy uuid.UUID
}

// AssignTaskCommand содержит данные для назначения исполнителя
type AssignTaskCommand struct {
	TaskID     uuid.UUID
	AssigneeID *uuid.UUID // nil = снять assignee
	AssignedBy uuid.UUID
}

// ChangePriorityCommand содержит данные для изменения приоритета
type ChangePriorityCommand struct {
	TaskID    uuid.UUID
	Priority  task.Priority // "Low", "Medium", "High", "Critical"
	ChangedBy uuid.UUID
}

// SetDueDateCommand содержит данные для установки дедлайна
type SetDueDateCommand struct {
	TaskID    uuid.UUID
	DueDate   *time.Time // nil = снять дедлайн
	ChangedBy uuid.UUID
}

// UpdateTitleCommand содержит данные для обновления заголовка
type UpdateTitleCommand struct {
	TaskID    uuid.UUID
	Title     string
	UpdatedBy uuid.UUID
}
