// Package appcore provides core application interfaces and shared utilities.
package appcore

import (
	"context"
	"time"

	"github.com/lllypuk/flowra/internal/domain/event"
)

// OutboxEntry represents an event waiting to be published to the event bus.
type OutboxEntry struct {
	ID            string
	EventID       string
	EventType     string
	AggregateID   string
	AggregateType string
	Payload       []byte
	CreatedAt     time.Time
	ProcessedAt   *time.Time
	RetryCount    int
	LastError     string
}

// Outbox defines the interface for transactional outbox operations.
// The outbox pattern guarantees event delivery by storing events in the same
// transaction as domain changes, then publishing them asynchronously.
type Outbox interface {
	// Add inserts an event into the outbox.
	// This should be called within the same database transaction as the domain changes.
	Add(ctx context.Context, evt event.DomainEvent) error

	// AddBatch inserts multiple events into the outbox atomically.
	AddBatch(ctx context.Context, events []event.DomainEvent) error

	// Poll retrieves unprocessed events up to the specified batch size.
	// Events are returned ordered by creation time (oldest first).
	Poll(ctx context.Context, batchSize int) ([]OutboxEntry, error)

	// MarkProcessed marks an event as successfully published.
	MarkProcessed(ctx context.Context, entryID string) error

	// MarkFailed records a publishing failure for retry.
	MarkFailed(ctx context.Context, entryID string, err error) error

	// Cleanup removes old processed entries older than the specified duration.
	Cleanup(ctx context.Context, olderThan time.Duration) (int64, error)

	// Count returns the number of unprocessed entries (for monitoring).
	Count(ctx context.Context) (int64, error)

	// Stats returns statistics about the outbox (count and oldest entry timestamp).
	Stats(ctx context.Context) (count int64, oldest time.Time, err error)
}
