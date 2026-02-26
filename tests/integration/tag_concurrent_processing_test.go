//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	guuid "github.com/google/uuid"
	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/tag"
	userdomain "github.com/lllypuk/flowra/internal/domain/user"
	domainuuid "github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	"github.com/lllypuk/flowra/tests/mocks"
	"github.com/lllypuk/flowra/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingChatCommandRepo struct {
	base           chatapp.CommandRepository
	saveDelay      time.Duration
	alwaysConflict bool
	conflicts      atomic.Int64
}

func (r *countingChatCommandRepo) Load(ctx context.Context, chatID domainuuid.UUID) (*chatdomain.Chat, error) {
	return r.base.Load(ctx, chatID)
}

func (r *countingChatCommandRepo) Save(ctx context.Context, c *chatdomain.Chat) error {
	if r.saveDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(r.saveDelay):
		}
	}

	if r.alwaysConflict {
		r.conflicts.Add(1)
		return errs.ErrConcurrentModification
	}

	err := r.base.Save(ctx, c)
	if errors.Is(err, errs.ErrConcurrentModification) {
		r.conflicts.Add(1)
	}
	return err
}

func (r *countingChatCommandRepo) GetEvents(ctx context.Context, chatID domainuuid.UUID) ([]event.DomainEvent, error) {
	return r.base.GetEvents(ctx, chatID)
}

func (r *countingChatCommandRepo) ConflictCount() int64 {
	return r.conflicts.Load()
}

type tagExecutorTestEnv struct {
	t              *testing.T
	ctx            context.Context
	chatRepo       *countingChatCommandRepo
	rawChatRepo    *mongodb.MongoChatRepository
	userRepo       *mongodb.MongoUserRepository
	executor       *tag.CommandExecutor
	chatID         domainuuid.UUID
	actorID        domainuuid.UUID
	actorGoogleID  guuid.UUID
	candidateUsers map[string]domainuuid.UUID
}

func TestTagConcurrentProcessing_RetriesAndAppliesConcurrentCommands(t *testing.T) {
	env := newTagExecutorTestEnv(t, countingRepoOptions{saveDelay: 250 * time.Microsecond})

	commands := make([]tag.Command, 0, 6)
	expectedTitles := make(map[string]struct{}, 2)

	statuses := []string{"In Progress", "Done"}
	priorities := []string{"High", "Critical"}

	for i := 0; i < 2; i++ {
		title := fmt.Sprintf("Concurrent Title %02d", i)
		expectedTitles[title] = struct{}{}
		commands = append(commands, tag.ChangeTitleCommand{
			ChatID: mustToGoogleUUID(t, env.chatID),
			Title:  title,
		})
	}

	for i := 0; i < 2; i++ {
		commands = append(commands, tag.ChangeStatusCommand{
			ChatID: mustToGoogleUUID(t, env.chatID),
			Status: statuses[i%len(statuses)],
		})
		commands = append(commands, tag.ChangePriorityCommand{
			ChatID:   mustToGoogleUUID(t, env.chatID),
			Priority: priorities[i%len(priorities)],
		})
	}

	runTagCommandsConcurrently(t, env.ctx, env.executor, env.actorGoogleID, commands)

	assert.Greater(t, env.chatRepo.ConflictCount(), int64(0), "expected optimistic-lock conflicts during concurrent updates")

	finalChat, err := env.rawChatRepo.Load(env.ctx, env.chatID)
	require.NoError(t, err)
	require.NotNil(t, finalChat)

	_, ok := expectedTitles[finalChat.Title()]
	assert.True(t, ok, "final title should be one of submitted titles")
	assert.Contains(t, statuses, finalChat.Status())
	assert.Contains(t, priorities, finalChat.Priority())
}

func TestTagConcurrentProcessing_AssignUserConcurrentUpdates(t *testing.T) {
	env := newTagExecutorTestEnv(t, countingRepoOptions{saveDelay: 250 * time.Microsecond})

	commands := make([]tag.Command, 0, 8)
	usernames := []string{"@alice", "@bob", "@carol", "@none"}

	for i := 0; i < 8; i++ {
		commands = append(commands, tag.AssignUserCommand{
			ChatID:   mustToGoogleUUID(t, env.chatID),
			Username: usernames[i%len(usernames)],
		})
	}

	runTagCommandsConcurrently(t, env.ctx, env.executor, env.actorGoogleID, commands)

	assert.Greater(t, env.chatRepo.ConflictCount(), int64(0), "expected optimistic-lock conflicts during concurrent assignee updates")

	finalChat, err := env.rawChatRepo.Load(env.ctx, env.chatID)
	require.NoError(t, err)

	if finalChat.AssigneeID() == nil {
		return
	}

	allowed := []domainuuid.UUID{
		env.candidateUsers["alice"],
		env.candidateUsers["bob"],
		env.candidateUsers["carol"],
	}
	assert.True(t, slices.Contains(allowed, *finalChat.AssigneeID()), "unexpected final assignee ID")
}

func TestTagConcurrentProcessing_StopsRetryOnContextCancellation(t *testing.T) {
	env := newTagExecutorTestEnv(t, countingRepoOptions{
		saveDelay:      0,
		alwaysConflict: true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := env.executor.Execute(ctx, tag.ChangeStatusCommand{
		ChatID: mustToGoogleUUID(t, env.chatID),
		Status: "In Progress",
	}, env.actorGoogleID)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.GreaterOrEqual(t, env.chatRepo.ConflictCount(), int64(1))
}

type countingRepoOptions struct {
	saveDelay      time.Duration
	alwaysConflict bool
}

func newTagExecutorTestEnv(t *testing.T, repoOpts countingRepoOptions) *tagExecutorTestEnv {
	t.Helper()

	db := testutil.SetupSharedTestMongoDBWithOptions(t, true)
	ctx := context.Background()

	eventStoreRepo := mocks.NewMockEventStore()
	rawChatRepo := mongodb.NewMongoChatRepository(eventStoreRepo, db.Collection("chats_read_model"))
	chatRepo := &countingChatCommandRepo{
		base:           rawChatRepo,
		saveDelay:      repoOpts.saveDelay,
		alwaysConflict: repoOpts.alwaysConflict,
	}
	userRepo := mongodb.NewMongoUserRepository(db.Collection("users"))

	actorID := seedUser(t, ctx, userRepo, "actor")
	candidateUsers := map[string]domainuuid.UUID{
		"alice": seedUser(t, ctx, userRepo, "alice"),
		"bob":   seedUser(t, ctx, userRepo, "bob"),
		"carol": seedUser(t, ctx, userRepo, "carol"),
	}
	chatID := seedTaskChat(t, ctx, rawChatRepo, actorID)

	actorGoogleID := mustToGoogleUUID(t, actorID)

	chatUseCases := &tag.ChatUseCases{
		ChangeStatus: chatapp.NewChangeStatusUseCase(chatRepo),
		AssignUser:   chatapp.NewAssignUserUseCase(chatRepo),
		SetPriority:  chatapp.NewSetPriorityUseCase(chatRepo),
		Rename:       chatapp.NewRenameChatUseCase(chatRepo),
	}

	return &tagExecutorTestEnv{
		t:              t,
		ctx:            ctx,
		chatRepo:       chatRepo,
		rawChatRepo:    rawChatRepo,
		userRepo:       userRepo,
		executor:       tag.NewCommandExecutor(chatUseCases, userRepo),
		chatID:         chatID,
		actorID:        actorID,
		actorGoogleID:  actorGoogleID,
		candidateUsers: candidateUsers,
	}
}

func seedTaskChat(
	t *testing.T,
	ctx context.Context,
	chatRepo *mongodb.MongoChatRepository,
	actorID domainuuid.UUID,
) domainuuid.UUID {
	t.Helper()

	chatAggregate, err := chatdomain.NewChat(domainuuid.NewUUID(), chatdomain.TypeDiscussion, false, actorID)
	require.NoError(t, err)
	require.NoError(t, chatRepo.Save(ctx, chatAggregate))

	convertUC := chatapp.NewConvertToTaskUseCase(chatRepo)
	_, err = convertUC.Execute(ctx, chatapp.ConvertToTaskCommand{
		ChatID:      chatAggregate.ID(),
		Title:       "Concurrent Tag Test",
		ConvertedBy: actorID,
	})
	require.NoError(t, err)

	return chatAggregate.ID()
}

func seedUser(t *testing.T, ctx context.Context, userRepo *mongodb.MongoUserRepository, username string) domainuuid.UUID {
	t.Helper()

	u, err := userdomain.NewUser(
		"ext-"+username,
		username,
		username+"@example.com",
		username,
	)
	require.NoError(t, err)
	require.NoError(t, userRepo.Save(ctx, u))
	return u.ID()
}

func runTagCommandsConcurrently(
	t *testing.T,
	ctx context.Context,
	executor *tag.CommandExecutor,
	actorID guuid.UUID,
	commands []tag.Command,
) {
	t.Helper()

	start := make(chan struct{})
	errCh := make(chan error, len(commands))

	var wg sync.WaitGroup
	for _, cmd := range commands {
		wg.Add(1)
		go func(cmd tag.Command) {
			defer wg.Done()
			<-start
			errCh <- executor.Execute(ctx, cmd, actorID)
		}(cmd)
	}

	close(start)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		require.NoError(t, err)
	}
}

func mustToGoogleUUID(t *testing.T, id domainuuid.UUID) guuid.UUID {
	t.Helper()

	googleID, err := id.ToGoogleUUID()
	require.NoError(t, err)
	return googleID
}
