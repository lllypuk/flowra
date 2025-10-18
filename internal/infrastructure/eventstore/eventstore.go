package eventstore

import (
	"context"
	"errors"

	"github.com/lllypuk/teams-up/internal/domain/event"
)

var (
	// ErrAggregateNotFound возвращается когда агрегат не найден
	ErrAggregateNotFound = errors.New("aggregate not found")

	// ErrConcurrencyConflict возвращается при конфликте версий (optimistic locking)
	ErrConcurrencyConflict = errors.New("concurrency conflict detected")

	// ErrInvalidVersion возвращается при невалидной версии
	ErrInvalidVersion = errors.New("invalid version")
)

// EventStore определяет интерфейс для сохранения и загрузки событий
type EventStore interface {
	// SaveEvents сохраняет события для агрегата
	// aggregateID - идентификатор агрегата
	// events - события для сохранения
	// expectedVersion - ожидаемая версия для optimistic locking (0 для нового агрегата)
	SaveEvents(ctx context.Context, aggregateID string, events []event.DomainEvent, expectedVersion int) error

	// LoadEvents загружает все события для агрегата
	// aggregateID - идентификатор агрегата
	// Возвращает события в хронологическом порядке
	LoadEvents(ctx context.Context, aggregateID string) ([]event.DomainEvent, error)

	// GetVersion возвращает текущую версию агрегата
	// aggregateID - идентификатор агрегата
	// Возвращает 0 если агрегат не найден
	GetVersion(ctx context.Context, aggregateID string) (int, error)
}
