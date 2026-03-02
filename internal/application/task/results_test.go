package task_test

import (
	"testing"

	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/stretchr/testify/require"
)

func TestTaskResult_NewSuccessResult(t *testing.T) {
	taskID := uuid.NewUUID()

	result := taskapp.NewSuccessResult(taskID, 7)

	require.Equal(t, taskID, result.TaskID)
	require.Equal(t, 7, result.Version)
	require.True(t, result.IsSuccess())
	require.False(t, result.IsFailure())
	require.Empty(t, result.Message)
}

func TestTaskResult_NewFailureResult(t *testing.T) {
	taskID := uuid.NewUUID()

	result := taskapp.NewFailureResult(taskID, "boom")

	require.Equal(t, taskID, result.TaskID)
	require.Zero(t, result.Version)
	require.False(t, result.IsSuccess())
	require.True(t, result.IsFailure())
	require.Equal(t, "boom", result.Message)
}
