package shared

import (
	"context"
	"errors"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Context keys
type contextKey string

const (
	userIDKey        contextKey = "userID"
	workspaceIDKey   contextKey = "workspaceID"
	correlationIDKey contextKey = "correlationID"
	traceIDKey       contextKey = "traceID"
)

var (
	ErrUserIDNotFound        = errors.New("user ID not found in context")
	ErrWorkspaceIDNotFound   = errors.New("workspace ID not found in context")
	ErrCorrelationIDNotFound = errors.New("correlation ID not found in context")
)

// GetUserID извлекает ID пользователя из контекста
func GetUserID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	if !ok {
		return "", ErrUserIDNotFound
	}
	return userID, nil
}

// WithUserID добавляет ID пользователя в контекст
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetWorkspaceID извлекает ID workspace из контекста
func GetWorkspaceID(ctx context.Context) (uuid.UUID, error) {
	workspaceID, ok := ctx.Value(workspaceIDKey).(uuid.UUID)
	if !ok {
		return "", ErrWorkspaceIDNotFound
	}
	return workspaceID, nil
}

// WithWorkspaceID добавляет ID workspace в контекст
func WithWorkspaceID(ctx context.Context, workspaceID uuid.UUID) context.Context {
	return context.WithValue(ctx, workspaceIDKey, workspaceID)
}

// GetCorrelationID извлекает correlation ID из контекста
func GetCorrelationID(ctx context.Context) (string, error) {
	correlationID, ok := ctx.Value(correlationIDKey).(string)
	if !ok {
		return "", ErrCorrelationIDNotFound
	}
	return correlationID, nil
}

// WithCorrelationID добавляет correlation ID в контекст
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationIDKey, correlationID)
}

// GetTraceID извлекает trace ID из контекста (для distributed tracing)
func GetTraceID(ctx context.Context) string {
	traceID, ok := ctx.Value(traceIDKey).(string)
	if !ok {
		return ""
	}
	return traceID
}

// WithTraceID добавляет trace ID в контекст
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}
