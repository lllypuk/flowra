package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/appcore"
	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	messageapp "github.com/lllypuk/flowra/internal/application/message"
	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/errs"
	messagedomain "github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/service"
	"github.com/lllypuk/flowra/tests/mocks"
)

type mockActionMessageSender struct {
	cmds []messageapp.SendMessageCommand
	err  error
}

func (m *mockActionMessageSender) Execute(
	_ context.Context,
	cmd messageapp.SendMessageCommand,
) (messageapp.Result, error) {
	m.cmds = append(m.cmds, cmd)
	if m.err != nil {
		return messageapp.Result{}, m.err
	}
	msg, err := messagedomain.NewMessageWithType(cmd.ChatID, cmd.AuthorID, cmd.Content, cmd.ParentMessageID,
		cmd.Type, cmd.ActorID)
	if err != nil {
		return messageapp.Result{}, err
	}
	return messageapp.Result{Value: msg}, nil
}

type mockProjectionSync struct {
	calls int
	err   error
}

func (m *mockProjectionSync) RebuildOne(context.Context, uuid.UUID) error {
	m.calls++
	return m.err
}

// repoChatReader serves participants straight from the command repository.
type repoChatReader struct{ repo *mocks.MockChatRepository }

func (r repoChatReader) FindByID(ctx context.Context, chatID uuid.UUID) (*chatapp.ReadModel, error) {
	c, err := r.repo.Load(ctx, chatID)
	if err != nil {
		return nil, errs.ErrNotFound
	}
	return &chatapp.ReadModel{ID: c.ID(), Participants: c.Participants()}, nil
}

type actionFixture struct {
	svc        *service.ActionService
	repo       *mocks.MockChatRepository
	sender     *mockActionMessageSender
	tasks      *mockProjectionSync
	chats      *mockProjectionSync
	chatID     uuid.UUID
	ownerID    uuid.UUID
	strangerID uuid.UUID
}

func newActionFixture(t *testing.T, runner service.ActionCommandRunner) *actionFixture {
	t.Helper()
	repo := mocks.NewMockChatRepository()
	users := mocks.NewMockUserRepository()
	f := &actionFixture{
		repo:       repo,
		sender:     &mockActionMessageSender{},
		tasks:      &mockProjectionSync{},
		chats:      &mockProjectionSync{},
		ownerID:    uuid.NewUUID(),
		strangerID: uuid.NewUUID(),
	}
	created, err := chatapp.NewCreateChatUseCase(repo).Execute(context.Background(), chatapp.CreateChatCommand{
		WorkspaceID: uuid.NewUUID(), Title: "Task", Type: chatdomain.TypeTask, IsPublic: true, CreatedBy: f.ownerID,
	})
	require.NoError(t, err)
	f.chatID = created.Value.ID()

	if runner == nil {
		runner = chatapp.NewCommandRunner(nil)
	}
	f.svc = service.NewActionService(
		service.ActionChatCommands{
			ChangeStatus: chatapp.NewChangeStatusUseCase(repo),
			SetPriority:  chatapp.NewSetPriorityUseCase(repo),
			AssignUser:   chatapp.NewAssignUserUseCase(repo, users),
			SetDueDate:   chatapp.NewSetDueDateUseCase(repo),
			Rename:       chatapp.NewRenameChatUseCase(repo),
			Close:        chatapp.NewCloseChatUseCase(repo),
			Reopen:       chatapp.NewReopenChatUseCase(repo),
		},
		runner,
		repoChatReader{repo: repo},
		f.sender,
		nil,
		service.WithTaskProjectionSync(f.tasks),
		service.WithChatProjectionSync(f.chats),
	)
	t.Cleanup(f.svc.Shutdown)
	return f
}

func (f *actionFixture) load(t *testing.T) *chatdomain.Chat {
	t.Helper()
	c, err := f.repo.Load(context.Background(), f.chatID)
	require.NoError(t, err)
	return c
}

func TestActionService_AppliesBeforeAnswering(t *testing.T) {
	ctx := context.Background()
	f := newActionFixture(t, nil)
	due := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)

	steps := []struct {
		name  string
		act   func() (*appcore.ActionResult, error)
		check func(*chatdomain.Chat)
	}{
		{"status", func() (*appcore.ActionResult, error) {
			return f.svc.ChangeStatus(ctx, f.chatID, "In Progress", f.ownerID, "")
		}, func(c *chatdomain.Chat) { assert.Equal(t, "In Progress", c.Status()) }},
		{"priority", func() (*appcore.ActionResult, error) {
			return f.svc.SetPriority(ctx, f.chatID, "High", f.ownerID, "")
		}, func(c *chatdomain.Chat) { assert.Equal(t, "High", c.Priority()) }},
		{"due date", func() (*appcore.ActionResult, error) {
			return f.svc.SetDueDate(ctx, f.chatID, &due, f.ownerID, "")
		}, func(c *chatdomain.Chat) { require.NotNil(t, c.DueDate()); assert.True(t, due.Equal(*c.DueDate())) }},
		{"rename", func() (*appcore.ActionResult, error) {
			return f.svc.Rename(ctx, f.chatID, "Renamed", f.ownerID, "")
		}, func(c *chatdomain.Chat) { assert.Equal(t, "Renamed", c.Title()) }},
		{"close", func() (*appcore.ActionResult, error) {
			return f.svc.Close(ctx, f.chatID, f.ownerID, "")
		}, func(c *chatdomain.Chat) { assert.Equal(t, chatdomain.StatusClosed, c.Status()) }},
		{"reopen", func() (*appcore.ActionResult, error) {
			return f.svc.Reopen(ctx, f.chatID, f.ownerID, "")
		}, func(c *chatdomain.Chat) { assert.NotEqual(t, chatdomain.StatusClosed, c.Status()) }},
	}
	for _, s := range steps {
		t.Run(s.name, func(t *testing.T) {
			res, err := s.act()
			require.NoError(t, err)
			assert.True(t, res.Success)
			assert.Equal(t, f.chatID, res.ChatID)
			s.check(f.load(t))
		})
	}
	assert.Equal(t, len(steps), f.tasks.calls, "every applied action syncs the task projection")
	for _, cmd := range f.sender.cmds {
		assert.NotContains(t, cmd.Content, "#", "actions no longer go through tag messages")
	}
}

func TestActionService_Rejections(t *testing.T) {
	ctx := context.Background()

	t.Run("not a participant", func(t *testing.T) {
		f := newActionFixture(t, nil)
		_, err := f.svc.ChangeStatus(ctx, f.chatID, "Done", f.strangerID, "")
		require.ErrorIs(t, err, errs.ErrForbidden)
		assert.Equal(t, "To Do", f.load(t).Status())
	})

	t.Run("domain error", func(t *testing.T) {
		f := newActionFixture(t, nil)
		_, err := f.svc.ChangeStatus(ctx, f.chatID, "Bogus", f.ownerID, "")
		require.Error(t, err)
		assert.Zero(t, f.tasks.calls)
	})

	t.Run("close twice is an invalid state, not an internal error", func(t *testing.T) {
		f := newActionFixture(t, nil)
		_, err := f.svc.Close(ctx, f.chatID, f.ownerID, "")
		require.NoError(t, err)
		_, err = f.svc.Close(ctx, f.chatID, f.ownerID, "")
		require.ErrorIs(t, err, errs.ErrInvalidState)
	})
}

func TestActionService_AppliedSurvivesProjectionAndNoteFailures(t *testing.T) {
	f := newActionFixture(t, nil)
	f.tasks.err = errors.New("projection down")
	f.sender.err = errors.New("messages down")

	res, err := f.svc.Close(context.Background(), f.chatID, f.ownerID, "")
	require.NoError(t, err)
	assert.True(t, res.Success)
	assert.Equal(t, chatdomain.StatusClosed, f.load(t).Status())
}

// stubRunner records the key and answers with a fixed outcome without running the command.
type stubRunner struct {
	key     appcore.CommandKey
	outcome chatapp.Outcome
}

func (s *stubRunner) Run(
	_ context.Context,
	_ uuid.UUID,
	key appcore.CommandKey,
	_ func(context.Context) (chatapp.Result, error),
) (chatapp.Outcome, error) {
	s.key = key
	return s.outcome, nil
}

func TestActionService_KeyAndReplay(t *testing.T) {
	runner := &stubRunner{outcome: chatapp.Outcome{Version: 5, Replayed: true}}
	f := newActionFixture(t, runner)

	res, err := f.svc.Close(context.Background(), f.chatID, f.ownerID, "step-1")
	require.NoError(t, err)

	assert.Equal(t, f.ownerID.String()+":step-1", runner.key.ID, "keys are scoped to the actor")
	assert.Equal(t, "close", runner.key.Fingerprint)
	assert.Equal(t, 5, res.Version)
	assert.True(t, res.Replayed)
	assert.Equal(t, 1, f.chats.calls, "a replay rebuilds the chat read model")
	assert.Empty(t, f.sender.cmds, "a replay posts no second note")
}
