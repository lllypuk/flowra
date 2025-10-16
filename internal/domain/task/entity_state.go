package task

import "github.com/lllypuk/teams-up/internal/domain/errs"

// EntityType представляет тип сущности
type EntityType string

const (
	// TypeTask обычная задача
	TypeTask EntityType = "task"
	// TypeBug баг/дефект
	TypeBug EntityType = "bug"
	// TypeEpic эпик (большая задача)
	TypeEpic EntityType = "epic"
)

// Status представляет статус задачи
type Status string

const (
	// StatusBacklog задача в бэклоге
	StatusBacklog Status = "Backlog"
	// StatusToDo задача готова к работе
	StatusToDo Status = "To Do"
	// StatusInProgress задача в работе
	StatusInProgress Status = "In Progress"
	// StatusInReview задача на ревью
	StatusInReview Status = "In Review"
	// StatusDone задача завершена
	StatusDone Status = "Done"
	// StatusCancelled задача отменена
	StatusCancelled Status = "Cancelled"
)

// Priority представляет приоритет задачи
type Priority string

const (
	// PriorityLow низкий приоритет
	PriorityLow Priority = "Low"
	// PriorityMedium средний приоритет
	PriorityMedium Priority = "Medium"
	// PriorityHigh высокий приоритет
	PriorityHigh Priority = "High"
	// PriorityCritical критический приоритет
	PriorityCritical Priority = "Critical"
)

// EntityState представляет состояние типизированной сущности
type EntityState struct {
	entityType EntityType
	status     Status
	priority   Priority
}

// NewEntityState создает новое состояние сущности
func NewEntityState(entityType EntityType) (*EntityState, error) {
	if !isValidEntityType(entityType) {
		return nil, errs.ErrInvalidInput
	}

	return &EntityState{
		entityType: entityType,
		status:     StatusBacklog,
		priority:   PriorityMedium,
	}, nil
}

// ChangeStatus изменяет статус с валидацией перехода
func (s *EntityState) ChangeStatus(newStatus Status) error {
	if !isValidStatus(newStatus) {
		return errs.ErrInvalidInput
	}

	if !s.isValidTransition(newStatus) {
		return errs.ErrInvalidTransition
	}

	s.status = newStatus
	return nil
}

// SetPriority устанавливает приоритет
func (s *EntityState) SetPriority(priority Priority) error {
	if !isValidPriority(priority) {
		return errs.ErrInvalidInput
	}

	s.priority = priority
	return nil
}

// isValidTransition проверяет валидность перехода между статусами
func (s *EntityState) isValidTransition(newStatus Status) bool {
	// Если статус не меняется, это всегда валидно
	if s.status == newStatus {
		return true
	}

	// Из Cancelled можно вернуться только в Backlog
	if s.status == StatusCancelled {
		return newStatus == StatusBacklog
	}

	// Из Done можно вернуться в InReview (reopening)
	if s.status == StatusDone {
		return newStatus == StatusInReview || newStatus == StatusCancelled
	}

	// Стандартные переходы вперед
	transitions := map[Status][]Status{ //nolint:exhaustive // task.StatusDone, task.StatusCancelled не имеют переходов вперед
		StatusBacklog:    {StatusToDo, StatusCancelled},
		StatusToDo:       {StatusInProgress, StatusBacklog, StatusCancelled},
		StatusInProgress: {StatusInReview, StatusToDo, StatusCancelled},
		StatusInReview:   {StatusDone, StatusInProgress, StatusCancelled},
	}

	allowedStatuses, exists := transitions[s.status]
	if !exists {
		return false
	}

	for _, allowed := range allowedStatuses {
		if allowed == newStatus {
			return true
		}
	}

	return false
}

// Getters

// Type возвращает тип сущности
func (s *EntityState) Type() EntityType { return s.entityType }

// Status возвращает статус
func (s *EntityState) Status() Status { return s.status }

// Priority возвращает приоритет
func (s *EntityState) Priority() Priority { return s.priority }

// Validation helpers

func isValidEntityType(t EntityType) bool {
	return t == TypeTask || t == TypeBug || t == TypeEpic
}

func isValidStatus(s Status) bool {
	return s == StatusBacklog ||
		s == StatusToDo ||
		s == StatusInProgress ||
		s == StatusInReview ||
		s == StatusDone ||
		s == StatusCancelled
}

func isValidPriority(p Priority) bool {
	return p == PriorityLow ||
		p == PriorityMedium ||
		p == PriorityHigh ||
		p == PriorityCritical
}
