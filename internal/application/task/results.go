package task

import (
	"github.com/flowra/flowra/internal/domain/event"
	"github.com/flowra/flowra/internal/domain/uuid"
)

// TaskResult — результат выполнения use case для Task
// Имя явно указывает на принадлежность к task для избежания путаницы с другими результатами
//
//nolint:revive // осознанное решение для ясности кода
type TaskResult struct {
	// TaskID идентификатор задачи
	TaskID uuid.UUID

	// Version текущая версия агрегата после выполнения операции
	Version int

	// Events события, сгенерированные в результате выполнения операции
	Events []event.DomainEvent

	// Success флаг успешного выполнения
	Success bool

	// Message дополнительное сообщение (для ошибок или предупреждений)
	Message string
}

// NewSuccessResult создает результат успешного выполнения
func NewSuccessResult(taskID uuid.UUID, version int, events []event.DomainEvent) TaskResult {
	return TaskResult{
		TaskID:  taskID,
		Version: version,
		Events:  events,
		Success: true,
	}
}

// NewFailureResult создает результат неудачного выполнения
func NewFailureResult(taskID uuid.UUID, message string) TaskResult {
	return TaskResult{
		TaskID:  taskID,
		Success: false,
		Message: message,
	}
}

// IsSuccess возвращает true если операция выполнена успешно
func (r TaskResult) IsSuccess() bool {
	return r.Success
}

// IsFailure возвращает true если операция завершилась с ошибкой
func (r TaskResult) IsFailure() bool {
	return !r.Success
}

// EventCount возвращает количество сгенерированных событий
func (r TaskResult) EventCount() int {
	return len(r.Events)
}
