package mongodb

import (
	"context"
	"errors"
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
	opts := options.UpdateOne().SetUpsert(true)

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errs.ErrAlreadyExists
		}
		return HandleMongoError(err, "user")
	}

	return nil
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
	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, HandleMongoError(err, "users")
	}

	return int(count), nil
}

// userDocument представляет структуру документа в MongoDB
type userDocument struct {
	UserID        string    `bson:"user_id"`
	KeycloakID    *string   `bson:"keycloak_id,omitempty"`
	Username      string    `bson:"username"`
	Email         string    `bson:"email"`
	DisplayName   string    `bson:"display_name"`
	IsSystemAdmin bool      `bson:"is_system_admin"`
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
		CreatedAt:     user.CreatedAt(),
		UpdatedAt:     user.UpdatedAt(),
	}

	if externalID := user.ExternalID(); externalID != "" {
		doc.KeycloakID = &externalID
	}

	return doc
}

// documentToUser преобразует Document в User
func (r *MongoUserRepository) documentToUser(doc *userDocument) (*userdomain.User, error) {
	if doc == nil {
		return nil, errs.ErrInvalidInput
	}

	// TODO: Полная реализация требует наличия constructor или setter методов в domain/user
	// Сейчас возвращаем nil с сообщением о необходимости полной реализации
	return nil, errors.New("documentToUser requires domain setter methods - not yet implemented")
}
