package mongodb_test

import (
	"context"
	"testing"
	"time"

	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	"github.com/lllypuk/flowra/tests/mocks"
	"github.com/lllypuk/flowra/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Helper functions

// setupTestRepository creates testovye repozitorii s mock event store
// returns takzhe readModelColl for tests read model
func setupTestRepository(t *testing.T) (
	*mongodb.MongoChatRepository,
	*mongodb.MongoChatReadModelRepository,
	*mocks.MockEventStore,
	*mongo.Collection,
) {
	t.Helper()

	eventStore := mocks.NewMockEventStore()

	// ispolzuem testcontainers for MongoDB 6
	_, db := testutil.SetupTestMongoDBWithClient(t)
	readModelColl := db.Collection("chat_read_model")

	// Creating indexes
	ctx := context.Background()
	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "chat_id", Value: 1}},
	}
	_, _ = readModelColl.Indexes().CreateOne(ctx, indexModel)

	commandRepo := mongodb.NewMongoChatRepository(eventStore, readModelColl)
	queryRepo := mongodb.NewMongoChatReadModelRepository(readModelColl, eventStore)

	return commandRepo, queryRepo, eventStore, readModelColl
}

// addChatToReadModel adds chat in read model for tests
func addChatToReadModel(ctx context.Context, t *testing.T, coll *mongo.Collection, c *chat.Chat) {
	t.Helper()

	// preobrazuem participants in stroki
	participantStrs := make([]string, len(c.Participants()))
	for i, p := range c.Participants() {
		participantStrs[i] = p.UserID().String()
	}

	// formiruem dokument for read model
	doc := bson.M{
		"chat_id":      c.ID().String(),
		"workspace_id": c.WorkspaceID().String(),
		"type":         string(c.Type()),
		"is_public":    c.IsPublic(),
		"created_by":   c.CreatedBy().String(),
		"created_at":   c.CreatedAt(),
		"participants": participantStrs,
	}

	// dobavlyaem dopolnitelnye fields for typed chats
	if c.Type() != chat.TypeDiscussion {
		doc["title"] = c.Title()
		doc["status"] = c.Status()
		doc["priority"] = c.Priority()

		if c.AssigneeID() != nil {
			doc["assigned_to"] = c.AssigneeID().String()
		}

		if c.DueDate() != nil {
			doc["due_date"] = *c.DueDate()
		}

		if c.Type() == chat.TypeBug {
			doc["severity"] = c.Severity()
		}
	}

	_, err := coll.InsertOne(ctx, doc)
	require.NoError(t, err)
}

// Tests for MongoChatRepository (Command Repository)

// TestMongoChatRepository_Load_SaveRoundTrip checks save and load chat
func TestMongoChatRepository_Load_SaveRoundTrip(t *testing.T) {
	commandRepo, _, eventStore, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	// Creating New chat
	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()
	c, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, userID)
	require.NoError(t, err)

	// dobavlyaem participant
	err = c.AddParticipant(uuid.NewUUID(), chat.RoleMember)
	require.NoError(t, err)

	// Save
	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// Checking, that event byli sav
	assert.NotEmpty(t, eventStore.AllEvents()[c.ID().String()])

	// Load
	loaded, err := commandRepo.Load(ctx, c.ID())
	require.NoError(t, err)

	assert.Equal(t, c.ID(), loaded.ID())
	assert.Equal(t, c.WorkspaceID(), loaded.WorkspaceID())
	assert.Equal(t, c.Type(), loaded.Type())
	assert.Equal(t, c.IsPublic(), loaded.IsPublic())
	assert.Len(t, loaded.Participants(), len(c.Participants()))
}

// TestMongoChatRepository_Load_NotFound checks work missing chat
func TestMongoChatRepository_Load_NotFound(t *testing.T) {
	commandRepo, _, _, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	_, err := commandRepo.Load(ctx, uuid.NewUUID())
	assert.Equal(t, errs.ErrNotFound, err)
}

// TestMongoChatRepository_Load_InvalidInput checks valid vhodnyh dannyh
func TestMongoChatRepository_Load_InvalidInput(t *testing.T) {
	commandRepo, _, _, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	// Zero UUID
	_, err := commandRepo.Load(ctx, uuid.UUID(""))
	assert.Equal(t, errs.ErrInvalidInput, err)
}

// TestMongoChatRepository_Save_NoChanges checks save bez izmeneniy
func TestMongoChatRepository_Save_NoChanges(t *testing.T) {
	commandRepo, _, eventStore, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()
	c, err := chat.NewChat(workspaceID, chat.TypeTask, false, userID)
	require.NoError(t, err)

	// Save once
	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	eventCountBefore := len(eventStore.AllEvents()[c.ID().String()])

	// Save again without changes
	err = commandRepo.Save(ctx, c)
	require.NoError(t, err) // Should be idempotent

	eventCountAfter := len(eventStore.AllEvents()[c.ID().String()])
	assert.Equal(t, eventCountBefore, eventCountAfter)
}

// TestMongoChatRepository_Save_InvalidInput checks valid nil chat
func TestMongoChatRepository_Save_InvalidInput(t *testing.T) {
	commandRepo, _, _, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	// nil chat
	err := commandRepo.Save(ctx, nil)
	assert.Equal(t, errs.ErrInvalidInput, err)
}

// TestMongoChatRepository_GetEvents checks retrieval events chat
func TestMongoChatRepository_GetEvents(t *testing.T) {
	commandRepo, _, _, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	// Creating chat
	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()
	c, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, userID)
	require.NoError(t, err)

	// dobavlyaem participant
	err = c.AddParticipant(uuid.NewUUID(), chat.RoleMember)
	require.NoError(t, err)

	// Saving
	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// poluchaem event
	events, err := commandRepo.GetEvents(ctx, c.ID())
	require.NoError(t, err)

	assert.NotEmpty(t, events)
	assert.GreaterOrEqual(t, len(events), 2) // ChatCreated + ParticipantAdded
}

// TestMongoChatRepository_GetEvents_NotFound checks work existing chat
func TestMongoChatRepository_GetEvents_NotFound(t *testing.T) {
	commandRepo, _, _, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	_, err := commandRepo.GetEvents(ctx, uuid.NewUUID())
	assert.Equal(t, errs.ErrNotFound, err)
}

// TestMongoChatRepository_GetEvents_InvalidInput checks valid vhodnyh dannyh
func TestMongoChatRepository_GetEvents_InvalidInput(t *testing.T) {
	commandRepo, _, _, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	_, err := commandRepo.GetEvents(ctx, uuid.UUID(""))
	assert.Equal(t, errs.ErrInvalidInput, err)
}

// TestMongoChatRepository_TypedChats checks work s typed chatami</parameter>
func TestMongoChatRepository_TypedChats(t *testing.T) {
	commandRepo, _, _, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()

	// Testing Task chat - creating Discussion and convert to Task
	taskChat, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, userID)
	require.NoError(t, err)

	err = taskChat.ConvertToTask("Test Task", userID)
	require.NoError(t, err)

	err = taskChat.SetPriority("Medium", userID)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, taskChat)
	require.NoError(t, err)

	loaded, err := commandRepo.Load(ctx, taskChat.ID())
	require.NoError(t, err)

	assert.Equal(t, chat.TypeTask, loaded.Type())
	assert.Equal(t, "Test Task", loaded.Title())
	assert.Equal(t, "Medium", loaded.Priority())
}

// TestMongoChatRepository_participantManagement checks upravlenie uchastnikami
func TestMongoChatRepository_ParticipantManagement(t *testing.T) {
	commandRepo, _, _, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()

	c, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, userID)
	require.NoError(t, err)

	// dobavlyaem participants
	participant1 := uuid.NewUUID()
	participant2 := uuid.NewUUID()

	err = c.AddParticipant(participant1, chat.RoleMember)
	require.NoError(t, err)

	err = c.AddParticipant(participant2, chat.RoleAdmin)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// Loading and checking
	loaded, err := commandRepo.Load(ctx, c.ID())
	require.NoError(t, err)

	assert.Len(t, loaded.Participants(), 3) // sozdatel + 2 participant
	assert.True(t, loaded.HasParticipant(userID))
	assert.True(t, loaded.HasParticipant(participant1))
	assert.True(t, loaded.HasParticipant(participant2))

	// udalyaem participant from load aggregate
	err = loaded.RemoveParticipant(participant1)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, loaded)
	require.NoError(t, err)

	// Loading snova
	loaded, err = commandRepo.Load(ctx, c.ID())
	require.NoError(t, err)

	assert.Len(t, loaded.Participants(), 2)
	assert.False(t, loaded.HasParticipant(participant1))
}

// TestMongoChatRepository_ChatStatusChanges checks change status chat
func TestMongoChatRepository_ChatStatusChanges(t *testing.T) {
	commandRepo, _, _, _ := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()

	// Creating Discussion chat and convert to Task
	c, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, userID)
	require.NoError(t, err)

	err = c.ConvertToTask("Test Task", userID)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// izmenyaem status
	err = c.ChangeStatus("In Progress", userID)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// Loading and checking
	loaded, err := commandRepo.Load(ctx, c.ID())
	require.NoError(t, err)

	assert.Equal(t, "In Progress", loaded.Status())

	// izmenyaem prioritet ispolzuya loaded aggregate
	err = loaded.SetPriority("High", userID)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, loaded)
	require.NoError(t, err)

	loaded2, err := commandRepo.Load(ctx, c.ID())
	require.NoError(t, err)

	assert.Equal(t, "High", loaded2.Priority())
}

// Tests for MongoChatReadModelRepository (Query Repository)

// TestMongoChatReadModelRepository_FindByWorkspace checks search chats po workspace
func TestMongoChatReadModelRepository_FindByWorkspace(t *testing.T) {
	_, queryRepo, _, readModelColl := setupTestRepository(t)
	if queryRepo == nil {
		return
	}

	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()

	// Creating several chats in odnom workspace
	for range 3 {
		c, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, userID)
		require.NoError(t, err)

		addChatToReadModel(ctx, t, readModelColl, c)
	}

	// Creating chat in drugom workspace
	otherWorkspaceID := uuid.NewUUID()
	otherChat, err := chat.NewChat(otherWorkspaceID, chat.TypeDiscussion, false, userID)
	require.NoError(t, err)

	addChatToReadModel(ctx, t, readModelColl, otherChat)

	filters := chatapp.Filters{
		Offset: 0,
		Limit:  10,
	}

	chats, err := queryRepo.FindByWorkspace(ctx, workspaceID, filters)
	require.NoError(t, err)

	assert.NotNil(t, chats)
	assert.Len(t, chats, 3)

	for _, c := range chats {
		assert.Equal(t, workspaceID, c.WorkspaceID)
	}
}

// TestMongoChatReadModelRepository_FindByWorkspace_WithTypeFilter checks filtratsiyu po tipu
func TestMongoChatReadModelRepository_FindByWorkspace_WithTypeFilter(t *testing.T) {
	_, queryRepo, _, readModelColl := setupTestRepository(t)
	if queryRepo == nil {
		return
	}

	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()

	// Creating chaty raznyh tipov
	taskChat, err := chat.NewChat(workspaceID, chat.TypeTask, false, userID)
	require.NoError(t, err)
	addChatToReadModel(ctx, t, readModelColl, taskChat)

	discussionChat, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, userID)
	require.NoError(t, err)
	addChatToReadModel(ctx, t, readModelColl, discussionChat)

	chatType := chat.TypeTask
	filters := chatapp.Filters{
		Type:   &chatType,
		Offset: 0,
		Limit:  10,
	}

	chats, err := queryRepo.FindByWorkspace(ctx, workspaceID, filters)
	require.NoError(t, err)

	assert.NotNil(t, chats)
	assert.Len(t, chats, 1)
	assert.Equal(t, chat.TypeTask, chats[0].Type)
}

// TestMongoChatReadModelRepository_FindByparticipant checks search chats po uchastniku
func TestMongoChatReadModelRepository_FindByParticipant(t *testing.T) {
	_, queryRepo, _, readModelColl := setupTestRepository(t)
	if queryRepo == nil {
		return
	}

	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()

	// Creating several chats and dobavlyaem user
	participantID := uuid.NewUUID()

	for range 3 {
		c, err := chat.NewChat(workspaceID, chat.TypeTask, false, userID)
		require.NoError(t, err)

		// dobavlyaem participant
		err = c.AddParticipant(participantID, chat.RoleMember)
		require.NoError(t, err)

		addChatToReadModel(ctx, t, readModelColl, c)
	}

	// dobavlyaem chat bez it is participant
	otherChat, err := chat.NewChat(workspaceID, chat.TypeTask, false, userID)
	require.NoError(t, err)
	addChatToReadModel(ctx, t, readModelColl, otherChat)

	chats, err := queryRepo.FindByParticipant(ctx, participantID, 0, 10)
	require.NoError(t, err)

	assert.NotNil(t, chats)
	assert.Len(t, chats, 3)
}

// TestMongoChatReadModelRepository_Count checks podschet chats
func TestMongoChatReadModelRepository_Count(t *testing.T) {
	_, queryRepo, _, readModelColl := setupTestRepository(t)
	if queryRepo == nil {
		return
	}

	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()

	// dobavlyaem chaty in workspace
	for range 5 {
		c, err := chat.NewChat(workspaceID, chat.TypeTask, false, userID)
		require.NoError(t, err)

		addChatToReadModel(ctx, t, readModelColl, c)
	}

	// dobavlyaem chat in drugoy workspace
	otherWorkspaceID := uuid.NewUUID()
	otherChat, err := chat.NewChat(otherWorkspaceID, chat.TypeTask, false, userID)
	require.NoError(t, err)

	addChatToReadModel(ctx, t, readModelColl, otherChat)

	count, err := queryRepo.Count(ctx, workspaceID)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, count, 5)
}

// TestMongoChatRepository_InputValidation checks valid vhodnyh dannyh
func TestMongoChatRepository_InputValidation(t *testing.T) {
	commandRepo, queryRepo, _, _ := setupTestRepository(t)
	if commandRepo == nil || queryRepo == nil {
		return
	}

	ctx := context.Background()

	// Load s nulevym UUID
	_, err := commandRepo.Load(ctx, uuid.UUID(""))
	assert.Equal(t, errs.ErrInvalidInput, err)

	// Save s nil chatom
	err = commandRepo.Save(ctx, nil)
	assert.Equal(t, errs.ErrInvalidInput, err)

	// FindByWorkspace s nulevym UUID
	filters := chatapp.Filters{Offset: 0, Limit: 10}
	_, err = queryRepo.FindByWorkspace(ctx, uuid.UUID(""), filters)
	assert.Equal(t, errs.ErrInvalidInput, err)

	// FindByparticipant s nulevym UUID
	_, err = queryRepo.FindByParticipant(ctx, uuid.UUID(""), 0, 10)
	assert.Equal(t, errs.ErrInvalidInput, err)

	// Count s nulevym UUID
	_, err = queryRepo.Count(ctx, uuid.UUID(""))
	assert.Equal(t, errs.ErrInvalidInput, err)
}

// TestMongoChatRepository_ComplexWorkflow checks slozhnyy workflow creating and updating chat
func TestMongoChatRepository_ComplexWorkflow(t *testing.T) {
	commandRepo, queryRepo, _, _ := setupTestRepository(t)
	if commandRepo == nil || queryRepo == nil {
		return
	}

	ctx := context.Background()

	workspaceID := uuid.NewUUID()
	creatorID := uuid.NewUUID()

	// 1. Creating Discussion chat and convert to Task
	c, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, creatorID)
	require.NoError(t, err)

	err = c.ConvertToTask("Complex Task", creatorID)
	require.NoError(t, err)

	err = c.SetPriority("Medium", creatorID)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// value ispolnitelya
	assigneeID := uuid.NewUUID()
	err = c.AssignUser(&assigneeID, creatorID)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// Setting srok
	dueDate := time.Now().Add(24 * time.Hour)
	err = c.SetDueDate(&dueDate, creatorID)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// dobavlyaem participants
	participantID := uuid.NewUUID()
	err = c.AddParticipant(participantID, chat.RoleMember)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// izmenyaem status
	err = c.ChangeStatus("In Progress", creatorID)
	require.NoError(t, err)

	err = commandRepo.Save(ctx, c)
	require.NoError(t, err)

	// Loading finalnoe state
	loaded, err := commandRepo.Load(ctx, c.ID())
	require.NoError(t, err)

	// Checking all changing
	assert.Equal(t, chat.TypeTask, loaded.Type())
	assert.Equal(t, "Complex Task", loaded.Title())
	assert.Equal(t, "Medium", loaded.Priority())
	assert.Equal(t, "In Progress", loaded.Status())
	assert.Equal(t, assigneeID, *loaded.AssigneeID())
	assert.Len(t, loaded.Participants(), 2) // sozdatel + uchastnik

	// Checking event
	events, err := commandRepo.GetEvents(ctx, c.ID())
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(events), 6) // minimum 6 events for takogo workflow
}
