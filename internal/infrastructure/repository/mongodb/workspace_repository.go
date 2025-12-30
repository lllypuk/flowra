package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	workspacedomain "github.com/lllypuk/flowra/internal/domain/workspace"
)

// MongoWorkspaceRepository реализует workspaceapp.Repository (application layer interface)
type MongoWorkspaceRepository struct {
	collection        *mongo.Collection
	membersCollection *mongo.Collection
}

// NewMongoWorkspaceRepository создает новый MongoDB Workspace Repository
func NewMongoWorkspaceRepository(
	collection *mongo.Collection,
	membersCollection *mongo.Collection,
) *MongoWorkspaceRepository {
	return &MongoWorkspaceRepository{
		collection:        collection,
		membersCollection: membersCollection,
	}
}

// FindByID находит рабочее пространство по ID
func (r *MongoWorkspaceRepository) FindByID(ctx context.Context, id uuid.UUID) (*workspacedomain.Workspace, error) {
	if id.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"workspace_id": id.String()}
	var doc workspaceDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, HandleMongoError(err, "workspace")
	}

	return r.documentToWorkspace(&doc)
}

// FindByKeycloakGroup находит рабочее пространство по ID группы Keycloak
func (r *MongoWorkspaceRepository) FindByKeycloakGroup(
	ctx context.Context,
	keycloakGroupID string,
) (*workspacedomain.Workspace, error) {
	if keycloakGroupID == "" {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"keycloak_group_id": keycloakGroupID}
	var doc workspaceDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, HandleMongoError(err, "workspace")
	}

	return r.documentToWorkspace(&doc)
}

// Save сохраняет рабочее пространство
func (r *MongoWorkspaceRepository) Save(ctx context.Context, workspace *workspacedomain.Workspace) error {
	if workspace == nil {
		return errs.ErrInvalidInput
	}

	if workspace.ID().IsZero() {
		return errs.ErrInvalidInput
	}

	doc := r.workspaceToDocument(workspace)
	filter := bson.M{"workspace_id": workspace.ID().String()}
	update := bson.M{"$set": doc}
	opts := options.UpdateOne().SetUpsert(true)

	_, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errs.ErrAlreadyExists
		}
		return HandleMongoError(err, "workspace")
	}

	return nil
}

// Delete удаляет рабочее пространство
func (r *MongoWorkspaceRepository) Delete(_ context.Context, id uuid.UUID) error {
	if id.IsZero() {
		return errs.ErrInvalidInput
	}

	session, err := r.collection.Database().Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(context.Background())

	_, err = session.WithTransaction(context.Background(), func(ctx context.Context) (any, error) {
		// Удаляем само рабочее пространство
		filter := bson.M{"workspace_id": id.String()}
		result, deleteErr := r.collection.DeleteOne(ctx, filter)
		if deleteErr != nil {
			return nil, HandleMongoError(deleteErr, "workspace")
		}

		if result.DeletedCount == 0 {
			return nil, errs.ErrNotFound
		}

		// Удаляем членов рабочего пространства
		memberFilter := bson.M{"workspace_id": id.String()}
		_, deleteErr = r.membersCollection.DeleteMany(ctx, memberFilter)
		if deleteErr != nil {
			return nil, fmt.Errorf("failed to delete workspace members: %w", deleteErr)
		}

		return struct{}{}, nil
	})

	return err
}

// List возвращает список рабочих пространств с пагинацией
func (r *MongoWorkspaceRepository) List(ctx context.Context, offset, limit int) ([]*workspacedomain.Workspace, error) {
	return listDocuments(ctx, r.collection, offset, limit, r.documentToWorkspace, "workspaces")
}

// Count возвращает общее количество рабочих пространств
func (r *MongoWorkspaceRepository) Count(ctx context.Context) (int, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, HandleMongoError(err, "workspaces")
	}

	return int(count), nil
}

// FindInviteByToken находит приглашение по токену
func (r *MongoWorkspaceRepository) FindInviteByToken(
	ctx context.Context,
	token string,
) (*workspacedomain.Invite, error) {
	if token == "" {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"invites.token": token}
	opts := options.FindOne()

	var doc workspaceDocument
	err := r.collection.FindOne(ctx, filter, opts).Decode(&doc)
	if err != nil {
		return nil, HandleMongoError(err, "invite")
	}

	// Находим конкретное приглашение в массиве
	for _, inv := range doc.Invites {
		if inv.Token == token {
			// Преобразуем в domain модель
			return &workspacedomain.Invite{
				// Нужны setter методы или конструктор
			}, nil
		}
	}

	return nil, errs.ErrNotFound
}

// workspaceDocument представляет структуру документа в MongoDB
type workspaceDocument struct {
	WorkspaceID     string           `bson:"workspace_id"`
	Name            string           `bson:"name"`
	KeycloakGroupID *string          `bson:"keycloak_group_id,omitempty"`
	CreatedBy       string           `bson:"created_by"`
	CreatedAt       time.Time        `bson:"created_at"`
	UpdatedAt       time.Time        `bson:"updated_at"`
	Invites         []inviteDocument `bson:"invites"`
}

// inviteDocument представляет приглашение в документе
type inviteDocument struct {
	Token     string    `bson:"token"`
	Email     string    `bson:"email"`
	ExpiresAt time.Time `bson:"expires_at"`
	CreatedAt time.Time `bson:"created_at"`
}

// workspaceToDocument преобразует Workspace в Document
func (r *MongoWorkspaceRepository) workspaceToDocument(ws *workspacedomain.Workspace) workspaceDocument {
	doc := workspaceDocument{
		WorkspaceID: ws.ID().String(),
		Name:        ws.Name(),
		CreatedBy:   ws.CreatedBy().String(),
		CreatedAt:   ws.CreatedAt(),
		UpdatedAt:   ws.UpdatedAt(),
		Invites:     make([]inviteDocument, 0),
	}

	// Добавляем приглашения (если они есть)
	// Это требует знания методов для получения приглашений из Workspace

	return doc
}

// documentToWorkspace преобразует Document в Workspace
func (r *MongoWorkspaceRepository) documentToWorkspace(doc *workspaceDocument) (*workspacedomain.Workspace, error) {
	if doc == nil {
		return nil, errs.ErrInvalidInput
	}

	// TODO: Полная реализация требует наличия constructor или setter методов в domain/workspace
	// Сейчас возвращаем nil с сообщением о необходимости полной реализации
	return nil, errors.New("documentToWorkspace requires domain setter methods - not yet implemented")
}
