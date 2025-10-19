package user

import "errors"

var (
	// ErrUsernameAlreadyExists возникает при попытке зарегистрировать пользователя с существующим username
	ErrUsernameAlreadyExists = errors.New("username already exists")

	// ErrEmailAlreadyExists возникает при попытке зарегистрировать пользователя с существующим email
	ErrEmailAlreadyExists = errors.New("email already exists")

	// ErrUserNotFound возникает когда пользователь не найден
	ErrUserNotFound = errors.New("user not found")

	// ErrNotSystemAdmin возникает когда операцию пытается выполнить не системный администратор
	ErrNotSystemAdmin = errors.New("only system administrators can perform this operation")

	// ErrInvalidEmail возникает при невалидном email
	ErrInvalidEmail = errors.New("invalid email format")

	// ErrInvalidUsername возникает при невалидном username
	ErrInvalidUsername = errors.New("invalid username format")
)
