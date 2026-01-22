package task

import (
	"context"
	"errors"
	"fmt"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/task"
)

// ChangeStatusUseCase handles changing task status
type ChangeStatusUseCase struct {
	taskRepo CommandRepository
}

// NewChangeStatusUseCase creates a new use case for changing status
func NewChangeStatusUseCase(taskRepo CommandRepository) *ChangeStatusUseCase {
	return &ChangeStatusUseCase{
		taskRepo: taskRepo,
	}
}

// Execute changes task status
func (uc *ChangeStatusUseCase) Execute(ctx context.Context, cmd ChangeStatusCommand) (TaskResult, error) {
	// 1. Validate command
	if err := uc.validate(cmd); err != nil {
		return TaskResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// 2. Load aggregate from repository
	aggregate, err := uc.taskRepo.Load(ctx, cmd.TaskID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return TaskResult{}, ErrTaskNotFound
		}
		return TaskResult{}, fmt.Errorf("failed to load task: %w", err)
	}

	// Store current version before operation
	versionBefore := aggregate.Version()

	// 3. Perform business operation
	err = aggregate.ChangeStatus(cmd.NewStatus, cmd.ChangedBy)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidTransition) {
			return TaskResult{}, ErrInvalidStatusTransition
		}
		return TaskResult{}, fmt.Errorf("failed to change status: %w", err)
	}

	// 4. Get new events
	newEvents := aggregate.UncommittedEvents()

	// If no new events (idempotent), return success
	if len(newEvents) == 0 {
		return TaskResult{
			TaskID:  cmd.TaskID,
			Version: versionBefore,
			Events:  newEvents,
			Success: true,
			Message: "Status unchanged (idempotent operation)",
		}, nil
	}

	// 5. Save via repository (handles EventStore + ReadModel)
	if saveErr := uc.taskRepo.Save(ctx, aggregate); saveErr != nil {
		if errors.Is(saveErr, errs.ErrConcurrentModification) {
			return TaskResult{}, ErrConcurrentUpdate
		}
		return TaskResult{}, fmt.Errorf("failed to save task: %w", saveErr)
	}

	// 6. Return result
	return NewSuccessResult(cmd.TaskID, aggregate.Version(), newEvents), nil
}

// validate checks command correctness
func (uc *ChangeStatusUseCase) validate(cmd ChangeStatusCommand) error {
	if cmd.TaskID.IsZero() {
		return ErrInvalidTaskID
	}

	if cmd.NewStatus == "" {
		return ErrInvalidStatus
	}

	if !isValidStatus(cmd.NewStatus) {
		return fmt.Errorf("%w: must be valid status", ErrInvalidStatus)
	}

	if cmd.ChangedBy.IsZero() {
		return ErrInvalidUserID
	}

	return nil
}

// isValidStatus validates status
func isValidStatus(status task.Status) bool {
	return status == task.StatusBacklog ||
		status == task.StatusToDo ||
		status == task.StatusInProgress ||
		status == task.StatusInReview ||
		status == task.StatusDone ||
		status == task.StatusCancelled
}
