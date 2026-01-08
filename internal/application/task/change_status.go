package task

import (
	"context"
	"errors"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/task"
)

// ChangeStatusUseCase handles change status tasks
type ChangeStatusUseCase struct {
	eventStore appcore.EventStore
}

// NewChangeStatusUseCase creates New use case for changing status
func NewChangeStatusUseCase(eventStore appcore.EventStore) *ChangeStatusUseCase {
	return &ChangeStatusUseCase{
		eventStore: eventStore,
	}
}

// Execute изменяет status tasks
func (uc *ChangeStatusUseCase) Execute(ctx context.Context, cmd ChangeStatusCommand) (TaskResult, error) {
	// 1. validation commands
	if err := uc.validate(cmd); err != nil {
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

	// 4. performing бизнес-операции
	err = aggregate.ChangeStatus(cmd.NewStatus, cmd.ChangedBy)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidTransition) {
			return TaskResult{}, ErrInvalidStatusTransition
		}
		return TaskResult{}, fmt.Errorf("failed to change status: %w", err)
	}

	// 5. retrieval only New events
	newEvents := aggregate.UncommittedEvents()

	// if New events no (идемпотентность), возвращаем success
	if len(newEvents) == 0 {
		return TaskResult{
			TaskID:  cmd.TaskID,
			Version: len(events),
			Events:  newEvents,
			Success: true,
			Message: "Status unchanged (idempotent operation)",
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

	// 7. Возврат result
	return NewSuccessResult(cmd.TaskID, expectedVersion+len(newEvents), newEvents), nil
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
