package healthcheck

import (
	"context"
	"fmt"
	"time"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/infrastructure/repair"
)

// RepairQueueChecker checks the repair queue status.
type RepairQueueChecker struct {
	repairQueue repair.Queue
	threshold   int64
}

// NewRepairQueueChecker creates a new repair queue health checker.
func NewRepairQueueChecker(repairQueue repair.Queue, threshold int64) *RepairQueueChecker {
	if threshold <= 0 {
		threshold = 10
	}

	return &RepairQueueChecker{
		repairQueue: repairQueue,
		threshold:   threshold,
	}
}

// Name returns the name of this health checker.
func (c *RepairQueueChecker) Name() string {
	return "repair_queue"
}

// Check performs the health check.
func (c *RepairQueueChecker) Check(ctx context.Context) appcore.HealthStatus {
	stats, err := c.repairQueue.GetStats(ctx)
	if err != nil {
		return appcore.HealthStatus{
			Healthy:   false,
			Message:   fmt.Sprintf("failed to get repair queue stats: %v", err),
			CheckedAt: time.Now(),
		}
	}

	healthy := stats.PendingCount < c.threshold && stats.FailedCount == 0

	details := map[string]any{
		"pending_repairs": stats.PendingCount,
		"failed_repairs":  stats.FailedCount,
		"threshold":       c.threshold,
	}

	message := fmt.Sprintf("repair queue: %d pending, %d failed", stats.PendingCount, stats.FailedCount)

	return appcore.HealthStatus{
		Healthy:   healthy,
		Message:   message,
		Details:   details,
		CheckedAt: time.Now(),
	}
}
