package task

import (
	"context"
	"errors"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// AggregateCommand represents общий interface for commands, workающих с агрегатом tasks
type AggregateCommand interface {
	GetTaskID() uuid.UUID
}

// AggregateOperation function for выполнения бизнес-операции on агрегате
type AggregateOperation func(aggregate *task.Aggregate) error

// BaseExecutor contains общую логику for выполнения commands с Event Sourcing
type BaseExecutor struct {
	eventStore appcore.EventStore
}

// NewBaseExecutor creates New базовый executor
func NewBaseExecutor(eventStore appcore.EventStore) *BaseExecutor {
	return &BaseExecutor{
		eventStore: eventStore,
	}
}

// Execute performs общую логику Event Sourcing for commands задач
// parameters:
// - ctx: конtext выполнения
// - taskID: identifierатор tasks
// - operation: бизнес-операция for выполнения on агрегате
// - idempotentMessage: message for случая, when операция идемпотентна
func (e *BaseExecutor) Execute(
	ctx context.Context,
	taskID uuid.UUID,
	operation AggregateOperation,
	idempotentMessage string,
) (TaskResult, error) {
	// 1. Loading events from Event Store
	events, err := e.eventStore.LoadEvents(ctx, taskID.String())
	if err != nil {
		if errors.Is(err, appcore.ErrAggregateNotFound) {
			return TaskResult{}, ErrTaskNotFound
		}
		return TaskResult{}, fmt.Errorf("failed to load events: %w", err)
	}

	if len(events) == 0 {
		return TaskResult{}, ErrTaskNotFound
	}

	// 2. Restoration aggregate from events
	aggregate := task.NewTaskAggregate(taskID)
	aggregate.ReplayEvents(events)

	// 3. performing бизнес-операции
	if opErr := operation(aggregate); opErr != nil {
		return TaskResult{}, opErr
	}

	// 4. retrieval only New events
	newEvents := aggregate.UncommittedEvents()

	// if New events no (идемпотентность), возвращаем success
	if len(newEvents) == 0 {
		return TaskResult{
			TaskID:  taskID,
			Version: len(events),
			Events:  newEvents,
			Success: true,
			Message: idempotentMessage,
		}, nil
	}

	// 5. storage New events
	expectedVersion := len(events)
	if saveErr := e.eventStore.SaveEvents(ctx, taskID.String(), newEvents, expectedVersion); saveErr != nil {
		if errors.Is(saveErr, appcore.ErrConcurrencyConflict) {
			return TaskResult{}, ErrConcurrentUpdate
		}
		return TaskResult{}, fmt.Errorf("failed to save events: %w", saveErr)
	}

	// 6. Возврат result
	return NewSuccessResult(taskID, expectedVersion+len(newEvents), newEvents), nil
}
