package healthcheck

import (
	"context"
	"fmt"
	"time"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/infrastructure/eventbus"
)

// DeadLetterChecker checks the dead letter queue status.
type DeadLetterChecker struct {
	dlqHandler *eventbus.DeadLetterHandler
}

// NewDeadLetterChecker creates a new dead letter queue health checker.
func NewDeadLetterChecker(dlqHandler *eventbus.DeadLetterHandler) *DeadLetterChecker {
	return &DeadLetterChecker{
		dlqHandler: dlqHandler,
	}
}

// Name returns the name of this health checker.
func (c *DeadLetterChecker) Name() string {
	return "dead_letter_queue"
}

// Check performs the health check.
func (c *DeadLetterChecker) Check(ctx context.Context) appcore.HealthStatus {
	count, err := c.dlqHandler.QueueLength(ctx)
	if err != nil {
		return appcore.HealthStatus{
			Healthy:   false,
			Message:   fmt.Sprintf("failed to get dead letter queue length: %v", err),
			CheckedAt: time.Now(),
		}
	}

	healthy := count == 0

	details := map[string]any{
		"dead_letters": count,
	}

	message := fmt.Sprintf("dead letter queue: %d events", count)

	return appcore.HealthStatus{
		Healthy:   healthy,
		Message:   message,
		Details:   details,
		CheckedAt: time.Now(),
	}
}
