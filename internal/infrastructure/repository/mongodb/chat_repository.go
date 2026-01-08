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

	"github.com/lllypuk/flowra/internal/application/appcore"
	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// MongoChatRepository implements chatapp.CommandRepository (application layer interface)
// using MongoDB and Event Sourcing
type MongoChatRepository struct {
	eventStore    appcore.EventStore
	readModelColl *mongo.Collection
	logger        *slog.Logger
}

// ChatRepoOption configures MongoChatRepository.
type ChatRepoOption func(*MongoChatRepository)

// WithChatRepoLogger sets the logger for chat repository.
func WithChatRepoLogger(logger *slog.Logger) ChatRepoOption {
	return func(r *MongoChatRepository) {
		r.logger = logger
	}
}

// NewMongoChatRepository creates a New MongoDB Chat Repository
func NewMongoChatRepository(
	eventStore appcore.EventStore,
	readModelColl *mongo.Collection,
	opts ...ChatRepoOption,
) *MongoChatRepository {
	r := &MongoChatRepository{
		eventStore:    eventStore,
		readModelColl: readModelColl,
		logger:        slog.Default(),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Load loads Chat from event store by restoring state from events (event sourcing)
func (r *MongoChatRepository) Load(ctx context.Context, chatID uuid.UUID) (*chatdomain.Chat, error) {
	if chatID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	// Load events from event store
	events, err := r.eventStore.LoadEvents(ctx, chatID.String())
	if err != nil {
		if errors.Is(err, appcore.ErrAggregateNotFound) {
			return nil, errs.ErrNotFound
		}
		r.logger.ErrorContext(ctx, "failed to load events from event store",
			slog.String("chat_id", chatID.String()),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to load events for chat %s: %w", chatID, err)
	}

	if len(events) == 0 {
		return nil, errs.ErrNotFound
	}

	// Create New Chat and apply events
	chat := &chatdomain.Chat{}
	for _, domainEvent := range events {
		if chatErr := chat.Apply(domainEvent); chatErr != nil {
			r.logger.ErrorContext(ctx, "failed to apply event to chat aggregate",
				slog.String("chat_id", chatID.String()),
				slog.String("event_type", domainEvent.EventType()),
				slog.String("error", chatErr.Error()),
			)
			return nil, fmt.Errorf("failed to apply event: %w", chatErr)
		}
	}

	// Mark events as committed (they are already saved)
	chat.MarkEventsAsCommitted()

	return chat, nil
}

// Save saves Chat by storing New events in event store and updating read model
func (r *MongoChatRepository) Save(ctx context.Context, chat *chatdomain.Chat) error {
	if chat == nil {
		return errs.ErrInvalidInput
	}

	uncommittedEvents := chat.GetUncommittedEvents()
	if len(uncommittedEvents) == 0 {
		return nil // Nothing to save
	}

	// 1. Save events to event store
	expectedVersion := chat.Version() - len(uncommittedEvents)
	err := r.eventStore.SaveEvents(ctx, chat.ID().String(), uncommittedEvents, expectedVersion)
	if err != nil {
		if errors.Is(err, appcore.ErrConcurrencyConflict) {
			r.logger.WarnContext(ctx, "concurrency conflict while saving chat events",
				slog.String("chat_id", chat.ID().String()),
				slog.Int("expected_version", expectedVersion),
				slog.Int("events_count", len(uncommittedEvents)),
			)
			return errs.ErrConcurrentModification
		}
		r.logger.ErrorContext(ctx, "failed to save chat events to event store",
			slog.String("chat_id", chat.ID().String()),
			slog.Int("events_count", len(uncommittedEvents)),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to save events: %w", err)
	}

	// 2. Update read model (denormalized representation)
	err = r.updateReadModel(ctx, chat)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to update chat read model",
			slog.String("chat_id", chat.ID().String()),
			slog.String("error", err.Error()),
		)
		// Don't fail - read model can be recalculated
	}

	// 3. Mark events as committed
	chat.MarkEventsAsCommitted()

	return nil
}

// GetEvents returns all event chat
func (r *MongoChatRepository) GetEvents(ctx context.Context, chatID uuid.UUID) ([]event.DomainEvent, error) {
	if chatID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	events, err := r.eventStore.LoadEvents(ctx, chatID.String())
	if err != nil {
		if errors.Is(err, appcore.ErrAggregateNotFound) {
			return nil, errs.ErrNotFound
		}
		r.logger.ErrorContext(ctx, "failed to get chat events",
			slog.String("chat_id", chatID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return events, nil
}

// updateReadModel obnovlyaet denormalizovannoe view in read model kollektsii
func (r *MongoChatRepository) updateReadModel(ctx context.Context, chat *chatdomain.Chat) error {
	// Checking, that u nas est bazovaya information for read model
	if chat.ID().IsZero() {
		return errs.ErrInvalidInput
	}

	// preobrazuem participants in stroki
	participantStrs := make([]string, len(chat.Participants()))
	for i, p := range chat.Participants() {
		participantStrs[i] = p.UserID().String()
	}

	// formiruem dokument read model
	doc := bson.M{
		"chat_id":      chat.ID().String(),
		"workspace_id": chat.WorkspaceID().String(),
		"type":         string(chat.Type()),
		"title":        chat.Title(), // Always save title for all chat types
		"is_public":    chat.IsPublic(),
		"created_by":   chat.CreatedBy().String(),
		"created_at":   chat.CreatedAt(),
		"participants": participantStrs,
	}

	// dobavlyaem dopolnitelnye fields for typed chats (task/bug/epic)
	if chat.Type() != chatdomain.TypeDiscussion {
		doc["status"] = chat.Status()
		doc["priority"] = chat.Priority()

		if chat.AssigneeID() != nil {
			doc["assigned_to"] = chat.AssigneeID().String()
		}

		if chat.DueDate() != nil {
			doc["due_date"] = *chat.DueDate()
		}

		if chat.Type() == chatdomain.TypeBug {
			doc["severity"] = chat.Severity()
		}
	}

	// ispolzuem upsert for creating or updating dokumenta
	filter := bson.M{"chat_id": chat.ID().String()}
	update := bson.M{"$set": doc}
	opts := options.UpdateOne().SetUpsert(true)

	_, err := r.readModelColl.UpdateOne(ctx, filter, update, opts)
	return HandleMongoError(err, "chat_read_model")
}

// MongoChatReadModelRepository realizuet chatapp.QueryRepository (application layer interface)
// for query operatsiy
type MongoChatReadModelRepository struct {
	collection *mongo.Collection
	eventStore appcore.EventStore
	logger     *slog.Logger
}

// ChatReadModelRepoOption configures MongoChatReadModelRepository.
type ChatReadModelRepoOption func(*MongoChatReadModelRepository)

// WithChatReadModelRepoLogger sets the logger for chat read model repository.
func WithChatReadModelRepoLogger(logger *slog.Logger) ChatReadModelRepoOption {
	return func(r *MongoChatReadModelRepository) {
		r.logger = logger
	}
}

// NewMongoChatReadModelRepository creates New MongoDB Chat Read Model Repository
func NewMongoChatReadModelRepository(
	collection *mongo.Collection,
	eventStore appcore.EventStore,
	opts ...ChatReadModelRepoOption,
) *MongoChatReadModelRepository {
	r := &MongoChatReadModelRepository{
		collection: collection,
		eventStore: eventStore,
		logger:     slog.Default(),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// FindByID finds chat by ID from read model
func (r *MongoChatReadModelRepository) FindByID(ctx context.Context, chatID uuid.UUID) (*chatapp.ReadModel, error) {
	if chatID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"chat_id": chatID.String()}
	var doc bson.M
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			r.logger.ErrorContext(ctx, "failed to find chat by ID",
				slog.String("chat_id", chatID.String()),
				slog.String("error", err.Error()),
			)
		}
		return nil, HandleMongoError(err, "chat")
	}

	return r.documentToReadModel(doc)
}

// FindByWorkspace finds chats in workspace with filters
func (r *MongoChatReadModelRepository) FindByWorkspace(
	ctx context.Context,
	workspaceID uuid.UUID,
	filters chatapp.Filters,
) ([]*chatapp.ReadModel, error) {
	if workspaceID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	// formiruem filter
	filter := bson.M{"workspace_id": workspaceID.String()}

	if filters.Type != nil {
		filter["type"] = string(*filters.Type)
	}

	if filters.IsPublic != nil {
		filter["is_public"] = *filters.IsPublic
	}

	if filters.UserID != nil {
		filter["participants"] = filters.UserID.String()
	}

	// formiruem optsii (paginatsiya, sort)
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(filters.Limit)).
		SetSkip(int64(filters.Offset))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to find chats by workspace",
			slog.String("workspace_id", workspaceID.String()),
			slog.String("error", err.Error()),
		)
		return nil, HandleMongoError(err, "chats")
	}
	defer cursor.Close(ctx)

	var readModels []*chatapp.ReadModel
	for cursor.Next(ctx) {
		var doc bson.M
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			r.logger.ErrorContext(ctx, "failed to decode chat read model",
				slog.String("workspace_id", workspaceID.String()),
				slog.String("error", decodeErr.Error()),
			)
			return nil, fmt.Errorf("failed to decode chat read model: %w", decodeErr)
		}

		rm, docErr := r.documentToReadModel(doc)
		if docErr != nil {
			r.logger.WarnContext(ctx, "skipping invalid chat document",
				slog.String("workspace_id", workspaceID.String()),
				slog.String("error", docErr.Error()),
			)
			continue // propuskaem nekorrektnye dokumenty
		}

		readModels = append(readModels, rm)
	}

	if err = cursor.Err(); err != nil {
		r.logger.ErrorContext(ctx, "cursor error while reading chats",
			slog.String("workspace_id", workspaceID.String()),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if readModels == nil {
		readModels = make([]*chatapp.ReadModel, 0)
	}

	return readModels, nil
}

// FindByParticipant finds chats for user
func (r *MongoChatReadModelRepository) FindByParticipant(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*chatapp.ReadModel, error) {
	if userID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"participants": userID.String()}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to find chats by participant",
			slog.String("user_id", userID.String()),
			slog.String("error", err.Error()),
		)
		return nil, HandleMongoError(err, "chats")
	}
	defer cursor.Close(ctx)

	var readModels []*chatapp.ReadModel
	for cursor.Next(ctx) {
		var doc bson.M
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			r.logger.WarnContext(ctx, "failed to decode chat document",
				slog.String("user_id", userID.String()),
				slog.String("error", decodeErr.Error()),
			)
			continue
		}

		rm, docErr := r.documentToReadModel(doc)
		if docErr != nil {
			r.logger.WarnContext(ctx, "skipping invalid chat document",
				slog.String("user_id", userID.String()),
				slog.String("error", docErr.Error()),
			)
			continue
		}

		readModels = append(readModels, rm)
	}

	if readModels == nil {
		readModels = make([]*chatapp.ReadModel, 0)
	}

	return readModels, nil
}

// Count returns count chats in workspace
func (r *MongoChatReadModelRepository) Count(ctx context.Context, workspaceID uuid.UUID) (int, error) {
	if workspaceID.IsZero() {
		return 0, errs.ErrInvalidInput
	}

	filter := bson.M{"workspace_id": workspaceID.String()}
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to count chats in workspace",
			slog.String("workspace_id", workspaceID.String()),
			slog.String("error", err.Error()),
		)
		return 0, HandleMongoError(err, "chats")
	}

	return int(count), nil
}

// documentToReadModel preobrazuet BSON dokument in ReadModel
func (r *MongoChatReadModelRepository) documentToReadModel(doc bson.M) (*chatapp.ReadModel, error) {
	chatIDStr, ok := doc["chat_id"].(string)
	if !ok {
		return nil, errors.New("invalid chat_id type")
	}

	workspaceIDStr, ok := doc["workspace_id"].(string)
	if !ok {
		return nil, errors.New("invalid workspace_id type")
	}

	chatType, ok := doc["type"].(string)
	if !ok {
		return nil, errors.New("invalid type")
	}

	createdByStr, ok := doc["created_by"].(string)
	if !ok {
		return nil, errors.New("invalid created_by type")
	}

	isPublic, ok := doc["is_public"].(bool)
	if !ok {
		isPublic = false
	}

	var createdAt time.Time
	if createdAtVal, createdOk := doc["created_at"].(time.Time); createdOk {
		createdAt = createdAtVal
	}

	// Read title (may be empty for discussions)
	var title string
	if titleVal, titleOk := doc["title"].(string); titleOk {
		title = titleVal
	}

	// preobrazuem participants from list user ID strok
	var participants []chatdomain.Participant
	if participantsVal, participantOk := doc["participants"].(bson.A); participantOk {
		for _, pVal := range participantsVal {
			if userIDStr, strOk := pVal.(string); strOk {
				// Read model hranit only user IDs, creating minimalnyy participant
				participants = append(participants, chatdomain.NewParticipant(
					uuid.UUID(userIDStr),
					chatdomain.RoleMember, // Role not stored in read model
				))
			}
		}
	}

	rm := &chatapp.ReadModel{
		ID:           uuid.UUID(chatIDStr),
		WorkspaceID:  uuid.UUID(workspaceIDStr),
		Type:         chatdomain.Type(chatType),
		Title:        title,
		IsPublic:     isPublic,
		CreatedBy:    uuid.UUID(createdByStr),
		CreatedAt:    createdAt,
		Participants: participants,
	}

	return rm, nil
}
