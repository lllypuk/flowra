# Task 01: Project Initialization (Phase 0)

**Фаза:** 0 - Project Setup
**Приоритет:** Critical
**Статус:** ✅ **COMPLETED**
**Дата создания:** 2025-10-04
**Дата завершения:** 2025-10-04

## Цель

Подготовить полноценное окружение разработки с правильной структурой проекта, инфраструктурными сервисами и
инструментами для разработки. После выполнения этой задачи проект должен быть готов к началу разработки доменной
модели (Phase 1).

---

## Подзадачи

### 0.1 Initialize Go Module

**Описание:** Инициализировать Go модуль и добавить базовые зависимости.

**Шаги:**

1. Инициализировать Go модуль:
   ```bash
   go mod init github.com/lllypuk/flowra
   ```

2. Добавить основные зависимости:
   ```bash
   # Web framework
   go get github.com/labstack/echo/v4
   go get github.com/labstack/echo/v4/middleware

   # MongoDB driver
   go get go.mongodb.org/mongo-driver/mongo
   go get go.mongodb.org/mongo-driver/bson

   # Redis client
   go get github.com/redis/go-redis/v9

   # UUID generation
   go get github.com/google/uuid

   # Configuration management
   go get github.com/spf13/viper

   # Logging
   # (standard log/slog в stdlib)

   # Testing
   go get github.com/stretchr/testify
   ```

3. Создать `go.mod` и `go.sum` файлы
4. Зафиксировать версии зависимостей: `go mod tidy`

**Критерии выполнения:**

- [x] `go.mod` содержит правильный module path ✅
- [x] Все базовые зависимости добавлены ✅
- [x] `go mod verify` проходит успешно ✅
- [x] Проект компилируется: `go build ./...` ✅

---

### 0.2 Setup Project Structure

**Описание:** Создать директории согласно архитектуре из `docs/07-code-structure.md` и MVP roadmap.

**Шаги:**

1. Создать основную структуру директорий:
   ```bash
   # Command entry points
   mkdir -p cmd/api
   mkdir -p cmd/worker
   mkdir -p cmd/migrator

   # Domain layer (Phase 1)
   mkdir -p internal/domain/event
   mkdir -p internal/domain/common
   mkdir -p internal/domain/user
   mkdir -p internal/domain/workspace
   mkdir -p internal/domain/chat
   mkdir -p internal/domain/task
   mkdir -p internal/domain/notification

   # Application layer (Phase 2)
   mkdir -p internal/application/chat
   mkdir -p internal/application/workspace
   mkdir -p internal/application/task
   mkdir -p internal/application/auth
   mkdir -p internal/application/eventhandler

   # Infrastructure layer (Phase 3)
   mkdir -p internal/infrastructure/mongodb
   mkdir -p internal/infrastructure/eventstore
   mkdir -p internal/infrastructure/eventbus
   mkdir -p internal/infrastructure/repository/mongodb
   mkdir -p internal/infrastructure/repository/redis
   mkdir -p internal/infrastructure/redis
   mkdir -p internal/infrastructure/keycloak
   mkdir -p internal/infrastructure/websocket

   # Interface layer (Phase 4)
   mkdir -p internal/handler/http
   mkdir -p internal/handler/websocket
   mkdir -p internal/middleware

   # Configuration and utilities
   mkdir -p internal/config
   mkdir -p pkg/logger

   # Frontend
   mkdir -p web/templates/layout
   mkdir -p web/templates/auth
   mkdir -p web/templates/workspace
   mkdir -p web/templates/board
   mkdir -p web/templates/chat
   mkdir -p web/templates/task
   mkdir -p web/components
   mkdir -p web/static/css
   mkdir -p web/static/js

   # Tests
   mkdir -p tests/integration
   mkdir -p tests/e2e
   mkdir -p tests/fixtures
   mkdir -p tests/testutil

   # Migrations
   mkdir -p migrations/mongodb

   # Configs
   mkdir -p configs

   # Scripts
   mkdir -p scripts
   ```

2. Создать `.gitkeep` файлы в пустых директориях (чтобы Git их отслеживал):
   ```bash
   find . -type d -empty -exec touch {}/.gitkeep \;
   ```

3. Создать базовый `.gitignore`:
   ```gitignore
   # Binaries
   /bin/
   *.exe
   *.exe~
   *.dll
   *.so
   *.dylib

   # Test binary, built with `go test -c`
   *.test

   # Output of the go coverage tool
   *.out
   coverage.html

   # Dependency directories
   vendor/

   # Go workspace file
   go.work

   # Environment files
   .env
   .env.local
   *.local.yaml

   # IDE
   .idea/
   .vscode/
   *.swp
   *.swo
   *~

   # OS
   .DS_Store
   Thumbs.db

   # Temporary files
   tmp/
   temp/

   # Logs
   *.log
   logs/
   ```

**Критерии выполнения:**

- [x] Все директории созданы согласно структуре ✅
- [x] `.gitignore` настроен ✅
- [x] Структура соответствует `docs/07-code-structure.md` ✅

---

### 0.3 Configure Development Environment

**Описание:** Настроить Docker Compose для локальной разработки и создать базовые конфигурационные файлы.

**Шаги:**

1. Создать `docker-compose.yml`:
   ```yaml
   version: '3.8'

   services:
     mongodb:
       image: mongo:7
       container_name: flowra-mongodb
       ports:
         - "27017:27017"
       environment:
         MONGO_INITDB_ROOT_USERNAME: admin
         MONGO_INITDB_ROOT_PASSWORD: admin123
       volumes:
         - mongodb_data:/data/db
       networks:
         - flowra-network

     redis:
       image: redis:7-alpine
       container_name: flowra-redis
       ports:
         - "6379:6379"
       volumes:
         - redis_data:/data
       networks:
         - flowra-network

     keycloak:
       image: quay.io/keycloak/keycloak:23.0
       container_name: flowra-keycloak
       environment:
         KEYCLOAK_ADMIN: admin
         KEYCLOAK_ADMIN_PASSWORD: admin123
         KC_DB: dev-file
       ports:
         - "8090:8080"
       command:
         - start-dev
       networks:
         - flowra-network

   volumes:
     mongodb_data:
     redis_data:

   networks:
     flowra-network:
       driver: bridge
   ```

2. Создать `configs/config.yaml`:
   ```yaml
   server:
     host: "0.0.0.0"
     port: 8080
     read_timeout: 30s
     write_timeout: 30s
     shutdown_timeout: 10s

   mongodb:
     uri: "mongodb://admin:admin123@localhost:27017"
     database: "flowra"
     timeout: 10s
     max_pool_size: 100

   redis:
     addr: "localhost:6379"
     password: ""
     db: 0
     pool_size: 10

   keycloak:
     url: "http://localhost:8090"
     realm: "flowra"
     client_id: "flowra-backend"
     client_secret: "your-client-secret"
     admin_username: "admin"
     admin_password: "admin123"

   auth:
     jwt_secret: "dev-secret-change-in-production"
     access_token_ttl: 15m
     refresh_token_ttl: 7d

   eventbus:
     type: "redis" # redis | inmemory
     redis_channel_prefix: "events."

   log:
     level: "debug" # debug | info | warn | error
     format: "json" # json | text

   websocket:
     read_buffer_size: 1024
     write_buffer_size: 1024
     ping_interval: 30s
     pong_timeout: 60s
   ```

3. Создать `configs/config.dev.yaml` (для переопределения значений при разработке):
   ```yaml
   log:
     level: "debug"
     format: "text"
   ```

4. Создать `configs/config.prod.yaml` (для продакшна):
   ```yaml
   log:
     level: "info"
     format: "json"

   auth:
     jwt_secret: "${JWT_SECRET}" # из переменной окружения

   keycloak:
     client_secret: "${KEYCLOAK_CLIENT_SECRET}"
   ```

5. Создать `Makefile` для удобства:
   ```makefile
   .PHONY: help dev build test lint docker-up docker-down clean

   help: ## Show this help
   	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

   dev: ## Run in development mode
   	go run cmd/api/main.go

   build: ## Build binaries
   	go build -o bin/api cmd/api/main.go
   	go build -o bin/worker cmd/worker/main.go
   	go build -o bin/migrator cmd/migrator/main.go

   test: ## Run tests
   	go test -v -race -coverprofile=coverage.out ./...

   test-integration: ## Run integration tests
   	go test -v -race -tags=integration ./tests/integration/...

   test-coverage: test ## Generate coverage report
   	go tool cover -html=coverage.out -o coverage.html

   lint: ## Run linters
   	golangci-lint run

   docker-up: ## Start Docker services
   	docker-compose up -d

   docker-down: ## Stop Docker services
   	docker-compose down

   docker-logs: ## Show Docker logs
   	docker-compose logs -f

   migrate-up: ## Run migrations up
   	go run cmd/migrator/main.go up

   migrate-down: ## Run migrations down
   	go run cmd/migrator/main.go down

   clean: ## Clean build artifacts
   	rm -rf bin/
   	rm -f coverage.out coverage.html

   deps: ## Download dependencies
   	go mod download
   	go mod tidy

   .DEFAULT_GOAL := help
   ```

6. Создать `.env.example`:
   ```bash
   # Server
   SERVER_PORT=8080

   # MongoDB
   MONGODB_URI=mongodb://admin:admin123@localhost:27017
   MONGODB_DATABASE=flowra

   # Redis
   REDIS_ADDR=localhost:6379

   # Keycloak
   KEYCLOAK_URL=http://localhost:8090
   KEYCLOAK_REALM=flowra
   KEYCLOAK_CLIENT_ID=flowra-backend
   KEYCLOAK_CLIENT_SECRET=your-client-secret

   # Auth
   JWT_SECRET=dev-secret-change-in-production

   # Logging
   LOG_LEVEL=debug
   ```

**Критерии выполнения:**

- [x] `docker-compose.yml` создан и корректен ✅
- [x] `make docker-up` запускает MongoDB, Redis, Keycloak ✅
- [x] Все сервисы доступны на указанных портах ✅
- [x] Конфигурационные файлы созданы ✅
- [x] Makefile работает, все команды выполняются ✅
- [x] `.env.example` содержит все необходимые переменные ✅

---

### 0.4 Setup Linting and Formatting

**Описание:** Настроить golangci-lint для проверки качества кода.

**Шаги:**

1. Установить golangci-lint (если не установлен):
   ```bash
   # macOS/Linux
   brew install golangci-lint

   # или
   curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
   ```

2. Создать `.golangci.yml`:
   ```yaml
   run:
     timeout: 5m
     tests: true
     modules-download-mode: readonly

   linters:
     enable:
       - errcheck       # проверка ошибок
       - gosimple       # упрощение кода
       - govet          # go vet
       - ineffassign    # неиспользуемые присваивания
       - staticcheck    # статический анализ
       - unused         # неиспользуемый код
       - gofmt          # форматирование
       - goimports      # импорты
       - misspell       # опечатки
       - unconvert      # ненужные конвертации
       - unparam        # неиспользуемые параметры
       - gosec          # security issues
       - bodyclose      # закрытие HTTP body
       - noctx          # HTTP запросы без context
       - errname        # именование error переменных
       - errorlint      # ошибки с wrap
       - gocritic       # множество проверок
       - revive         # замена golint

   linters-settings:
     errcheck:
       check-type-assertions: true
       check-blank: true

     govet:
       check-shadowing: true

     gofmt:
       simplify: true

     goimports:
       local-prefixes: github.com/yourorg/flowra

     gocritic:
       enabled-tags:
         - diagnostic
         - style
         - performance
       disabled-checks:
         - ifElseChain
         - hugeParam

     revive:
       rules:
         - name: exported
           severity: warning
         - name: package-comments
           severity: warning

   issues:
     exclude-use-default: false
     max-issues-per-linter: 0
     max-same-issues: 0

   output:
     sort-results: true
   ```

3. (Опционально) Создать pre-commit hook в `.git/hooks/pre-commit`:
   ```bash
   #!/bin/sh

   # Run linters
   make lint

   # Run tests
   make test

   # If any fail, prevent commit
   if [ $? -ne 0 ]; then
       echo "Linting or tests failed. Commit aborted."
       exit 1
   fi
   ```

   Сделать исполняемым: `chmod +x .git/hooks/pre-commit`

4. Добавить команды форматирования в Makefile:
   ```makefile
   fmt: ## Format code
   	gofmt -s -w .
   	goimports -w -local github.com/yourorg/flowra .

   fmt-check: ## Check formatting
   	test -z $(shell gofmt -l .)
   ```

**Критерии выполнения:**

- [x] `.golangci.yml` создан ✅
- [x] `golangci-lint run` выполняется без ошибок ✅
- [x] `make lint` работает ✅
- [x] `make fmt` форматирует код ✅
- [ ] (Опционально) Pre-commit hook настроен

---

### 0.5 Initialize Testing Framework

**Описание:** Настроить testify и создать helpers для тестов.

**Шаги:**

1. Убедиться, что testify установлен (добавлено в 0.1):
   ```bash
   go get github.com/stretchr/testify
   ```

2. Создать `tests/testutil/helpers.go`:
   ```go
   package testutil

   import (
       "context"
       "testing"
       "time"

       "github.com/stretchr/testify/require"
   )

   // NewTestContext создает context с таймаутом для тестов
   func NewTestContext(t *testing.T) context.Context {
       ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
       t.Cleanup(cancel)
       return ctx
   }

   // AssertNoError проверяет отсутствие ошибки и останавливает тест
   func AssertNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
       t.Helper()
       require.NoError(t, err, msgAndArgs...)
   }
   ```

3. Создать `tests/testutil/mongodb.go` (для integration tests):
   ```go
   package testutil

   import (
       "context"
       "testing"
       "time"

       "go.mongodb.org/mongo-driver/mongo"
       "go.mongodb.org/mongo-driver/mongo/options"
   )

   // SetupTestMongoDB создает подключение к тестовой MongoDB
   // Использует testcontainers или docker-compose
   func SetupTestMongoDB(t *testing.T) *mongo.Database {
       ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
       defer cancel()

       // Для интеграционных тестов используем отдельную БД
       uri := "mongodb://admin:admin123@localhost:27017"
       client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
       if err != nil {
           t.Fatalf("Failed to connect to MongoDB: %v", err)
       }

       // Проверка соединения
       err = client.Ping(ctx, nil)
       if err != nil {
           t.Fatalf("Failed to ping MongoDB: %v", err)
       }

       // Создаем тестовую БД с уникальным именем
       dbName := "flowra_test_" + t.Name()
       db := client.Database(dbName)

       // Cleanup: удаляем БД после теста
       t.Cleanup(func() {
           ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
           defer cancel()
           _ = db.Drop(ctx)
           _ = client.Disconnect(ctx)
       })

       return db
   }
   ```

4. Создать `tests/testutil/redis.go`:
   ```go
   package testutil

   import (
       "context"
       "testing"

       "github.com/redis/go-redis/v9"
   )

   // SetupTestRedis создает подключение к тестовому Redis
   func SetupTestRedis(t *testing.T) *redis.Client {
       client := redis.NewClient(&redis.Options{
           Addr: "localhost:6379",
           DB:   15, // используем отдельную БД для тестов
       })

       ctx := context.Background()
       err := client.Ping(ctx).Err()
       if err != nil {
           t.Fatalf("Failed to connect to Redis: %v", err)
       }

       // Cleanup: очищаем БД после теста
       t.Cleanup(func() {
           _ = client.FlushDB(ctx).Err()
           _ = client.Close()
       })

       return client
   }
   ```

5. Создать пример теста `tests/example_test.go`:
   ```go
   package tests

   import (
       "testing"

       "github.com/stretchr/testify/assert"
       "github.com/stretchr/testify/require"
   )

   func TestExample(t *testing.T) {
       // Arrange
       input := 2

       // Act
       result := input + 2

       // Assert
       assert.Equal(t, 4, result)
       require.NotZero(t, result)
   }
   ```

6. Добавить в Makefile команды для разных типов тестов:
   ```makefile
   test-unit: ## Run unit tests only
   	go test -v -race -short ./internal/...

   test-integration: ## Run integration tests
   	go test -v -race -tags=integration ./tests/integration/...

   test-e2e: ## Run e2e tests
   	go test -v -tags=e2e ./tests/e2e/...
   ```

**Критерии выполнения:**

- [x] testify установлен ✅
- [x] Test helpers созданы в `tests/testutil/` ✅
- [x] Пример теста проходит: `go test ./tests/` ✅
- [x] `make test` выполняется успешно ✅
- [x] Helpers для MongoDB и Redis готовы к использованию ✅

---

## Deliverable

После выполнения всех подзадач должно быть готово:

✅ **Проект с правильной структурой директорий**

- Все директории созданы согласно архитектуре
- `.gitignore` настроен

✅ **Go модуль инициализирован**

- `go.mod` с правильными зависимостями
- Проект компилируется без ошибок

✅ **Docker services запускаются**

- MongoDB доступна на порту 27017
- Redis доступен на порту 6379
- Keycloak доступен на порту 8090
- `make docker-up` работает

✅ **Конфигурация готова**

- `configs/config.yaml` с базовыми настройками
- Environment-specific конфиги (dev, prod)
- `.env.example` для разработчиков

✅ **Инструменты разработки настроены**

- `golangci-lint` работает
- `Makefile` с полезными командами
- Testing framework готов

✅ **Можно начинать разработку**

- `make dev` запускает проект (когда будет реализован cmd/api/main.go)
- `make test` запускает тесты
- `make lint` проверяет качество кода

---

## Проверка выполнения

Выполните следующие команды для проверки:

```bash
# 1. Проверка структуры
ls -la internal/domain
ls -la cmd

# 2. Проверка Go модуля
go mod verify
go build ./...

# 3. Запуск Docker services
make docker-up
docker ps  # должны быть видны mongodb, redis, keycloak

# 4. Проверка доступности сервисов
# MongoDB
mongosh "mongodb://admin:admin123@localhost:27017" --eval "db.runCommand({ ping: 1 })"

# Redis
redis-cli ping

# Keycloak
curl http://localhost:8090

# 5. Проверка линтинга
make lint

# 6. Проверка тестов
make test

# 7. Проверка Makefile
make help
```

Все команды должны выполняться успешно.

---

## Следующие шаги

После завершения Phase 0 переходим к **Phase 1: Domain Layer — Core Aggregates**:

- Начинаем с задачи **1.1 Base Domain Infrastructure**
- Создаем интерфейсы для domain events
- Реализуем первые value objects

См. `docs/08-mvp-roadmap.md` Phase 1 для деталей.

---

## Примечания

- Эта задача не содержит бизнес-логики, только инфраструктуру
- Все пути и названия должны точно соответствовать roadmap
- Docker services должны быть доступны для следующих фаз
- Конфигурация будет расширяться по мере добавления фич
- Версии зависимостей зафиксированы в `go.mod`

**Важно:** Не создавайте файлы с реализацией (`main.go`, handlers, repositories) на этом этапе. Phase 0 — только
структура и инструменты.

---

## ✅ Результаты выполнения

**Дата завершения:** 2025-10-04

### Выполненные компоненты:

1. **Go Module** - `github.com/lllypuk/flowra`
   - Echo v4.13.4 (web framework)
   - MongoDB driver v1.17.4
   - Redis v9.14.0
   - UUID v1.6.0
   - Viper v1.21.0 (configuration)
   - Testify v1.11.1 (testing)

2. **Структура проекта** - все директории созданы:
   - `cmd/` - entry points (api, worker, migrator)
   - `internal/domain/` - domain layer (7 aggregates)
   - `internal/application/` - application services (5 modules)
   - `internal/infrastructure/` - infrastructure implementations
   - `tests/` - test infrastructure с helpers

3. **Конфигурация**:
   - `docker-compose.yml` - MongoDB 7, Redis 7, Keycloak 23
   - `configs/config.yaml` - базовая конфигурация
   - `configs/config.dev.yaml` - development overrides
   - `configs/config.prod.yaml` - production settings
   - `.env.example` - environment template

4. **Инструменты разработки**:
   - `Makefile` - 15+ команд (build, test, lint, docker, fmt)
   - `.golangci.yml` - configured для github.com/lllypuk/flowra
   - `.gitignore` - полный набор правил

5. **Testing Infrastructure**:
   - `tests/testutil/helpers.go` - test context и assertions
   - `tests/testutil/mongodb.go` - MongoDB test setup
   - `tests/testutil/redis.go` - Redis test setup
   - `tests/example_test.go` - работающий пример теста

### Проверки пройдены:
- ✅ `go mod verify` - all modules verified
- ✅ `go build ./...` - compilation successful
- ✅ `go test ./tests/` - tests passed
- ✅ `make help` - 15+ commands available
- ✅ Directory structure matches roadmap

**Статус:** Готово к Phase 1 (Domain Layer)
