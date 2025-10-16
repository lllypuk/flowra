package chat

import (
	"context"
	"time"

	"github.com/lllypuk/teams-up/internal/domain/event"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
)

// Repository определяет интерфейс для работы с Chat aggregate (Event Sourcing)
type Repository interface {
	// Load загружает Chat из event store
	Load(ctx context.Context, chatID uuid.UUID) (*Chat, error)

	// Save сохраняет новые события Chat в event store
	Save(ctx context.Context, chat *Chat) error

	// GetEvents возвращает все события чата
	GetEvents(ctx context.Context, chatID uuid.UUID) ([]event.DomainEvent, error)
}

// ReadModelRepository определяет интерфейс для read model (проекции)
type ReadModelRepository interface {
	// FindByID находит чат по ID (из read model)
	FindByID(ctx context.Context, chatID uuid.UUID) (*ChatReadModel, error)

	// FindByWorkspace находит чаты workspace с фильтрами
	FindByWorkspace(ctx context.Context, workspaceID uuid.UUID, filters ChatFilters) ([]*ChatReadModel, error)

	// FindByParticipant находит чаты пользователя
	FindByParticipant(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*ChatReadModel, error)

	// Count возвращает общее количество чатов в workspace
	Count(ctx context.Context, workspaceID uuid.UUID) (int, error)
}

// ChatReadModel представляет read model для чата (материализованное представление)
type ChatReadModel struct {
	ID            uuid.UUID
	WorkspaceID   uuid.UUID
	Type          ChatType
	IsPublic      bool
	CreatedBy     uuid.UUID
	CreatedAt     time.Time
	LastMessageAt *time.Time
	MessageCount  int
	Participants  []Participant
}

// ChatFilters фильтры для поиска чатов
type ChatFilters struct {
	Type     *ChatType
	IsPublic *bool
	UserID   *uuid.UUID // участник
	Offset   int
	Limit    int
}
