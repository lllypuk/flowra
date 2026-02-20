package task

import (
	"context"
	"errors"
	"fmt"

	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
)

// AddAttachmentUseCase handles adding attachments to tasks.
type AddAttachmentUseCase struct {
	base *BaseExecutor
}

// NewAddAttachmentUseCase creates a new AddAttachmentUseCase.
func NewAddAttachmentUseCase(taskRepo CommandRepository) *AddAttachmentUseCase {
	return &AddAttachmentUseCase{base: NewBaseExecutor(taskRepo)}
}

// Execute adds an attachment to a task.
func (uc *AddAttachmentUseCase) Execute(ctx context.Context, cmd AddAttachmentCommand) (TaskResult, error) {
	if err := uc.validate(cmd); err != nil {
		return TaskResult{}, fmt.Errorf("validation failed: %w", err)
	}

	return uc.base.Execute(ctx, cmd.TaskID, func(agg *taskdomain.Aggregate) error {
		return agg.AddAttachment(cmd.FileID, cmd.FileName, cmd.FileSize, cmd.MimeType, cmd.AddedBy)
	}, "attachment already exists")
}

func (uc *AddAttachmentUseCase) validate(cmd AddAttachmentCommand) error {
	if cmd.TaskID.IsZero() {
		return ErrInvalidTaskID
	}
	if cmd.FileID.IsZero() {
		return errors.New("file_id is required")
	}
	if cmd.FileName == "" {
		return errors.New("file_name is required")
	}
	if cmd.FileSize <= 0 {
		return errors.New("file_size must be positive")
	}
	if cmd.MimeType == "" {
		return errors.New("mime_type is required")
	}
	if cmd.AddedBy.IsZero() {
		return ErrInvalidUserID
	}
	return nil
}
