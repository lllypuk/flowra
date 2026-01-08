package eventstore

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/event"
)

// MongoEventStore реализует EventStore с использованием MongoDB
type MongoEventStore struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
	serializer *EventSerializer
	logger     *slog.Logger
}

// Option configures MongoEventStore.
type Option func(*MongoEventStore)

// WithLogger sets the logger for event store.
func WithLogger(logger *slog.Logger) Option {
	return func(s *MongoEventStore) {
		s.logger = logger
	}
}

// NewMongoEventStore creates New MongoDB Event Store
func NewMongoEventStore(client *mongo.Client, databaseName string, opts ...Option) *MongoEventStore {
	database := client.Database(databaseName)
	collection := database.Collection("events")

	s := &MongoEventStore{
		client:     client,
		database:   database,
		collection: collection,
		serializer: NewEventSerializer(),
		logger:     slog.Default(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// SaveEvents saves event for aggregate с оптимистичной блокировкой
func (s *MongoEventStore) SaveEvents(
	ctx context.Context,
	aggregateID string,
	events []event.DomainEvent,
	expectedVersion int,
) error {
	if len(events) == 0 {
		return nil
	}

	// Runningаем сессию for транзакции
	session, err := s.client.StartSession()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to start MongoDB session for event store",
			slog.String("aggregate_id", aggregateID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	// Выполняем операцию in транзакции
	_, err = session.WithTransaction(ctx, func(txCtx context.Context) (any, error) {
		// 1. Checking current version (optimistic locking)
		currentVersion, errVersion := s.getCurrentVersion(txCtx, aggregateID)
		if errVersion != nil {
			s.logger.ErrorContext(ctx, "failed to get current version for aggregate",
				slog.String("aggregate_id", aggregateID),
				slog.String("error", errVersion.Error()),
			)
			return nil, errVersion
		}

		if currentVersion != expectedVersion {
			s.logger.WarnContext(ctx, "concurrency conflict in event store",
				slog.String("aggregate_id", aggregateID),
				slog.Int("expected_version", expectedVersion),
				slog.Int("current_version", currentVersion),
			)
			return nil, appcore.ErrConcurrencyConflict
		}

		// 2. Serializing event
		documents, errSerialize := s.serializer.SerializeMany(events)
		if errSerialize != nil {
			s.logger.ErrorContext(ctx, "failed to serialize events",
				slog.String("aggregate_id", aggregateID),
				slog.Int("events_count", len(events)),
				slog.String("error", errSerialize.Error()),
			)
			return nil, errSerialize
		}

		// 3. Преобразуем in interface{} for InsertMany
		docs := make([]any, len(documents))
		for i, doc := range documents {
			docs[i] = doc
		}

		// 4. Вставляем event (bulk)
		_, errInsert := s.collection.InsertMany(txCtx, docs)
		if errInsert != nil {
			// Checking error дублирования ключа (конфликт concurrency)
			if mongo.IsDuplicateKeyError(errInsert) {
				s.logger.WarnContext(ctx, "duplicate key error in event store (concurrency)",
					slog.String("aggregate_id", aggregateID),
					slog.Int("events_count", len(events)),
				)
				return nil, appcore.ErrConcurrencyConflict
			}
			s.logger.ErrorContext(ctx, "failed to insert events to event store",
				slog.String("aggregate_id", aggregateID),
				slog.Int("events_count", len(events)),
				slog.String("error", errInsert.Error()),
			)
			return nil, fmt.Errorf("failed to insert events: %w", errInsert)
		}

		return nil, nil //nolint:nilnil // Transaction success returns nil for both values
	})

	if err != nil && !errors.Is(err, appcore.ErrConcurrencyConflict) {
		s.logger.ErrorContext(ctx, "event store transaction failed",
			slog.String("aggregate_id", aggregateID),
			slog.Int("events_count", len(events)),
			slog.String("error", err.Error()),
		)
	}

	return err
}

// LoadEvents loads all event for aggregate
func (s *MongoEventStore) LoadEvents(ctx context.Context, aggregateID string) ([]event.DomainEvent, error) {
	filter := bson.M{"aggregate_id": aggregateID}
	opts := options.Find().SetSort(bson.D{{Key: "version", Value: 1}})

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to find events in event store",
			slog.String("aggregate_id", aggregateID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to find events: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []*EventDocument
	if err = cursor.All(ctx, &docs); err != nil {
		s.logger.ErrorContext(ctx, "failed to decode events from event store",
			slog.String("aggregate_id", aggregateID),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to decode events: %w", err)
	}

	// if no документов, возвращаем error
	if len(docs) == 0 {
		return nil, appcore.ErrAggregateNotFound
	}

	// Деserializing event via сериализатор
	events, err := s.serializer.DeserializeMany(docs)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to deserialize events from event store",
			slog.String("aggregate_id", aggregateID),
			slog.Int("docs_count", len(docs)),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return events, nil
}

// GetVersion returns current version aggregate
func (s *MongoEventStore) GetVersion(ctx context.Context, aggregateID string) (int, error) {
	filter := bson.M{"aggregate_id": aggregateID}
	opts := options.FindOne().SetSort(bson.D{{Key: "version", Value: -1}})

	var doc EventDocument
	err := s.collection.FindOne(ctx, filter, opts).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, nil // no events еще
		}
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}

	return doc.Version, nil
}

// getCurrentVersion receivает current version aggregate (внутренний method)
func (s *MongoEventStore) getCurrentVersion(ctx context.Context, aggregateID string) (int, error) {
	return s.GetVersion(ctx, aggregateID)
}
