package task

import (
	"context"
	"errors"
	"fmt"

	"github.com/lllypuk/teams-up/internal/domain/task"
	"github.com/lllypuk/teams-up/internal/infrastructure/eventstore"
	"github.com/lllypuk/teams-up/internal/usecase/shared"
)

// AssignTaskUseCase обрабатывает назначение исполнителя задачи
type AssignTaskUseCase struct {
	eventStore     eventstore.EventStore
	userRepository shared.UserRepository
}

// NewAssignTaskUseCase создает новый use case для назначения исполнителя
func NewAssignTaskUseCase(
	eventStore eventstore.EventStore,
	userRepository shared.UserRepository,
) *AssignTaskUseCase {
	return &AssignTaskUseCase{
		eventStore:     eventStore,
		userRepository: userRepository,
	}
}

// Execute назначает исполнителя задаче
func (uc *AssignTaskUseCase) Execute(ctx context.Context, cmd AssignTaskCommand) (TaskResult, error) {
	// 1. Валидация команды
	if err := uc.validate(ctx, cmd); err != nil {
		return TaskResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// 2. Загрузка событий из Event Store
	events, err := uc.eventStore.LoadEvents(ctx, cmd.TaskID.String())
	if err != nil {
		if errors.Is(err, eventstore.ErrAggregateNotFound) {
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
	err = aggregate.Assign(cmd.AssigneeID, cmd.AssignedBy)
	if err != nil {
		return TaskResult{}, fmt.Errorf("failed to assign task: %w", err)
	}

	// 5. Получение новых событий
	newEvents := aggregate.UncommittedEvents()

	// Если новых событий нет (идемпотентность), возвращаем успех
	if len(newEvents) == 0 {
		return TaskResult{
			TaskID:  cmd.TaskID,
			Version: len(events),
			Events:  newEvents,
			Success: true,
			Message: "Assignee unchanged (idempotent operation)",
		}, nil
	}

	// 6. Сохранение новых событий
	expectedVersion := len(events)
	if saveErr := uc.eventStore.SaveEvents(ctx, cmd.TaskID.String(), newEvents, expectedVersion); saveErr != nil {
		if errors.Is(saveErr, eventstore.ErrConcurrencyConflict) {
			return TaskResult{}, ErrConcurrentUpdate
		}
		return TaskResult{}, fmt.Errorf("failed to save events: %w", saveErr)
	}

	// 7. Возврат результата
	return NewSuccessResult(cmd.TaskID, expectedVersion+len(newEvents), newEvents), nil
}

// validate проверяет корректность команды
func (uc *AssignTaskUseCase) validate(ctx context.Context, cmd AssignTaskCommand) error {
	if cmd.TaskID.IsZero() {
		return ErrInvalidTaskID
	}

	if cmd.AssignedBy.IsZero() {
		return ErrInvalidUserID
	}

	// Если AssigneeID указан (не снятие assignee), проверяем существование пользователя
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
