# Task 3.1: Application Entry Points

**Приоритет:** 🔴 КРИТИЧЕСКИЙ
**Время:** 4-5 дней
**Зависимости:** Phase 1, Phase 2 завершены

---

## Объединяет Tasks

- Task 3.1.1: API Server (cmd/api/main.go)
- Task 3.1.2: Worker Service (cmd/worker/main.go)
- Task 3.1.3: Database Migrator (cmd/migrator/main.go)

---

## Цель

Собрать все компоненты воедино и запустить приложение.

---

## Файлы

```
cmd/
├── api/
│   └── main.go          (HTTP/WebSocket server)
├── worker/
│   └── main.go          (background event processor)
└── migrator/
    └── main.go          (database migrations)

internal/config/
├── config.go            (config structure)
└── loader.go            (load from yaml + env)

configs/
└── config.yaml          (default config)

migrations/mongodb/
├── 001_event_store_schema.js
├── 002_chat_read_model.js
├── 003_messages.js
├── 004_users.js
├── 005_workspaces.js
└── 006_notifications.js
```

---

## 1. API Server (cmd/api/main.go)

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/lllypuk/flowra/internal/config"
    "github.com/lllypuk/flowra/internal/infrastructure/eventstore"
    "github.com/lllypuk/flowra/internal/infrastructure/eventbus"
    "github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
    "github.com/lllypuk/flowra/internal/infrastructure/repository/redis"
    "github.com/lllypuk/flowra/internal/infrastructure/keycloak"
    "github.com/lllypuk/flowra/internal/infrastructure/websocket"
    "github.com/lllypuk/flowra/internal/application/chat"
    "github.com/lllypuk/flowra/internal/application/message"
    httphandler "github.com/lllypuk/flowra/internal/handler/http"
    "github.com/lllypuk/flowra/internal/middleware"
)

func main() {
    // 1. Load configuration
    cfg, err := config.Load("configs/config.yaml")
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }

    // 2. Initialize logger
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))

    // 3. Connect to MongoDB
    mongoClient, err := mongodb.Connect(cfg.MongoDB)
    if err != nil {
        logger.Error("Failed to connect to MongoDB", "error", err)
        os.Exit(1)
    }
    defer mongoClient.Disconnect(context.Background())

    // 4. Connect to Redis
    redisClient := redis.NewClient(&redis.Options{
        Addr: cfg.Redis.Addr,
    })
    defer redisClient.Close()

    // 5. Initialize Keycloak client
    keycloakClient := keycloak.NewHTTPClient(cfg.Keycloak.URL, cfg.Keycloak.Realm, cfg.Keycloak.ClientID, cfg.Keycloak.ClientSecret)
    tokenValidator := keycloak.NewTokenValidator(cfg.Keycloak.URL, cfg.Keycloak.Realm)

    // 6. Initialize infrastructure
    eventStore := eventstore.NewMongoEventStore(mongoClient, cfg.MongoDB.Database)
    eventBus := eventbus.NewRedisEventBus(redisClient)

    // 7. Initialize repositories
    chatRepo := mongodb.NewChatRepository(mongoClient, cfg.MongoDB.Database, eventStore)
    messageRepo := mongodb.NewMessageRepository(mongoClient, cfg.MongoDB.Database)
    userRepo := mongodb.NewUserRepository(mongoClient, cfg.MongoDB.Database)
    workspaceRepo := mongodb.NewWorkspaceRepository(mongoClient, cfg.MongoDB.Database)
    notificationRepo := mongodb.NewNotificationRepository(mongoClient, cfg.MongoDB.Database)
    sessionRepo := redis.NewSessionRepository(redisClient)

    // 8. Initialize use cases
    createChatUC := chat.NewCreateChatUseCase(eventStore, userRepo, workspaceRepo)
    getChatUC := chat.NewGetChatUseCase(chatRepo, eventStore)
    listChatsUC := chat.NewListChatsUseCase(chatRepo)
    sendMessageUC := message.NewSendMessageUseCase(messageRepo, chatRepo, eventStore, tagProcessor)
    // ... all other use cases

    // 9. Initialize WebSocket hub
    wsHub := websocket.NewHub()
    go wsHub.Run()

    // 10. Initialize HTTP handlers
    authHandler := httphandler.NewAuthHandler(keycloakClient, sessionRepo)
    chatHandler := httphandler.NewChatHandler(createChatUC, getChatUC, listChatsUC)
    messageHandler := httphandler.NewMessageHandler(sendMessageUC /* ... */)
    workspaceHandler := httphandler.NewWorkspaceHandler(/* ... */)
    notificationHandler := httphandler.NewNotificationHandler(/* ... */)
    wsHandler := websocket.NewHandler(wsHub, tokenValidator)

    // 11. Initialize middleware
    authMiddleware := middleware.NewAuthMiddleware(tokenValidator, userRepo)
    workspaceMiddleware := middleware.NewWorkspaceMiddleware(workspaceRepo)
    rateLimiter := middleware.NewRateLimiter(redisClient)

    // 12. Setup router
    router := httphandler.NewRouter(
        authHandler,
        chatHandler,
        messageHandler,
        workspaceHandler,
        notificationHandler,
        wsHandler,
        authMiddleware,
        workspaceMiddleware,
        rateLimiter,
        logger,
    )

    // 13. Start HTTP server
    go func() {
        logger.Info("Starting server", "port", cfg.Server.Port)
        if err := router.Start(":" + cfg.Server.Port); err != nil {
            logger.Error("Server error", "error", err)
        }
    }()

    // 14. Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    <-quit

    logger.Info("Shutting down server...")

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := router.Shutdown(ctx); err != nil {
        logger.Error("Server shutdown error", "error", err)
    }

    wsHub.Shutdown()
    eventBus.Shutdown()

    logger.Info("Server stopped")
}
```

---

## 2. Worker Service (cmd/worker/main.go)

```go
package main

func main() {
    // Similar setup as API server

    // Initialize event handlers
    tagProcessorHandler := eventhandler.NewTagProcessorHandler(/* ... */)
    notificationHandler := eventhandler.NewNotificationHandler(/* ... */)
    projectionHandler := eventhandler.NewProjectionHandler(/* ... */)

    // Subscribe to events
    eventBus.Subscribe("MessagePosted", tagProcessorHandler)
    eventBus.Subscribe("ChatCreated", notificationHandler)
    eventBus.Subscribe("StatusChanged", projectionHandler)

    logger.Info("Worker started")

    // Wait for shutdown signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    <-quit

    logger.Info("Worker shutting down...")
    eventBus.Shutdown()
    logger.Info("Worker stopped")
}
```

---

## 3. Database Migrator (cmd/migrator/main.go)

```go
package main

func main() {
    // Load config
    cfg, _ := config.Load("configs/config.yaml")

    // Connect to MongoDB
    client, _ := mongodb.Connect(cfg.MongoDB)
    defer client.Disconnect(context.Background())

    // Apply migrations
    migrator := migration.NewMigrator(client, "migrations/mongodb")

    if err := migrator.Up(); err != nil {
        log.Fatal("Migration failed:", err)
    }

    log.Println("Migrations applied successfully")
}
```

---

## 4. Configuration (config.go)

```go
type Config struct {
    Server   ServerConfig
    MongoDB  MongoDBConfig
    Redis    RedisConfig
    Keycloak KeycloakConfig
    Log      LogConfig
}

type ServerConfig struct {
    Port string `yaml:"port"`
}

type MongoDBConfig struct {
    URI      string `yaml:"uri"`
    Database string `yaml:"database"`
}

type RedisConfig struct {
    Addr string `yaml:"addr"`
}

type KeycloakConfig struct {
    URL          string `yaml:"url"`
    Realm        string `yaml:"realm"`
    ClientID     string `yaml:"client_id"`
    ClientSecret string `yaml:"client_secret"`
}

func Load(path string) (*Config, error) {
    // Load from yaml
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }

    // Override with env variables
    if port := os.Getenv("APP_SERVER_PORT"); port != "" {
        cfg.Server.Port = port
    }
    if uri := os.Getenv("APP_MONGODB_URI"); uri != "" {
        cfg.MongoDB.URI = uri
    }

    return &cfg, nil
}
```

---

## 5. Default Config (configs/config.yaml)

```yaml
server:
  port: "8080"

mongodb:
  uri: "mongodb://localhost:27017"
  database: "flowra"

redis:
  addr: "localhost:6379"

keycloak:
  url: "http://localhost:8090"
  realm: "flowra"
  client_id: "flowra-app"
  client_secret: "secret"

log:
  level: "info"
```

---

## Running

```bash
# Apply migrations
go run cmd/migrator/main.go

# Start API server
go run cmd/api/main.go

# Start worker (in separate terminal)
go run cmd/worker/main.go

# Or use Makefile
make migrate
make run-api
make run-worker
```

---

## Критерии успеха

- ✅ **Приложение запускается**
- ✅ **All dependencies корректно инжектятся**
- ✅ **Health check проходит**
- ✅ **Graceful shutdown работает**
- ✅ **Configuration управляема**

---

Phase 3 завершена → **Phase 4: Frontend**
