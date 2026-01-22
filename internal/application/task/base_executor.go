package task

import (
	"context"
	"errors"
	"fmt"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// AggregateCommand represents a common interface for commands working with task aggregate
type AggregateCommand interface {
	GetTaskID() uuid.UUID
}

// AggregateOperation is a function for performing business operations on the aggregate
type AggregateOperation func(aggregate *task.Aggregate) error

// BaseExecutor contains common logic for executing commands with Event Sourcing
type BaseExecutor struct {
	taskRepo CommandRepository
}

// NewBaseExecutor creates a new base executor
func NewBaseExecutor(taskRepo CommandRepository) *BaseExecutor {
	return &BaseExecutor{
		taskRepo: taskRepo,
	}
}

// Execute performs common Event Sourcing logic for task commands
// Parameters:
// - ctx: execution context
// - taskID: task identifier
// - operation: business operation to perform on the aggregate
// - idempotentMessage: message for when the operation is idempotent
func (e *BaseExecutor) Execute(
	ctx context.Context,
	taskID uuid.UUID,
	operation AggregateOperation,
	idempotentMessage string,
) (TaskResult, error) {
	// 1. Load aggregate from repository
	aggregate, err := e.taskRepo.Load(ctx, taskID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return TaskResult{}, ErrTaskNotFound
		}
		return TaskResult{}, fmt.Errorf("failed to load task: %w", err)
	}

	// Store current version before operation
	versionBefore := aggregate.Version()

	// 2. Perform business operation
	if opErr := operation(aggregate); opErr != nil {
		return TaskResult{}, opErr
	}

	// 3. Get new events
	newEvents := aggregate.UncommittedEvents()

	// If no new events (idempotent), return success
	if len(newEvents) == 0 {
		return TaskResult{
			TaskID:  taskID,
			Version: versionBefore,
			Events:  newEvents,
			Success: true,
			Message: idempotentMessage,
		}, nil
	}

	// 4. Save via repository (handles EventStore + ReadModel)
	if saveErr := e.taskRepo.Save(ctx, aggregate); saveErr != nil {
		if errors.Is(saveErr, errs.ErrConcurrentModification) {
			return TaskResult{}, ErrConcurrentUpdate
		}
		return TaskResult{}, fmt.Errorf("failed to save task: %w", saveErr)
	}

	// 5. Return result
	return NewSuccessResult(taskID, aggregate.Version(), newEvents), nil
}
