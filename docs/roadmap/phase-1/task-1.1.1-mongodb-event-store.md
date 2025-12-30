# Task 1.1.1: MongoDB Event Store

**Приоритет:** 🔴 КРИТИЧЕСКИЙ
**Статус:** ✅ Completed
**Время:** 3-4 дня
**Зависимости:** Phase 0 завершена

---

## Проблема

In-memory event store работает для тестирования, но не персистентен. Production-ready приложению нужен надежный, масштабируемый event store с:
- Persistence (events сохраняются навсегда)
- Optimistic concurrency control
- Event replay capability
- High performance (thousands of events/sec)

---

## Цель

Реализовать production-ready MongoDB Event Store с оптимистической блокировкой и event sourcing support.

---

## Файлы для создания

```
internal/infrastructure/eventstore/
├── mongodb_store.go           (implementation)
├── mongodb_store_test.go      (unit tests)
├── serializer.go              (event serialization)
├── serializer_test.go         (serializer tests)
└── integration_test.go        (MongoDB integration tests)

migrations/mongodb/
└── 001_event_store_schema.js  (indexes)
```

---

## Детальный план реализации

### 1. MongoDB Schema Design

#### Collection: `events`

```javascript
// migrations/mongodb/001_event_store_schema.js

db.createCollection("events");

// Indexes
db.events.createIndex(
    { aggregate_id: 1, version: 1 },
    { unique: true, name: "aggregate_version_unique" }
);

db.events.createIndex(
    { aggregate_type: 1, created_at: -1 },
    { name: "aggregate_type_created" }
);

db.events.createIndex(
    { "metadata.correlation_id": 1 },
    { name: "correlation_id" }
);

db.events.createIndex(
    { created_at: -1 },
    { name: "created_at_desc" }
);
```

#### Document Structure

```json
{
  "_id": ObjectId("..."),
  "aggregate_id": "uuid-v4-string",
  "aggregate_type": "Chat|Message|Task|Notification|User|Workspace",
  "event_type": "ChatCreated|MessagePosted|StatusChanged|...",
  "version": 1,
  "data": {
    // Event-specific data (BSON)
    "chatID": "uuid",
    "workspaceID": "uuid",
    "type": "Discussion",
    "title": "New Chat",
    // ... event fields
  },
  "metadata": {
    "timestamp": ISODate("2025-11-11T10:00:00Z"),
    "user_id": "uuid-v4-string",
    "correlation_id": "uuid-v4-string",
    "causation_id": "uuid-v4-string"  // optional
  },
  "created_at": ISODate("2025-11-11T10:00:00Z")
}
```

---

### 2. Event Serialization (serializer.go)

Event serialization/deserialization для MongoDB BSON.

```go
package eventstore

import (
    "fmt"
    "time"
    "github.com/google/uuid"
    "go.mongodb.org/mongo-driver/v2/bson"

    "github.com/lllypuk/flowra/internal/application/shared"
    chatevents "github.com/lllypuk/flowra/internal/domain/chat/events"
    messageevents "github.com/lllypuk/flowra/internal/domain/message/events"
    // ... other event packages
)

// EventDocument - MongoDB document for event
type EventDocument struct {
    ID            bson.ObjectID          `bson:"_id,omitempty"`
    AggregateID   string                 `bson:"aggregate_id"`
    AggregateType string                 `bson:"aggregate_type"`
    EventType     string                 `bson:"event_type"`
    Version       int                    `bson:"version"`
    Data          bson.M                 `bson:"data"`
    Metadata      EventMetadata          `bson:"metadata"`
    CreatedAt     time.Time              `bson:"created_at"`
}

// EventMetadata - metadata для события
type EventMetadata struct {
    Timestamp     time.Time `bson:"timestamp"`
    UserID        string    `bson:"user_id,omitempty"`
    CorrelationID string    `bson:"correlation_id"`
    CausationID   string    `bson:"causation_id,omitempty"`
}

// EventSerializer - serializer для events
type EventSerializer struct {
    eventRegistry map[string]func() shared.DomainEvent
}

// NewEventSerializer - конструктор
func NewEventSerializer() *EventSerializer {
    serializer := &EventSerializer{
        eventRegistry: make(map[string]func() shared.DomainEvent),
    }

    // Register all event types
    serializer.registerChatEvents()
    serializer.registerMessageEvents()
    serializer.registerTaskEvents()
    // ... other domains

    return serializer
}

// Serialize - domain event → MongoDB document
func (s *EventSerializer) Serialize(event shared.DomainEvent) (*EventDocument, error) {
    // Extract metadata
    metadata := EventMetadata{
        Timestamp:     event.OccurredAt(),
        CorrelationID: uuid.New().String(),  // TODO: get from context
    }

    if event.UserID() != uuid.Nil {
        metadata.UserID = event.UserID().String()
    }

    // Serialize event data to BSON
    data, err := bson.Marshal(event)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal event: %w", err)
    }

    var dataMap bson.M
    if err := bson.Unmarshal(data, &dataMap); err != nil {
        return nil, fmt.Errorf("failed to unmarshal event to map: %w", err)
    }

    doc := &EventDocument{
        AggregateID:   event.AggregateID().String(),
        AggregateType: s.getAggregateType(event),
        EventType:     event.EventType(),
        Version:       event.Version(),
        Data:          dataMap,
        Metadata:      metadata,
        CreatedAt:     time.Now(),
    }

    return doc, nil
}

// Deserialize - MongoDB document → domain event
func (s *EventSerializer) Deserialize(doc *EventDocument) (shared.DomainEvent, error) {
    // Get event factory
    factory, ok := s.eventRegistry[doc.EventType]
    if !ok {
        return nil, fmt.Errorf("unknown event type: %s", doc.EventType)
    }

    // Create event instance
    event := factory()

    // Unmarshal data into event
    data, err := bson.Marshal(doc.Data)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal data: %w", err)
    }

    if err := bson.Unmarshal(data, event); err != nil {
        return nil, fmt.Errorf("failed to unmarshal event: %w", err)
    }

    return event, nil
}

// getAggregateType - extract aggregate type from event
func (s *EventSerializer) getAggregateType(event shared.DomainEvent) string {
    switch event.(type) {
    case *chatevents.ChatCreated, *chatevents.ParticipantAdded, *chatevents.StatusChanged:
        return "Chat"
    case *messageevents.MessagePosted, *messageevents.MessageEdited:
        return "Message"
    // ... other aggregates
    default:
        return "Unknown"
    }
}

// registerChatEvents - register all Chat events
func (s *EventSerializer) registerChatEvents() {
    s.eventRegistry["ChatCreated"] = func() shared.DomainEvent {
        return &chatevents.ChatCreated{}
    }
    s.eventRegistry["ParticipantAdded"] = func() shared.DomainEvent {
        return &chatevents.ParticipantAdded{}
    }
    s.eventRegistry["ParticipantRemoved"] = func() shared.DomainEvent {
        return &chatevents.ParticipantRemoved{}
    }
    s.eventRegistry["ChatConvertedToTask"] = func() shared.DomainEvent {
        return &chatevents.ChatConvertedToTask{}
    }
    s.eventRegistry["StatusChanged"] = func() shared.DomainEvent {
        return &chatevents.StatusChanged{}
    }
    s.eventRegistry["UserAssigned"] = func() shared.DomainEvent {
        return &chatevents.UserAssigned{}
    }
    s.eventRegistry["PrioritySet"] = func() shared.DomainEvent {
        return &chatevents.PrioritySet{}
    }
    s.eventRegistry["DueDateSet"] = func() shared.DomainEvent {
        return &chatevents.DueDateSet{}
    }
    s.eventRegistry["SeveritySet"] = func() shared.DomainEvent {
        return &chatevents.SeveritySet{}
    }
    s.eventRegistry["ChatRenamed"] = func() shared.DomainEvent {
        return &chatevents.ChatRenamed{}
    }
}

// registerMessageEvents - register all Message events
func (s *EventSerializer) registerMessageEvents() {
    s.eventRegistry["MessagePosted"] = func() shared.DomainEvent {
        return &messageevents.MessagePosted{}
    }
    s.eventRegistry["MessageEdited"] = func() shared.DomainEvent {
        return &messageevents.MessageEdited{}
    }
    s.eventRegistry["MessageDeleted"] = func() shared.DomainEvent {
        return &messageevents.MessageDeleted{}
    }
    s.eventRegistry["ReactionAdded"] = func() shared.DomainEvent {
        return &messageevents.ReactionAdded{}
    }
    s.eventRegistry["ReactionRemoved"] = func() shared.DomainEvent {
        return &messageevents.ReactionRemoved{}
    }
}

// registerTaskEvents, etc...
```

---

### 3. MongoDB Event Store Implementation (mongodb_store.go)

```go
package eventstore

import (
    "context"
    "fmt"
    "github.com/google/uuid"
    "go.mongodb.org/mongo-driver/v2/bson"
    "go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"

    "github.com/lllypuk/flowra/internal/application/shared"
)

// MongoEventStore - MongoDB implementation of EventStore
type MongoEventStore struct {
    client     *mongo.Client
    database   *mongo.Database
    collection *mongo.Collection
    serializer *EventSerializer
}

// NewMongoEventStore - конструктор
func NewMongoEventStore(client *mongo.Client, databaseName string) *MongoEventStore {
    database := client.Database(databaseName)
    collection := database.Collection("events")

    return &MongoEventStore{
        client:     client,
        database:   database,
        collection: collection,
        serializer: NewEventSerializer(),
    }
}

// SaveEvents - save events with optimistic concurrency control
func (s *MongoEventStore) SaveEvents(
    ctx context.Context,
    aggregateID uuid.UUID,
    events []shared.DomainEvent,
    expectedVersion int,
) error {
    if len(events) == 0 {
        return nil
    }

    // Start session for transaction
    session, err := s.client.StartSession()
    if err != nil {
        return fmt.Errorf("failed to start session: %w", err)
    }
    defer session.EndSession(ctx)

    // Execute in transaction
    _, err = session.WithTransaction(ctx, func(ctx context.Context) (interface{}, error) {
        // 1. Check current version (optimistic locking)
        currentVersion, err := s.getCurrentVersion(ctx, aggregateID)
        if err != nil {
            return nil, err
        }

        if currentVersion != expectedVersion {
            return nil, &shared.ConcurrencyError{
                AggregateID:     aggregateID,
                ExpectedVersion: expectedVersion,
                ActualVersion:   currentVersion,
            }
        }

        // 2. Serialize events
        documents := make([]interface{}, len(events))
        for i, event := range events {
            doc, err := s.serializer.Serialize(event)
            if err != nil {
                return nil, fmt.Errorf("failed to serialize event %d: %w", i, err)
            }
            documents[i] = doc
        }

        // 3. Insert events (bulk)
        _, err = s.collection.InsertMany(ctx, documents)
        if err != nil {
            // Check for duplicate key error (concurrency conflict)
            if mongo.IsDuplicateKeyError(err) {
                return nil, &shared.ConcurrencyError{
                    AggregateID:     aggregateID,
                    ExpectedVersion: expectedVersion,
                    ActualVersion:   -1,  // unknown
                }
            }
            return nil, fmt.Errorf("failed to insert events: %w", err)
        }

        return nil, nil
    })

    return err
}

// LoadEvents - load all events for aggregate
func (s *MongoEventStore) LoadEvents(ctx context.Context, aggregateID uuid.UUID) ([]shared.DomainEvent, error) {
    filter := bson.M{"aggregate_id": aggregateID.String()}
    opts := options.Find().SetSort(bson.D{{Key: "version", Value: 1}})

    cursor, err := s.collection.Find(ctx, filter, opts)
    if err != nil {
        return nil, fmt.Errorf("failed to find events: %w", err)
    }
    defer cursor.Close(ctx)

    var events []shared.DomainEvent
    for cursor.Next(ctx) {
        var doc EventDocument
        if err := cursor.Decode(&doc); err != nil {
            return nil, fmt.Errorf("failed to decode event: %w", err)
        }

        event, err := s.serializer.Deserialize(&doc)
        if err != nil {
            return nil, fmt.Errorf("failed to deserialize event: %w", err)
        }

        events = append(events, event)
    }

    if err := cursor.Err(); err != nil {
        return nil, fmt.Errorf("cursor error: %w", err)
    }

    return events, nil
}

// LoadEventsAfter - load events after specific version (for incremental replay)
func (s *MongoEventStore) LoadEventsAfter(ctx context.Context, aggregateID uuid.UUID, version int) ([]shared.DomainEvent, error) {
    filter := bson.M{
        "aggregate_id": aggregateID.String(),
        "version":      bson.M{"$gt": version},
    }
    opts := options.Find().SetSort(bson.D{{Key: "version", Value: 1}})

    cursor, err := s.collection.Find(ctx, filter, opts)
    if err != nil {
        return nil, fmt.Errorf("failed to find events: %w", err)
    }
    defer cursor.Close(ctx)

    var events []shared.DomainEvent
    for cursor.Next(ctx) {
        var doc EventDocument
        if err := cursor.Decode(&doc); err != nil {
            return nil, fmt.Errorf("failed to decode event: %w", err)
        }

        event, err := s.serializer.Deserialize(&doc)
        if err != nil {
            return nil, fmt.Errorf("failed to deserialize event: %w", err)
        }

        events = append(events, event)
    }

    return events, nil
}

// getCurrentVersion - get current aggregate version
func (s *MongoEventStore) getCurrentVersion(ctx context.Context, aggregateID uuid.UUID) (int, error) {
    filter := bson.M{"aggregate_id": aggregateID.String()}
    opts := options.FindOne().SetSort(bson.D{{Key: "version", Value: -1}})

    var doc EventDocument
    err := s.collection.FindOne(ctx, filter, opts).Decode(&doc)
    if err != nil {
        if err == mongo.ErrNoDocuments {
            return 0, nil  // no events yet
        }
        return 0, fmt.Errorf("failed to get current version: %w", err)
    }

    return doc.Version, nil
}

// GetAllEvents - get all events (for replay, admin tools)
func (s *MongoEventStore) GetAllEvents(ctx context.Context) ([]shared.DomainEvent, error) {
    opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})

    cursor, err := s.collection.Find(ctx, bson.M{}, opts)
    if err != nil {
        return nil, fmt.Errorf("failed to find events: %w", err)
    }
    defer cursor.Close(ctx)

    var events []shared.DomainEvent
    for cursor.Next(ctx) {
        var doc EventDocument
        if err := cursor.Decode(&doc); err != nil {
            return nil, fmt.Errorf("failed to decode event: %w", err)
        }

        event, err := s.serializer.Deserialize(&doc)
        if err != nil {
            return nil, fmt.Errorf("failed to deserialize event: %w", err)
        }

        events = append(events, event)
    }

    return events, nil
}
```

---

### 4. Unit Tests (mongodb_store_test.go)

```go
package eventstore_test

import (
    "context"
    "testing"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/lllypuk/flowra/internal/infrastructure/eventstore"
    chatevents "github.com/lllypuk/flowra/internal/domain/chat/events"
    "github.com/lllypuk/flowra/tests/testutil"
)

func TestMongoEventStore_SaveAndLoadEvents(t *testing.T) {
    // Setup MongoDB
    client := testutil.SetupMongoDBv2(t)
    store := eventstore.NewMongoEventStore(client, "test_db")

    ctx := context.Background()
    aggregateID := uuid.New()

    // Create events
    events := []shared.DomainEvent{
        &chatevents.ChatCreated{
            ChatID:      aggregateID,
            WorkspaceID: uuid.New(),
            Type:        "Discussion",
            Title:       "Test Chat",
            CreatedBy:   uuid.New(),
            OccurredAt:  time.Now(),
            Version:     1,
        },
        &chatevents.ParticipantAdded{
            ChatID:     aggregateID,
            UserID:     uuid.New(),
            Role:       "Member",
            OccurredAt: time.Now(),
            Version:    2,
        },
    }

    // Act: Save events
    err := store.SaveEvents(ctx, aggregateID, events, 0)
    require.NoError(t, err)

    // Act: Load events
    loadedEvents, err := store.LoadEvents(ctx, aggregateID)
    require.NoError(t, err)

    // Assert
    assert.Len(t, loadedEvents, 2)
    assert.IsType(t, &chatevents.ChatCreated{}, loadedEvents[0])
    assert.IsType(t, &chatevents.ParticipantAdded{}, loadedEvents[1])
}

func TestMongoEventStore_OptimisticLocking_ConflictDetected(t *testing.T) {
    // Test concurrency conflict detection
}

func TestMongoEventStore_LoadEventsAfter(t *testing.T) {
    // Test incremental event loading
}
```

---

## Критерии успеха

- ✅ **MongoDB Event Store реализован**
- ✅ **Append 100 events < 50ms**
- ✅ **Load 1000 events < 100ms**
- ✅ **Optimistic concurrency control работает**
- ✅ **No data loss** при конкурентной записи
- ✅ **Serialization/Deserialization корректны**
- ✅ **Test coverage >85%**
- ✅ **Integration tests проходят**

---

## Следующий шаг

После завершения → **Task 1.1.2: MongoDB Repositories**
