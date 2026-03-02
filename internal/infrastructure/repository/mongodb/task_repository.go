package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lllypuk/flowra/internal/application/appcore"
	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/domain/errs"
	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// MongoTaskRepository provides query-only access to task read model.
type MongoTaskRepository struct {
	collection *mongo.Collection
	eventStore appcore.EventStore
	logger     *slog.Logger
}

// TaskRepoOption configures MongoTaskRepository.
type TaskRepoOption func(*MongoTaskRepository)

// WithTaskRepoLogger sets the logger for task repository.
func WithTaskRepoLogger(logger *slog.Logger) TaskRepoOption {
	return func(r *MongoTaskRepository) {
		r.logger = logger
	}
}

// NewMongoTaskRepository creates new MongoDB task query repository.
func NewMongoTaskRepository(
	eventStore appcore.EventStore,
	readModelColl *mongo.Collection,
	opts ...TaskRepoOption,
) *MongoTaskRepository {
	r := &MongoTaskRepository{
		collection: readModelColl,
		eventStore: eventStore,
		logger:     slog.Default(),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// FindByID finds a task by ID from read model.
func (r *MongoTaskRepository) FindByID(ctx context.Context, taskID uuid.UUID) (*taskapp.ReadModel, error) {
	if taskID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"task_id": taskID.String()}
	var doc taskReadModelDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, HandleMongoError(err, "task")
	}

	return r.documentToReadModel(&doc)
}

// FindByChatID finds a task by associated chat ID.
func (r *MongoTaskRepository) FindByChatID(ctx context.Context, chatID uuid.UUID) (*taskapp.ReadModel, error) {
	if chatID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"chat_id": chatID.String()}
	var doc taskReadModelDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, HandleMongoError(err, "task")
	}

	return r.documentToReadModel(&doc)
}

// FindByAssignee finds tasks by assignee.
func (r *MongoTaskRepository) FindByAssignee(
	ctx context.Context,
	assigneeID uuid.UUID,
	filters taskapp.Filters,
) ([]*taskapp.ReadModel, error) {
	if assigneeID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"assigned_to": assigneeID.String()}
	r.applyFilters(filter, filters)

	return r.findMany(ctx, filter, filters)
}

// FindByStatus finds tasks by status.
func (r *MongoTaskRepository) FindByStatus(
	ctx context.Context,
	status taskdomain.Status,
	filters taskapp.Filters,
) ([]*taskapp.ReadModel, error) {
	filter := bson.M{"status": string(status)}
	r.applyFilters(filter, filters)

	return r.findMany(ctx, filter, filters)
}

// List returns list of tasks with filters.
func (r *MongoTaskRepository) List(ctx context.Context, filters taskapp.Filters) ([]*taskapp.ReadModel, error) {
	filter := bson.M{}
	r.applyFilters(filter, filters)

	return r.findMany(ctx, filter, filters)
}

// Count returns count of tasks with filters.
func (r *MongoTaskRepository) Count(ctx context.Context, filters taskapp.Filters) (int, error) {
	filter := bson.M{}
	r.applyFilters(filter, filters)

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, HandleMongoError(err, "tasks")
	}

	return int(count), nil
}

// applyFilters applies filters to MongoDB query.
func (r *MongoTaskRepository) applyFilters(filter bson.M, filters taskapp.Filters) {
	if filters.ChatID != nil {
		filter["chat_id"] = filters.ChatID.String()
	}
	if filters.AssigneeID != nil {
		filter["assigned_to"] = filters.AssigneeID.String()
	}
	if filters.Status != nil {
		filter["status"] = string(*filters.Status)
	}
	if filters.Priority != nil {
		filter["priority"] = string(*filters.Priority)
	}
	if filters.EntityType != nil {
		filter["entity_type"] = string(*filters.EntityType)
	}
	if filters.CreatedBy != nil {
		filter["created_by"] = filters.CreatedBy.String()
	}
	if filters.Search != "" {
		filter["title"] = bson.M{"$regex": filters.Search, "$options": "i"}
	}
}

// findMany performs search with pagination.
func (r *MongoTaskRepository) findMany(
	ctx context.Context,
	filter bson.M,
	filters taskapp.Filters,
) ([]*taskapp.ReadModel, error) {
	limit := DefaultLimitWithMax(filters.Limit, DefaultPaginationLimit, MaxPaginationLimit)

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(filters.Offset))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, HandleMongoError(err, "tasks")
	}
	defer cursor.Close(ctx)

	var results []*taskapp.ReadModel
	for cursor.Next(ctx) {
		var doc taskReadModelDocument
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			continue
		}

		rm, docErr := r.documentToReadModel(&doc)
		if docErr != nil {
			continue
		}

		results = append(results, rm)
	}

	if err = cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if results == nil {
		results = make([]*taskapp.ReadModel, 0)
	}

	return results, nil
}

// taskReadModelDocument represents read model document.
type taskReadModelDocument struct {
	TaskID      string                   `bson:"task_id"`
	ChatID      string                   `bson:"chat_id"`
	Title       string                   `bson:"title"`
	EntityType  string                   `bson:"entity_type"`
	Status      string                   `bson:"status"`
	Priority    string                   `bson:"priority"`
	Severity    string                   `bson:"severity,omitempty"`
	AssignedTo  *string                  `bson:"assigned_to,omitempty"`
	DueDate     *time.Time               `bson:"due_date,omitempty"`
	CreatedBy   string                   `bson:"created_by"`
	CreatedAt   time.Time                `bson:"created_at"`
	Version     int                      `bson:"version"`
	Attachments []taskAttachmentDocument `bson:"attachments,omitempty"`
}

// taskAttachmentDocument represents an attachment in the read model document.
type taskAttachmentDocument struct {
	FileID   string `bson:"file_id"`
	FileName string `bson:"file_name"`
	FileSize int64  `bson:"file_size"`
	MimeType string `bson:"mime_type"`
}

// documentToReadModel converts BSON document to task read model.
func (r *MongoTaskRepository) documentToReadModel(doc *taskReadModelDocument) (*taskapp.ReadModel, error) {
	if doc == nil {
		return nil, errs.ErrInvalidInput
	}

	rm := &taskapp.ReadModel{
		ID:         uuid.UUID(doc.TaskID),
		ChatID:     uuid.UUID(doc.ChatID),
		Title:      doc.Title,
		EntityType: taskdomain.EntityType(doc.EntityType),
		Status:     taskdomain.Status(doc.Status),
		Priority:   taskdomain.Priority(doc.Priority),
		Severity:   doc.Severity,
		CreatedBy:  uuid.UUID(doc.CreatedBy),
		CreatedAt:  doc.CreatedAt,
		Version:    doc.Version,
	}

	if doc.AssignedTo != nil {
		assignee := uuid.UUID(*doc.AssignedTo)
		rm.AssignedTo = &assignee
	}

	if doc.DueDate != nil {
		rm.DueDate = doc.DueDate
	}

	for _, a := range doc.Attachments {
		rm.Attachments = append(rm.Attachments, taskapp.AttachmentReadModel{
			FileID:   uuid.UUID(a.FileID),
			FileName: a.FileName,
			FileSize: a.FileSize,
			MimeType: a.MimeType,
		})
	}

	return rm, nil
}

// Compile-time interface checks.
var _ taskapp.QueryRepository = (*MongoTaskRepository)(nil)
