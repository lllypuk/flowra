//go:build integration

package integration_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	taskapp "github.com/lllypuk/flowra/internal/application/task"
	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
	domainerrs "github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/event"
	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/eventstore"
	mongoinfra "github.com/lllypuk/flowra/internal/infrastructure/mongodb"
	"github.com/lllypuk/flowra/internal/infrastructure/projector"
	mongorepo "github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	"github.com/lllypuk/flowra/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type chatSoTTestEnv struct {
	db            *mongo.Database
	eventStore    *eventstore.MongoEventStore
	chatRepo      *mongorepo.MongoChatRepository
	taskRepo      *mongorepo.MongoTaskRepository
	chatProjector *projector.ChatProjector
	taskProjector *projector.ChatToTaskReadModelProjector
}

type chatReadModelDoc struct {
	ChatID     string     `bson:"chat_id"`
	Type       string     `bson:"type"`
	Status     string     `bson:"status"`
	Priority   string     `bson:"priority"`
	AssignedTo *string    `bson:"assigned_to,omitempty"`
	DueDate    *time.Time `bson:"due_date,omitempty"`
}

func TestChatSoT_TypedLifecycleEmitsOnlyChatEventsAndKeepsReadModelsConsistent(t *testing.T) {
	ctx := context.Background()
	env := newChatSoTTestEnv(t)

	workspaceID := uuid.NewUUID()
	actorID := uuid.NewUUID()
	assigneeID := uuid.NewUUID()
	dueDate := time.Now().UTC().Add(72 * time.Hour).Truncate(time.Second)

	createResult, err := chatapp.NewCreateChatUseCase(env.chatRepo).Execute(ctx, chatapp.CreateChatCommand{
		WorkspaceID: workspaceID,
		Title:       "Regression task",
		Type:        chatdomain.TypeTask,
		IsPublic:    true,
		CreatedBy:   actorID,
	})
	require.NoError(t, err)
	chatID := createResult.Value.ID()

	_, err = chatapp.NewChangeStatusUseCase(env.chatRepo).Execute(ctx, chatapp.ChangeStatusCommand{
		ChatID:    chatID,
		Status:    string(taskdomain.StatusInProgress),
		ChangedBy: actorID,
	})
	require.NoError(t, err)

	_, err = chatapp.NewSetPriorityUseCase(env.chatRepo).Execute(ctx, chatapp.SetPriorityCommand{
		ChatID:   chatID,
		Priority: string(taskdomain.PriorityHigh),
		SetBy:    actorID,
	})
	require.NoError(t, err)

	_, err = chatapp.NewAssignUserUseCase(env.chatRepo).Execute(ctx, chatapp.AssignUserCommand{
		ChatID:     chatID,
		AssigneeID: &assigneeID,
		AssignedBy: actorID,
	})
	require.NoError(t, err)

	_, err = chatapp.NewSetDueDateUseCase(env.chatRepo).Execute(ctx, chatapp.SetDueDateCommand{
		ChatID:  chatID,
		DueDate: &dueDate,
		SetBy:   actorID,
	})
	require.NoError(t, err)

	require.NoError(t, env.syncReadModels(ctx, chatID))

	taskRM, err := env.taskRepo.FindByID(ctx, chatID)
	require.NoError(t, err)
	assert.Equal(t, taskdomain.TypeTask, taskRM.EntityType)
	assert.Equal(t, taskdomain.StatusInProgress, taskRM.Status)
	assert.Equal(t, taskdomain.PriorityHigh, taskRM.Priority)
	if assert.NotNil(t, taskRM.AssignedTo) {
		assert.Equal(t, assigneeID, *taskRM.AssignedTo)
	}
	if assert.NotNil(t, taskRM.DueDate) {
		assert.WithinDuration(t, dueDate, *taskRM.DueDate, time.Second)
	}

	chatRM := env.mustFindChatReadModel(t, ctx, chatID)
	assert.Equal(t, chatID.String(), chatRM.ChatID)
	assert.Equal(t, string(chatdomain.TypeTask), chatRM.Type)
	assert.Equal(t, string(taskdomain.StatusInProgress), chatRM.Status)
	assert.Equal(t, string(taskdomain.PriorityHigh), chatRM.Priority)
	if assert.NotNil(t, chatRM.AssignedTo) {
		assert.Equal(t, assigneeID.String(), *chatRM.AssignedTo)
	}
	if assert.NotNil(t, chatRM.DueDate) {
		assert.WithinDuration(t, dueDate, *chatRM.DueDate, time.Second)
	}

	events, err := env.eventStore.LoadEvents(ctx, chatID.String())
	require.NoError(t, err)
	require.NotEmpty(t, events)

	var createdCount, typeChangedCount int
	for _, evt := range events {
		assert.True(t, strings.HasPrefix(evt.EventType(), "chat."), "unexpected non-chat event %q", evt.EventType())
		if evt.EventType() == chatdomain.EventTypeChatCreated {
			createdCount++
		}
		if evt.EventType() == chatdomain.EventTypeChatTypeChanged {
			typeChangedCount++
		}
	}
	assert.Equal(t, 1, createdCount, "chat should be created once")
	assert.Equal(t, 1, typeChangedCount, "type conversion should happen once")

	taskByChat, err := env.taskRepo.FindByChatID(ctx, chatID)
	require.NoError(t, err)
	assert.Equal(t, taskRM.ID, taskByChat.ID)

	taskCount, err := env.db.Collection(mongoinfra.CollectionTaskReadModel).
		CountDocuments(ctx, bson.M{"chat_id": chatID.String()})
	require.NoError(t, err)
	assert.Equal(t, int64(1), taskCount, "task read model must not contain duplicates")

	chatCount, err := env.db.Collection(mongoinfra.CollectionChatReadModel).
		CountDocuments(ctx, bson.M{"chat_id": chatID.String()})
	require.NoError(t, err)
	assert.Equal(t, int64(1), chatCount, "chat read model must not contain duplicates")
}

func TestChatSoT_CloseReopenLifecycleProjectsRealStateTransitions(t *testing.T) {
	ctx := context.Background()
	env := newChatSoTTestEnv(t)

	workspaceID := uuid.NewUUID()
	actorID := uuid.NewUUID()

	createResult, err := chatapp.NewCreateChatUseCase(env.chatRepo).Execute(ctx, chatapp.CreateChatCommand{
		WorkspaceID: workspaceID,
		Title:       "Lifecycle task",
		Type:        chatdomain.TypeTask,
		IsPublic:    true,
		CreatedBy:   actorID,
	})
	require.NoError(t, err)
	chatID := createResult.Value.ID()

	_, err = chatapp.NewCloseChatUseCase(env.eventStore).Execute(ctx, chatapp.CloseChatCommand{
		ChatID:   chatID,
		ClosedBy: actorID,
	})
	require.NoError(t, err)
	require.NoError(t, env.syncReadModels(ctx, chatID))

	afterCloseTask, err := env.taskRepo.FindByID(ctx, chatID)
	require.NoError(t, err)
	assert.Equal(t, taskdomain.StatusDone, afterCloseTask.Status)

	afterCloseChat := env.mustFindChatReadModel(t, ctx, chatID)
	assert.Equal(t, chatdomain.StatusClosed, afterCloseChat.Status)

	_, err = chatapp.NewReopenChatUseCase(env.eventStore).Execute(ctx, chatapp.ReopenChatCommand{
		ChatID:     chatID,
		ReopenedBy: actorID,
	})
	require.NoError(t, err)
	require.NoError(t, env.syncReadModels(ctx, chatID))

	afterReopenTask, err := env.taskRepo.FindByID(ctx, chatID)
	require.NoError(t, err)
	assert.Equal(t, taskdomain.StatusToDo, afterReopenTask.Status)

	afterReopenChat := env.mustFindChatReadModel(t, ctx, chatID)
	assert.Equal(t, string(taskdomain.StatusToDo), afterReopenChat.Status)

	events, err := env.eventStore.LoadEvents(ctx, chatID.String())
	require.NoError(t, err)
	assert.True(t, hasEventType(events, chatdomain.EventTypeChatClosed))
	assert.True(t, hasEventType(events, chatdomain.EventTypeChatReopened))
	for _, evt := range events {
		assert.True(t, strings.HasPrefix(evt.EventType(), "chat."), "unexpected non-chat event %q", evt.EventType())
	}
}

func TestChatSoT_SidebarBoardRegressionQueriesStayConsistentAfterReload(t *testing.T) {
	ctx := context.Background()
	env := newChatSoTTestEnv(t)

	workspaceID := uuid.NewUUID()
	actorID := uuid.NewUUID()
	assigneeID := uuid.NewUUID()
	dueDate := time.Now().UTC().Add(72 * time.Hour).Truncate(time.Second)

	createResult, err := chatapp.NewCreateChatUseCase(env.chatRepo).Execute(ctx, chatapp.CreateChatCommand{
		WorkspaceID: workspaceID,
		Title:       "Board sidebar regression task",
		Type:        chatdomain.TypeTask,
		IsPublic:    true,
		CreatedBy:   actorID,
	})
	require.NoError(t, err)
	chatID := createResult.Value.ID()

	_, err = chatapp.NewChangeStatusUseCase(env.chatRepo).Execute(ctx, chatapp.ChangeStatusCommand{
		ChatID:    chatID,
		Status:    string(taskdomain.StatusInProgress),
		ChangedBy: actorID,
	})
	require.NoError(t, err)
	require.NoError(t, env.syncReadModels(ctx, chatID))

	_, err = chatapp.NewSetPriorityUseCase(env.chatRepo).Execute(ctx, chatapp.SetPriorityCommand{
		ChatID:   chatID,
		Priority: string(taskdomain.PriorityHigh),
		SetBy:    actorID,
	})
	require.NoError(t, err)
	require.NoError(t, env.syncReadModels(ctx, chatID))

	_, err = chatapp.NewAssignUserUseCase(env.chatRepo).Execute(ctx, chatapp.AssignUserCommand{
		ChatID:     chatID,
		AssigneeID: &assigneeID,
		AssignedBy: actorID,
	})
	require.NoError(t, err)
	require.NoError(t, env.syncReadModels(ctx, chatID))

	_, err = chatapp.NewAssignUserUseCase(env.chatRepo).Execute(ctx, chatapp.AssignUserCommand{
		ChatID:     chatID,
		AssigneeID: nil,
		AssignedBy: actorID,
	})
	require.NoError(t, err)
	require.NoError(t, env.syncReadModels(ctx, chatID))

	_, err = chatapp.NewSetDueDateUseCase(env.chatRepo).Execute(ctx, chatapp.SetDueDateCommand{
		ChatID:  chatID,
		DueDate: &dueDate,
		SetBy:   actorID,
	})
	require.NoError(t, err)
	require.NoError(t, env.syncReadModels(ctx, chatID))

	_, err = chatapp.NewChangeStatusUseCase(env.chatRepo).Execute(ctx, chatapp.ChangeStatusCommand{
		ChatID:    chatID,
		Status:    string(taskdomain.StatusDone),
		ChangedBy: actorID,
	})
	require.NoError(t, err)
	require.NoError(t, env.syncReadModels(ctx, chatID))

	assertSidebarBoardParity(t, ctx, env, workspaceID, chatID, actorID, dueDate)

	// Simulate page reload/re-fetch by rebuilding projections one more time.
	require.NoError(t, env.syncReadModels(ctx, chatID))

	assertSidebarBoardParity(t, ctx, env, workspaceID, chatID, actorID, dueDate)
}

func TestChatSoT_ConcurrentTypedMutationsSingleAggregateRemainDeterministic(t *testing.T) {
	ctx := context.Background()
	env := newChatSoTTestEnv(t)

	workspaceID := uuid.NewUUID()
	actorID := uuid.NewUUID()

	createResult, err := chatapp.NewCreateChatUseCase(env.chatRepo).Execute(ctx, chatapp.CreateChatCommand{
		WorkspaceID: workspaceID,
		Title:       "Concurrent typed mutation",
		Type:        chatdomain.TypeTask,
		IsPublic:    true,
		CreatedBy:   actorID,
	})
	require.NoError(t, err)
	chatID := createResult.Value.ID()

	userRepo := mongorepo.NewMongoUserRepository(env.db.Collection("users"))
	assigneeID := seedUser(t, ctx, userRepo, "typed-concurrency-assignee")

	concurrentRepo := &countingChatCommandRepo{
		base:      env.chatRepo,
		saveDelay: 300 * time.Microsecond,
	}

	changeStatusUC := chatapp.NewChangeStatusUseCase(concurrentRepo)
	assignUserUC := chatapp.NewAssignUserUseCase(concurrentRepo, userRepo)
	setPriorityUC := chatapp.NewSetPriorityUseCase(concurrentRepo)

	mutations := []func(context.Context) error{
		func(runCtx context.Context) error {
			_, execErr := changeStatusUC.Execute(runCtx, chatapp.ChangeStatusCommand{
				ChatID:    chatID,
				Status:    string(taskdomain.StatusInProgress),
				ChangedBy: actorID,
			})
			return execErr
		},
		func(runCtx context.Context) error {
			_, execErr := assignUserUC.Execute(runCtx, chatapp.AssignUserCommand{
				ChatID:     chatID,
				AssigneeID: &assigneeID,
				AssignedBy: actorID,
			})
			return execErr
		},
		func(runCtx context.Context) error {
			_, execErr := setPriorityUC.Execute(runCtx, chatapp.SetPriorityCommand{
				ChatID:   chatID,
				Priority: string(taskdomain.PriorityHigh),
				SetBy:    actorID,
			})
			return execErr
		},
	}

	start := make(chan struct{})
	errCh := make(chan error, len(mutations)*4)

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		for _, mutation := range mutations {
			wg.Add(1)
			go func(mutate func(context.Context) error) {
				defer wg.Done()
				<-start
				errCh <- retryOnOptimisticConflict(ctx, 10, mutate)
			}(mutation)
		}
	}

	close(start)
	wg.Wait()
	close(errCh)

	for mutationErr := range errCh {
		require.NoError(t, mutationErr)
	}

	require.Greater(t, concurrentRepo.ConflictCount(), int64(0), "expected optimistic-lock conflicts")
	require.NoError(t, env.syncReadModels(ctx, chatID))

	taskRM, err := env.taskRepo.FindByID(ctx, chatID)
	require.NoError(t, err)
	assert.Equal(t, taskdomain.StatusInProgress, taskRM.Status)
	assert.Equal(t, taskdomain.PriorityHigh, taskRM.Priority)
	if assert.NotNil(t, taskRM.AssignedTo) {
		assert.Equal(t, assigneeID, *taskRM.AssignedTo)
	}
}

func newChatSoTTestEnv(t *testing.T) *chatSoTTestEnv {
	t.Helper()

	client, db := testutil.SetupSharedTestMongoDBWithClientOptions(t, true)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	eventStoreRepo := eventstore.NewMongoEventStore(client, db.Name(), eventstore.WithLogger(logger))

	chatReadModelColl := db.Collection(mongoinfra.CollectionChatReadModel)
	taskReadModelColl := db.Collection(mongoinfra.CollectionTaskReadModel)

	chatRepo := mongorepo.NewMongoChatRepository(eventStoreRepo, chatReadModelColl, mongorepo.WithChatRepoLogger(logger))
	taskRepo := mongorepo.NewMongoTaskRepository(eventStoreRepo, taskReadModelColl, mongorepo.WithTaskRepoLogger(logger))

	return &chatSoTTestEnv{
		db:            db,
		eventStore:    eventStoreRepo,
		chatRepo:      chatRepo,
		taskRepo:      taskRepo,
		chatProjector: projector.NewChatProjector(eventStoreRepo, chatReadModelColl, logger),
		taskProjector: projector.NewChatToTaskReadModelProjector(eventStoreRepo, taskReadModelColl, logger),
	}
}

func (e *chatSoTTestEnv) syncReadModels(ctx context.Context, chatID uuid.UUID) error {
	if err := e.chatProjector.RebuildOne(ctx, chatID); err != nil {
		return err
	}
	return e.taskProjector.RebuildOne(ctx, chatID)
}

func (e *chatSoTTestEnv) mustFindChatReadModel(t *testing.T, ctx context.Context, chatID uuid.UUID) chatReadModelDoc {
	t.Helper()

	var doc chatReadModelDoc
	err := e.db.Collection(mongoinfra.CollectionChatReadModel).
		FindOne(ctx, bson.M{"chat_id": chatID.String()}).
		Decode(&doc)
	require.NoError(t, err)
	return doc
}

func hasEventType(events []event.DomainEvent, target string) bool {
	for _, evt := range events {
		if evt.EventType() == target {
			return true
		}
	}
	return false
}

func assertSidebarBoardParity(
	t *testing.T,
	ctx context.Context,
	env *chatSoTTestEnv,
	workspaceID uuid.UUID,
	chatID uuid.UUID,
	actorID uuid.UUID,
	dueDate time.Time,
) {
	t.Helper()

	chatResult, err := chatapp.NewGetChatUseCase(env.eventStore).Execute(ctx, chatapp.GetChatQuery{
		ChatID:      chatID,
		RequestedBy: actorID,
	})
	require.NoError(t, err)
	require.NotNil(t, chatResult)
	require.NotNil(t, chatResult.Chat)
	require.NotNil(t, chatResult.Chat.Status)
	require.NotNil(t, chatResult.Chat.Priority)
	require.NotNil(t, chatResult.Chat.DueDate)
	assert.Equal(t, string(taskdomain.StatusDone), *chatResult.Chat.Status)
	assert.Equal(t, string(taskdomain.PriorityHigh), *chatResult.Chat.Priority)
	assert.Nil(t, chatResult.Chat.AssignedTo, "assignee should be cleared")
	assert.WithinDuration(t, dueDate, *chatResult.Chat.DueDate, time.Second)

	byChat, err := env.taskRepo.FindByChatID(ctx, chatID)
	require.NoError(t, err)
	assert.Equal(t, chatID, byChat.ChatID)
	assert.Equal(t, taskdomain.StatusDone, byChat.Status)
	assert.Equal(t, taskdomain.PriorityHigh, byChat.Priority)
	assert.Nil(t, byChat.AssignedTo, "board card should be unassigned")
	if assert.NotNil(t, byChat.DueDate) {
		assert.WithinDuration(t, dueDate, *byChat.DueDate, time.Second)
	}

	doneStatus := taskdomain.StatusDone
	highPriority := taskdomain.PriorityHigh
	boardFilters := taskapp.Filters{
		WorkspaceID: &workspaceID,
		Status:      &doneStatus,
		Priority:    &highPriority,
	}

	boardTasks, err := env.taskRepo.List(ctx, boardFilters)
	require.NoError(t, err)

	matchCount := 0
	for _, item := range boardTasks {
		if item.ChatID == chatID {
			matchCount++
		}
	}
	assert.Equal(t, 1, matchCount, "board query must return exactly one card for chat")

	boardCount, err := env.taskRepo.Count(ctx, boardFilters)
	require.NoError(t, err)
	assert.Equal(t, 1, boardCount, "board count must stay deduplicated for chat")

	chatRM := env.mustFindChatReadModel(t, ctx, chatID)
	assert.Equal(t, string(taskdomain.StatusDone), chatRM.Status)
	assert.Equal(t, string(taskdomain.PriorityHigh), chatRM.Priority)
	assert.Nil(t, chatRM.AssignedTo)
	if assert.NotNil(t, chatRM.DueDate) {
		assert.WithinDuration(t, dueDate, *chatRM.DueDate, time.Second)
	}

	taskCount, err := env.db.Collection(mongoinfra.CollectionTaskReadModel).
		CountDocuments(ctx, bson.M{"chat_id": chatID.String()})
	require.NoError(t, err)
	assert.Equal(t, int64(1), taskCount, "task read model must not contain duplicates")

	chatCount, err := env.db.Collection(mongoinfra.CollectionChatReadModel).
		CountDocuments(ctx, bson.M{"chat_id": chatID.String()})
	require.NoError(t, err)
	assert.Equal(t, int64(1), chatCount, "chat read model must not contain duplicates")
}

func retryOnOptimisticConflict(
	ctx context.Context,
	maxAttempts int,
	run func(context.Context) error,
) error {
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := run(ctx); err != nil {
			lastErr = err
			if !errors.Is(err, domainerrs.ErrConcurrentModification) {
				return err
			}
			continue
		}
		return nil
	}

	return lastErr
}
