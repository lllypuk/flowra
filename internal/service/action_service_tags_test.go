package service_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	messageapp "github.com/lllypuk/flowra/internal/application/message"
	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/event"
	messagedomain "github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/tag"
	"github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/service"
	"github.com/lllypuk/flowra/tests/mocks"
)

// lockedMessageRepo serializes saves: tag processing and bot replies save from other goroutines.
type lockedMessageRepo struct {
	*messageapp.MockMessageRepository

	mu    sync.Mutex
	saved []*messagedomain.Message
}

func (r *lockedMessageRepo) Save(ctx context.Context, msg *messagedomain.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saved = append(r.saved, msg)
	return r.MockMessageRepository.Save(ctx, msg)
}

func (r *lockedMessageRepo) contents(msgType messagedomain.Type) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for _, msg := range r.saved {
		if msg.Type() == msgType {
			out = append(out, msg.Content())
		}
	}
	return out
}

type nopBus struct{}

func (nopBus) Publish(context.Context, event.DomainEvent) error { return nil }

type noUsers struct{}

func (noUsers) FindByUsername(context.Context, string) (*user.User, error) {
	return nil, errs.ErrNotFound
}
func (noUsers) FindByID(context.Context, uuid.UUID) (*user.User, error) { return nil, errs.ErrNotFound }

func TestActionService_NoteDoesNotRunTagsFromParameters(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewMockChatRepository()
	ownerID := uuid.NewUUID()
	created, err := chatapp.NewCreateChatUseCase(repo).Execute(ctx, chatapp.CreateChatCommand{
		WorkspaceID: uuid.NewUUID(), Title: "Task", Type: chatdomain.TypeTask, IsPublic: true, CreatedBy: ownerID,
	})
	require.NoError(t, err)
	chatID := created.Value.ID()

	readRepo := messageapp.NewMockChatRepository()
	readRepo.AddChat(chatID, []uuid.UUID{ownerID})
	readRepo.Chats[chatID.String()].Type = chatdomain.TypeTask

	messages := &lockedMessageRepo{MockMessageRepository: messageapp.NewMockMessageRepository()}
	executor := tag.NewCommandExecutor(&tag.ChatUseCases{
		Rename:      chatapp.NewRenameChatUseCase(repo),
		SetPriority: chatapp.NewSetPriorityUseCase(repo),
	}, noUsers{})
	send := messageapp.NewSendMessageUseCase(messages, readRepo, nil, nopBus{}, tag.NewProcessor(), executor,
		uuid.NewUUID())

	svc := service.NewActionService(
		service.ActionChatCommands{Rename: chatapp.NewRenameChatUseCase(repo)},
		chatapp.NewCommandRunner(nil),
		repoChatReader{repo: repo},
		send,
		nil,
	)
	t.Cleanup(svc.Shutdown)

	const title = "Draft\n#title Other"
	_, err = svc.Rename(ctx, chatID, title, ownerID, "")
	require.NoError(t, err)
	require.Eventually(t, func() bool { return len(messages.contents(messagedomain.TypeSystem)) == 1 },
		5*time.Second, 20*time.Millisecond, "the batched note is posted")
	assert.Contains(t, messages.contents(messagedomain.TypeSystem)[0], "#title Other", "the note quotes the title")

	// A person's message with a tag still runs; its bot reply is saved after the tag ran, and the note's tag,
	// posted earlier, would have run by then too.
	_, err = send.Execute(ctx, messageapp.SendMessageCommand{
		ChatID: chatID, AuthorID: ownerID, Content: "#priority High",
	})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return len(messages.contents(messagedomain.TypeBot)) == 1 },
		5*time.Second, 20*time.Millisecond, "the person's tag runs and gets a reply")

	c, err := repo.Load(ctx, chatID)
	require.NoError(t, err)
	assert.Equal(t, "High", c.Priority())
	assert.Equal(t, title, c.Title(), "the tag inside the action parameter must not run")
}
