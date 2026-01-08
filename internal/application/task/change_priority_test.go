package task_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/eventstore"
)

func TestChangePriorityUseCase_Success(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	createUseCase := taskapp.NewCreateTaskUseCase(store)
	priorityUseCase := taskapp.NewChangePriorityUseCase(store)

	// Creating task s Medium priority (default)
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// menyaem prioritet
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
	priorities := []task.Priority{
		task.PriorityLow,
		task.PriorityMedium,
		task.PriorityHigh,
		task.PriorityCritical,
	}

	for _, priority := range priorities {
		t.Run(string(priority), func(t *testing.T) {
			// Arrange
			store := eventstore.NewInMemoryEventStore()
			createUseCase := taskapp.NewCreateTaskUseCase(store)
			priorityUseCase := taskapp.NewChangePriorityUseCase(store)

			createCmd := taskapp.CreateTaskCommand{
				ChatID:    uuid.NewUUID(),
				Title:     "Test Task",
				Priority:  task.PriorityLow,
				CreatedBy: uuid.NewUUID(),
			}
			createResult, err := createUseCase.Execute(context.Background(), createCmd)
			require.NoError(t, err)

			// Act
			priorityCmd := taskapp.ChangePriorityCommand{
				TaskID:    createResult.TaskID,
				Priority:  priority,
				ChangedBy: uuid.NewUUID(),
			}
			result, err := priorityUseCase.Execute(context.Background(), priorityCmd)

			// Assert
			if priority == task.PriorityLow {
				// idempotentnost - tot zhe prioritet
				require.NoError(t, err)
				assert.Empty(t, result.Events)
			} else {
				require.NoError(t, err)
				event, ok := result.Events[0].(*task.PriorityChanged)
				require.True(t, ok)
				assert.Equal(t, priority, event.NewPriority)
			}
		})
	}
}

func TestChangePriorityUseCase_Idempotent(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	createUseCase := taskapp.NewCreateTaskUseCase(store)
	priorityUseCase := taskapp.NewChangePriorityUseCase(store)

	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		Priority:  task.PriorityHigh,
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: povtornaya setting togo zhe priority
	priorityCmd := taskapp.ChangePriorityCommand{
		TaskID:    createResult.TaskID,
		Priority:  task.PriorityHigh,
		ChangedBy: uuid.NewUUID(),
	}
	result, err := priorityUseCase.Execute(context.Background(), priorityCmd)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, result.Events, "No New events for idempotent operation")
	assert.Equal(t, 1, result.Version, "Version should not change")
	assert.True(t, result.IsSuccess())
	assert.Equal(t, "Priority unchanged (idempotent operation)", result.Message)
}

func TestChangePriorityUseCase_MultiplePriorityChanges(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	createUseCase := taskapp.NewCreateTaskUseCase(store)
	priorityUseCase := taskapp.NewChangePriorityUseCase(store)

	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	userID := uuid.NewUUID()

	// Act & Assert: Medium → High → Critical → Low
	result1, err := priorityUseCase.Execute(context.Background(), taskapp.ChangePriorityCommand{
		TaskID:    createResult.TaskID,
		Priority:  task.PriorityHigh,
		ChangedBy: userID,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result1.Version)

	result2, err := priorityUseCase.Execute(context.Background(), taskapp.ChangePriorityCommand{
		TaskID:    createResult.TaskID,
		Priority:  task.PriorityCritical,
		ChangedBy: userID,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, result2.Version)

	result3, err := priorityUseCase.Execute(context.Background(), taskapp.ChangePriorityCommand{
		TaskID:    createResult.TaskID,
		Priority:  task.PriorityLow,
		ChangedBy: userID,
	})
	require.NoError(t, err)
	assert.Equal(t, 4, result3.Version)

	// Checking full history
	storedEvents, err := store.LoadEvents(context.Background(), createResult.TaskID.String())
	require.NoError(t, err)
	assert.Len(t, storedEvents, 4) // Created + 3x PriorityChanged
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
				Priority:  "Urgent",
				ChangedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrInvalidPriority,
		},
		{
			name: "Case Sensitive - lowercase",
			cmd: taskapp.ChangePriorityCommand{
				TaskID:    uuid.NewUUID(),
				Priority:  "high",
				ChangedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrInvalidPriority,
		},
		{
			name: "Case Sensitive - uppercase",
			cmd: taskapp.ChangePriorityCommand{
				TaskID:    uuid.NewUUID(),
				Priority:  "HIGH",
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
			store := eventstore.NewInMemoryEventStore()
			useCase := taskapp.NewChangePriorityUseCase(store)

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
	store := eventstore.NewInMemoryEventStore()
	useCase := taskapp.NewChangePriorityUseCase(store)

	cmd := taskapp.ChangePriorityCommand{
		TaskID:    uuid.NewUUID(), // not suschestvuet
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
