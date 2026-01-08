package appcore

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

// GetUserID extracts the user ID from the context
func GetUserID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	if !ok {
		return "", ErrUserIDNotFound
	}
	return userID, nil
}

// WithUserID adds the user ID to the context
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetWorkspaceID extracts the workspace ID from the context
func GetWorkspaceID(ctx context.Context) (uuid.UUID, error) {
	workspaceID, ok := ctx.Value(workspaceIDKey).(uuid.UUID)
	if !ok {
		return "", ErrWorkspaceIDNotFound
	}
	return workspaceID, nil
}

// WithWorkspaceID adds the workspace ID to the context
func WithWorkspaceID(ctx context.Context, workspaceID uuid.UUID) context.Context {
	return context.WithValue(ctx, workspaceIDKey, workspaceID)
}

// GetCorrelationID extracts the correlation ID from the context
func GetCorrelationID(ctx context.Context) (string, error) {
	correlationID, ok := ctx.Value(correlationIDKey).(string)
	if !ok {
		return "", ErrCorrelationIDNotFound
	}
	return correlationID, nil
}

// WithCorrelationID adds the correlation ID to the context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, correlationIDKey, correlationID)
}

// GetTraceID extracts the trace ID from the context (for distributed tracing)
func GetTraceID(ctx context.Context) string {
	traceID, ok := ctx.Value(traceIDKey).(string)
	if !ok {
		return ""
	}
	return traceID
}

// WithTraceID adds the trace ID to the context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}
