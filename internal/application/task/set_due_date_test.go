package task_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/tests/mocks"
)

func TestSetDueDateUseCase_Success(t *testing.T) {
	// Arrange
	repo := mocks.NewMockTaskRepository()
	createUseCase := taskapp.NewCreateTaskUseCase(repo)
	dueDateUseCase := taskapp.NewSetDueDateUseCase(repo)

	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Set due date
	dueDate := time.Now().Add(7 * 24 * time.Hour) // in a week
	userID := uuid.NewUUID()
	dueDateCmd := taskapp.SetDueDateCommand{
		TaskID:    createResult.TaskID,
		DueDate:   &dueDate,
		ChangedBy: userID,
	}

	// Act
	result, err := dueDateUseCase.Execute(context.Background(), dueDateCmd)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 2, result.Version)
	require.Len(t, result.Events, 1)

	event, ok := result.Events[0].(*task.DueDateChanged)
	require.True(t, ok, "Expected *task.DueDateChanged event")
	assert.Equal(t, createResult.TaskID, uuid.UUID(event.AggregateID()))
	assert.Nil(t, event.OldDueDate)
	assert.NotNil(t, event.NewDueDate)
	assert.Equal(t, userID, event.ChangedBy)
}

func TestSetDueDateUseCase_Remove(t *testing.T) {
	// Arrange
	repo := mocks.NewMockTaskRepository()
	createUseCase := taskapp.NewCreateTaskUseCase(repo)
	dueDateUseCase := taskapp.NewSetDueDateUseCase(repo)

	// Create task with due date
	initialDueDate := time.Now().Add(24 * time.Hour)
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		DueDate:   &initialDueDate,
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: remove due date (nil)
	removeCmd := taskapp.SetDueDateCommand{
		TaskID:    createResult.TaskID,
		DueDate:   nil, // remove
		ChangedBy: uuid.NewUUID(),
	}
	result, err := dueDateUseCase.Execute(context.Background(), removeCmd)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 2, result.Version)
	require.Len(t, result.Events, 1)

	event, ok := result.Events[0].(*task.DueDateChanged)
	require.True(t, ok)
	assert.NotNil(t, event.OldDueDate)
	assert.Nil(t, event.NewDueDate)
}

func TestSetDueDateUseCase_Change(t *testing.T) {
	// Arrange
	repo := mocks.NewMockTaskRepository()
	createUseCase := taskapp.NewCreateTaskUseCase(repo)
	dueDateUseCase := taskapp.NewSetDueDateUseCase(repo)

	// Create task with due date
	initialDueDate := time.Now().Add(24 * time.Hour)
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		DueDate:   &initialDueDate,
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: change due date
	newDueDate := time.Now().Add(48 * time.Hour)
	changeCmd := taskapp.SetDueDateCommand{
		TaskID:    createResult.TaskID,
		DueDate:   &newDueDate,
		ChangedBy: uuid.NewUUID(),
	}
	result, err := dueDateUseCase.Execute(context.Background(), changeCmd)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 2, result.Version)
	require.Len(t, result.Events, 1)

	event, ok := result.Events[0].(*task.DueDateChanged)
	require.True(t, ok)
	assert.NotNil(t, event.OldDueDate)
	assert.NotNil(t, event.NewDueDate)
}

func TestSetDueDateUseCase_Idempotent(t *testing.T) {
	// Arrange
	repo := mocks.NewMockTaskRepository()
	createUseCase := taskapp.NewCreateTaskUseCase(repo)
	dueDateUseCase := taskapp.NewSetDueDateUseCase(repo)

	dueDate := time.Now().Add(24 * time.Hour)
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		DueDate:   &dueDate,
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: repeat the same due date
	dueDateCmd := taskapp.SetDueDateCommand{
		TaskID:    createResult.TaskID,
		DueDate:   &dueDate, // same as before
		ChangedBy: uuid.NewUUID(),
	}
	result, err := dueDateUseCase.Execute(context.Background(), dueDateCmd)

	// Assert: should succeed but without new events
	require.NoError(t, err)
	assert.Empty(t, result.Events, "No new events should be generated for idempotent operation")
	assert.Equal(t, 1, result.Version, "Version should not change")
	assert.True(t, result.IsSuccess())
	assert.Equal(t, "Due date unchanged (idempotent operation)", result.Message)
}

func TestSetDueDateUseCase_NilIdempotent(t *testing.T) {
	// Arrange
	repo := mocks.NewMockTaskRepository()
	createUseCase := taskapp.NewCreateTaskUseCase(repo)
	dueDateUseCase := taskapp.NewSetDueDateUseCase(repo)

	// Create task without due date
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		DueDate:   nil,
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: set due date to nil (same as current)
	dueDateCmd := taskapp.SetDueDateCommand{
		TaskID:    createResult.TaskID,
		DueDate:   nil, // same as before
		ChangedBy: uuid.NewUUID(),
	}
	result, err := dueDateUseCase.Execute(context.Background(), dueDateCmd)

	// Assert: should succeed but without new events
	require.NoError(t, err)
	assert.Empty(t, result.Events, "No new events should be generated for idempotent operation")
}

func TestSetDueDateUseCase_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		cmd         taskapp.SetDueDateCommand
		expectedErr error
	}{
		{
			name: "Empty TaskID",
			cmd: taskapp.SetDueDateCommand{
				TaskID:    uuid.UUID(""),
				DueDate:   nil,
				ChangedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrInvalidTaskID,
		},
		{
			name: "Date too far in past",
			cmd: taskapp.SetDueDateCommand{
				TaskID:    uuid.NewUUID(),
				DueDate:   ptrTime(time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)),
				ChangedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrInvalidDate,
		},
		{
			name: "Empty ChangedBy",
			cmd: taskapp.SetDueDateCommand{
				TaskID:    uuid.NewUUID(),
				DueDate:   nil,
				ChangedBy: uuid.UUID(""),
			},
			expectedErr: taskapp.ErrInvalidUserID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := mocks.NewMockTaskRepository()
			useCase := taskapp.NewSetDueDateUseCase(repo)

			// Act
			result, err := useCase.Execute(context.Background(), tt.cmd)

			// Assert
			require.Error(t, err)
			require.ErrorIs(t, err, tt.expectedErr)
			assert.Empty(t, result.Events)
		})
	}
}

func TestSetDueDateUseCase_TaskNotFound(t *testing.T) {
	// Arrange
	repo := mocks.NewMockTaskRepository()
	useCase := taskapp.NewSetDueDateUseCase(repo)

	cmd := taskapp.SetDueDateCommand{
		TaskID:    uuid.NewUUID(), // does not exist
		DueDate:   nil,
		ChangedBy: uuid.NewUUID(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, taskapp.ErrTaskNotFound)
	assert.Empty(t, result.Events)
}

// Helper function
func ptrTime(t time.Time) *time.Time {
	return &t
}
