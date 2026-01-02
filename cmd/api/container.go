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
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
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

// Container holds all application dependencies and manages their lifecycle.
// It implements httpserver.HealthChecker for unified health endpoint support.
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

// Ensure Container implements httpserver.HealthChecker.
var _ httpserver.HealthChecker = (*Container)(nil)

// ContainerOption configures the Container.
type ContainerOption func(*Container)

// WithLogger sets a custom logger for the container.
func WithLogger(logger *slog.Logger) ContainerOption {
	return func(c *Container) {
		c.Logger = logger
	}
}

// NewContainer creates a new dependency injection container.
// The wiring mode (real/mock) is determined by config.App.Mode.
func NewContainer(cfg *config.Config, opts ...ContainerOption) (*Container, error) {
	c := &Container{
		Config: cfg,
		Logger: slog.Default(),
	}

	for _, opt := range opts {
		opt(c)
	}

	// Log the wiring mode
	c.logWiringMode()

	// Initialize all components in order
	if err := c.setupInfrastructure(); err != nil {
		// Clean up any partially initialized resources
		_ = c.Close()
		return nil, fmt.Errorf("failed to setup infrastructure: %w", err)
	}

	c.setupRepositories()
	c.setupUseCases()
	c.setupEventHandlers()

	// Setup HTTP handlers based on wiring mode
	if c.Config.App.IsMockMode() {
		c.setupHTTPHandlersMock()
	} else {
		c.setupHTTPHandlersReal()
	}

	// Validate that all required components are initialized
	if err := c.validateWiring(); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("wiring validation failed: %w", err)
	}

	return c, nil
}

// logWiringMode logs the current wiring mode configuration.
func (c *Container) logWiringMode() {
	mode := c.Config.App.Mode
	if mode == "" {
		mode = config.AppModeReal
	}

	if c.Config.App.IsMockMode() {
		c.Logger.Warn("container starting in MOCK mode",
			slog.String("mode", string(mode)),
			slog.Bool("is_development", c.Config.IsDevelopment()),
			slog.Bool("is_production", c.Config.IsProduction()),
		)
	} else {
		c.Logger.Info("container starting in REAL mode",
			slog.String("mode", string(mode)),
			slog.Bool("is_development", c.Config.IsDevelopment()),
			slog.Bool("is_production", c.Config.IsProduction()),
		)
	}
}

// validateWiring ensures all required dependencies are properly initialized.
// In real mode, this is strict. In mock mode, placeholders are allowed.
func (c *Container) validateWiring() error {
	var errs []error

	// Infrastructure is always required
	if c.MongoDB == nil {
		errs = append(errs, errors.New("mongodb client not initialized"))
	}
	if c.Redis == nil {
		errs = append(errs, errors.New("redis client not initialized"))
	}
	if c.Hub == nil {
		errs = append(errs, errors.New("websocket hub not initialized"))
	}
	if c.EventBus == nil {
		errs = append(errs, errors.New("event bus not initialized"))
	}

	// Auth components are always required
	if c.TokenValidator == nil {
		errs = append(errs, errors.New("token validator not initialized"))
	}
	if c.AccessChecker == nil {
		errs = append(errs, errors.New("access checker not initialized"))
	}

	// In real mode, all handlers must be initialized (no placeholders)
	if c.Config.App.IsRealMode() {
		if c.AuthHandler == nil {
			errs = append(errs, errors.New("auth handler not initialized in real mode"))
		}
		if c.WorkspaceHandler == nil {
			errs = append(errs, errors.New("workspace handler not initialized in real mode"))
		}
		if c.ChatHandler == nil {
			errs = append(errs, errors.New("chat handler not initialized in real mode"))
		}
		if c.WSHandler == nil {
			errs = append(errs, errors.New("websocket handler not initialized in real mode"))
		}
		// Note: MessageHandler, TaskHandler, NotificationHandler, UserHandler
		// may be nil in real mode if their use cases are not fully implemented yet.
		// This will result in placeholder endpoints returning 501.
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
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

// setupHTTPHandlersReal initializes HTTP handlers with real implementations.
// This wires handlers to actual use cases and services.
func (c *Container) setupHTTPHandlersReal() {
	c.Logger.Debug("setting up HTTP handlers with REAL implementations")

	// TODO: Wire real AuthService implementation when available
	// For now, use mock but log a warning
	c.Logger.Warn("AuthHandler: using mock implementation (real auth service not yet available)")
	mockAuthService := httphandler.NewMockAuthService()
	mockUserRepo := httphandler.NewMockUserRepository()
	c.AuthHandler = httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

	// TODO: Wire real WorkspaceService implementation when available
	c.Logger.Warn("WorkspaceHandler: using mock implementation (real workspace service not yet available)")
	mockWorkspaceService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()
	c.WorkspaceHandler = httphandler.NewWorkspaceHandler(mockWorkspaceService, mockMemberService)

	// TODO: Wire real ChatService implementation when available
	c.Logger.Warn("ChatHandler: using mock implementation (real chat service not yet available)")
	mockChatService := httphandler.NewMockChatService()
	c.ChatHandler = httphandler.NewChatHandler(mockChatService)

	// WebSocket handler uses real Hub
	c.WSHandler = wshandler.NewHandler(
		c.Hub,
		wshandler.WithHandlerLogger(c.Logger),
		wshandler.WithHandlerConfig(wshandler.HandlerConfig{
			ReadBufferSize:  c.Config.WebSocket.ReadBufferSize,
			WriteBufferSize: c.Config.WebSocket.WriteBufferSize,
			Logger:          c.Logger,
		}),
	)

	// Setup token validator for auth middleware
	c.TokenValidator = middleware.NewStaticTokenValidator(c.Config.Auth.JWTSecret)

	// TODO: Wire real WorkspaceAccessChecker implementation
	c.Logger.Warn("AccessChecker: using mock implementation (real access checker not yet available)")
	c.AccessChecker = middleware.NewMockWorkspaceAccessChecker()

	// Note: MessageHandler, TaskHandler, NotificationHandler, UserHandler
	// are left nil - routes.go will create placeholder endpoints for them.
	// This is intentional until their use cases are fully implemented.

	c.Logger.Debug("HTTP handlers initialized (real mode with temporary mocks)")
}

// setupHTTPHandlersMock initializes HTTP handlers with mock implementations.
// This is for development and testing only.
func (c *Container) setupHTTPHandlersMock() {
	c.Logger.Debug("setting up HTTP handlers with MOCK implementations")

	// Mock auth service and user repo
	mockAuthService := httphandler.NewMockAuthService()
	mockUserRepo := httphandler.NewMockUserRepository()
	c.AuthHandler = httphandler.NewAuthHandler(mockAuthService, mockUserRepo)

	// Mock workspace services
	mockWorkspaceService := httphandler.NewMockWorkspaceService()
	mockMemberService := httphandler.NewMockMemberService()
	c.WorkspaceHandler = httphandler.NewWorkspaceHandler(mockWorkspaceService, mockMemberService)

	// Mock chat service
	mockChatService := httphandler.NewMockChatService()
	c.ChatHandler = httphandler.NewChatHandler(mockChatService)

	// WebSocket handler with real Hub (even in mock mode, we need real WS)
	c.WSHandler = wshandler.NewHandler(
		c.Hub,
		wshandler.WithHandlerLogger(c.Logger),
		wshandler.WithHandlerConfig(wshandler.HandlerConfig{
			ReadBufferSize:  c.Config.WebSocket.ReadBufferSize,
			WriteBufferSize: c.Config.WebSocket.WriteBufferSize,
			Logger:          c.Logger,
		}),
	)

	// Static token validator for development
	c.TokenValidator = middleware.NewStaticTokenValidator(c.Config.Auth.JWTSecret)

	// Mock workspace access checker
	c.AccessChecker = middleware.NewMockWorkspaceAccessChecker()

	c.Logger.Debug("HTTP handlers initialized (mock mode)")
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

// IsReady implements httpserver.HealthChecker.
// It checks if all infrastructure components are healthy.
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

// GetHealthStatus implements httpserver.HealthChecker.
// It returns detailed health status of all components.
func (c *Container) GetHealthStatus(ctx context.Context) []httpserver.ComponentStatus {
	var statuses []httpserver.ComponentStatus

	// MongoDB status
	mongoStatus := httpserver.ComponentStatus{Name: "mongodb", Status: httpserver.StatusHealthy}
	if c.MongoDB == nil {
		mongoStatus.Status = httpserver.StatusUnhealthy
		mongoStatus.Message = "client not initialized"
	} else if err := c.MongoDB.Ping(ctx, nil); err != nil {
		mongoStatus.Status = httpserver.StatusUnhealthy
		mongoStatus.Message = err.Error()
	}
	statuses = append(statuses, mongoStatus)

	// Redis status
	redisStatus := httpserver.ComponentStatus{Name: "redis", Status: httpserver.StatusHealthy}
	if c.Redis == nil {
		redisStatus.Status = httpserver.StatusUnhealthy
		redisStatus.Message = "client not initialized"
	} else if err := c.Redis.Ping(ctx).Err(); err != nil {
		redisStatus.Status = httpserver.StatusUnhealthy
		redisStatus.Message = err.Error()
	}
	statuses = append(statuses, redisStatus)

	// WebSocket Hub status
	hubStatus := httpserver.ComponentStatus{Name: "websocket_hub", Status: httpserver.StatusHealthy}
	if c.Hub == nil {
		hubStatus.Status = httpserver.StatusUnhealthy
		hubStatus.Message = "hub not initialized"
	} else if !c.Hub.IsRunning() {
		hubStatus.Status = httpserver.StatusUnhealthy
		hubStatus.Message = "hub not running"
	}
	statuses = append(statuses, hubStatus)

	// EventBus status
	eventBusStatus := httpserver.ComponentStatus{Name: "eventbus", Status: httpserver.StatusHealthy}
	if c.EventBus == nil {
		eventBusStatus.Status = httpserver.StatusUnhealthy
		eventBusStatus.Message = "event bus not initialized"
	} else if !c.EventBus.IsRunning() {
		eventBusStatus.Status = httpserver.StatusDegraded
		eventBusStatus.Message = "event bus not running"
	}
	statuses = append(statuses, eventBusStatus)

	return statuses
}
