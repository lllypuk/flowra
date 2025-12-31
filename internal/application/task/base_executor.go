package task

import (
	"context"
	"errors"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// AggregateCommand представляет общий интерфейс для команд, работающих с агрегатом задачи
type AggregateCommand interface {
	GetTaskID() uuid.UUID
}

// AggregateOperation функция для выполнения бизнес-операции на агрегате
type AggregateOperation func(aggregate *task.Aggregate) error

// BaseExecutor содержит общую логику для выполнения команд с Event Sourcing
type BaseExecutor struct {
	eventStore appcore.EventStore
}

// NewBaseExecutor создает новый базовый executor
func NewBaseExecutor(eventStore appcore.EventStore) *BaseExecutor {
	return &BaseExecutor{
		eventStore: eventStore,
	}
}

// Execute выполняет общую логику Event Sourcing для команд задач
// Параметры:
// - ctx: контекст выполнения
// - taskID: идентификатор задачи
// - operation: бизнес-операция для выполнения на агрегате
// - idempotentMessage: сообщение для случая, когда операция идемпотентна
func (e *BaseExecutor) Execute(
	ctx context.Context,
	taskID uuid.UUID,
	operation AggregateOperation,
	idempotentMessage string,
) (TaskResult, error) {
	// 1. Загрузка событий из Event Store
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

	// 2. Восстановление агрегата из событий
	aggregate := task.NewTaskAggregate(taskID)
	aggregate.ReplayEvents(events)

	// 3. Выполнение бизнес-операции
	if opErr := operation(aggregate); opErr != nil {
		return TaskResult{}, opErr
	}

	// 4. Получение только новых событий
	newEvents := aggregate.UncommittedEvents()

	// Если новых событий нет (идемпотентность), возвращаем успех
	if len(newEvents) == 0 {
		return TaskResult{
			TaskID:  taskID,
			Version: len(events),
			Events:  newEvents,
			Success: true,
			Message: idempotentMessage,
		}, nil
	}

	// 5. Сохранение новых событий
	expectedVersion := len(events)
	if saveErr := e.eventStore.SaveEvents(ctx, taskID.String(), newEvents, expectedVersion); saveErr != nil {
		if errors.Is(saveErr, appcore.ErrConcurrencyConflict) {
			return TaskResult{}, ErrConcurrentUpdate
		}
		return TaskResult{}, fmt.Errorf("failed to save events: %w", saveErr)
	}

	// 6. Возврат результата
	return NewSuccessResult(taskID, expectedVersion+len(newEvents), newEvents), nil
}
