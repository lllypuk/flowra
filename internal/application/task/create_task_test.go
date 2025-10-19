package task_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	taskapp "github.com/lllypuk/teams-up/internal/application/task"
	"github.com/lllypuk/teams-up/internal/domain/task"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
	"github.com/lllypuk/teams-up/internal/infrastructure/eventstore"
)

func TestCreateTaskUseCase_Success(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	useCase := taskapp.NewCreateTaskUseCase(store)

	chatID := uuid.NewUUID()
	userID := uuid.NewUUID()
	assigneeID := uuid.NewUUID()
	dueDate := time.Now().Add(24 * time.Hour)

	cmd := taskapp.CreateTaskCommand{
		ChatID:     chatID,
		Title:      "Implement OAuth authentication",
		EntityType: task.TypeTask,
		Priority:   task.PriorityHigh,
		AssigneeID: &assigneeID,
		DueDate:    &dueDate,
		CreatedBy:  userID,
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	assert.True(t, result.IsSuccess())
	assert.False(t, result.TaskID.IsZero())
	assert.Equal(t, 1, result.Version)
	require.Len(t, result.Events, 1)

	// Проверяем событие
	event, ok := result.Events[0].(*task.Created)
	require.True(t, ok, "Expected *task.Created event")
	assert.Equal(t, chatID, event.ChatID)
	assert.Equal(t, "Implement OAuth authentication", event.Title)
	assert.Equal(t, task.TypeTask, event.EntityType)
	assert.Equal(t, task.StatusToDo, event.Status)
	assert.Equal(t, task.PriorityHigh, event.Priority)
	assert.Equal(t, &assigneeID, event.AssigneeID)
	assert.NotNil(t, event.DueDate)
	assert.Equal(t, userID, event.CreatedBy)

	// Проверяем, что события сохранены в Event Store
	storedEvents, err := store.LoadEvents(context.Background(), result.TaskID.String())
	require.NoError(t, err)
	assert.Len(t, storedEvents, 1)
}

func TestCreateTaskUseCase_WithDefaults(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	useCase := taskapp.NewCreateTaskUseCase(store)

	cmd := taskapp.CreateTaskCommand{
		ChatID: uuid.NewUUID(),
		Title:  "Simple task",
		// EntityType не указан - должен стать TypeTask
		// Priority не указан - должен стать PriorityMedium
		// AssigneeID не указан
		// DueDate не указан
		CreatedBy: uuid.NewUUID(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	assert.True(t, result.IsSuccess())

	event, ok := result.Events[0].(*task.Created)
	require.True(t, ok)
	assert.Equal(t, task.TypeTask, event.EntityType, "Default entity type should be TypeTask")
	assert.Equal(t, task.PriorityMedium, event.Priority, "Default priority should be Medium")
	assert.Nil(t, event.AssigneeID, "AssigneeID should be nil")
	assert.Nil(t, event.DueDate, "DueDate should be nil")
}

func TestCreateTaskUseCase_TitleTrimming(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	useCase := taskapp.NewCreateTaskUseCase(store)

	cmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "   Task with spaces   ",
		CreatedBy: uuid.NewUUID(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	require.NoError(t, err)

	event, ok := result.Events[0].(*task.Created)
	require.True(t, ok)
	assert.Equal(t, "Task with spaces", event.Title, "Title should be trimmed")
}

func TestCreateTaskUseCase_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		cmd         taskapp.CreateTaskCommand
		expectedErr error
	}{
		{
			name: "Empty ChatID",
			cmd: taskapp.CreateTaskCommand{
				ChatID:    uuid.UUID(""),
				Title:     "Test",
				CreatedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrInvalidChatID,
		},
		{
			name: "Empty Title",
			cmd: taskapp.CreateTaskCommand{
				ChatID:    uuid.NewUUID(),
				Title:     "",
				CreatedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrEmptyTitle,
		},
		{
			name: "Whitespace-only Title",
			cmd: taskapp.CreateTaskCommand{
				ChatID:    uuid.NewUUID(),
				Title:     "   ",
				CreatedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrEmptyTitle,
		},
		{
			name: "Title too long",
			cmd: taskapp.CreateTaskCommand{
				ChatID:    uuid.NewUUID(),
				Title:     string(make([]byte, 501)), // 501 символ
				CreatedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrInvalidTitle,
		},
		{
			name: "Invalid EntityType",
			cmd: taskapp.CreateTaskCommand{
				ChatID:     uuid.NewUUID(),
				Title:      "Test",
				EntityType: "invalid",
				CreatedBy:  uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrInvalidEntityType,
		},
		{
			name: "Invalid Priority",
			cmd: taskapp.CreateTaskCommand{
				ChatID:    uuid.NewUUID(),
				Title:     "Test",
				Priority:  "Urgent", // не существует
				CreatedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrInvalidPriority,
		},
		{
			name: "Empty CreatedBy",
			cmd: taskapp.CreateTaskCommand{
				ChatID:    uuid.NewUUID(),
				Title:     "Test",
				CreatedBy: uuid.UUID(""),
			},
			expectedErr: taskapp.ErrInvalidUserID,
		},
		{
			name: "Date in far past",
			cmd: taskapp.CreateTaskCommand{
				ChatID:    uuid.NewUUID(),
				Title:     "Test",
				DueDate:   ptr(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)),
				CreatedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrInvalidDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			store := eventstore.NewInMemoryEventStore()
			useCase := taskapp.NewCreateTaskUseCase(store)

			// Act
			result, err := useCase.Execute(context.Background(), tt.cmd)

			// Assert
			require.Error(t, err)
			require.ErrorIs(t, err, tt.expectedErr)
			assert.True(t, result.TaskID.IsZero())
			assert.Empty(t, result.Events)
			assert.False(t, result.IsSuccess())
		})
	}
}

func TestCreateTaskUseCase_AllEntityTypes(t *testing.T) {
	tests := []struct {
		name       string
		entityType task.EntityType
	}{
		{"Task", task.TypeTask},
		{"Bug", task.TypeBug},
		{"Epic", task.TypeEpic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			store := eventstore.NewInMemoryEventStore()
			useCase := taskapp.NewCreateTaskUseCase(store)

			cmd := taskapp.CreateTaskCommand{
				ChatID:     uuid.NewUUID(),
				Title:      "Test " + tt.name,
				EntityType: tt.entityType,
				CreatedBy:  uuid.NewUUID(),
			}

			// Act
			result, err := useCase.Execute(context.Background(), cmd)

			// Assert
			require.NoError(t, err)
			event, ok := result.Events[0].(*task.Created)
			require.True(t, ok)
			assert.Equal(t, tt.entityType, event.EntityType)
		})
	}
}

func TestCreateTaskUseCase_AllPriorities(t *testing.T) {
	tests := []struct {
		name     string
		priority task.Priority
	}{
		{"Low", task.PriorityLow},
		{"Medium", task.PriorityMedium},
		{"High", task.PriorityHigh},
		{"Critical", task.PriorityCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			store := eventstore.NewInMemoryEventStore()
			useCase := taskapp.NewCreateTaskUseCase(store)

			cmd := taskapp.CreateTaskCommand{
				ChatID:    uuid.NewUUID(),
				Title:     "Test priority",
				Priority:  tt.priority,
				CreatedBy: uuid.NewUUID(),
			}

			// Act
			result, err := useCase.Execute(context.Background(), cmd)

			// Assert
			require.NoError(t, err)
			event, ok := result.Events[0].(*task.Created)
			require.True(t, ok)
			assert.Equal(t, tt.priority, event.Priority)
		})
	}
}

func TestCreateTaskUseCase_InitialStatusIsAlwaysToDo(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	useCase := taskapp.NewCreateTaskUseCase(store)

	cmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test task",
		CreatedBy: uuid.NewUUID(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	event, ok := result.Events[0].(*task.Created)
	require.True(t, ok)
	assert.Equal(t, task.StatusToDo, event.Status, "Initial status must always be To Do")
}

// Helper function
func ptr[T any](v T) *T {
	return &v
}
