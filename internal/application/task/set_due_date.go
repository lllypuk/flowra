package task

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/domain/task"
)

const minYear = 2020

// SetDueDateUseCase handles setting task due date
type SetDueDateUseCase struct {
	baseExecutor *BaseExecutor
}

// NewSetDueDateUseCase creates a new use case for setting due date
func NewSetDueDateUseCase(taskRepo CommandRepository) *SetDueDateUseCase {
	return &SetDueDateUseCase{
		baseExecutor: NewBaseExecutor(taskRepo),
	}
}

// Execute sets task due date
func (uc *SetDueDateUseCase) Execute(ctx context.Context, cmd SetDueDateCommand) (TaskResult, error) {
	// Validate command
	if err := uc.validate(cmd); err != nil {
		return TaskResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// Perform operation via base executor
	return uc.baseExecutor.Execute(
		ctx,
		cmd.TaskID,
		func(aggregate *task.Aggregate) error {
			return aggregate.SetDueDate(cmd.DueDate, cmd.ChangedBy)
		},
		"Due date unchanged (idempotent operation)",
	)
}

// validate checks command correctness
func (uc *SetDueDateUseCase) validate(cmd SetDueDateCommand) error {
	if cmd.TaskID.IsZero() {
		return ErrInvalidTaskID
	}

	// DueDate mozhet byt nil (snyatie deadline) — it is valid

	// Sanity check: date not dolzhna byt slishkom daleko in proshlom
	if cmd.DueDate != nil && cmd.DueDate.Year() < minYear {
		return fmt.Errorf("%w: date is too far in the past", ErrInvalidDate)
	}

	if cmd.ChangedBy.IsZero() {
		return ErrInvalidUserID
	}

	return nil
}
