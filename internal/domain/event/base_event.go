package event

import "time"

// BaseEvent is a base implementation of DomainEvent
type BaseEvent struct {
	EType         string    `json:"event_type"     bson:"event_type"`
	AggID         string    `json:"aggregate_id"   bson:"aggregate_id"`
	AggType       string    `json:"aggregate_type" bson:"aggregate_type"`
	OccAt         time.Time `json:"occurred_at"    bson:"occurred_at"`
	Ver           int       `json:"version"        bson:"version"`
	EventMetadata Metadata  `json:"metadata"       bson:"metadata"`
}

// NewBaseEvent creates a new base event
func NewBaseEvent(eventType, aggregateID, aggregateType string, version int, metadata Metadata) BaseEvent {
	return BaseEvent{
		EType:         eventType,
		AggID:         aggregateID,
		AggType:       aggregateType,
		OccAt:         time.Now(),
		Ver:           version,
		EventMetadata: metadata,
	}
}

// EventType returns the event type
func (e BaseEvent) EventType() string {
	return e.EType
}

// AggregateID returns the aggregate ID
func (e BaseEvent) AggregateID() string {
	return e.AggID
}

// AggregateType returns the aggregate type
func (e BaseEvent) AggregateType() string {
	return e.AggType
}

// OccurredAt returns the time when the event occurred
func (e BaseEvent) OccurredAt() time.Time {
	return e.OccAt
}

// Version returns the aggregate version
func (e BaseEvent) Version() int {
	return e.Ver
}

// Metadata returns the event metadata
func (e BaseEvent) Metadata() Metadata {
	return e.EventMetadata
}
