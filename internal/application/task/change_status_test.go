package task_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	taskapp "github.com/flowra/flowra/internal/application/task"
	"github.com/flowra/flowra/internal/domain/task"
	"github.com/flowra/flowra/internal/domain/uuid"
	"github.com/flowra/flowra/internal/infrastructure/eventstore"
)

func TestChangeStatusUseCase_Success(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	createUseCase := taskapp.NewCreateTaskUseCase(store)
	changeStatusUseCase := taskapp.NewChangeStatusUseCase(store)

	// Создаем задачу
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Меняем статус
	userID := uuid.NewUUID()
	changeCmd := taskapp.ChangeStatusCommand{
		TaskID:    createResult.TaskID,
		NewStatus: task.StatusInProgress,
		ChangedBy: userID,
	}

	// Act
	result, err := changeStatusUseCase.Execute(context.Background(), changeCmd)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, createResult.TaskID, result.TaskID)
	assert.Equal(t, 2, result.Version) // 1 событие создания + 1 событие изменения статуса
	require.Len(t, result.Events, 1)

	// Проверяем событие
	event, ok := result.Events[0].(*task.StatusChanged)
	require.True(t, ok, "Expected *task.StatusChanged event")
	assert.Equal(t, createResult.TaskID, uuid.UUID(event.AggregateID()))
	assert.Equal(t, task.StatusToDo, event.OldStatus)
	assert.Equal(t, task.StatusInProgress, event.NewStatus)
	assert.Equal(t, userID, event.ChangedBy)

	// Проверяем, что события сохранены
	storedEvents, err := store.LoadEvents(context.Background(), result.TaskID.String())
	require.NoError(t, err)
	assert.Len(t, storedEvents, 2)
}

func TestChangeStatusUseCase_MultipleTransitions(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	createUseCase := taskapp.NewCreateTaskUseCase(store)
	changeStatusUseCase := taskapp.NewChangeStatusUseCase(store)

	// Создаем задачу
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	userID := uuid.NewUUID()

	// Act & Assert
	// To Do → In Progress
	result1, err := changeStatusUseCase.Execute(context.Background(), taskapp.ChangeStatusCommand{
		TaskID:    createResult.TaskID,
		NewStatus: task.StatusInProgress,
		ChangedBy: userID,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result1.Version)

	// In Progress → In Review
	result2, err := changeStatusUseCase.Execute(context.Background(), taskapp.ChangeStatusCommand{
		TaskID:    createResult.TaskID,
		NewStatus: task.StatusInReview,
		ChangedBy: userID,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, result2.Version)

	// In Review → Done
	result3, err := changeStatusUseCase.Execute(context.Background(), taskapp.ChangeStatusCommand{
		TaskID:    createResult.TaskID,
		NewStatus: task.StatusDone,
		ChangedBy: userID,
	})
	require.NoError(t, err)
	assert.Equal(t, 4, result3.Version)

	// Проверяем полную историю
	storedEvents, err := store.LoadEvents(context.Background(), createResult.TaskID.String())
	require.NoError(t, err)
	assert.Len(t, storedEvents, 4) // Create + 3x StatusChanged
}

func TestChangeStatusUseCase_Idempotent(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	createUseCase := taskapp.NewCreateTaskUseCase(store)
	changeStatusUseCase := taskapp.NewChangeStatusUseCase(store)

	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Первое изменение статуса
	changeCmd := taskapp.ChangeStatusCommand{
		TaskID:    createResult.TaskID,
		NewStatus: task.StatusInProgress,
		ChangedBy: uuid.NewUUID(),
	}
	result1, err := changeStatusUseCase.Execute(context.Background(), changeCmd)
	require.NoError(t, err)
	assert.Len(t, result1.Events, 1)

	// Act: Повторное изменение на тот же статус
	result2, err := changeStatusUseCase.Execute(context.Background(), changeCmd)

	// Assert: Должно быть успешно, но без новых событий
	require.NoError(t, err)
	assert.Empty(t, result2.Events, "No new events should be generated for idempotent operation")
	assert.Equal(t, result1.Version, result2.Version, "Version should not change")
	assert.True(t, result2.IsSuccess())
	assert.Equal(t, "Status unchanged (idempotent operation)", result2.Message)
}

func TestChangeStatusUseCase_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		cmd         taskapp.ChangeStatusCommand
		expectedErr error
	}{
		{
			name: "Empty TaskID",
			cmd: taskapp.ChangeStatusCommand{
				TaskID:    uuid.UUID(""),
				NewStatus: task.StatusDone,
				ChangedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrInvalidTaskID,
		},
		{
			name: "Empty Status",
			cmd: taskapp.ChangeStatusCommand{
				TaskID:    uuid.NewUUID(),
				NewStatus: "",
				ChangedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrInvalidStatus,
		},
		{
			name: "Invalid Status",
			cmd: taskapp.ChangeStatusCommand{
				TaskID:    uuid.NewUUID(),
				NewStatus: "Completed", // не существует
				ChangedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrInvalidStatus,
		},
		{
			name: "Empty ChangedBy",
			cmd: taskapp.ChangeStatusCommand{
				TaskID:    uuid.NewUUID(),
				NewStatus: task.StatusDone,
				ChangedBy: uuid.UUID(""),
			},
			expectedErr: taskapp.ErrInvalidUserID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			store := eventstore.NewInMemoryEventStore()
			useCase := taskapp.NewChangeStatusUseCase(store)

			// Act
			result, err := useCase.Execute(context.Background(), tt.cmd)

			// Assert
			require.Error(t, err)
			require.ErrorIs(t, err, tt.expectedErr)
			assert.Empty(t, result.Events)
		})
	}
}

func TestChangeStatusUseCase_TaskNotFound(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	useCase := taskapp.NewChangeStatusUseCase(store)

	cmd := taskapp.ChangeStatusCommand{
		TaskID:    uuid.NewUUID(), // не существует
		NewStatus: task.StatusDone,
		ChangedBy: uuid.NewUUID(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, taskapp.ErrTaskNotFound)
	assert.Empty(t, result.Events)
}

func TestChangeStatusUseCase_InvalidStatusTransition(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	createUseCase := taskapp.NewCreateTaskUseCase(store)
	changeStatusUseCase := taskapp.NewChangeStatusUseCase(store)

	// Создаем задачу в статусе To Do
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: Пытаемся перейти из To Do сразу в Done (невалидный переход)
	changeCmd := taskapp.ChangeStatusCommand{
		TaskID:    createResult.TaskID,
		NewStatus: task.StatusDone,
		ChangedBy: uuid.NewUUID(),
	}
	result, err := changeStatusUseCase.Execute(context.Background(), changeCmd)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, taskapp.ErrInvalidStatusTransition)
	assert.Empty(t, result.Events)
}

func TestChangeStatusUseCase_AllValidTransitions(t *testing.T) {
	tests := []struct {
		name       string
		from       task.Status
		to         task.Status
		shouldPass bool
	}{
		// Из To Do
		{"To Do → In Progress", task.StatusToDo, task.StatusInProgress, true},
		{"To Do → Backlog", task.StatusToDo, task.StatusBacklog, true},
		{"To Do → Cancelled", task.StatusToDo, task.StatusCancelled, true},
		{"To Do → Done (invalid)", task.StatusToDo, task.StatusDone, false},

		// Из In Progress
		{"In Progress → In Review", task.StatusInProgress, task.StatusInReview, true},
		{"In Progress → To Do", task.StatusInProgress, task.StatusToDo, true},
		{"In Progress → Cancelled", task.StatusInProgress, task.StatusCancelled, true},
		{"In Progress → Done (invalid)", task.StatusInProgress, task.StatusDone, false},

		// Из In Review
		{"In Review → Done", task.StatusInReview, task.StatusDone, true},
		{"In Review → In Progress", task.StatusInReview, task.StatusInProgress, true},
		{"In Review → Cancelled", task.StatusInReview, task.StatusCancelled, true},
		{"In Review → To Do (invalid)", task.StatusInReview, task.StatusToDo, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			store := eventstore.NewInMemoryEventStore()
			createUseCase := taskapp.NewCreateTaskUseCase(store)
			changeStatusUseCase := taskapp.NewChangeStatusUseCase(store)

			// Создаем задачу
			createCmd := taskapp.CreateTaskCommand{
				ChatID:    uuid.NewUUID(),
				Title:     "Test Task",
				CreatedBy: uuid.NewUUID(),
			}
			createResult, err := createUseCase.Execute(context.Background(), createCmd)
			require.NoError(t, err)

			// Переводим задачу в нужный начальный статус
			if tt.from != task.StatusToDo {
				// Сначала переводим в валидный промежуточный статус
				switch tt.from { //nolint:exhaustive // Only testing specific transitions from In Progress and In Review
				case task.StatusInProgress:
					_, err = changeStatusUseCase.Execute(context.Background(), taskapp.ChangeStatusCommand{
						TaskID:    createResult.TaskID,
						NewStatus: task.StatusInProgress,
						ChangedBy: uuid.NewUUID(),
					})
					require.NoError(t, err)
				case task.StatusInReview:
					_, err = changeStatusUseCase.Execute(context.Background(), taskapp.ChangeStatusCommand{
						TaskID:    createResult.TaskID,
						NewStatus: task.StatusInProgress,
						ChangedBy: uuid.NewUUID(),
					})
					require.NoError(t, err)
					_, err = changeStatusUseCase.Execute(context.Background(), taskapp.ChangeStatusCommand{
						TaskID:    createResult.TaskID,
						NewStatus: task.StatusInReview,
						ChangedBy: uuid.NewUUID(),
					})
					require.NoError(t, err)
				}
			}

			// Act: Пытаемся выполнить проверяемый переход
			changeCmd := taskapp.ChangeStatusCommand{
				TaskID:    createResult.TaskID,
				NewStatus: tt.to,
				ChangedBy: uuid.NewUUID(),
			}
			result, err := changeStatusUseCase.Execute(context.Background(), changeCmd)

			// Assert
			if tt.shouldPass {
				require.NoError(t, err, "Expected transition to succeed")
				assert.Len(t, result.Events, 1, "Expected one event")
			} else {
				require.Error(t, err, "Expected transition to fail")
				require.ErrorIs(t, err, taskapp.ErrInvalidStatusTransition)
			}
		})
	}
}

func TestChangeStatusUseCase_Backlog(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	createUseCase := taskapp.NewCreateTaskUseCase(store)
	changeStatusUseCase := taskapp.NewChangeStatusUseCase(store)

	// Создаем задачу
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	userID := uuid.NewUUID()

	// Act & Assert: To Do → Backlog → To Do
	_, err = changeStatusUseCase.Execute(context.Background(), taskapp.ChangeStatusCommand{
		TaskID:    createResult.TaskID,
		NewStatus: task.StatusBacklog,
		ChangedBy: userID,
	})
	require.NoError(t, err)

	result, err := changeStatusUseCase.Execute(context.Background(), taskapp.ChangeStatusCommand{
		TaskID:    createResult.TaskID,
		NewStatus: task.StatusToDo,
		ChangedBy: userID,
	})
	require.NoError(t, err)
	assert.Len(t, result.Events, 1)

	event, ok := result.Events[0].(*task.StatusChanged)
	require.True(t, ok)
	assert.Equal(t, task.StatusBacklog, event.OldStatus)
	assert.Equal(t, task.StatusToDo, event.NewStatus)
}

func TestChangeStatusUseCase_CancelledTransition(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	createUseCase := taskapp.NewCreateTaskUseCase(store)
	changeStatusUseCase := taskapp.NewChangeStatusUseCase(store)

	// Создаем задачу
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	userID := uuid.NewUUID()

	// Act: To Do → Cancelled
	result1, err := changeStatusUseCase.Execute(context.Background(), taskapp.ChangeStatusCommand{
		TaskID:    createResult.TaskID,
		NewStatus: task.StatusCancelled,
		ChangedBy: userID,
	})
	require.NoError(t, err)
	assert.Len(t, result1.Events, 1)

	// Assert: Cancelled → Backlog (единственный валидный переход из Cancelled)
	result2, err := changeStatusUseCase.Execute(context.Background(), taskapp.ChangeStatusCommand{
		TaskID:    createResult.TaskID,
		NewStatus: task.StatusBacklog,
		ChangedBy: userID,
	})
	require.NoError(t, err)
	assert.Len(t, result2.Events, 1)

	// Cancelled → To Do должно быть невалидно
	_, err = changeStatusUseCase.Execute(context.Background(), taskapp.ChangeStatusCommand{
		TaskID:    createResult.TaskID,
		NewStatus: task.StatusCancelled,
		ChangedBy: userID,
	})
	require.NoError(t, err)

	_, err = changeStatusUseCase.Execute(context.Background(), taskapp.ChangeStatusCommand{
		TaskID:    createResult.TaskID,
		NewStatus: task.StatusToDo,
		ChangedBy: userID,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, taskapp.ErrInvalidStatusTransition)
}

func TestChangeStatusUseCase_DoneReopening(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	createUseCase := taskapp.NewCreateTaskUseCase(store)
	changeStatusUseCase := taskapp.NewChangeStatusUseCase(store)

	// Создаем задачу и доводим до Done
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	userID := uuid.NewUUID()

	// To Do → In Progress → In Review → Done
	_, err = changeStatusUseCase.Execute(context.Background(), taskapp.ChangeStatusCommand{
		TaskID:    createResult.TaskID,
		NewStatus: task.StatusInProgress,
		ChangedBy: userID,
	})
	require.NoError(t, err)

	_, err = changeStatusUseCase.Execute(context.Background(), taskapp.ChangeStatusCommand{
		TaskID:    createResult.TaskID,
		NewStatus: task.StatusInReview,
		ChangedBy: userID,
	})
	require.NoError(t, err)

	_, err = changeStatusUseCase.Execute(context.Background(), taskapp.ChangeStatusCommand{
		TaskID:    createResult.TaskID,
		NewStatus: task.StatusDone,
		ChangedBy: userID,
	})
	require.NoError(t, err)

	// Act: Done → In Review (reopening)
	result, err := changeStatusUseCase.Execute(context.Background(), taskapp.ChangeStatusCommand{
		TaskID:    createResult.TaskID,
		NewStatus: task.StatusInReview,
		ChangedBy: userID,
	})

	// Assert
	require.NoError(t, err)
	assert.Len(t, result.Events, 1)

	event, ok := result.Events[0].(*task.StatusChanged)
	require.True(t, ok)
	assert.Equal(t, task.StatusDone, event.OldStatus)
	assert.Equal(t, task.StatusInReview, event.NewStatus)
}
