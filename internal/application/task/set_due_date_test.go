package task_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	taskapp "github.com/flowra/flowra/internal/application/task"
	"github.com/flowra/flowra/internal/domain/task"
	"github.com/flowra/flowra/internal/domain/uuid"
	"github.com/flowra/flowra/internal/infrastructure/eventstore"
)

func TestSetDueDateUseCase_Success(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	createUseCase := taskapp.NewCreateTaskUseCase(store)
	dueDateUseCase := taskapp.NewSetDueDateUseCase(store)

	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Устанавливаем дедлайн
	dueDate := time.Now().Add(7 * 24 * time.Hour) // через неделю
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
	assert.Equal(t, dueDate.Unix(), event.NewDueDate.Unix())
	assert.Equal(t, userID, event.ChangedBy)
}

func TestSetDueDateUseCase_RemoveDueDate(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	createUseCase := taskapp.NewCreateTaskUseCase(store)
	dueDateUseCase := taskapp.NewSetDueDateUseCase(store)

	// Создаем задачу с дедлайном
	dueDate := time.Now().Add(7 * 24 * time.Hour)
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		DueDate:   &dueDate,
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: Снимаем дедлайн (nil)
	dueDateCmd := taskapp.SetDueDateCommand{
		TaskID:    createResult.TaskID,
		DueDate:   nil,
		ChangedBy: uuid.NewUUID(),
	}
	result, err := dueDateUseCase.Execute(context.Background(), dueDateCmd)

	// Assert
	require.NoError(t, err)
	require.Len(t, result.Events, 1)

	event, ok := result.Events[0].(*task.DueDateChanged)
	require.True(t, ok)
	assert.NotNil(t, event.OldDueDate)
	assert.Nil(t, event.NewDueDate)
}

func TestSetDueDateUseCase_Idempotent(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	createUseCase := taskapp.NewCreateTaskUseCase(store)
	dueDateUseCase := taskapp.NewSetDueDateUseCase(store)

	dueDate := time.Now().Add(7 * 24 * time.Hour)
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		DueDate:   &dueDate,
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: Повторная установка той же даты
	dueDateCmd := taskapp.SetDueDateCommand{
		TaskID:    createResult.TaskID,
		DueDate:   &dueDate,
		ChangedBy: uuid.NewUUID(),
	}
	result, err := dueDateUseCase.Execute(context.Background(), dueDateCmd)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, result.Events, "No new events for idempotent operation")
	assert.Equal(t, 1, result.Version, "Version should not change")
	assert.True(t, result.IsSuccess())
	assert.Equal(t, "Due date unchanged (idempotent operation)", result.Message)
}

func TestSetDueDateUseCase_IdempotentRemove(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	createUseCase := taskapp.NewCreateTaskUseCase(store)
	dueDateUseCase := taskapp.NewSetDueDateUseCase(store)

	// Создаем задачу без дедлайна
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: Пытаемся снять дедлайн, когда его нет
	dueDateCmd := taskapp.SetDueDateCommand{
		TaskID:    createResult.TaskID,
		DueDate:   nil,
		ChangedBy: uuid.NewUUID(),
	}
	result, err := dueDateUseCase.Execute(context.Background(), dueDateCmd)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, result.Events)
	assert.Equal(t, 1, result.Version)
}

func TestSetDueDateUseCase_PastDate(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	createUseCase := taskapp.NewCreateTaskUseCase(store)
	dueDateUseCase := taskapp.NewSetDueDateUseCase(store)

	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: Устанавливаем дату в прошлом (просроченная задача)
	pastDate := time.Now().Add(-7 * 24 * time.Hour)
	dueDateCmd := taskapp.SetDueDateCommand{
		TaskID:    createResult.TaskID,
		DueDate:   &pastDate,
		ChangedBy: uuid.NewUUID(),
	}
	result, err := dueDateUseCase.Execute(context.Background(), dueDateCmd)

	// Assert: Должно быть успешно (дата в прошлом допустима)
	require.NoError(t, err)
	assert.Len(t, result.Events, 1)

	event, ok := result.Events[0].(*task.DueDateChanged)
	require.True(t, ok)
	assert.Equal(t, pastDate.Unix(), event.NewDueDate.Unix())
}

func TestSetDueDateUseCase_MultipleDateChanges(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	createUseCase := taskapp.NewCreateTaskUseCase(store)
	dueDateUseCase := taskapp.NewSetDueDateUseCase(store)

	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	userID := uuid.NewUUID()

	// Act & Assert: nil → date1 → date2 → nil
	date1 := time.Now().Add(7 * 24 * time.Hour)
	result1, err := dueDateUseCase.Execute(context.Background(), taskapp.SetDueDateCommand{
		TaskID:    createResult.TaskID,
		DueDate:   &date1,
		ChangedBy: userID,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result1.Version)

	date2 := time.Now().Add(14 * 24 * time.Hour)
	result2, err := dueDateUseCase.Execute(context.Background(), taskapp.SetDueDateCommand{
		TaskID:    createResult.TaskID,
		DueDate:   &date2,
		ChangedBy: userID,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, result2.Version)

	result3, err := dueDateUseCase.Execute(context.Background(), taskapp.SetDueDateCommand{
		TaskID:    createResult.TaskID,
		DueDate:   nil,
		ChangedBy: userID,
	})
	require.NoError(t, err)
	assert.Equal(t, 4, result3.Version)

	// Проверяем полную историю
	storedEvents, err := store.LoadEvents(context.Background(), createResult.TaskID.String())
	require.NoError(t, err)
	assert.Len(t, storedEvents, 4) // Created + 3x DueDateChanged
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
				DueDate:   ptr(time.Now()),
				ChangedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrInvalidTaskID,
		},
		{
			name: "Date Too Far in Past",
			cmd: taskapp.SetDueDateCommand{
				TaskID:    uuid.NewUUID(),
				DueDate:   ptr(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)),
				ChangedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrInvalidDate,
		},
		{
			name: "Empty ChangedBy",
			cmd: taskapp.SetDueDateCommand{
				TaskID:    uuid.NewUUID(),
				DueDate:   ptr(time.Now()),
				ChangedBy: uuid.UUID(""),
			},
			expectedErr: taskapp.ErrInvalidUserID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			store := eventstore.NewInMemoryEventStore()
			useCase := taskapp.NewSetDueDateUseCase(store)

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
	store := eventstore.NewInMemoryEventStore()
	useCase := taskapp.NewSetDueDateUseCase(store)

	dueDate := time.Now().Add(7 * 24 * time.Hour)
	cmd := taskapp.SetDueDateCommand{
		TaskID:    uuid.NewUUID(), // не существует
		DueDate:   &dueDate,
		ChangedBy: uuid.NewUUID(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, taskapp.ErrTaskNotFound)
	assert.Empty(t, result.Events)
}
