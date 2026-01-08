package task

import (
	"context"
	"fmt"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/task"
)

// ChangePriorityUseCase handles change priority tasks
type ChangePriorityUseCase struct {
	baseExecutor *BaseExecutor
}

// NewChangePriorityUseCase creates New use case for changing priority
func NewChangePriorityUseCase(eventStore appcore.EventStore) *ChangePriorityUseCase {
	return &ChangePriorityUseCase{
		baseExecutor: NewBaseExecutor(eventStore),
	}
}

// Execute izmenyaet prioritet tasks
func (uc *ChangePriorityUseCase) Execute(ctx context.Context, cmd ChangePriorityCommand) (TaskResult, error) {
	// validation commands
	if err := uc.validate(cmd); err != nil {
		return TaskResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// performing operatsii via bazovyy executor
	return uc.baseExecutor.Execute(
		ctx,
		cmd.TaskID,
		func(aggregate *task.Aggregate) error {
			return aggregate.ChangePriority(cmd.Priority, cmd.ChangedBy)
		},
		"Priority unchanged (idempotent operation)",
	)
}

// validate checks command correctness
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
