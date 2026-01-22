package repair

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// TaskType defines the type of repair task.
type TaskType string

const (
	// TaskTypeReadModelSync indicates a read model synchronization task.
	TaskTypeReadModelSync TaskType = "readmodel_sync"
)

// Task represents a repair task that needs to be processed.
type Task struct {
	ID            string     `bson:"_id,omitempty"`
	AggregateID   string     `bson:"aggregate_id"`
	AggregateType string     `bson:"aggregate_type"`
	TaskType      TaskType   `bson:"task_type"`
	Error         string     `bson:"error"`
	CreatedAt     time.Time  `bson:"created_at"`
	RetryCount    int        `bson:"retry_count"`
	LastRetryAt   *time.Time `bson:"last_retry_at,omitempty"`
	CompletedAt   *time.Time `bson:"completed_at,omitempty"`
	Status        string     `bson:"status"` // "pending", "processing", "completed", "failed"
}

// Queue manages repair tasks for failed read model updates.
type Queue interface {
	// Add adds a new repair task to the queue.
	Add(ctx context.Context, task Task) error

	// Poll retrieves pending tasks from the queue.
	// Returns up to batchSize tasks with status "pending".
	Poll(ctx context.Context, batchSize int) ([]Task, error)

	// MarkCompleted marks a task as completed.
	MarkCompleted(ctx context.Context, taskID string) error

	// MarkFailed marks a task as failed and increments retry count.
	MarkFailed(ctx context.Context, taskID string, err error) error

	// GetStats returns queue statistics.
	GetStats(ctx context.Context) (*QueueStats, error)
}

// QueueStats contains statistics about the repair queue.
type QueueStats struct {
	PendingCount    int64
	ProcessingCount int64
	CompletedCount  int64
	FailedCount     int64
	TotalCount      int64
}

// MongoQueue implements Queue using MongoDB.
type MongoQueue struct {
	collection *mongo.Collection
	logger     *slog.Logger
}

// NewMongoQueue creates a new MongoDB-based repair queue.
func NewMongoQueue(collection *mongo.Collection, logger *slog.Logger) *MongoQueue {
	if logger == nil {
		logger = slog.Default()
	}
	return &MongoQueue{
		collection: collection,
		logger:     logger,
	}
}

// Add adds a new repair task to the queue.
func (q *MongoQueue) Add(ctx context.Context, task Task) error {
	// Set default values
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	if task.Status == "" {
		task.Status = "pending"
	}
	if task.RetryCount == 0 {
		task.RetryCount = 0
	}

	_, err := q.collection.InsertOne(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to insert repair task: %w", err)
	}

	q.logger.InfoContext(ctx, "added repair task to queue",
		slog.String("aggregate_id", task.AggregateID),
		slog.String("aggregate_type", task.AggregateType),
		slog.String("task_type", string(task.TaskType)),
	)

	return nil
}

// Poll retrieves pending tasks from the queue.
func (q *MongoQueue) Poll(ctx context.Context, batchSize int) ([]Task, error) {
	if batchSize <= 0 {
		batchSize = 10
	}

	// Find pending tasks ordered by created_at (oldest first)
	filter := bson.M{"status": "pending"}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: 1}}).
		SetLimit(int64(batchSize))

	cursor, err := q.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query repair tasks: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []Task
	if decodeErr := cursor.All(ctx, &tasks); decodeErr != nil {
		return nil, fmt.Errorf("failed to decode repair tasks: %w", decodeErr)
	}

	// Mark tasks as processing
	for i := range tasks {
		if tasks[i].ID != "" {
			update := bson.M{
				"$set": bson.M{
					"status":        "processing",
					"last_retry_at": time.Now(),
				},
				"$inc": bson.M{
					"retry_count": 1,
				},
			}
			taskFilter := bson.M{"_id": tasks[i].ID}
			_, updateErr := q.collection.UpdateOne(ctx, taskFilter, update)
			if updateErr != nil {
				q.logger.WarnContext(ctx, "failed to mark task as processing",
					slog.String("task_id", tasks[i].ID),
					slog.String("error", updateErr.Error()),
				)
			}
		}
	}

	return tasks, nil
}

// MarkCompleted marks a task as completed.
func (q *MongoQueue) MarkCompleted(ctx context.Context, taskID string) error {
	now := time.Now()
	filter := bson.M{"_id": taskID}
	update := bson.M{
		"$set": bson.M{
			"status":       "completed",
			"completed_at": now,
		},
	}

	result, err := q.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to mark task as completed: %w", err)
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("task not found: %s", taskID)
	}

	q.logger.InfoContext(ctx, "marked repair task as completed",
		slog.String("task_id", taskID),
	)

	return nil
}

// MarkFailed marks a task as failed and increments retry count.
func (q *MongoQueue) MarkFailed(ctx context.Context, taskID string, taskErr error) error {
	filter := bson.M{"_id": taskID}
	update := bson.M{
		"$set": bson.M{
			"status": "failed",
			"error":  taskErr.Error(),
		},
	}

	result, err := q.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to mark task as failed: %w", err)
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("task not found: %s", taskID)
	}

	q.logger.WarnContext(ctx, "marked repair task as failed",
		slog.String("task_id", taskID),
		slog.String("error", taskErr.Error()),
	)

	return nil
}

// GetStats returns queue statistics.
func (q *MongoQueue) GetStats(ctx context.Context) (*QueueStats, error) {
	stats := &QueueStats{}

	// Count by status
	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$status"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}

	cursor, err := q.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}
	defer cursor.Close(ctx)

	type statusCount struct {
		Status string `bson:"_id"`
		Count  int64  `bson:"count"`
	}

	var results []statusCount
	if decodeErr := cursor.All(ctx, &results); decodeErr != nil {
		return nil, fmt.Errorf("failed to decode queue stats: %w", decodeErr)
	}

	for _, result := range results {
		switch result.Status {
		case "pending":
			stats.PendingCount = result.Count
		case "processing":
			stats.ProcessingCount = result.Count
		case "completed":
			stats.CompletedCount = result.Count
		case "failed":
			stats.FailedCount = result.Count
		}
		stats.TotalCount += result.Count
	}

	return stats, nil
}
