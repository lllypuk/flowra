package eventstore

import (
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
)

// EventDocument представляет документ события в MongoDB
type EventDocument struct {
	ID bson.ObjectID `bson:"_id,omitempty"`

	AggregateID   string                `bson:"aggregate_id"`
	AggregateType string                `bson:"aggregate_type"`
	EventType     string                `bson:"event_type"`
	Version       int                   `bson:"version"`
	Data          bson.M                `bson:"data"`
	Metadata      EventMetadataDocument `bson:"metadata"`
	OccurredAt    time.Time             `bson:"occurred_at"`
	CreatedAt     time.Time             `bson:"created_at"`
}

// EventMetadataDocument представляет метаданные события в MongoDB
type EventMetadataDocument struct {
	Timestamp     time.Time `bson:"timestamp"`
	UserID        string    `bson:"user_id,omitempty"`
	CorrelationID string    `bson:"correlation_id"`
	CausationID   string    `bson:"causation_id,omitempty"`
	IPAddress     string    `bson:"ip_address,omitempty"`
	UserAgent     string    `bson:"user_agent,omitempty"`
}

// EventSerializer выполняет сериализацию и десериализацию событий для MongoDB
type EventSerializer struct {
}

// NewEventSerializer создает новый сериализатор событий
func NewEventSerializer() *EventSerializer {
	return &EventSerializer{}
}

// Serialize преобразует доменное событие в MongoDB документ
func (s *EventSerializer) Serialize(e event.DomainEvent) (*EventDocument, error) {
	// Преобразуем событие в JSON и обратно в BSON.M
	// для более надежной сериализации сложных типов
	jsonData, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event to JSON: %w", err)
	}

	var dataMap bson.M
	if err2 := json.Unmarshal(jsonData, &dataMap); err2 != nil {
		return nil, fmt.Errorf("failed to unmarshal event to map: %w", err2)
	}

	// Преобразуем метаданные
	metadata := e.Metadata()
	metadataDoc := EventMetadataDocument{
		Timestamp:     metadata.Timestamp,
		UserID:        metadata.UserID,
		CorrelationID: metadata.CorrelationID,
		CausationID:   metadata.CausationID,
		IPAddress:     metadata.IPAddress,
		UserAgent:     metadata.UserAgent,
	}

	doc := &EventDocument{
		AggregateID:   e.AggregateID(),
		AggregateType: e.AggregateType(),
		EventType:     e.EventType(),
		Version:       e.Version(),
		Data:          dataMap,
		Metadata:      metadataDoc,
		OccurredAt:    e.OccurredAt(),
		CreatedAt:     time.Now(),
	}

	return doc, nil
}

// SerializeMany сериализует несколько событий сразу
func (s *EventSerializer) SerializeMany(events []event.DomainEvent) ([]*EventDocument, error) {
	documents := make([]*EventDocument, 0, len(events))

	for _, e := range events {
		doc, err := s.Serialize(e)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize event at index %d: %w", len(documents), err)
		}
		documents = append(documents, doc)
	}

	return documents, nil
}

// Deserialize преобразует MongoDB документ обратно в доменное событие
func (s *EventSerializer) Deserialize(doc *EventDocument) (event.DomainEvent, error) {
	// Конвертируем BSON.M в байты для десериализации
	bsonBytes, err := bson.Marshal(doc.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal BSON data: %w", err)
	}

	// Создаем конкретный тип события на основе EventType
	var evt event.DomainEvent

	switch doc.EventType {
	case chatdomain.EventTypeChatCreated:
		evt = &chatdomain.Created{}
	case chatdomain.EventTypeParticipantAdded:
		evt = &chatdomain.ParticipantAdded{}
	case chatdomain.EventTypeParticipantRemoved:
		evt = &chatdomain.ParticipantRemoved{}
	case chatdomain.EventTypeChatTypeChanged:
		evt = &chatdomain.TypeChanged{}
	case chatdomain.EventTypeStatusChanged:
		evt = &chatdomain.StatusChanged{}
	case chatdomain.EventTypeUserAssigned:
		evt = &chatdomain.UserAssigned{}
	case chatdomain.EventTypeAssigneeRemoved:
		evt = &chatdomain.AssigneeRemoved{}
	case chatdomain.EventTypePrioritySet:
		evt = &chatdomain.PrioritySet{}
	case chatdomain.EventTypeDueDateSet:
		evt = &chatdomain.DueDateSet{}
	case chatdomain.EventTypeDueDateRemoved:
		evt = &chatdomain.DueDateRemoved{}
	case chatdomain.EventTypeChatRenamed:
		evt = &chatdomain.Renamed{}
	case chatdomain.EventTypeSeveritySet:
		evt = &chatdomain.SeveritySet{}
	case chatdomain.EventTypeChatDeleted:
		evt = &chatdomain.Deleted{}
	default:
		return nil, fmt.Errorf("unknown event type: %s", doc.EventType)
	}

	// Десериализуем BSON напрямую в конкретный тип события
	if unmarshalErr := bson.Unmarshal(bsonBytes, evt); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", unmarshalErr)
	}

	return evt, nil
}

// DeserializeMany десериализует несколько документов сразу
func (s *EventSerializer) DeserializeMany(docs []*EventDocument) ([]event.DomainEvent, error) {
	events := make([]event.DomainEvent, 0, len(docs))

	for i, doc := range docs {
		evt, err := s.Deserialize(doc)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize event at index %d: %w", i, err)
		}
		events = append(events, evt)
	}

	return events, nil
}
