package task

import "errors"

var (
	// Validation errors - ошибки валидации входных данных

	// ErrInvalidChatID возвращается когда ChatID невалиден
	ErrInvalidChatID = errors.New("invalid chat ID")

	// ErrEmptyTitle возвращается когда заголовок пустой
	ErrEmptyTitle = errors.New("task title cannot be empty")

	// ErrInvalidPriority возвращается когда приоритет невалиден
	ErrInvalidPriority = errors.New("invalid priority value")

	// ErrEmptyPriority возвращается когда приоритет пустой
	ErrEmptyPriority = errors.New("priority cannot be empty")

	// ErrInvalidStatus возвращается когда статус невалиден
	ErrInvalidStatus = errors.New("invalid status value")

	// ErrInvalidUserID возвращается когда ID пользователя невалиден
	ErrInvalidUserID = errors.New("invalid user ID")

	// ErrInvalidTaskID возвращается когда ID задачи невалиден
	ErrInvalidTaskID = errors.New("invalid task ID")

	// ErrInvalidDate возвращается когда дата невалидна
	ErrInvalidDate = errors.New("invalid date value")

	// ErrInvalidEntityType возвращается когда тип сущности невалиден
	ErrInvalidEntityType = errors.New("invalid entity type")

	// ErrInvalidTitle возвращается когда заголовок невалиден (слишком длинный и т.д.)
	ErrInvalidTitle = errors.New("invalid task title")

	// Business logic errors - ошибки бизнес-логики

	// ErrTaskNotFound возвращается когда задача не найдена
	ErrTaskNotFound = errors.New("task not found")

	// ErrUnauthorized возвращается когда пользователь не авторизован для операции
	ErrUnauthorized = errors.New("user not authorized for this operation")

	// ErrConcurrentUpdate возвращается при конфликте версий (optimistic locking)
	ErrConcurrentUpdate = errors.New("concurrent update detected")

	// ErrInvalidStatusTransition возвращается при невалидном переходе статуса
	ErrInvalidStatusTransition = errors.New("invalid status transition")

	// ErrUserNotFound возвращается когда пользователь не найден
	ErrUserNotFound = errors.New("user not found")

	// ErrTaskAlreadyExists возвращается когда задача уже существует
	ErrTaskAlreadyExists = errors.New("task already exists")

	// ErrDueDateInPast возвращается когда дедлайн указан в прошлом
	ErrDueDateInPast = errors.New("due date cannot be in the past")
)
