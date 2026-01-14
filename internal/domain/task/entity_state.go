package task

import (
	"github.com/lllypuk/flowra/internal/domain/errs"
)

// EntityType represents type entity
type EntityType string

const (
	// TypeTask obychnaya task
	TypeTask EntityType = "task"
	// TypeBug bug/defekt
	TypeBug EntityType = "bug"
	// TypeEpic epik (bolshaya task)
	TypeEpic EntityType = "epic"
	// TypeDiscussion obsuzhdenie (not tipizirovannaya suschnost)
	TypeDiscussion EntityType = "discussion"
)

// Status represents status tasks
type Status string

const (
	// StatusBacklog task in bekloge
	StatusBacklog Status = "Backlog"
	// StatusToDo task gotova to work
	StatusToDo Status = "To Do"
	// StatusInProgress task in work
	StatusInProgress Status = "In Progress"
	// StatusInReview task on revyu
	StatusInReview Status = "In Review"
	// StatusDone task zavershena
	StatusDone Status = "Done"
	// StatusCancelled task otmenena
	StatusCancelled Status = "Cancelled"
)

// Priority represents prioritet tasks
type Priority string

const (
	// PriorityLow nizkiy prioritet
	PriorityLow Priority = "Low"
	// PriorityMedium sredniy prioritet
	PriorityMedium Priority = "Medium"
	// PriorityHigh vysokiy prioritet
	PriorityHigh Priority = "High"
	// PriorityCritical kriticheskiy prioritet
	PriorityCritical Priority = "Critical"
)

// EntityState represents state tipizirovannoy entity
type EntityState struct {
	entityType EntityType
	status     Status
	priority   Priority
}

// NewEntityState creates new state entity
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

// ChangeStatus izmenyaet status s valid perehoda
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

// SetPriority sets prioritet
func (s *EntityState) SetPriority(priority Priority) error {
	if !isValidPriority(priority) {
		return errs.ErrInvalidInput
	}

	s.priority = priority
	return nil
}

// isValidTransition validates perehoda between statusami
func (s *EntityState) isValidTransition(newStatus Status) bool {
	// if status not menyaetsya, it is always valid
	if s.status == newStatus {
		return true
	}

	// from Cancelled mozhno vernutsya only in Backlog
	if s.status == StatusCancelled {
		return newStatus == StatusBacklog
	}

	// any other transition is allowed (Kanban-style board)
	return true
}

// Getters

// Type returns type entity
func (s *EntityState) Type() EntityType { return s.entityType }

// Status returns status
func (s *EntityState) Status() Status { return s.status }

// Priority returns prioritet
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
