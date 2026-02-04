package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/infrastructure/metrics"
)

// Default outbox worker configuration values.
const (
	defaultOutboxPollInterval = 100 * time.Millisecond
	defaultOutboxBatchSize    = 100
	defaultOutboxMaxRetries   = 5
	defaultOutboxCleanupAge   = 7 * 24 * time.Hour // 7 days
)

// OutboxWorkerConfig contains configuration for the outbox worker.
type OutboxWorkerConfig struct {
	// PollInterval is the time between polling the outbox for new events.
	PollInterval time.Duration

	// BatchSize is the maximum number of events to process in each poll cycle.
	BatchSize int

	// MaxRetries is the maximum number of retry attempts for failed publishes.
	MaxRetries int

	// CleanupAge is the age after which processed entries are cleaned up.
	CleanupAge time.Duration

	// CleanupInterval is how often to run the cleanup process.
	CleanupInterval time.Duration

	// Enabled determines if the worker should run.
	Enabled bool
}

// DefaultOutboxWorkerConfig returns sensible default configuration.
func DefaultOutboxWorkerConfig() OutboxWorkerConfig {
	return OutboxWorkerConfig{
		PollInterval:    defaultOutboxPollInterval,
		BatchSize:       defaultOutboxBatchSize,
		MaxRetries:      defaultOutboxMaxRetries,
		CleanupAge:      defaultOutboxCleanupAge,
		CleanupInterval: 1 * time.Hour,
		Enabled:         true,
	}
}

// OutboxWorker processes events from the outbox and publishes them to the event bus.
type OutboxWorker struct {
	outbox   appcore.Outbox
	eventBus event.Bus
	logger   *slog.Logger
	config   OutboxWorkerConfig
	metrics  *metrics.OutboxMetrics
}

// NewOutboxWorker creates a new outbox worker.
func NewOutboxWorker(
	outbox appcore.Outbox,
	eventBus event.Bus,
	logger *slog.Logger,
	config OutboxWorkerConfig,
	metrics *metrics.OutboxMetrics,
) *OutboxWorker {
	if logger == nil {
		logger = slog.Default()
	}

	return &OutboxWorker{
		outbox:   outbox,
		eventBus: eventBus,
		logger:   logger,
		config:   config,
		metrics:  metrics,
	}
}

// Run starts the outbox worker and runs until the context is cancelled.
func (w *OutboxWorker) Run(ctx context.Context) error {
	if !w.config.Enabled {
		w.logger.InfoContext(ctx, "outbox worker is disabled")
		return nil
	}

	w.logger.InfoContext(ctx, "starting outbox worker",
		slog.Duration("poll_interval", w.config.PollInterval),
		slog.Int("batch_size", w.config.BatchSize),
		slog.Int("max_retries", w.config.MaxRetries),
	)

	pollTicker := time.NewTicker(w.config.PollInterval)
	defer pollTicker.Stop()

	cleanupTicker := time.NewTicker(w.config.CleanupInterval)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.InfoContext(ctx, "outbox worker stopped")
			return ctx.Err()

		case <-pollTicker.C:
			// Update metrics before processing
			w.updateGaugeMetrics(ctx)

			if err := w.processBatch(ctx); err != nil {
				w.logger.ErrorContext(ctx, "failed to process outbox batch",
					slog.String("error", err.Error()),
				)
			}

		case <-cleanupTicker.C:
			deleted, err := w.outbox.Cleanup(ctx, w.config.CleanupAge)
			if err != nil {
				w.logger.ErrorContext(ctx, "failed to cleanup outbox",
					slog.String("error", err.Error()),
				)
			} else if w.metrics != nil && deleted > 0 {
				w.metrics.CleanupDeletedTotal.Add(float64(deleted))
			}
		}
	}
}

// processBatch polls and processes a batch of events from the outbox.
func (w *OutboxWorker) processBatch(ctx context.Context) error {
	entries, err := w.outbox.Poll(ctx, w.config.BatchSize)
	if err != nil {
		return fmt.Errorf("failed to poll outbox: %w", err)
	}

	if len(entries) == 0 {
		return nil
	}

	// Record poll batch size
	if w.metrics != nil {
		w.metrics.PollBatchSize.Observe(float64(len(entries)))
	}

	w.logger.DebugContext(ctx, "processing outbox batch",
		slog.Int("count", len(entries)),
	)

	var processed, failed int
	for _, entry := range entries {
		if processErr := w.processEntry(ctx, entry); processErr != nil {
			failed++
			w.logger.WarnContext(ctx, "failed to process outbox entry",
				slog.String("entry_id", entry.ID),
				slog.String("event_type", entry.EventType),
				slog.String("error", processErr.Error()),
			)
		} else {
			processed++
		}
	}

	if processed > 0 || failed > 0 {
		w.logger.DebugContext(ctx, "outbox batch completed",
			slog.Int("processed", processed),
			slog.Int("failed", failed),
		)
	}

	return nil
}

// processEntry publishes a single outbox entry to the event bus.
func (w *OutboxWorker) processEntry(ctx context.Context, entry appcore.OutboxEntry) error {
	// Record processing duration (from creation to now)
	defer func() {
		if w.metrics != nil {
			processingDuration := time.Since(entry.CreatedAt).Seconds()
			w.metrics.ProcessingDuration.WithLabelValues(entry.EventType).Observe(processingDuration)
		}
	}()

	// Check if max retries exceeded
	if entry.RetryCount >= w.config.MaxRetries {
		w.logger.ErrorContext(ctx, "outbox entry exceeded max retries, marking as processed",
			slog.String("entry_id", entry.ID),
			slog.String("event_type", entry.EventType),
			slog.Int("retry_count", entry.RetryCount),
			slog.String("last_error", entry.LastError),
		)
		// Mark as processed to prevent infinite retries
		if err := w.outbox.MarkProcessed(ctx, entry.ID); err != nil {
			return err
		}
		// Record as failed
		if w.metrics != nil {
			w.metrics.EventsProcessed.WithLabelValues(entry.EventType, "failed").Inc()
		}
		return nil
	}

	// Create event from outbox entry
	evt := &outboxEvent{
		eventType:     entry.EventType,
		aggregateID:   entry.AggregateID,
		aggregateType: entry.AggregateType,
		occurredAt:    entry.CreatedAt,
		payload:       entry.Payload,
	}

	// Publish to event bus with timing
	publishStart := time.Now()
	if err := w.eventBus.Publish(ctx, evt); err != nil {
		// Record retry metric
		if w.metrics != nil {
			w.metrics.RetryTotal.WithLabelValues(entry.EventType).Inc()
		}

		// Mark as failed for retry
		if markErr := w.outbox.MarkFailed(ctx, entry.ID, err); markErr != nil {
			w.logger.ErrorContext(ctx, "failed to mark outbox entry as failed",
				slog.String("entry_id", entry.ID),
				slog.String("error", markErr.Error()),
			)
		}
		return fmt.Errorf("failed to publish event: %w", err)
	}

	// Record publish duration
	if w.metrics != nil {
		publishDuration := time.Since(publishStart).Seconds()
		w.metrics.PublishDuration.WithLabelValues(entry.EventType).Observe(publishDuration)
	}

	// Mark as processed
	if err := w.outbox.MarkProcessed(ctx, entry.ID); err != nil {
		return fmt.Errorf("failed to mark entry as processed: %w", err)
	}

	// Record successful processing
	if w.metrics != nil {
		w.metrics.EventsProcessed.WithLabelValues(entry.EventType, "success").Inc()
	}

	return nil
}

// GetStats returns current outbox statistics for monitoring.
func (w *OutboxWorker) GetStats(ctx context.Context) (OutboxStats, error) {
	count, err := w.outbox.Count(ctx)
	if err != nil {
		return OutboxStats{}, err
	}

	return OutboxStats{
		PendingCount: count,
	}, nil
}

// updateGaugeMetrics updates gauge metrics (pending count, oldest event age).
func (w *OutboxWorker) updateGaugeMetrics(ctx context.Context) {
	if w.metrics == nil {
		return
	}

	count, oldest, err := w.outbox.Stats(ctx)
	if err != nil {
		w.logger.WarnContext(ctx, "failed to get outbox stats for metrics",
			slog.String("error", err.Error()),
		)
		return
	}

	// Update pending count
	w.metrics.EventsPending.Set(float64(count))

	// Update oldest event age (0 if no events)
	if !oldest.IsZero() && count > 0 {
		age := time.Since(oldest).Seconds()
		w.metrics.OldestEventAge.Set(age)
	} else {
		w.metrics.OldestEventAge.Set(0)
	}
}

// OutboxStats contains outbox statistics for monitoring.
type OutboxStats struct {
	PendingCount int64
}

// outboxEvent implements event.DomainEvent for events reconstructed from the outbox.
type outboxEvent struct {
	eventType     string
	aggregateID   string
	aggregateType string
	occurredAt    time.Time
	version       int
	metadata      event.Metadata
	payload       []byte
}

func (e *outboxEvent) EventType() string        { return e.eventType }
func (e *outboxEvent) AggregateID() string      { return e.aggregateID }
func (e *outboxEvent) AggregateType() string    { return e.aggregateType }
func (e *outboxEvent) OccurredAt() time.Time    { return e.occurredAt }
func (e *outboxEvent) Version() int             { return e.version }
func (e *outboxEvent) Metadata() event.Metadata { return e.metadata }

// Payload returns the raw JSON payload of the event.
func (e *outboxEvent) Payload() json.RawMessage { return e.payload }

// ProcessOnce processes a single batch of events (useful for testing).
func (w *OutboxWorker) ProcessOnce(ctx context.Context) error {
	return w.processBatch(ctx)
}
