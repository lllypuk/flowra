package mocks

import (
	"context"
	"errors"
	"time"

	"github.com/lllypuk/flowra/internal/application/appcore"
	taskapp "github.com/lllypuk/flowra/internal/application/task"
	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

var ErrTaskChatNotFound = errors.New("task for chat not found")

const taskLookupLimit = 10000

// TaskLister returns the read models the action mock mutates in place.
type TaskLister interface {
	ListTasks(ctx context.Context, filters taskapp.Filters) ([]*taskapp.ReadModel, error)
}

// MockTaskActionService applies task actions directly to the lister's read models.
type MockTaskActionService struct {
	tasks TaskLister
}

func NewMockTaskActionService(tasks TaskLister) *MockTaskActionService {
	return &MockTaskActionService{tasks: tasks}
}

func (m *MockTaskActionService) ChangeStatus(
	ctx context.Context,
	chatID uuid.UUID,
	newStatus string,
	_ uuid.UUID,
) (*appcore.ActionResult, error) {
	return m.apply(ctx, chatID, func(rm *taskapp.ReadModel) { rm.Status = taskdomain.Status(newStatus) })
}

func (m *MockTaskActionService) SetPriority(
	ctx context.Context,
	chatID uuid.UUID,
	priority string,
	_ uuid.UUID,
) (*appcore.ActionResult, error) {
	return m.apply(ctx, chatID, func(rm *taskapp.ReadModel) { rm.Priority = taskdomain.Priority(priority) })
}

func (m *MockTaskActionService) AssignUser(
	ctx context.Context,
	chatID uuid.UUID,
	assigneeID *uuid.UUID,
	_ uuid.UUID,
) (*appcore.ActionResult, error) {
	return m.apply(ctx, chatID, func(rm *taskapp.ReadModel) { rm.AssignedTo = assigneeID })
}

func (m *MockTaskActionService) SetDueDate(
	ctx context.Context,
	chatID uuid.UUID,
	dueDate *time.Time,
	_ uuid.UUID,
) (*appcore.ActionResult, error) {
	return m.apply(ctx, chatID, func(rm *taskapp.ReadModel) { rm.DueDate = dueDate })
}

func (m *MockTaskActionService) apply(
	ctx context.Context,
	chatID uuid.UUID,
	change func(*taskapp.ReadModel),
) (*appcore.ActionResult, error) {
	tasks, err := m.tasks.ListTasks(ctx, taskapp.Filters{Limit: taskLookupLimit})
	if err != nil {
		return nil, err
	}
	for _, rm := range tasks {
		if rm.ChatID == chatID {
			change(rm)
			rm.Version++
			return &appcore.ActionResult{MessageID: uuid.NewUUID(), Success: true}, nil
		}
	}
	return nil, ErrTaskChatNotFound
}
