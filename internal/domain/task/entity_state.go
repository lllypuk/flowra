package task

import (
	"slices"

	"github.com/lllypuk/flowra/internal/domain/errs"
)

// EntityType represents type сущности
type EntityType string

const (
	// TypeTask обычная task
	TypeTask EntityType = "task"
	// TypeBug bug/дефект
	TypeBug EntityType = "bug"
	// TypeEpic эпик (большая task)
	TypeEpic EntityType = "epic"
	// TypeDiscussion обсуждение (not типизированная сущность)
	TypeDiscussion EntityType = "discussion"
)

// Status represents status tasks
type Status string

const (
	// StatusBacklog task in бэклоге
	StatusBacklog Status = "Backlog"
	// StatusToDo task готова to workе
	StatusToDo Status = "To Do"
	// StatusInProgress task in workе
	StatusInProgress Status = "In Progress"
	// StatusInReview task on ревью
	StatusInReview Status = "In Review"
	// StatusDone task завершена
	StatusDone Status = "Done"
	// StatusCancelled task отменена
	StatusCancelled Status = "Cancelled"
)

// Priority represents приоритет tasks
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

// EntityState represents state типизированной сущности
type EntityState struct {
	entityType EntityType
	status     Status
	priority   Priority
}

// NewEntityState creates новое state сущности
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

// ChangeStatus изменяет status с validацией перехода
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

// isValidTransition validates перехода between статусами
func (s *EntityState) isValidTransition(newStatus Status) bool {
	// if status not меняется, it is always validно
	if s.status == newStatus {
		return true
	}

	// from Cancelled можно вернуться only in Backlog
	if s.status == StatusCancelled {
		return newStatus == StatusBacklog
	}

	// from Done можно вернуться in InReview (reopening)
	if s.status == StatusDone {
		return newStatus == StatusInReview || newStatus == StatusCancelled
	}

	// Стандартные переходы вbefore
	transitions := map[Status][]Status{ //nolint:exhaustive // task.StatusDone, task.StatusCancelled not имеют переходов вbefore
		StatusBacklog:    {StatusToDo, StatusCancelled},
		StatusToDo:       {StatusInProgress, StatusBacklog, StatusCancelled},
		StatusInProgress: {StatusInReview, StatusToDo, StatusCancelled},
		StatusInReview:   {StatusDone, StatusInProgress, StatusCancelled},
	}

	allowedStatuses, exists := transitions[s.status]
	if !exists {
		return false
	}

	return slices.Contains(allowedStatuses, newStatus)
}

// Getters

// Type returns type сущности
func (s *EntityState) Type() EntityType { return s.entityType }

// Status returns status
func (s *EntityState) Status() Status { return s.status }

// Priority returns приоритет
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
