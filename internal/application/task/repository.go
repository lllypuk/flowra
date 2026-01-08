package task

import (
	"context"
	"time"

	"github.com/lllypuk/flowra/internal/domain/event"
	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// CommandRepository предоставляет методы for workы с агрегатом Task
// via Event Sourcing (writing)
type CommandRepository interface {
	// Load loads Task from event store путем reconstruction state from events
	Load(ctx context.Context, taskID uuid.UUID) (*taskdomain.Aggregate, error)

	// Save saves новые event Task in event store
	Save(ctx context.Context, task *taskdomain.Aggregate) error

	// GetEvents returns all event tasks
	GetEvents(ctx context.Context, taskID uuid.UUID) ([]event.DomainEvent, error)
}

// QueryRepository предоставляет методы for чтения данных Task
// from read model (денормализованное view)
type QueryRepository interface {
	// FindByID finds задачу по ID (from read model)
	FindByID(ctx context.Context, taskID uuid.UUID) (*ReadModel, error)

	// FindByChatID finds задачу по ID chat
	FindByChatID(ctx context.Context, chatID uuid.UUID) (*ReadModel, error)

	// FindByAssignee finds tasks наvalueенные user
	FindByAssignee(ctx context.Context, assigneeID uuid.UUID, filters Filters) ([]*ReadModel, error)

	// FindByStatus finds tasks с определенным статусом
	FindByStatus(ctx context.Context, status taskdomain.Status, filters Filters) ([]*ReadModel, error)

	// List returns list задач с фильтрами
	List(ctx context.Context, filters Filters) ([]*ReadModel, error)

	// Count returns count задач с фильтрами
	Count(ctx context.Context, filters Filters) (int, error)
}

// Repository combines Command and Query репозитории
type Repository interface {
	CommandRepository
	QueryRepository
}

// Filters contains parameters filtering for запросов
type Filters struct {
	ChatID     *uuid.UUID
	AssigneeID *uuid.UUID
	Status     *taskdomain.Status
	Priority   *taskdomain.Priority
	EntityType *taskdomain.EntityType
	CreatedBy  *uuid.UUID
	Offset     int
	Limit      int
}

// ReadModel represents денормализованное view Task for запросов
type ReadModel struct {
	ID         uuid.UUID
	ChatID     uuid.UUID
	Title      string
	EntityType taskdomain.EntityType
	Status     taskdomain.Status
	Priority   taskdomain.Priority
	AssignedTo *uuid.UUID
	DueDate    *time.Time
	CreatedBy  uuid.UUID
	CreatedAt  time.Time
	Version    int
}
