package task

import (
	"context"
	"errors"
	"fmt"

	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
)

// RemoveAttachmentUseCase handles removing attachments from tasks.
type RemoveAttachmentUseCase struct {
	base *BaseExecutor
}

// NewRemoveAttachmentUseCase creates a new RemoveAttachmentUseCase.
func NewRemoveAttachmentUseCase(taskRepo CommandRepository) *RemoveAttachmentUseCase {
	return &RemoveAttachmentUseCase{base: NewBaseExecutor(taskRepo)}
}

// Execute removes an attachment from a task.
func (uc *RemoveAttachmentUseCase) Execute(ctx context.Context, cmd RemoveAttachmentCommand) (TaskResult, error) {
	if err := uc.validate(cmd); err != nil {
		return TaskResult{}, fmt.Errorf("validation failed: %w", err)
	}

	return uc.base.Execute(ctx, cmd.TaskID, func(agg *taskdomain.Aggregate) error {
		return agg.RemoveAttachment(cmd.FileID, cmd.RemovedBy)
	}, "attachment not found")
}

func (uc *RemoveAttachmentUseCase) validate(cmd RemoveAttachmentCommand) error {
	if cmd.TaskID.IsZero() {
		return ErrInvalidTaskID
	}
	if cmd.FileID.IsZero() {
		return errors.New("file_id is required")
	}
	if cmd.RemovedBy.IsZero() {
		return ErrInvalidUserID
	}
	return nil
}
