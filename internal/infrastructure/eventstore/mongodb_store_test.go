//go:build integration

package eventstore_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/infrastructure/eventstore"
	"github.com/lllypuk/flowra/tests/testutil"
)

func TestMongoEventStore_SaveAndLoadEvents(t *testing.T) {
	// Setup
	db := testutil.SetupTestMongoDB(t)
	client := db.Client()
	store := eventstore.NewMongoEventStore(client, db.Name())
	ctx := context.Background()

	// Prepare test data
	aggregateID := "test-agg-123"
	metadata := event.NewMetadata("user-123", "corr-456", "")
	baseEvent := event.NewBaseEvent("TestEventCreated", aggregateID, "TestAggregate", 1, metadata)
	testEvent := &TestEvent{
		BaseEvent: baseEvent,
		TestData:  "test value",
	}

	// Act: Save events
	err := store.SaveEvents(ctx, aggregateID, []event.DomainEvent{testEvent}, 0)
	require.NoError(t, err)

	// Act: Load events
	loadedEvents, err := store.LoadEvents(ctx, aggregateID)
	require.NoError(t, err)

	// Assert
	assert.Len(t, loadedEvents, 1)
	assert.Equal(t, aggregateID, loadedEvents[0].AggregateID())
	assert.Equal(t, "TestEventCreated", loadedEvents[0].EventType())
	assert.Equal(t, "TestAggregate", loadedEvents[0].AggregateType())
	assert.Equal(t, 1, loadedEvents[0].Version())
}

func TestMongoEventStore_SaveMultipleEvents(t *testing.T) {
	// Setup
	db := testutil.SetupTestMongoDB(t)
	client := db.Client()
	store := eventstore.NewMongoEventStore(client, db.Name())
	ctx := context.Background()

	// Prepare test data
	aggregateID := "test-agg-123"
	events := []event.DomainEvent{}

	for i := 1; i <= 3; i++ {
		metadata := event.NewMetadata("user-123", "corr-456", "")
		baseEvent := event.NewBaseEvent(
			"TestEventCreated",
			aggregateID,
			"TestAggregate",
			i,
			metadata,
		)
		events = append(events, &TestEvent{
			BaseEvent: baseEvent,
			TestData:  "value-" + string(rune(i)),
		})
	}

	// Act: Save events
	err := store.SaveEvents(ctx, aggregateID, events, 0)
	require.NoError(t, err)

	// Act: Load events
	loadedEvents, err := store.LoadEvents(ctx, aggregateID)
	require.NoError(t, err)

	// Assert
	assert.Len(t, loadedEvents, 3)
	assert.Equal(t, 1, loadedEvents[0].Version())
	assert.Equal(t, 2, loadedEvents[1].Version())
	assert.Equal(t, 3, loadedEvents[2].Version())
}

func TestMongoEventStore_OptimisticLocking_ConflictDetected(t *testing.T) {
	// Setup
	db := testutil.SetupTestMongoDB(t)
	client := db.Client()
	store := eventstore.NewMongoEventStore(client, db.Name())
	ctx := context.Background()

	// Prepare test data
	aggregateID := "test-agg-123"
	metadata := event.NewMetadata("user-123", "corr-456", "")
	baseEvent := event.NewBaseEvent("TestEventCreated", aggregateID, "TestAggregate", 1, metadata)
	testEvent := &TestEvent{
		BaseEvent: baseEvent,
		TestData:  "test value",
	}

	// Act: Save first event (expectedVersion = 0, aggregate doesn't exist)
	err := store.SaveEvents(ctx, aggregateID, []event.DomainEvent{testEvent}, 0)
	require.NoError(t, err)

	// Act: Try to save with wrong expectedVersion (should fail)
	baseEvent2 := event.NewBaseEvent("TestEventCreated", aggregateID, "TestAggregate", 2, metadata)
	testEvent2 := &TestEvent{
		BaseEvent: baseEvent2,
		TestData:  "test value 2",
	}
	err = store.SaveEvents(ctx, aggregateID, []event.DomainEvent{testEvent2}, 0)

	// Assert
	assert.Equal(t, appcore.ErrConcurrencyConflict, err)
}

func TestMongoEventStore_OptimisticLocking_CorrectExpectedVersion(t *testing.T) {
	// Setup
	db := testutil.SetupTestMongoDB(t)
	client := db.Client()
	store := eventstore.NewMongoEventStore(client, db.Name())
	ctx := context.Background()

	// Prepare test data
	aggregateID := "test-agg-123"
	metadata := event.NewMetadata("user-123", "corr-456", "")

	// First event
	baseEvent := event.NewBaseEvent("TestEventCreated", aggregateID, "TestAggregate", 1, metadata)
	testEvent := &TestEvent{
		BaseEvent: baseEvent,
		TestData:  "test value",
	}

	// Act: Save first event
	err := store.SaveEvents(ctx, aggregateID, []event.DomainEvent{testEvent}, 0)
	require.NoError(t, err)

	// Second event with correct expectedVersion
	baseEvent2 := event.NewBaseEvent("TestEventCreated", aggregateID, "TestAggregate", 2, metadata)
	testEvent2 := &TestEvent{
		BaseEvent: baseEvent2,
		TestData:  "test value 2",
	}
	err = store.SaveEvents(ctx, aggregateID, []event.DomainEvent{testEvent2}, 1)

	// Assert
	require.NoError(t, err)

	// Verify both events are saved
	loadedEvents, err := store.LoadEvents(ctx, aggregateID)
	require.NoError(t, err)
	assert.Len(t, loadedEvents, 2)
}

func TestMongoEventStore_GetVersion_NewAggregate(t *testing.T) {
	// Setup
	db := testutil.SetupTestMongoDB(t)
	client := db.Client()
	store := eventstore.NewMongoEventStore(client, db.Name())
	ctx := context.Background()

	// Act
	version, err := store.GetVersion(ctx, "non-existent-agg")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 0, version)
}

func TestMongoEventStore_GetVersion_ExistingAggregate(t *testing.T) {
	// Setup
	db := testutil.SetupTestMongoDB(t)
	client := db.Client()
	store := eventstore.NewMongoEventStore(client, db.Name())
	ctx := context.Background()

	// Prepare test data
	aggregateID := "test-agg-123"
	metadata := event.NewMetadata("user-123", "corr-456", "")
	baseEvent := event.NewBaseEvent("TestEventCreated", aggregateID, "TestAggregate", 1, metadata)
	testEvent := &TestEvent{
		BaseEvent: baseEvent,
		TestData:  "test value",
	}

	// Act: Save event
	err := store.SaveEvents(ctx, aggregateID, []event.DomainEvent{testEvent}, 0)
	require.NoError(t, err)

	// Act: Get version
	version, err := store.GetVersion(ctx, aggregateID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 1, version)
}

func TestMongoEventStore_LoadEvents_NotFound(t *testing.T) {
	// Setup
	db := testutil.SetupTestMongoDB(t)
	client := db.Client()
	store := eventstore.NewMongoEventStore(client, db.Name())
	ctx := context.Background()

	// Act
	events, err := store.LoadEvents(ctx, "non-existent-agg")

	// Assert
	assert.Equal(t, appcore.ErrAggregateNotFound, err)
	assert.nil(t, events)
}

func TestMongoEventStore_SaveEmptyEventsList(t *testing.T) {
	// Setup
	db := testutil.SetupTestMongoDB(t)
	client := db.Client()
	store := eventstore.NewMongoEventStore(client, db.Name())
	ctx := context.Background()

	// Act
	err := store.SaveEvents(ctx, "test-agg", []event.DomainEvent{}, 0)

	// Assert
	require.NoError(t, err)
}

func TestMongoEventStore_SaveEventsInOrder(t *testing.T) {
	// Setup
	db := testutil.SetupTestMongoDB(t)
	client := db.Client()
	store := eventstore.NewMongoEventStore(client, db.Name())
	ctx := context.Background()

	// Prepare test data
	aggregateID := "test-agg-123"
	events := []event.DomainEvent{}

	for i := 1; i <= 5; i++ {
		metadata := event.NewMetadata("user-123", "corr-456", "")
		// Немного задержки for разных временных меток
		metadata.Timestamp = time.Now().Add(time.Duration(i) * time.Millisecond)
		baseEvent := event.NewBaseEvent(
			"TestEventCreated",
			aggregateID,
			"TestAggregate",
			i,
			metadata,
		)
		events = append(events, &TestEvent{
			BaseEvent: baseEvent,
			TestData:  "value-" + string(rune('0'+byte(i))),
		})
	}

	// Act: Save events
	err := store.SaveEvents(ctx, aggregateID, events, 0)
	require.NoError(t, err)

	// Act: Load events
	loadedEvents, err := store.LoadEvents(ctx, aggregateID)
	require.NoError(t, err)

	// Assert: Events should be in order by version
	assert.Len(t, loadedEvents, 5)
	for i, e := range loadedEvents {
		assert.Equal(t, i+1, e.Version())
	}
}

func TestMongoEventStore_MongodbClient(t *testing.T) {
	// Test with direct mongo client
	db := testutil.SetupTestMongoDB(t)
	client := db.Client()
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	store := eventstore.NewMongoEventStore(client, db.Name())
	assert.NotNil(t, store)
}

func TestMongoEventStore_ConcurrentSaves(t *testing.T) {
	// Setup
	db := testutil.SetupTestMongoDB(t)
	client := db.Client()
	store := eventstore.NewMongoEventStore(client, db.Name())
	ctx := context.Background()

	aggregateID := "test-agg-123"

	// First save
	metadata := event.NewMetadata("user-123", "corr-456", "")
	baseEvent := event.NewBaseEvent("TestEventCreated", aggregateID, "TestAggregate", 1, metadata)
	testEvent := &TestEvent{
		BaseEvent: baseEvent,
		TestData:  "value 1",
	}

	err := store.SaveEvents(ctx, aggregateID, []event.DomainEvent{testEvent}, 0)
	require.NoError(t, err)

	// Second concurrent save with different expected version
	baseEvent2 := event.NewBaseEvent("TestEventCreated", aggregateID, "TestAggregate", 2, metadata)
	testEvent2 := &TestEvent{
		BaseEvent: baseEvent2,
		TestData:  "value 2",
	}

	// This should succeed because expectedVersion is 1
	err = store.SaveEvents(ctx, aggregateID, []event.DomainEvent{testEvent2}, 1)
	require.NoError(t, err)

	// Load all events
	loadedEvents, err := store.LoadEvents(ctx, aggregateID)
	require.NoError(t, err)
	assert.Len(t, loadedEvents, 2)
}
