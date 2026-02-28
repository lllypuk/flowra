//go:build integration

package integration_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
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
