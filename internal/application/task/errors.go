package task

import (
	"net/http"
)

// appError is a helper type that implements httpserver.HTTPError interface.
type appError struct {
	msg        string
	httpStatus int
	httpCode   string
	httpMsg    string
}

func (e *appError) Error() string       { return e.msg }
func (e *appError) HTTPStatus() int     { return e.httpStatus }
func (e *appError) HTTPCode() string    { return e.httpCode }
func (e *appError) HTTPMessage() string { return e.httpMsg }

var (
	// Validation errors - ошибки валидации входных данных

	// ErrInvalidChatID возвращается когда ChatID невалиден
	ErrInvalidChatID = &appError{
		msg:        "invalid chat ID",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_CHAT_ID",
		httpMsg:    "invalid chat ID",
	}

	// ErrEmptyTitle возвращается когда заголовок пустой
	ErrEmptyTitle = &appError{
		msg:        "task title cannot be empty",
		httpStatus: http.StatusBadRequest,
		httpCode:   "EMPTY_TITLE",
		httpMsg:    "task title cannot be empty",
	}

	// ErrInvalidPriority возвращается когда приоритет невалиден
	ErrInvalidPriority = &appError{
		msg:        "invalid priority value",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_PRIORITY",
		httpMsg:    "invalid priority value",
	}

	// ErrEmptyPriority возвращается когда приоритет пустой
	ErrEmptyPriority = &appError{
		msg:        "priority cannot be empty",
		httpStatus: http.StatusBadRequest,
		httpCode:   "EMPTY_PRIORITY",
		httpMsg:    "priority cannot be empty",
	}

	// ErrInvalidStatus возвращается когда статус невалиден
	ErrInvalidStatus = &appError{
		msg:        "invalid status value",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_STATUS",
		httpMsg:    "invalid status value",
	}

	// ErrInvalidUserID возвращается когда ID пользователя невалиден
	ErrInvalidUserID = &appError{
		msg:        "invalid user ID",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_USER_ID",
		httpMsg:    "invalid user ID",
	}

	// ErrInvalidTaskID возвращается когда ID задачи невалиден
	ErrInvalidTaskID = &appError{
		msg:        "invalid task ID",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_TASK_ID",
		httpMsg:    "invalid task ID",
	}

	// ErrInvalidDate возвращается когда дата невалидна
	ErrInvalidDate = &appError{
		msg:        "invalid date value",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_DATE",
		httpMsg:    "invalid date value",
	}

	// ErrInvalidEntityType возвращается когда тип сущности невалиден
	ErrInvalidEntityType = &appError{
		msg:        "invalid entity type",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_ENTITY_TYPE",
		httpMsg:    "invalid entity type",
	}

	// ErrInvalidTitle возвращается когда заголовок невалиден (слишком длинный и т.д.)
	ErrInvalidTitle = &appError{
		msg:        "invalid task title",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_TITLE",
		httpMsg:    "invalid task title",
	}

	// Business logic errors - ошибки бизнес-логики

	// ErrTaskNotFound возвращается когда задача не найдена
	ErrTaskNotFound = &appError{
		msg:        "task not found",
		httpStatus: http.StatusNotFound,
		httpCode:   "TASK_NOT_FOUND",
		httpMsg:    "task not found",
	}

	// ErrUnauthorized возвращается когда пользователь не авторизован для операции
	ErrUnauthorized = &appError{
		msg:        "user not authorized for this operation",
		httpStatus: http.StatusForbidden,
		httpCode:   "FORBIDDEN",
		httpMsg:    "not authorized for this operation",
	}

	// ErrConcurrentUpdate возвращается при конфликте версий (optimistic locking)
	ErrConcurrentUpdate = &appError{
		msg:        "concurrent update detected",
		httpStatus: http.StatusConflict,
		httpCode:   "CONCURRENT_UPDATE",
		httpMsg:    "task was modified by another request",
	}

	// ErrInvalidStatusTransition возвращается при невалидном переходе статуса
	ErrInvalidStatusTransition = &appError{
		msg:        "invalid status transition",
		httpStatus: http.StatusUnprocessableEntity,
		httpCode:   "INVALID_STATUS_TRANSITION",
		httpMsg:    "invalid status transition",
	}

	// ErrUserNotFound возвращается когда пользователь не найден
	ErrUserNotFound = &appError{
		msg:        "user not found",
		httpStatus: http.StatusBadRequest,
		httpCode:   "USER_NOT_FOUND",
		httpMsg:    "assignee user not found",
	}

	// ErrTaskAlreadyExists возвращается когда задача уже существует
	ErrTaskAlreadyExists = &appError{
		msg:        "task already exists",
		httpStatus: http.StatusConflict,
		httpCode:   "TASK_ALREADY_EXISTS",
		httpMsg:    "task already exists",
	}

	// ErrDueDateInPast возвращается когда дедлайн указан в прошлом
	ErrDueDateInPast = &appError{
		msg:        "due date cannot be in the past",
		httpStatus: http.StatusBadRequest,
		httpCode:   "DUE_DATE_IN_PAST",
		httpMsg:    "due date cannot be in the past",
	}
)
