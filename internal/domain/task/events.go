package task

import (
	"encoding/json"
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

// Created event creating tasks
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

// NewTaskCreated creates new event TaskCreated
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

// Updated event updating tasks
type Updated struct {
	event.BaseEvent

	Title string
}

// NewTaskUpdated creates new event TaskUpdated
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

// Deleted event removing tasks
type Deleted struct {
	event.BaseEvent
}

// NewTaskDeleted creates new event TaskDeleted
func NewTaskDeleted(
	taskID uuid.UUID,
	metadata event.Metadata,
) *Deleted {
	return &Deleted{
		BaseEvent: event.NewBaseEvent(EventTypeTaskDeleted, taskID.String(), "Task", 1, metadata),
	}
}

// StatusChanged event changing status
type StatusChanged struct {
	event.BaseEvent

	OldStatus Status
	NewStatus Status
	ChangedBy uuid.UUID
}

// NewStatusChanged creates new event StatusChanged
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

// Payload returns the event payload for WebSocket broadcasting
func (e *StatusChanged) Payload() json.RawMessage {
	payload := map[string]any{
		"task_id": e.AggregateID(),
		"changes": map[string]any{
			"status": map[string]any{
				"old": string(e.OldStatus),
				"new": string(e.NewStatus),
			},
		},
		"changed_by": e.ChangedBy.String(),
		"version":    e.Version(),
	}
	data, _ := json.Marshal(payload)
	return data
}

// AssigneeChanged event changing ispolnitelya
type AssigneeChanged struct {
	event.BaseEvent

	OldAssignee *uuid.UUID
	NewAssignee *uuid.UUID
	ChangedBy   uuid.UUID
}

// NewAssigneeChanged creates new event AssigneeChanged
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

// Payload returns the event payload for WebSocket broadcasting
func (e *AssigneeChanged) Payload() json.RawMessage {
	var oldAssignee, newAssignee *string
	if e.OldAssignee != nil {
		old := e.OldAssignee.String()
		oldAssignee = &old
	}
	if e.NewAssignee != nil {
		newVal := e.NewAssignee.String()
		newAssignee = &newVal
	}

	payload := map[string]any{
		"task_id": e.AggregateID(),
		"changes": map[string]any{
			"assignee": map[string]any{
				"old": oldAssignee,
				"new": newAssignee,
			},
		},
		"changed_by": e.ChangedBy.String(),
		"version":    e.Version(),
	}
	data, _ := json.Marshal(payload)
	return data
}

// PriorityChanged event changing priority
type PriorityChanged struct {
	event.BaseEvent

	OldPriority Priority
	NewPriority Priority
	ChangedBy   uuid.UUID
}

// NewPriorityChanged creates new event PriorityChanged
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

// Payload returns the event payload for WebSocket broadcasting
func (e *PriorityChanged) Payload() json.RawMessage {
	payload := map[string]any{
		"task_id": e.AggregateID(),
		"changes": map[string]any{
			"priority": map[string]any{
				"old": string(e.OldPriority),
				"new": string(e.NewPriority),
			},
		},
		"changed_by": e.ChangedBy.String(),
		"version":    e.Version(),
	}
	data, _ := json.Marshal(payload)
	return data
}

// DueDateChanged event changing sroka vypolneniya
type DueDateChanged struct {
	event.BaseEvent

	OldDueDate *time.Time
	NewDueDate *time.Time
	ChangedBy  uuid.UUID
}

// NewDueDateChanged creates new event DueDateChanged
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

// Payload returns the event payload for WebSocket broadcasting
func (e *DueDateChanged) Payload() json.RawMessage {
	var oldDate, newDate *string
	if e.OldDueDate != nil {
		old := e.OldDueDate.Format("2006-01-02")
		oldDate = &old
	}
	if e.NewDueDate != nil {
		newVal := e.NewDueDate.Format("2006-01-02")
		newDate = &newVal
	}

	payload := map[string]any{
		"task_id": e.AggregateID(),
		"changes": map[string]any{
			"due_date": map[string]any{
				"old": oldDate,
				"new": newDate,
			},
		},
		"changed_by": e.ChangedBy.String(),
		"version":    e.Version(),
	}
	data, _ := json.Marshal(payload)
	return data
}

// CustomFieldSet event setting kastomnogo fields
type CustomFieldSet struct {
	event.BaseEvent

	Key   string
	Value string
}

// NewCustomFieldSet creates new event CustomFieldSet
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
