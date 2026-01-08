package task

import (
	"context"
	"time"

	"github.com/lllypuk/flowra/internal/domain/event"
	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// CommandRepository predostavlyaet metody for work s agregatom Task
// via Event Sourcing (writing)
type CommandRepository interface {
	// Load loads Task from event store putem reconstruction state from events
	Load(ctx context.Context, taskID uuid.UUID) (*taskdomain.Aggregate, error)

	// Save saves novye event Task in event store
	Save(ctx context.Context, task *taskdomain.Aggregate) error

	// GetEvents returns all event tasks
	GetEvents(ctx context.Context, taskID uuid.UUID) ([]event.DomainEvent, error)
}

// QueryRepository predostavlyaet metody for chteniya dannyh Task
// from read model (denormalizovannoe view)
type QueryRepository interface {
	// FindByID finds zadachu po ID (from read model)
	FindByID(ctx context.Context, taskID uuid.UUID) (*ReadModel, error)

	// FindByChatID finds zadachu po ID chat
	FindByChatID(ctx context.Context, chatID uuid.UUID) (*ReadModel, error)

	// FindByAssignee finds tasks value user
	FindByAssignee(ctx context.Context, assigneeID uuid.UUID, filters Filters) ([]*ReadModel, error)

	// FindByStatus finds tasks s opredelennym statusom
	FindByStatus(ctx context.Context, status taskdomain.Status, filters Filters) ([]*ReadModel, error)

	// List returns list zadach s filtrami
	List(ctx context.Context, filters Filters) ([]*ReadModel, error)

	// Count returns count zadach s filtrami
	Count(ctx context.Context, filters Filters) (int, error)
}

// Repository combines Command and Query repozitorii
type Repository interface {
	CommandRepository
	QueryRepository
}

// Filters contains parameters filtering for zaprosov
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

// ReadModel represents denormalizovannoe view Task for zaprosov
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
