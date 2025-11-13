package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	messageapp "github.com/lllypuk/flowra/internal/application/message"
	"github.com/lllypuk/flowra/internal/domain/errs"
	messagedomain "github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// MongoMessageRepository реализует messageapp.Repository (application layer interface)
type MongoMessageRepository struct {
	collection *mongo.Collection
}

// NewMongoMessageRepository создает новый MongoDB Message Repository
func NewMongoMessageRepository(collection *mongo.Collection) *MongoMessageRepository {
	return &MongoMessageRepository{
		collection: collection,
	}
}

// FindByID находит сообщение по ID
func (r *MongoMessageRepository) FindByID(ctx context.Context, id uuid.UUID) (*messagedomain.Message, error) {
	if id.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"message_id": id.String()}
	var doc messageDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, HandleMongoError(err, "message")
	}

	return r.documentToMessage(&doc)
}

// FindByChatID находит сообщения в чате с пагинацией (от новых к старым)
func (r *MongoMessageRepository) FindByChatID(
	ctx context.Context,
	chatID uuid.UUID,
	pagination messageapp.Pagination,
) ([]*messagedomain.Message, error) {
	if chatID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	if pagination.Limit == 0 {
		pagination.Limit = 50 // default
	}

	filter := bson.M{"chat_id": chatID.String()}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(pagination.Limit)).
		SetSkip(int64(pagination.Offset))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, HandleMongoError(err, "messages")
	}
	defer cursor.Close(ctx)

	var messages []*messagedomain.Message
	for cursor.Next(ctx) {
		var doc messageDocument
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			continue // Пропускаем некорректные документы
		}

		msg, docErr := r.documentToMessage(&doc)
		if docErr != nil {
			continue
		}

		messages = append(messages, msg)
	}

	if err = cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if messages == nil {
		messages = make([]*messagedomain.Message, 0)
	}

	return messages, nil
}

// FindThread находит все ответы в треде
func (r *MongoMessageRepository) FindThread(
	ctx context.Context,
	parentMessageID uuid.UUID,
) ([]*messagedomain.Message, error) {
	if parentMessageID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"parent_id": parentMessageID.String()}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, HandleMongoError(err, "message_thread")
	}
	defer cursor.Close(ctx)

	var messages []*messagedomain.Message
	for cursor.Next(ctx) {
		var doc messageDocument
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			continue
		}

		msg, docErr := r.documentToMessage(&doc)
		if docErr != nil {
			continue
		}

		messages = append(messages, msg)
	}

	if messages == nil {
		messages = make([]*messagedomain.Message, 0)
	}

	return messages, nil
}

// CountByChatID возвращает количество сообщений в чате
func (r *MongoMessageRepository) CountByChatID(ctx context.Context, chatID uuid.UUID) (int, error) {
	if chatID.IsZero() {
		return 0, errs.ErrInvalidInput
	}

	filter := bson.M{
		"chat_id":    chatID.String(),
		"is_deleted": false,
	}
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, HandleMongoError(err, "messages")
	}

	return int(count), nil
}

// Save сохраняет сообщение (создание или обновление)
func (r *MongoMessageRepository) Save(ctx context.Context, message *messagedomain.Message) error {
	if message == nil {
		return errs.ErrInvalidInput
	}

	if message.ID().IsZero() {
		return errs.ErrInvalidInput
	}

	doc := r.messageToDocument(message)

	filter := bson.M{"message_id": message.ID().String()}
	update := bson.M{"$set": doc}
	opts := options.UpdateOne().SetUpsert(true)

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	return HandleMongoError(err, "message")
}

// Delete физически удаляет сообщение
func (r *MongoMessageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if id.IsZero() {
		return errs.ErrInvalidInput
	}

	filter := bson.M{"message_id": id.String()}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return HandleMongoError(err, "message")
	}

	if result.DeletedCount == 0 {
		return errs.ErrNotFound
	}

	return nil
}

// messageDocument представляет структуру документа в MongoDB
type messageDocument struct {
	MessageID   string     `bson:"message_id"`
	ChatID      string     `bson:"chat_id"`
	AuthorID    string     `bson:"sent_by"`
	Content     string     `bson:"content"`
	ParentID    *string    `bson:"parent_id,omitempty"`
	CreatedAt   time.Time  `bson:"created_at"`
	EditedAt    *time.Time `bson:"edited_at,omitempty"`
	IsDeleted   bool       `bson:"is_deleted"`
	DeletedAt   *time.Time `bson:"deleted_at,omitempty"`
	Attachments []string   `bson:"attachments"`
	Reactions   bson.M     `bson:"reactions,omitempty"`
}

// messageToDocument преобразует Message в Document
func (r *MongoMessageRepository) messageToDocument(msg *messagedomain.Message) messageDocument {
	doc := messageDocument{
		MessageID:   msg.ID().String(),
		ChatID:      msg.ChatID().String(),
		AuthorID:    msg.AuthorID().String(),
		Content:     msg.Content(),
		CreatedAt:   msg.CreatedAt(),
		IsDeleted:   msg.IsDeleted(),
		Attachments: make([]string, 0),
		Reactions:   bson.M{},
	}

	parentID := msg.ParentMessageID()
	if !parentID.IsZero() {
		parentIDStr := parentID.String()
		doc.ParentID = &parentIDStr
	}

	if msg.EditedAt() != nil {
		doc.EditedAt = msg.EditedAt()
	}

	if msg.DeletedAt() != nil {
		doc.DeletedAt = msg.DeletedAt()
	}

	// Преобразуем вложения (если есть)
	// Это требует знания структуры Attachment в domain/message

	return doc
}

// documentToMessage преобразует Document в Message
func (r *MongoMessageRepository) documentToMessage(doc *messageDocument) (*messagedomain.Message, error) {
	if doc == nil {
		return nil, errs.ErrInvalidInput
	}

	// TODO: Полная реализация требует наличия setter методов или factory method в domain/message
	// Сейчас возвращаем nil с сообщением о необходимости полной реализации
	// При полной реализации нужно:
	// 1. Распарсить doc.ChatID, doc.AuthorID, doc.ParentID в UUID
	// 2. Создать сообщение через NewMessage()
	// 3. Установить дополнительные поля (editedAt, deletedAt, isDeleted) через setter методы
	// 4. Вернуть полное сообщение

	return nil, errors.New("documentToMessage requires domain setter methods - not yet implemented")
}
