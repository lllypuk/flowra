package mongodb_test

import (
	"context"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestMongoChatRepository_UpdateReadModel_AssigneeUnsetAfterClear(t *testing.T) {
	commandRepo, _, _, readModelColl := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()
	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()
	assigneeID := uuid.NewUUID()

	c, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, userID)
	require.NoError(t, err)
	require.NoError(t, c.ConvertToTask("Nullable Assignee Cleanup", userID))
	require.NoError(t, c.AssignUser(&assigneeID, userID))
	require.NoError(t, commandRepo.Save(ctx, c))

	doc := mustGetChatReadModelDoc(t, ctx, readModelColl, c.ID())
	require.Contains(t, doc, "assigned_to")
	assert.Equal(t, assigneeID.String(), doc["assigned_to"])

	require.NoError(t, c.AssignUser(nil, userID))
	require.NoError(t, commandRepo.Save(ctx, c))

	doc = mustGetChatReadModelDoc(t, ctx, readModelColl, c.ID())
	assert.NotContains(t, doc, "assigned_to")
}

func TestMongoChatRepository_UpdateReadModel_DueDateUnsetAfterClear(t *testing.T) {
	commandRepo, _, _, readModelColl := setupTestRepository(t)
	if commandRepo == nil {
		return
	}

	ctx := context.Background()
	workspaceID := uuid.NewUUID()
	userID := uuid.NewUUID()
	dueDate := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)

	c, err := chat.NewChat(workspaceID, chat.TypeDiscussion, false, userID)
	require.NoError(t, err)
	require.NoError(t, c.ConvertToTask("Nullable Due Date Cleanup", userID))
	require.NoError(t, c.SetDueDate(&dueDate, userID))
	require.NoError(t, commandRepo.Save(ctx, c))

	doc := mustGetChatReadModelDoc(t, ctx, readModelColl, c.ID())
	require.Contains(t, doc, "due_date")

	require.NoError(t, c.SetDueDate(nil, userID))
	require.NoError(t, commandRepo.Save(ctx, c))

	doc = mustGetChatReadModelDoc(t, ctx, readModelColl, c.ID())
	assert.NotContains(t, doc, "due_date")
}

func mustGetChatReadModelDoc(t *testing.T, ctx context.Context, coll *mongo.Collection, chatID uuid.UUID) bson.M {
	t.Helper()

	var doc bson.M
	err := coll.FindOne(ctx, bson.M{"chat_id": chatID.String()}).Decode(&doc)
	require.NoError(t, err)

	return doc
}
