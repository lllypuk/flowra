package event

import "time"

// BaseEvent базовая реализация DomainEvent
type BaseEvent struct {
	EType         string    `json:"event_type"     bson:"event_type"`
	AggID         string    `json:"aggregate_id"   bson:"aggregate_id"`
	AggType       string    `json:"aggregate_type" bson:"aggregate_type"`
	OccAt         time.Time `json:"occurred_at"    bson:"occurred_at"`
	Ver           int       `json:"version"        bson:"version"`
	EventMetadata Metadata  `json:"metadata"       bson:"metadata"`
}

// NewBaseEvent создает новое базовое событие
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

// EventType возвращает тип события
func (e BaseEvent) EventType() string {
	return e.EType
}

// AggregateID возвращает ID агрегата
func (e BaseEvent) AggregateID() string {
	return e.AggID
}

// AggregateType возвращает тип агрегата
func (e BaseEvent) AggregateType() string {
	return e.AggType
}

// OccurredAt возвращает время возникновения события
func (e BaseEvent) OccurredAt() time.Time {
	return e.OccAt
}

// Version возвращает версию агрегата
func (e BaseEvent) Version() int {
	return e.Ver
}

// Metadata возвращает метаданные события
func (e BaseEvent) Metadata() Metadata {
	return e.EventMetadata
}
