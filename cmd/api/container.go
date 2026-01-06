// Package main provides the API server entry point.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	"github.com/lllypuk/flowra/internal/application/notification"
	wsapp "github.com/lllypuk/flowra/internal/application/workspace"
	"github.com/lllypuk/flowra/internal/config"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	wshandler "github.com/lllypuk/flowra/internal/handler/websocket"
	"github.com/lllypuk/flowra/internal/infrastructure/auth"
	"github.com/lllypuk/flowra/internal/infrastructure/eventbus"
	"github.com/lllypuk/flowra/internal/infrastructure/eventstore"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
	"github.com/lllypuk/flowra/internal/infrastructure/keycloak"
	"github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	"github.com/lllypuk/flowra/internal/infrastructure/websocket"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/lllypuk/flowra/internal/service"
	"github.com/lllypuk/flowra/web"

	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
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
	ChatQueryRepo    *mongodb.MongoChatReadModelRepository
	MessageRepo      *mongodb.MongoMessageRepository
	TaskRepo         *mongodb.MongoTaskRepository
	NotificationRepo *mongodb.MongoNotificationRepository

	// Use Cases
	CreateNotificationUC *notification.CreateNotificationUseCase

	// Services (for external access if needed)
	WorkspaceService *service.WorkspaceService
	MemberService    *service.MemberService
	ChatService      *service.ChatService

	// HTTP Handlers
	AuthHandler         *httphandler.AuthHandler
	WorkspaceHandler    *httphandler.WorkspaceHandler
	ChatHandler         *httphandler.ChatHandler
	MessageHandler      *httphandler.MessageHandler
	TaskHandler         *httphandler.TaskHandler
	NotificationHandler *httphandler.NotificationHandler
	UserHandler         *httphandler.UserHandler
	WSHandler           *wshandler.Handler

	// Template Rendering
	TemplateRenderer *httphandler.TemplateRenderer
	TemplateHandler  *httphandler.TemplateHandler

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

	// Setup template rendering
	if err := c.setupTemplateRenderer(); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("failed to setup template renderer: %w", err)
	}

	// Setup HTTP handlers based on wiring mode
	c.setupHTTPHandlers()

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

	// Validate infrastructure components
	errs = c.validateInfrastructure(errs)

	// Validate auth components
	errs = c.validateAuthComponents(errs)

	// Validate handlers in real mode
	errs = c.validateHandlers(errs)

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// validateInfrastructure checks that all infrastructure components are initialized.
func (c *Container) validateInfrastructure(errs []error) []error {
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
	return errs
}

// validateAuthComponents checks that auth components are initialized.
func (c *Container) validateAuthComponents(errs []error) []error {
	if c.TokenValidator == nil {
		errs = append(errs, errors.New("token validator not initialized"))
	}
	if c.AccessChecker == nil {
		errs = append(errs, errors.New("access checker not initialized"))
	}
	return errs
}

// validateHandlers checks handler initialization in real mode.
func (c *Container) validateHandlers(errs []error) []error {
	if !c.Config.App.IsRealMode() {
		return errs
	}

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

	// Check for mock access checker in production
	if c.Config.IsProduction() && c.AccessChecker != nil {
		if _, isMock := c.AccessChecker.(*middleware.MockWorkspaceAccessChecker); isMock {
			errs = append(errs, errors.New("mock access checker is not allowed in production"))
		}
	}

	return errs
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

	// Chat repository (event sourced - command side)
	c.ChatRepo = mongodb.NewMongoChatRepository(
		c.EventStore,
		db.Collection("chats_read_model"),
	)

	// Chat read model repository (query side)
	c.ChatQueryRepo = mongodb.NewMongoChatReadModelRepository(
		db.Collection("chats_read_model"),
		c.EventStore,
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

// setupTemplateRenderer initializes the template renderer and handler.
func (c *Container) setupTemplateRenderer() error {
	renderer, err := httphandler.NewTemplateRenderer(httphandler.TemplateRendererConfig{
		FS:      web.TemplatesFS,
		Logger:  c.Logger,
		DevMode: c.Config.IsDevelopment(),
	})
	if err != nil {
		return fmt.Errorf("failed to create template renderer: %w", err)
	}

	c.TemplateRenderer = renderer
	// Create template handler - workspace and member services will be set later during setupHTTPHandlers
	c.TemplateHandler = httphandler.NewTemplateHandler(renderer, c.Logger, nil, nil)

	c.Logger.Debug("template renderer initialized",
		slog.Bool("dev_mode", c.Config.IsDevelopment()),
	)

	return nil
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

// setupHTTPHandlers initializes HTTP handlers with real implementations.
// This wires handlers to actual use cases and services.
func (c *Container) setupHTTPHandlers() {
	c.Logger.Debug("setting up HTTP handlers with REAL implementations")

	// === 1. Access Checker (Real) ===
	c.AccessChecker = service.NewRealWorkspaceAccessChecker(c.WorkspaceRepo)
	c.Logger.Debug("access checker initialized (real)")

	// === 2. Member Service (Real) ===
	c.MemberService = service.NewMemberService(c.WorkspaceRepo, c.WorkspaceRepo)
	c.Logger.Debug("member service initialized (real)")

	// === 3. Workspace Service (Real) ===
	c.WorkspaceService = c.createWorkspaceService()
	c.Logger.Debug("workspace service initialized (real)")

	// === 4. Workspace Handler with Real Services ===
	c.WorkspaceHandler = httphandler.NewWorkspaceHandler(c.WorkspaceService, c.MemberService)

	// Inject services into template handler
	if c.TemplateHandler != nil {
		c.TemplateHandler.SetServices(c.WorkspaceService, c.MemberService)
	}

	// === 5. Chat Service (Real) ===
	c.ChatService = c.createChatService()
	c.ChatHandler = httphandler.NewChatHandler(c.ChatService)
	c.Logger.Debug("chat service initialized (real)")

	// === 6. Auth Service ===
	authService := c.createAuthService()
	c.AuthHandler = httphandler.NewAuthHandler(authService, c.createUserRepoAdapter())

	// === 7. WebSocket Handler (unchanged) ===
	c.WSHandler = wshandler.NewHandler(
		c.Hub,
		wshandler.WithHandlerLogger(c.Logger),
		wshandler.WithHandlerConfig(wshandler.HandlerConfig{
			ReadBufferSize:  c.Config.WebSocket.ReadBufferSize,
			WriteBufferSize: c.Config.WebSocket.WriteBufferSize,
			Logger:          c.Logger,
		}),
	)

	// === 8. Token Validator (unchanged) ===
	c.TokenValidator = middleware.NewStaticTokenValidator(c.Config.Auth.JWTSecret)

	// Note: MessageHandler, TaskHandler, NotificationHandler, UserHandler
	// are left nil - routes.go will create placeholder endpoints for them.
	// This is intentional until their use cases are fully implemented.

	c.Logger.Info("HTTP handlers initialized with REAL implementations")
}

// createWorkspaceService creates the workspace service with all dependencies.
func (c *Container) createWorkspaceService() *service.WorkspaceService {
	// Create Keycloak client or NoOp if not configured
	var keycloakClient wsapp.KeycloakClient
	if c.Config.Keycloak.URL != "" && c.Config.Keycloak.ClientID != "" {
		c.Logger.Debug("using real Keycloak client for workspace service")
		// Note: Real Keycloak admin client for group management would go here
		// For now, we use NoOp as the admin API is not yet implemented
		keycloakClient = service.NewNoOpKeycloakClient()
	} else {
		c.Logger.Debug("using NoOp Keycloak client for workspace service")
		keycloakClient = service.NewNoOpKeycloakClient()
	}

	// Create use cases
	createUC := wsapp.NewCreateWorkspaceUseCase(c.WorkspaceRepo, keycloakClient)
	getUC := wsapp.NewGetWorkspaceUseCase(c.WorkspaceRepo)
	updateUC := wsapp.NewUpdateWorkspaceUseCase(c.WorkspaceRepo)

	return service.NewWorkspaceService(service.WorkspaceServiceConfig{
		CreateUC:    createUC,
		GetUC:       getUC,
		UpdateUC:    updateUC,
		CommandRepo: c.WorkspaceRepo,
		QueryRepo:   c.WorkspaceRepo,
	})
}

// createChatService creates the chat service with all dependencies.
func (c *Container) createChatService() *service.ChatService {
	// Create use cases
	createUC := chatapp.NewCreateChatUseCase(c.EventStore)
	getUC := chatapp.NewGetChatUseCase(c.EventStore)
	listUC := chatapp.NewListChatsUseCase(c.ChatQueryRepo, c.EventStore)
	renameUC := chatapp.NewRenameChatUseCase(c.EventStore)
	addPartUC := chatapp.NewAddParticipantUseCase(c.EventStore)
	removePartUC := chatapp.NewRemoveParticipantUseCase(c.EventStore)

	return service.NewChatService(service.ChatServiceConfig{
		CreateUC:     createUC,
		GetUC:        getUC,
		ListUC:       listUC,
		RenameUC:     renameUC,
		AddPartUC:    addPartUC,
		RemovePartUC: removePartUC,
		EventStore:   c.EventStore,
	})
}

// createAuthService creates the auth service.
// Uses mock if Keycloak is not configured, real otherwise.
func (c *Container) createAuthService() httphandler.AuthService {
	// Check if Keycloak is configured
	if c.Config.Keycloak.URL == "" || c.Config.Keycloak.ClientID == "" {
		c.Logger.Warn("Keycloak not configured, using mock auth service")
		return httphandler.NewMockAuthService()
	}

	// Create real auth service with Keycloak
	oauthClient := keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
		KeycloakURL:  c.Config.Keycloak.URL,
		Realm:        c.Config.Keycloak.Realm,
		ClientID:     c.Config.Keycloak.ClientID,
		ClientSecret: c.Config.Keycloak.ClientSecret,
		Logger:       c.Logger,
	})

	tokenStore := auth.NewTokenStore(auth.TokenStoreConfig{
		Client: c.Redis,
	})

	c.Logger.Debug("auth service initialized with Keycloak",
		slog.String("url", c.Config.Keycloak.URL),
		slog.String("realm", c.Config.Keycloak.Realm),
	)

	return service.NewAuthService(service.AuthServiceConfig{
		OAuthClient: oauthClient,
		TokenStore:  tokenStore,
		UserRepo:    c.UserRepo,
		Logger:      c.Logger,
	})
}

// createUserRepoAdapter creates an adapter for UserRepository that works with echo.Context.
// This bridges the gap between service layer (uses context.Context) and handler layer (uses echo.Context).
func (c *Container) createUserRepoAdapter() httphandler.UserRepository {
	return &userRepoAdapter{repo: c.UserRepo}
}

// userRepoAdapter adapts MongoUserRepository to httphandler.UserRepository.
type userRepoAdapter struct {
	repo *mongodb.MongoUserRepository
}

// FindByID implements httphandler.UserRepository.
func (a *userRepoAdapter) FindByID(ctx echo.Context, id uuid.UUID) (*user.User, error) {
	return a.repo.FindByID(ctx.Request().Context(), id)
}

// FindByExternalID implements httphandler.UserRepository.
func (a *userRepoAdapter) FindByExternalID(ctx echo.Context, externalID string) (*user.User, error) {
	return a.repo.FindByExternalID(ctx.Request().Context(), externalID)
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
