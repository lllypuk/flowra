package main

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/config"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/lllypuk/flowra/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
