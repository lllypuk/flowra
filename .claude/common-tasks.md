# Common Tasks для Claude

## Обзор

Этот документ содержит часто выполняемые задачи и команды для работы с проектом New Teams Up. Используй его как справочник для быстрого выполнения типичных операций разработки.

## Структура разработки

### Создание нового микросервиса

1. **Создать структуру каталогов**:
```bash
mkdir -p cmd/{service-name}
mkdir -p internal/{service-name}/{domain,application,infrastructure,presentation}
mkdir -p internal/{service-name}/domain/{entities,repositories,services}
mkdir -p internal/{service-name}/application/{commands,queries,handlers}
mkdir -p internal/{service-name}/infrastructure/{database,http,messaging}
mkdir -p internal/{service-name}/presentation/{handlers,dto,middleware}
```

2. **Создать main.go для сервиса**:
```go
// cmd/{service-name}/main.go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "{project}/internal/{service-name}/infrastructure/http"
    "{project}/pkg/config"
    "{project}/pkg/logger"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }

    logger := logger.New(cfg.Logger)

    server := http.NewServer(cfg.Server, logger)

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go func() {
        if err := server.Start(); err != nil {
            logger.Error("Server failed to start", "error", err)
            cancel()
        }
    }()

    // Graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    select {
    case <-sigChan:
        logger.Info("Shutdown signal received")
    case <-ctx.Done():
        logger.Info("Context cancelled")
    }

    if err := server.Shutdown(ctx); err != nil {
        logger.Error("Server shutdown failed", "error", err)
    }
}
```

### Создание новой доменной сущности

1. **Entity структура**:
```go
// internal/{service}/domain/entities/{entity}.go
package entities

import (
    "time"
    "errors"
)

var (
    Err{Entity}NotFound = errors.New("{entity} not found")
    ErrInvalid{Entity} = errors.New("invalid {entity}")
)

type {Entity}ID string

type {Entity} struct {
    id        {Entity}ID
    // поля сущности
    createdAt time.Time
    updatedAt time.Time
}

func New{Entity}(...params) (*{Entity}, error) {
    // валидация и создание
    if err := validate{Entity}(...params); err != nil {
        return nil, err
    }

    return &{Entity}{
        id:        generate{Entity}ID(),
        // инициализация полей
        createdAt: time.Now(),
        updatedAt: time.Now(),
    }, nil
}

// Геттеры
func (e *{Entity}) ID() {Entity}ID { return e.id }
func (e *{Entity}) CreatedAt() time.Time { return e.createdAt }
func (e *{Entity}) UpdatedAt() time.Time { return e.updatedAt }

// Бизнес методы
func (e *{Entity}) DoSomething() error {
    // бизнес логика
    e.updatedAt = time.Now()
    return nil
}

// Приватные методы
func validate{Entity}(...params) error {
    // валидация
    return nil
}

func generate{Entity}ID() {Entity}ID {
    // генерация ID
    return {Entity}ID("generated-id")
}
```

2. **Repository интерфейс**:
```go
// internal/{service}/domain/repositories/{entity}_repository.go
package repositories

import (
    "context"
    "{project}/internal/{service}/domain/entities"
)

type {Entity}Repository interface {
    Save(ctx context.Context, entity *entities.{Entity}) error
    FindByID(ctx context.Context, id entities.{Entity}ID) (*entities.{Entity}, error)
    FindAll(ctx context.Context, filters ...Filter) ([]*entities.{Entity}, error)
    Delete(ctx context.Context, id entities.{Entity}ID) error
    Exists(ctx context.Context, id entities.{Entity}ID) (bool, error)
}

type Filter interface {
    Apply(query string, args []interface{}) (string, []interface{})
}
```

### Создание Use Case

```go
// internal/{service}/application/commands/create_{entity}_command.go
package commands

type Create{Entity}Command struct {
    // поля команды
}

func (c Create{Entity}Command) Validate() error {
    // валидация команды
    return nil
}
```

```go
// internal/{service}/application/handlers/create_{entity}_handler.go
package handlers

import (
    "context"
    "fmt"

    "{project}/internal/{service}/domain/entities"
    "{project}/internal/{service}/domain/repositories"
    "{project}/internal/{service}/application/commands"
    "{project}/pkg/logger"
)

type Create{Entity}Handler struct {
    repo   repositories.{Entity}Repository
    logger logger.Logger
}

func NewCreate{Entity}Handler(repo repositories.{Entity}Repository, logger logger.Logger) *Create{Entity}Handler {
    return &Create{Entity}Handler{
        repo:   repo,
        logger: logger,
    }
}

func (h *Create{Entity}Handler) Handle(ctx context.Context, cmd commands.Create{Entity}Command) (*entities.{Entity}, error) {
    if err := cmd.Validate(); err != nil {
        return nil, fmt.Errorf("validating command: %w", err)
    }

    entity, err := entities.New{Entity}(/* params from cmd */)
    if err != nil {
        return nil, fmt.Errorf("creating {entity}: %w", err)
    }

    if err := h.repo.Save(ctx, entity); err != nil {
        return nil, fmt.Errorf("saving {entity}: %w", err)
    }

    h.logger.Info("{entity} created",
        "id", entity.ID(),
    )

    return entity, nil
}
```

### Создание HTTP Handler

```go
// internal/{service}/presentation/handlers/{entity}_handler.go
package handlers

import (
    "encoding/json"
    "net/http"

    "github.com/gorilla/mux"

    "{project}/internal/{service}/application/commands"
    "{project}/internal/{service}/application/handlers"
    "{project}/internal/{service}/presentation/dto"
    "{project}/pkg/logger"
)

type {Entity}Handler struct {
    createHandler *handlers.Create{Entity}Handler
    logger        logger.Logger
}

func New{Entity}Handler(createHandler *handlers.Create{Entity}Handler, logger logger.Logger) *{Entity}Handler {
    return &{Entity}Handler{
        createHandler: createHandler,
        logger:        logger,
    }
}

func (h *{Entity}Handler) Create(w http.ResponseWriter, r *http.Request) {
    var req dto.Create{Entity}Request
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.respondWithError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
        return
    }

    cmd := commands.Create{Entity}Command{
        // маппинг из request в command
    }

    entity, err := h.createHandler.Handle(r.Context(), cmd)
    if err != nil {
        h.handleError(w, err)
        return
    }

    response := dto.To{Entity}Response(entity)
    h.respondWithData(w, http.StatusCreated, response)
}

func (h *{Entity}Handler) respondWithError(w http.ResponseWriter, statusCode int, code, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)

    errorResponse := map[string]interface{}{
        "error": map[string]string{
            "code":    code,
            "message": message,
        },
    }
    json.NewEncoder(w).Encode(errorResponse)
}

func (h *{Entity}Handler) respondWithData(w http.ResponseWriter, statusCode int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)

    response := map[string]interface{}{
        "data": data,
    }
    json.NewEncoder(w).Encode(response)
}
```

## Database Operations

### Создание новой миграции

```bash
# MongoDB не требует SQL миграций
# Schema versioning управляется через application code
# Индексы создаются при инициализации приложения
```

### Шаблон миграции

```sql
-- migrations/{version}_create_{table}_table.up.sql
CREATE TABLE {table} (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- другие поля
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Индексы
CREATE INDEX idx_{table}_created_at ON {table}(created_at);
CREATE INDEX idx_{table}_deleted_at ON {table}(deleted_at) WHERE deleted_at IS NULL;

-- migrations/{version}_create_{table}_table.down.sql
DROP TABLE IF EXISTS {table};
```

### Repository Implementation

```go
// internal/{service}/infrastructure/database/{entity}_repository.go
package database

import (
    "context"
    "database/sql"
    "fmt"

    "github.com/jmoiron/sqlx"

    "{project}/internal/{service}/domain/entities"
    "{project}/internal/{service}/domain/repositories"
    "{project}/pkg/logger"
)

type MongoDB{Entity}Repository struct {
    db     *mongo.Database
    logger logger.Logger
}

func NewMongoDB{Entity}Repository(db *mongo.Database, logger logger.Logger) *MongoDB{Entity}Repository {
    return &MongoDB{Entity}Repository{
        db:     db,
        logger: logger,
    }
}

func (r *MongoDB{Entity}Repository) Save(ctx context.Context, entity *entities.{Entity}) error {
    query := `
        INSERT INTO {table} (id, field1, field2, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (id) DO UPDATE SET
            field1 = EXCLUDED.field1,
            field2 = EXCLUDED.field2,
            updated_at = EXCLUDED.updated_at`

    _, err := r.db.ExecContext(ctx, query,
        entity.ID(),
        entity.Field1(),
        entity.Field2(),
        entity.CreatedAt(),
        entity.UpdatedAt(),
    )

    if err != nil {
        r.logger.Error("failed to save {entity}",
            "id", entity.ID(),
            "error", err,
        )
        return fmt.Errorf("saving {entity}: %w", err)
    }

    return nil
}

func (r *MongoDB{Entity}Repository) FindByID(ctx context.Context, id entities.{Entity}ID) (*entities.{Entity}, error) {
    query := `
        SELECT id, field1, field2, created_at, updated_at
        FROM {table}
        WHERE id = $1 AND deleted_at IS NULL`

    var row struct {
        ID        string    `db:"id"`
        Field1    string    `db:"field1"`
        Field2    string    `db:"field2"`
        CreatedAt time.Time `db:"created_at"`
        UpdatedAt time.Time `db:"updated_at"`
    }

    err := r.db.GetContext(ctx, &row, query, id)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, entities.Err{Entity}NotFound
        }
        return nil, fmt.Errorf("querying {entity} by id: %w", err)
    }

    return r.toDomainEntity(row), nil
}

func (r *MongoDB{Entity}Repository) toDomainEntity(row struct{...}) *entities.{Entity} {
    // конвертация из DB модели в доменную сущность
}
```

## Testing Patterns

### Unit Test Template

```go
// internal/{service}/{layer}/{file}_test.go
package {package}

import (
    "context"
    "testing"

    "github.com/golang/mock/gomock"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "{project}/internal/{service}/mocks"
)

func Test{Service}_{Method}_Success(t *testing.T) {
    // Arrange
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockRepo := mocks.NewMock{Entity}Repository(ctrl)
    service := New{Service}(mockRepo)

    // Setup mocks
    mockRepo.EXPECT().
        SomeMethod(gomock.Any(), gomock.Any()).
        Return(expectedResult, nil)

    // Act
    result, err := service.MethodUnderTest(context.Background(), input)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, expected, result)
}

func Test{Service}_{Method}_Error(t *testing.T) {
    // Arrange
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockRepo := mocks.NewMock{Entity}Repository(ctrl)
    service := New{Service}(mockRepo)

    expectedError := errors.New("some error")
    mockRepo.EXPECT().
        SomeMethod(gomock.Any(), gomock.Any()).
        Return(nil, expectedError)

    // Act
    _, err := service.MethodUnderTest(context.Background(), input)

    // Assert
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "some error")
}
```

### Integration Test Template

```go
//go:build integration

package integration

import (
    "context"
    "testing"

    "github.com/stretchr/testify/suite"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/mongodb"
    "go.mongodb.org/mongo-driver/v2/mongo"
    "go.mongodb.org/mongo-driver/v2/mongo/options"
)

type {Service}IntegrationTestSuite struct {
    suite.Suite
    client    *mongo.Client
    container *mongodb.MongoDBContainer
    service   *{Service}
}

func (s *{Service}IntegrationTestSuite) SetupSuite() {
    ctx := context.Background()

    // Start MongoDB container
    mongoContainer, err := mongodb.RunContainer(ctx,
        testcontainers.WithImage("mongo:8"),
        mongodb.WithUsername("admin"),
        mongodb.WithPassword("admin123"),
    )
    s.Require().NoError(err)

    s.container = mongoContainer

    // Get connection string and connect
    uri, err := mongoContainer.ConnectionString(ctx)
    s.Require().NoError(err)

    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
    s.Require().NoError(err)
    s.client = client

    // Setup service
    db := client.Database("testdb")
    s.service = New{Service}(NewRepository(db))
}

func (s *{Service}IntegrationTestSuite) TearDownSuite() {
    ctx := context.Background()
    s.client.Disconnect(ctx)
    s.container.Terminate(ctx)
}

func (s *{Service}IntegrationTestSuite) SetupTest() {
    s.clearTestData()
}

func TestIntegration{Service}(t *testing.T) {
    suite.Run(t, new({Service}IntegrationTestSuite))
}
```

## Monitoring & Logging

### Добавление метрик

```go
// pkg/metrics/{service}_metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    {operation}Counter = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "{service}_{operation}_total",
            Help: "Total number of {operation} operations",
        },
        []string{"status"},
    )

    {operation}Duration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "{service}_{operation}_duration_seconds",
            Help: "Duration of {operation} operations",
        },
        []string{"status"},
    )
)

func RecordOperation(operation string, duration float64, success bool) {
    status := "success"
    if !success {
        status = "error"
    }

    {operation}Counter.WithLabelValues(status).Inc()
    {operation}Duration.WithLabelValues(status).Observe(duration)
}
```

### Structured Logging

```go
import "go.uber.org/zap"

// В начале операции
logger.Info("starting {operation}",
    zap.String("entity_id", id.String()),
    zap.String("operation", "{operation}"),
    zap.String("user_id", userID),
)

// При успехе
logger.Info("{operation} completed successfully",
    zap.String("entity_id", result.ID().String()),
    zap.Duration("duration", time.Since(start)),
)

// При ошибке
logger.Error("{operation} failed",
    zap.Error(err),
    zap.String("entity_id", id.String()),
    zap.String("reason", "validation_failed"),
    zap.Duration("duration", time.Since(start)),
)
```

## Docker & Deployment

### Dockerfile Template

```dockerfile
# Multi-stage build
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/{service}/

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy config files
COPY --from=builder /app/configs ./configs

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
CMD ["./main"]
```

### Docker Compose Service

```yaml
# docker-compose.yml
version: '3.8'

services:
  {service}:
    build:
      context: .
      dockerfile: Dockerfile.{service}
    ports:
      - "808{x}:8080"
    environment:
      - ENV=development
      - MONGODB_URI=mongodb://admin:admin123@mongodb:27017
      - REDIS_HOST=redis
    depends_on:
      - mongodb
      - redis
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## Useful Commands

### Development

```bash
# Start development environment
make dev

# Run tests
make test
make test-unit
make test-integration

# Code quality
make lint
make fmt
make vet

# Generate mocks
go generate ./...

# Build
make build

# Database
make migrate-up
make migrate-down
make seed
```

### Git Workflow

```bash
# Feature development
git checkout -b feature/TASK-123-new-feature
git add .
git commit -m "feat: add new feature for user management"
git push origin feature/TASK-123-new-feature

# Bug fix
git checkout -b fix/TASK-456-fix-validation
git commit -m "fix: resolve email validation issue"

# Hotfix
git checkout -b hotfix/critical-security-patch
git commit -m "fix: patch critical security vulnerability"
```

### Troubleshooting Commands

```bash
# Check logs
docker logs {container_name}
kubectl logs -f deployment/{service}-deployment

# Database debugging
mongosh mongodb://admin:admin123@localhost:27017/teams_up
show collections # list collections
db.{collection}.find().limit(10) # query collection

# Performance profiling
go tool pprof http://localhost:8080/debug/pprof/profile
go tool pprof http://localhost:8080/debug/pprof/heap

# Check metrics
curl http://localhost:8080/metrics

# Health check
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

## Quick References

### HTTP Status Codes
- `200` - OK (successful GET, PUT)
- `201` - Created (successful POST)
- `204` - No Content (successful DELETE)
- `400` - Bad Request (validation error)
- `401` - Unauthorized (authentication required)
- `403` - Forbidden (insufficient permissions)
- `404` - Not Found (resource doesn't exist)
- `409` - Conflict (resource already exists)
- `422` - Unprocessable Entity (semantic error)
- `500` - Internal Server Error (unexpected error)

### Common Error Codes
- `VALIDATION_FAILED` - Input validation error
- `RESOURCE_NOT_FOUND` - Requested resource not found
- `RESOURCE_ALREADY_EXISTS` - Resource already exists
- `UNAUTHORIZED` - Authentication required
- `FORBIDDEN` - Insufficient permissions
- `INTERNAL_ERROR` - Unexpected server error

### Environment Variables
- `ENV` - Environment (development, staging, production)
- `LOG_LEVEL` - Logging level (debug, info, warn, error)
- `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`
- `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`
- `JWT_SECRET`, `JWT_EXPIRES_IN`
- `SERVER_HOST`, `SERVER_PORT`

---

*Используй этот документ как быстрый справочник для типичных задач разработки в проекте New Teams Up.*
