package event

import "time"

// DomainEvent представляет доменное событие
type DomainEvent interface {
	// EventType возвращает тип события
	EventType() string

	// AggregateID возвращает ID агрегата
	AggregateID() string

	// AggregateType возвращает тип агрегата
	AggregateType() string

	// OccurredAt возвращает время возникновения события
	OccurredAt() time.Time

	// Version возвращает версию агрегата
	Version() int

	// Metadata возвращает метаданные события
	Metadata() Metadata
}
