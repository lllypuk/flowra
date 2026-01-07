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
	taskapp "github.com/lllypuk/flowra/internal/application/task"
	wsapp "github.com/lllypuk/flowra/internal/application/workspace"
	"github.com/lllypuk/flowra/internal/config"
	notificationdomain "github.com/lllypuk/flowra/internal/domain/notification"
	taskdomain "github.com/lllypuk/flowra/internal/domain/task"
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
	keycloakTokenBuffer    = 30 * time.Second
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
	TemplateRenderer            *httphandler.TemplateRenderer
	TemplateHandler             *httphandler.TemplateHandler
	NotificationTemplateHandler *httphandler.NotificationTemplateHandler
	ChatTemplateHandler         *httphandler.ChatTemplateHandler
	BoardTemplateHandler        *httphandler.BoardTemplateHandler
	TaskDetailTemplateHandler   *httphandler.TaskDetailTemplateHandler

	// Auth middleware components
	TokenValidator middleware.TokenValidator
	UserResolver   middleware.UserResolver
	AccessChecker  middleware.WorkspaceAccessChecker
	JWTValidator   keycloak.JWTValidator // for cleanup on shutdown

	// OAuth client (for Keycloak integration)
	OAuthClient *keycloak.OAuthClient
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
	c.EventStore = eventstore.NewMongoEventStore(
		c.MongoDB,
		c.MongoDBName,
		eventstore.WithLogger(c.Logger),
	)
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
	c.UserRepo = mongodb.NewMongoUserRepository(
		db.Collection("users"),
		mongodb.WithUserRepoLogger(c.Logger),
	)

	// Workspace repository
	c.WorkspaceRepo = mongodb.NewMongoWorkspaceRepository(
		db.Collection("workspaces"),
		db.Collection("workspace_members"),
		mongodb.WithWorkspaceRepoLogger(c.Logger),
	)

	// Chat repository (event sourced - command side)
	c.ChatRepo = mongodb.NewMongoChatRepository(
		c.EventStore,
		db.Collection("chats_read_model"),
		mongodb.WithChatRepoLogger(c.Logger),
	)

	// Chat read model repository (query side)
	c.ChatQueryRepo = mongodb.NewMongoChatReadModelRepository(
		db.Collection("chats_read_model"),
		c.EventStore,
		mongodb.WithChatReadModelRepoLogger(c.Logger),
	)

	// Message repository
	c.MessageRepo = mongodb.NewMongoMessageRepository(
		db.Collection("messages"),
		mongodb.WithMessageRepoLogger(c.Logger),
	)

	// Task repository (event sourced)
	c.TaskRepo = mongodb.NewMongoTaskRepository(
		c.EventStore,
		db.Collection("tasks_read_model"),
		mongodb.WithTaskRepoLogger(c.Logger),
	)

	// Notification repository
	c.NotificationRepo = mongodb.NewMongoNotificationRepository(
		db.Collection("notifications"),
		mongodb.WithNotificationRepoLogger(c.Logger),
	)

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

	// Inject OAuth client into template handler for login/callback
	if c.TemplateHandler != nil && c.OAuthClient != nil {
		c.TemplateHandler.SetOAuthClient(&oauthClientAdapter{client: c.OAuthClient})
		c.Logger.Debug("OAuth client injected into template handler")
	}

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

	// === 8. Token Validator and User Resolver ===
	c.setupTokenValidator()
	c.setupUserResolver()

	// Configure page auth middleware with token validator and user resolver
	httphandler.SetPageAuthConfig(&httphandler.PageAuthConfig{
		TokenValidator: c.TokenValidator,
		UserResolver:   c.UserResolver,
		Logger:         c.Logger,
	})

	// === 9. Notification Service and Template Handler ===
	c.setupNotificationTemplateHandler()

	// === 10. Chat Template Handler ===
	c.setupChatTemplateHandler()

	// === 11. Board Template Handler ===
	c.setupBoardTemplateHandler()

	// === 12. Task Detail Template Handler ===
	c.setupTaskDetailTemplateHandler()

	// Note: MessageHandler, TaskHandler, NotificationHandler, UserHandler
	// are left nil - routes.go will create placeholder endpoints for them.
	// This is intentional until their use cases are fully implemented.

	c.Logger.Info("HTTP handlers initialized with REAL implementations")
}

// createWorkspaceService creates the workspace service with all dependencies.
func (c *Container) createWorkspaceService() *service.WorkspaceService {
	// Create Keycloak client or NoOp if not configured/enabled
	var keycloakClient wsapp.KeycloakClient
	if c.Config.Keycloak.Enabled && c.Config.Keycloak.URL != "" && c.Config.Keycloak.AdminUsername != "" {
		c.Logger.Debug("using real Keycloak GroupClient for workspace service",
			slog.String("url", c.Config.Keycloak.URL),
			slog.String("realm", c.Config.Keycloak.Realm),
		)

		// Create admin token manager for authentication
		tokenManager := keycloak.NewAdminTokenManager(keycloak.AdminTokenConfig{
			KeycloakURL: c.Config.Keycloak.URL,
			Realm:       "master", // Admin operations are typically against master realm
			ClientID:    "admin-cli",
			Username:    c.Config.Keycloak.AdminUsername,
			Password:    c.Config.Keycloak.AdminPassword,
			TokenBuffer: keycloakTokenBuffer,
		})

		// Create group client for workspace management
		keycloakClient = keycloak.NewGroupClient(keycloak.GroupClientConfig{
			KeycloakURL: c.Config.Keycloak.URL,
			Realm:       c.Config.Keycloak.Realm,
		}, tokenManager)
	} else {
		c.Logger.Debug("using NoOp Keycloak client for workspace service (admin not configured)")
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

	// Create OAuth client (store in container for reuse)
	c.OAuthClient = keycloak.NewOAuthClient(keycloak.OAuthClientConfig{
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
		OAuthClient: c.OAuthClient,
		TokenStore:  tokenStore,
		UserRepo:    c.UserRepo,
		Logger:      c.Logger,
	})
}

// setupNotificationTemplateHandler creates the notification template handler with all dependencies.
func (c *Container) setupNotificationTemplateHandler() {
	// Create notification service that implements NotificationTemplateService
	notifService := c.createNotificationTemplateService()

	// Create template handler
	c.NotificationTemplateHandler = httphandler.NewNotificationTemplateHandler(
		c.TemplateRenderer,
		c.Logger,
		notifService,
	)

	c.Logger.Debug("notification template handler initialized")
}

// setupChatTemplateHandler creates the chat template handler with all dependencies.
func (c *Container) setupChatTemplateHandler() {
	// Create chat template service adapter
	chatService := c.createChatTemplateService()

	// For now, message service is nil - will be implemented with message use cases
	c.ChatTemplateHandler = httphandler.NewChatTemplateHandler(
		c.TemplateRenderer,
		c.Logger,
		chatService,
		nil, // MessageTemplateService - TODO: implement when message use cases are ready
	)

	c.Logger.Debug("chat template handler initialized")
}

// createChatTemplateService creates a service implementing ChatTemplateService.
func (c *Container) createChatTemplateService() httphandler.ChatTemplateService {
	return &chatTemplateServiceAdapter{
		chatService: c.ChatService,
	}
}

// chatTemplateServiceAdapter adapts ChatService to ChatTemplateService.
type chatTemplateServiceAdapter struct {
	chatService *service.ChatService
}

// CreateChat implements ChatTemplateService.
func (a *chatTemplateServiceAdapter) CreateChat(
	ctx context.Context,
	cmd chatapp.CreateChatCommand,
) (chatapp.Result, error) {
	if a.chatService == nil {
		return chatapp.Result{}, chatapp.ErrChatNotFound
	}
	return a.chatService.CreateChat(ctx, cmd)
}

// GetChat implements ChatTemplateService.
func (a *chatTemplateServiceAdapter) GetChat(
	ctx context.Context,
	query chatapp.GetChatQuery,
) (*chatapp.GetChatResult, error) {
	if a.chatService == nil {
		return nil, chatapp.ErrChatNotFound
	}
	return a.chatService.GetChat(ctx, query)
}

// ListChats implements ChatTemplateService.
func (a *chatTemplateServiceAdapter) ListChats(
	ctx context.Context,
	query chatapp.ListChatsQuery,
) (*chatapp.ListChatsResult, error) {
	if a.chatService == nil {
		return &chatapp.ListChatsResult{}, nil
	}
	return a.chatService.ListChats(ctx, query)
}

// setupBoardTemplateHandler creates the board template handler with all dependencies.
func (c *Container) setupBoardTemplateHandler() {
	// Create board task service adapter
	taskService := c.createBoardTaskService()

	// Create board member service adapter
	memberService := c.createBoardMemberService()

	c.BoardTemplateHandler = httphandler.NewBoardTemplateHandler(
		c.TemplateRenderer,
		c.Logger,
		taskService,
		memberService,
	)

	c.Logger.Debug("board template handler initialized")
}

// createBoardTaskService creates a service implementing BoardTaskService.
func (c *Container) createBoardTaskService() httphandler.BoardTaskService {
	return &boardTaskServiceAdapter{
		collection: c.MongoDB.Database(c.MongoDBName).Collection("tasks_read_model"),
	}
}

// boardTaskServiceAdapter adapts MongoDB collection to BoardTaskService.
type boardTaskServiceAdapter struct {
	collection *mongo.Collection
}

// ListTasks implements BoardTaskService.
func (a *boardTaskServiceAdapter) ListTasks(
	ctx context.Context,
	filters taskapp.Filters,
) ([]*taskapp.ReadModel, error) {
	if a.collection == nil {
		return []*taskapp.ReadModel{}, nil
	}
	return a.queryTasks(ctx, filters)
}

// CountTasks implements BoardTaskService.
func (a *boardTaskServiceAdapter) CountTasks(
	ctx context.Context,
	filters taskapp.Filters,
) (int, error) {
	if a.collection == nil {
		return 0, nil
	}
	filter := a.buildFilter(filters)
	count, err := a.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// GetTask implements BoardTaskService.
func (a *boardTaskServiceAdapter) GetTask(ctx context.Context, taskID uuid.UUID) (*taskapp.ReadModel, error) {
	if a.collection == nil {
		return nil, taskapp.ErrTaskNotFound
	}
	filter := map[string]any{"_id": taskID.String()}
	var result taskReadModelDoc
	if err := a.collection.FindOne(ctx, filter).Decode(&result); err != nil {
		return nil, taskapp.ErrTaskNotFound
	}
	return result.toReadModel(), nil
}

// queryTasks queries tasks with filters.
func (a *boardTaskServiceAdapter) queryTasks(
	ctx context.Context,
	filters taskapp.Filters,
) ([]*taskapp.ReadModel, error) {
	filter := a.buildFilter(filters)

	opts := options.Find()
	if filters.Limit > 0 {
		opts.SetLimit(int64(filters.Limit))
	}
	if filters.Offset > 0 {
		opts.SetSkip(int64(filters.Offset))
	}
	opts.SetSort(map[string]int{"created_at": -1})

	cursor, err := a.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []*taskapp.ReadModel
	for cursor.Next(ctx) {
		var doc taskReadModelDoc
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			continue
		}
		results = append(results, doc.toReadModel())
	}

	return results, nil
}

// buildFilter builds a MongoDB filter from task filters.
func (a *boardTaskServiceAdapter) buildFilter(filters taskapp.Filters) map[string]any {
	filter := make(map[string]any)

	if filters.Status != nil {
		filter["status"] = string(*filters.Status)
	}
	if filters.Priority != nil {
		filter["priority"] = string(*filters.Priority)
	}
	if filters.EntityType != nil {
		filter["entity_type"] = string(*filters.EntityType)
	}
	if filters.AssigneeID != nil {
		filter["assigned_to"] = filters.AssigneeID.String()
	}
	if filters.ChatID != nil {
		filter["chat_id"] = filters.ChatID.String()
	}

	return filter
}

// taskReadModelDoc represents a task document in MongoDB.
type taskReadModelDoc struct {
	ID         string     `bson:"_id"`
	ChatID     string     `bson:"chat_id"`
	Title      string     `bson:"title"`
	EntityType string     `bson:"entity_type"`
	Status     string     `bson:"status"`
	Priority   string     `bson:"priority"`
	AssignedTo *string    `bson:"assigned_to,omitempty"`
	DueDate    *time.Time `bson:"due_date,omitempty"`
	CreatedBy  string     `bson:"created_by"`
	CreatedAt  time.Time  `bson:"created_at"`
	Version    int        `bson:"version"`
}

// toReadModel converts the document to a ReadModel.
func (d *taskReadModelDoc) toReadModel() *taskapp.ReadModel {
	id, _ := uuid.ParseUUID(d.ID)
	chatID, _ := uuid.ParseUUID(d.ChatID)
	createdBy, _ := uuid.ParseUUID(d.CreatedBy)

	model := &taskapp.ReadModel{
		ID:         id,
		ChatID:     chatID,
		Title:      d.Title,
		EntityType: taskdomain.EntityType(d.EntityType),
		Status:     taskdomain.Status(d.Status),
		Priority:   taskdomain.Priority(d.Priority),
		DueDate:    d.DueDate,
		CreatedBy:  createdBy,
		CreatedAt:  d.CreatedAt,
		Version:    d.Version,
	}

	if d.AssignedTo != nil {
		assignedTo, _ := uuid.ParseUUID(*d.AssignedTo)
		model.AssignedTo = &assignedTo
	}

	return model
}

// createBoardMemberService creates a service implementing BoardMemberService.
func (c *Container) createBoardMemberService() httphandler.BoardMemberService {
	return &boardMemberServiceAdapter{
		memberService: c.MemberService,
	}
}

// boardMemberServiceAdapter adapts MemberService to BoardMemberService.
type boardMemberServiceAdapter struct {
	memberService *service.MemberService
}

// ListWorkspaceMembers implements BoardMemberService.
func (a *boardMemberServiceAdapter) ListWorkspaceMembers(
	ctx context.Context,
	workspaceID uuid.UUID,
	offset, limit int,
) ([]httphandler.MemberViewData, error) {
	if a.memberService == nil {
		return []httphandler.MemberViewData{}, nil
	}
	members, _, err := a.memberService.ListMembers(ctx, workspaceID, offset, limit)
	if err != nil {
		return nil, err
	}

	result := make([]httphandler.MemberViewData, 0, len(members))
	for _, m := range members {
		result = append(result, httphandler.MemberViewData{
			UserID:   m.UserID().String(),
			Username: "user" + m.UserID().String()[:8], // TODO: get actual username
			Role:     m.Role().String(),
			JoinedAt: m.JoinedAt(),
		})
	}
	return result, nil
}

// setupTaskDetailTemplateHandler creates the task detail template handler with all dependencies.
func (c *Container) setupTaskDetailTemplateHandler() {
	// Create task detail service adapter
	taskService := c.createTaskDetailService()

	// For now, event service is nil - will be implemented for activity timeline
	c.TaskDetailTemplateHandler = httphandler.NewTaskDetailTemplateHandler(
		c.TemplateRenderer,
		c.Logger,
		taskService,
		nil, // TaskEventService - TODO: implement for activity timeline
		c.createBoardMemberService(),
	)

	c.Logger.Debug("task detail template handler initialized")
}

// createTaskDetailService creates a service implementing TaskDetailService.
// Reuses the boardTaskServiceAdapter since both interfaces require the same GetTask method.
func (c *Container) createTaskDetailService() httphandler.TaskDetailService {
	return c.createBoardTaskService()
}

// createNotificationTemplateService creates a service implementing NotificationTemplateService.
func (c *Container) createNotificationTemplateService() httphandler.NotificationTemplateService {
	// Create use cases
	listUC := notification.NewListNotificationsUseCase(c.NotificationRepo)
	countUC := notification.NewCountUnreadUseCase(c.NotificationRepo)
	markAsReadUC := notification.NewMarkAsReadUseCase(c.NotificationRepo)
	getUC := notification.NewGetNotificationUseCase(c.NotificationRepo)

	return &notificationTemplateService{
		listUC:       listUC,
		countUC:      countUC,
		markAsReadUC: markAsReadUC,
		getUC:        getUC,
	}
}

// notificationTemplateService implements httphandler.NotificationTemplateService.
type notificationTemplateService struct {
	listUC       *notification.ListNotificationsUseCase
	countUC      *notification.CountUnreadUseCase
	markAsReadUC *notification.MarkAsReadUseCase
	getUC        *notification.GetNotificationUseCase
}

// ListNotifications lists notifications for a user.
func (s *notificationTemplateService) ListNotifications(
	ctx context.Context,
	query notification.ListNotificationsQuery,
) (notification.ListResult, error) {
	return s.listUC.Execute(ctx, query)
}

// CountUnread counts unread notifications for a user.
func (s *notificationTemplateService) CountUnread(
	ctx context.Context,
	query notification.CountUnreadQuery,
) (notification.CountResult, error) {
	return s.countUC.Execute(ctx, query)
}

// MarkAsRead marks a notification as read.
func (s *notificationTemplateService) MarkAsRead(
	ctx context.Context,
	cmd notification.MarkAsReadCommand,
) (notification.Result, error) {
	return s.markAsReadUC.Execute(ctx, cmd)
}

// GetNotification gets a notification by ID.
func (s *notificationTemplateService) GetNotification(
	ctx context.Context,
	notificationID uuid.UUID,
	userID uuid.UUID,
) (*notificationdomain.Notification, error) {
	query := notification.GetNotificationQuery{
		NotificationID: notificationID,
		UserID:         userID,
	}
	result, err := s.getUC.Execute(ctx, query)
	if err != nil {
		return nil, err
	}
	return result.Value, nil
}

// createUserRepoAdapter creates an adapter for UserRepository that works with echo.Context.
// This bridges the gap between service layer (uses context.Context) and handler layer (uses echo.Context).
func (c *Container) createUserRepoAdapter() httphandler.UserRepository {
	return &userRepoAdapter{repo: c.UserRepo}
}

// setupTokenValidator configures the JWT token validator.
// Uses KeycloakValidatorAdapter when Keycloak is enabled, otherwise falls back to static validator.
func (c *Container) setupTokenValidator() {
	if c.Config.Keycloak.Enabled && c.Config.Keycloak.URL != "" {
		// Create Keycloak JWT validator
		// JWTAudience is separate from ClientID: empty = skip audience validation
		jwtValidator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
			KeycloakURL:     c.Config.Keycloak.URL,
			Realm:           c.Config.Keycloak.Realm,
			ClientID:        c.Config.Keycloak.JWTAudience, // Use JWTAudience for validation, not OAuth ClientID
			Leeway:          c.Config.Keycloak.JWT.Leeway,
			RefreshInterval: c.Config.Keycloak.JWT.RefreshInterval,
			Logger:          c.Logger,
		})
		if err != nil {
			c.Logger.Warn("failed to create Keycloak JWT validator, falling back to static validator",
				slog.String("error", err.Error()),
			)
			c.TokenValidator = middleware.NewStaticTokenValidator(c.Config.Auth.JWTSecret)
			return
		}

		// Store for cleanup
		c.JWTValidator = jwtValidator

		// Wrap with adapter
		c.TokenValidator = middleware.NewKeycloakValidatorAdapter(jwtValidator)

		c.Logger.Info("token validator initialized with Keycloak",
			slog.String("url", c.Config.Keycloak.URL),
			slog.String("realm", c.Config.Keycloak.Realm),
		)
	} else {
		c.Logger.Warn("Keycloak not enabled, using static token validator (development mode)")
		c.TokenValidator = middleware.NewStaticTokenValidator(c.Config.Auth.JWTSecret)
	}
}

// setupUserResolver configures the user resolver for mapping external IDs to internal IDs.
func (c *Container) setupUserResolver() {
	c.UserResolver = &userResolver{
		userRepo: c.UserRepo,
		logger:   c.Logger,
	}
	c.Logger.Debug("user resolver initialized")
}

// userResolver implements middleware.UserResolver.
type userResolver struct {
	userRepo *mongodb.MongoUserRepository
	logger   *slog.Logger
}

// ResolveUser finds or creates a user by external ID and returns their internal ID.
func (r *userResolver) ResolveUser(ctx context.Context, externalID, username, email string) (uuid.UUID, error) {
	// Try to find existing user by external ID
	existingUser, err := r.userRepo.FindByExternalID(ctx, externalID)
	if err == nil {
		return existingUser.ID(), nil
	}

	// User not found - create new user
	r.logger.InfoContext(ctx, "creating new user from Keycloak",
		slog.String("external_id", externalID),
		slog.String("username", username),
		slog.String("email", email),
	)

	newUser, createErr := user.NewUser(externalID, username, email, username)
	if createErr != nil {
		r.logger.ErrorContext(ctx, "failed to create user",
			slog.String("external_id", externalID),
			slog.String("error", createErr.Error()),
		)
		return uuid.UUID(""), createErr
	}

	if saveErr := r.userRepo.Save(ctx, newUser); saveErr != nil {
		r.logger.ErrorContext(ctx, "failed to save user",
			slog.String("external_id", externalID),
			slog.String("error", saveErr.Error()),
		)
		return uuid.UUID(""), saveErr
	}

	r.logger.InfoContext(ctx, "user created successfully",
		slog.String("user_id", newUser.ID().String()),
		slog.String("external_id", externalID),
	)

	return newUser.ID(), nil
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

	// Close JWT Validator (stops JWKS refresh goroutine)
	if c.JWTValidator != nil {
		if err := c.JWTValidator.Close(); err != nil {
			errs = append(errs, fmt.Errorf("jwt validator close: %w", err))
		} else {
			c.Logger.Debug("jwt validator closed")
		}
	}

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

// oauthClientAdapter adapts keycloak.OAuthClient to httphandler.OAuthClient interface.
type oauthClientAdapter struct {
	client *keycloak.OAuthClient
}

// AuthorizationURL implements httphandler.OAuthClient.
func (a *oauthClientAdapter) AuthorizationURL(redirectURI, state string) string {
	return a.client.AuthorizationURL(redirectURI, state)
}

// ExchangeCode implements httphandler.OAuthClient.
func (a *oauthClientAdapter) ExchangeCode(
	ctx context.Context,
	code, redirectURI string,
) (*httphandler.OAuthTokenResponse, error) {
	resp, err := a.client.ExchangeCode(ctx, code, redirectURI)
	if err != nil {
		return nil, err
	}
	return &httphandler.OAuthTokenResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresIn:    resp.ExpiresIn,
	}, nil
}
