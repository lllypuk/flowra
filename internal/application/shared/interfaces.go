package shared

import "context"

// UseCase — базовый интерфейс для всех use cases
// TCommand - тип команды (входные данные)
// TResult - тип результата (выходные данные)
type UseCase[TCommand any, TResult any] interface {
	// Execute выполняет use case с заданной командой
	Execute(ctx context.Context, cmd TCommand) (TResult, error)
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
