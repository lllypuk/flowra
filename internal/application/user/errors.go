package user

import "errors"

var (
	// ErrUsernameAlreadyExists is returned when trying to register a user with an existing username
	ErrUsernameAlreadyExists = errors.New("username already exists")

	// ErrEmailAlreadyExists is returned when trying to register a user with an existing email
	ErrEmailAlreadyExists = errors.New("email already exists")

	// ErrUserNotFound is returned when user is not found
	ErrUserNotFound = errors.New("user not found")

	// ErrNotSystemAdmin is returned when a non-system administrator tries to perform a restricted operation
	ErrNotSystemAdmin = errors.New("only system administrators can perform this operation")

	// ErrInvalidEmail is returned when email format is invalid
	ErrInvalidEmail = errors.New("invalid email format")

	// ErrInvalidUsername is returned when username format is invalid
	ErrInvalidUsername = errors.New("invalid username format")
)
