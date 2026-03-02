package mongodb_test

import (
	"context"
	"testing"
	"time"

	taskapp "github.com/lllypuk/flowra/internal/application/task"
	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	mongodbinfra "github.com/lllypuk/flowra/internal/infrastructure/mongodb"
	"github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	"github.com/lllypuk/flowra/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMongoTaskRepository_FindByIDAndChatID(t *testing.T) {
	_, db := testutil.SetupTestMongoDBWithClient(t)
	coll := db.Collection(mongodbinfra.CollectionTaskReadModel)
	repo := mongodb.NewMongoTaskRepository(nil, coll)

	taskID := uuid.NewUUID()
	chatID := uuid.NewUUID()
	createdBy := uuid.NewUUID()
	assignee := uuid.NewUUID()
	dueDate := time.Now().UTC().Truncate(time.Second)
	attachmentID := uuid.NewUUID()

	_, err := coll.InsertOne(context.Background(), bson.M{
		"task_id":     taskID.String(),
		"chat_id":     chatID.String(),
		"title":       "Fix login",
		"entity_type": string(taskdomain.TypeBug),
		"status":      string(taskdomain.StatusInProgress),
		"priority":    string(taskdomain.PriorityHigh),
		"severity":    "Critical",
		"assigned_to": assignee.String(),
		"due_date":    dueDate,
		"created_by":  createdBy.String(),
		"created_at":  dueDate,
		"version":     7,
		"attachments": []bson.M{{
			"file_id":   attachmentID.String(),
			"file_name": "spec.pdf",
			"file_size": int64(1024),
			"mime_type": "application/pdf",
		}},
	})
	require.NoError(t, err)

	byID, err := repo.FindByID(context.Background(), taskID)
	require.NoError(t, err)
	require.NotNil(t, byID)
	assert.Equal(t, chatID, byID.ChatID)
	assert.Equal(t, taskdomain.TypeBug, byID.EntityType)
	assert.Equal(t, taskdomain.StatusInProgress, byID.Status)
	assert.Equal(t, taskdomain.PriorityHigh, byID.Priority)
	assert.Equal(t, "Critical", byID.Severity)
	require.NotNil(t, byID.AssignedTo)
	assert.Equal(t, assignee, *byID.AssignedTo)
	require.NotNil(t, byID.DueDate)
	assert.Equal(t, dueDate, *byID.DueDate)
	require.Len(t, byID.Attachments, 1)
	assert.Equal(t, attachmentID, byID.Attachments[0].FileID)

	byChatID, err := repo.FindByChatID(context.Background(), chatID)
	require.NoError(t, err)
	require.NotNil(t, byChatID)
	assert.Equal(t, taskID, byChatID.ID)
}

func TestMongoTaskRepository_ListAndCountWithFilters(t *testing.T) {
	_, db := testutil.SetupTestMongoDBWithClient(t)
	coll := db.Collection(mongodbinfra.CollectionTaskReadModel)
	repo := mongodb.NewMongoTaskRepository(nil, coll)

	workspaceTaskA := uuid.NewUUID()
	workspaceTaskB := uuid.NewUUID()
	createdBy := uuid.NewUUID()

	docs := []any{
		bson.M{
			"task_id":     workspaceTaskA.String(),
			"chat_id":     workspaceTaskA.String(),
			"title":       "A",
			"entity_type": string(taskdomain.TypeTask),
			"status":      string(taskdomain.StatusBacklog),
			"priority":    string(taskdomain.PriorityMedium),
			"created_by":  createdBy.String(),
			"created_at":  time.Now().UTC(),
			"version":     1,
		},
		bson.M{
			"task_id":     workspaceTaskB.String(),
			"chat_id":     workspaceTaskB.String(),
			"title":       "B",
			"entity_type": string(taskdomain.TypeTask),
			"status":      string(taskdomain.StatusDone),
			"priority":    string(taskdomain.PriorityHigh),
			"created_by":  createdBy.String(),
			"created_at":  time.Now().UTC().Add(1 * time.Minute),
			"version":     1,
		},
	}
	_, err := coll.InsertMany(context.Background(), docs)
	require.NoError(t, err)

	status := taskdomain.StatusDone
	items, err := repo.List(context.Background(), taskapp.Filters{Status: &status})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, workspaceTaskB, items[0].ID)

	count, err := repo.Count(context.Background(), taskapp.Filters{Status: &status})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
