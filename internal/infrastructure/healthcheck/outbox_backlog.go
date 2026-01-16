// Package healthcheck provides health check implementations for monitoring system consistency.
package healthcheck

import (
	"context"
	"fmt"
	"time"

	"github.com/lllypuk/flowra/internal/application/appcore"
)

// Default thresholds for outbox backlog.
const (
	defaultWarningThreshold  = 100
	defaultCriticalThreshold = 1000
)

// OutboxBacklogChecker checks the outbox backlog size and age.
type OutboxBacklogChecker struct {
	outbox            appcore.Outbox
	warningThreshold  int64
	criticalThreshold int64
}

// OutboxBacklogOption configures OutboxBacklogChecker.
type OutboxBacklogOption func(*OutboxBacklogChecker)

// WithWarningThreshold sets the warning threshold for backlog count.
func WithWarningThreshold(threshold int64) OutboxBacklogOption {
	return func(c *OutboxBacklogChecker) {
		c.warningThreshold = threshold
	}
}

// WithCriticalThreshold sets the critical threshold for backlog count.
func WithCriticalThreshold(threshold int64) OutboxBacklogOption {
	return func(c *OutboxBacklogChecker) {
		c.criticalThreshold = threshold
	}
}

// NewOutboxBacklogChecker creates a new outbox backlog health checker.
func NewOutboxBacklogChecker(outbox appcore.Outbox, opts ...OutboxBacklogOption) *OutboxBacklogChecker {
	c := &OutboxBacklogChecker{
		outbox:            outbox,
		warningThreshold:  defaultWarningThreshold,
		criticalThreshold: defaultCriticalThreshold,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Name returns the name of this health checker.
func (c *OutboxBacklogChecker) Name() string {
	return "outbox_backlog"
}

// Check performs the health check.
func (c *OutboxBacklogChecker) Check(ctx context.Context) appcore.HealthStatus {
	count, oldest, err := c.outbox.Stats(ctx)
	if err != nil {
		return appcore.HealthStatus{
			Healthy:   false,
			Message:   fmt.Sprintf("failed to get outbox stats: %v", err),
			CheckedAt: time.Now(),
		}
	}

	var lag time.Duration
	if !oldest.IsZero() {
		lag = time.Since(oldest)
	}

	healthy := count < c.warningThreshold

	details := map[string]any{
		"backlog_count":      count,
		"warning_threshold":  c.warningThreshold,
		"critical_threshold": c.criticalThreshold,
	}

	if !oldest.IsZero() {
		details["oldest_event_age"] = lag.String()
	}

	message := fmt.Sprintf("outbox backlog: %d events", count)
	if !oldest.IsZero() {
		message = fmt.Sprintf("outbox backlog: %d events, oldest: %v ago", count, lag.Round(time.Second))
	}

	return appcore.HealthStatus{
		Healthy:   healthy,
		Message:   message,
		Details:   details,
		CheckedAt: time.Now(),
	}
}
