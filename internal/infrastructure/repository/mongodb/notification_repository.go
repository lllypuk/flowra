package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/lllypuk/flowra/internal/domain/errs"
	notificationdomain "github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// MongoNotificationRepository реализует notificationapp.Repository (application layer interface)
type MongoNotificationRepository struct {
	collection *mongo.Collection
}

// NewMongoNotificationRepository создает новый MongoDB Notification Repository
func NewMongoNotificationRepository(collection *mongo.Collection) *MongoNotificationRepository {
	return &MongoNotificationRepository{
		collection: collection,
	}
}

// FindByID находит уведомление по ID
func (r *MongoNotificationRepository) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (*notificationdomain.Notification, error) {
	if id.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"notification_id": id.String()}
	var doc notificationDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, HandleMongoError(err, "notification")
	}

	return r.documentToNotification(&doc)
}

// FindByUserID находит все уведомления пользователя с пагинацией
func (r *MongoNotificationRepository) FindByUserID(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*notificationdomain.Notification, error) {
	if userID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	limit = DefaultLimit(limit, DefaultPaginationLimit)

	filter := bson.M{"user_id": userID.String()}
	opts := FindWithPaginationDesc(offset, limit)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, HandleMongoError(err, "notifications")
	}
	defer cursor.Close(ctx)

	var notifications []*notificationdomain.Notification
	for cursor.Next(ctx) {
		var doc notificationDocument
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			continue
		}

		notif, docErr := r.documentToNotification(&doc)
		if docErr != nil {
			continue
		}

		notifications = append(notifications, notif)
	}

	if err = cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if notifications == nil {
		notifications = make([]*notificationdomain.Notification, 0)
	}

	return notifications, nil
}

// FindUnreadByUserID находит непрочитанные уведомления пользователя
func (r *MongoNotificationRepository) FindUnreadByUserID(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]*notificationdomain.Notification, error) {
	if userID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	limit = DefaultLimit(limit, DefaultPaginationLimit)

	filter := bson.M{
		"user_id": userID.String(),
		"read_at": nil,
	}
	opts := FindWithPaginationDesc(0, limit)

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, HandleMongoError(err, "notifications")
	}
	defer cursor.Close(ctx)

	var notifications []*notificationdomain.Notification
	for cursor.Next(ctx) {
		var doc notificationDocument
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			continue
		}

		notif, docErr := r.documentToNotification(&doc)
		if docErr != nil {
			continue
		}

		notifications = append(notifications, notif)
	}

	if notifications == nil {
		notifications = make([]*notificationdomain.Notification, 0)
	}

	return notifications, nil
}

// CountUnreadByUserID возвращает количество непрочитанных уведомлений
func (r *MongoNotificationRepository) CountUnreadByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	if userID.IsZero() {
		return 0, errs.ErrInvalidInput
	}

	filter := bson.M{
		"user_id": userID.String(),
		"read_at": nil,
	}
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, HandleMongoError(err, "notifications")
	}

	return int(count), nil
}

// Save сохраняет уведомление
func (r *MongoNotificationRepository) Save(ctx context.Context, notification *notificationdomain.Notification) error {
	if notification == nil {
		return errs.ErrInvalidInput
	}

	if notification.ID().IsZero() {
		return errs.ErrInvalidInput
	}

	doc := r.notificationToDocument(notification)
	filter := bson.M{"notification_id": notification.ID().String()}
	update := bson.M{"$set": doc}

	_, err := r.collection.UpdateOne(ctx, filter, update, UpsertOptions())
	return HandleMongoError(err, "notification")
}

// Delete удаляет уведомление
func (r *MongoNotificationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if id.IsZero() {
		return errs.ErrInvalidInput
	}

	filter := bson.M{"notification_id": id.String()}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return HandleMongoError(err, "notification")
	}

	if result.DeletedCount == 0 {
		return errs.ErrNotFound
	}

	return nil
}

// DeleteByUserID удаляет все уведомления пользователя
func (r *MongoNotificationRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	if userID.IsZero() {
		return errs.ErrInvalidInput
	}

	filter := bson.M{"user_id": userID.String()}
	_, err := r.collection.DeleteMany(ctx, filter)
	return HandleMongoError(err, "notifications")
}

// notificationDocument представляет структуру документа в MongoDB
type notificationDocument struct {
	NotificationID string     `bson:"notification_id"`
	UserID         string     `bson:"user_id"`
	Type           string     `bson:"type"`
	Title          string     `bson:"title"`
	Message        string     `bson:"message"`
	ResourceID     *string    `bson:"resource_id,omitempty"`
	ReadAt         *time.Time `bson:"read_at,omitempty"`
	CreatedAt      time.Time  `bson:"created_at"`
}

// notificationToDocument преобразует Notification в Document
func (r *MongoNotificationRepository) notificationToDocument(
	notif *notificationdomain.Notification,
) notificationDocument {
	return notificationDocument{
		NotificationID: notif.ID().String(),
		UserID:         notif.UserID().String(),
		Type:           string(notif.Type()),
		Title:          notif.Title(),
		Message:        notif.Message(),
		ResourceID:     StringPtr(notif.ResourceID()),
		ReadAt:         notif.ReadAt(),
		CreatedAt:      notif.CreatedAt(),
	}
}

// documentToNotification преобразует Document в Notification
func (r *MongoNotificationRepository) documentToNotification(
	doc *notificationDocument,
) (*notificationdomain.Notification, error) {
	if doc == nil {
		return nil, errs.ErrInvalidInput
	}

	id, err := uuid.ParseUUID(doc.NotificationID)
	if err != nil {
		return nil, errs.ErrInvalidInput
	}

	userID, err := uuid.ParseUUID(doc.UserID)
	if err != nil {
		return nil, errs.ErrInvalidInput
	}

	return notificationdomain.Reconstruct(
		id,
		userID,
		notificationdomain.Type(doc.Type),
		doc.Title,
		doc.Message,
		StringValue(doc.ResourceID),
		doc.ReadAt,
		doc.CreatedAt,
	), nil
}
