// Package main provides the API server entry point.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/lllypuk/flowra/internal/application/notification"
	"github.com/lllypuk/flowra/internal/config"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	wshandler "github.com/lllypuk/flowra/internal/handler/websocket"
	"github.com/lllypuk/flowra/internal/infrastructure/eventbus"
	"github.com/lllypuk/flowra/internal/infrastructure/eventstore"
	"github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	"github.com/lllypuk/flowra/internal/infrastructure/websocket"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Container initialization timeouts.
const (
	containerInitTimeout   = 30 * time.Second
	redisPingTimeout       = 5 * time.Second
	mongoDisconnectTimeout = 10 * time.Second
)

// Health status constants.
const (
	healthStatusHealthy   = "healthy"
	healthStatusUnhealthy = "unhealthy"
	healthStatusDegraded  = "degraded"
)

// Container holds all application dependencies and manages their lifecycle.
type Container struct {
	// Configuration
	Config *config.Config
	Logger *slog.Logger

	// Infrastructure
	MongoDB      *mongo.Client
	MongoDBName  string
	Redis        *redis.Client
	EventStore   *eventstore.MongoEventStore
	EventBus     *eventbus.RedisEventBus
	Hub          *websocket.Hub
	NotifHandler *eventbus.NotificationHandler
	LogHandler   *eventbus.LoggingHandler

	// Repositories
	UserRepo         *mongodb.MongoUserRepository
	WorkspaceRepo    *mongodb.MongoWorkspaceRepository
	ChatRepo         *mongodb.MongoChatRepository
	MessageRepo      *mongodb.MongoMessageRepository
	TaskRepo         *mongodb.MongoTaskRepository
	NotificationRepo *mongodb.MongoNotificationRepository

	// Use Cases
	CreateNotificationUC *notification.CreateNotificationUseCase

	// HTTP Handlers
	AuthHandler         *httphandler.AuthHandler
	WorkspaceHandler    *httphandler.WorkspaceHandler
	ChatHandler         *httphandler.ChatHandler
	MessageHandler      *httphandler.MessageHandler
	TaskHandler         *httphandler.TaskHandler
	NotificationHandler *httphandler.NotificationHandler
	UserHandler         *httphandler.UserHandler
	WSHandler           *wshandler.Handler

	// Auth middleware components
	TokenValidator middleware.TokenValidator
	AccessChecker  middleware.WorkspaceAccessChecker
}

// ContainerOption configures the Container.
type ContainerOption func(*Container)

// WithLogger sets a custom logger for the container.
func WithLogger(logger *slog.Logger) ContainerOption {
	return func(c *Container) {
		c.Logger = logger
	}
}

// NewContainer creates a new dependency injection container.
func NewContainer(cfg *config.Config, opts ...ContainerOption) (*Container, error) {
	c := &Container{
		Config: cfg,
		Logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(c)
	}

	// Initialize all components in order
	if err := c.setupInfrastructure(); err != nil {
		// Clean up any partially initialized resources
		_ = c.Close()
		return nil, fmt.Errorf("failed to setup infrastructure: %w", err)
	}

	c.setupRepositories()
	c.setupUseCases()
	c.setupEventHandlers()
	c.setupHTTPHandlers()

	return c, nil
}

// setupInfrastructure initializes infrastructure components (MongoDB, Redis, EventBus, Hub).
func (c *Container) setupInfrastructure() error {
	ctx, cancel := context.WithTimeout(context.Background(), containerInitTimeout)
	defer cancel()

	// Setup MongoDB
	if err := c.setupMongoDB(ctx); err != nil {
		return fmt.Errorf("mongodb: %w", err)
	}

	// Setup Redis
	if err := c.setupRedis(ctx); err != nil {
		return fmt.Errorf("redis: %w", err)
	}

	// Setup EventStore
	c.setupEventStore()

	// Setup EventBus
	c.setupEventBus()

	// Setup WebSocket Hub
	c.setupHub()

	return nil
}

// setupMongoDB initializes the MongoDB client.
func (c *Container) setupMongoDB(ctx context.Context) error {
	clientOpts := options.Client().
		ApplyURI(c.Config.MongoDB.URI).
		SetMaxPoolSize(c.Config.MongoDB.MaxPoolSize)

	client, connectErr := mongo.Connect(clientOpts)
	if connectErr != nil {
		return fmt.Errorf("failed to connect: %w", connectErr)
	}

	// Verify connection
	pingCtx, cancel := context.WithTimeout(ctx, c.Config.MongoDB.Timeout)
	defer cancel()

	if pingErr := client.Ping(pingCtx, nil); pingErr != nil {
		return fmt.Errorf("failed to ping: %w", pingErr)
	}

	c.MongoDB = client
	c.MongoDBName = c.Config.MongoDB.Database

	c.Logger.InfoContext(ctx, "connected to MongoDB",
		slog.String("database", c.Config.MongoDB.Database),
	)

	return nil
}

// setupRedis initializes the Redis client.
func (c *Container) setupRedis(ctx context.Context) error {
	c.Redis = redis.NewClient(&redis.Options{
		Addr:     c.Config.Redis.Addr,
		Password: c.Config.Redis.Password,
		DB:       c.Config.Redis.DB,
		PoolSize: c.Config.Redis.PoolSize,
	})

	// Verify connection
	pingCtx, cancel := context.WithTimeout(ctx, redisPingTimeout)
	defer cancel()

	if pingErr := c.Redis.Ping(pingCtx).Err(); pingErr != nil {
		return fmt.Errorf("failed to ping: %w", pingErr)
	}

	c.Logger.InfoContext(ctx, "connected to Redis",
		slog.String("addr", c.Config.Redis.Addr),
	)

	return nil
}

// setupEventStore initializes the event store.
func (c *Container) setupEventStore() {
	c.EventStore = eventstore.NewMongoEventStore(c.MongoDB, c.MongoDBName)
	c.Logger.Debug("event store initialized")
}

// setupEventBus initializes the event bus.
func (c *Container) setupEventBus() {
	c.EventBus = eventbus.NewRedisEventBus(
		c.Redis,
		eventbus.WithLogger(c.Logger),
		eventbus.WithChannelPrefix(c.Config.EventBus.RedisChannelPrefix),
	)

	c.Logger.Debug("event bus initialized",
		slog.String("type", c.Config.EventBus.Type),
		slog.String("prefix", c.Config.EventBus.RedisChannelPrefix),
	)
}

// setupHub initializes the WebSocket hub.
func (c *Container) setupHub() {
	c.Hub = websocket.NewHub(
		websocket.WithHubLogger(c.Logger),
	)

	c.Logger.Debug("websocket hub initialized")
}

// setupRepositories initializes all repository implementations.
func (c *Container) setupRepositories() {
	db := c.MongoDB.Database(c.MongoDBName)

	// User repository
	c.UserRepo = mongodb.NewMongoUserRepository(db.Collection("users"))

	// Workspace repository
	c.WorkspaceRepo = mongodb.NewMongoWorkspaceRepository(
		db.Collection("workspaces"),
		db.Collection("workspace_members"),
	)

	// Chat repository (event sourced)
	c.ChatRepo = mongodb.NewMongoChatRepository(
		c.EventStore,
		db.Collection("chats_read_model"),
	)

	// Message repository
	c.MessageRepo = mongodb.NewMongoMessageRepository(db.Collection("messages"))

	// Task repository (event sourced)
	c.TaskRepo = mongodb.NewMongoTaskRepository(
		c.EventStore,
		db.Collection("tasks_read_model"),
	)

	// Notification repository
	c.NotificationRepo = mongodb.NewMongoNotificationRepository(db.Collection("notifications"))

	c.Logger.Debug("repositories initialized")
}

// setupUseCases initializes all use cases.
func (c *Container) setupUseCases() {
	// Notification use case is needed by event handlers
	c.CreateNotificationUC = notification.NewCreateNotificationUseCase(
		c.NotificationRepo,
	)

	c.Logger.Debug("use cases initialized")
}

// setupEventHandlers initializes and registers event handlers with the event bus.
func (c *Container) setupEventHandlers() {
	// Create notification handler for processing domain events
	c.NotifHandler = eventbus.NewNotificationHandler(
		c.CreateNotificationUC,
		eventbus.WithNotificationLogger(c.Logger),
	)

	// Create logging handler for debugging
	c.LogHandler = eventbus.NewLoggingHandler(c.Logger)

	c.Logger.Debug("event handlers initialized")
}

// registerEventHandlers registers all event handlers with the event bus.
// This should be called after the event bus is ready to start.
func (c *Container) registerEventHandlers() error {
	return eventbus.RegisterAllHandlers(
		c.EventBus,
		c.NotifHandler,
		c.LogHandler,
		c.Logger,
	)
}

// setupHTTPHandlers initializes all HTTP handlers.
func (c *Container) setupHTTPHandlers() {
	// Setup mock auth service and user repo for development
	// In production, these would be replaced with real implementations
	mockAuthService := httphandler.NewMockAuthService()
	mockUserRepo := httphandler.NewMockUserRepository()

	c.AuthHandler = httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

	// Setup workspace handler with mock services
	mockWorkspaceService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()
	c.WorkspaceHandler = httphandler.NewWorkspaceHandler(mockWorkspaceService, mockMemberService)

	// Setup chat handler with mock service
	mockChatService := httphandler.NewMockChatService()
	c.ChatHandler = httphandler.NewChatHandler(mockChatService)

	// Setup WebSocket handler
	c.WSHandler = wshandler.NewHandler(
		c.Hub,
		wshandler.WithHandlerLogger(c.Logger),
		wshandler.WithHandlerConfig(wshandler.HandlerConfig{
			ReadBufferSize:  c.Config.WebSocket.ReadBufferSize,
			WriteBufferSize: c.Config.WebSocket.WriteBufferSize,
			Logger:          c.Logger,
		}),
	)

	// Setup token validator for auth middleware (dev mode)
	c.TokenValidator = middleware.NewStaticTokenValidator(c.Config.Auth.JWTSecret)

	// Setup workspace access checker
	c.AccessChecker = middleware.NewMockWorkspaceAccessChecker()

	c.Logger.Debug("HTTP handlers initialized")
}

// Close gracefully closes all container resources.
// Resources are closed in reverse order of initialization.
func (c *Container) Close() error {
	c.Logger.Info("closing container resources...")

	var errs []error

	// Close Hub
	if c.Hub != nil {
		c.Hub.Stop()
		c.Logger.Debug("websocket hub stopped")
	}

	// Close EventBus
	if c.EventBus != nil {
		if err := c.EventBus.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("event bus shutdown: %w", err))
		} else {
			c.Logger.Debug("event bus stopped")
		}
	}

	// Close Redis
	if c.Redis != nil {
		if err := c.Redis.Close(); err != nil {
			errs = append(errs, fmt.Errorf("redis close: %w", err))
		} else {
			c.Logger.Debug("redis connection closed")
		}
	}

	// Close MongoDB
	if c.MongoDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), mongoDisconnectTimeout)
		defer cancel()

		if err := c.MongoDB.Disconnect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("mongodb disconnect: %w", err))
		} else {
			c.Logger.Debug("mongodb connection closed")
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	c.Logger.Info("all container resources closed")
	return nil
}

// StartEventBus starts the event bus and registers all handlers.
// This should be called before the HTTP server starts accepting requests.
func (c *Container) StartEventBus(ctx context.Context) error {
	// Register event handlers first
	if err := c.registerEventHandlers(); err != nil {
		return fmt.Errorf("failed to register event handlers: %w", err)
	}

	// Start the event bus in a goroutine
	go func() {
		if err := c.EventBus.Start(ctx); err != nil {
			c.Logger.Error("event bus error", slog.String("error", err.Error()))
		}
	}()

	c.Logger.InfoContext(ctx, "event bus started")
	return nil
}

// StartHub starts the WebSocket hub.
// This should be called before the HTTP server starts accepting requests.
func (c *Container) StartHub(ctx context.Context) {
	go c.Hub.Run(ctx)
	c.Logger.InfoContext(ctx, "websocket hub started")
}

// IsReady checks if all infrastructure components are healthy.
func (c *Container) IsReady(ctx context.Context) bool {
	// Check MongoDB
	if c.MongoDB == nil {
		return false
	}
	if err := c.MongoDB.Ping(ctx, nil); err != nil {
		c.Logger.WarnContext(ctx, "mongodb health check failed", slog.String("error", err.Error()))
		return false
	}

	// Check Redis
	if c.Redis == nil {
		return false
	}
	if err := c.Redis.Ping(ctx).Err(); err != nil {
		c.Logger.WarnContext(ctx, "redis health check failed", slog.String("error", err.Error()))
		return false
	}

	// Check Hub
	if c.Hub == nil || !c.Hub.IsRunning() {
		c.Logger.WarnContext(ctx, "websocket hub is not running")
		return false
	}

	return true
}

// HealthStatus represents the health status of a component.
type HealthStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// GetHealthStatus returns detailed health status of all components.
func (c *Container) GetHealthStatus(ctx context.Context) []HealthStatus {
	var statuses []HealthStatus

	// MongoDB status
	mongoStatus := HealthStatus{Name: "mongodb", Status: healthStatusHealthy}
	if c.MongoDB == nil {
		mongoStatus.Status = healthStatusUnhealthy
		mongoStatus.Message = "client not initialized"
	} else if err := c.MongoDB.Ping(ctx, nil); err != nil {
		mongoStatus.Status = healthStatusUnhealthy
		mongoStatus.Message = err.Error()
	}
	statuses = append(statuses, mongoStatus)

	// Redis status
	redisStatus := HealthStatus{Name: "redis", Status: healthStatusHealthy}
	if c.Redis == nil {
		redisStatus.Status = healthStatusUnhealthy
		redisStatus.Message = "client not initialized"
	} else if err := c.Redis.Ping(ctx).Err(); err != nil {
		redisStatus.Status = healthStatusUnhealthy
		redisStatus.Message = err.Error()
	}
	statuses = append(statuses, redisStatus)

	// WebSocket Hub status
	hubStatus := HealthStatus{Name: "websocket_hub", Status: healthStatusHealthy}
	if c.Hub == nil {
		hubStatus.Status = healthStatusUnhealthy
		hubStatus.Message = "hub not initialized"
	} else if !c.Hub.IsRunning() {
		hubStatus.Status = healthStatusUnhealthy
		hubStatus.Message = "hub not running"
	}
	statuses = append(statuses, hubStatus)

	// EventBus status
	eventBusStatus := HealthStatus{Name: "eventbus", Status: healthStatusHealthy}
	if c.EventBus == nil {
		eventBusStatus.Status = healthStatusUnhealthy
		eventBusStatus.Message = "event bus not initialized"
	} else if !c.EventBus.IsRunning() {
		eventBusStatus.Status = healthStatusDegraded
		eventBusStatus.Message = "event bus not running"
	}
	statuses = append(statuses, eventBusStatus)

	return statuses
}
