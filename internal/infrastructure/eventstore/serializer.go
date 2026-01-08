package eventstore

import (
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
)

// EventDocument represents dokument event in MongoDB
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

// EventMetadataDocument represents metadannye event in MongoDB
type EventMetadataDocument struct {
	Timestamp     time.Time `bson:"timestamp"`
	UserID        string    `bson:"user_id,omitempty"`
	CorrelationID string    `bson:"correlation_id"`
	CausationID   string    `bson:"causation_id,omitempty"`
	IPAddress     string    `bson:"ip_address,omitempty"`
	UserAgent     string    `bson:"user_agent,omitempty"`
}

// EventSerializer performs serializatsiyu and deserializatsiyu events for MongoDB
type EventSerializer struct {
}

// NewEventSerializer creates New serializator events
func NewEventSerializer() *EventSerializer {
	return &EventSerializer{}
}

// Serialize preobrazuet domennoe event in MongoDB dokument
func (s *EventSerializer) Serialize(e event.DomainEvent) (*EventDocument, error) {
	// preobrazuem event in JSON and obratno in BSON.M
	// for bolee nadezhnoy serializatsii slozhnyh tipov
	jsonData, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event to JSON: %w", err)
	}

	var dataMap bson.M
	if err2 := json.Unmarshal(jsonData, &dataMap); err2 != nil {
		return nil, fmt.Errorf("failed to unmarshal event to map: %w", err2)
	}

	// preobrazuem metadannye
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

// SerializeMany serializuet several events srazu
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

// Deserialize preobrazuet MongoDB dokument obratno in domennoe event
func (s *EventSerializer) Deserialize(doc *EventDocument) (event.DomainEvent, error) {
	// konvertiruem BSON.M in bayty for deserializatsii
	bsonBytes, err := bson.Marshal(doc.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal BSON data: %w", err)
	}

	// Creating konkretnyy type event on osnove EventType
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

	// serializing BSON napryamuyu in konkretnyy type event
	if unmarshalErr := bson.Unmarshal(bsonBytes, evt); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", unmarshalErr)
	}

	return evt, nil
}

// DeserializeMany deserializuet several dokumentov srazu
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
