package appcore

import (
	"context"
	"errors"

	"github.com/lllypuk/flowra/internal/domain/event"
)

var (
	// ErrAggregateNotFound is returned when the aggregate is not found
	ErrAggregateNotFound = errors.New("aggregate not found")

	// ErrConcurrencyConflict is returned on version conflict (optimistic locking)
	ErrConcurrencyConflict = errors.New("concurrency conflict detected")

	// ErrInvalidVersion is returned when the version is invalid
	ErrInvalidVersion = errors.New("invalid version")
)

// EventStore defines the interface for saving and loading events.
// The interface is declared here (on the consumer side - application layer),
// not in infrastructure, following idiomatic Go approach.
type EventStore interface {
	// SaveEvents saves events for an aggregate.
	// aggregateID - the aggregate identifier
	// events - events to save
	// expectedVersion - expected version for optimistic locking (0 for a new aggregate)
	SaveEvents(ctx context.Context, aggregateID string, events []event.DomainEvent, expectedVersion int) error

	// LoadEvents loads all events for an aggregate.
	// aggregateID - the aggregate identifier
	// Returns events in chronological order
	LoadEvents(ctx context.Context, aggregateID string) ([]event.DomainEvent, error)

	// GetVersion returns the current version of an aggregate.
	// aggregateID - the aggregate identifier
	// Returns 0 if the aggregate is not found
	GetVersion(ctx context.Context, aggregateID string) (int, error)
}
