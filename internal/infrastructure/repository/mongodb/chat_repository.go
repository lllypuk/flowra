package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	"github.com/lllypuk/flowra/internal/application/shared"
	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// MongoChatRepository реализует chatapp.CommandRepository (application layer interface)
// с использованием MongoDB и Event Sourcing
type MongoChatRepository struct {
	eventStore    shared.EventStore
	readModelColl *mongo.Collection
}

// NewMongoChatRepository создает новый MongoDB Chat Repository
func NewMongoChatRepository(eventStore shared.EventStore, readModelColl *mongo.Collection) *MongoChatRepository {
	return &MongoChatRepository{
		eventStore:    eventStore,
		readModelColl: readModelColl,
	}
}

// Load загружает Chat из event store путем восстановления состояния из событий (event sourcing)
func (r *MongoChatRepository) Load(ctx context.Context, chatID uuid.UUID) (*chatdomain.Chat, error) {
	if chatID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	// Загружаем события из event store
	events, err := r.eventStore.LoadEvents(ctx, chatID.String())
	if err != nil {
		if errors.Is(err, shared.ErrAggregateNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, fmt.Errorf("failed to load events for chat %s: %w", chatID, err)
	}

	if len(events) == 0 {
		return nil, errs.ErrNotFound
	}

	// Создаем новый Chat и применяем события
	chat := &chatdomain.Chat{}
	for _, domainEvent := range events {
		if chatErr := chat.Apply(domainEvent); chatErr != nil {
			return nil, fmt.Errorf("failed to apply event: %w", chatErr)
		}
	}

	// Помечаем события как committed (они уже сохранены)
	chat.MarkEventsAsCommitted()

	return chat, nil
}

// Save сохраняет Chat путем сохранения новых событий в event store и обновления read model
func (r *MongoChatRepository) Save(ctx context.Context, chat *chatdomain.Chat) error {
	if chat == nil {
		return errs.ErrInvalidInput
	}

	uncommittedEvents := chat.GetUncommittedEvents()
	if len(uncommittedEvents) == 0 {
		return nil // Нечего сохранять
	}

	// 1. Сохраняем события в event store
	expectedVersion := chat.Version() - len(uncommittedEvents)
	err := r.eventStore.SaveEvents(ctx, chat.ID().String(), uncommittedEvents, expectedVersion)
	if err != nil {
		if errors.Is(err, shared.ErrConcurrencyConflict) {
			return errs.ErrConcurrentModification
		}
		return fmt.Errorf("failed to save events: %w", err)
	}

	// 2. Обновляем read model (денормализованное представление)
	err = r.updateReadModel(ctx, chat)
	if err != nil {
		// Логируем ошибку, но не падаем (read model можно пересчитать)
		// TODO: добавить proper logging когда будет настроен logger
		_ = err // ignore read model update errors
	}

	// 3. Помечаем события как committed
	chat.MarkEventsAsCommitted()

	return nil
}

// GetEvents возвращает все события чата
func (r *MongoChatRepository) GetEvents(ctx context.Context, chatID uuid.UUID) ([]event.DomainEvent, error) {
	if chatID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	events, err := r.eventStore.LoadEvents(ctx, chatID.String())
	if err != nil {
		if errors.Is(err, shared.ErrAggregateNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}

	return events, nil
}

// updateReadModel обновляет денормализованное представление в read model коллекции
func (r *MongoChatRepository) updateReadModel(ctx context.Context, chat *chatdomain.Chat) error {
	// Проверяем, что у нас есть базовая информация для read model
	if chat.ID().IsZero() {
		return errs.ErrInvalidInput
	}

	// Преобразуем участников в строки
	participantStrs := make([]string, len(chat.Participants()))
	for i, p := range chat.Participants() {
		participantStrs[i] = p.UserID().String()
	}

	// Формируем документ read model
	doc := bson.M{
		"chat_id":      chat.ID().String(),
		"workspace_id": chat.WorkspaceID().String(),
		"type":         string(chat.Type()),
		"is_public":    chat.IsPublic(),
		"created_by":   chat.CreatedBy().String(),
		"created_at":   chat.CreatedAt(),
		"participants": participantStrs,
	}

	// Добавляем дополнительные поля для typed чатов
	if chat.Type() != chatdomain.TypeDiscussion {
		doc["title"] = chat.Title()
		doc["status"] = chat.Status()
		doc["priority"] = chat.Priority()

		if chat.AssigneeID() != nil {
			doc["assigned_to"] = chat.AssigneeID().String()
		}

		if chat.DueDate() != nil {
			doc["due_date"] = *chat.DueDate()
		}

		if chat.Type() == chatdomain.TypeBug {
			doc["severity"] = chat.Severity()
		}
	}

	// Используем upsert для создания или обновления документа
	filter := bson.M{"chat_id": chat.ID().String()}
	update := bson.M{"$set": doc}
	opts := options.UpdateOne().SetUpsert(true)

	_, err := r.readModelColl.UpdateOne(ctx, filter, update, opts)
	return HandleMongoError(err, "chat_read_model")
}

// MongoChatReadModelRepository реализует chatapp.QueryRepository (application layer interface)
// для query операций
type MongoChatReadModelRepository struct {
	collection *mongo.Collection
	eventStore shared.EventStore
}

// NewMongoChatReadModelRepository создает новый MongoDB Chat Read Model Repository
func NewMongoChatReadModelRepository(
	collection *mongo.Collection,
	eventStore shared.EventStore,
) *MongoChatReadModelRepository {
	return &MongoChatReadModelRepository{
		collection: collection,
		eventStore: eventStore,
	}
}

// FindByID находит чат по ID из read model
func (r *MongoChatReadModelRepository) FindByID(ctx context.Context, chatID uuid.UUID) (*chatapp.ReadModel, error) {
	if chatID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"chat_id": chatID.String()}
	var doc bson.M
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, HandleMongoError(err, "chat")
	}

	return r.documentToReadModel(doc)
}

// FindByWorkspace находит чаты workspace с фильтрами
func (r *MongoChatReadModelRepository) FindByWorkspace(
	ctx context.Context,
	workspaceID uuid.UUID,
	filters chatapp.Filters,
) ([]*chatapp.ReadModel, error) {
	if workspaceID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	// Формируем фильтр
	filter := bson.M{"workspace_id": workspaceID.String()}

	if filters.Type != nil {
		filter["type"] = string(*filters.Type)
	}

	if filters.IsPublic != nil {
		filter["is_public"] = *filters.IsPublic
	}

	if filters.UserID != nil {
		filter["participants"] = filters.UserID.String()
	}

	// Формируем опции (пагинация, сортировка)
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(filters.Limit)).
		SetSkip(int64(filters.Offset))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, HandleMongoError(err, "chats")
	}
	defer cursor.Close(ctx)

	var readModels []*chatapp.ReadModel
	for cursor.Next(ctx) {
		var doc bson.M
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			return nil, fmt.Errorf("failed to decode chat read model: %w", decodeErr)
		}

		rm, docErr := r.documentToReadModel(doc)
		if docErr != nil {
			continue // Пропускаем некорректные документы
		}

		readModels = append(readModels, rm)
	}

	if err = cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if readModels == nil {
		readModels = make([]*chatapp.ReadModel, 0)
	}

	return readModels, nil
}

// FindByParticipant находит чаты пользователя
func (r *MongoChatReadModelRepository) FindByParticipant(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*chatapp.ReadModel, error) {
	if userID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"participants": userID.String()}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, HandleMongoError(err, "chats")
	}
	defer cursor.Close(ctx)

	var readModels []*chatapp.ReadModel
	for cursor.Next(ctx) {
		var doc bson.M
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			continue
		}

		rm, docErr := r.documentToReadModel(doc)
		if docErr != nil {
			continue
		}

		readModels = append(readModels, rm)
	}

	if readModels == nil {
		readModels = make([]*chatapp.ReadModel, 0)
	}

	return readModels, nil
}

// Count возвращает количество чатов в workspace
func (r *MongoChatReadModelRepository) Count(ctx context.Context, workspaceID uuid.UUID) (int, error) {
	if workspaceID.IsZero() {
		return 0, errs.ErrInvalidInput
	}

	filter := bson.M{"workspace_id": workspaceID.String()}
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, HandleMongoError(err, "chats")
	}

	return int(count), nil
}

// documentToReadModel преобразует BSON документ в ReadModel
func (r *MongoChatReadModelRepository) documentToReadModel(doc bson.M) (*chatapp.ReadModel, error) {
	chatIDStr, ok := doc["chat_id"].(string)
	if !ok {
		return nil, errors.New("invalid chat_id type")
	}

	workspaceIDStr, ok := doc["workspace_id"].(string)
	if !ok {
		return nil, errors.New("invalid workspace_id type")
	}

	chatType, ok := doc["type"].(string)
	if !ok {
		return nil, errors.New("invalid type")
	}

	createdByStr, ok := doc["created_by"].(string)
	if !ok {
		return nil, errors.New("invalid created_by type")
	}

	isPublic, ok := doc["is_public"].(bool)
	if !ok {
		isPublic = false
	}

	var createdAt time.Time
	if createdAtVal, createdOk := doc["created_at"].(time.Time); createdOk {
		createdAt = createdAtVal
	}

	// Преобразуем участников
	var participants []chatdomain.Participant
	// Примечание: read model не хранит полную информацию о участниках (роль и т.д.)
	// Для полной информации используйте Load() на aggregate через event sourcing
	if participantsVal, participantOk := doc["participants"].(bson.A); participantOk {
		_ = participantsVal // Требует полного восстановления из событий
	}

	rm := &chatapp.ReadModel{
		ID:           uuid.UUID(chatIDStr),
		WorkspaceID:  uuid.UUID(workspaceIDStr),
		Type:         chatdomain.Type(chatType),
		IsPublic:     isPublic,
		CreatedBy:    uuid.UUID(createdByStr),
		CreatedAt:    createdAt,
		Participants: participants,
	}

	return rm, nil
}
