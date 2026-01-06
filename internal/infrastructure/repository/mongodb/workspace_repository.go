package mongodb

import (
	"context"
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

	_, err := r.collection.UpdateOne(ctx, filter, update, UpsertOptions())
	return HandleMongoError(err, "workspace")
}

// Delete удаляет рабочее пространство и всех его членов
func (r *MongoWorkspaceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if id.IsZero() {
		return errs.ErrInvalidInput
	}

	// Удаляем само рабочее пространство
	filter := bson.M{"workspace_id": id.String()}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return HandleMongoError(err, "workspace")
	}

	if result.DeletedCount == 0 {
		return errs.ErrNotFound
	}

	// Удаляем членов рабочего пространства
	// Примечание: в production среде рекомендуется использовать транзакции (replica set)
	memberFilter := bson.M{"workspace_id": id.String()}
	_, err = r.membersCollection.DeleteMany(ctx, memberFilter)
	if err != nil {
		return fmt.Errorf("failed to delete workspace members: %w", err)
	}

	return nil
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

// memberDocument представляет члена workspace в отдельной коллекции
type memberDocument struct {
	UserID      string    `bson:"user_id"`
	WorkspaceID string    `bson:"workspace_id"`
	Role        string    `bson:"role"`
	JoinedAt    time.Time `bson:"joined_at"`
}

// GetMember возвращает члена workspace по userID
func (r *MongoWorkspaceRepository) GetMember(
	ctx context.Context,
	workspaceID uuid.UUID,
	userID uuid.UUID,
) (*workspacedomain.Member, error) {
	if workspaceID.IsZero() || userID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{
		"workspace_id": workspaceID.String(),
		"user_id":      userID.String(),
	}

	var doc memberDocument
	err := r.membersCollection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, HandleMongoError(err, "member")
	}

	member := workspacedomain.ReconstructMember(
		userID,
		workspaceID,
		workspacedomain.Role(doc.Role),
		doc.JoinedAt,
	)
	return &member, nil
}

// IsMember проверяет, является ли пользователь членом workspace
func (r *MongoWorkspaceRepository) IsMember(
	ctx context.Context,
	workspaceID uuid.UUID,
	userID uuid.UUID,
) (bool, error) {
	if workspaceID.IsZero() || userID.IsZero() {
		return false, errs.ErrInvalidInput
	}

	filter := bson.M{
		"workspace_id": workspaceID.String(),
		"user_id":      userID.String(),
	}

	count, err := r.membersCollection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, HandleMongoError(err, "member")
	}

	return count > 0, nil
}

// ListWorkspacesByUser возвращает workspaces, в которых пользователь является членом
func (r *MongoWorkspaceRepository) ListWorkspacesByUser(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*workspacedomain.Workspace, error) {
	if userID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	// Находим все workspace_id, где пользователь является членом
	filter := bson.M{"user_id": userID.String()}
	opts := options.Find().
		SetSort(bson.D{{Key: "joined_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.membersCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, HandleMongoError(err, "members")
	}
	defer cursor.Close(ctx)

	var workspaceIDs []string
	for cursor.Next(ctx) {
		var doc memberDocument
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			continue
		}
		workspaceIDs = append(workspaceIDs, doc.WorkspaceID)
	}

	if len(workspaceIDs) == 0 {
		return make([]*workspacedomain.Workspace, 0), nil
	}

	// Загружаем workspaces по найденным ID
	wsFilter := bson.M{"workspace_id": bson.M{"$in": workspaceIDs}}
	wsCursor, err := r.collection.Find(ctx, wsFilter)
	if err != nil {
		return nil, HandleMongoError(err, "workspaces")
	}
	defer wsCursor.Close(ctx)

	// Создаём map для сохранения порядка
	workspaceMap := make(map[string]*workspacedomain.Workspace)
	for wsCursor.Next(ctx) {
		var doc workspaceDocument
		if decodeErr := wsCursor.Decode(&doc); decodeErr != nil {
			continue
		}

		ws, docErr := r.documentToWorkspace(&doc)
		if docErr != nil {
			continue
		}

		workspaceMap[doc.WorkspaceID] = ws
	}

	// Собираем результат в порядке workspaceIDs
	workspaces := make([]*workspacedomain.Workspace, 0, len(workspaceIDs))
	for _, wsID := range workspaceIDs {
		if ws, ok := workspaceMap[wsID]; ok {
			workspaces = append(workspaces, ws)
		}
	}

	return workspaces, nil
}

// CountWorkspacesByUser возвращает количество workspaces пользователя
func (r *MongoWorkspaceRepository) CountWorkspacesByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	if userID.IsZero() {
		return 0, errs.ErrInvalidInput
	}

	filter := bson.M{"user_id": userID.String()}
	count, err := r.membersCollection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, HandleMongoError(err, "members")
	}

	return int(count), nil
}

// AddMember добавляет члена в workspace
func (r *MongoWorkspaceRepository) AddMember(ctx context.Context, member *workspacedomain.Member) error {
	if member == nil {
		return errs.ErrInvalidInput
	}

	if member.UserID().IsZero() || member.WorkspaceID().IsZero() {
		return errs.ErrInvalidInput
	}

	doc := memberDocument{
		UserID:      member.UserID().String(),
		WorkspaceID: member.WorkspaceID().String(),
		Role:        member.Role().String(),
		JoinedAt:    member.JoinedAt(),
	}

	filter := bson.M{
		"workspace_id": member.WorkspaceID().String(),
		"user_id":      member.UserID().String(),
	}
	update := bson.M{"$set": doc}

	_, err := r.membersCollection.UpdateOne(ctx, filter, update, UpsertOptions())
	return HandleMongoError(err, "member")
}

// UpdateMember обновляет данные члена workspace
func (r *MongoWorkspaceRepository) UpdateMember(ctx context.Context, member *workspacedomain.Member) error {
	if member == nil {
		return errs.ErrInvalidInput
	}

	if member.UserID().IsZero() || member.WorkspaceID().IsZero() {
		return errs.ErrInvalidInput
	}

	filter := bson.M{
		"workspace_id": member.WorkspaceID().String(),
		"user_id":      member.UserID().String(),
	}

	update := bson.M{
		"$set": bson.M{
			"role": member.Role().String(),
		},
	}

	result, err := r.membersCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return HandleMongoError(err, "member")
	}

	if result.MatchedCount == 0 {
		return errs.ErrNotFound
	}

	return nil
}

// RemoveMember удаляет члена из workspace
func (r *MongoWorkspaceRepository) RemoveMember(
	ctx context.Context,
	workspaceID uuid.UUID,
	userID uuid.UUID,
) error {
	if workspaceID.IsZero() || userID.IsZero() {
		return errs.ErrInvalidInput
	}

	filter := bson.M{
		"workspace_id": workspaceID.String(),
		"user_id":      userID.String(),
	}

	result, err := r.membersCollection.DeleteOne(ctx, filter)
	if err != nil {
		return HandleMongoError(err, "member")
	}

	if result.DeletedCount == 0 {
		return errs.ErrNotFound
	}

	return nil
}

// ListMembers возвращает всех членов workspace
func (r *MongoWorkspaceRepository) ListMembers(
	ctx context.Context,
	workspaceID uuid.UUID,
	offset, limit int,
) ([]*workspacedomain.Member, error) {
	if workspaceID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"workspace_id": workspaceID.String()}
	opts := options.Find().
		SetSort(bson.D{{Key: "joined_at", Value: 1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.membersCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, HandleMongoError(err, "members")
	}
	defer cursor.Close(ctx)

	var members []*workspacedomain.Member
	for cursor.Next(ctx) {
		var doc memberDocument
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			continue
		}

		userID, parseErr := uuid.ParseUUID(doc.UserID)
		if parseErr != nil {
			continue
		}

		wsID, parseErr := uuid.ParseUUID(doc.WorkspaceID)
		if parseErr != nil {
			continue
		}

		member := workspacedomain.ReconstructMember(
			userID,
			wsID,
			workspacedomain.Role(doc.Role),
			doc.JoinedAt,
		)
		members = append(members, &member)
	}

	if members == nil {
		members = make([]*workspacedomain.Member, 0)
	}

	return members, nil
}

// CountMembers возвращает количество членов workspace
func (r *MongoWorkspaceRepository) CountMembers(ctx context.Context, workspaceID uuid.UUID) (int, error) {
	if workspaceID.IsZero() {
		return 0, errs.ErrInvalidInput
	}

	filter := bson.M{"workspace_id": workspaceID.String()}
	count, err := r.membersCollection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, HandleMongoError(err, "members")
	}

	return int(count), nil
}
