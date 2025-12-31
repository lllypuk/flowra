package appcore_test

import (
	"context"
	"testing"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserIDContext(t *testing.T) {
	t.Run("set and get userID", func(t *testing.T) {
		userID := uuid.NewUUID()
		ctx := appcore.WithUserID(context.Background(), userID)

		retrievedID, err := appcore.GetUserID(ctx)
		require.NoError(t, err)
		assert.Equal(t, userID, retrievedID)
	})

	t.Run("get userID from empty context", func(t *testing.T) {
		_, err := appcore.GetUserID(context.Background())
		require.Error(t, err)
		assert.Equal(t, appcore.ErrUserIDNotFound, err)
	})
}

func TestWorkspaceIDContext(t *testing.T) {
	t.Run("set and get workspaceID", func(t *testing.T) {
		workspaceID := uuid.NewUUID()
		ctx := appcore.WithWorkspaceID(context.Background(), workspaceID)

		retrievedID, err := appcore.GetWorkspaceID(ctx)
		require.NoError(t, err)
		assert.Equal(t, workspaceID, retrievedID)
	})

	t.Run("get workspaceID from empty context", func(t *testing.T) {
		_, err := appcore.GetWorkspaceID(context.Background())
		require.Error(t, err)
		assert.Equal(t, appcore.ErrWorkspaceIDNotFound, err)
	})
}

func TestCorrelationIDContext(t *testing.T) {
	t.Run("set and get correlationID", func(t *testing.T) {
		correlationID := "test-correlation-id-123"
		ctx := appcore.WithCorrelationID(context.Background(), correlationID)

		retrievedID, err := appcore.GetCorrelationID(ctx)
		require.NoError(t, err)
		assert.Equal(t, correlationID, retrievedID)
	})

	t.Run("get correlationID from empty context", func(t *testing.T) {
		_, err := appcore.GetCorrelationID(context.Background())
		require.Error(t, err)
		assert.Equal(t, appcore.ErrCorrelationIDNotFound, err)
	})
}

func TestTraceIDContext(t *testing.T) {
	t.Run("set and get traceID", func(t *testing.T) {
		traceID := "test-trace-id-456"
		ctx := appcore.WithTraceID(context.Background(), traceID)

		retrievedID := appcore.GetTraceID(ctx)
		assert.Equal(t, traceID, retrievedID)
	})

	t.Run("get traceID from empty context returns empty string", func(t *testing.T) {
		traceID := appcore.GetTraceID(context.Background())
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
		ctx = appcore.WithUserID(ctx, userID)
		ctx = appcore.WithWorkspaceID(ctx, workspaceID)
		ctx = appcore.WithCorrelationID(ctx, correlationID)
		ctx = appcore.WithTraceID(ctx, traceID)

		// Verify all values can be retrieved
		retrievedUserID, err := appcore.GetUserID(ctx)
		require.NoError(t, err)
		assert.Equal(t, userID, retrievedUserID)

		retrievedWorkspaceID, err := appcore.GetWorkspaceID(ctx)
		require.NoError(t, err)
		assert.Equal(t, workspaceID, retrievedWorkspaceID)

		retrievedCorrelationID, err := appcore.GetCorrelationID(ctx)
		require.NoError(t, err)
		assert.Equal(t, correlationID, retrievedCorrelationID)

		retrievedTraceID := appcore.GetTraceID(ctx)
		assert.Equal(t, traceID, retrievedTraceID)
	})
}
