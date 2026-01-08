package task

import (
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// TaskResult — result выполнения use case for Task
// Имя явно указывает on принадлежность to task for избежания путаницы с другими результатами
//
//nolint:revive // осознанное решение for ясности кода
type TaskResult struct {
	// TaskID identifierатор tasks
	TaskID uuid.UUID

	// Version текущая версия aggregate after выполнения операции
	Version int

	// Events event, сгенерированные in результате выполнения операции
	Events []event.DomainEvent

	// Success флаг успешного выполнения
	Success bool

	// Message дополнительное message (for errors or предупреждений)
	Message string
}

// NewSuccessResult creates result успешного выполнения
func NewSuccessResult(taskID uuid.UUID, version int, events []event.DomainEvent) TaskResult {
	return TaskResult{
		TaskID:  taskID,
		Version: version,
		Events:  events,
		Success: true,
	}
}

// NewFailureResult creates result неудачного выполнения
func NewFailureResult(taskID uuid.UUID, message string) TaskResult {
	return TaskResult{
		TaskID:  taskID,
		Success: false,
		Message: message,
	}
}

// IsSuccess returns true if операция выполнена successfully
func (r TaskResult) IsSuccess() bool {
	return r.Success
}

// IsFailure returns true if операция завершилась с ошибкой
func (r TaskResult) IsFailure() bool {
	return !r.Success
}

// EventCount returns count сгенерированных events
func (r TaskResult) EventCount() int {
	return len(r.Events)
}
