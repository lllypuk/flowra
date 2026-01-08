package chat

import (
	"context"
	"time"

	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// ReadModel представляет read model для чата (материализованное представление)
type ReadModel struct {
	ID            uuid.UUID
	WorkspaceID   uuid.UUID
	Type          chat.Type
	Title         string
	IsPublic      bool
	CreatedBy     uuid.UUID
	CreatedAt     time.Time
	LastMessageAt *time.Time
	MessageCount  int
	Participants  []chat.Participant
}

// Filters представляет фильтры для поиска чатов
type Filters struct {
	Type     *chat.Type
	IsPublic *bool
	UserID   *uuid.UUID // участник
	Offset   int
	Limit    int
}

// CommandRepository определяет интерфейс для команд (изменение состояния) чатов
// Интерфейс объявлен на стороне потребителя (application layer)
// Использует Event Sourcing pattern
type CommandRepository interface {
	// Load загружает Chat из event store путем восстановления состояния из событий
	Load(ctx context.Context, chatID uuid.UUID) (*chat.Chat, error)

	// Save сохраняет новые события Chat в event store
	Save(ctx context.Context, c *chat.Chat) error

	// GetEvents возвращает все события чата
	GetEvents(ctx context.Context, chatID uuid.UUID) ([]event.DomainEvent, error)
}

// QueryRepository определяет интерфейс для запросов (только чтение) чатов
// Интерфейс объявлен на стороне потребителя (application layer)
// Использует Read Model для быстрых запросов
type QueryRepository interface {
	// FindByID находит чат по ID (из read model)
	FindByID(ctx context.Context, chatID uuid.UUID) (*ReadModel, error)

	// FindByWorkspace находит чаты workspace с фильтрами
	FindByWorkspace(ctx context.Context, workspaceID uuid.UUID, filters Filters) ([]*ReadModel, error)

	// FindByParticipant находит чаты пользователя
	FindByParticipant(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*ReadModel, error)

	// Count возвращает общее количество чатов в workspace
	Count(ctx context.Context, workspaceID uuid.UUID) (int, error)
}

// Repository объединяет Command и Query интерфейсы для удобства
// Используется когда use case нужны оба типа операций
type Repository interface {
	CommandRepository
	QueryRepository
}
