package task

import (
	"context"
	"fmt"
	"strings"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// CreateTaskUseCase handles creation новой tasks
type CreateTaskUseCase struct {
	eventStore appcore.EventStore
}

// NewCreateTaskUseCase creates New instance CreateTaskUseCase
func NewCreateTaskUseCase(eventStore appcore.EventStore) *CreateTaskUseCase {
	return &CreateTaskUseCase{
		eventStore: eventStore,
	}
}

// Execute creates New задачу
func (uc *CreateTaskUseCase) Execute(ctx context.Context, cmd CreateTaskCommand) (TaskResult, error) {
	// 1. validation commands
	if err := uc.validate(cmd); err != nil {
		return TaskResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// 2. Applying values by default
	cmd = uc.applyDefaults(cmd)

	// 3. creation нового aggregate
	taskID := uuid.NewUUID()
	aggregate := task.NewTaskAggregate(taskID)

	// 4. performing бизнес-операции
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

	// 5. retrieval events
	events := aggregate.UncommittedEvents()

	// 6. storage events in Event Store
	if err := uc.eventStore.SaveEvents(ctx, taskID.String(), events, 0); err != nil {
		return TaskResult{}, fmt.Errorf("failed to save events: %w", err)
	}

	// 7. Возврат result
	return NewSuccessResult(taskID, aggregate.Version(), events), nil
}

// validate checks command correctness
func (uc *CreateTaskUseCase) validate(cmd CreateTaskCommand) error {
	// ChatID обязателен
	if cmd.ChatID.IsZero() {
		return ErrInvalidChatID
	}

	// Title обязателен and not empty
	if strings.TrimSpace(cmd.Title) == "" {
		return ErrEmptyTitle
	}

	// Title not должен быть слишком длинным
	const maxTitleLength = 500
	if len(cmd.Title) > maxTitleLength {
		return fmt.Errorf("%w: title exceeds %d characters", ErrInvalidTitle, maxTitleLength)
	}

	// EntityType должен быть validным
	if !isValidEntityType(cmd.EntityType) {
		return fmt.Errorf("%w: must be task, bug, or epic", ErrInvalidEntityType)
	}

	// Priority должен быть validным, if указан
	if !isValidPriority(cmd.Priority) {
		return fmt.Errorf("%w: must be Low, Medium, High, or Critical", ErrInvalidPriority)
	}

	// CreatedBy обязателен
	if cmd.CreatedBy.IsZero() {
		return ErrInvalidUserID
	}

	// DueDate not должна быть in далеком прошлом (sanity check)
	if cmd.DueDate != nil && cmd.DueDate.Year() < 2020 {
		return fmt.Errorf("%w: date is too far in the past", ErrInvalidDate)
	}

	return nil
}

// applyDefaults применяет values by default
func (uc *CreateTaskUseCase) applyDefaults(cmd CreateTaskCommand) CreateTaskCommand {
	// if EntityType not указан, ставим task
	if cmd.EntityType == "" {
		cmd.EntityType = task.TypeTask
	}

	// if Priority not указан, ставим Medium
	if cmd.Priority == "" {
		cmd.Priority = task.PriorityMedium
	}

	// Trim пробелы in Title
	cmd.Title = strings.TrimSpace(cmd.Title)

	return cmd
}

// isValidEntityType validates type сущности
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
