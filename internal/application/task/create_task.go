package task

import (
	"context"
	"fmt"
	"strings"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// CreateTaskUseCase обрабатывает создание новой задачи
type CreateTaskUseCase struct {
	eventStore appcore.EventStore
}

// NewCreateTaskUseCase создает новый экземпляр CreateTaskUseCase
func NewCreateTaskUseCase(eventStore appcore.EventStore) *CreateTaskUseCase {
	return &CreateTaskUseCase{
		eventStore: eventStore,
	}
}

// Execute создает новую задачу
func (uc *CreateTaskUseCase) Execute(ctx context.Context, cmd CreateTaskCommand) (TaskResult, error) {
	// 1. Валидация команды
	if err := uc.validate(cmd); err != nil {
		return TaskResult{}, fmt.Errorf("validation failed: %w", err)
	}

	// 2. Применение значений по умолчанию
	cmd = uc.applyDefaults(cmd)

	// 3. Создание нового агрегата
	taskID := uuid.NewUUID()
	aggregate := task.NewTaskAggregate(taskID)

	// 4. Выполнение бизнес-операции
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

	// 5. Получение событий
	events := aggregate.UncommittedEvents()

	// 6. Сохранение событий в Event Store
	if err := uc.eventStore.SaveEvents(ctx, taskID.String(), events, 0); err != nil {
		return TaskResult{}, fmt.Errorf("failed to save events: %w", err)
	}

	// 7. Возврат результата
	return NewSuccessResult(taskID, aggregate.Version(), events), nil
}

// validate проверяет корректность команды
func (uc *CreateTaskUseCase) validate(cmd CreateTaskCommand) error {
	// ChatID обязателен
	if cmd.ChatID.IsZero() {
		return ErrInvalidChatID
	}

	// Title обязателен и не пустой
	if strings.TrimSpace(cmd.Title) == "" {
		return ErrEmptyTitle
	}

	// Title не должен быть слишком длинным
	const maxTitleLength = 500
	if len(cmd.Title) > maxTitleLength {
		return fmt.Errorf("%w: title exceeds %d characters", ErrInvalidTitle, maxTitleLength)
	}

	// EntityType должен быть валидным
	if !isValidEntityType(cmd.EntityType) {
		return fmt.Errorf("%w: must be task, bug, or epic", ErrInvalidEntityType)
	}

	// Priority должен быть валидным, если указан
	if !isValidPriority(cmd.Priority) {
		return fmt.Errorf("%w: must be Low, Medium, High, or Critical", ErrInvalidPriority)
	}

	// CreatedBy обязателен
	if cmd.CreatedBy.IsZero() {
		return ErrInvalidUserID
	}

	// DueDate не должна быть в далеком прошлом (sanity check)
	if cmd.DueDate != nil && cmd.DueDate.Year() < 2020 {
		return fmt.Errorf("%w: date is too far in the past", ErrInvalidDate)
	}

	return nil
}

// applyDefaults применяет значения по умолчанию
func (uc *CreateTaskUseCase) applyDefaults(cmd CreateTaskCommand) CreateTaskCommand {
	// Если EntityType не указан, ставим task
	if cmd.EntityType == "" {
		cmd.EntityType = task.TypeTask
	}

	// Если Priority не указан, ставим Medium
	if cmd.Priority == "" {
		cmd.Priority = task.PriorityMedium
	}

	// Trim пробелы в Title
	cmd.Title = strings.TrimSpace(cmd.Title)

	return cmd
}

// isValidEntityType проверяет валидность типа сущности
func isValidEntityType(entityType task.EntityType) bool {
	return entityType == task.TypeTask ||
		entityType == task.TypeBug ||
		entityType == task.TypeEpic ||
		entityType == ""
}

// isValidPriority проверяет валидность приоритета
func isValidPriority(priority task.Priority) bool {
	return priority == task.PriorityLow ||
		priority == task.PriorityMedium ||
		priority == task.PriorityHigh ||
		priority == task.PriorityCritical ||
		priority == ""
}
