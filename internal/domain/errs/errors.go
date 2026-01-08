package errs

import "errors"

var (
	// ErrNotFound is returned when a resource is not found
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists is returned when a resource already exists
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrInvalidInput is returned when input data is invalid
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized is returned when access is not authorized
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when an action is forbidden
	ErrForbidden = errors.New("forbidden")

	// ErrConcurrentModification is returned when a version conflict occurs
	ErrConcurrentModification = errors.New("concurrent modification detected")

	// ErrInvalidState is returned when aggregate state is invalid
	ErrInvalidState = errors.New("invalid aggregate state")

	// ErrInvalidTransition is returned when a state transition is invalid
	ErrInvalidTransition = errors.New("invalid state transition")
)
