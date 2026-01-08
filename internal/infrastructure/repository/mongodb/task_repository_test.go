package mongodb_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/domain/errs"
	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
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

// setupTaskTestRepository creates testovye repozitorii s mock event store
func setupTaskTestRepository(t *testing.T) (
	*mongodb.MongoTaskRepository,
	*mongodb.MongoTaskQueryRepository,
	*mocks.MockEventStore,
) {
	t.Helper()

	eventStore := mocks.NewMockEventStore()

	// ispolzuem testcontainers for MongoDB 6
	_, db := testutil.SetupTestMongoDBWithClient(t)
	readModelColl := db.Collection("task_read_model")

	// Creating indexes
	ctx := context.Background()
	indexModels := []mongo.IndexModel{
		{Keys: bson.D{{Key: "task_id", Value: 1}}},
		{Keys: bson.D{{Key: "chat_id", Value: 1}}},
		{Keys: bson.D{{Key: "assigned_to", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "priority", Value: 1}}},
	}
	_, _ = readModelColl.Indexes().CreateMany(ctx, indexModels)

	commandRepo := mongodb.NewMongoTaskRepository(eventStore, readModelColl)
	queryRepo := mongodb.NewMongoTaskQueryRepository(readModelColl, eventStore)

	return commandRepo, queryRepo, eventStore
}

// setupTaskFullRepository creates full repozitoriy for tests
func setupTaskFullRepository(t *testing.T) *mongodb.MongoTaskFullRepository {
	t.Helper()

	eventStore := mocks.NewMockEventStore()

	_, db := testutil.SetupTestMongoDBWithClient(t)
	readModelColl := db.Collection("task_read_model")

	// Creating indexes
	ctx := context.Background()
	indexModels := []mongo.IndexModel{
		{Keys: bson.D{{Key: "task_id", Value: 1}}},
		{Keys: bson.D{{Key: "chat_id", Value: 1}}},
		{Keys: bson.D{{Key: "assigned_to", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
	}
	_, _ = readModelColl.Indexes().CreateMany(ctx, indexModels)

	repo := mongodb.NewMongoTaskFullRepository(eventStore, readModelColl)

	return repo
}

// createTestTask creates test aggregate tasks
func createTestTask(t *testing.T) *taskdomain.Aggregate {
	t.Helper()

	taskID := uuid.NewUUID()
	chatID := uuid.NewUUID()
	createdBy := uuid.NewUUID()

	aggregate := taskdomain.NewTaskAggregate(taskID)
	err := aggregate.Create(
		chatID,
		"Test Task",
		taskdomain.TypeTask,
		taskdomain.PriorityMedium,
		nil,
		nil,
		createdBy,
	)
	require.NoError(t, err)

	return aggregate
}

// Tests for MongoTaskRepository (Command Repository)

func TestMongoTaskRepository_Save_And_Load(t *testing.T) {
	commandRepo, _, eventStore := setupTaskTestRepository(t)

	ctx := context.Background()

	// Create task aggregate
	taskID := uuid.NewUUID()
	chatID := uuid.NewUUID()
	createdBy := uuid.NewUUID()

	aggregate := taskdomain.NewTaskAggregate(taskID)
	err := aggregate.Create(
		chatID,
		"Test Task",
		taskdomain.TypeTask,
		taskdomain.PriorityMedium,
		nil,
		nil,
		createdBy,
	)
	require.NoError(t, err)

	// Save
	err = commandRepo.Save(ctx, aggregate)
	require.NoError(t, err)

	// Verify events were saved
	assert.NotEmpty(t, eventStore.AllEvents()[taskID.String()])

	// Load
	loaded, err := commandRepo.Load(ctx, taskID)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, taskID, loaded.ID())
	assert.Equal(t, chatID, loaded.ChatID())
	assert.Equal(t, "Test Task", loaded.Title())
	assert.Equal(t, taskdomain.TypeTask, loaded.EntityType())
	assert.Equal(t, taskdomain.StatusToDo, loaded.Status())
	assert.Equal(t, taskdomain.PriorityMedium, loaded.Priority())
}

func TestMongoTaskRepository_Load_NotFound(t *testing.T) {
	commandRepo, _, _ := setupTaskTestRepository(t)

	ctx := context.Background()

	_, err := commandRepo.Load(ctx, uuid.NewUUID())
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

func TestMongoTaskRepository_Load_InvalidInput(t *testing.T) {
	commandRepo, _, _ := setupTaskTestRepository(t)

	ctx := context.Background()

	_, err := commandRepo.Load(ctx, uuid.UUID(""))
	assert.ErrorIs(t, err, errs.ErrInvalidInput)
}

func TestMongoTaskRepository_Save_NoChanges(t *testing.T) {
	commandRepo, _, _ := setupTaskTestRepository(t)

	ctx := context.Background()

	// Create and save task
	aggregate := createTestTask(t)
	err := commandRepo.Save(ctx, aggregate)
	require.NoError(t, err)

	// Load the task (it is clears uncommittedEvents)
	loaded, err := commandRepo.Load(ctx, aggregate.ID())
	require.NoError(t, err)

	// Save without changes - should succeed without doing anything
	err = commandRepo.Save(ctx, loaded)
	assert.NoError(t, err)
}

func TestMongoTaskRepository_Save_InvalidInput(t *testing.T) {
	commandRepo, _, _ := setupTaskTestRepository(t)

	ctx := context.Background()

	err := commandRepo.Save(ctx, nil)
	assert.ErrorIs(t, err, errs.ErrInvalidInput)
}

func TestMongoTaskRepository_GetEvents(t *testing.T) {
	commandRepo, _, _ := setupTaskTestRepository(t)

	ctx := context.Background()

	// Create and save task
	aggregate := createTestTask(t)
	err := commandRepo.Save(ctx, aggregate)
	require.NoError(t, err)

	// Get events
	events, err := commandRepo.GetEvents(ctx, aggregate.ID())
	require.NoError(t, err)

	// Should have at least one event (TaskCreated)
	assert.NotEmpty(t, events)
}

func TestMongoTaskRepository_GetEvents_NotFound(t *testing.T) {
	commandRepo, _, _ := setupTaskTestRepository(t)

	ctx := context.Background()

	_, err := commandRepo.GetEvents(ctx, uuid.NewUUID())
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

func TestMongoTaskRepository_GetEvents_InvalidInput(t *testing.T) {
	commandRepo, _, _ := setupTaskTestRepository(t)

	ctx := context.Background()

	_, err := commandRepo.GetEvents(ctx, uuid.UUID(""))
	assert.ErrorIs(t, err, errs.ErrInvalidInput)
}

func TestMongoTaskRepository_ConcurrentModification(t *testing.T) {
	commandRepo, _, _ := setupTaskTestRepository(t)

	ctx := context.Background()

	// Create and save initial task
	taskID := uuid.NewUUID()
	chatID := uuid.NewUUID()
	createdBy := uuid.NewUUID()

	aggregate := taskdomain.NewTaskAggregate(taskID)
	_ = aggregate.Create(chatID, "Test Task", taskdomain.TypeTask, taskdomain.PriorityMedium, nil, nil, createdBy)
	_ = commandRepo.Save(ctx, aggregate)

	// Load two instances
	instance1, _ := commandRepo.Load(ctx, taskID)
	instance2, _ := commandRepo.Load(ctx, taskID)

	// Modify and save first instance
	_ = instance1.ChangeStatus(taskdomain.StatusInProgress, createdBy)
	err1 := commandRepo.Save(ctx, instance1)
	require.NoError(t, err1)

	// Modify and try to save second instance - should fail
	_ = instance2.ChangePriority(taskdomain.PriorityHigh, createdBy)
	err2 := commandRepo.Save(ctx, instance2)

	require.Error(t, err2)
	assert.ErrorIs(t, err2, errs.ErrConcurrentModification)
}

func TestMongoTaskRepository_WithAssignee(t *testing.T) {
	commandRepo, _, _ := setupTaskTestRepository(t)

	ctx := context.Background()

	taskID := uuid.NewUUID()
	chatID := uuid.NewUUID()
	createdBy := uuid.NewUUID()
	assignee := uuid.NewUUID()

	aggregate := taskdomain.NewTaskAggregate(taskID)
	err := aggregate.Create(
		chatID,
		"Task with Assignee",
		taskdomain.TypeTask,
		taskdomain.PriorityHigh,
		&assignee,
		nil,
		createdBy,
	)
	require.NoError(t, err)

	// Save
	err = commandRepo.Save(ctx, aggregate)
	require.NoError(t, err)

	// Load
	loaded, err := commandRepo.Load(ctx, taskID)
	require.NoError(t, err)

	// Assert
	require.NotNil(t, loaded.AssignedTo())
	assert.Equal(t, assignee, *loaded.AssignedTo())
}

func TestMongoTaskRepository_WithDueDate(t *testing.T) {
	commandRepo, _, _ := setupTaskTestRepository(t)

	ctx := context.Background()

	taskID := uuid.NewUUID()
	chatID := uuid.NewUUID()
	createdBy := uuid.NewUUID()
	dueDate := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Millisecond)

	aggregate := taskdomain.NewTaskAggregate(taskID)
	err := aggregate.Create(
		chatID,
		"Task with DueDate",
		taskdomain.TypeTask,
		taskdomain.PriorityMedium,
		nil,
		&dueDate,
		createdBy,
	)
	require.NoError(t, err)

	// Save
	err = commandRepo.Save(ctx, aggregate)
	require.NoError(t, err)

	// Load
	loaded, err := commandRepo.Load(ctx, taskID)
	require.NoError(t, err)

	// Assert
	require.NotNil(t, loaded.DueDate())
	assert.True(t, dueDate.Equal(*loaded.DueDate()))
}

func TestMongoTaskRepository_StatusChanges(t *testing.T) {
	commandRepo, _, _ := setupTaskTestRepository(t)

	ctx := context.Background()

	// Create task
	aggregate := createTestTask(t)
	createdBy := aggregate.CreatedBy()

	err := commandRepo.Save(ctx, aggregate)
	require.NoError(t, err)

	// Load and change status
	loaded, err := commandRepo.Load(ctx, aggregate.ID())
	require.NoError(t, err)
	assert.Equal(t, taskdomain.StatusToDo, loaded.Status())

	// Change to InProgress
	err = loaded.ChangeStatus(taskdomain.StatusInProgress, createdBy)
	require.NoError(t, err)
	err = commandRepo.Save(ctx, loaded)
	require.NoError(t, err)

	// Load and verify
	reloaded, err := commandRepo.Load(ctx, aggregate.ID())
	require.NoError(t, err)
	assert.Equal(t, taskdomain.StatusInProgress, reloaded.Status())
}

// Tests for MongoTaskQueryRepository (Query Repository)

func TestMongoTaskQueryRepository_FindByID(t *testing.T) {
	commandRepo, queryRepo, _ := setupTaskTestRepository(t)

	ctx := context.Background()

	// Create and save task
	aggregate := createTestTask(t)
	err := commandRepo.Save(ctx, aggregate)
	require.NoError(t, err)

	// Find by ID
	result, err := queryRepo.FindByID(ctx, aggregate.ID())
	require.NoError(t, err)

	assert.Equal(t, aggregate.ID(), result.ID)
	assert.Equal(t, aggregate.ChatID(), result.ChatID)
	assert.Equal(t, aggregate.Title(), result.Title)
	assert.Equal(t, aggregate.Status(), result.Status)
}

func TestMongoTaskQueryRepository_FindByID_NotFound(t *testing.T) {
	_, queryRepo, _ := setupTaskTestRepository(t)

	ctx := context.Background()

	_, err := queryRepo.FindByID(ctx, uuid.NewUUID())
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

func TestMongoTaskQueryRepository_FindByID_InvalidInput(t *testing.T) {
	_, queryRepo, _ := setupTaskTestRepository(t)

	ctx := context.Background()

	_, err := queryRepo.FindByID(ctx, uuid.UUID(""))
	assert.ErrorIs(t, err, errs.ErrInvalidInput)
}

func TestMongoTaskQueryRepository_FindByChatID(t *testing.T) {
	commandRepo, queryRepo, _ := setupTaskTestRepository(t)

	ctx := context.Background()

	// Create and save task
	aggregate := createTestTask(t)
	err := commandRepo.Save(ctx, aggregate)
	require.NoError(t, err)

	// Find by ChatID
	result, err := queryRepo.FindByChatID(ctx, aggregate.ChatID())
	require.NoError(t, err)

	assert.Equal(t, aggregate.ID(), result.ID)
	assert.Equal(t, aggregate.ChatID(), result.ChatID)
}

func TestMongoTaskQueryRepository_FindByChatID_NotFound(t *testing.T) {
	_, queryRepo, _ := setupTaskTestRepository(t)

	ctx := context.Background()

	_, err := queryRepo.FindByChatID(ctx, uuid.NewUUID())
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

func TestMongoTaskQueryRepository_FindByAssignee(t *testing.T) {
	repo := setupTaskFullRepository(t)

	ctx := context.Background()

	// Create tasks with same assignee
	assignee := uuid.NewUUID()
	createdBy := uuid.NewUUID()

	for i := range 3 {
		taskID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		aggregate := taskdomain.NewTaskAggregate(taskID)
		_ = aggregate.Create(
			chatID,
			fmt.Sprintf("Task %d", i),
			taskdomain.TypeTask,
			taskdomain.PriorityMedium,
			&assignee,
			nil,
			createdBy,
		)
		_ = repo.Save(ctx, aggregate)
	}

	// Query
	results, err := repo.FindByAssignee(ctx, assignee, taskapp.Filters{Limit: 10})
	require.NoError(t, err)

	assert.Len(t, results, 3)
	for _, r := range results {
		require.NotNil(t, r.AssignedTo)
		assert.Equal(t, assignee, *r.AssignedTo)
	}
}

func TestMongoTaskQueryRepository_FindByAssignee_InvalidInput(t *testing.T) {
	_, queryRepo, _ := setupTaskTestRepository(t)

	ctx := context.Background()

	_, err := queryRepo.FindByAssignee(ctx, uuid.UUID(""), taskapp.Filters{})
	assert.ErrorIs(t, err, errs.ErrInvalidInput)
}

func TestMongoTaskQueryRepository_FindByStatus(t *testing.T) {
	repo := setupTaskFullRepository(t)

	ctx := context.Background()

	createdBy := uuid.NewUUID()

	// Create tasks with different statuses
	for i := range 2 {
		taskID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		aggregate := taskdomain.NewTaskAggregate(taskID)
		_ = aggregate.Create(
			chatID,
			fmt.Sprintf("Task ToDo %d", i),
			taskdomain.TypeTask,
			taskdomain.PriorityMedium,
			nil,
			nil,
			createdBy,
		)
		_ = repo.Save(ctx, aggregate)
	}

	// Create one InProgress task
	taskID := uuid.NewUUID()
	chatID := uuid.NewUUID()
	aggregate := taskdomain.NewTaskAggregate(taskID)
	_ = aggregate.Create(chatID, "Task InProgress", taskdomain.TypeTask, taskdomain.PriorityMedium, nil, nil, createdBy)
	_ = repo.Save(ctx, aggregate)

	loaded, _ := repo.Load(ctx, taskID)
	_ = loaded.ChangeStatus(taskdomain.StatusInProgress, createdBy)
	_ = repo.Save(ctx, loaded)

	// Query ToDo tasks
	todoResults, err := repo.FindByStatus(ctx, taskdomain.StatusToDo, taskapp.Filters{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, todoResults, 2)

	// Query InProgress tasks
	inProgressResults, err := repo.FindByStatus(ctx, taskdomain.StatusInProgress, taskapp.Filters{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, inProgressResults, 1)
}

func TestMongoTaskQueryRepository_List(t *testing.T) {
	repo := setupTaskFullRepository(t)

	ctx := context.Background()

	createdBy := uuid.NewUUID()

	// Create multiple tasks
	for i := range 5 {
		taskID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		aggregate := taskdomain.NewTaskAggregate(taskID)
		_ = aggregate.Create(
			chatID,
			fmt.Sprintf("Task %d", i),
			taskdomain.TypeTask,
			taskdomain.PriorityMedium,
			nil,
			nil,
			createdBy,
		)
		_ = repo.Save(ctx, aggregate)
	}

	// List all
	results, err := repo.List(ctx, taskapp.Filters{Limit: 10})
	require.NoError(t, err)

	assert.Len(t, results, 5)
}

func TestMongoTaskQueryRepository_List_WithFilters(t *testing.T) {
	repo := setupTaskFullRepository(t)

	ctx := context.Background()

	createdBy := uuid.NewUUID()

	// Create tasks with different priorities
	for i := range 3 {
		taskID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		aggregate := taskdomain.NewTaskAggregate(taskID)
		_ = aggregate.Create(
			chatID,
			fmt.Sprintf("High Priority Task %d", i),
			taskdomain.TypeTask,
			taskdomain.PriorityHigh,
			nil,
			nil,
			createdBy,
		)
		_ = repo.Save(ctx, aggregate)
	}

	for i := range 2 {
		taskID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		aggregate := taskdomain.NewTaskAggregate(taskID)
		_ = aggregate.Create(
			chatID,
			fmt.Sprintf("Low Priority Task %d", i),
			taskdomain.TypeTask,
			taskdomain.PriorityLow,
			nil,
			nil,
			createdBy,
		)
		_ = repo.Save(ctx, aggregate)
	}

	// List with priority filter
	highPriority := taskdomain.PriorityHigh
	results, err := repo.List(ctx, taskapp.Filters{
		Priority: &highPriority,
		Limit:    10,
	})
	require.NoError(t, err)

	assert.Len(t, results, 3)
	for _, r := range results {
		assert.Equal(t, taskdomain.PriorityHigh, r.Priority)
	}
}

func TestMongoTaskQueryRepository_List_Pagination(t *testing.T) {
	repo := setupTaskFullRepository(t)

	ctx := context.Background()

	createdBy := uuid.NewUUID()

	// Create 10 tasks
	for i := range 10 {
		taskID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		aggregate := taskdomain.NewTaskAggregate(taskID)
		_ = aggregate.Create(
			chatID,
			fmt.Sprintf("Task %d", i),
			taskdomain.TypeTask,
			taskdomain.PriorityMedium,
			nil,
			nil,
			createdBy,
		)
		_ = repo.Save(ctx, aggregate)
	}

	// Get first page
	page1, err := repo.List(ctx, taskapp.Filters{Offset: 0, Limit: 5})
	require.NoError(t, err)
	assert.Len(t, page1, 5)

	// Get second page
	page2, err := repo.List(ctx, taskapp.Filters{Offset: 5, Limit: 5})
	require.NoError(t, err)
	assert.Len(t, page2, 5)

	// Verify no overlap
	page1IDs := make(map[uuid.UUID]bool)
	for _, r := range page1 {
		page1IDs[r.ID] = true
	}
	for _, r := range page2 {
		assert.False(t, page1IDs[r.ID], "Found overlapping task in pages")
	}
}

func TestMongoTaskQueryRepository_Count(t *testing.T) {
	repo := setupTaskFullRepository(t)

	ctx := context.Background()

	createdBy := uuid.NewUUID()

	// Create tasks
	for i := range 7 {
		taskID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		aggregate := taskdomain.NewTaskAggregate(taskID)
		_ = aggregate.Create(
			chatID,
			fmt.Sprintf("Task %d", i),
			taskdomain.TypeTask,
			taskdomain.PriorityMedium,
			nil,
			nil,
			createdBy,
		)
		_ = repo.Save(ctx, aggregate)
	}

	// Count all
	count, err := repo.Count(ctx, taskapp.Filters{})
	require.NoError(t, err)

	assert.Equal(t, 7, count)
}

func TestMongoTaskQueryRepository_Count_WithFilters(t *testing.T) {
	repo := setupTaskFullRepository(t)

	ctx := context.Background()

	createdBy := uuid.NewUUID()
	assignee := uuid.NewUUID()

	// Create tasks with and without assignee
	for i := range 3 {
		taskID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		aggregate := taskdomain.NewTaskAggregate(taskID)
		_ = aggregate.Create(
			chatID,
			fmt.Sprintf("Assigned Task %d", i),
			taskdomain.TypeTask,
			taskdomain.PriorityMedium,
			&assignee,
			nil,
			createdBy,
		)
		_ = repo.Save(ctx, aggregate)
	}

	for i := range 2 {
		taskID := uuid.NewUUID()
		chatID := uuid.NewUUID()

		aggregate := taskdomain.NewTaskAggregate(taskID)
		_ = aggregate.Create(
			chatID,
			fmt.Sprintf("Unassigned Task %d", i),
			taskdomain.TypeTask,
			taskdomain.PriorityMedium,
			nil,
			nil,
			createdBy,
		)
		_ = repo.Save(ctx, aggregate)
	}

	// Count with filter
	count, err := repo.Count(ctx, taskapp.Filters{AssigneeID: &assignee})
	require.NoError(t, err)

	assert.Equal(t, 3, count)
}

func TestMongoTaskQueryRepository_EntityTypeFilter(t *testing.T) {
	repo := setupTaskFullRepository(t)

	ctx := context.Background()

	createdBy := uuid.NewUUID()

	// Create Task type
	taskID := uuid.NewUUID()
	chatID := uuid.NewUUID()
	aggregate := taskdomain.NewTaskAggregate(taskID)
	_ = aggregate.Create(chatID, "A Task", taskdomain.TypeTask, taskdomain.PriorityMedium, nil, nil, createdBy)
	_ = repo.Save(ctx, aggregate)

	// Create Bug type
	bugID := uuid.NewUUID()
	bugChatID := uuid.NewUUID()
	bugAggregate := taskdomain.NewTaskAggregate(bugID)
	_ = bugAggregate.Create(bugChatID, "A Bug", taskdomain.TypeBug, taskdomain.PriorityHigh, nil, nil, createdBy)
	_ = repo.Save(ctx, bugAggregate)

	// Filter by Task type
	taskType := taskdomain.TypeTask
	taskResults, err := repo.List(ctx, taskapp.Filters{EntityType: &taskType, Limit: 10})
	require.NoError(t, err)
	assert.Len(t, taskResults, 1)
	assert.Equal(t, taskdomain.TypeTask, taskResults[0].EntityType)

	// Filter by Bug type
	bugType := taskdomain.TypeBug
	bugResults, err := repo.List(ctx, taskapp.Filters{EntityType: &bugType, Limit: 10})
	require.NoError(t, err)
	assert.Len(t, bugResults, 1)
	assert.Equal(t, taskdomain.TypeBug, bugResults[0].EntityType)
}

func TestMongoTaskFullRepository_ImplementsInterface(t *testing.T) {
	repo := setupTaskFullRepository(t)

	// Verify it implements the full Repository interface
	var _ taskapp.Repository = repo
	var _ taskapp.CommandRepository = repo
	var _ taskapp.QueryRepository = repo
}

func TestMongoTaskRepository_ComplexWorkflow(t *testing.T) {
	repo := setupTaskFullRepository(t)

	ctx := context.Background()

	taskID := uuid.NewUUID()
	chatID := uuid.NewUUID()
	createdBy := uuid.NewUUID()
	assignee := uuid.NewUUID()

	// 1. Create task
	aggregate := taskdomain.NewTaskAggregate(taskID)
	err := aggregate.Create(
		chatID,
		"Complex Task",
		taskdomain.TypeTask,
		taskdomain.PriorityLow,
		nil,
		nil,
		createdBy,
	)
	require.NoError(t, err)
	err = repo.Save(ctx, aggregate)
	require.NoError(t, err)

	// 2. Load and assign
	loaded, err := repo.Load(ctx, taskID)
	require.NoError(t, err)
	err = loaded.Assign(&assignee, createdBy)
	require.NoError(t, err)
	err = repo.Save(ctx, loaded)
	require.NoError(t, err)

	// 3. Load and change priority
	loaded, err = repo.Load(ctx, taskID)
	require.NoError(t, err)
	err = loaded.ChangePriority(taskdomain.PriorityCritical, createdBy)
	require.NoError(t, err)
	err = repo.Save(ctx, loaded)
	require.NoError(t, err)

	// 4. Load and change status through workflow
	loaded, err = repo.Load(ctx, taskID)
	require.NoError(t, err)
	_ = loaded.ChangeStatus(taskdomain.StatusInProgress, createdBy)
	_ = repo.Save(ctx, loaded)

	loaded, _ = repo.Load(ctx, taskID)
	_ = loaded.ChangeStatus(taskdomain.StatusInReview, createdBy)
	_ = repo.Save(ctx, loaded)

	loaded, _ = repo.Load(ctx, taskID)
	_ = loaded.ChangeStatus(taskdomain.StatusDone, createdBy)
	_ = repo.Save(ctx, loaded)

	// 5. Verify final state via read model
	readModel, err := repo.FindByID(ctx, taskID)
	require.NoError(t, err)

	assert.Equal(t, "Complex Task", readModel.Title)
	assert.Equal(t, taskdomain.PriorityCritical, readModel.Priority)
	assert.Equal(t, taskdomain.StatusDone, readModel.Status)
	require.NotNil(t, readModel.AssignedTo)
	assert.Equal(t, assignee, *readModel.AssignedTo)

	// 6. Verify event history
	events, err := repo.GetEvents(ctx, taskID)
	require.NoError(t, err)
	// Created + Assigned + PriorityChanged + 3 StatusChanges = 6 events
	assert.Len(t, events, 6)
}
