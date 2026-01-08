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
	// Validation errors - input data validation errors

	// ErrInvalidChatID is returned when ChatID is invalid
	ErrInvalidChatID = &appError{
		msg:        "invalid chat ID",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_CHAT_ID",
		httpMsg:    "invalid chat ID",
	}

	// ErrEmptyTitle is returned when title is empty
	ErrEmptyTitle = &appError{
		msg:        "task title cannot be empty",
		httpStatus: http.StatusBadRequest,
		httpCode:   "EMPTY_TITLE",
		httpMsg:    "task title cannot be empty",
	}

	// ErrInvalidPriority is returned when priority is invalid
	ErrInvalidPriority = &appError{
		msg:        "invalid priority value",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_PRIORITY",
		httpMsg:    "invalid priority value",
	}

	// ErrEmptyPriority is returned when priority is empty
	ErrEmptyPriority = &appError{
		msg:        "priority cannot be empty",
		httpStatus: http.StatusBadRequest,
		httpCode:   "EMPTY_PRIORITY",
		httpMsg:    "priority cannot be empty",
	}

	// ErrInvalidStatus is returned when status is invalid
	ErrInvalidStatus = &appError{
		msg:        "invalid status value",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_STATUS",
		httpMsg:    "invalid status value",
	}

	// ErrInvalidUserID is returned when user ID is invalid
	ErrInvalidUserID = &appError{
		msg:        "invalid user ID",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_USER_ID",
		httpMsg:    "invalid user ID",
	}

	// ErrInvalidTaskID is returned when task ID is invalid
	ErrInvalidTaskID = &appError{
		msg:        "invalid task ID",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_TASK_ID",
		httpMsg:    "invalid task ID",
	}

	// ErrInvalidDate is returned when date is invalid
	ErrInvalidDate = &appError{
		msg:        "invalid date value",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_DATE",
		httpMsg:    "invalid date value",
	}

	// ErrInvalidEntityType is returned when entity type is invalid
	ErrInvalidEntityType = &appError{
		msg:        "invalid entity type",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_ENTITY_TYPE",
		httpMsg:    "invalid entity type",
	}

	// ErrInvalidTitle is returned when title is invalid (too long, etc.)
	ErrInvalidTitle = &appError{
		msg:        "invalid task title",
		httpStatus: http.StatusBadRequest,
		httpCode:   "INVALID_TITLE",
		httpMsg:    "invalid task title",
	}

	// Business logic errors

	// ErrTaskNotFound is returned when task is not found
	ErrTaskNotFound = &appError{
		msg:        "task not found",
		httpStatus: http.StatusNotFound,
		httpCode:   "TASK_NOT_FOUND",
		httpMsg:    "task not found",
	}

	// ErrUnauthorized is returned when user is not authorized for the operation
	ErrUnauthorized = &appError{
		msg:        "user not authorized for this operation",
		httpStatus: http.StatusForbidden,
		httpCode:   "FORBIDDEN",
		httpMsg:    "not authorized for this operation",
	}

	// ErrConcurrentUpdate is returned on version conflict (optimistic locking)
	ErrConcurrentUpdate = &appError{
		msg:        "concurrent update detected",
		httpStatus: http.StatusConflict,
		httpCode:   "CONCURRENT_UPDATE",
		httpMsg:    "task was modified by another request",
	}

	// ErrInvalidStatusTransition is returned on invalid status transition
	ErrInvalidStatusTransition = &appError{
		msg:        "invalid status transition",
		httpStatus: http.StatusUnprocessableEntity,
		httpCode:   "INVALID_STATUS_TRANSITION",
		httpMsg:    "invalid status transition",
	}

	// ErrUserNotFound is returned when user is not found
	ErrUserNotFound = &appError{
		msg:        "user not found",
		httpStatus: http.StatusBadRequest,
		httpCode:   "USER_NOT_FOUND",
		httpMsg:    "assignee user not found",
	}

	// ErrTaskAlreadyExists is returned when task already exists
	ErrTaskAlreadyExists = &appError{
		msg:        "task already exists",
		httpStatus: http.StatusConflict,
		httpCode:   "TASK_ALREADY_EXISTS",
		httpMsg:    "task already exists",
	}

	// ErrDueDateInPast is returned when due date is in the past
	ErrDueDateInPast = &appError{
		msg:        "due date cannot be in the past",
		httpStatus: http.StatusBadRequest,
		httpCode:   "DUE_DATE_IN_PAST",
		httpMsg:    "due date cannot be in the past",
	}
)
