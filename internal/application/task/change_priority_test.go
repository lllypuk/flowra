package task_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/teams-up/internal/domain/task"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
	"github.com/lllypuk/teams-up/internal/infrastructure/eventstore"
	taskusecase "github.com/lllypuk/teams-up/internal/usecase/task"
)

func TestChangePriorityUseCase_Success(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	createUseCase := taskusecase.NewCreateTaskUseCase(store)
	priorityUseCase := taskusecase.NewChangePriorityUseCase(store)

	// Создаем задачу с Medium priority (default)
	createCmd := taskusecase.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Меняем приоритет
	userID := uuid.NewUUID()
	priorityCmd := taskusecase.ChangePriorityCommand{
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
			createUseCase := taskusecase.NewCreateTaskUseCase(store)
			priorityUseCase := taskusecase.NewChangePriorityUseCase(store)

			createCmd := taskusecase.CreateTaskCommand{
				ChatID:    uuid.NewUUID(),
				Title:     "Test Task",
				Priority:  task.PriorityLow,
				CreatedBy: uuid.NewUUID(),
			}
			createResult, err := createUseCase.Execute(context.Background(), createCmd)
			require.NoError(t, err)

			// Act
			priorityCmd := taskusecase.ChangePriorityCommand{
				TaskID:    createResult.TaskID,
				Priority:  priority,
				ChangedBy: uuid.NewUUID(),
			}
			result, err := priorityUseCase.Execute(context.Background(), priorityCmd)

			// Assert
			if priority == task.PriorityLow {
				// Идемпотентность - тот же приоритет
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
	createUseCase := taskusecase.NewCreateTaskUseCase(store)
	priorityUseCase := taskusecase.NewChangePriorityUseCase(store)

	createCmd := taskusecase.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		Priority:  task.PriorityHigh,
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: Повторная установка того же приоритета
	priorityCmd := taskusecase.ChangePriorityCommand{
		TaskID:    createResult.TaskID,
		Priority:  task.PriorityHigh,
		ChangedBy: uuid.NewUUID(),
	}
	result, err := priorityUseCase.Execute(context.Background(), priorityCmd)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, result.Events, "No new events for idempotent operation")
	assert.Equal(t, 1, result.Version, "Version should not change")
	assert.True(t, result.IsSuccess())
	assert.Equal(t, "Priority unchanged (idempotent operation)", result.Message)
}

func TestChangePriorityUseCase_MultiplePriorityChanges(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	createUseCase := taskusecase.NewCreateTaskUseCase(store)
	priorityUseCase := taskusecase.NewChangePriorityUseCase(store)

	createCmd := taskusecase.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	userID := uuid.NewUUID()

	// Act & Assert: Medium → High → Critical → Low
	result1, err := priorityUseCase.Execute(context.Background(), taskusecase.ChangePriorityCommand{
		TaskID:    createResult.TaskID,
		Priority:  task.PriorityHigh,
		ChangedBy: userID,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result1.Version)

	result2, err := priorityUseCase.Execute(context.Background(), taskusecase.ChangePriorityCommand{
		TaskID:    createResult.TaskID,
		Priority:  task.PriorityCritical,
		ChangedBy: userID,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, result2.Version)

	result3, err := priorityUseCase.Execute(context.Background(), taskusecase.ChangePriorityCommand{
		TaskID:    createResult.TaskID,
		Priority:  task.PriorityLow,
		ChangedBy: userID,
	})
	require.NoError(t, err)
	assert.Equal(t, 4, result3.Version)

	// Проверяем полную историю
	storedEvents, err := store.LoadEvents(context.Background(), createResult.TaskID.String())
	require.NoError(t, err)
	assert.Len(t, storedEvents, 4) // Created + 3x PriorityChanged
}

func TestChangePriorityUseCase_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		cmd         taskusecase.ChangePriorityCommand
		expectedErr error
	}{
		{
			name: "Empty TaskID",
			cmd: taskusecase.ChangePriorityCommand{
				TaskID:    uuid.UUID(""),
				Priority:  task.PriorityHigh,
				ChangedBy: uuid.NewUUID(),
			},
			expectedErr: taskusecase.ErrInvalidTaskID,
		},
		{
			name: "Empty Priority",
			cmd: taskusecase.ChangePriorityCommand{
				TaskID:    uuid.NewUUID(),
				Priority:  "",
				ChangedBy: uuid.NewUUID(),
			},
			expectedErr: taskusecase.ErrEmptyPriority,
		},
		{
			name: "Invalid Priority",
			cmd: taskusecase.ChangePriorityCommand{
				TaskID:    uuid.NewUUID(),
				Priority:  "Urgent",
				ChangedBy: uuid.NewUUID(),
			},
			expectedErr: taskusecase.ErrInvalidPriority,
		},
		{
			name: "Case Sensitive - lowercase",
			cmd: taskusecase.ChangePriorityCommand{
				TaskID:    uuid.NewUUID(),
				Priority:  "high",
				ChangedBy: uuid.NewUUID(),
			},
			expectedErr: taskusecase.ErrInvalidPriority,
		},
		{
			name: "Case Sensitive - uppercase",
			cmd: taskusecase.ChangePriorityCommand{
				TaskID:    uuid.NewUUID(),
				Priority:  "HIGH",
				ChangedBy: uuid.NewUUID(),
			},
			expectedErr: taskusecase.ErrInvalidPriority,
		},
		{
			name: "Empty ChangedBy",
			cmd: taskusecase.ChangePriorityCommand{
				TaskID:    uuid.NewUUID(),
				Priority:  task.PriorityHigh,
				ChangedBy: uuid.UUID(""),
			},
			expectedErr: taskusecase.ErrInvalidUserID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			store := eventstore.NewInMemoryEventStore()
			useCase := taskusecase.NewChangePriorityUseCase(store)

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
	useCase := taskusecase.NewChangePriorityUseCase(store)

	cmd := taskusecase.ChangePriorityCommand{
		TaskID:    uuid.NewUUID(), // не существует
		Priority:  task.PriorityHigh,
		ChangedBy: uuid.NewUUID(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, taskusecase.ErrTaskNotFound)
	assert.Empty(t, result.Events)
}
