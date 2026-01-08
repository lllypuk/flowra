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
	"github.com/lllypuk/flowra/tests/mocks"
)

func TestAssignTaskUseCase_Success(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	userRepo := mocks.NewMockUserRepository()

	createUseCase := taskapp.NewCreateTaskUseCase(store)
	assignUseCase := taskapp.NewAssignTaskUseCase(store, userRepo)

	// Creating task
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Creating user in моке
	assigneeID := uuid.NewUUID()
	userRepo.AddUser(assigneeID, "alice", "Alice Smith")

	// Наvalueаем исполнителя
	assignerID := uuid.NewUUID()
	assignCmd := taskapp.AssignTaskCommand{
		TaskID:     createResult.TaskID,
		AssigneeID: &assigneeID,
		AssignedBy: assignerID,
	}

	// Act
	result, err := assignUseCase.Execute(context.Background(), assignCmd)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, createResult.TaskID, result.TaskID)
	assert.Equal(t, 2, result.Version)
	require.Len(t, result.Events, 1)

	// Checking event
	event, ok := result.Events[0].(*task.AssigneeChanged)
	require.True(t, ok, "Expected *task.AssigneeChanged event")
	assert.Equal(t, createResult.TaskID, uuid.UUID(event.AggregateID()))
	assert.nil(t, event.OldAssignee)
	assert.Equal(t, &assigneeID, event.NewAssignee)
	assert.Equal(t, assignerID, event.ChangedBy)
}

func TestAssignTaskUseCase_Unassign(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	userRepo := mocks.NewMockUserRepository()

	createUseCase := taskapp.NewCreateTaskUseCase(store)
	assignUseCase := taskapp.NewAssignTaskUseCase(store, userRepo)

	// Creating task с assignee
	assigneeID := uuid.NewUUID()
	userRepo.AddUser(assigneeID, "bob", "Bob Johnson")

	createCmd := taskapp.CreateTaskCommand{
		ChatID:     uuid.NewUUID(),
		Title:      "Test Task",
		AssigneeID: &assigneeID,
		CreatedBy:  uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: Снимаем assignee (nil)
	unassignCmd := taskapp.AssignTaskCommand{
		TaskID:     createResult.TaskID,
		AssigneeID: nil, // снятие
		AssignedBy: uuid.NewUUID(),
	}
	result, err := assignUseCase.Execute(context.Background(), unassignCmd)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 2, result.Version)
	require.Len(t, result.Events, 1)

	event, ok := result.Events[0].(*task.AssigneeChanged)
	require.True(t, ok)
	assert.Equal(t, &assigneeID, event.OldAssignee)
	assert.nil(t, event.NewAssignee)
}

func TestAssignTaskUseCase_Reassign(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	userRepo := mocks.NewMockUserRepository()

	createUseCase := taskapp.NewCreateTaskUseCase(store)
	assignUseCase := taskapp.NewAssignTaskUseCase(store, userRepo)

	// Creating двух users
	alice := uuid.NewUUID()
	bob := uuid.NewUUID()
	userRepo.AddUser(alice, "alice", "Alice")
	userRepo.AddUser(bob, "bob", "Bob")

	// Creating task, наvalueенную on Alice
	createCmd := taskapp.CreateTaskCommand{
		ChatID:     uuid.NewUUID(),
		Title:      "Test Task",
		AssigneeID: &alice,
		CreatedBy:  uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: Перенаvalueаем on Bob
	reassignCmd := taskapp.AssignTaskCommand{
		TaskID:     createResult.TaskID,
		AssigneeID: &bob,
		AssignedBy: uuid.NewUUID(),
	}
	result, err := assignUseCase.Execute(context.Background(), reassignCmd)

	// Assert
	require.NoError(t, err)
	require.Len(t, result.Events, 1)

	event, ok := result.Events[0].(*task.AssigneeChanged)
	require.True(t, ok)
	assert.Equal(t, &alice, event.OldAssignee)
	assert.Equal(t, &bob, event.NewAssignee)
}

func TestAssignTaskUseCase_Idempotent(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	userRepo := mocks.NewMockUserRepository()

	createUseCase := taskapp.NewCreateTaskUseCase(store)
	assignUseCase := taskapp.NewAssignTaskUseCase(store, userRepo)

	assigneeID := uuid.NewUUID()
	userRepo.AddUser(assigneeID, "alice", "Alice")

	// Creating task, уже наvalueенную on Alice
	createCmd := taskapp.CreateTaskCommand{
		ChatID:     uuid.NewUUID(),
		Title:      "Test Task",
		AssigneeID: &assigneeID,
		CreatedBy:  uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: Повторно наvalueаем on Alice
	assignCmd := taskapp.AssignTaskCommand{
		TaskID:     createResult.TaskID,
		AssigneeID: &assigneeID,
		AssignedBy: uuid.NewUUID(),
	}
	result, err := assignUseCase.Execute(context.Background(), assignCmd)

	// Assert: not должно быть New events
	require.NoError(t, err)
	assert.Empty(t, result.Events, "Should not generate event for idempotent operation")
	assert.Equal(t, 1, result.Version, "Version should not change")
	assert.True(t, result.IsSuccess())
	assert.Equal(t, "Assignee unchanged (idempotent operation)", result.Message)
}

func TestAssignTaskUseCase_IdempotentUnassign(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	userRepo := mocks.NewMockUserRepository()

	createUseCase := taskapp.NewCreateTaskUseCase(store)
	assignUseCase := taskapp.NewAssignTaskUseCase(store, userRepo)

	// Creating task без assignee
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: Пытаемся снять assignee, when его no
	unassignCmd := taskapp.AssignTaskCommand{
		TaskID:     createResult.TaskID,
		AssigneeID: nil,
		AssignedBy: uuid.NewUUID(),
	}
	result, err := assignUseCase.Execute(context.Background(), unassignCmd)

	// Assert: not должно быть New events
	require.NoError(t, err)
	assert.Empty(t, result.Events)
	assert.Equal(t, 1, result.Version)
}

func TestAssignTaskUseCase_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*mocks.MockUserRepository)
		cmd         taskapp.AssignTaskCommand
		expectedErr error
	}{
		{
			name:      "Empty TaskID",
			setupMock: func(_ *mocks.MockUserRepository) {},
			cmd: taskapp.AssignTaskCommand{
				TaskID:     uuid.UUID(""),
				AssigneeID: ptr(uuid.NewUUID()),
				AssignedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrInvalidTaskID,
		},
		{
			name:      "Empty AssignedBy",
			setupMock: func(_ *mocks.MockUserRepository) {},
			cmd: taskapp.AssignTaskCommand{
				TaskID:     uuid.NewUUID(),
				AssigneeID: ptr(uuid.NewUUID()),
				AssignedBy: uuid.UUID(""),
			},
			expectedErr: taskapp.ErrInvalidUserID,
		},
		{
			name: "User Not Found",
			setupMock: func(_ *mocks.MockUserRepository) {
				// not добавляем user
			},
			cmd: taskapp.AssignTaskCommand{
				TaskID:     uuid.NewUUID(),
				AssigneeID: ptr(uuid.NewUUID()),
				AssignedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			store := eventstore.NewInMemoryEventStore()
			userRepo := mocks.NewMockUserRepository()
			tt.setupMock(userRepo)

			useCase := taskapp.NewAssignTaskUseCase(store, userRepo)

			// Act
			result, err := useCase.Execute(context.Background(), tt.cmd)

			// Assert
			require.Error(t, err)
			require.ErrorIs(t, err, tt.expectedErr)
			assert.Empty(t, result.Events)
		})
	}
}

func TestAssignTaskUseCase_TaskNotFound(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	userRepo := mocks.NewMockUserRepository()

	assigneeID := uuid.NewUUID()
	userRepo.AddUser(assigneeID, "alice", "Alice")

	useCase := taskapp.NewAssignTaskUseCase(store, userRepo)

	cmd := taskapp.AssignTaskCommand{
		TaskID:     uuid.NewUUID(), // not существует
		AssigneeID: &assigneeID,
		AssignedBy: uuid.NewUUID(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, taskapp.ErrTaskNotFound)
	assert.Empty(t, result.Events)
}

func TestAssignTaskUseCase_MultipleReassignments(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	userRepo := mocks.NewMockUserRepository()

	createUseCase := taskapp.NewCreateTaskUseCase(store)
	assignUseCase := taskapp.NewAssignTaskUseCase(store, userRepo)

	// Creating трех users
	alice := uuid.NewUUID()
	bob := uuid.NewUUID()
	charlie := uuid.NewUUID()
	userRepo.AddUser(alice, "alice", "Alice")
	userRepo.AddUser(bob, "bob", "Bob")
	userRepo.AddUser(charlie, "charlie", "Charlie")

	// Creating task без assignee
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	managerID := uuid.NewUUID()

	// Act & Assert: nil → Alice
	result1, err := assignUseCase.Execute(context.Background(), taskapp.AssignTaskCommand{
		TaskID:     createResult.TaskID,
		AssigneeID: &alice,
		AssignedBy: managerID,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result1.Version)

	// Alice → Bob
	result2, err := assignUseCase.Execute(context.Background(), taskapp.AssignTaskCommand{
		TaskID:     createResult.TaskID,
		AssigneeID: &bob,
		AssignedBy: managerID,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, result2.Version)

	// Bob → Charlie
	result3, err := assignUseCase.Execute(context.Background(), taskapp.AssignTaskCommand{
		TaskID:     createResult.TaskID,
		AssigneeID: &charlie,
		AssignedBy: managerID,
	})
	require.NoError(t, err)
	assert.Equal(t, 4, result3.Version)

	// Charlie → nil
	result4, err := assignUseCase.Execute(context.Background(), taskapp.AssignTaskCommand{
		TaskID:     createResult.TaskID,
		AssigneeID: nil,
		AssignedBy: managerID,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, result4.Version)

	// Checking full history
	storedEvents, err := store.LoadEvents(context.Background(), createResult.TaskID.String())
	require.NoError(t, err)
	assert.Len(t, storedEvents, 5) // Created + 4x AssigneeChanged
}

func TestAssignTaskUseCase_UnassignValidation(t *testing.T) {
	// Arrange
	store := eventstore.NewInMemoryEventStore()
	userRepo := mocks.NewMockUserRepository()

	createUseCase := taskapp.NewCreateTaskUseCase(store)
	assignUseCase := taskapp.NewAssignTaskUseCase(store, userRepo)

	assigneeID := uuid.NewUUID()
	userRepo.AddUser(assigneeID, "alice", "Alice")

	createCmd := taskapp.CreateTaskCommand{
		ChatID:     uuid.NewUUID(),
		Title:      "Test Task",
		AssigneeID: &assigneeID,
		CreatedBy:  uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: Снимаем assignee - not требуется validation user
	unassignCmd := taskapp.AssignTaskCommand{
		TaskID:     createResult.TaskID,
		AssigneeID: nil,
		AssignedBy: uuid.NewUUID(),
	}
	result, err := assignUseCase.Execute(context.Background(), unassignCmd)

	// Assert: Должно пройти successfully без проверки существования user
	require.NoError(t, err)
	assert.Len(t, result.Events, 1)
}
