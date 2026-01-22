package mongodb

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/errs"
	userdomain "github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// MongoUserRepository realizuet userapp.Repository (application layer interface)
type MongoUserRepository struct {
	collection *mongo.Collection
	logger     *slog.Logger
}

// UserRepoOption configures MongoUserRepository.
type UserRepoOption func(*MongoUserRepository)

// WithUserRepoLogger sets the logger for user repository.
func WithUserRepoLogger(logger *slog.Logger) UserRepoOption {
	return func(r *MongoUserRepository) {
		r.logger = logger
	}
}

// NewMongoUserRepository creates New MongoDB User Repository
func NewMongoUserRepository(collection *mongo.Collection, opts ...UserRepoOption) *MongoUserRepository {
	r := &MongoUserRepository{
		collection: collection,
		logger:     slog.Default(),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// FindByID finds user po ID
func (r *MongoUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*userdomain.User, error) {
	if id.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"user_id": id.String()}
	var doc userDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			r.logger.ErrorContext(ctx, "failed to find user by ID",
				slog.String("user_id", id.String()),
				slog.String("error", err.Error()),
			)
		}
		return nil, HandleMongoError(err, "user")
	}

	return r.documentToUser(&doc)
}

// FindByExternalID finds user po ID from vneshney sistemy autentifikatsii
func (r *MongoUserRepository) FindByExternalID(ctx context.Context, externalID string) (*userdomain.User, error) {
	if externalID == "" {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"keycloak_id": externalID}
	var doc userDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			r.logger.ErrorContext(ctx, "failed to find user by external ID",
				slog.String("external_id", externalID),
				slog.String("error", err.Error()),
			)
		}
		return nil, HandleMongoError(err, "user")
	}

	return r.documentToUser(&doc)
}

// FindByEmail finds user po email
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

// FindByUsername finds user po username
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

// GetByUsername implements appcore.UserRepository.
// It finds a user by username and returns minimal user info.
func (r *MongoUserRepository) GetByUsername(ctx context.Context, username string) (*appcore.User, error) {
	user, err := r.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return &appcore.User{
		ID:       user.ID(),
		Username: user.Username(),
		FullName: user.DisplayName(),
	}, nil
}

// GetByID implements appcore.UserRepository.
// It finds a user by ID and returns minimal user info.
func (r *MongoUserRepository) GetByID(ctx context.Context, userID uuid.UUID) (*appcore.User, error) {
	user, err := r.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &appcore.User{
		ID:       user.ID(),
		Username: user.Username(),
		FullName: user.DisplayName(),
	}, nil
}

// Exists checks, suschestvuet li user s zadannym ID
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

// ExistsByUsername checks, suschestvuet li user s zadannym username
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

// ExistsByEmail checks, suschestvuet li user s zadannym email
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

// Save saves user
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
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to save user",
			slog.String("user_id", user.ID().String()),
			slog.String("email", user.Email()),
			slog.String("error", err.Error()),
		)
	}
	return HandleMongoError(err, "user")
}

// Delete udalyaet user
func (r *MongoUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if id.IsZero() {
		return errs.ErrInvalidInput
	}

	filter := bson.M{"user_id": id.String()}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to delete user",
			slog.String("user_id", id.String()),
			slog.String("error", err.Error()),
		)
		return HandleMongoError(err, "user")
	}

	if result.DeletedCount == 0 {
		return errs.ErrNotFound
	}

	return nil
}

// List returns list users s paginatsiey
func (r *MongoUserRepository) List(ctx context.Context, offset, limit int) ([]*userdomain.User, error) {
	return listDocuments(ctx, r.collection, offset, limit, r.documentToUser, "users")
}

// Count returns obschee count users
func (r *MongoUserRepository) Count(ctx context.Context) (int, error) {
	count, err := CountAll(ctx, r.collection)
	if err != nil {
		return 0, HandleMongoError(err, "users")
	}
	return count, nil
}

// userDocument represents strukturu dokumenta in MongoDB
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

// userToDocument preobrazuet User in Document
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

// documentToUser preobrazuet Document in User
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

// ListExternalIDs returns list all external ID (Keycloak ID) users
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
