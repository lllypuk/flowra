package task

import (
	"context"
	"fmt"

	"github.com/lllypuk/teams-up/internal/domain/task"
	"github.com/lllypuk/teams-up/internal/infrastructure/eventstore"
)

// ChangePriorityUseCase обрабатывает изменение приоритета задачи
type ChangePriorityUseCase struct {
	baseExecutor *BaseExecutor
}

// NewChangePriorityUseCase создает новый use case для изменения приоритета
func NewChangePriorityUseCase(eventStore eventstore.EventStore) *ChangePriorityUseCase {
	return &ChangePriorityUseCase{
		baseExecutor: NewBaseExecutor(eventStore),
	}
}

// Execute изменяет приоритет задачи
func (uc *ChangePriorityUseCase) Execute(ctx context.Context, cmd ChangePriorityCommand) (TaskResult, error) {
	// Валидация команды
	if err := uc.validate(cmd); err != nil {
		return TaskResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// Выполнение операции через базовый executor
	return uc.baseExecutor.Execute(
		ctx,
		cmd.TaskID,
		func(aggregate *task.Aggregate) error {
			return aggregate.ChangePriority(cmd.Priority, cmd.ChangedBy)
		},
		"Priority unchanged (idempotent operation)",
	)
}

// validate проверяет корректность команды
func (uc *ChangePriorityUseCase) validate(cmd ChangePriorityCommand) error {
	if cmd.TaskID.IsZero() {
		return ErrInvalidTaskID
	}

	if cmd.Priority == "" {
		return ErrEmptyPriority
	}

	if !isValidPriority(cmd.Priority) {
		return fmt.Errorf("%w: must be Low, Medium, High, or Critical", ErrInvalidPriority)
	}

	if cmd.ChangedBy.IsZero() {
		return ErrInvalidUserID
	}

	return nil
}
