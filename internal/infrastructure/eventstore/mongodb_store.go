package eventstore

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

// NewMongoEventStore создает новый MongoDB Event Store
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

// SaveEvents сохраняет события для агрегата с оптимистичной блокировкой
func (s *MongoEventStore) SaveEvents(
	ctx context.Context,
	aggregateID string,
	events []event.DomainEvent,
	expectedVersion int,
) error {
	if len(events) == 0 {
		return nil
	}

	// Запускаем сессию для транзакции
	session, err := s.client.StartSession()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to start MongoDB session for event store",
			slog.String("aggregate_id", aggregateID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	// Выполняем операцию в транзакции
	_, err = session.WithTransaction(ctx, func(txCtx context.Context) (any, error) {
		// 1. Проверяем текущую версию (оптимистичная блокировка)
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

		// 2. Сериализуем события
		documents, errSerialize := s.serializer.SerializeMany(events)
		if errSerialize != nil {
			s.logger.ErrorContext(ctx, "failed to serialize events",
				slog.String("aggregate_id", aggregateID),
				slog.Int("events_count", len(events)),
				slog.String("error", errSerialize.Error()),
			)
			return nil, errSerialize
		}

		// 3. Преобразуем в interface{} для InsertMany
		docs := make([]any, len(documents))
		for i, doc := range documents {
			docs[i] = doc
		}

		// 4. Вставляем события (bulk)
		_, errInsert := s.collection.InsertMany(txCtx, docs)
		if errInsert != nil {
			// Проверяем ошибку дублирования ключа (конфликт concurrency)
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

// LoadEvents загружает все события для агрегата
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

	// Если нет документов, возвращаем ошибку
	if len(docs) == 0 {
		return nil, appcore.ErrAggregateNotFound
	}

	// Десериализуем события
	events, err := s.deserializeMany(docs)
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

// GetVersion возвращает текущую версию агрегата
func (s *MongoEventStore) GetVersion(ctx context.Context, aggregateID string) (int, error) {
	filter := bson.M{"aggregate_id": aggregateID}
	opts := options.FindOne().SetSort(bson.D{{Key: "version", Value: -1}})

	var doc EventDocument
	err := s.collection.FindOne(ctx, filter, opts).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, nil // Нет событий еще
		}
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}

	return doc.Version, nil
}

// getCurrentVersion получает текущую версию агрегата (внутренний метод)
func (s *MongoEventStore) getCurrentVersion(ctx context.Context, aggregateID string) (int, error) {
	return s.GetVersion(ctx, aggregateID)
}

// deserializeMany десериализует несколько документов в события
func (s *MongoEventStore) deserializeMany(docs []*EventDocument) ([]event.DomainEvent, error) {
	events := make([]event.DomainEvent, 0, len(docs))

	for i, doc := range docs {
		e, err := s.deserializeDocument(doc)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize event at index %d: %w", i, err)
		}
		events = append(events, e)
	}

	return events, nil
}

// deserializeDocument десериализует один документ в событие
func (s *MongoEventStore) deserializeDocument(doc *EventDocument) (event.DomainEvent, error) {
	// Преобразуем BSON.M обратно в JSON для десериализации
	jsonData, err := bson.MarshalExtJSON(doc.Data, false, false)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal BSON to JSON: %w", err)
	}

	// Создаем объект события в зависимости от типа
	// Для простоты используем generic подход через JSON
	var eventData map[string]any
	if errUnmarshal := json.Unmarshal(jsonData, &eventData); errUnmarshal != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", errUnmarshal)
	}

	// Создаем метаданные события
	metadata := event.Metadata{
		UserID:        doc.Metadata.UserID,
		CorrelationID: doc.Metadata.CorrelationID,
		CausationID:   doc.Metadata.CausationID,
		Timestamp:     doc.Metadata.Timestamp,
		IPAddress:     doc.Metadata.IPAddress,
		UserAgent:     doc.Metadata.UserAgent,
	}

	// Создаем базовое событие с восстановленными данными
	baseEvent := event.NewBaseEvent(
		doc.EventType,
		doc.AggregateID,
		doc.AggregateType,
		doc.Version,
		metadata,
	)

	// Для полной десериализации специфичных типов событий,
	// нужно использовать тип события для создания конкретного объекта
	// Здесь мы возвращаем wrapper, который содержит все необходимые данные
	return &StoredEvent{
		BaseEvent: baseEvent,
		Data:      eventData,
	}, nil
}

// StoredEvent представляет событие, загруженное из хранилища
// Это временный wrapper для восстановления полной информации о событии
type StoredEvent struct {
	BaseEvent event.BaseEvent
	Data      map[string]any
}

// EventType возвращает тип события
func (e *StoredEvent) EventType() string {
	return e.BaseEvent.EventType()
}

// AggregateID возвращает ID агрегата
func (e *StoredEvent) AggregateID() string {
	return e.BaseEvent.AggregateID()
}

// AggregateType возвращает тип агрегата
func (e *StoredEvent) AggregateType() string {
	return e.BaseEvent.AggregateType()
}

// OccurredAt возвращает время возникновения события
func (e *StoredEvent) OccurredAt() time.Time {
	return e.BaseEvent.OccurredAt()
}

// Version возвращает версию агрегата
func (e *StoredEvent) Version() int {
	return e.BaseEvent.Version()
}

// Metadata возвращает метаданные события
func (e *StoredEvent) Metadata() event.Metadata {
	return e.BaseEvent.Metadata()
}
