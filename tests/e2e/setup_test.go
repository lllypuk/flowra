//go:build e2e

// Package e2e provides end-to-end tests for the Flowra application.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/lllypuk/flowra/internal/config"
	"github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/internal/infrastructure/eventbus"
	"github.com/lllypuk/flowra/internal/infrastructure/eventstore"
	"github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	wsinfra "github.com/lllypuk/flowra/internal/infrastructure/websocket"
	"github.com/lllypuk/flowra/internal/middleware"
	"github.com/lllypuk/flowra/tests/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Test timeouts and constants
const (
	serverStartupTimeout = 5 * time.Second
	requestTimeout       = 10 * time.Second
	wsConnectTimeout     = 5 * time.Second
	wsReadTimeout        = 10 * time.Second
	eventPropagationWait = 500 * time.Millisecond
)

// E2ETestSuite represents the E2E test environment with all infrastructure.
type E2ETestSuite struct {
	t *testing.T

	// Infrastructure
	MongoClient *mongo.Client
	MongoDB     *mongo.Database
	Redis       *redis.Client

	// Application components
	Echo           *echo.Echo
	EventStore     *eventstore.MongoEventStore
	EventBus       *eventbus.RedisEventBus
	WSHub          *wsinfra.Hub
	TokenValidator *E2ETokenValidator

	// Repositories
	UserRepo         *mongodb.MongoUserRepository
	WorkspaceRepo    *mongodb.MongoWorkspaceRepository
	ChatRepo         *mongodb.MongoChatRepository
	MessageRepo      *mongodb.MongoMessageRepository
	TaskRepo         *mongodb.MongoTaskRepository
	NotificationRepo *mongodb.MongoNotificationRepository

	// HTTP Handlers with mocks for easier testing
	MockAuthService      *httphandler.MockAuthService
	MockUserRepo         *httphandler.MockUserRepository
	MockWorkspaceService *httphandler.MockWorkspaceService
	MockMemberService    *httphandler.MockMemberService
	MockChatService      *httphandler.MockChatService
	MockMessageService   *httphandler.MockMessageService
	MockTaskService      *httphandler.MockTaskService

	// Server state
	serverAddr   string
	serverCancel context.CancelFunc
	serverWg     sync.WaitGroup

	// Test state
	users   map[string]*TestUser
	usersMu sync.RWMutex
}

// TestUser represents a test user with credentials.
type TestUser struct {
	ID       uuid.UUID
	Username string
	Email    string
	Token    string
	User     *user.User
}

// E2ETokenValidator is a simple token validator for E2E tests.
type E2ETokenValidator struct {
	tokens map[string]*middleware.TokenClaims
	mu     sync.RWMutex
}

// NewE2ETokenValidator creates a new E2E token validator.
func NewE2ETokenValidator() *E2ETokenValidator {
	return &E2ETokenValidator{
		tokens: make(map[string]*middleware.TokenClaims),
	}
}

// RegisterToken registers a token with claims.
func (v *E2ETokenValidator) RegisterToken(token string, claims *middleware.TokenClaims) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.tokens[token] = claims
}

// ValidateToken validates a token and returns claims.
func (v *E2ETokenValidator) ValidateToken(_ context.Context, token string) (*middleware.TokenClaims, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	claims, ok := v.tokens[token]
	if !ok {
		return nil, middleware.ErrInvalidToken
	}
	if claims.ExpiresAt.Before(time.Now()) {
		return claims, middleware.ErrTokenExpired
	}
	return claims, nil
}

// NewE2ETestSuite creates a new E2E test suite.
func NewE2ETestSuite(t *testing.T) *E2ETestSuite {
	t.Helper()

	// Setup MongoDB using shared container
	client, db := testutil.SetupSharedTestMongoDBWithClient(t)

	// Setup Redis using shared container
	redisClient := testutil.SetupTestRedis(t)

	suite := &E2ETestSuite{
		t:              t,
		MongoClient:    client,
		MongoDB:        db,
		Redis:          redisClient,
		TokenValidator: NewE2ETokenValidator(),
		users:          make(map[string]*TestUser),
	}

	// Initialize infrastructure
	suite.setupInfrastructure()

	// Initialize repositories
	suite.setupRepositories()

	// Initialize mock services
	suite.setupMockServices()

	// Setup and start HTTP server
	suite.setupServer()

	return suite
}

// setupInfrastructure initializes infrastructure components.
func (s *E2ETestSuite) setupInfrastructure() {
	// Create event store
	s.EventStore = eventstore.NewMongoEventStore(s.MongoClient, s.MongoDB.Name())

	// Create event bus
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	s.EventBus = eventbus.NewRedisEventBus(
		s.Redis,
		eventbus.WithLogger(logger),
		eventbus.WithChannelPrefix("e2e_test:"),
	)

	// Create WebSocket hub
	s.WSHub = wsinfra.NewHub(
		wsinfra.WithHubLogger(logger),
	)
}

// setupRepositories initializes all repositories.
func (s *E2ETestSuite) setupRepositories() {
	s.UserRepo = mongodb.NewMongoUserRepository(s.MongoDB.Collection("users"))
	s.WorkspaceRepo = mongodb.NewMongoWorkspaceRepository(
		s.MongoDB.Collection("workspaces"),
		s.MongoDB.Collection("workspace_members"),
	)
	s.ChatRepo = mongodb.NewMongoChatRepository(
		s.EventStore,
		s.MongoDB.Collection("chats_read_model"),
	)
	s.MessageRepo = mongodb.NewMongoMessageRepository(s.MongoDB.Collection("messages"))
	s.TaskRepo = mongodb.NewMongoTaskRepository(
		s.EventStore,
		s.MongoDB.Collection("tasks_read_model"),
	)
	s.NotificationRepo = mongodb.NewMongoNotificationRepository(s.MongoDB.Collection("notifications"))
}

// setupMockServices initializes mock services for HTTP handlers.
func (s *E2ETestSuite) setupMockServices() {
	s.MockAuthService = httphandler.NewMockAuthService()
	s.MockUserRepo = httphandler.NewMockUserRepository()
	s.MockWorkspaceService = httphandler.NewMockWorkspaceService()
	s.MockMemberService = httphandler.NewMockMemberService()
	s.MockChatService = httphandler.NewMockChatService()
	s.MockMessageService = httphandler.NewMockMessageService()
	s.MockTaskService = httphandler.NewMockTaskService()
}

// setupServer creates and starts the HTTP server.
func (s *E2ETestSuite) setupServer() {
	s.Echo = echo.New()
	s.Echo.HideBanner = true
	s.Echo.HidePort = true

	// Create logger that discards output
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Setup middleware
	authMiddleware := middleware.Auth(middleware.AuthConfig{
		Logger:         logger,
		TokenValidator: s.TokenValidator,
		SkipPaths: []string{
			"/health",
			"/ready",
			"/api/v1/auth/login",
		},
		AllowExpiredForPaths: []string{
			"/api/v1/auth/refresh",
		},
	})

	// Add middleware
	s.Echo.Use(authMiddleware)

	// Register health endpoints
	s.Echo.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
	})
	s.Echo.GET("/ready", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
	})

	// Create handlers
	authHandler := httphandler.NewAuthHandler(s.MockAuthService, s.MockUserRepo)
	workspaceHandler := httphandler.NewWorkspaceHandler(s.MockWorkspaceService, s.MockMemberService)
	chatHandler := httphandler.NewChatHandler(s.MockChatService)
	messageHandler := httphandler.NewMessageHandler(s.MockMessageService)
	taskHandler := httphandler.NewTaskHandler(s.MockTaskService)

	// Register routes
	api := s.Echo.Group("/api/v1")

	// Auth routes (public)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/logout", authHandler.Logout)
	api.POST("/auth/refresh", authHandler.Refresh)
	api.GET("/auth/me", authHandler.Me)

	// Workspace routes
	api.POST("/workspaces", workspaceHandler.Create)
	api.GET("/workspaces", workspaceHandler.List)
	api.GET("/workspaces/:id", workspaceHandler.Get)
	api.PUT("/workspaces/:id", workspaceHandler.Update)
	api.DELETE("/workspaces/:id", workspaceHandler.Delete)
	api.POST("/workspaces/:id/members", workspaceHandler.AddMember)
	api.DELETE("/workspaces/:id/members/:user_id", workspaceHandler.RemoveMember)
	api.PUT("/workspaces/:id/members/:user_id/role", workspaceHandler.UpdateMemberRole)

	// Chat routes - chat handler uses c.Param("workspace_id") and c.Param("id") for chat ID
	api.POST("/workspaces/:workspace_id/chats", chatHandler.Create)
	api.GET("/workspaces/:workspace_id/chats", chatHandler.List)
	api.GET("/workspaces/:workspace_id/chats/:id", chatHandler.Get)
	api.PUT("/workspaces/:workspace_id/chats/:id", chatHandler.Update)
	api.DELETE("/workspaces/:workspace_id/chats/:id", chatHandler.Delete)
	api.POST("/workspaces/:workspace_id/chats/:id/participants", chatHandler.AddParticipant)
	api.DELETE("/workspaces/:workspace_id/chats/:id/participants/:user_id", chatHandler.RemoveParticipant)

	// Message routes
	api.POST("/workspaces/:workspace_id/chats/:chat_id/messages", messageHandler.Send)
	api.GET("/workspaces/:workspace_id/chats/:chat_id/messages", messageHandler.List)
	api.PUT("/workspaces/:workspace_id/chats/:chat_id/messages/:id", messageHandler.Edit)
	api.DELETE("/workspaces/:workspace_id/chats/:chat_id/messages/:id", messageHandler.Delete)

	// Task routes - note: task handler uses c.Param("task_id") for task ID
	api.POST("/workspaces/:workspace_id/tasks", taskHandler.Create)
	api.GET("/workspaces/:workspace_id/tasks", taskHandler.List)
	api.GET("/workspaces/:workspace_id/tasks/:task_id", taskHandler.Get)
	api.PUT("/workspaces/:workspace_id/tasks/:task_id/status", taskHandler.ChangeStatus)
	api.PUT("/workspaces/:workspace_id/tasks/:task_id/assignee", taskHandler.Assign)
	api.PUT("/workspaces/:workspace_id/tasks/:task_id/priority", taskHandler.ChangePriority)
	api.PUT("/workspaces/:workspace_id/tasks/:task_id/due-date", taskHandler.SetDueDate)
	api.DELETE("/workspaces/:workspace_id/tasks/:task_id", taskHandler.Delete)

	// WebSocket route (simplified for testing)
	api.GET("/ws", func(c echo.Context) error {
		userID := middleware.GetUserID(c)
		if userID.IsZero() {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}
		// For now just acknowledge - full WS testing is in websocket_test.go
		return c.JSON(http.StatusOK, map[string]string{"status": "ws_available"})
	})

	// Find available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(s.t, err)
	s.serverAddr = listener.Addr().String()
	_ = listener.Close()

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	s.serverCancel = cancel

	// Start EventBus
	go func() {
		_ = s.EventBus.Start(ctx)
	}()

	// Start WebSocket Hub
	go s.WSHub.Run(ctx)

	// Start HTTP server
	s.serverWg.Add(1)
	go func() {
		defer s.serverWg.Done()
		if err := s.Echo.Start(s.serverAddr); err != nil && err != http.ErrServerClosed {
			s.t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to be ready
	s.waitForServer()

	// Register cleanup
	s.t.Cleanup(func() {
		s.Shutdown()
	})
}

// waitForServer waits for the server to be ready.
func (s *E2ETestSuite) waitForServer() {
	ctx, cancel := context.WithTimeout(context.Background(), serverStartupTimeout)
	defer cancel()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.t.Fatalf("Server failed to start within %v", serverStartupTimeout)
		case <-ticker.C:
			resp, err := http.Get(s.BaseURL() + "/health")
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return
				}
			}
		}
	}
}

// Shutdown gracefully shuts down the test suite.
func (s *E2ETestSuite) Shutdown() {
	if s.serverCancel != nil {
		s.serverCancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if s.Echo != nil {
		_ = s.Echo.Shutdown(ctx)
	}

	if s.WSHub != nil {
		s.WSHub.Stop()
	}

	if s.EventBus != nil {
		_ = s.EventBus.Shutdown()
	}

	s.serverWg.Wait()
}

// BaseURL returns the base URL of the test server.
func (s *E2ETestSuite) BaseURL() string {
	return "http://" + s.serverAddr
}

// APIURL returns the API base URL.
func (s *E2ETestSuite) APIURL() string {
	return s.BaseURL() + "/api/v1"
}

// WSURL returns the WebSocket URL.
func (s *E2ETestSuite) WSURL() string {
	return "ws://" + s.serverAddr + "/api/v1/ws"
}

// CreateTestUser creates a test user and returns it with an auth token.
func (s *E2ETestSuite) CreateTestUser(username string) *TestUser {
	s.t.Helper()

	id := uuid.NewUUID()
	email := username + "@test.local"
	externalID := "ext-" + id.String()

	// Create domain user
	u, err := user.NewUser(externalID, username, email, username)
	require.NoError(s.t, err)

	// Override ID for testing
	testUser := &TestUser{
		ID:       u.ID(),
		Username: username,
		Email:    email,
		Token:    "test-token-" + u.ID().String(),
		User:     u,
	}

	// Register token with validator
	s.TokenValidator.RegisterToken(testUser.Token, &middleware.TokenClaims{
		UserID:         testUser.ID,
		ExternalUserID: externalID,
		Username:       username,
		Email:          email,
		Roles:          []string{"user"},
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	})

	// Add to mock auth service
	s.MockAuthService.AddUser("code-"+username, u)
	s.MockUserRepo.AddUser(u)

	// Store user
	s.usersMu.Lock()
	s.users[username] = testUser
	s.usersMu.Unlock()

	return testUser
}

// CreateTestWorkspace creates a test workspace and adds the owner as member.
func (s *E2ETestSuite) CreateTestWorkspace(name string, owner *TestUser) *workspace.Workspace {
	s.t.Helper()

	ws, err := workspace.NewWorkspace(name, "Test workspace", "keycloak-group-test", owner.ID)
	require.NoError(s.t, err)

	// Add to mock service with member count
	s.MockWorkspaceService.AddWorkspace(ws, 1)

	// Add owner as member in mock member service
	member := workspace.NewMember(owner.ID, ws.ID(), workspace.RoleOwner)
	s.MockMemberService.AddMemberToMock(&member)
	s.MockMemberService.SetOwner(ws.ID(), owner.ID)

	return ws
}

// AddWorkspaceMember adds a user as member to a workspace.
func (s *E2ETestSuite) AddWorkspaceMember(ws *workspace.Workspace, user *TestUser, role workspace.Role) {
	s.t.Helper()

	member := workspace.NewMember(user.ID, ws.ID(), role)
	s.MockMemberService.AddMemberToMock(&member)
}

// GetTestUser returns a test user by username.
func (s *E2ETestSuite) GetTestUser(username string) *TestUser {
	s.usersMu.RLock()
	defer s.usersMu.RUnlock()
	return s.users[username]
}

// --- HTTP Client Helpers ---

// HTTPClient provides HTTP client methods for E2E tests.
type HTTPClient struct {
	t       *testing.T
	baseURL string
	token   string
}

// NewHTTPClient creates a new HTTP client for testing.
func (s *E2ETestSuite) NewHTTPClient(token string) *HTTPClient {
	return &HTTPClient{
		t:       s.t,
		baseURL: s.APIURL(),
		token:   token,
	}
}

// DoRequest performs an HTTP request and returns the response.
func (c *HTTPClient) DoRequest(method, path string, body interface{}) *http.Response {
	c.t.Helper()

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(c.t, err)
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	require.NoError(c.t, err)

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(c.t, err)

	return resp
}

// Get performs a GET request.
func (c *HTTPClient) Get(path string) *http.Response {
	return c.DoRequest(http.MethodGet, path, nil)
}

// Post performs a POST request.
func (c *HTTPClient) Post(path string, body interface{}) *http.Response {
	return c.DoRequest(http.MethodPost, path, body)
}

// Put performs a PUT request.
func (c *HTTPClient) Put(path string, body interface{}) *http.Response {
	return c.DoRequest(http.MethodPut, path, body)
}

// Delete performs a DELETE request.
func (c *HTTPClient) Delete(path string) *http.Response {
	return c.DoRequest(http.MethodDelete, path, nil)
}

// ParseResponse parses a JSON response into the given type.
func ParseResponse[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()

	var result T
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	return result
}

// ParseSuccessResponse parses a successful response with data field.
func ParseSuccessResponse[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()

	var wrapper struct {
		Success bool `json:"success"`
		Data    T    `json:"data"`
	}
	err := json.NewDecoder(resp.Body).Decode(&wrapper)
	require.NoError(t, err)
	require.True(t, wrapper.Success, "expected success response")

	return wrapper.Data
}

// AssertStatus asserts the response status code.
func AssertStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d: %s", expected, resp.StatusCode, string(body))
	}
}

// --- WebSocket Helpers ---

// WSClient provides WebSocket client methods for E2E tests.
type WSClient struct {
	t    *testing.T
	conn *websocket.Conn
}

// ConnectWebSocket establishes a WebSocket connection.
func (s *E2ETestSuite) ConnectWebSocket(token string) *WSClient {
	s.t.Helper()

	dialer := websocket.Dialer{
		HandshakeTimeout: wsConnectTimeout,
	}

	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)

	conn, resp, err := dialer.Dial(s.WSURL(), header)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			s.t.Fatalf("WebSocket dial failed: %v, response: %s", err, string(body))
		}
		s.t.Fatalf("WebSocket dial failed: %v", err)
	}

	return &WSClient{
		t:    s.t,
		conn: conn,
	}
}

// Close closes the WebSocket connection.
func (c *WSClient) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

// SendJSON sends a JSON message.
func (c *WSClient) SendJSON(v interface{}) {
	c.t.Helper()
	err := c.conn.WriteJSON(v)
	require.NoError(c.t, err)
}

// ReadJSON reads a JSON message with timeout.
func (c *WSClient) ReadJSON(v interface{}, timeout time.Duration) error {
	c.t.Helper()
	_ = c.conn.SetReadDeadline(time.Now().Add(timeout))
	return c.conn.ReadJSON(v)
}

// Subscribe sends a subscribe message.
func (c *WSClient) Subscribe(chatID uuid.UUID) {
	c.t.Helper()
	c.SendJSON(map[string]string{
		"type":    "subscribe",
		"chat_id": chatID.String(),
	})
}

// WSEvent represents a WebSocket event.
type WSEvent struct {
	Type   string                 `json:"type"`
	ChatID string                 `json:"chat_id,omitempty"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

// ReadEvent reads a WebSocket event with timeout.
func (c *WSClient) ReadEvent(timeout time.Duration) (*WSEvent, error) {
	var event WSEvent
	if err := c.ReadJSON(&event, timeout); err != nil {
		return nil, err
	}
	return &event, nil
}

// --- Test Config ---

// E2ETestConfig returns a test configuration.
func E2ETestConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = 0 // Random port
	cfg.Log.Level = "error"
	cfg.Log.Format = "text"
	cfg.EventBus.Type = "redis"
	cfg.EventBus.RedisChannelPrefix = "e2e_test:"
	return cfg
}

// --- Test Main ---

var sharedSuite *E2ETestSuite
var sharedSuiteMu sync.Mutex
var sharedSuiteOnce sync.Once

// GetSharedSuite returns a shared test suite (for tests that don't need isolation).
func GetSharedSuite(t *testing.T) *E2ETestSuite {
	sharedSuiteMu.Lock()
	defer sharedSuiteMu.Unlock()

	if sharedSuite == nil {
		sharedSuite = NewE2ETestSuite(t)
	}
	return sharedSuite
}

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Cleanup shared containers
	testutil.CleanupSharedContainer()
	testutil.CleanupSharedRedisContainer()

	os.Exit(code)
}
