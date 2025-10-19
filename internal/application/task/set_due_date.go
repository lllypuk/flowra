package task

import (
	"context"
	"fmt"

	"github.com/lllypuk/teams-up/internal/application/shared"
	"github.com/lllypuk/teams-up/internal/domain/task"
)

const minYear = 2020

// SetDueDateUseCase обрабатывает установку дедлайна задачи
type SetDueDateUseCase struct {
	baseExecutor *BaseExecutor
}

// NewSetDueDateUseCase создает новый use case для установки дедлайна
func NewSetDueDateUseCase(eventStore shared.EventStore) *SetDueDateUseCase {
	return &SetDueDateUseCase{
		baseExecutor: NewBaseExecutor(eventStore),
	}
}

// Execute устанавливает дедлайн задачи
func (uc *SetDueDateUseCase) Execute(ctx context.Context, cmd SetDueDateCommand) (TaskResult, error) {
	// Валидация команды
	if err := uc.validate(cmd); err != nil {
		return TaskResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// Выполнение операции через базовый executor
	return uc.baseExecutor.Execute(
		ctx,
		cmd.TaskID,
		func(aggregate *task.Aggregate) error {
			return aggregate.SetDueDate(cmd.DueDate, cmd.ChangedBy)
		},
		"Due date unchanged (idempotent operation)",
	)
}

// validate проверяет корректность команды
func (uc *SetDueDateUseCase) validate(cmd SetDueDateCommand) error {
	if cmd.TaskID.IsZero() {
		return ErrInvalidTaskID
	}

	// DueDate может быть nil (снятие дедлайна) — это валидно

	// Sanity check: дата не должна быть слишком далеко в прошлом
	if cmd.DueDate != nil && cmd.DueDate.Year() < minYear {
		return fmt.Errorf("%w: date is too far in the past", ErrInvalidDate)
	}

	if cmd.ChangedBy.IsZero() {
		return ErrInvalidUserID
	}

	return nil
}
