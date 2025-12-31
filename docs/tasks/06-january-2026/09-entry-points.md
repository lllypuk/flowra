# 09: Entry Points (cmd/api)

**Приоритет:** 🔴 Critical  
**Статус:** ⏳ Не начато  
**Дни:** 22-24 января  
**Зависит от:** Все предыдущие задачи

---

## Описание

Создать entry points для приложения: `cmd/api/main.go`, dependency injection container и конфигурацию роутов. Это финальная сборка всех компонентов в работающее приложение.

---

## Файлы для создания

```
cmd/api/
├── main.go                 (~500 LOC)
├── container.go            (~400 LOC)
└── routes.go               (~300 LOC)

internal/config/
├── config.go               (~200 LOC)
└── loader.go               (~150 LOC)
```

---

## Детали реализации

### 1. main.go

Точка входа приложения:

```go
func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("failed to load config:", err)
    }
    
    // Setup logger
    logger := setupLogger(cfg.LogLevel)
    
    // Build DI container
    container, err := buildContainer(cfg, logger)
    if err != nil {
        log.Fatal("failed to build container:", err)
    }
    defer container.Close()
    
    // Setup router
    router := setupRoutes(container)
    
    // Start Event Bus
    if err := container.EventBus.Start(context.Background()); err != nil {
        log.Fatal("failed to start event bus:", err)
    }
    
    // Start WebSocket Hub
    go container.Hub.Run(context.Background())
    
    // Graceful shutdown
    go gracefulShutdown(router, container)
    
    // Start server
    logger.Info("starting server", "address", cfg.Server.Address)
    if err := router.Start(cfg.Server.Address); err != http.ErrServerClosed {
        log.Fatal("server error:", err)
    }
}
```

### 2. Dependency Injection Container

```go
type Container struct {
    // Configuration
    Config *config.Config
    Logger *slog.Logger
    
    // Infrastructure
    MongoDB    *mongo.Client
    Redis      *redis.Client
    EventStore EventStore
    EventBus   EventBus
    Hub        *websocket.Hub
    
    // Repositories
    UserRepo         user.Repository
    WorkspaceRepo    workspace.Repository
    ChatRepo         chat.Repository
    MessageRepo      message.Repository
    TaskRepo         task.Repository
    NotificationRepo notification.Repository
    
    // Use Cases
    CreateChatUC     *chat.CreateChatUseCase
    SendMessageUC    *message.SendMessageUseCase
    CreateTaskUC     *task.CreateTaskUseCase
    // ... все остальные use cases
    
    // Handlers
    AuthHandler      *http.AuthHandler
    WorkspaceHandler *http.WorkspaceHandler
    ChatHandler      *http.ChatHandler
    MessageHandler   *http.MessageHandler
    TaskHandler      *http.TaskHandler
    NotifHandler     *http.NotificationHandler
    UserHandler      *http.UserHandler
    WSHandler        *wshandler.Handler
}

func buildContainer(cfg *config.Config, logger *slog.Logger) (*Container, error) {
    c := &Container{
        Config: cfg,
        Logger: logger,
    }
    
    // 1. Infrastructure
    if err := c.setupInfrastructure(); err != nil {
        return nil, fmt.Errorf("infrastructure: %w", err)
    }
    
    // 2. Repositories
    c.setupRepositories()
    
    // 3. Use Cases
    c.setupUseCases()
    
    // 4. Handlers
    c.setupHandlers()
    
    // 5. Event Handlers
    c.registerEventHandlers()
    
    return c, nil
}

func (c *Container) Close() error {
    var errs []error
    
    if c.EventBus != nil {
        if err := c.EventBus.Shutdown(); err != nil {
            errs = append(errs, err)
        }
    }
    
    if c.MongoDB != nil {
        if err := c.MongoDB.Disconnect(context.Background()); err != nil {
            errs = append(errs, err)
        }
    }
    
    if c.Redis != nil {
        if err := c.Redis.Close(); err != nil {
            errs = append(errs, err)
        }
    }
    
    return errors.Join(errs...)
}
```

### 3. Routes Setup

```go
func setupRoutes(c *Container) *echo.Echo {
    e := echo.New()
    
    // Global middleware
    e.Use(middleware.Logger())
    e.Use(middleware.Recover())
    e.Use(middlewares.RequestID())
    e.Use(middlewares.CORS(c.Config.CORS))
    
    // Health check
    e.GET("/health", healthCheck)
    e.GET("/ready", readinessCheck(c))
    
    // API v1
    v1 := e.Group("/api/v1")
    
    // Public routes
    v1.POST("/auth/login", c.AuthHandler.Login)
    v1.POST("/auth/refresh", c.AuthHandler.Refresh)
    
    // Authenticated routes
    auth := v1.Group("", middlewares.Auth(c.Config.JWT))
    
    auth.POST("/auth/logout", c.AuthHandler.Logout)
    auth.GET("/auth/me", c.AuthHandler.Me)
    
    // Workspaces
    auth.POST("/workspaces", c.WorkspaceHandler.Create)
    auth.GET("/workspaces", c.WorkspaceHandler.List)
    
    ws := auth.Group("/workspaces/:workspace_id", 
        middlewares.WorkspaceAccess(c.WorkspaceRepo))
    
    ws.GET("", c.WorkspaceHandler.Get)
    ws.PUT("", c.WorkspaceHandler.Update)
    ws.DELETE("", c.WorkspaceHandler.Delete)
    
    // Chats
    ws.POST("/chats", c.ChatHandler.Create)
    ws.GET("/chats", c.ChatHandler.List)
    
    // Tasks
    ws.POST("/tasks", c.TaskHandler.Create)
    ws.GET("/tasks", c.TaskHandler.List)
    
    // ... остальные роуты
    
    // WebSocket
    auth.GET("/ws", c.WSHandler.HandleWebSocket)
    
    return e
}
```

### 4. Configuration

```go
type Config struct {
    Server   ServerConfig
    MongoDB  MongoDBConfig
    Redis    RedisConfig
    JWT      JWTConfig
    CORS     CORSConfig
    OAuth    OAuthConfig
    LogLevel string
}

type ServerConfig struct {
    Address         string
    ReadTimeout     time.Duration
    WriteTimeout    time.Duration
    ShutdownTimeout time.Duration
}

type MongoDBConfig struct {
    URI      string
    Database string
}

type RedisConfig struct {
    Address  string
    Password string
    DB       int
}

func Load() (*Config, error) {
    // 1. Load from configs/config.yaml
    // 2. Override with environment variables
    // 3. Validate required fields
    return cfg, nil
}
```

### 5. Graceful Shutdown

```go
func gracefulShutdown(server *echo.Echo, container *Container) {
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    <-quit
    
    container.Logger.Info("shutting down server...")
    
    ctx, cancel := context.WithTimeout(
        context.Background(), 
        container.Config.Server.ShutdownTimeout,
    )
    defer cancel()
    
    // 1. Stop accepting new connections
    if err := server.Shutdown(ctx); err != nil {
        container.Logger.Error("server shutdown error", "error", err)
    }
    
    // 2. Close container resources
    if err := container.Close(); err != nil {
        container.Logger.Error("container close error", "error", err)
    }
    
    container.Logger.Info("server stopped")
}
```

---

## Health Checks

### /health (Liveness)

Проверяет, что приложение живо:

```go
func healthCheck(c echo.Context) error {
    return c.JSON(200, map[string]string{
        "status": "ok",
    })
}
```

### /ready (Readiness)

Проверяет, что все зависимости доступны:

```go
func readinessCheck(container *Container) echo.HandlerFunc {
    return func(c echo.Context) error {
        // Check MongoDB
        if err := container.MongoDB.Ping(c.Request().Context(), nil); err != nil {
            return c.JSON(503, map[string]string{
                "status": "not ready",
                "error":  "mongodb unavailable",
            })
        }
        
        // Check Redis
        if err := container.Redis.Ping(c.Request().Context()).Err(); err != nil {
            return c.JSON(503, map[string]string{
                "status": "not ready", 
                "error":  "redis unavailable",
            })
        }
        
        return c.JSON(200, map[string]string{
            "status": "ready",
        })
    }
}
```

---

## Критерии приёмки

- [ ] `go run cmd/api/main.go` запускает приложение
- [ ] Configuration загружается из YAML и ENV
- [ ] Все dependencies корректно инициализируются
- [ ] DI container правильно связывает компоненты
- [ ] Все роуты зарегистрированы
- [ ] Health check endpoints работают
- [ ] Graceful shutdown корректен
- [ ] Event Bus запускается при старте
- [ ] WebSocket Hub запускается
- [ ] Логирование работает

---

## Чеклист

### main.go
- [ ] Configuration loading
- [ ] Logger setup
- [ ] Container building
- [ ] Router setup
- [ ] Server start
- [ ] Graceful shutdown

### container.go
- [ ] MongoDB connection
- [ ] Redis connection
- [ ] EventStore initialization
- [ ] EventBus initialization
- [ ] All repositories
- [ ] All use cases
- [ ] All handlers
- [ ] Event handlers registration
- [ ] Close method

### routes.go
- [ ] Global middleware
- [ ] Health checks
- [ ] Auth routes
- [ ] Workspace routes
- [ ] Chat routes
- [ ] Message routes
- [ ] Task routes
- [ ] Notification routes
- [ ] User routes
- [ ] WebSocket route

### config/
- [ ] Config structure
- [ ] YAML loader
- [ ] ENV override
- [ ] Validation

---

## Зависимости

### Требуется
- Все предыдущие задачи (01-08)
- Все handlers реализованы
- Все use cases реализованы
- Все repositories реализованы

### Блокирует
- [10-e2e-tests.md](10-e2e-tests.md) — нужен работающий сервер
- [11-documentation.md](11-documentation.md) — документация API

---

## Заметки

- Используем manual DI вместо wire/dig для простоты
- Configuration validation должна быть строгой
- Порядок инициализации важен: infra → repos → use cases → handlers
- Container.Close() должен закрывать ресурсы в обратном порядке
- Рассмотреть feature flags для включения/выключения функциональности

---

## Команды для тестирования

```bash
# Запуск сервера
go run cmd/api/main.go

# С кастомным конфигом
CONFIG_PATH=./configs/dev.yaml go run cmd/api/main.go

# Health check
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

---

*Создано: 2026-01-01*