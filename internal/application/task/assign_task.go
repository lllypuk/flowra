package task

import (
	"context"
	"errors"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/errs"
)

// AssignTaskUseCase handles task assignee assignment
type AssignTaskUseCase struct {
	taskRepo       CommandRepository
	userRepository appcore.UserRepository
}

// NewAssignTaskUseCase creates a new use case for assigning tasks
func NewAssignTaskUseCase(
	taskRepo CommandRepository,
	userRepository appcore.UserRepository,
) *AssignTaskUseCase {
	return &AssignTaskUseCase{
		taskRepo:       taskRepo,
		userRepository: userRepository,
	}
}

// Execute assigns a task to a user
func (uc *AssignTaskUseCase) Execute(ctx context.Context, cmd AssignTaskCommand) (TaskResult, error) {
	// 1. Validate command
	if err := uc.validate(ctx, cmd); err != nil {
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
	err = aggregate.Assign(cmd.AssigneeID, cmd.AssignedBy)
	if err != nil {
		return TaskResult{}, fmt.Errorf("failed to assign task: %w", err)
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
			Message: "Assignee unchanged (idempotent operation)",
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
func (uc *AssignTaskUseCase) validate(ctx context.Context, cmd AssignTaskCommand) error {
	if cmd.TaskID.IsZero() {
		return ErrInvalidTaskID
	}

	if cmd.AssignedBy.IsZero() {
		return ErrInvalidUserID
	}

	// if AssigneeID ukazan (not snyatie assignee), checking suschestvovanie user
	if cmd.AssigneeID != nil && !cmd.AssigneeID.IsZero() {
		exists, err := uc.userRepository.Exists(ctx, *cmd.AssigneeID)
		if err != nil {
			return fmt.Errorf("failed to check user existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("%w: user %s", ErrUserNotFound, cmd.AssigneeID)
		}
	}

	return nil
}
