package task_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/tests/mocks"
)

func TestChangePriorityUseCase_Success(t *testing.T) {
	// Arrange
	repo := mocks.NewMockTaskRepository()
	createUseCase := taskapp.NewCreateTaskUseCase(repo)
	priorityUseCase := taskapp.NewChangePriorityUseCase(repo)

	// Create task with Medium priority (default)
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Change priority
	userID := uuid.NewUUID()
	priorityCmd := taskapp.ChangePriorityCommand{
		TaskID:    createResult.TaskID,
		Priority:  task.PriorityHigh,
		ChangedBy: userID,
	}

	// Act
	result, err := priorityUseCase.Execute(context.Background(), priorityCmd)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 2, result.Version)
	require.Len(t, result.Events, 1)

	event, ok := result.Events[0].(*task.PriorityChanged)
	require.True(t, ok, "Expected *task.PriorityChanged event")
	assert.Equal(t, createResult.TaskID, uuid.UUID(event.AggregateID()))
	assert.Equal(t, task.PriorityMedium, event.OldPriority)
	assert.Equal(t, task.PriorityHigh, event.NewPriority)
	assert.Equal(t, userID, event.ChangedBy)
}

func TestChangePriorityUseCase_AllPriorities(t *testing.T) {
	tests := []struct {
		name     string
		from     task.Priority
		to       task.Priority
		expected task.Priority
	}{
		{"Medium → High", task.PriorityMedium, task.PriorityHigh, task.PriorityHigh},
		{"High → Critical", task.PriorityHigh, task.PriorityCritical, task.PriorityCritical},
		{"Critical → Low", task.PriorityCritical, task.PriorityLow, task.PriorityLow},
		{"Low → Medium", task.PriorityLow, task.PriorityMedium, task.PriorityMedium},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := mocks.NewMockTaskRepository()
			createUseCase := taskapp.NewCreateTaskUseCase(repo)
			priorityUseCase := taskapp.NewChangePriorityUseCase(repo)

			// Create task with initial priority
			createCmd := taskapp.CreateTaskCommand{
				ChatID:    uuid.NewUUID(),
				Title:     "Test Task",
				Priority:  tt.from,
				CreatedBy: uuid.NewUUID(),
			}
			createResult, err := createUseCase.Execute(context.Background(), createCmd)
			require.NoError(t, err)

			// Act
			priorityCmd := taskapp.ChangePriorityCommand{
				TaskID:    createResult.TaskID,
				Priority:  tt.to,
				ChangedBy: uuid.NewUUID(),
			}
			result, err := priorityUseCase.Execute(context.Background(), priorityCmd)

			// Assert
			require.NoError(t, err)
			event, ok := result.Events[0].(*task.PriorityChanged)
			require.True(t, ok)
			assert.Equal(t, tt.from, event.OldPriority)
			assert.Equal(t, tt.expected, event.NewPriority)
		})
	}
}

func TestChangePriorityUseCase_Idempotent(t *testing.T) {
	// Arrange
	repo := mocks.NewMockTaskRepository()
	createUseCase := taskapp.NewCreateTaskUseCase(repo)
	priorityUseCase := taskapp.NewChangePriorityUseCase(repo)

	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		Priority:  task.PriorityHigh,
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: repeat the same priority
	priorityCmd := taskapp.ChangePriorityCommand{
		TaskID:    createResult.TaskID,
		Priority:  task.PriorityHigh, // same as before
		ChangedBy: uuid.NewUUID(),
	}
	result, err := priorityUseCase.Execute(context.Background(), priorityCmd)

	// Assert: should succeed but without new events
	require.NoError(t, err)
	assert.Empty(t, result.Events, "No new events should be generated for idempotent operation")
	assert.Equal(t, 1, result.Version, "Version should not change")
	assert.True(t, result.IsSuccess())
	assert.Equal(t, "Priority unchanged (idempotent operation)", result.Message)
}

func TestChangePriorityUseCase_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		cmd         taskapp.ChangePriorityCommand
		expectedErr error
	}{
		{
			name: "Empty TaskID",
			cmd: taskapp.ChangePriorityCommand{
				TaskID:    uuid.UUID(""),
				Priority:  task.PriorityHigh,
				ChangedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrInvalidTaskID,
		},
		{
			name: "Empty Priority",
			cmd: taskapp.ChangePriorityCommand{
				TaskID:    uuid.NewUUID(),
				Priority:  "",
				ChangedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrEmptyPriority,
		},
		{
			name: "Invalid Priority",
			cmd: taskapp.ChangePriorityCommand{
				TaskID:    uuid.NewUUID(),
				Priority:  "Urgent", // does not exist
				ChangedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrInvalidPriority,
		},
		{
			name: "Empty ChangedBy",
			cmd: taskapp.ChangePriorityCommand{
				TaskID:    uuid.NewUUID(),
				Priority:  task.PriorityHigh,
				ChangedBy: uuid.UUID(""),
			},
			expectedErr: taskapp.ErrInvalidUserID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := mocks.NewMockTaskRepository()
			useCase := taskapp.NewChangePriorityUseCase(repo)

			// Act
			result, err := useCase.Execute(context.Background(), tt.cmd)

			// Assert
			require.Error(t, err)
			require.ErrorIs(t, err, tt.expectedErr)
			assert.Empty(t, result.Events)
		})
	}
}

func TestChangePriorityUseCase_TaskNotFound(t *testing.T) {
	// Arrange
	repo := mocks.NewMockTaskRepository()
	useCase := taskapp.NewChangePriorityUseCase(repo)

	cmd := taskapp.ChangePriorityCommand{
		TaskID:    uuid.NewUUID(), // does not exist
		Priority:  task.PriorityHigh,
		ChangedBy: uuid.NewUUID(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, taskapp.ErrTaskNotFound)
	assert.Empty(t, result.Events)
}
