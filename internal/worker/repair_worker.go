package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/repair"
)

// Default repair worker configuration values.
const (
	defaultRepairPollInterval = 30 * time.Second
	defaultRepairBatchSize    = 10
	defaultRepairMaxRetries   = 3
)

// RepairWorkerConfig contains configuration for the repair worker.
type RepairWorkerConfig struct {
	// PollInterval is the time between polling the repair queue.
	PollInterval time.Duration

	// BatchSize is the maximum number of tasks to process in each poll cycle.
	BatchSize int

	// MaxRetries is the maximum number of retry attempts for failed repairs.
	MaxRetries int

	// Enabled determines if the worker should run.
	Enabled bool
}

// DefaultRepairWorkerConfig returns sensible default configuration.
func DefaultRepairWorkerConfig() RepairWorkerConfig {
	return RepairWorkerConfig{
		PollInterval: defaultRepairPollInterval,
		BatchSize:    defaultRepairBatchSize,
		MaxRetries:   defaultRepairMaxRetries,
		Enabled:      true,
	}
}

// RepairWorker processes repair tasks from the queue and rebuilds read models.
type RepairWorker struct {
	repairQueue   repair.Queue
	chatProjector appcore.ReadModelProjector
	taskProjector appcore.ReadModelProjector
	logger        *slog.Logger
	config        RepairWorkerConfig
}

// NewRepairWorker creates a new repair worker.
func NewRepairWorker(
	repairQueue repair.Queue,
	chatProjector appcore.ReadModelProjector,
	taskProjector appcore.ReadModelProjector,
	logger *slog.Logger,
	config RepairWorkerConfig,
) *RepairWorker {
	if logger == nil {
		logger = slog.Default()
	}

	return &RepairWorker{
		repairQueue:   repairQueue,
		chatProjector: chatProjector,
		taskProjector: taskProjector,
		logger:        logger,
		config:        config,
	}
}

// Start starts the repair worker.
func (w *RepairWorker) Start(ctx context.Context) error {
	if !w.config.Enabled {
		w.logger.InfoContext(ctx, "repair worker disabled")
		return nil
	}

	w.logger.InfoContext(ctx, "starting repair worker",
		slog.Duration("poll_interval", w.config.PollInterval),
		slog.Int("batch_size", w.config.BatchSize),
		slog.Int("max_retries", w.config.MaxRetries),
	)

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	// Process immediately on start
	w.processBatch(ctx)

	for {
		select {
		case <-ctx.Done():
			w.logger.InfoContext(ctx, "repair worker stopped")
			return ctx.Err()
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

// processBatch processes a batch of repair tasks.
func (w *RepairWorker) processBatch(ctx context.Context) {
	tasks, err := w.repairQueue.Poll(ctx, w.config.BatchSize)
	if err != nil {
		w.logger.ErrorContext(ctx, "failed to poll repair queue",
			slog.String("error", err.Error()),
		)
		return
	}

	if len(tasks) == 0 {
		return
	}

	w.logger.InfoContext(ctx, "processing repair tasks",
		slog.Int("count", len(tasks)),
	)

	for _, task := range tasks {
		if processErr := w.processTask(ctx, task); processErr != nil {
			w.logger.ErrorContext(ctx, "failed to process repair task",
				slog.String("task_id", task.ID),
				slog.String("aggregate_id", task.AggregateID),
				slog.String("aggregate_type", task.AggregateType),
				slog.String("error", processErr.Error()),
			)

			// Check if max retries exceeded
			if task.RetryCount >= w.config.MaxRetries {
				w.logger.WarnContext(ctx, "max retries exceeded, marking task as failed",
					slog.String("task_id", task.ID),
					slog.Int("retry_count", task.RetryCount),
				)
				if markErr := w.repairQueue.MarkFailed(ctx, task.ID, processErr); markErr != nil {
					w.logger.ErrorContext(ctx, "failed to mark task as failed",
						slog.String("task_id", task.ID),
						slog.String("error", markErr.Error()),
					)
				}
			} else {
				// Task will remain in "processing" state and will be picked up again
				// after timeout or on next restart
				w.logger.InfoContext(ctx, "task will be retried",
					slog.String("task_id", task.ID),
					slog.Int("retry_count", task.RetryCount),
					slog.Int("max_retries", w.config.MaxRetries),
				)
			}
			continue
		}

		// Mark task as completed
		if completeErr := w.repairQueue.MarkCompleted(ctx, task.ID); completeErr != nil {
			w.logger.ErrorContext(ctx, "failed to mark task as completed",
				slog.String("task_id", task.ID),
				slog.String("error", completeErr.Error()),
			)
		}
	}
}

// processTask processes a single repair task.
func (w *RepairWorker) processTask(ctx context.Context, task repair.Task) error {
	w.logger.InfoContext(ctx, "processing repair task",
		slog.String("task_id", task.ID),
		slog.String("aggregate_id", task.AggregateID),
		slog.String("aggregate_type", task.AggregateType),
		slog.String("task_type", string(task.TaskType)),
	)

	switch task.TaskType {
	case repair.TaskTypeReadModelSync:
		return w.processReadModelSync(ctx, task)
	default:
		return fmt.Errorf("unknown task type: %s", task.TaskType)
	}
}

// processReadModelSync processes a read model synchronization task.
func (w *RepairWorker) processReadModelSync(ctx context.Context, task repair.Task) error {
	aggregateID, err := uuid.ParseUUID(task.AggregateID)
	if err != nil {
		return fmt.Errorf("invalid aggregate ID: %w", err)
	}

	var projector appcore.ReadModelProjector
	switch task.AggregateType {
	case "chat":
		projector = w.chatProjector
	case "task":
		projector = w.taskProjector
	default:
		return fmt.Errorf("unsupported aggregate type: %s", task.AggregateType)
	}

	// Rebuild the read model
	if rebuildErr := projector.RebuildOne(ctx, aggregateID); rebuildErr != nil {
		return fmt.Errorf("failed to rebuild read model: %w", rebuildErr)
	}

	w.logger.InfoContext(ctx, "successfully rebuilt read model",
		slog.String("aggregate_id", task.AggregateID),
		slog.String("aggregate_type", task.AggregateType),
	)

	return nil
}

// GetStats returns repair queue statistics.
func (w *RepairWorker) GetStats(ctx context.Context) (*repair.QueueStats, error) {
	return w.repairQueue.GetStats(ctx)
}
