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

// setupTaskWithAttachment creates a task aggregate in the repo and returns IDs.
func setupTaskWithAttachment(t *testing.T, repo *mocks.MockTaskRepository) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	taskID := uuid.NewUUID()
	fileID := uuid.NewUUID()
	userID := uuid.NewUUID()

	agg := task.NewTaskAggregate(taskID)
	err := agg.Create(uuid.NewUUID(), "Test Task", task.TypeTask, task.PriorityMedium, nil, nil, userID)
	require.NoError(t, err)
	err = agg.AddAttachment(fileID, "report.pdf", 1024, "application/pdf", userID)
	require.NoError(t, err)
	err = repo.Save(context.Background(), agg)
	require.NoError(t, err)
	return taskID, fileID, userID
}

func TestAddAttachmentUseCase_Success(t *testing.T) {
	// Arrange
	repo := mocks.NewMockTaskRepository()
	useCase := taskapp.NewAddAttachmentUseCase(repo)

	taskID := uuid.NewUUID()
	userID := uuid.NewUUID()
	agg := task.NewTaskAggregate(taskID)
	err := agg.Create(uuid.NewUUID(), "Test Task", task.TypeTask, task.PriorityMedium, nil, nil, userID)
	require.NoError(t, err)
	require.NoError(t, repo.Save(context.Background(), agg))

	fileID := uuid.NewUUID()
	cmd := taskapp.AddAttachmentCommand{
		TaskID:   taskID,
		FileID:   fileID,
		FileName: "design.png",
		FileSize: 2048,
		MimeType: "image/png",
		AddedBy:  userID,
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	assert.True(t, result.IsSuccess())
	assert.Equal(t, taskID, result.TaskID)
	require.Len(t, result.Events, 1)

	evt, ok := result.Events[0].(*task.AttachmentAdded)
	require.True(t, ok, "Expected *task.AttachmentAdded event")
	assert.Equal(t, fileID, evt.FileID)
	assert.Equal(t, "design.png", evt.FileName)
	assert.Equal(t, int64(2048), evt.FileSize)
	assert.Equal(t, "image/png", evt.MimeType)
	assert.Equal(t, userID, evt.AddedBy)

	assert.Equal(t, 2, repo.SaveCallCount()) // create + add
}

func TestAddAttachmentUseCase_Idempotent(t *testing.T) {
	// Arrange
	repo := mocks.NewMockTaskRepository()
	useCase := taskapp.NewAddAttachmentUseCase(repo)
	taskID, fileID, userID := setupTaskWithAttachment(t, repo)

	cmd := taskapp.AddAttachmentCommand{
		TaskID:   taskID,
		FileID:   fileID,
		FileName: "report.pdf",
		FileSize: 1024,
		MimeType: "application/pdf",
		AddedBy:  userID,
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	assert.True(t, result.IsSuccess())
	assert.Empty(t, result.Events, "idempotent call should produce no events")
}

func TestAddAttachmentUseCase_TaskNotFound(t *testing.T) {
	repo := mocks.NewMockTaskRepository()
	useCase := taskapp.NewAddAttachmentUseCase(repo)

	cmd := taskapp.AddAttachmentCommand{
		TaskID:   uuid.NewUUID(),
		FileID:   uuid.NewUUID(),
		FileName: "file.txt",
		FileSize: 100,
		MimeType: "text/plain",
		AddedBy:  uuid.NewUUID(),
	}

	_, err := useCase.Execute(context.Background(), cmd)
	require.ErrorIs(t, err, taskapp.ErrTaskNotFound)
}

func TestAddAttachmentUseCase_ValidationErrors(t *testing.T) {
	validCmd := taskapp.AddAttachmentCommand{
		TaskID:   uuid.NewUUID(),
		FileID:   uuid.NewUUID(),
		FileName: "file.txt",
		FileSize: 100,
		MimeType: "text/plain",
		AddedBy:  uuid.NewUUID(),
	}

	tests := []struct {
		name string
		cmd  taskapp.AddAttachmentCommand
	}{
		{
			name: "Empty TaskID",
			cmd: func() taskapp.AddAttachmentCommand {
				c := validCmd
				c.TaskID = ""
				return c
			}(),
		},
		{
			name: "Empty FileID",
			cmd: func() taskapp.AddAttachmentCommand {
				c := validCmd
				c.FileID = ""
				return c
			}(),
		},
		{
			name: "Empty FileName",
			cmd: func() taskapp.AddAttachmentCommand {
				c := validCmd
				c.FileName = ""
				return c
			}(),
		},
		{
			name: "Zero FileSize",
			cmd: func() taskapp.AddAttachmentCommand {
				c := validCmd
				c.FileSize = 0
				return c
			}(),
		},
		{
			name: "Negative FileSize",
			cmd: func() taskapp.AddAttachmentCommand {
				c := validCmd
				c.FileSize = -1
				return c
			}(),
		},
		{
			name: "Empty MimeType",
			cmd: func() taskapp.AddAttachmentCommand {
				c := validCmd
				c.MimeType = ""
				return c
			}(),
		},
		{
			name: "Empty AddedBy",
			cmd: func() taskapp.AddAttachmentCommand {
				c := validCmd
				c.AddedBy = ""
				return c
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockTaskRepository()
			useCase := taskapp.NewAddAttachmentUseCase(repo)

			_, err := useCase.Execute(context.Background(), tt.cmd)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "validation failed")
		})
	}
}

func TestRemoveAttachmentUseCase_Success(t *testing.T) {
	// Arrange
	repo := mocks.NewMockTaskRepository()
	useCase := taskapp.NewRemoveAttachmentUseCase(repo)
	taskID, fileID, userID := setupTaskWithAttachment(t, repo)

	cmd := taskapp.RemoveAttachmentCommand{
		TaskID:    taskID,
		FileID:    fileID,
		RemovedBy: userID,
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	assert.True(t, result.IsSuccess())
	assert.Equal(t, taskID, result.TaskID)
	require.Len(t, result.Events, 1)

	evt, ok := result.Events[0].(*task.AttachmentRemoved)
	require.True(t, ok, "Expected *task.AttachmentRemoved event")
	assert.Equal(t, fileID, evt.FileID)
	assert.Equal(t, userID, evt.RemovedBy)
}

func TestRemoveAttachmentUseCase_Idempotent(t *testing.T) {
	// Arrange
	repo := mocks.NewMockTaskRepository()
	useCase := taskapp.NewRemoveAttachmentUseCase(repo)
	taskID, _, userID := setupTaskWithAttachment(t, repo)

	cmd := taskapp.RemoveAttachmentCommand{
		TaskID:    taskID,
		FileID:    uuid.NewUUID(), // non-existent file
		RemovedBy: userID,
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	require.NoError(t, err)
	assert.True(t, result.IsSuccess())
	assert.Empty(t, result.Events, "removing non-existent attachment should be idempotent")
}

func TestRemoveAttachmentUseCase_TaskNotFound(t *testing.T) {
	repo := mocks.NewMockTaskRepository()
	useCase := taskapp.NewRemoveAttachmentUseCase(repo)

	cmd := taskapp.RemoveAttachmentCommand{
		TaskID:    uuid.NewUUID(),
		FileID:    uuid.NewUUID(),
		RemovedBy: uuid.NewUUID(),
	}

	_, err := useCase.Execute(context.Background(), cmd)
	require.ErrorIs(t, err, taskapp.ErrTaskNotFound)
}

func TestRemoveAttachmentUseCase_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		cmd  taskapp.RemoveAttachmentCommand
	}{
		{
			name: "Empty TaskID",
			cmd: taskapp.RemoveAttachmentCommand{
				TaskID:    "",
				FileID:    uuid.NewUUID(),
				RemovedBy: uuid.NewUUID(),
			},
		},
		{
			name: "Empty FileID",
			cmd: taskapp.RemoveAttachmentCommand{
				TaskID:    uuid.NewUUID(),
				FileID:    "",
				RemovedBy: uuid.NewUUID(),
			},
		},
		{
			name: "Empty RemovedBy",
			cmd: taskapp.RemoveAttachmentCommand{
				TaskID:    uuid.NewUUID(),
				FileID:    uuid.NewUUID(),
				RemovedBy: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockTaskRepository()
			useCase := taskapp.NewRemoveAttachmentUseCase(repo)

			_, err := useCase.Execute(context.Background(), tt.cmd)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "validation failed")
		})
	}
}
