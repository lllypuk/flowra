package eventstore_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/infrastructure/eventstore"
)

// TestEvent простое event for testing.
type TestEvent struct {
	event.BaseEvent

	TestData string
}

func TestEventSerializer_Serialize(t *testing.T) {
	serializer := eventstore.NewEventSerializer()

	// Creating тестовое event
	metadata := event.NewMetadata("user-123", "corr-456", "caus-789")
	baseEvent := event.NewBaseEvent("TestEventCreated", "agg-123", "TestAggregate", 1, metadata)

	// Creating полное event
	testEvent := &TestEvent{
		BaseEvent: baseEvent,
		TestData:  "test value",
	}

	// Testing serialization
	doc, err := serializer.Serialize(testEvent)
	require.NoError(t, err)
	assert.NotNil(t, doc)

	// Checking основные fields
	assert.Equal(t, "agg-123", doc.AggregateID)
	assert.Equal(t, "TestAggregate", doc.AggregateType)
	assert.Equal(t, "TestEventCreated", doc.EventType)
	assert.Equal(t, 1, doc.Version)
	assert.Equal(t, "user-123", doc.Metadata.UserID)
	assert.Equal(t, "corr-456", doc.Metadata.CorrelationID)
	assert.Equal(t, "caus-789", doc.Metadata.CausationID)

	// Checking that data savены
	assert.NotNil(t, doc.Data)
	assert.Equal(t, "test value", doc.Data["TestData"])

	// Checking временные метки
	assert.False(t, doc.OccurredAt.IsZero())
	assert.False(t, doc.CreatedAt.IsZero())
}

func TestEventSerializer_SerializeMany(t *testing.T) {
	serializer := eventstore.NewEventSerializer()

	// Creating several тестовых events
	events := []event.DomainEvent{}
	for i := 1; i <= 3; i++ {
		metadata := event.NewMetadata("user-123", "corr-456", "")
		baseEvent := event.NewBaseEvent(
			"TestEventCreated",
			"agg-123",
			"TestAggregate",
			i,
			metadata,
		)
		events = append(events, &TestEvent{
			BaseEvent: baseEvent,
			TestData:  "value-" + string(rune(i)),
		})
	}

	// Testing serialization множества
	docs, err := serializer.SerializeMany(events)
	require.NoError(t, err)
	assert.Len(t, docs, 3)

	// Checking версии
	assert.Equal(t, 1, docs[0].Version)
	assert.Equal(t, 2, docs[1].Version)
	assert.Equal(t, 3, docs[2].Version)
}

func TestEventSerializer_SerializeWithMetadata(t *testing.T) {
	serializer := eventstore.NewEventSerializer()

	// Creating event с полными метаданными
	metadata := event.NewMetadata("user-123", "corr-456", "caus-789")
	metadata = metadata.WithIPAddress("192.168.1.1")
	metadata = metadata.WithUserAgent("Mozilla/5.0")

	baseEvent := event.NewBaseEvent("TestEventCreated", "agg-123", "TestAggregate", 1, metadata)
	testEvent := &TestEvent{
		BaseEvent: baseEvent,
		TestData:  "test value",
	}

	// Testing serialization
	doc, err := serializer.Serialize(testEvent)
	require.NoError(t, err)

	// Checking all метаданные
	assert.Equal(t, "192.168.1.1", doc.Metadata.IPAddress)
	assert.Equal(t, "Mozilla/5.0", doc.Metadata.UserAgent)
	assert.Equal(t, "user-123", doc.Metadata.UserID)
}

func TestEventSerializer_SerializeWithEmptyMetadata(t *testing.T) {
	serializer := eventstore.NewEventSerializer()

	// Creating event с пустыми метаданными
	baseEvent := event.NewBaseEvent("TestEventCreated", "agg-123", "TestAggregate", 1, event.Metadata{})
	testEvent := &TestEvent{
		BaseEvent: baseEvent,
		TestData:  "test value",
	}

	// Testing serialization
	doc, err := serializer.Serialize(testEvent)
	require.NoError(t, err)

	// Checking that документ создан без errors
	assert.NotNil(t, doc)
	assert.Empty(t, doc.Metadata.UserID)
	assert.Empty(t, doc.Metadata.IPAddress)
}

func TestEventSerializer_PreservesOccurredAtTime(t *testing.T) {
	serializer := eventstore.NewEventSerializer()

	// Creating event с specific временем
	now := time.Now().Truncate(time.Millisecond)
	metadata := event.NewMetadata("user-123", "corr-456", "")
	metadata.Timestamp = now

	baseEvent := event.NewBaseEvent("TestEventCreated", "agg-123", "TestAggregate", 1, metadata)
	testEvent := &TestEvent{
		BaseEvent: baseEvent,
		TestData:  "test value",
	}

	// Testing serialization
	doc, err := serializer.Serialize(testEvent)
	require.NoError(t, err)

	// Checking that time savено
	assert.Equal(t, now, doc.Metadata.Timestamp)
}
