package task

import (
	"context"
	"fmt"
	"strings"

	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// CreateTaskUseCase handles creation of new tasks
type CreateTaskUseCase struct {
	taskRepo CommandRepository
}

// NewCreateTaskUseCase creates a new instance of CreateTaskUseCase
func NewCreateTaskUseCase(taskRepo CommandRepository) *CreateTaskUseCase {
	return &CreateTaskUseCase{
		taskRepo: taskRepo,
	}
}

// Execute creates a new task
func (uc *CreateTaskUseCase) Execute(ctx context.Context, cmd CreateTaskCommand) (TaskResult, error) {
	// 1. Validate command
	if err := uc.validate(cmd); err != nil {
		return TaskResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// 2. Apply default values
	cmd = uc.applyDefaults(cmd)

	// 3. Create new aggregate
	taskID := uuid.NewUUID()
	aggregate := task.NewTaskAggregate(taskID)

	// 4. Perform business operation
	if err := aggregate.Create(
		cmd.ChatID,
		cmd.Title,
		cmd.EntityType,
		cmd.Priority,
		cmd.AssigneeID,
		cmd.DueDate,
		cmd.CreatedBy,
	); err != nil {
		return TaskResult{}, fmt.Errorf("failed to create task: %w", err)
	}

	// 5. Get events before saving (for response)
	events := aggregate.UncommittedEvents()

	// 6. Save via repository (handles EventStore + ReadModel)
	if err := uc.taskRepo.Save(ctx, aggregate); err != nil {
		return TaskResult{}, fmt.Errorf("failed to save task: %w", err)
	}

	// 7. Return result
	return NewSuccessResult(taskID, aggregate.Version(), events), nil
}

// validate checks command correctness
func (uc *CreateTaskUseCase) validate(cmd CreateTaskCommand) error {
	// ChatID obyazatelen
	if cmd.ChatID.IsZero() {
		return ErrInvalidChatID
	}

	// Title obyazatelen and not empty
	if strings.TrimSpace(cmd.Title) == "" {
		return ErrEmptyTitle
	}

	// Title not dolzhen byt slishkom dlinnym
	const maxTitleLength = 500
	if len(cmd.Title) > maxTitleLength {
		return fmt.Errorf("%w: title exceeds %d characters", ErrInvalidTitle, maxTitleLength)
	}

	// EntityType dolzhen byt valid
	if !isValidEntityType(cmd.EntityType) {
		return fmt.Errorf("%w: must be task, bug, or epic", ErrInvalidEntityType)
	}

	// Priority dolzhen byt valid, if ukazan
	if !isValidPriority(cmd.Priority) {
		return fmt.Errorf("%w: must be Low, Medium, High, or Critical", ErrInvalidPriority)
	}

	// CreatedBy obyazatelen
	if cmd.CreatedBy.IsZero() {
		return ErrInvalidUserID
	}

	// DueDate not dolzhna byt in dalekom proshlom (sanity check)
	if cmd.DueDate != nil && cmd.DueDate.Year() < 2020 {
		return fmt.Errorf("%w: date is too far in the past", ErrInvalidDate)
	}

	return nil
}

// applyDefaults primenyaet values by default
func (uc *CreateTaskUseCase) applyDefaults(cmd CreateTaskCommand) CreateTaskCommand {
	// if EntityType not ukazan, stavim task
	if cmd.EntityType == "" {
		cmd.EntityType = task.TypeTask
	}

	// if Priority not ukazan, stavim Medium
	if cmd.Priority == "" {
		cmd.Priority = task.PriorityMedium
	}

	// Trim probely in Title
	cmd.Title = strings.TrimSpace(cmd.Title)

	return cmd
}

// isValidEntityType validates type entity
func isValidEntityType(entityType task.EntityType) bool {
	return entityType == task.TypeTask ||
		entityType == task.TypeBug ||
		entityType == task.TypeEpic ||
		entityType == ""
}

// isValidPriority validates priority
func isValidPriority(priority task.Priority) bool {
	return priority == task.PriorityLow ||
		priority == task.PriorityMedium ||
		priority == task.PriorityHigh ||
		priority == task.PriorityCritical ||
		priority == ""
}
