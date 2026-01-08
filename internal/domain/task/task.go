package task

import (
	"slices"
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Aggregate represents Task aggregate с поддержкой Event Sourcing
type Aggregate struct {
	// Идентификатор aggregate
	id uuid.UUID

	// current state (восстанавливается from events)
	chatID       uuid.UUID
	title        string
	entityType   EntityType
	status       Status
	priority     Priority
	assignedTo   *uuid.UUID
	dueDate      *time.Time
	customFields map[string]string
	createdAt    time.Time
	createdBy    uuid.UUID

	// Event Sourcing fields
	version            int
	uncommittedEvents  []event.DomainEvent
	appliedEventCounts int
}

// NewTaskAggregate creates New empty aggregate
func NewTaskAggregate(id uuid.UUID) *Aggregate {
	return &Aggregate{
		id:                id,
		customFields:      make(map[string]string),
		uncommittedEvents: make([]event.DomainEvent, 0),
	}
}

// Create creates New задачу (генерирует event TaskCreated)
func (a *Aggregate) Create(
	chatID uuid.UUID,
	title string,
	entityType EntityType,
	priority Priority,
	assigneeID *uuid.UUID,
	dueDate *time.Time,
	createdBy uuid.UUID,
) error {
	// check, that task еще not создана
	if a.version > 0 {
		return errs.ErrAlreadyExists
	}

	// Creating event
	evt := NewTaskCreated(
		a.id,
		chatID,
		title,
		entityType,
		StatusToDo, // начальный status always "To Do"
		priority,
		assigneeID,
		dueDate,
		createdBy,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
		},
	)

	// Применяем event
	a.apply(evt)

	return nil
}

// ChangeStatus изменяет status tasks
func (a *Aggregate) ChangeStatus(newStatus Status, changedBy uuid.UUID) error {
	if a.version == 0 {
		return errs.ErrNotFound
	}

	// check validности перехода
	if !a.isValidStatusTransition(newStatus) {
		return errs.ErrInvalidTransition
	}

	// if status not меняется, ничего not делаем
	if a.status == newStatus {
		return nil
	}

	oldStatus := a.status

	evt := NewStatusChanged(
		a.id,
		oldStatus,
		newStatus,
		changedBy,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
		},
	)

	a.apply(evt)

	return nil
}

// Assign наvalueает исполнителя
func (a *Aggregate) Assign(assigneeID *uuid.UUID, assignedBy uuid.UUID) error {
	if a.version == 0 {
		return errs.ErrNotFound
	}

	// if assignee not меняется, ничего not делаем
	if a.assignedTo != nil && assigneeID != nil && *a.assignedTo == *assigneeID {
		return nil
	}

	if a.assignedTo == nil && assigneeID == nil {
		return nil
	}

	oldAssignee := a.assignedTo

	evt := NewAssigneeChanged(
		a.id,
		oldAssignee,
		assigneeID,
		assignedBy,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
		},
	)

	a.apply(evt)

	return nil
}

// ChangePriority изменяет приоритет
func (a *Aggregate) ChangePriority(newPriority Priority, changedBy uuid.UUID) error {
	if a.version == 0 {
		return errs.ErrNotFound
	}

	// if приоритет not меняется, ничего not делаем
	if a.priority == newPriority {
		return nil
	}

	oldPriority := a.priority

	evt := NewPriorityChanged(
		a.id,
		oldPriority,
		newPriority,
		changedBy,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
		},
	)

	a.apply(evt)

	return nil
}

// SetDueDate устанавливает or изменяет дедлайн
func (a *Aggregate) SetDueDate(newDueDate *time.Time, changedBy uuid.UUID) error {
	if a.version == 0 {
		return errs.ErrNotFound
	}

	// if date not меняется, ничего not делаем
	if a.dueDate != nil && newDueDate != nil && a.dueDate.Equal(*newDueDate) {
		return nil
	}

	if a.dueDate == nil && newDueDate == nil {
		return nil
	}

	oldDueDate := a.dueDate

	evt := NewDueDateChanged(
		a.id,
		oldDueDate,
		newDueDate,
		changedBy,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
		},
	)

	a.apply(evt)

	return nil
}

// apply применяет event to агрегату and добавляет его in uncommittedEvents
func (a *Aggregate) apply(evt event.DomainEvent) {
	a.applyChange(evt)
	a.uncommittedEvents = append(a.uncommittedEvents, evt)
}

// applyChange применяет changing from event to состоянию aggregate
func (a *Aggregate) applyChange(evt event.DomainEvent) {
	switch e := evt.(type) {
	case *Created:
		a.chatID = e.ChatID
		a.title = e.Title
		a.entityType = e.EntityType
		a.status = e.Status
		a.priority = e.Priority
		a.assignedTo = e.AssigneeID
		a.dueDate = e.DueDate
		a.createdAt = evt.OccurredAt()
		a.createdBy = e.CreatedBy

	case *StatusChanged:
		a.status = e.NewStatus

	case *AssigneeChanged:
		a.assignedTo = e.NewAssignee

	case *PriorityChanged:
		a.priority = e.NewPriority

	case *DueDateChanged:
		a.dueDate = e.NewDueDate

	case *CustomFieldSet:
		if e.Value == "" {
			delete(a.customFields, e.Key)
		} else {
			a.customFields[e.Key] = e.Value
		}
	}

	a.version++
	a.appliedEventCounts++
}

// ReplayEvents восстанавливает state aggregate from events
func (a *Aggregate) ReplayEvents(events []event.DomainEvent) {
	for _, evt := range events {
		a.applyChange(evt)
	}
}

// UncommittedEvents returns event, которые еще not были savены
func (a *Aggregate) UncommittedEvents() []event.DomainEvent {
	return a.uncommittedEvents
}

// MarkEventsAsCommitted clears list неsavенных events
func (a *Aggregate) MarkEventsAsCommitted() {
	a.uncommittedEvents = make([]event.DomainEvent, 0)
}

// Version returns current version aggregate
func (a *Aggregate) Version() int {
	return a.version
}

// isValidStatusTransition validates перехода between статусами
func (a *Aggregate) isValidStatusTransition(newStatus Status) bool {
	// if status not меняется, it is always validно
	if a.status == newStatus {
		return true
	}

	// from Cancelled можно вернуться only in Backlog
	if a.status == StatusCancelled {
		return newStatus == StatusBacklog
	}

	// from Done можно вернуться in InReview (reopening)
	if a.status == StatusDone {
		return newStatus == StatusInReview || newStatus == StatusCancelled
	}

	// Стандартные переходы вbefore
	transitions := map[Status][]Status{ //nolint:exhaustive // StatusDone and StatusCancelled обрабатываются above
		StatusBacklog:    {StatusToDo, StatusCancelled},
		StatusToDo:       {StatusInProgress, StatusBacklog, StatusCancelled},
		StatusInProgress: {StatusInReview, StatusToDo, StatusCancelled},
		StatusInReview:   {StatusDone, StatusInProgress, StatusCancelled},
	}

	allowedStatuses, exists := transitions[a.status]
	if !exists {
		return false
	}

	return slices.Contains(allowedStatuses, newStatus)
}

// Getters

// ID returns ID aggregate
func (a *Aggregate) ID() uuid.UUID { return a.id }

// ChatID returns ID chat
func (a *Aggregate) ChatID() uuid.UUID { return a.chatID }

// Title returns заголовок
func (a *Aggregate) Title() string { return a.title }

// EntityType returns type сущности
func (a *Aggregate) EntityType() EntityType { return a.entityType }

// Status returns status
func (a *Aggregate) Status() Status { return a.status }

// Priority returns приоритет
func (a *Aggregate) Priority() Priority { return a.priority }

// AssignedTo returns ID наvalueенного user
func (a *Aggregate) AssignedTo() *uuid.UUID { return a.assignedTo }

// DueDate returns дедлайн
func (a *Aggregate) DueDate() *time.Time { return a.dueDate }

// CreatedAt returns creation time
func (a *Aggregate) CreatedAt() time.Time { return a.createdAt }

// CreatedBy returns creator ID
func (a *Aggregate) CreatedBy() uuid.UUID { return a.createdBy }
