package task

import (
	"context"
	"errors"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/task"
)

// AssignTaskUseCase handles assignment ispolnitelya tasks
type AssignTaskUseCase struct {
	eventStore     appcore.EventStore
	userRepository appcore.UserRepository
}

// NewAssignTaskUseCase creates New use case for assigning ispolnitelya
func NewAssignTaskUseCase(
	eventStore appcore.EventStore,
	userRepository appcore.UserRepository,
) *AssignTaskUseCase {
	return &AssignTaskUseCase{
		eventStore:     eventStore,
		userRepository: userRepository,
	}
}

// Execute value ispolnitelya zadache
func (uc *AssignTaskUseCase) Execute(ctx context.Context, cmd AssignTaskCommand) (TaskResult, error) {
	// 1. validation commands
	if err := uc.validate(ctx, cmd); err != nil {
		return TaskResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// 2. Loading events from Event Store
	events, err := uc.eventStore.LoadEvents(ctx, cmd.TaskID.String())
	if err != nil {
		if errors.Is(err, appcore.ErrAggregateNotFound) {
			return TaskResult{}, ErrTaskNotFound
		}
		return TaskResult{}, fmt.Errorf("failed to load events: %w", err)
	}

	if len(events) == 0 {
		return TaskResult{}, ErrTaskNotFound
	}

	// 3. Restoration aggregate from events
	aggregate := task.NewTaskAggregate(cmd.TaskID)
	aggregate.ReplayEvents(events)

	// 4. performing biznes-operatsii
	err = aggregate.Assign(cmd.AssigneeID, cmd.AssignedBy)
	if err != nil {
		return TaskResult{}, fmt.Errorf("failed to assign task: %w", err)
	}

	// 5. retrieval New events
	newEvents := aggregate.UncommittedEvents()

	// if no new events (idempotent), return success
	if len(newEvents) == 0 {
		return TaskResult{
			TaskID:  cmd.TaskID,
			Version: len(events),
			Events:  newEvents,
			Success: true,
			Message: "Assignee unchanged (idempotent operation)",
		}, nil
	}

	// 6. storage New events
	expectedVersion := len(events)
	if saveErr := uc.eventStore.SaveEvents(ctx, cmd.TaskID.String(), newEvents, expectedVersion); saveErr != nil {
		if errors.Is(saveErr, appcore.ErrConcurrencyConflict) {
			return TaskResult{}, ErrConcurrentUpdate
		}
		return TaskResult{}, fmt.Errorf("failed to save events: %w", saveErr)
	}

	// 7. vozvrat result
	return NewSuccessResult(cmd.TaskID, expectedVersion+len(newEvents), newEvents), nil
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
