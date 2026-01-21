package eventstore

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
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

// createEventByType creates empty event instance by event type.
func createEventByType(eventType string) (event.DomainEvent, error) {
	switch eventType {
	// Chat events
	case chatdomain.EventTypeChatCreated:
		return &chatdomain.Created{}, nil
	case chatdomain.EventTypeParticipantAdded:
		return &chatdomain.ParticipantAdded{}, nil
	case chatdomain.EventTypeParticipantRemoved:
		return &chatdomain.ParticipantRemoved{}, nil
	case chatdomain.EventTypeChatTypeChanged:
		return &chatdomain.TypeChanged{}, nil
	case chatdomain.EventTypeStatusChanged:
		return &chatdomain.StatusChanged{}, nil
	case chatdomain.EventTypeUserAssigned:
		return &chatdomain.UserAssigned{}, nil
	case chatdomain.EventTypeAssigneeRemoved:
		return &chatdomain.AssigneeRemoved{}, nil
	case chatdomain.EventTypePrioritySet:
		return &chatdomain.PrioritySet{}, nil
	case chatdomain.EventTypeDueDateSet:
		return &chatdomain.DueDateSet{}, nil
	case chatdomain.EventTypeDueDateRemoved:
		return &chatdomain.DueDateRemoved{}, nil
	case chatdomain.EventTypeChatRenamed:
		return &chatdomain.Renamed{}, nil
	case chatdomain.EventTypeSeveritySet:
		return &chatdomain.SeveritySet{}, nil
	case chatdomain.EventTypeChatDeleted:
		return &chatdomain.Deleted{}, nil
	case chatdomain.EventTypeChatClosed:
		return &chatdomain.Closed{}, nil
	case chatdomain.EventTypeChatReopened:
		return &chatdomain.Reopened{}, nil
	// Task events
	case taskdomain.EventTypeTaskCreated:
		return &taskdomain.Created{}, nil
	case taskdomain.EventTypeTaskUpdated:
		return &taskdomain.Updated{}, nil
	case taskdomain.EventTypeTaskDeleted:
		return &taskdomain.Deleted{}, nil
	case taskdomain.EventTypeStatusChanged:
		return &taskdomain.StatusChanged{}, nil
	case taskdomain.EventTypeAssigneeChanged:
		return &taskdomain.AssigneeChanged{}, nil
	case taskdomain.EventTypePriorityChanged:
		return &taskdomain.PriorityChanged{}, nil
	case taskdomain.EventTypeDueDateChanged:
		return &taskdomain.DueDateChanged{}, nil
	case taskdomain.EventTypeCustomFieldSet:
		return &taskdomain.CustomFieldSet{}, nil
	default:
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}
}

// Deserialize preobrazuet MongoDB dokument obratno in domennoe event
func (s *EventSerializer) Deserialize(doc *EventDocument) (event.DomainEvent, error) {
	bsonBytes, err := bson.Marshal(doc.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal BSON data: %w", err)
	}

	evt, err := createEventByType(doc.EventType)
	if err != nil {
		return nil, err
	}

	if unmarshalErr := bson.Unmarshal(bsonBytes, evt); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", unmarshalErr)
	}

	// Fix version and other fields from document (not from Data)
	// The Data field may contain stale values because SaveEvents overwrites doc.Version
	// but doesn't update doc.Data
	if err := setEventFields(evt, doc); err != nil {
		return nil, fmt.Errorf("failed to set event fields: %w", err)
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

// setEventFields sets the correct values for BaseEvent fields from the EventDocument.
// This is necessary because SaveEvents overwrites doc.Version but doesn't update doc.Data,
// so the version in Data may be stale.
func setEventFields(evt event.DomainEvent, doc *EventDocument) error {
	// Use reflection to access the embedded BaseEvent
	v := reflect.ValueOf(evt).Elem()
	if !v.IsValid() {
		return fmt.Errorf("invalid event value")
	}

	// Find the BaseEvent field
	baseField := v.FieldByName("BaseEvent")
	if !baseField.IsValid() {
		return fmt.Errorf("event does not have BaseEvent field")
	}

	// Set the version from document (authoritative source)
	verField := baseField.FieldByName("Ver")
	if verField.IsValid() && verField.CanSet() {
		verField.SetInt(int64(doc.Version))
	}

	// Set other fields from document for consistency
	aggIDField := baseField.FieldByName("AggID")
	if aggIDField.IsValid() && aggIDField.CanSet() {
		aggIDField.SetString(doc.AggregateID)
	}

	aggTypeField := baseField.FieldByName("AggType")
	if aggTypeField.IsValid() && aggTypeField.CanSet() {
		aggTypeField.SetString(doc.AggregateType)
	}

	occAtField := baseField.FieldByName("OccAt")
	if occAtField.IsValid() && occAtField.CanSet() {
		occAtField.Set(reflect.ValueOf(doc.OccurredAt))
	}

	return nil
}
