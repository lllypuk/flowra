package event

import (
	"context"
	"time"
)

// DomainEvent represents a domain event
type DomainEvent interface {
	// EventType returns the event type
	EventType() string

	// AggregateID returns the aggregate ID
	AggregateID() string

	// AggregateType returns the aggregate type
	AggregateType() string

	// OccurredAt returns the time when the event occurred
	OccurredAt() time.Time

	// Version returns the aggregate version
	Version() int

	// Metadata returns the event metadata
	Metadata() Metadata
}

// Bus is an interface for publishing events
type Bus interface {
	// Publish publishes an event
	Publish(ctx context.Context, event DomainEvent) error
}
