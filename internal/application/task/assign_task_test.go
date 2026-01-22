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

func TestAssignTaskUseCase_Success(t *testing.T) {
	// Arrange
	repo := mocks.NewMockTaskRepository()
	userRepo := mocks.NewMockUserRepository()

	createUseCase := taskapp.NewCreateTaskUseCase(repo)
	assignUseCase := taskapp.NewAssignTaskUseCase(repo, userRepo)

	// Create task
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Create user in mock
	assigneeID := uuid.NewUUID()
	userRepo.AddUser(assigneeID, "alice", "Alice Smith")

	// Assign user
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

	// Verify event
	event, ok := result.Events[0].(*task.AssigneeChanged)
	require.True(t, ok, "Expected *task.AssigneeChanged event")
	assert.Equal(t, createResult.TaskID, uuid.UUID(event.AggregateID()))
	assert.Nil(t, event.OldAssignee)
	assert.Equal(t, &assigneeID, event.NewAssignee)
	assert.Equal(t, assignerID, event.ChangedBy)
}

func TestAssignTaskUseCase_Unassign(t *testing.T) {
	// Arrange
	repo := mocks.NewMockTaskRepository()
	userRepo := mocks.NewMockUserRepository()

	createUseCase := taskapp.NewCreateTaskUseCase(repo)
	assignUseCase := taskapp.NewAssignTaskUseCase(repo, userRepo)

	// Create task with assignee
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

	// Act: unassign (nil)
	unassignCmd := taskapp.AssignTaskCommand{
		TaskID:     createResult.TaskID,
		AssigneeID: nil, // unassign
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
	assert.Nil(t, event.NewAssignee)
}

func TestAssignTaskUseCase_Reassign(t *testing.T) {
	// Arrange
	repo := mocks.NewMockTaskRepository()
	userRepo := mocks.NewMockUserRepository()

	createUseCase := taskapp.NewCreateTaskUseCase(repo)
	assignUseCase := taskapp.NewAssignTaskUseCase(repo, userRepo)

	// Create two users
	alice := uuid.NewUUID()
	bob := uuid.NewUUID()
	userRepo.AddUser(alice, "alice", "Alice")
	userRepo.AddUser(bob, "bob", "Bob")

	// Create task, assign to Alice
	createCmd := taskapp.CreateTaskCommand{
		ChatID:     uuid.NewUUID(),
		Title:      "Test Task",
		AssigneeID: &alice,
		CreatedBy:  uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: reassign from Alice to Bob
	reassignCmd := taskapp.AssignTaskCommand{
		TaskID:     createResult.TaskID,
		AssigneeID: &bob,
		AssignedBy: uuid.NewUUID(),
	}
	result, err := assignUseCase.Execute(context.Background(), reassignCmd)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 2, result.Version)
	require.Len(t, result.Events, 1)

	event, ok := result.Events[0].(*task.AssigneeChanged)
	require.True(t, ok)
	assert.Equal(t, &alice, event.OldAssignee)
	assert.Equal(t, &bob, event.NewAssignee)
}

func TestAssignTaskUseCase_Idempotent(t *testing.T) {
	// Arrange
	repo := mocks.NewMockTaskRepository()
	userRepo := mocks.NewMockUserRepository()

	createUseCase := taskapp.NewCreateTaskUseCase(repo)
	assignUseCase := taskapp.NewAssignTaskUseCase(repo, userRepo)

	assigneeID := uuid.NewUUID()
	userRepo.AddUser(assigneeID, "charlie", "Charlie")

	createCmd := taskapp.CreateTaskCommand{
		ChatID:     uuid.NewUUID(),
		Title:      "Test Task",
		AssigneeID: &assigneeID,
		CreatedBy:  uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: repeat the same assignment
	assignCmd := taskapp.AssignTaskCommand{
		TaskID:     createResult.TaskID,
		AssigneeID: &assigneeID, // same as before
		AssignedBy: uuid.NewUUID(),
	}
	result, err := assignUseCase.Execute(context.Background(), assignCmd)

	// Assert: should succeed but without new events
	require.NoError(t, err)
	assert.Empty(t, result.Events, "No new events should be generated for idempotent operation")
	assert.Equal(t, 1, result.Version, "Version should not change")
	assert.True(t, result.IsSuccess())
	assert.Equal(t, "Assignee unchanged (idempotent operation)", result.Message)
}

func TestAssignTaskUseCase_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		cmd         taskapp.AssignTaskCommand
		expectedErr error
	}{
		{
			name: "Empty TaskID",
			cmd: taskapp.AssignTaskCommand{
				TaskID:     uuid.UUID(""),
				AssigneeID: nil,
				AssignedBy: uuid.NewUUID(),
			},
			expectedErr: taskapp.ErrInvalidTaskID,
		},
		{
			name: "Empty AssignedBy",
			cmd: taskapp.AssignTaskCommand{
				TaskID:     uuid.NewUUID(),
				AssigneeID: nil,
				AssignedBy: uuid.UUID(""),
			},
			expectedErr: taskapp.ErrInvalidUserID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			repo := mocks.NewMockTaskRepository()
			userRepo := mocks.NewMockUserRepository()
			useCase := taskapp.NewAssignTaskUseCase(repo, userRepo)

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
	repo := mocks.NewMockTaskRepository()
	userRepo := mocks.NewMockUserRepository()
	useCase := taskapp.NewAssignTaskUseCase(repo, userRepo)

	cmd := taskapp.AssignTaskCommand{
		TaskID:     uuid.NewUUID(), // does not exist
		AssigneeID: nil,
		AssignedBy: uuid.NewUUID(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, taskapp.ErrTaskNotFound)
	assert.Empty(t, result.Events)
}

func TestAssignTaskUseCase_UserNotFound(t *testing.T) {
	// Arrange
	repo := mocks.NewMockTaskRepository()
	userRepo := mocks.NewMockUserRepository()

	createUseCase := taskapp.NewCreateTaskUseCase(repo)
	assignUseCase := taskapp.NewAssignTaskUseCase(repo, userRepo)

	// Create task
	createCmd := taskapp.CreateTaskCommand{
		ChatID:    uuid.NewUUID(),
		Title:     "Test Task",
		CreatedBy: uuid.NewUUID(),
	}
	createResult, err := createUseCase.Execute(context.Background(), createCmd)
	require.NoError(t, err)

	// Act: assign non-existent user
	nonExistentUser := uuid.NewUUID()
	assignCmd := taskapp.AssignTaskCommand{
		TaskID:     createResult.TaskID,
		AssigneeID: &nonExistentUser,
		AssignedBy: uuid.NewUUID(),
	}
	result, err := assignUseCase.Execute(context.Background(), assignCmd)

	// Assert
	require.Error(t, err)
	require.ErrorIs(t, err, taskapp.ErrUserNotFound)
	assert.Empty(t, result.Events)
}
