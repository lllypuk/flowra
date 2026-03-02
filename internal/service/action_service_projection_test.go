package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/application/appcore"
	messageapp "github.com/lllypuk/flowra/internal/application/message"
	messagedomain "github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockActionMessageSender struct {
	calls   int
	lastCmd messageapp.SendMessageCommand
	cmds    []messageapp.SendMessageCommand
	err     error
}

func (m *mockActionMessageSender) Execute(
	_ context.Context,
	cmd messageapp.SendMessageCommand,
) (messageapp.Result, error) {
	m.calls++
	m.lastCmd = cmd
	m.cmds = append(m.cmds, cmd)
	if m.err != nil {
		return messageapp.Result{}, m.err
	}

	msg, err := messagedomain.NewMessageWithType(
		cmd.ChatID,
		cmd.AuthorID,
		cmd.Content,
		cmd.ParentMessageID,
		cmd.Type,
		cmd.ActorID,
	)
	if err != nil {
		return messageapp.Result{}, err
	}

	return messageapp.Result{Value: msg}, nil
}

type mockTaskProjectionSync struct {
	calls      int
	lastChatID uuid.UUID
	err        error
}

func (m *mockTaskProjectionSync) RebuildOne(_ context.Context, chatID uuid.UUID) error {
	m.calls++
	m.lastChatID = chatID
	return m.err
}

func TestActionService_TaskTagActionsSyncProjection(t *testing.T) {
	chatID := uuid.NewUUID()
	actorID := uuid.NewUUID()
	dueDate := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name      string
		wantTag   string
		executeFn func(*service.ActionService) error
	}{
		{
			name:    "status",
			wantTag: "#status In Progress",
			executeFn: func(svc *service.ActionService) error {
				_, err := svc.ChangeStatus(context.Background(), chatID, "In Progress", actorID)
				return err
			},
		},
		{
			name:    "priority",
			wantTag: "#priority High",
			executeFn: func(svc *service.ActionService) error {
				_, err := svc.SetPriority(context.Background(), chatID, "High", actorID)
				return err
			},
		},
		{
			name:    "assignee clear",
			wantTag: "#assignee @none",
			executeFn: func(svc *service.ActionService) error {
				_, err := svc.AssignUser(context.Background(), chatID, nil, actorID)
				return err
			},
		},
		{
			name:    "due date set",
			wantTag: "#due 2026-03-15",
			executeFn: func(svc *service.ActionService) error {
				_, err := svc.SetDueDate(context.Background(), chatID, &dueDate, actorID)
				return err
			},
		},
		{
			name:    "due date clear",
			wantTag: "#due",
			executeFn: func(svc *service.ActionService) error {
				_, err := svc.SetDueDate(context.Background(), chatID, nil, actorID)
				return err
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sender := &mockActionMessageSender{}
			projector := &mockTaskProjectionSync{}
			svc := service.NewActionService(sender, nil, service.WithTaskProjectionSync(projector))
			t.Cleanup(svc.Shutdown)

			err := tc.executeFn(svc)
			require.NoError(t, err)
			assert.Equal(t, 1, sender.calls)
			assert.Equal(t, tc.wantTag, sender.lastCmd.Content)
			assert.Equal(t, 1, projector.calls)
			assert.Equal(t, chatID, projector.lastChatID)
		})
	}
}

func TestActionService_ProjectionSyncFailureReturnsError(t *testing.T) {
	chatID := uuid.NewUUID()
	actorID := uuid.NewUUID()
	sender := &mockActionMessageSender{}
	projector := &mockTaskProjectionSync{err: errors.New("projection failed")}
	svc := service.NewActionService(sender, nil, service.WithTaskProjectionSync(projector))
	t.Cleanup(svc.Shutdown)

	result, err := svc.ChangeStatus(context.Background(), chatID, "Done", actorID)
	require.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "failed to sync task projection")
	assert.Equal(t, 1, sender.calls)
	assert.Equal(t, 1, projector.calls)
}

func TestActionService_DoesNotSyncProjectionWhenSendMessageFails(t *testing.T) {
	chatID := uuid.NewUUID()
	actorID := uuid.NewUUID()
	sender := &mockActionMessageSender{err: errors.New("send failed")}
	projector := &mockTaskProjectionSync{}
	svc := service.NewActionService(sender, nil, service.WithTaskProjectionSync(projector))
	t.Cleanup(svc.Shutdown)

	result, err := svc.SetPriority(context.Background(), chatID, "High", actorID)
	require.Error(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Equal(t, 1, sender.calls)
	assert.Equal(t, 0, projector.calls)
}

func TestActionService_CloseAndReopenSyncProjection(t *testing.T) {
	chatID := uuid.NewUUID()
	actorID := uuid.NewUUID()

	testCases := []struct {
		name        string
		wantTag     string
		wantMessage string
		executeFn   func(*service.ActionService) error
	}{
		{
			name:        "close",
			wantTag:     "#close",
			wantMessage: "✅ Chat closed",
			executeFn: func(svc *service.ActionService) error {
				_, err := svc.Close(context.Background(), chatID, actorID)
				return err
			},
		},
		{
			name:        "reopen",
			wantTag:     "#reopen",
			wantMessage: "✅ Chat reopened",
			executeFn: func(svc *service.ActionService) error {
				_, err := svc.Reopen(context.Background(), chatID, actorID)
				return err
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sender := &mockActionMessageSender{}
			projector := &mockTaskProjectionSync{}
			svc := service.NewActionService(sender, nil, service.WithTaskProjectionSync(projector))
			t.Cleanup(svc.Shutdown)

			err := tc.executeFn(svc)
			require.NoError(t, err)
			require.Len(t, sender.cmds, 2)
			assert.Equal(t, tc.wantTag, sender.cmds[0].Content)
			assert.Equal(t, tc.wantMessage, sender.cmds[1].Content)
			assert.Equal(t, 1, projector.calls)
			assert.Equal(t, chatID, projector.lastChatID)
		})
	}
}

func TestActionService_CloseAndReopen_StopOnProjectionSyncFailure(t *testing.T) {
	chatID := uuid.NewUUID()
	actorID := uuid.NewUUID()

	testCases := []struct {
		name      string
		executeFn func(*service.ActionService) (*appcore.ActionResult, error)
	}{
		{
			name: "close",
			executeFn: func(svc *service.ActionService) (*appcore.ActionResult, error) {
				return svc.Close(context.Background(), chatID, actorID)
			},
		},
		{
			name: "reopen",
			executeFn: func(svc *service.ActionService) (*appcore.ActionResult, error) {
				return svc.Reopen(context.Background(), chatID, actorID)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sender := &mockActionMessageSender{}
			projector := &mockTaskProjectionSync{err: errors.New("projection failed")}
			svc := service.NewActionService(sender, nil, service.WithTaskProjectionSync(projector))
			t.Cleanup(svc.Shutdown)

			result, err := tc.executeFn(svc)
			require.Error(t, err)
			require.NotNil(t, result)
			assert.False(t, result.Success)
			assert.Contains(t, result.Error, "failed to sync task projection")
			assert.Equal(t, 1, sender.calls, "human-readable system message should not be sent after sync failure")
			assert.Equal(t, 1, projector.calls)
		})
	}
}
