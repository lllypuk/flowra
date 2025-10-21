package shared_test

import (
	"context"
	"testing"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserIDContext(t *testing.T) {
	t.Run("set and get userID", func(t *testing.T) {
		userID := uuid.NewUUID()
		ctx := shared.WithUserID(context.Background(), userID)

		retrievedID, err := shared.GetUserID(ctx)
		require.NoError(t, err)
		assert.Equal(t, userID, retrievedID)
	})

	t.Run("get userID from empty context", func(t *testing.T) {
		_, err := shared.GetUserID(context.Background())
		require.Error(t, err)
		assert.Equal(t, shared.ErrUserIDNotFound, err)
	})
}

func TestWorkspaceIDContext(t *testing.T) {
	t.Run("set and get workspaceID", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		ctx := shared.WithWorkspaceID(context.Background(), workspaceID)

		retrievedID, err := shared.GetWorkspaceID(ctx)
		require.NoError(t, err)
		assert.Equal(t, workspaceID, retrievedID)
	})

	t.Run("get workspaceID from empty context", func(t *testing.T) {
		_, err := shared.GetWorkspaceID(context.Background())
		require.Error(t, err)
		assert.Equal(t, shared.ErrWorkspaceIDNotFound, err)
	})
}

func TestCorrelationIDContext(t *testing.T) {
	t.Run("set and get correlationID", func(t *testing.T) {
		correlationID := "test-correlation-id-123"
		ctx := shared.WithCorrelationID(context.Background(), correlationID)

		retrievedID, err := shared.GetCorrelationID(ctx)
		require.NoError(t, err)
		assert.Equal(t, correlationID, retrievedID)
	})

	t.Run("get correlationID from empty context", func(t *testing.T) {
		_, err := shared.GetCorrelationID(context.Background())
		require.Error(t, err)
		assert.Equal(t, shared.ErrCorrelationIDNotFound, err)
	})
}

func TestTraceIDContext(t *testing.T) {
	t.Run("set and get traceID", func(t *testing.T) {
		traceID := "test-trace-id-456"
		ctx := shared.WithTraceID(context.Background(), traceID)

		retrievedID := shared.GetTraceID(ctx)
		assert.Equal(t, traceID, retrievedID)
	})

	t.Run("get traceID from empty context returns empty string", func(t *testing.T) {
		traceID := shared.GetTraceID(context.Background())
		assert.Empty(t, traceID)
	})
}

func TestMultipleContextValues(t *testing.T) {
	t.Run("set multiple values in context", func(t *testing.T) {
		userID := uuid.NewUUID()
		workspaceID := uuid.NewUUID()
		correlationID := "correlation-123"
		traceID := "trace-456"

		ctx := context.Background()
		ctx = shared.WithUserID(ctx, userID)
		ctx = shared.WithWorkspaceID(ctx, workspaceID)
		ctx = shared.WithCorrelationID(ctx, correlationID)
		ctx = shared.WithTraceID(ctx, traceID)

		// Verify all values can be retrieved
		retrievedUserID, err := shared.GetUserID(ctx)
		require.NoError(t, err)
		assert.Equal(t, userID, retrievedUserID)

		retrievedWorkspaceID, err := shared.GetWorkspaceID(ctx)
		require.NoError(t, err)
		assert.Equal(t, workspaceID, retrievedWorkspaceID)

		retrievedCorrelationID, err := shared.GetCorrelationID(ctx)
		require.NoError(t, err)
		assert.Equal(t, correlationID, retrievedCorrelationID)

		retrievedTraceID := shared.GetTraceID(ctx)
		assert.Equal(t, traceID, retrievedTraceID)
	})
}
