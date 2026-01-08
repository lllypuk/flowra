package appcore

import "context"

// UseCase is the base interface for all use cases.
// TCommand - the command type (input data)
// TResult - the result type (output data)
type UseCase[TCommand any, TResult any] interface {
	// Execute executes the use case with the given command
	Execute(ctx context.Context, cmd TCommand) (TResult, error)
}

// Command is a marker interface for commands (modify state)
type Command interface {
	CommandName() string
}

// Query is a marker interface for queries (read-only)
type Query interface {
	QueryName() string
}

// Validator is an interface for command validation
type Validator[T any] interface {
	// Validate checks the validity of the command
	Validate(cmd T) error
}

// CommandHandler is an interface for command handlers.
// Combines UseCase and Validator
type CommandHandler[TCommand any, TResult any] interface {
	UseCase[TCommand, TResult]
	Validator[TCommand]
}

// Result is the base result structure
type Result[T any] struct {
	Value   T
	Version int
	Error   error
}

// IsSuccess checks if the operation completed successfully
func (r Result[T]) IsSuccess() bool {
	return r.Error == nil
}

// IsFailure checks if the operation completed with an error
func (r Result[T]) IsFailure() bool {
	return r.Error != nil
}

// EventSourcedResult is a result for event-sourced operations
type EventSourcedResult[T any] struct {
	Result[T]

	Events []any // domain events
}

// UnitOfWork is an interface for transaction management
type UnitOfWork interface {
	Begin(ctx context.Context) error
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}
