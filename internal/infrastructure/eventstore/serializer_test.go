package eventstore_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/infrastructure/eventstore"
)

// TestEvent простое событие для тестирования.
type TestEvent struct {
	event.BaseEvent

	TestData string
}

func TestEventSerializer_Serialize(t *testing.T) {
	serializer := eventstore.NewEventSerializer()

	// Создаем тестовое событие
	metadata := event.NewMetadata("user-123", "corr-456", "caus-789")
	baseEvent := event.NewBaseEvent("TestEventCreated", "agg-123", "TestAggregate", 1, metadata)

	// Создаем полное событие
	testEvent := &TestEvent{
		BaseEvent: baseEvent,
		TestData:  "test value",
	}

	// Тестируем сериализацию
	doc, err := serializer.Serialize(testEvent)
	require.NoError(t, err)
	assert.NotNil(t, doc)

	// Проверяем основные поля
	assert.Equal(t, "agg-123", doc.AggregateID)
	assert.Equal(t, "TestAggregate", doc.AggregateType)
	assert.Equal(t, "TestEventCreated", doc.EventType)
	assert.Equal(t, 1, doc.Version)
	assert.Equal(t, "user-123", doc.Metadata.UserID)
	assert.Equal(t, "corr-456", doc.Metadata.CorrelationID)
	assert.Equal(t, "caus-789", doc.Metadata.CausationID)

	// Проверяем что данные сохранены
	assert.NotNil(t, doc.Data)
	assert.Equal(t, "test value", doc.Data["TestData"])

	// Проверяем временные метки
	assert.False(t, doc.OccurredAt.IsZero())
	assert.False(t, doc.CreatedAt.IsZero())
}

func TestEventSerializer_SerializeMany(t *testing.T) {
	serializer := eventstore.NewEventSerializer()

	// Создаем несколько тестовых событий
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

	// Тестируем сериализацию множества
	docs, err := serializer.SerializeMany(events)
	require.NoError(t, err)
	assert.Len(t, docs, 3)

	// Проверяем версии
	assert.Equal(t, 1, docs[0].Version)
	assert.Equal(t, 2, docs[1].Version)
	assert.Equal(t, 3, docs[2].Version)
}

func TestEventSerializer_SerializeWithMetadata(t *testing.T) {
	serializer := eventstore.NewEventSerializer()

	// Создаем событие с полными метаданными
	metadata := event.NewMetadata("user-123", "corr-456", "caus-789")
	metadata = metadata.WithIPAddress("192.168.1.1")
	metadata = metadata.WithUserAgent("Mozilla/5.0")

	baseEvent := event.NewBaseEvent("TestEventCreated", "agg-123", "TestAggregate", 1, metadata)
	testEvent := &TestEvent{
		BaseEvent: baseEvent,
		TestData:  "test value",
	}

	// Тестируем сериализацию
	doc, err := serializer.Serialize(testEvent)
	require.NoError(t, err)

	// Проверяем все метаданные
	assert.Equal(t, "192.168.1.1", doc.Metadata.IPAddress)
	assert.Equal(t, "Mozilla/5.0", doc.Metadata.UserAgent)
	assert.Equal(t, "user-123", doc.Metadata.UserID)
}

func TestEventSerializer_SerializeWithEmptyMetadata(t *testing.T) {
	serializer := eventstore.NewEventSerializer()

	// Создаем событие с пустыми метаданными
	baseEvent := event.NewBaseEvent("TestEventCreated", "agg-123", "TestAggregate", 1, event.Metadata{})
	testEvent := &TestEvent{
		BaseEvent: baseEvent,
		TestData:  "test value",
	}

	// Тестируем сериализацию
	doc, err := serializer.Serialize(testEvent)
	require.NoError(t, err)

	// Проверяем что документ создан без ошибок
	assert.NotNil(t, doc)
	assert.Empty(t, doc.Metadata.UserID)
	assert.Empty(t, doc.Metadata.IPAddress)
}

func TestEventSerializer_PreservesOccurredAtTime(t *testing.T) {
	serializer := eventstore.NewEventSerializer()

	// Создаем событие с конкретным временем
	now := time.Now().Truncate(time.Millisecond)
	metadata := event.NewMetadata("user-123", "corr-456", "")
	metadata.Timestamp = now

	baseEvent := event.NewBaseEvent("TestEventCreated", "agg-123", "TestAggregate", 1, metadata)
	testEvent := &TestEvent{
		BaseEvent: baseEvent,
		TestData:  "test value",
	}

	// Тестируем сериализацию
	doc, err := serializer.Serialize(testEvent)
	require.NoError(t, err)

	// Проверяем что время сохранено
	assert.Equal(t, now, doc.Metadata.Timestamp)
}
