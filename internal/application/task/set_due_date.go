package task

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/task"
)

const minYear = 2020

// SetDueDateUseCase handles ustanovku deadline tasks
type SetDueDateUseCase struct {
	baseExecutor *BaseExecutor
}

// NewSetDueDateUseCase creates New use case for setting deadline
func NewSetDueDateUseCase(eventStore appcore.EventStore) *SetDueDateUseCase {
	return &SetDueDateUseCase{
		baseExecutor: NewBaseExecutor(eventStore),
	}
}

// Execute sets deadline tasks
func (uc *SetDueDateUseCase) Execute(ctx context.Context, cmd SetDueDateCommand) (TaskResult, error) {
	// validation commands
	if err := uc.validate(cmd); err != nil {
		return TaskResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// performing operatsii via bazovyy executor
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
