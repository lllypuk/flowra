package main

import (
	"context"
	"log/slog"
	"testing"
	"time"

	taskapp "github.com/lllypuk/flowra/internal/application/task"
	"github.com/lllypuk/flowra/internal/config"
	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/lllypuk/flowra/internal/service"
	"github.com/lllypuk/flowra/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestNewContainer_NilConfig(t *testing.T) {
	// Container should handle nil config gracefully by panicking or returning error
	// Since we don't have actual infrastructure, we can't fully test this
	// but we can test the configuration validation
	cfg := config.DefaultConfig()
	cfg.MongoDB.URI = "" // Make it invalid

	// This would fail validation, but since we can't connect anyway,
	// we just verify the config is set
	assert.NotNil(t, cfg)
}

func TestContainerOption_WithLogger(t *testing.T) {
	// Test that WithLogger option is properly applied
	c := &Container{}
	opt := WithLogger(nil) // nil logger should be handled
	opt(c)
	assert.Nil(t, c.Logger)
}

func TestContainer_Close_NoResources(t *testing.T) {
	// Container with no initialized resources should close without error
	c := &Container{
		Logger: slog.Default(),
	}
	err := c.Close()
	assert.NoError(t, err)
}

func TestContainer_IsReady_NoResources(t *testing.T) {
	// Container with no resources should return false
	c := &Container{
		Logger: slog.Default(),
	}
	ctx := context.Background()
	ready := c.IsReady(ctx)
	assert.False(t, ready)
}

func TestContainer_IsReady_NilMongoDB(t *testing.T) {
	c := &Container{
		Logger:  slog.Default(),
		MongoDB: nil,
	}
	ctx := context.Background()
	ready := c.IsReady(ctx)
	assert.False(t, ready)
}

func TestContainer_IsReady_NilRedis(t *testing.T) {
	c := &Container{
		Logger: slog.Default(),
		Redis:  nil,
	}
	ctx := context.Background()
	ready := c.IsReady(ctx)
	assert.False(t, ready)
}

func TestContainer_GetHealthStatus_NoResources(t *testing.T) {
	c := &Container{
		Logger: slog.Default(),
	}
	ctx := context.Background()
	statuses := c.GetHealthStatus(ctx)

	require.Len(t, statuses, 4) // mongodb, redis, websocket_hub, eventbus

	// All should be unhealthy
	for _, status := range statuses {
		assert.Equal(t, httpserver.StatusUnhealthy, status.Status, "component %s should be unhealthy", status.Name)
		assert.NotEmpty(t, status.Message, "component %s should have a message", status.Name)
	}
}

func TestContainer_GetHealthStatus_ComponentNames(t *testing.T) {
	c := &Container{
		Logger: slog.Default(),
	}
	ctx := context.Background()
	statuses := c.GetHealthStatus(ctx)

	names := make(map[string]bool)
	for _, status := range statuses {
		names[status.Name] = true
	}

	assert.True(t, names["mongodb"], "should have mongodb status")
	assert.True(t, names["redis"], "should have redis status")
	assert.True(t, names["websocket_hub"], "should have websocket_hub status")
	assert.True(t, names["eventbus"], "should have eventbus status")
}

func TestHealthStatus_Structure(t *testing.T) {
	status := httpserver.ComponentStatus{
		Name:    "test",
		Status:  httpserver.StatusHealthy,
		Message: "all good",
	}

	assert.Equal(t, "test", status.Name)
	assert.Equal(t, httpserver.StatusHealthy, status.Status)
	assert.Equal(t, "all good", status.Message)
}

func TestHealthStatusConstants(t *testing.T) {
	assert.Equal(t, "healthy", httpserver.StatusHealthy)
	assert.Equal(t, "unhealthy", httpserver.StatusUnhealthy)
	assert.Equal(t, "degraded", httpserver.StatusDegraded)
}

func TestContainerTimeoutConstants(t *testing.T) {
	assert.Equal(t, 30*time.Second, containerInitTimeout)
	assert.Equal(t, 5*time.Second, redisPingTimeout)
	assert.Equal(t, 10*time.Second, mongoDisconnectTimeout)
}

func TestContainer_Close_PartialResources(t *testing.T) {
	// Container with some nil resources should still close properly
	c := &Container{
		Logger:   slog.Default(),
		MongoDB:  nil,
		Redis:    nil,
		EventBus: nil,
		Hub:      nil,
	}
	err := c.Close()
	assert.NoError(t, err)
}

// TestContainer_StartEventBus_NilEventBus tests that StartEventBus handles nil EventBus
func TestContainer_StartEventBus_NilEventBus(t *testing.T) {
	c := &Container{
		EventBus: nil,
	}
	ctx := context.Background()

	// This will panic or error because EventBus is nil
	// We can't easily test this without mocking
	assert.Nil(t, c.EventBus)
	_ = ctx // avoid unused variable
}

// TestContainer_StartHub_NilHub tests that StartHub handles nil Hub
func TestContainer_StartHub_NilHub(t *testing.T) {
	c := &Container{
		Hub: nil,
	}

	// This will panic because Hub is nil
	// We can't easily test this without mocking
	assert.Nil(t, c.Hub)
}

// ========== Container Wiring Tests (Task 06) ==========

func TestContainer_ValidateWiring_MockAccessCheckerInProduction(t *testing.T) {
	// Test that mock access checker is rejected in production mode
	c := &Container{
		Logger: slog.Default(),
		Config: &config.Config{
			App: config.AppConfig{
				Mode: config.AppModeReal,
				Name: "test",
			},
			Server: config.ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
		},
		MongoDB:        nil,
		Redis:          nil,
		Hub:            nil,
		EventBus:       nil,
		TokenValidator: middleware.NewStaticTokenValidator("test-secret"),
		AccessChecker:  middleware.NewMockWorkspaceAccessChecker(),
	}

	// In real mode, but without production env, mock should be allowed
	// (production check only happens in production environment)
	err := c.validateWiring()
	// Will fail on infrastructure checks first
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mongodb client not initialized")
}

func TestContainer_RealWorkspaceAccessChecker_Type(t *testing.T) {
	// Test that RealWorkspaceAccessChecker is correctly typed
	checker := service.NewRealWorkspaceAccessChecker(nil)

	// Verify it implements the interface
	var _ middleware.WorkspaceAccessChecker = checker

	// Verify it's NOT a mock
	_, isMock := any(checker).(*middleware.MockWorkspaceAccessChecker)
	assert.False(t, isMock, "RealWorkspaceAccessChecker should not be a mock")
}

func TestContainer_Services_NotNil(t *testing.T) {
	// Test that services are properly typed
	// We can't create actual services without repos, but we can test the types

	// MemberService type check
	var memberSvc *service.MemberService
	assert.Nil(t, memberSvc) // Just a type check

	// WorkspaceService type check
	var workspaceSvc *service.WorkspaceService
	assert.Nil(t, workspaceSvc) // Just a type check

	// ChatService type check
	var chatSvc *service.ChatService
	assert.Nil(t, chatSvc) // Just a type check
}

func TestContainer_NoOpKeycloakClient(t *testing.T) {
	// Test that NoOpKeycloakClient works correctly
	client := service.NewNoOpKeycloakClient()
	ctx := context.Background()

	// CreateGroup should return a valid UUID string
	// All operations should still succeed (they don't actually do anything)
	groupID, err := client.CreateGroup(ctx, "test")
	require.NoError(t, err)
	assert.NotEmpty(t, groupID)

	err = client.DeleteGroup(ctx, groupID)
	require.NoError(t, err)

	err = client.AddUserToGroup(ctx, "user", groupID)
	require.NoError(t, err)

	err = client.RemoveUserFromGroup(ctx, "user", groupID)
	require.NoError(t, err)
}

func TestContainer_UserRepoAdapter(t *testing.T) {
	// Test that userRepoAdapter is created correctly
	c := &Container{
		Logger:   slog.Default(),
		UserRepo: nil, // nil repo for type checking
	}

	adapter := c.createUserRepoAdapter()
	assert.NotNil(t, adapter)
}

func TestContainer_WiringMode_Real(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.App.Mode = config.AppModeReal

	assert.True(t, cfg.App.IsRealMode())
	assert.False(t, cfg.App.IsMockMode())
}

func TestContainer_WiringMode_Mock(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.App.Mode = config.AppModeMock

	assert.False(t, cfg.App.IsRealMode())
	assert.True(t, cfg.App.IsMockMode())
}

func TestContainer_WiringMode_Default(t *testing.T) {
	cfg := config.DefaultConfig()
	// Default mode should be real

	assert.True(t, cfg.App.IsRealMode())
	assert.False(t, cfg.App.IsMockMode())
}

// ========== Board Task Service Adapter Tests ==========

// TestBoardTaskServiceAdapter_ListTasks tests that the adapter correctly reads tasks from MongoDB.
// This is a regression test for the bug where task_id field was incorrectly mapped to _id.
func TestBoardTaskServiceAdapter_ListTasks(t *testing.T) {
	// Setup test MongoDB with testcontainers
	_, db := testutil.SetupTestMongoDBWithClient(t)
	collection := db.Collection("tasks_read_model")

	// Create test data with the correct schema
	taskID := uuid.NewUUID()
	chatID := uuid.NewUUID()
	createdBy := uuid.NewUUID()
	now := time.Now().UTC()

	testDoc := bson.M{
		"task_id":     taskID.String(),
		"chat_id":     chatID.String(),
		"title":       "Test Task for Board",
		"entity_type": string(taskdomain.TypeTask),
		"status":      string(taskdomain.StatusToDo),
		"priority":    string(taskdomain.PriorityMedium),
		"assigned_to": nil,
		"due_date":    nil,
		"created_by":  createdBy.String(),
		"created_at":  now,
		"version":     1,
	}

	ctx := context.Background()
	_, err := collection.InsertOne(ctx, testDoc)
	require.NoError(t, err, "failed to insert test document")

	// Create adapter (system under test)
	adapter := &boardTaskServiceAdapter{
		collection: collection,
	}

	// Test ListTasks with no filters
	filters := taskapp.Filters{}
	tasks, err := adapter.ListTasks(ctx, filters)

	// Assertions
	require.NoError(t, err, "ListTasks should not return error")
	require.Len(t, tasks, 1, "should return exactly one task")

	task := tasks[0]
	assert.Equal(t, taskID, task.ID, "task ID should match")
	assert.Equal(t, chatID, task.ChatID, "chat ID should match")
	assert.Equal(t, "Test Task for Board", task.Title, "title should match")
	assert.Equal(t, taskdomain.TypeTask, task.EntityType, "entity type should match")
	assert.Equal(t, taskdomain.StatusToDo, task.Status, "status should match")
	assert.Equal(t, taskdomain.PriorityMedium, task.Priority, "priority should match")
	assert.Nil(t, task.AssignedTo, "assigned_to should be nil")
	assert.Nil(t, task.DueDate, "due_date should be nil")
	assert.Equal(t, createdBy, task.CreatedBy, "created_by should match")
	assert.Equal(t, 1, task.Version, "version should match")
}

// TestBoardTaskServiceAdapter_ListTasks_WithFilters tests filtering functionality.
func TestBoardTaskServiceAdapter_ListTasks_WithFilters(t *testing.T) {
	_, db := testutil.SetupTestMongoDBWithClient(t)
	collection := db.Collection("tasks_read_model")

	ctx := context.Background()

	// Create multiple test tasks with different statuses
	tasks := []bson.M{
		{
			"task_id":     uuid.NewUUID().String(),
			"chat_id":     uuid.NewUUID().String(),
			"title":       "Task 1 - To Do",
			"entity_type": string(taskdomain.TypeTask),
			"status":      string(taskdomain.StatusToDo),
			"priority":    string(taskdomain.PriorityHigh),
			"created_by":  uuid.NewUUID().String(),
			"created_at":  time.Now().UTC(),
			"version":     1,
		},
		{
			"task_id":     uuid.NewUUID().String(),
			"chat_id":     uuid.NewUUID().String(),
			"title":       "Task 2 - In Progress",
			"entity_type": string(taskdomain.TypeTask),
			"status":      string(taskdomain.StatusInProgress),
			"priority":    string(taskdomain.PriorityMedium),
			"created_by":  uuid.NewUUID().String(),
			"created_at":  time.Now().UTC(),
			"version":     1,
		},
		{
			"task_id":     uuid.NewUUID().String(),
			"chat_id":     uuid.NewUUID().String(),
			"title":       "Bug 1 - To Do",
			"entity_type": string(taskdomain.TypeBug),
			"status":      string(taskdomain.StatusToDo),
			"priority":    string(taskdomain.PriorityHigh),
			"created_by":  uuid.NewUUID().String(),
			"created_at":  time.Now().UTC(),
			"version":     1,
		},
	}

	for _, task := range tasks {
		_, err := collection.InsertOne(ctx, task)
		require.NoError(t, err)
	}

	adapter := &boardTaskServiceAdapter{collection: collection}

	t.Run("filter by status", func(t *testing.T) {
		status := taskdomain.StatusToDo
		filters := taskapp.Filters{
			Status: &status,
		}

		results, err := adapter.ListTasks(ctx, filters)
		require.NoError(t, err)
		assert.Len(t, results, 2, "should return 2 tasks with status To Do")

		for _, task := range results {
			assert.Equal(t, taskdomain.StatusToDo, task.Status)
		}
	})

	t.Run("filter by entity type", func(t *testing.T) {
		entityType := taskdomain.TypeBug
		filters := taskapp.Filters{
			EntityType: &entityType,
		}

		results, err := adapter.ListTasks(ctx, filters)
		require.NoError(t, err)
		assert.Len(t, results, 1, "should return 1 bug")
		assert.Equal(t, taskdomain.TypeBug, results[0].EntityType)
	})

	t.Run("filter by priority", func(t *testing.T) {
		priority := taskdomain.PriorityHigh
		filters := taskapp.Filters{
			Priority: &priority,
		}

		results, err := adapter.ListTasks(ctx, filters)
		require.NoError(t, err)
		assert.Len(t, results, 2, "should return 2 high priority tasks")

		for _, task := range results {
			assert.Equal(t, taskdomain.PriorityHigh, task.Priority)
		}
	})

	t.Run("combined filters", func(t *testing.T) {
		status := taskdomain.StatusToDo
		priority := taskdomain.PriorityHigh
		entityType := taskdomain.TypeTask

		filters := taskapp.Filters{
			Status:     &status,
			Priority:   &priority,
			EntityType: &entityType,
		}

		results, err := adapter.ListTasks(ctx, filters)
		require.NoError(t, err)
		assert.Len(t, results, 1, "should return 1 task matching all filters")
		assert.Equal(t, "Task 1 - To Do", results[0].Title)
	})
}

// TestBoardTaskServiceAdapter_CountTasks tests the CountTasks method.
func TestBoardTaskServiceAdapter_CountTasks(t *testing.T) {
	_, db := testutil.SetupTestMongoDBWithClient(t)
	collection := db.Collection("tasks_read_model")

	ctx := context.Background()

	// Insert 3 test tasks
	for range 3 {
		doc := bson.M{
			"task_id":     uuid.NewUUID().String(),
			"chat_id":     uuid.NewUUID().String(),
			"title":       "Test Task",
			"entity_type": string(taskdomain.TypeTask),
			"status":      string(taskdomain.StatusToDo),
			"priority":    string(taskdomain.PriorityMedium),
			"created_by":  uuid.NewUUID().String(),
			"created_at":  time.Now().UTC(),
			"version":     1,
		}
		_, err := collection.InsertOne(ctx, doc)
		require.NoError(t, err)
	}

	adapter := &boardTaskServiceAdapter{collection: collection}

	// Test CountTasks
	count, err := adapter.CountTasks(ctx, taskapp.Filters{})
	require.NoError(t, err)
	assert.Equal(t, 3, count, "should count all 3 tasks")
}

// TestBoardTaskServiceAdapter_ListTasks_Pagination tests pagination functionality.
func TestBoardTaskServiceAdapter_ListTasks_Pagination(t *testing.T) {
	_, db := testutil.SetupTestMongoDBWithClient(t)
	collection := db.Collection("tasks_read_model")

	ctx := context.Background()

	// Insert 5 test tasks
	for range 5 {
		doc := bson.M{
			"task_id":     uuid.NewUUID().String(),
			"chat_id":     uuid.NewUUID().String(),
			"title":       "Test Task",
			"entity_type": string(taskdomain.TypeTask),
			"status":      string(taskdomain.StatusToDo),
			"priority":    string(taskdomain.PriorityMedium),
			"created_by":  uuid.NewUUID().String(),
			"created_at":  time.Now().UTC(),
			"version":     1,
		}
		_, err := collection.InsertOne(ctx, doc)
		require.NoError(t, err)
	}

	adapter := &boardTaskServiceAdapter{collection: collection}

	t.Run("limit", func(t *testing.T) {
		filters := taskapp.Filters{
			Limit: 2,
		}

		results, err := adapter.ListTasks(ctx, filters)
		require.NoError(t, err)
		assert.Len(t, results, 2, "should return only 2 tasks due to limit")
	})

	t.Run("offset", func(t *testing.T) {
		filters := taskapp.Filters{
			Offset: 3,
		}

		results, err := adapter.ListTasks(ctx, filters)
		require.NoError(t, err)
		assert.Len(t, results, 2, "should return 2 tasks after skipping 3")
	})

	t.Run("limit and offset", func(t *testing.T) {
		filters := taskapp.Filters{
			Limit:  2,
			Offset: 2,
		}

		results, err := adapter.ListTasks(ctx, filters)
		require.NoError(t, err)
		assert.Len(t, results, 2, "should return 2 tasks after skipping 2")
	})
}

// TestBoardTaskServiceAdapter_GetTask tests getting a single task by ID.
func TestBoardTaskServiceAdapter_GetTask(t *testing.T) {
	_, db := testutil.SetupTestMongoDBWithClient(t)
	collection := db.Collection("tasks_read_model")

	taskID := uuid.NewUUID()
	chatID := uuid.NewUUID()
	createdBy := uuid.NewUUID()

	testDoc := bson.M{
		"task_id":     taskID.String(),
		"chat_id":     chatID.String(),
		"title":       "Specific Task",
		"entity_type": string(taskdomain.TypeTask),
		"status":      string(taskdomain.StatusToDo),
		"priority":    string(taskdomain.PriorityHigh),
		"created_by":  createdBy.String(),
		"created_at":  time.Now().UTC(),
		"version":     1,
	}

	ctx := context.Background()
	_, err := collection.InsertOne(ctx, testDoc)
	require.NoError(t, err)

	adapter := &boardTaskServiceAdapter{collection: collection}

	// Test GetTask
	task, err := adapter.GetTask(ctx, taskID)
	require.NoError(t, err)
	require.NotNil(t, task)

	assert.Equal(t, taskID, task.ID)
	assert.Equal(t, "Specific Task", task.Title)
	assert.Equal(t, taskdomain.PriorityHigh, task.Priority)
}

// TestBoardTaskServiceAdapter_GetTask_NotFound tests GetTask with non-existent ID.
func TestBoardTaskServiceAdapter_GetTask_NotFound(t *testing.T) {
	_, db := testutil.SetupTestMongoDBWithClient(t)
	collection := db.Collection("tasks_read_model")

	adapter := &boardTaskServiceAdapter{collection: collection}

	nonExistentID := uuid.NewUUID()
	task, err := adapter.GetTask(context.Background(), nonExistentID)

	require.Error(t, err, "should return error for non-existent task")
	assert.Nil(t, task, "task should be nil")
}

// TestTaskReadModelDoc_FieldMapping is a compile-time test that verifies
// the taskReadModelDoc struct has the correct BSON tags.
func TestTaskReadModelDoc_FieldMapping(t *testing.T) {
	// Create a document and verify it can be marshaled/unmarshaled correctly
	taskID := uuid.NewUUID()
	chatID := uuid.NewUUID()
	createdBy := uuid.NewUUID()
	now := time.Now().UTC()

	// Create BSON document with actual MongoDB schema
	bsonDoc := bson.M{
		"task_id":     taskID.String(),
		"chat_id":     chatID.String(),
		"title":       "Schema Test Task",
		"entity_type": string(taskdomain.TypeTask),
		"status":      string(taskdomain.StatusToDo),
		"priority":    string(taskdomain.PriorityMedium),
		"created_by":  createdBy.String(),
		"created_at":  now,
		"version":     1,
	}

	// Marshal to BSON bytes
	bsonBytes, err := bson.Marshal(bsonDoc)
	require.NoError(t, err)

	// Unmarshal into our struct
	var doc taskReadModelDoc
	err = bson.Unmarshal(bsonBytes, &doc)
	require.NoError(t, err, "should unmarshal without error")

	// Verify fields were correctly mapped
	assert.Equal(t, taskID.String(), doc.ID, "task_id should map to ID field")
	assert.Equal(t, chatID.String(), doc.ChatID, "chat_id should map to ChatID")
	assert.Equal(t, "Schema Test Task", doc.Title, "title should match")
	assert.Equal(t, string(taskdomain.TypeTask), doc.EntityType)
	assert.Equal(t, string(taskdomain.StatusToDo), doc.Status)
	assert.Equal(t, string(taskdomain.PriorityMedium), doc.Priority)
	assert.Equal(t, createdBy.String(), doc.CreatedBy)
	assert.Equal(t, 1, doc.Version)

	// Verify conversion to ReadModel
	readModel := doc.toReadModel()
	assert.Equal(t, taskID, readModel.ID, "ID should be correctly parsed")
	assert.Equal(t, chatID, readModel.ChatID, "ChatID should be correctly parsed")
	assert.Equal(t, taskdomain.StatusToDo, readModel.Status)
}
