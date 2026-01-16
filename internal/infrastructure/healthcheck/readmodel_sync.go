package healthcheck

import (
	"context"
	"fmt"
	"time"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ReadModelSyncChecker checks for ReadModel synchronization issues.
// This is a simplified version that can be enhanced later with sampling functionality.
type ReadModelSyncChecker struct {
	chatReadModel *mongo.Collection
	taskReadModel *mongo.Collection
	sampleSize    int
}

// NewReadModelSyncChecker creates a new ReadModel sync health checker.
func NewReadModelSyncChecker(
	chatReadModel *mongo.Collection,
	taskReadModel *mongo.Collection,
	sampleSize int,
) *ReadModelSyncChecker {
	if sampleSize <= 0 {
		sampleSize = 100
	}

	return &ReadModelSyncChecker{
		chatReadModel: chatReadModel,
		taskReadModel: taskReadModel,
		sampleSize:    sampleSize,
	}
}

// Name returns the name of this health checker.
func (c *ReadModelSyncChecker) Name() string {
	return "readmodel_sync"
}

// Check performs the health check.
// Currently returns a placeholder status. Full implementation would require
// EventStore sampling functionality to compare versions between EventStore and ReadModel.
func (c *ReadModelSyncChecker) Check(ctx context.Context) appcore.HealthStatus {
	// For now, we just check if collections are accessible
	chatCount, err := c.chatReadModel.CountDocuments(ctx, map[string]any{})
	if err != nil {
		return appcore.HealthStatus{
			Healthy:   false,
			Message:   fmt.Sprintf("failed to access chat read model: %v", err),
			CheckedAt: time.Now(),
		}
	}

	taskCount, err := c.taskReadModel.CountDocuments(ctx, map[string]any{})
	if err != nil {
		return appcore.HealthStatus{
			Healthy:   false,
			Message:   fmt.Sprintf("failed to access task read model: %v", err),
			CheckedAt: time.Now(),
		}
	}

	// In future, we would sample random aggregates and compare versions
	// For now, we just report that collections are accessible
	details := map[string]any{
		"chat_count":  chatCount,
		"task_count":  taskCount,
		"sample_size": c.sampleSize,
		"note":        "Full version comparison not yet implemented",
	}

	message := fmt.Sprintf("read models accessible: %d chats, %d tasks", chatCount, taskCount)

	return appcore.HealthStatus{
		Healthy:   true,
		Message:   message,
		Details:   details,
		CheckedAt: time.Now(),
	}
}
