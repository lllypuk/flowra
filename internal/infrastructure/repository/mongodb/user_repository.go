package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lllypuk/flowra/internal/domain/errs"
	userdomain "github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// MongoUserRepository реализует userapp.Repository (application layer interface)
type MongoUserRepository struct {
	collection *mongo.Collection
}

// NewMongoUserRepository создает новый MongoDB User Repository
func NewMongoUserRepository(collection *mongo.Collection) *MongoUserRepository {
	return &MongoUserRepository{
		collection: collection,
	}
}

// FindByID находит пользователя по ID
func (r *MongoUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*userdomain.User, error) {
	if id.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"user_id": id.String()}
	var doc userDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, HandleMongoError(err, "user")
	}

	return r.documentToUser(&doc)
}

// FindByExternalID находит пользователя по ID из внешней системы аутентификации
func (r *MongoUserRepository) FindByExternalID(ctx context.Context, externalID string) (*userdomain.User, error) {
	if externalID == "" {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"keycloak_id": externalID}
	var doc userDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, HandleMongoError(err, "user")
	}

	return r.documentToUser(&doc)
}

// FindByEmail находит пользователя по email
func (r *MongoUserRepository) FindByEmail(ctx context.Context, email string) (*userdomain.User, error) {
	if email == "" {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"email": email}
	var doc userDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, HandleMongoError(err, "user")
	}

	return r.documentToUser(&doc)
}

// FindByUsername находит пользователя по username
func (r *MongoUserRepository) FindByUsername(ctx context.Context, username string) (*userdomain.User, error) {
	if username == "" {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"username": username}
	var doc userDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, HandleMongoError(err, "user")
	}

	return r.documentToUser(&doc)
}

// Exists проверяет, существует ли пользователь с заданным ID
func (r *MongoUserRepository) Exists(ctx context.Context, userID uuid.UUID) (bool, error) {
	if userID.IsZero() {
		return false, errs.ErrInvalidInput
	}

	filter := bson.M{"user_id": userID.String()}
	count, err := r.collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, HandleMongoError(err, "user")
	}

	return count > 0, nil
}

// ExistsByUsername проверяет, существует ли пользователь с заданным username
func (r *MongoUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	if username == "" {
		return false, errs.ErrInvalidInput
	}

	filter := bson.M{"username": username}
	count, err := r.collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, HandleMongoError(err, "user")
	}

	return count > 0, nil
}

// ExistsByEmail проверяет, существует ли пользователь с заданным email
func (r *MongoUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if email == "" {
		return false, errs.ErrInvalidInput
	}

	filter := bson.M{"email": email}
	count, err := r.collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, HandleMongoError(err, "user")
	}

	return count > 0, nil
}

// Save сохраняет пользователя
func (r *MongoUserRepository) Save(ctx context.Context, user *userdomain.User) error {
	if user == nil {
		return errs.ErrInvalidInput
	}

	if user.ID().IsZero() {
		return errs.ErrInvalidInput
	}

	doc := r.userToDocument(user)
	filter := bson.M{"user_id": user.ID().String()}
	update := bson.M{"$set": doc}

	_, err := r.collection.UpdateOne(ctx, filter, update, UpsertOptions())
	return HandleMongoError(err, "user")
}

// Delete удаляет пользователя
func (r *MongoUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if id.IsZero() {
		return errs.ErrInvalidInput
	}

	filter := bson.M{"user_id": id.String()}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return HandleMongoError(err, "user")
	}

	if result.DeletedCount == 0 {
		return errs.ErrNotFound
	}

	return nil
}

// List возвращает список пользователей с пагинацией
func (r *MongoUserRepository) List(ctx context.Context, offset, limit int) ([]*userdomain.User, error) {
	return listDocuments(ctx, r.collection, offset, limit, r.documentToUser, "users")
}

// Count возвращает общее количество пользователей
func (r *MongoUserRepository) Count(ctx context.Context) (int, error) {
	count, err := CountAll(ctx, r.collection)
	if err != nil {
		return 0, HandleMongoError(err, "users")
	}
	return count, nil
}

// userDocument представляет структуру документа в MongoDB
type userDocument struct {
	UserID        string    `bson:"user_id"`
	KeycloakID    *string   `bson:"keycloak_id,omitempty"`
	Username      string    `bson:"username"`
	Email         string    `bson:"email"`
	DisplayName   string    `bson:"display_name"`
	IsSystemAdmin bool      `bson:"is_system_admin"`
	IsActive      bool      `bson:"is_active"`
	CreatedAt     time.Time `bson:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at"`
}

// userToDocument преобразует User в Document
func (r *MongoUserRepository) userToDocument(user *userdomain.User) userDocument {
	doc := userDocument{
		UserID:        user.ID().String(),
		Username:      user.Username(),
		Email:         user.Email(),
		DisplayName:   user.DisplayName(),
		IsSystemAdmin: user.IsSystemAdmin(),
		IsActive:      user.IsActive(),
		CreatedAt:     user.CreatedAt(),
		UpdatedAt:     user.UpdatedAt(),
	}

	doc.KeycloakID = StringPtr(user.ExternalID())

	return doc
}

// documentToUser преобразует Document в User
func (r *MongoUserRepository) documentToUser(doc *userDocument) (*userdomain.User, error) {
	if doc == nil {
		return nil, errs.ErrInvalidInput
	}

	id, err := uuid.ParseUUID(doc.UserID)
	if err != nil {
		return nil, errs.ErrInvalidInput
	}

	externalID := StringValue(doc.KeycloakID)

	return userdomain.Reconstruct(
		id,
		externalID,
		doc.Username,
		doc.Email,
		doc.DisplayName,
		doc.IsSystemAdmin,
		doc.IsActive,
		doc.CreatedAt,
		doc.UpdatedAt,
	), nil
}

// ListExternalIDs возвращает список всех external ID (Keycloak ID) пользователей
func (r *MongoUserRepository) ListExternalIDs(ctx context.Context) ([]string, error) {
	filter := bson.M{
		"keycloak_id": bson.M{"$exists": true, "$ne": nil},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetProjection(bson.M{"keycloak_id": 1}))
	if err != nil {
		return nil, HandleMongoError(err, "users")
	}
	defer cursor.Close(ctx)

	var externalIDs []string
	for cursor.Next(ctx) {
		var doc struct {
			KeycloakID *string `bson:"keycloak_id"`
		}
		decodeErr := cursor.Decode(&doc)
		if decodeErr != nil {
			continue
		}
		if doc.KeycloakID != nil && *doc.KeycloakID != "" {
			externalIDs = append(externalIDs, *doc.KeycloakID)
		}
	}

	cursorErr := cursor.Err()
	if cursorErr != nil {
		return nil, HandleMongoError(cursorErr, "users")
	}

	return externalIDs, nil
}
