package common

import "errors"

var (
	// ErrNotFound возвращается, когда ресурс не найден
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists возвращается, когда ресурс уже существует
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrInvalidInput возвращается при невалидных входных данных
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized возвращается при отсутствии прав доступа
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden возвращается при запрещенном действии
	ErrForbidden = errors.New("forbidden")

	// ErrConcurrentModification возвращается при конфликте версий
	ErrConcurrentModification = errors.New("concurrent modification detected")

	// ErrInvalidState возвращается при невалидном состоянии агрегата
	ErrInvalidState = errors.New("invalid aggregate state")

	// ErrInvalidTransition возвращается при невалидном переходе состояния
	ErrInvalidTransition = errors.New("invalid state transition")
)
