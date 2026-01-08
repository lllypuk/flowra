package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	workspacedomain "github.com/lllypuk/flowra/internal/domain/workspace"
)

// MongoWorkspaceRepository realizuet workspaceapp.Repository (application layer interface)
type MongoWorkspaceRepository struct {
	collection        *mongo.Collection
	membersCollection *mongo.Collection
	logger            *slog.Logger
}

// WorkspaceRepoOption configures MongoWorkspaceRepository.
type WorkspaceRepoOption func(*MongoWorkspaceRepository)

// WithWorkspaceRepoLogger sets the logger for workspace repository.
func WithWorkspaceRepoLogger(logger *slog.Logger) WorkspaceRepoOption {
	return func(r *MongoWorkspaceRepository) {
		r.logger = logger
	}
}

// NewMongoWorkspaceRepository creates New MongoDB Workspace Repository
func NewMongoWorkspaceRepository(
	collection *mongo.Collection,
	membersCollection *mongo.Collection,
	opts ...WorkspaceRepoOption,
) *MongoWorkspaceRepository {
	r := &MongoWorkspaceRepository{
		collection:        collection,
		membersCollection: membersCollection,
		logger:            slog.Default(),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// FindByID finds workspace space po ID
func (r *MongoWorkspaceRepository) FindByID(ctx context.Context, id uuid.UUID) (*workspacedomain.Workspace, error) {
	if id.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"workspace_id": id.String()}
	var doc workspaceDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			r.logger.ErrorContext(ctx, "failed to find workspace by ID",
				slog.String("workspace_id", id.String()),
				slog.String("error", err.Error()),
			)
		}
		return nil, HandleMongoError(err, "workspace")
	}

	return r.documentToWorkspace(&doc)
}

// FindByKeycloakGroup finds workspace space po ID groups Keycloak
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
		if !errors.Is(err, mongo.ErrNoDocuments) {
			r.logger.ErrorContext(ctx, "failed to find workspace by Keycloak group",
				slog.String("keycloak_group_id", keycloakGroupID),
				slog.String("error", err.Error()),
			)
		}
		return nil, HandleMongoError(err, "workspace")
	}

	return r.documentToWorkspace(&doc)
}

// Save saves workspace space
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
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to save workspace",
			slog.String("workspace_id", workspace.ID().String()),
			slog.String("name", workspace.Name()),
			slog.String("error", err.Error()),
		)
	}
	return HandleMongoError(err, "workspace")
}

// Delete udalyaet workspace space and all ego chlenov
func (r *MongoWorkspaceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if id.IsZero() {
		return errs.ErrInvalidInput
	}

	// udalyaem samo workspace space
	filter := bson.M{"workspace_id": id.String()}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to delete workspace",
			slog.String("workspace_id", id.String()),
			slog.String("error", err.Error()),
		)
		return HandleMongoError(err, "workspace")
	}

	if result.DeletedCount == 0 {
		return errs.ErrNotFound
	}

	// udalyaem chlenov workspace prostranstva
	// primechanie: in production srede rekomenduetsya user tranzaktsii (replica set)
	memberFilter := bson.M{"workspace_id": id.String()}
	_, err = r.membersCollection.DeleteMany(ctx, memberFilter)
	if err != nil {
		return fmt.Errorf("failed to delete workspace members: %w", err)
	}

	return nil
}

// List returns list workspace prostranstv s paginatsiey
func (r *MongoWorkspaceRepository) List(ctx context.Context, offset, limit int) ([]*workspacedomain.Workspace, error) {
	return listDocuments(ctx, r.collection, offset, limit, r.documentToWorkspace, "workspaces")
}

// Count returns obschee count workspace prostranstv
func (r *MongoWorkspaceRepository) Count(ctx context.Context) (int, error) {
	count, err := CountAll(ctx, r.collection)
	if err != nil {
		return 0, HandleMongoError(err, "workspaces")
	}
	return count, nil
}

// FindInviteByToken finds priglashenie po tokenu
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

	// nahodim konkretnoe priglashenie in massive
	for _, inv := range doc.Invites {
		if inv.Token == token {
			return r.documentToInvite(&inv)
		}
	}

	return nil, errs.ErrNotFound
}

// workspaceDocument represents strukturu dokumenta in MongoDB
type workspaceDocument struct {
	WorkspaceID     string           `bson:"workspace_id"`
	Name            string           `bson:"name"`
	Description     string           `bson:"description"`
	KeycloakGroupID string           `bson:"keycloak_group_id"`
	CreatedBy       string           `bson:"created_by"`
	CreatedAt       time.Time        `bson:"created_at"`
	UpdatedAt       time.Time        `bson:"updated_at"`
	Invites         []inviteDocument `bson:"invites"`
}

// inviteDocument represents priglashenie in dokumente
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

// workspaceToDocument preobrazuet Workspace in Document
func (r *MongoWorkspaceRepository) workspaceToDocument(ws *workspacedomain.Workspace) workspaceDocument {
	// preobrazuem priglasheniya
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
		Description:     ws.Description(),
		KeycloakGroupID: ws.KeycloakGroupID(),
		CreatedBy:       ws.CreatedBy().String(),
		CreatedAt:       ws.CreatedAt(),
		UpdatedAt:       ws.UpdatedAt(),
		Invites:         invites,
	}
}

// documentToWorkspace preobrazuet Document in Workspace
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

	// vosstanavlivaem priglasheniya
	invites := make([]*workspacedomain.Invite, 0, len(doc.Invites))
	for _, inv := range doc.Invites {
		invite, invErr := r.documentToInvite(&inv)
		if invErr != nil {
			continue // propuskaem nekorrektnye priglasheniya
		}
		invites = append(invites, invite)
	}

	return workspacedomain.Reconstruct(
		id,
		doc.Name,
		doc.Description,
		doc.KeycloakGroupID,
		createdBy,
		doc.CreatedAt,
		doc.UpdatedAt,
		invites,
	), nil
}

// documentToInvite preobrazuet inviteDocument in Invite
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

// memberDocument represents chlena workspace in otdelnoy kollektsii
type memberDocument struct {
	UserID      string    `bson:"user_id"`
	WorkspaceID string    `bson:"workspace_id"`
	Role        string    `bson:"role"`
	JoinedAt    time.Time `bson:"joined_at"`
}

// GetMember returns chlena workspace po userID
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

// IsMember checks, is li user chlenom workspace
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

// ListWorkspacesByUser returns workspaces, in kotoryh user is chlenom
func (r *MongoWorkspaceRepository) ListWorkspacesByUser(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*workspacedomain.Workspace, error) {
	if userID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	// nahodim all workspace_id, gde user is chlenom
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

	// Loading workspaces po naydennym ID
	wsFilter := bson.M{"workspace_id": bson.M{"$in": workspaceIDs}}
	wsCursor, err := r.collection.Find(ctx, wsFilter)
	if err != nil {
		return nil, HandleMongoError(err, "workspaces")
	}
	defer wsCursor.Close(ctx)

	// sozdayom map for saving poryadka
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

	// sobiraem result in poryadke workspaceIDs
	workspaces := make([]*workspacedomain.Workspace, 0, len(workspaceIDs))
	for _, wsID := range workspaceIDs {
		if ws, ok := workspaceMap[wsID]; ok {
			workspaces = append(workspaces, ws)
		}
	}

	return workspaces, nil
}

// CountWorkspacesByUser returns count workspaces user
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

// AddMember adds chlena in workspace
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

// UpdateMember obnovlyaet data chlena workspace
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

// RemoveMember udalyaet chlena from workspace
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

// ListMembers returns all chlenov workspace
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

// CountMembers returns count chlenov workspace
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
