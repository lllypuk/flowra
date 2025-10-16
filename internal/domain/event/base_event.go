package event

import "time"

// BaseEvent базовая реализация DomainEvent
type BaseEvent struct {
	eventType     string
	aggregateID   string
	aggregateType string
	occurredAt    time.Time
	version       int
	metadata      Metadata
}

// NewBaseEvent создает новое базовое событие
func NewBaseEvent(eventType, aggregateID, aggregateType string, version int, metadata Metadata) BaseEvent {
	return BaseEvent{
		eventType:     eventType,
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		occurredAt:    time.Now(),
		version:       version,
		metadata:      metadata,
	}
}

// EventType возвращает тип события
func (e BaseEvent) EventType() string {
	return e.eventType
}

// AggregateID возвращает ID агрегата
func (e BaseEvent) AggregateID() string {
	return e.aggregateID
}

// AggregateType возвращает тип агрегата
func (e BaseEvent) AggregateType() string {
	return e.aggregateType
}

// OccurredAt возвращает время возникновения события
func (e BaseEvent) OccurredAt() time.Time {
	return e.occurredAt
}

// Version возвращает версию агрегата
func (e BaseEvent) Version() int {
	return e.version
}

// Metadata возвращает метаданные события
func (e BaseEvent) Metadata() Metadata {
	return e.metadata
}
