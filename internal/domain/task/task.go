package task

import (
	"slices"
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Aggregate представляет Task aggregate с поддержкой Event Sourcing
type Aggregate struct {
	// Идентификатор aggregate
	id uuid.UUID

	// Текущее состояние (восстанавливается из событий)
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

	// Event Sourcing поля
	version            int
	uncommittedEvents  []event.DomainEvent
	appliedEventCounts int
}

// NewTaskAggregate создает новый пустой агрегат
func NewTaskAggregate(id uuid.UUID) *Aggregate {
	return &Aggregate{
		id:                id,
		customFields:      make(map[string]string),
		uncommittedEvents: make([]event.DomainEvent, 0),
	}
}

// Create создает новую задачу (генерирует событие TaskCreated)
func (a *Aggregate) Create(
	chatID uuid.UUID,
	title string,
	entityType EntityType,
	priority Priority,
	assigneeID *uuid.UUID,
	dueDate *time.Time,
	createdBy uuid.UUID,
) error {
	// Проверка, что задача еще не создана
	if a.version > 0 {
		return errs.ErrAlreadyExists
	}

	// Создаем событие
	evt := NewTaskCreated(
		a.id,
		chatID,
		title,
		entityType,
		StatusToDo, // начальный статус всегда "To Do"
		priority,
		assigneeID,
		dueDate,
		createdBy,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
		},
	)

	// Применяем событие
	a.apply(evt)

	return nil
}

// ChangeStatus изменяет статус задачи
func (a *Aggregate) ChangeStatus(newStatus Status, changedBy uuid.UUID) error {
	if a.version == 0 {
		return errs.ErrNotFound
	}

	// Проверка валидности перехода
	if !a.isValidStatusTransition(newStatus) {
		return errs.ErrInvalidTransition
	}

	// Если статус не меняется, ничего не делаем
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

// Assign назначает исполнителя
func (a *Aggregate) Assign(assigneeID *uuid.UUID, assignedBy uuid.UUID) error {
	if a.version == 0 {
		return errs.ErrNotFound
	}

	// Если assignee не меняется, ничего не делаем
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

	// Если приоритет не меняется, ничего не делаем
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

// SetDueDate устанавливает или изменяет дедлайн
func (a *Aggregate) SetDueDate(newDueDate *time.Time, changedBy uuid.UUID) error {
	if a.version == 0 {
		return errs.ErrNotFound
	}

	// Если дата не меняется, ничего не делаем
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

// apply применяет событие к агрегату и добавляет его в uncommittedEvents
func (a *Aggregate) apply(evt event.DomainEvent) {
	a.applyChange(evt)
	a.uncommittedEvents = append(a.uncommittedEvents, evt)
}

// applyChange применяет изменения из события к состоянию агрегата
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

// ReplayEvents восстанавливает состояние агрегата из событий
func (a *Aggregate) ReplayEvents(events []event.DomainEvent) {
	for _, evt := range events {
		a.applyChange(evt)
	}
}

// UncommittedEvents возвращает события, которые еще не были сохранены
func (a *Aggregate) UncommittedEvents() []event.DomainEvent {
	return a.uncommittedEvents
}

// MarkEventsAsCommitted очищает список несохраненных событий
func (a *Aggregate) MarkEventsAsCommitted() {
	a.uncommittedEvents = make([]event.DomainEvent, 0)
}

// Version возвращает текущую версию агрегата
func (a *Aggregate) Version() int {
	return a.version
}

// isValidStatusTransition проверяет валидность перехода между статусами
func (a *Aggregate) isValidStatusTransition(newStatus Status) bool {
	// Если статус не меняется, это всегда валидно
	if a.status == newStatus {
		return true
	}

	// Из Cancelled можно вернуться только в Backlog
	if a.status == StatusCancelled {
		return newStatus == StatusBacklog
	}

	// Из Done можно вернуться в InReview (reopening)
	if a.status == StatusDone {
		return newStatus == StatusInReview || newStatus == StatusCancelled
	}

	// Стандартные переходы вперед
	transitions := map[Status][]Status{ //nolint:exhaustive // StatusDone и StatusCancelled обрабатываются выше
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

// ID возвращает ID агрегата
func (a *Aggregate) ID() uuid.UUID { return a.id }

// ChatID возвращает ID чата
func (a *Aggregate) ChatID() uuid.UUID { return a.chatID }

// Title возвращает заголовок
func (a *Aggregate) Title() string { return a.title }

// EntityType возвращает тип сущности
func (a *Aggregate) EntityType() EntityType { return a.entityType }

// Status возвращает статус
func (a *Aggregate) Status() Status { return a.status }

// Priority возвращает приоритет
func (a *Aggregate) Priority() Priority { return a.priority }

// AssignedTo возвращает ID назначенного пользователя
func (a *Aggregate) AssignedTo() *uuid.UUID { return a.assignedTo }

// DueDate возвращает дедлайн
func (a *Aggregate) DueDate() *time.Time { return a.dueDate }

// CreatedAt возвращает время создания
func (a *Aggregate) CreatedAt() time.Time { return a.createdAt }

// CreatedBy возвращает ID создателя
func (a *Aggregate) CreatedBy() uuid.UUID { return a.createdBy }
