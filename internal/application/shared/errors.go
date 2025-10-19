package shared

import (
	"errors"
	"fmt"
)

// Common application errors
var (
	// Validation errors
	ErrValidationFailed = errors.New("validation failed")
	ErrInvalidID        = errors.New("invalid ID")
	ErrEmptyField       = errors.New("required field is empty")
	ErrInvalidFormat    = errors.New("invalid format")

	// Authorization errors
	ErrUnauthorized            = errors.New("unauthorized")
	ErrForbidden               = errors.New("forbidden")
	ErrInsufficientPermissions = errors.New("insufficient permissions")

	// Not found errors
	ErrNotFound          = errors.New("resource not found")
	ErrChatNotFound      = errors.New("chat not found")
	ErrMessageNotFound   = errors.New("message not found")
	ErrUserNotFound      = errors.New("user not found")
	ErrWorkspaceNotFound = errors.New("workspace not found")
	ErrTaskNotFound      = errors.New("task not found")

	// Conflict errors
	ErrConflict         = errors.New("conflict")
	ErrAlreadyExists    = errors.New("resource already exists")
	ErrConcurrentUpdate = errors.New("concurrent update detected")

	// Infrastructure errors
	ErrDatabaseError   = errors.New("database error")
	ErrEventStoreError = errors.New("event store error")
	ErrEventBusError   = errors.New("event bus error")
)

// ValidationError — ошибка валидации с контекстом
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// NewValidationError создает ValidationError
func NewValidationError(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}

// AuthorizationError — ошибка авторизации
type AuthorizationError struct {
	UserID   string
	Resource string
	Action   string
}

func (e AuthorizationError) Error() string {
	return fmt.Sprintf("user %s is not authorized to %s on %s", e.UserID, e.Action, e.Resource)
}

// NewAuthorizationError создает AuthorizationError
func NewAuthorizationError(userID, resource, action string) error {
	return &AuthorizationError{
		UserID:   userID,
		Resource: resource,
		Action:   action,
	}
}

// NotFoundError — ошибка "не найдено"
type NotFoundError struct {
	Resource string
	ID       string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("%s with ID %s not found", e.Resource, e.ID)
}

// NewNotFoundError создает NotFoundError
func NewNotFoundError(resource, id string) error {
	return &NotFoundError{
		Resource: resource,
		ID:       id,
	}
}

// ConflictError — ошибка конфликта
type ConflictError struct {
	Resource string
	Reason   string
}

func (e ConflictError) Error() string {
	return fmt.Sprintf("conflict on %s: %s", e.Resource, e.Reason)
}

// NewConflictError создает ConflictError
func NewConflictError(resource, reason string) error {
	return &ConflictError{
		Resource: resource,
		Reason:   reason,
	}
}
