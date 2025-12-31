package task

import (
	"context"
	"time"

	"github.com/lllypuk/flowra/internal/domain/event"
	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// CommandRepository предоставляет методы для работы с агрегатом Task
// через Event Sourcing (запись)
type CommandRepository interface {
	// Load загружает Task из event store путем восстановления состояния из событий
	Load(ctx context.Context, taskID uuid.UUID) (*taskdomain.Aggregate, error)

	// Save сохраняет новые события Task в event store
	Save(ctx context.Context, task *taskdomain.Aggregate) error

	// GetEvents возвращает все события задачи
	GetEvents(ctx context.Context, taskID uuid.UUID) ([]event.DomainEvent, error)
}

// QueryRepository предоставляет методы для чтения данных Task
// из read model (денормализованное представление)
type QueryRepository interface {
	// FindByID находит задачу по ID (из read model)
	FindByID(ctx context.Context, taskID uuid.UUID) (*ReadModel, error)

	// FindByChatID находит задачу по ID чата
	FindByChatID(ctx context.Context, chatID uuid.UUID) (*ReadModel, error)

	// FindByAssignee находит задачи назначенные пользователю
	FindByAssignee(ctx context.Context, assigneeID uuid.UUID, filters Filters) ([]*ReadModel, error)

	// FindByStatus находит задачи с определенным статусом
	FindByStatus(ctx context.Context, status taskdomain.Status, filters Filters) ([]*ReadModel, error)

	// List возвращает список задач с фильтрами
	List(ctx context.Context, filters Filters) ([]*ReadModel, error)

	// Count возвращает количество задач с фильтрами
	Count(ctx context.Context, filters Filters) (int, error)
}

// Repository объединяет Command и Query репозитории
type Repository interface {
	CommandRepository
	QueryRepository
}

// Filters содержит параметры фильтрации для запросов
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

// ReadModel представляет денормализованное представление Task для запросов
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
