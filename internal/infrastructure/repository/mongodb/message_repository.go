package mongodb

import (
	"context"
	"fmt"
	"regexp"
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

	pagination.Limit = DefaultLimit(pagination.Limit, DefaultPaginationLimit)

	filter := bson.M{"chat_id": chatID.String()}
	opts := FindWithPaginationDesc(pagination.Offset, pagination.Limit)

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
	_, err := r.collection.UpdateOne(ctx, filter, update, UpsertOptions())
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

// CountThreadReplies возвращает количество ответов в треде
func (r *MongoMessageRepository) CountThreadReplies(
	ctx context.Context,
	parentMessageID uuid.UUID,
) (int, error) {
	if parentMessageID.IsZero() {
		return 0, errs.ErrInvalidInput
	}

	filter := bson.M{
		"parent_id":  parentMessageID.String(),
		"is_deleted": false,
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, HandleMongoError(err, "messages")
	}

	return int(count), nil
}

// AddReaction добавляет реакцию к сообщению
func (r *MongoMessageRepository) AddReaction(
	ctx context.Context,
	messageID uuid.UUID,
	emojiCode string,
	userID uuid.UUID,
) error {
	if messageID.IsZero() || emojiCode == "" || userID.IsZero() {
		return errs.ErrInvalidInput
	}

	reaction := reactionDocument{
		UserID:    userID.String(),
		EmojiCode: emojiCode,
		AddedAt:   time.Now().UTC(),
	}

	filter := bson.M{"message_id": messageID.String()}
	update := bson.M{
		"$push": bson.M{"reactions": reaction},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return HandleMongoError(err, "message")
	}

	if result.MatchedCount == 0 {
		return errs.ErrNotFound
	}

	return nil
}

// RemoveReaction удаляет реакцию с сообщения
func (r *MongoMessageRepository) RemoveReaction(
	ctx context.Context,
	messageID uuid.UUID,
	emojiCode string,
	userID uuid.UUID,
) error {
	if messageID.IsZero() || emojiCode == "" || userID.IsZero() {
		return errs.ErrInvalidInput
	}

	filter := bson.M{"message_id": messageID.String()}
	update := bson.M{
		"$pull": bson.M{
			"reactions": bson.M{
				"emoji_code": emojiCode,
				"user_id":    userID.String(),
			},
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return HandleMongoError(err, "message")
	}

	if result.MatchedCount == 0 {
		return errs.ErrNotFound
	}

	return nil
}

// GetReactionUsers возвращает пользователей, поставивших определенную реакцию
func (r *MongoMessageRepository) GetReactionUsers(
	ctx context.Context,
	messageID uuid.UUID,
	emojiCode string,
) ([]uuid.UUID, error) {
	if messageID.IsZero() || emojiCode == "" {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"message_id": messageID.String()}
	var doc messageDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, HandleMongoError(err, "message")
	}

	var userIDs []uuid.UUID
	for _, reaction := range doc.Reactions {
		if reaction.EmojiCode == emojiCode {
			userID, parseErr := uuid.ParseUUID(reaction.UserID)
			if parseErr == nil {
				userIDs = append(userIDs, userID)
			}
		}
	}

	if userIDs == nil {
		userIDs = make([]uuid.UUID, 0)
	}

	return userIDs, nil
}

// SearchInChat ищет сообщения в чате по тексту
func (r *MongoMessageRepository) SearchInChat(
	ctx context.Context,
	chatID uuid.UUID,
	query string,
	offset, limit int,
) ([]*messagedomain.Message, error) {
	if chatID.IsZero() || query == "" {
		return nil, errs.ErrInvalidInput
	}

	limit = DefaultLimitWithMax(limit, DefaultPaginationLimit, MaxPaginationLimit)

	// Escape regex special characters for safe search
	escapedQuery := regexp.QuoteMeta(query)

	filter := bson.M{
		"chat_id":    chatID.String(),
		"is_deleted": false,
		"content": bson.M{
			"$regex":   escapedQuery,
			"$options": "i", // case-insensitive
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, HandleMongoError(err, "messages")
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

	if err = cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if messages == nil {
		messages = make([]*messagedomain.Message, 0)
	}

	return messages, nil
}

// FindByAuthor находит сообщения автора в чате
func (r *MongoMessageRepository) FindByAuthor(
	ctx context.Context,
	chatID uuid.UUID,
	authorID uuid.UUID,
	offset, limit int,
) ([]*messagedomain.Message, error) {
	if chatID.IsZero() || authorID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	limit = DefaultLimitWithMax(limit, DefaultPaginationLimit, MaxPaginationLimit)

	filter := bson.M{
		"chat_id":    chatID.String(),
		"sent_by":    authorID.String(),
		"is_deleted": false,
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, HandleMongoError(err, "messages")
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

	if err = cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if messages == nil {
		messages = make([]*messagedomain.Message, 0)
	}

	return messages, nil
}

// messageDocument представляет структуру документа в MongoDB
type messageDocument struct {
	MessageID   string               `bson:"message_id"`
	ChatID      string               `bson:"chat_id"`
	AuthorID    string               `bson:"sent_by"`
	Content     string               `bson:"content"`
	ParentID    *string              `bson:"parent_id,omitempty"`
	CreatedAt   time.Time            `bson:"created_at"`
	EditedAt    *time.Time           `bson:"edited_at,omitempty"`
	IsDeleted   bool                 `bson:"is_deleted"`
	DeletedAt   *time.Time           `bson:"deleted_at,omitempty"`
	Attachments []attachmentDocument `bson:"attachments"`
	Reactions   []reactionDocument   `bson:"reactions"`
}

// attachmentDocument представляет вложение в документе
type attachmentDocument struct {
	FileID   string `bson:"file_id"`
	FileName string `bson:"file_name"`
	FileSize int64  `bson:"file_size"`
	MimeType string `bson:"mime_type"`
}

// reactionDocument представляет реакцию в документе
type reactionDocument struct {
	UserID    string    `bson:"user_id"`
	EmojiCode string    `bson:"emoji_code"`
	AddedAt   time.Time `bson:"added_at"`
}

// messageToDocument преобразует Message в Document
func (r *MongoMessageRepository) messageToDocument(msg *messagedomain.Message) messageDocument {
	// Преобразуем вложения
	attachments := make([]attachmentDocument, 0, len(msg.Attachments()))
	for _, a := range msg.Attachments() {
		attachments = append(attachments, attachmentDocument{
			FileID:   a.FileID().String(),
			FileName: a.FileName(),
			FileSize: a.FileSize(),
			MimeType: a.MimeType(),
		})
	}

	// Преобразуем реакции
	reactions := make([]reactionDocument, 0, len(msg.Reactions()))
	for _, r := range msg.Reactions() {
		reactions = append(reactions, reactionDocument{
			UserID:    r.UserID().String(),
			EmojiCode: r.EmojiCode(),
			AddedAt:   r.AddedAt(),
		})
	}

	// Обрабатываем parent ID
	var parentID *string
	if !msg.ParentMessageID().IsZero() {
		parentIDStr := msg.ParentMessageID().String()
		parentID = &parentIDStr
	}

	return messageDocument{
		MessageID:   msg.ID().String(),
		ChatID:      msg.ChatID().String(),
		AuthorID:    msg.AuthorID().String(),
		Content:     msg.Content(),
		ParentID:    parentID,
		CreatedAt:   msg.CreatedAt(),
		EditedAt:    msg.EditedAt(),
		IsDeleted:   msg.IsDeleted(),
		DeletedAt:   msg.DeletedAt(),
		Attachments: attachments,
		Reactions:   reactions,
	}
}

// documentToMessage преобразует Document в Message
func (r *MongoMessageRepository) documentToMessage(doc *messageDocument) (*messagedomain.Message, error) {
	if doc == nil {
		return nil, errs.ErrInvalidInput
	}

	id, err := uuid.ParseUUID(doc.MessageID)
	if err != nil {
		return nil, errs.ErrInvalidInput
	}

	chatID, err := uuid.ParseUUID(doc.ChatID)
	if err != nil {
		return nil, errs.ErrInvalidInput
	}

	authorID, err := uuid.ParseUUID(doc.AuthorID)
	if err != nil {
		return nil, errs.ErrInvalidInput
	}

	var parentMessageID uuid.UUID
	if doc.ParentID != nil {
		parentMessageID, err = uuid.ParseUUID(*doc.ParentID)
		if err != nil {
			return nil, errs.ErrInvalidInput
		}
	}

	// Восстанавливаем вложения
	attachments := make([]messagedomain.Attachment, 0, len(doc.Attachments))
	for _, a := range doc.Attachments {
		fileID, parseErr := uuid.ParseUUID(a.FileID)
		if parseErr != nil {
			continue // пропускаем некорректные вложения
		}
		attachments = append(attachments, messagedomain.ReconstructAttachment(
			fileID,
			a.FileName,
			a.FileSize,
			a.MimeType,
		))
	}

	// Восстанавливаем реакции
	reactions := make([]messagedomain.Reaction, 0, len(doc.Reactions))
	for _, r := range doc.Reactions {
		userID, parseErr := uuid.ParseUUID(r.UserID)
		if parseErr != nil {
			continue // пропускаем некорректные реакции
		}
		reactions = append(reactions, messagedomain.ReconstructReaction(
			userID,
			r.EmojiCode,
			r.AddedAt,
		))
	}

	return messagedomain.Reconstruct(
		id,
		chatID,
		authorID,
		doc.Content,
		parentMessageID,
		doc.CreatedAt,
		doc.EditedAt,
		doc.IsDeleted,
		doc.DeletedAt,
		attachments,
		reactions,
	), nil
}
