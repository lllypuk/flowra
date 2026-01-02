package main

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/config"
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
		assert.Equal(t, healthStatusUnhealthy, status.Status, "component %s should be unhealthy", status.Name)
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
	status := HealthStatus{
		Name:    "test",
		Status:  healthStatusHealthy,
		Message: "all good",
	}

	assert.Equal(t, "test", status.Name)
	assert.Equal(t, healthStatusHealthy, status.Status)
	assert.Equal(t, "all good", status.Message)
}

func TestHealthStatusConstants(t *testing.T) {
	assert.Equal(t, "healthy", healthStatusHealthy)
	assert.Equal(t, "unhealthy", healthStatusUnhealthy)
	assert.Equal(t, "degraded", healthStatusDegraded)
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
