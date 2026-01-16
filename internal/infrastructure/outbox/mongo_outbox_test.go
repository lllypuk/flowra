package outbox_test

import (
	"context"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/infrastructure/outbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// mockEvent implements event.DomainEvent for testing.
type mockEvent struct {
	eventType     string
	aggregateID   string
	aggregateType string
	occurredAt    time.Time
	version       int
	metadata      event.Metadata
}

func (e *mockEvent) EventType() string        { return e.eventType }
func (e *mockEvent) AggregateID() string      { return e.aggregateID }
func (e *mockEvent) AggregateType() string    { return e.aggregateType }
func (e *mockEvent) OccurredAt() time.Time    { return e.occurredAt }
func (e *mockEvent) Version() int             { return e.version }
func (e *mockEvent) Metadata() event.Metadata { return e.metadata }

func newMockEvent(eventType, aggregateID, aggregateType string) *mockEvent {
	return &mockEvent{
		eventType:     eventType,
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		occurredAt:    time.Now().UTC(),
		version:       1,
		metadata:      event.Metadata{},
	}
}

// setupTestCollection creates a test MongoDB collection.
// Returns nil if MongoDB is not available (skip test).
func setupTestCollection(t *testing.T) *mongo.Collection {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to connect to MongoDB
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skip("MongoDB not available for testing")
		return nil
	}

	// Ping to verify connection
	if pingErr := client.Ping(ctx, nil); pingErr != nil {
		t.Skip("MongoDB not available for testing")
		return nil
	}

	// Create a unique test database/collection
	dbName := "test_outbox_" + time.Now().Format("20060102150405")
	db := client.Database(dbName)
	collection := db.Collection("outbox")

	// Cleanup on test completion
	t.Cleanup(func() {
		_ = db.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	})

	return collection
}

func TestMongoOutbox_Add(t *testing.T) {
	collection := setupTestCollection(t)
	if collection == nil {
		return
	}

	ob := outbox.NewMongoOutbox(collection)
	ctx := context.Background()

	evt := newMockEvent("chat.created", "chat-123", "chat")

	err := ob.Add(ctx, evt)
	require.NoError(t, err)

	// Verify the event was added
	count, err := collection.CountDocuments(ctx, bson.M{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestMongoOutbox_AddBatch(t *testing.T) {
	collection := setupTestCollection(t)
	if collection == nil {
		return
	}

	ob := outbox.NewMongoOutbox(collection)
	ctx := context.Background()

	events := []event.DomainEvent{
		newMockEvent("chat.created", "chat-1", "chat"),
		newMockEvent("chat.updated", "chat-2", "chat"),
		newMockEvent("task.created", "task-1", "task"),
	}

	err := ob.AddBatch(ctx, events)
	require.NoError(t, err)

	// Verify all events were added
	count, err := collection.CountDocuments(ctx, bson.M{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestMongoOutbox_Poll(t *testing.T) {
	collection := setupTestCollection(t)
	if collection == nil {
		return
	}

	ob := outbox.NewMongoOutbox(collection)
	ctx := context.Background()

	// Add some events
	events := []event.DomainEvent{
		newMockEvent("chat.created", "chat-1", "chat"),
		newMockEvent("chat.updated", "chat-2", "chat"),
	}
	err := ob.AddBatch(ctx, events)
	require.NoError(t, err)

	// Poll for events
	entries, err := ob.Poll(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	// Verify entries have correct data
	assert.Equal(t, "chat.created", entries[0].EventType)
	assert.Equal(t, "chat-1", entries[0].AggregateID)
	assert.Equal(t, "chat", entries[0].AggregateType)
}

func TestMongoOutbox_MarkProcessed(t *testing.T) {
	collection := setupTestCollection(t)
	if collection == nil {
		return
	}

	ob := outbox.NewMongoOutbox(collection)
	ctx := context.Background()

	// Add an event
	evt := newMockEvent("chat.created", "chat-123", "chat")
	err := ob.Add(ctx, evt)
	require.NoError(t, err)

	// Poll to get the entry ID
	entries, err := ob.Poll(ctx, 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	entryID := entries[0].ID

	// Mark as processed
	err = ob.MarkProcessed(ctx, entryID)
	require.NoError(t, err)

	// Poll again - should be empty (processed entries are not returned)
	entries, err = ob.Poll(ctx, 10)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestMongoOutbox_MarkFailed(t *testing.T) {
	collection := setupTestCollection(t)
	if collection == nil {
		return
	}

	ob := outbox.NewMongoOutbox(collection)
	ctx := context.Background()

	// Add an event
	evt := newMockEvent("chat.created", "chat-123", "chat")
	err := ob.Add(ctx, evt)
	require.NoError(t, err)

	// Poll to get the entry ID
	entries, err := ob.Poll(ctx, 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	entryID := entries[0].ID

	// Mark as failed
	testErr := assert.AnError
	err = ob.MarkFailed(ctx, entryID, testErr)
	require.NoError(t, err)

	// Poll again - should still return the entry (not processed)
	entries, err = ob.Poll(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 1, entries[0].RetryCount)
	assert.Contains(t, entries[0].LastError, "assert.AnError")
}

func TestMongoOutbox_Cleanup(t *testing.T) {
	collection := setupTestCollection(t)
	if collection == nil {
		return
	}

	ob := outbox.NewMongoOutbox(collection)
	ctx := context.Background()

	// Add and process an event
	evt := newMockEvent("chat.created", "chat-123", "chat")
	err := ob.Add(ctx, evt)
	require.NoError(t, err)

	entries, err := ob.Poll(ctx, 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	err = ob.MarkProcessed(ctx, entries[0].ID)
	require.NoError(t, err)

	// Cleanup with very short duration should remove the entry
	deleted, err := ob.Cleanup(ctx, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Verify the entry was deleted
	count, err := collection.CountDocuments(ctx, bson.M{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestMongoOutbox_Count(t *testing.T) {
	collection := setupTestCollection(t)
	if collection == nil {
		return
	}

	ob := outbox.NewMongoOutbox(collection)
	ctx := context.Background()

	// Initially should be 0
	count, err := ob.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Add some events
	events := []event.DomainEvent{
		newMockEvent("chat.created", "chat-1", "chat"),
		newMockEvent("chat.updated", "chat-2", "chat"),
	}
	err = ob.AddBatch(ctx, events)
	require.NoError(t, err)

	// Count should be 2
	count, err = ob.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Mark one as processed
	entries, _ := ob.Poll(ctx, 10)
	_ = ob.MarkProcessed(ctx, entries[0].ID)

	// Count should be 1 (only unprocessed)
	count, err = ob.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestMongoOutbox_AddNilEvent(t *testing.T) {
	collection := setupTestCollection(t)
	if collection == nil {
		return
	}

	ob := outbox.NewMongoOutbox(collection)
	ctx := context.Background()

	err := ob.Add(ctx, nil)
	assert.Error(t, err)
}

func TestMongoOutbox_AddBatchWithNilEvent(t *testing.T) {
	collection := setupTestCollection(t)
	if collection == nil {
		return
	}

	ob := outbox.NewMongoOutbox(collection)
	ctx := context.Background()

	events := []event.DomainEvent{
		newMockEvent("chat.created", "chat-1", "chat"),
		nil,
	}

	err := ob.AddBatch(ctx, events)
	assert.Error(t, err)
}

func TestMongoOutbox_MarkProcessedNotFound(t *testing.T) {
	collection := setupTestCollection(t)
	if collection == nil {
		return
	}

	ob := outbox.NewMongoOutbox(collection)
	ctx := context.Background()

	err := ob.MarkProcessed(ctx, "non-existent-id")
	assert.Error(t, err)
}

// Ensure MongoOutbox implements appcore.Outbox.
var _ appcore.Outbox = (*outbox.MongoOutbox)(nil)
