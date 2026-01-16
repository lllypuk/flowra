package appcore

import (
	"context"

	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// ReadModelProjector rebuilds and maintains read models from event store.
// Interface is declared on consumer side (application layer) following Go idioms.
type ReadModelProjector interface {
	// RebuildOne rebuilds read model for a single aggregate from its events.
	// Returns ErrAggregateNotFound if no events exist for the aggregate.
	RebuildOne(ctx context.Context, aggregateID uuid.UUID) error

	// RebuildAll rebuilds read models for all aggregates of this type.
	// Continues processing even if individual rebuilds fail.
	// Returns error only if the rebuild process itself cannot start.
	RebuildAll(ctx context.Context) error

	// ProcessEvent applies a single event to the read model.
	// Used for incremental updates from event handlers.
	ProcessEvent(ctx context.Context, event event.DomainEvent) error

	// VerifyConsistency checks if read model matches the state derived from events.
	// Returns true if consistent, false if discrepancies found.
	VerifyConsistency(ctx context.Context, aggregateID uuid.UUID) (bool, error)
}
