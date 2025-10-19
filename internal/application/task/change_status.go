package task

import (
	"context"
	"errors"
	"fmt"

	"github.com/lllypuk/teams-up/internal/application/shared"
	"github.com/lllypuk/teams-up/internal/domain/errs"
	"github.com/lllypuk/teams-up/internal/domain/task"
)

// ChangeStatusUseCase обрабатывает изменение статуса задачи
type ChangeStatusUseCase struct {
	eventStore shared.EventStore
}

// NewChangeStatusUseCase создает новый use case для изменения статуса
func NewChangeStatusUseCase(eventStore shared.EventStore) *ChangeStatusUseCase {
	return &ChangeStatusUseCase{
		eventStore: eventStore,
	}
}

// Execute изменяет статус задачи
func (uc *ChangeStatusUseCase) Execute(ctx context.Context, cmd ChangeStatusCommand) (TaskResult, error) {
	// 1. Валидация команды
	if err := uc.validate(cmd); err != nil {
		return TaskResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// 2. Загрузка событий из Event Store
	events, err := uc.eventStore.LoadEvents(ctx, cmd.TaskID.String())
	if err != nil {
		if errors.Is(err, shared.ErrAggregateNotFound) {
			return TaskResult{}, ErrTaskNotFound
		}
		return TaskResult{}, fmt.Errorf("failed to load events: %w", err)
	}

	if len(events) == 0 {
		return TaskResult{}, ErrTaskNotFound
	}

	// 3. Восстановление агрегата из событий
	aggregate := task.NewTaskAggregate(cmd.TaskID)
	aggregate.ReplayEvents(events)

	// 4. Выполнение бизнес-операции
	err = aggregate.ChangeStatus(cmd.NewStatus, cmd.ChangedBy)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidTransition) {
			return TaskResult{}, ErrInvalidStatusTransition
		}
		return TaskResult{}, fmt.Errorf("failed to change status: %w", err)
	}

	// 5. Получение только новых событий
	newEvents := aggregate.UncommittedEvents()

	// Если новых событий нет (идемпотентность), возвращаем успех
	if len(newEvents) == 0 {
		return TaskResult{
			TaskID:  cmd.TaskID,
			Version: len(events),
			Events:  newEvents,
			Success: true,
			Message: "Status unchanged (idempotent operation)",
		}, nil
	}

	// 6. Сохранение новых событий
	expectedVersion := len(events)
	if saveErr := uc.eventStore.SaveEvents(ctx, cmd.TaskID.String(), newEvents, expectedVersion); saveErr != nil {
		if errors.Is(saveErr, shared.ErrConcurrencyConflict) {
			return TaskResult{}, ErrConcurrentUpdate
		}
		return TaskResult{}, fmt.Errorf("failed to save events: %w", saveErr)
	}

	// 7. Возврат результата
	return NewSuccessResult(cmd.TaskID, expectedVersion+len(newEvents), newEvents), nil
}

// validate проверяет корректность команды
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

// isValidStatus проверяет валидность статуса
func isValidStatus(status task.Status) bool {
	return status == task.StatusBacklog ||
		status == task.StatusToDo ||
		status == task.StatusInProgress ||
		status == task.StatusInReview ||
		status == task.StatusDone ||
		status == task.StatusCancelled
}
