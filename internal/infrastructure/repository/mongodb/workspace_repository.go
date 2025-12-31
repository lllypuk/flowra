package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

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

	_, err := r.collection.UpdateOne(ctx, filter, update, UpsertOptions())
	return HandleMongoError(err, "workspace")
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
	count, err := CountAll(ctx, r.collection)
	if err != nil {
		return 0, HandleMongoError(err, "workspaces")
	}
	return count, nil
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

	var doc workspaceDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, HandleMongoError(err, "invite")
	}

	// Находим конкретное приглашение в массиве
	for _, inv := range doc.Invites {
		if inv.Token == token {
			return r.documentToInvite(&inv)
		}
	}

	return nil, errs.ErrNotFound
}

// workspaceDocument представляет структуру документа в MongoDB
type workspaceDocument struct {
	WorkspaceID     string           `bson:"workspace_id"`
	Name            string           `bson:"name"`
	KeycloakGroupID string           `bson:"keycloak_group_id"`
	CreatedBy       string           `bson:"created_by"`
	CreatedAt       time.Time        `bson:"created_at"`
	UpdatedAt       time.Time        `bson:"updated_at"`
	Invites         []inviteDocument `bson:"invites"`
}

// inviteDocument представляет приглашение в документе
type inviteDocument struct {
	InviteID    string    `bson:"invite_id"`
	WorkspaceID string    `bson:"workspace_id"`
	Token       string    `bson:"token"`
	CreatedBy   string    `bson:"created_by"`
	CreatedAt   time.Time `bson:"created_at"`
	ExpiresAt   time.Time `bson:"expires_at"`
	MaxUses     int       `bson:"max_uses"`
	UsedCount   int       `bson:"used_count"`
	IsRevoked   bool      `bson:"is_revoked"`
}

// workspaceToDocument преобразует Workspace в Document
func (r *MongoWorkspaceRepository) workspaceToDocument(ws *workspacedomain.Workspace) workspaceDocument {
	// Преобразуем приглашения
	invites := make([]inviteDocument, 0, len(ws.Invites()))
	for _, inv := range ws.Invites() {
		invites = append(invites, inviteDocument{
			InviteID:    inv.ID().String(),
			WorkspaceID: inv.WorkspaceID().String(),
			Token:       inv.Token(),
			CreatedBy:   inv.CreatedBy().String(),
			CreatedAt:   inv.CreatedAt(),
			ExpiresAt:   inv.ExpiresAt(),
			MaxUses:     inv.MaxUses(),
			UsedCount:   inv.UsedCount(),
			IsRevoked:   inv.IsRevoked(),
		})
	}

	return workspaceDocument{
		WorkspaceID:     ws.ID().String(),
		Name:            ws.Name(),
		KeycloakGroupID: ws.KeycloakGroupID(),
		CreatedBy:       ws.CreatedBy().String(),
		CreatedAt:       ws.CreatedAt(),
		UpdatedAt:       ws.UpdatedAt(),
		Invites:         invites,
	}
}

// documentToWorkspace преобразует Document в Workspace
func (r *MongoWorkspaceRepository) documentToWorkspace(doc *workspaceDocument) (*workspacedomain.Workspace, error) {
	if doc == nil {
		return nil, errs.ErrInvalidInput
	}

	id, err := uuid.ParseUUID(doc.WorkspaceID)
	if err != nil {
		return nil, errs.ErrInvalidInput
	}

	createdBy, err := uuid.ParseUUID(doc.CreatedBy)
	if err != nil {
		return nil, errs.ErrInvalidInput
	}

	// Восстанавливаем приглашения
	invites := make([]*workspacedomain.Invite, 0, len(doc.Invites))
	for _, inv := range doc.Invites {
		invite, invErr := r.documentToInvite(&inv)
		if invErr != nil {
			continue // пропускаем некорректные приглашения
		}
		invites = append(invites, invite)
	}

	return workspacedomain.Reconstruct(
		id,
		doc.Name,
		doc.KeycloakGroupID,
		createdBy,
		doc.CreatedAt,
		doc.UpdatedAt,
		invites,
	), nil
}

// documentToInvite преобразует inviteDocument в Invite
func (r *MongoWorkspaceRepository) documentToInvite(doc *inviteDocument) (*workspacedomain.Invite, error) {
	if doc == nil {
		return nil, errs.ErrInvalidInput
	}

	id, err := uuid.ParseUUID(doc.InviteID)
	if err != nil {
		return nil, errs.ErrInvalidInput
	}

	workspaceID, err := uuid.ParseUUID(doc.WorkspaceID)
	if err != nil {
		return nil, errs.ErrInvalidInput
	}

	createdBy, err := uuid.ParseUUID(doc.CreatedBy)
	if err != nil {
		return nil, errs.ErrInvalidInput
	}

	return workspacedomain.ReconstructInvite(
		id,
		workspaceID,
		doc.Token,
		createdBy,
		doc.CreatedAt,
		doc.ExpiresAt,
		doc.MaxUses,
		doc.UsedCount,
		doc.IsRevoked,
	), nil
}
