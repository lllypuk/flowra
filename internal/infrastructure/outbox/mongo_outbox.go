// Package outbox provides the transactional outbox pattern implementation.
package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/google/uuid"
	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/event"
)

// outboxDocument represents the MongoDB document structure for outbox entries.
type outboxDocument struct {
	ID            string     `bson:"_id"`
	EventID       string     `bson:"event_id"`
	EventType     string     `bson:"event_type"`
	AggregateID   string     `bson:"aggregate_id"`
	AggregateType string     `bson:"aggregate_type"`
	Payload       []byte     `bson:"payload"`
	CreatedAt     time.Time  `bson:"created_at"`
	ProcessedAt   *time.Time `bson:"processed_at,omitempty"`
	RetryCount    int        `bson:"retry_count"`
	LastError     string     `bson:"last_error,omitempty"`
}

// MongoOutbox implements appcore.Outbox using MongoDB.
type MongoOutbox struct {
	collection *mongo.Collection
	logger     *slog.Logger
}

// Option configures MongoOutbox.
type Option func(*MongoOutbox)

// WithLogger sets the logger for the outbox.
func WithLogger(logger *slog.Logger) Option {
	return func(o *MongoOutbox) {
		o.logger = logger
	}
}

// NewMongoOutbox creates a new MongoDB-backed outbox.
func NewMongoOutbox(collection *mongo.Collection, opts ...Option) *MongoOutbox {
	o := &MongoOutbox{
		collection: collection,
		logger:     slog.Default(),
	}

	for _, opt := range opts {
		opt(o)
	}

	return o
}

// Add inserts an event into the outbox.
func (o *MongoOutbox) Add(ctx context.Context, evt event.DomainEvent) error {
	if evt == nil {
		return errors.New("event cannot be nil")
	}

	doc, err := o.eventToDocument(evt)
	if err != nil {
		return fmt.Errorf("failed to convert event to document: %w", err)
	}

	_, err = o.collection.InsertOne(ctx, doc)
	if err != nil {
		o.logger.ErrorContext(ctx, "failed to insert event into outbox",
			slog.String("event_type", evt.EventType()),
			slog.String("aggregate_id", evt.AggregateID()),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to insert event into outbox: %w", err)
	}

	o.logger.DebugContext(ctx, "event added to outbox",
		slog.String("entry_id", doc.ID),
		slog.String("event_type", evt.EventType()),
		slog.String("aggregate_id", evt.AggregateID()),
	)

	return nil
}

// AddBatch inserts multiple events into the outbox atomically.
func (o *MongoOutbox) AddBatch(ctx context.Context, events []event.DomainEvent) error {
	if len(events) == 0 {
		return nil
	}

	docs := make([]any, len(events))
	for i, evt := range events {
		if evt == nil {
			return fmt.Errorf("event at index %d cannot be nil", i)
		}

		doc, err := o.eventToDocument(evt)
		if err != nil {
			return fmt.Errorf("failed to convert event at index %d: %w", i, err)
		}
		docs[i] = doc
	}

	_, err := o.collection.InsertMany(ctx, docs)
	if err != nil {
		o.logger.ErrorContext(ctx, "failed to insert events batch into outbox",
			slog.Int("count", len(events)),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to insert events batch into outbox: %w", err)
	}

	o.logger.DebugContext(ctx, "events batch added to outbox",
		slog.Int("count", len(events)),
	)

	return nil
}

// Poll retrieves unprocessed events up to the specified batch size.
func (o *MongoOutbox) Poll(ctx context.Context, batchSize int) ([]appcore.OutboxEntry, error) {
	if batchSize <= 0 {
		batchSize = 100
	}

	// Find unprocessed entries, ordered by creation time
	filter := bson.M{"processed_at": nil}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: 1}}).
		SetLimit(int64(batchSize))

	cursor, err := o.collection.Find(ctx, filter, opts)
	if err != nil {
		o.logger.ErrorContext(ctx, "failed to poll outbox",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to poll outbox: %w", err)
	}
	defer cursor.Close(ctx)

	var entries []appcore.OutboxEntry
	for cursor.Next(ctx) {
		var doc outboxDocument
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			o.logger.WarnContext(ctx, "failed to decode outbox entry",
				slog.String("error", decodeErr.Error()),
			)
			continue
		}

		entries = append(entries, o.documentToEntry(&doc))
	}

	if cursorErr := cursor.Err(); cursorErr != nil {
		return nil, fmt.Errorf("cursor error while polling outbox: %w", cursorErr)
	}

	return entries, nil
}

// MarkProcessed marks an event as successfully published.
func (o *MongoOutbox) MarkProcessed(ctx context.Context, entryID string) error {
	now := time.Now().UTC()
	filter := bson.M{"_id": entryID}
	update := bson.M{"$set": bson.M{"processed_at": now}}

	result, err := o.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		o.logger.ErrorContext(ctx, "failed to mark outbox entry as processed",
			slog.String("entry_id", entryID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to mark entry as processed: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("outbox entry not found: %s", entryID)
	}

	o.logger.DebugContext(ctx, "outbox entry marked as processed",
		slog.String("entry_id", entryID),
	)

	return nil
}

// MarkFailed records a publishing failure for retry.
func (o *MongoOutbox) MarkFailed(ctx context.Context, entryID string, publishErr error) error {
	errMsg := ""
	if publishErr != nil {
		errMsg = publishErr.Error()
	}

	filter := bson.M{"_id": entryID}
	update := bson.M{
		"$inc": bson.M{"retry_count": 1},
		"$set": bson.M{"last_error": errMsg},
	}

	result, err := o.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		o.logger.ErrorContext(ctx, "failed to mark outbox entry as failed",
			slog.String("entry_id", entryID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to mark entry as failed: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("outbox entry not found: %s", entryID)
	}

	o.logger.DebugContext(ctx, "outbox entry marked as failed",
		slog.String("entry_id", entryID),
		slog.Int64("modified", result.ModifiedCount),
	)

	return nil
}

// Cleanup removes old processed entries older than the specified duration.
func (o *MongoOutbox) Cleanup(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-olderThan)

	filter := bson.M{
		"processed_at": bson.M{"$ne": nil, "$lt": cutoff},
	}

	result, err := o.collection.DeleteMany(ctx, filter)
	if err != nil {
		o.logger.ErrorContext(ctx, "failed to cleanup outbox",
			slog.String("error", err.Error()),
		)
		return 0, fmt.Errorf("failed to cleanup outbox: %w", err)
	}

	if result.DeletedCount > 0 {
		o.logger.InfoContext(ctx, "cleaned up old outbox entries",
			slog.Int64("deleted", result.DeletedCount),
			slog.Duration("older_than", olderThan),
		)
	}

	return result.DeletedCount, nil
}

// Count returns the number of unprocessed entries.
func (o *MongoOutbox) Count(ctx context.Context) (int64, error) {
	filter := bson.M{"processed_at": nil}
	count, err := o.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count outbox entries: %w", err)
	}
	return count, nil
}

// Stats returns statistics about the outbox (count and oldest entry timestamp).
func (o *MongoOutbox) Stats(ctx context.Context) (int64, time.Time, error) {
	filter := bson.M{"processed_at": nil}

	// Count unprocessed entries
	count, err := o.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("failed to count unprocessed entries: %w", err)
	}

	// If no entries, return zero time
	if count == 0 {
		return 0, time.Time{}, nil
	}

	// Find oldest entry
	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: 1}})
	var doc outboxDocument
	err = o.collection.FindOne(ctx, filter, opts).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return count, time.Time{}, nil
		}
		return count, time.Time{}, fmt.Errorf("failed to find oldest entry: %w", err)
	}

	return count, doc.CreatedAt, nil
}

// eventToDocument converts a domain event to an outbox document.
func (o *MongoOutbox) eventToDocument(evt event.DomainEvent) (*outboxDocument, error) {
	payload, err := json.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event payload: %w", err)
	}

	return &outboxDocument{
		ID:            uuid.New().String(),
		EventID:       uuid.New().String(),
		EventType:     evt.EventType(),
		AggregateID:   evt.AggregateID(),
		AggregateType: evt.AggregateType(),
		Payload:       payload,
		CreatedAt:     time.Now().UTC(),
		RetryCount:    0,
	}, nil
}

// documentToEntry converts an outbox document to an OutboxEntry.
func (o *MongoOutbox) documentToEntry(doc *outboxDocument) appcore.OutboxEntry {
	return appcore.OutboxEntry{
		ID:            doc.ID,
		EventID:       doc.EventID,
		EventType:     doc.EventType,
		AggregateID:   doc.AggregateID,
		AggregateType: doc.AggregateType,
		Payload:       doc.Payload,
		CreatedAt:     doc.CreatedAt,
		ProcessedAt:   doc.ProcessedAt,
		RetryCount:    doc.RetryCount,
		LastError:     doc.LastError,
	}
}

// Ensure MongoOutbox implements appcore.Outbox.
var _ appcore.Outbox = (*MongoOutbox)(nil)
