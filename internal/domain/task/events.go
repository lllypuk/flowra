package task

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Event types
const (
	EventTypeTaskCreated     = "task.created"
	EventTypeTaskUpdated     = "task.updated"
	EventTypeTaskDeleted     = "task.deleted"
	EventTypeStatusChanged   = "task.status_changed"
	EventTypeAssigneeChanged = "task.assignee_changed"
	EventTypePriorityChanged = "task.priority_changed"
	EventTypeDueDateChanged  = "task.due_date_changed"
	EventTypeCustomFieldSet  = "task.custom_field_set"
)

// Created событие создания задачи
type Created struct {
	event.BaseEvent

	ChatID     uuid.UUID
	Title      string
	EntityType EntityType
	Status     Status
	Priority   Priority
	AssigneeID *uuid.UUID
	DueDate    *time.Time
	CreatedBy  uuid.UUID
}

// NewTaskCreated создает новое событие TaskCreated
func NewTaskCreated(
	taskID, chatID uuid.UUID,
	title string,
	entityType EntityType,
	status Status,
	priority Priority,
	assigneeID *uuid.UUID,
	dueDate *time.Time,
	createdBy uuid.UUID,
	metadata event.Metadata,
) *Created {
	return &Created{
		BaseEvent:  event.NewBaseEvent(EventTypeTaskCreated, taskID.String(), "Task", 1, metadata),
		ChatID:     chatID,
		Title:      title,
		EntityType: entityType,
		Status:     status,
		Priority:   priority,
		AssigneeID: assigneeID,
		DueDate:    dueDate,
		CreatedBy:  createdBy,
	}
}

// Updated событие обновления задачи
type Updated struct {
	event.BaseEvent

	Title string
}

// NewTaskUpdated создает новое событие TaskUpdated
func NewTaskUpdated(
	taskID uuid.UUID,
	title string,
	metadata event.Metadata,
) *Updated {
	return &Updated{
		BaseEvent: event.NewBaseEvent(EventTypeTaskUpdated, taskID.String(), "Task", 1, metadata),
		Title:     title,
	}
}

// Deleted событие удаления задачи
type Deleted struct {
	event.BaseEvent
}

// NewTaskDeleted создает новое событие TaskDeleted
func NewTaskDeleted(
	taskID uuid.UUID,
	metadata event.Metadata,
) *Deleted {
	return &Deleted{
		BaseEvent: event.NewBaseEvent(EventTypeTaskDeleted, taskID.String(), "Task", 1, metadata),
	}
}

// StatusChanged событие изменения статуса
type StatusChanged struct {
	event.BaseEvent

	OldStatus Status
	NewStatus Status
	ChangedBy uuid.UUID
}

// NewStatusChanged создает новое событие StatusChanged
func NewStatusChanged(
	taskID uuid.UUID,
	oldStatus, newStatus Status,
	changedBy uuid.UUID,
	metadata event.Metadata,
) *StatusChanged {
	return &StatusChanged{
		BaseEvent: event.NewBaseEvent(EventTypeStatusChanged, taskID.String(), "Task", 1, metadata),
		OldStatus: oldStatus,
		NewStatus: newStatus,
		ChangedBy: changedBy,
	}
}

// AssigneeChanged событие изменения исполнителя
type AssigneeChanged struct {
	event.BaseEvent

	OldAssignee *uuid.UUID
	NewAssignee *uuid.UUID
	ChangedBy   uuid.UUID
}

// NewAssigneeChanged создает новое событие AssigneeChanged
func NewAssigneeChanged(
	taskID uuid.UUID,
	oldAssignee, newAssignee *uuid.UUID,
	changedBy uuid.UUID,
	metadata event.Metadata,
) *AssigneeChanged {
	return &AssigneeChanged{
		BaseEvent:   event.NewBaseEvent(EventTypeAssigneeChanged, taskID.String(), "Task", 1, metadata),
		OldAssignee: oldAssignee,
		NewAssignee: newAssignee,
		ChangedBy:   changedBy,
	}
}

// PriorityChanged событие изменения приоритета
type PriorityChanged struct {
	event.BaseEvent

	OldPriority Priority
	NewPriority Priority
	ChangedBy   uuid.UUID
}

// NewPriorityChanged создает новое событие PriorityChanged
func NewPriorityChanged(
	taskID uuid.UUID,
	oldPriority, newPriority Priority,
	changedBy uuid.UUID,
	metadata event.Metadata,
) *PriorityChanged {
	return &PriorityChanged{
		BaseEvent:   event.NewBaseEvent(EventTypePriorityChanged, taskID.String(), "Task", 1, metadata),
		OldPriority: oldPriority,
		NewPriority: newPriority,
		ChangedBy:   changedBy,
	}
}

// DueDateChanged событие изменения срока выполнения
type DueDateChanged struct {
	event.BaseEvent

	OldDueDate *time.Time
	NewDueDate *time.Time
	ChangedBy  uuid.UUID
}

// NewDueDateChanged создает новое событие DueDateChanged
func NewDueDateChanged(
	taskID uuid.UUID,
	oldDueDate, newDueDate *time.Time,
	changedBy uuid.UUID,
	metadata event.Metadata,
) *DueDateChanged {
	return &DueDateChanged{
		BaseEvent:  event.NewBaseEvent(EventTypeDueDateChanged, taskID.String(), "Task", 1, metadata),
		OldDueDate: oldDueDate,
		NewDueDate: newDueDate,
		ChangedBy:  changedBy,
	}
}

// CustomFieldSet событие установки кастомного поля
type CustomFieldSet struct {
	event.BaseEvent

	Key   string
	Value string
}

// NewCustomFieldSet создает новое событие CustomFieldSet
func NewCustomFieldSet(
	taskID uuid.UUID,
	key, value string,
	metadata event.Metadata,
) *CustomFieldSet {
	return &CustomFieldSet{
		BaseEvent: event.NewBaseEvent(EventTypeCustomFieldSet, taskID.String(), "Task", 1, metadata),
		Key:       key,
		Value:     value,
	}
}
