package appcore

import "context"

// UseCase — базовый интерфейс для всех use cases
// TCommand - тип команды (входные данные)
// TResult - тип результата (выходные данные)
type UseCase[TCommand any, TResult any] interface {
	// Execute выполняет use case с заданной командой
	Execute(ctx context.Context, cmd TCommand) (TResult, error)
}

// Command — маркер интерфейс для команд (изменяют состояние)
type Command interface {
	CommandName() string
}

// Query — маркер интерфейс для запросов (только чтение)
type Query interface {
	QueryName() string
}

// Validator — интерфейс для валидации команд
type Validator[T any] interface {
	// Validate проверяет валидность команды
	Validate(cmd T) error
}

// CommandHandler — интерфейс для обработчиков команд
// Объединяет UseCase и Validator
type CommandHandler[TCommand any, TResult any] interface {
	UseCase[TCommand, TResult]
	Validator[TCommand]
}

// Result — базовая структура результата
type Result[T any] struct {
	Value   T
	Version int
	Error   error
}

// IsSuccess проверяет, что операция завершилась успешно
func (r Result[T]) IsSuccess() bool {
	return r.Error == nil
}

// IsFailure проверяет, что операция завершилась с ошибкой
func (r Result[T]) IsFailure() bool {
	return r.Error != nil
}

// EventSourcedResult — результат для event-sourced операций
type EventSourcedResult[T any] struct {
	Result[T]

	Events []any // domain events
}

// UnitOfWork — интерфейс для транзакционности
type UnitOfWork interface {
	Begin(ctx context.Context) error
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}
