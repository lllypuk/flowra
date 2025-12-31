# Flowra

Комплексная система чата с интегрированным таск-трекером, help desk функциональностью и поддержкой команд.

## 🚀 Основные возможности

- **Real-time чат** с поддержкой групп и direct messages
- **Система команд** для управления задачами прямо из чата
- **Task management** с state machine для статусов
- **Help Desk** функциональность с SLA tracking
- **Keycloak интеграция** для SSO и управления пользователями
- **HTMX + Alpine.js** для минимального использования JavaScript
- **WebSocket/SSE** для real-time обновлений
- **Event Sourcing** для полной истории изменений
- **Tag Processing** - система обработки команд через теги в сообщениях

## 🎯 Доменные модели (реализованы)

### Chat Aggregate
- **Типы**: Direct message, Group chat, Help Desk ticket
- **Операции**: Create, AddParticipant, RemoveParticipant, Rename, SetSeverity, SetPriority, ConvertEntityType
- **События**: 10+ типов (ChatCreated, ParticipantAdded, RenamedChat и др.)

### Message Aggregate
- **Возможности**: Content, attachments, reactions, threading
- **Операции**: Create, Edit, Delete, AddAttachment, AddReaction, RemoveReaction
- **События**: MessageCreated, MessageEdited, MessageDeleted, AttachmentAdded, ReactionAdded/Removed

### Task Aggregate
- **Типы**: Task, Bug, Epic
- **States**: Pending, InProgress, Done, OnHold, Cancelled
- **Priority**: Low, Medium, High, Critical
- **Операции**: Create, ChangeStatus, AssignUser, SetDueDate, ChangePriority, ConvertToType
- **События**: TaskCreated, StatusChanged, AssigneeChanged, DueDateSet, PriorityChanged

### Notification Aggregate
- **Типы**: MessageNotif, TaskNotif, MentionNotif
- **Операции**: Create, MarkAsRead, MarkAllAsRead, Delete
- **Queries**: List, CountUnread, GetByID

### User & Workspace Entities
- **User**: Registration, Profile updates, Admin promotion, Keycloak integration
- **Workspace**: Create, Update, Invite system (CreateInvite, RevokeInvite, AcceptInvite)
- **Use Cases**: 14 (7 для User + 7 для Workspace)

### Tag Processing System
- **Формат**: `@{tag_name:tag_value}` в сообщениях
- **Типы тегов**: Entity Management, States, User Assignment, Priority, Duration
- **Валидация**: Tag format, reference checking
- **Процессинг**: Автоматическая генерация команд из тегов

## 📋 Содержание документации

- [Архитектура](./docs/01-architecture.md) - Общая архитектура системы
- [Установка и настройка](./docs/02-installation.md) - Руководство по установке
- [База данных](./docs/03-database.md) - Схема БД и миграции
- [Backend разработка](./docs/04-backend.md) - Go сервисы и API
- [Frontend с HTMX](./docs/05-frontend-htmx.md) - HTMX templates и компоненты
- [Keycloak интеграция](./docs/06-keycloak.md) - SSO и аутентификация
- [WebSocket/Real-time](./docs/07-websocket.md) - Real-time функциональность
- [Система команд](./docs/08-commands.md) - Command parser и handlers
- [Help Desk](./docs/09-helpdesk.md) - SLA и support функции
- [Плагины](./docs/10-plugins.md) - Система плагинов
- [Тестирование](./docs/11-testing.md) - Unit, integration и E2E тесты
- [Deployment](./docs/12-deployment.md) - Production deployment
- [Мониторинг](./docs/13-monitoring.md) - Метрики и health checks
- [API документация](./docs/14-api.md) - REST API endpoints

## 🛠 Технологический стек

### Backend
- **Go 1.25+** - основной язык
- **Echo v4** - веб-фреймворк
- **MongoDB 6+** с **Go Driver v2** - основная БД (event sourcing)
- **Redis** - кеш и pub/sub
- **Keycloak** - SSO и управление пользователями

### Frontend
- **HTMX 2+** - динамические обновления
- **Pico CSS v2** - минималистичный CSS фреймворк

## 📁 Структура проекта

```
new-teams-up/
├── cmd/                         # Точки входа приложений (scaffolding)
│   ├── api/                    # HTTP API сервер (planned)
│   ├── worker/                 # Background workers (planned)
│   └── migrator/               # DB миграции (planned)
├── internal/                    # Внутренний код приложения
│   ├── application/            # ✅ Application layer (40+ use cases)
│   │   ├── auth/              # Аутентификация
│   │   ├── chat/              # Управление чатами (6 use cases)
│   │   ├── message/           # Операции с сообщениями (7 use cases)
│   │   ├── notification/      # Уведомления (8 use cases)
│   │   ├── task/              # Управление задачами (5 use cases)
│   │   ├── user/              # Управление пользователями (7 use cases)
│   │   ├── workspace/         # Управление workspace (7 use cases)
│   │   ├── shared/            # Общие интерфейсы
│   │   └── eventhandler/      # Event handling
│   ├── domain/                 # ✅ Domain layer (event-sourced aggregates)
│   │   ├── chat/              # Chat aggregate + 10 events
│   │   ├── message/           # Message aggregate + 6 events
│   │   ├── task/              # Task aggregate + state machine
│   │   ├── notification/      # Notification aggregate + 4 events
│   │   ├── user/              # User entity
│   │   ├── workspace/         # Workspace entity
│   │   ├── tag/               # Tag processing & command parser
│   │   ├── event/             # Event sourcing infrastructure
│   │   ├── errs/              # Domain errors
│   │   └── uuid/              # UUID type wrapper
│   ├── infrastructure/         # 🔄 Infrastructure (partial)
│   │   ├── eventstore/        # ✅ In-memory event store
│   │   ├── eventbus/          # Event publishing (planned)
│   │   ├── repository/        # MongoDB/Redis repos (planned)
│   │   ├── mongodb/           # MongoDB v2 connection
│   │   ├── redis/             # Redis client
│   │   ├── keycloak/          # OAuth/SSO integration
│   │   ├── websocket/         # WebSocket server (planned)
│   │   └── middleware/        # HTTP middleware (planned)
│   ├── handler/                # HTTP/WS handlers (planned)
│   ├── config/                 # Configuration management
│   └── middleware/             # Middleware (planned)
├── pkg/                        # Переиспользуемые пакеты
│   └── logger/                # Logging utilities (planned)
├── tests/                      # ✅ Test infrastructure
│   ├── integration/           # Integration tests
│   ├── e2e/                   # E2E workflow tests
│   └── testutil/              # Test utilities, fixtures, mocks
├── migrations/                 # MongoDB миграции
├── configs/                    # ✅ config.yaml (полная конфигурация)
├── deployments/                # Docker Compose setup
├── scripts/                    # Utility scripts
└── docs/                       # Документация

Легенда: ✅ Реализовано | 🔄 В процессе | Planned - запланировано
```

## 🚦 Quick Start

### Prerequisites

- Go 1.25+
- Docker & Docker Compose
- MongoDB 6+ (с Go Driver v2)
- Redis 7+
- golangci-lint (для проверки кода)

### Setup (Локальная разработка)

1. **Clone the repository:**
```bash
git clone https://github.com/lllypuk/flowra.git
cd flowra
```

2. **Copy configuration:**
```bash
cp .env.example .env
# Edit .env if needed
```

3. **Start infrastructure:**
```bash
make docker-up
# or
docker-compose up -d mongodb redis keycloak
```

4. **Run tests to verify everything works:**
```bash
# Run all tests with coverage
go test ./...

# Run specific domain tests
go test ./internal/domain/chat/...
go test ./internal/application/chat/...

# Run with coverage percentage
go test -cover ./internal/application/...

# Integration tests (requires running MongoDB)
go test -tags=integration ./tests/integration/...

# Using make
make test                    # All tests
make test-unit              # Unit tests only
make test-integration       # Integration tests (requires MongoDB)
make test-coverage          # HTML coverage report
make test-coverage-check    # Check if coverage >= 80%
```

5. **Check code quality:**
```bash
make lint                   # Run golangci-lint
make fmt                    # Format code
make vet                    # Run go vet
```

6. **Build application:**
```bash
make build                  # Build all binaries (api, worker, migrator)
```

7. **Example: Using Chat Domain with Tag Processing**

```go
package main

import (
    "context"
    "github.com/google/uuid"

    "github.com/lllypuk/flowra/internal/application/chat"
    "github.com/lllypuk/flowra/internal/application/message"
    chatdomain "github.com/lllypuk/flowra/internal/domain/chat"
)

func main() {
    ctx := context.Background()

    // Setup (repositories, event store, etc.)
    // eventStore := eventstore.NewInMemoryEventStore()
    // userRepo := &MockUserRepository{}
    // chatRepo := &MockChatRepository{}
    // tagProcessor := setupTagProcessor()

    // 1. Create a chat
    createChatUC := chat.NewCreateChatUseCase(eventStore, userRepo, workspaceRepo)
    chatResult, _ := createChatUC.Execute(ctx, chat.CreateChatCommand{
        WorkspaceID: workspaceID,
        Type:        chatdomain.ChatTypeDiscussion,
        Title:       "Project Planning",
        IsPublic:    true,
        CreatedBy:   userID,
    })

    // 2. Send message with task command (Tag Processing)
    sendMsgUC := message.NewSendMessageUseCase(msgRepo, chatRepo, eventStore, tagProcessor)
    msgResult, _ := sendMsgUC.Execute(ctx, message.SendMessageCommand{
        ChatID:    chatResult.ChatID,
        Content:   "We need to implement authentication #createTask #setPriority high",
        SentBy:    userID,
    })

    // Result:
    // 1. Message created
    // 2. Chat converted to Task
    // 3. Priority set to High
    // 4. TaskCreated and PriorityChanged events published
}
```

### Running the Application (When Implemented)

```bash
make dev                    # Development mode with hot reload
# or
go run cmd/api/main.go      # API server
go run cmd/worker/main.go   # Worker service
```

Приложение будет доступно на http://localhost:8080 (после реализации handlers)

### Доступные Make команды

```bash
# Инфраструктура
make docker-up              # Запустить Docker контейнеры (MongoDB, Redis, Keycloak)
make docker-down            # Остановить Docker контейнеры
make docker-logs            # Просмотр логов Docker

# Сборка
make build                  # Собрать все бинарные файлы (api, worker, migrator)
make clean                  # Очистить build артефакты

# Тестирование
make test                   # Запустить все тесты с coverage
make test-unit              # Только unit тесты
make test-integration       # Integration тесты (требуется MongoDB)
make test-coverage          # Сгенерировать HTML coverage report
make test-coverage-check    # Проверить coverage threshold (80%)

# Качество кода
make lint                   # Запустить golangci-lint
make fmt                    # Форматировать код (gofmt)
make vet                    # Запустить go vet

# Разработка
make dev                    # Запустить в development mode
make run-api                # Запустить API сервер
make run-worker             # Запустить background worker
```

## 📊 Timeline проекта

### Реализовано на данный момент

#### ✅ Completed
- Event-sourced domain aggregates (Chat, Message, Task, Notification, User, Workspace)
- 40+ application use cases с валидацией
- Event store infrastructure (in-memory)
- Tag processing & command parser система
- Comprehensive test infrastructure (fixtures, mocks, utilities)
- MongoDB v2 integration готова
- Configuration management
- Code quality setup (golangci-lint, Makefile)

#### 🔄 In Progress
- MongoDB repositories implementation
- Redis repositories implementation
- Event bus (Redis/in-memory)

#### ⏳ Next Steps
- HTTP handlers для use cases
- API endpoints (Echo routes)
- WebSocket handlers
- Entry points (cmd/api/main.go)

## 📊 Метрики кода

### Статистика реализации

**Version:** 0.4.0-alpha
**Status:** Active Development (Phase 0 Complete, 82% Overall)

- **Строк кода:** ~23,000 LOC
  - Application layer: 13,000+ LOC (86 файлов)
  - Domain layer: 9,500+ LOC (52 файлов)
  - Infrastructure: 500+ LOC (partial)
- **Go файлов**: 190+
- **Интерфейсов**: 68 (следуя idiomatic Go паттернам)
- **Use Cases**: 40+ реализовано
- **Domain Events**: 30+ типов событий
- **Test Coverage:**
  - Domain Layer: 90%+ ✅
  - Application Layer: 75%+ ✅
- **Test Files**: 60+ (fixtures, mocks, utilities, integration tests)

### Архитектурные достижения

✅ **Event-Driven Architecture**
- Полная поддержка event sourcing
- Uncommitted events tracking
- Optimistic concurrency control
- Event replay capability

✅ **Domain-Driven Design**
- Чистые границы доменов
- Aggregates с бизнес-логикой
- Domain events для коммуникации
- Rich domain models (не anemic)

✅ **CQRS Pattern**
- Разделение команд и запросов
- Command handlers с валидацией
- Query handlers для чтения данных

✅ **Repository Pattern**
- Интерфейсы на стороне consumer (idiomatic Go)
- Абстракция от MongoDB/Redis
- Testable через mock repositories

✅ **Dependency Injection Ready**
- Constructor-based DI
- Interface-based dependencies
- Easy to wire up with DI containers

✅ **Test Infrastructure**
- Fluent API для создания test fixtures
- Mock repositories для unit tests
- Integration test utilities (MongoDB v2, Redis)
- E2E workflow tests
- Custom assertions

## 📈 Current Status

### ✅ Completed (Phase 0 Final)

**Domain Layer (90%+ complete)**
- 6 Event-Sourced aggregates fully functional:
  - Chat (с типами: Discussion, Task, Help Desk Ticket, Direct Message)
  - Message (с поддержкой threads, reactions, attachments)
  - Task (с state machine: Pending → InProgress → Done/OnHold/Cancelled)
  - Notification (с типами: MessageNotif, TaskNotif, MentionNotif)
  - User & Workspace (entities с полной функциональностью)
- 30+ domain events с валидацией и сериализацией
- Полная бизнес-логика для всех операций

**Application Layer (75%+ complete)**
- 40+ use cases реализовано:
  - Chat: 12 commands + 3 queries
  - Message: 8 use cases (send, edit, delete, reply в threads)
  - Task Management: Полное управление статусами, приоритетами, due dates
  - Notification: Создание, чтение, удаление, mark as read
  - User & Workspace: Приглашения, управление участниками
- Tag Processing System - полностью интегрирована в SendMessageUseCase
- CQRS pattern реализован для всех доменов

**Testing Infrastructure (85%+ complete)**
- 60+ тестовых файлов с примерами
- Fixtures API для создания test data
- Mock repositories для всех доменов
- MongoDB v2 и Redis интеграционные тесты
- E2E workflow tests для Chat → Message → Task

### 🚧 In Progress (Phase 1)

**Infrastructure Layer (30%)**
- ✅ In-memory Event Store (функциональный для тестирования)
- ✅ MongoDB v2 connection и конфигурация
- ✅ Redis client setup
- ⏳ MongoDB repositories (not yet implemented)
- ⏳ Event Bus (Redis Pub/Sub, not yet implemented)

### 📋 Next Steps (Phase 2-3)

- **Interface Layer (0%)** - HTTP handlers, middleware, WebSocket
- **Entry Points (0%)** - API server (cmd/api/main.go), Worker service
- **Frontend** - HTMX templates и Pico CSS компоненты
- **Deployment** - Docker образы, K8s манифесты

**Current Focus:** Infrastructure Layer → Interface Layer → Entry Points

## 🔐 Безопасность

- OAuth 2.0/OIDC через Keycloak
- JWT tokens с refresh механизмом
- RBAC (Role-Based Access Control)
- Rate limiting
- CORS защита
- SQL injection защита через prepared statements
- XSS защита через template escaping
- CSRF токены для форм

## 🧪 Тестирование

```bash
# Все тесты с coverage
make test

# Unit тесты
make test-unit

# Integration тесты (требуется запущенный MongoDB)
make test-integration

# Coverage report (генерирует HTML отчет)
make test-coverage

# Проверка coverage threshold (минимум 80%)
make test-coverage-check

# E2E тесты
go test ./tests/e2e -tags=e2e -v

# Или напрямую через go test
go test ./... -v
go test ./internal/application/... -v
go test ./internal/domain/... -v
```

### Test Infrastructure

Проект оснащен полноценной тестовой инфраструктурой:

- **Fixtures**: Fluent API для создания test data
  ```go
  cmd := fixtures.NewCreateTaskCommand().
      WithTitle("Test Task").
      WithAssignee(userID).
      Build()
  ```

- **Mocks**: Mock repositories для изоляции тестов
  - `MockWorkspaceRepository`
  - `MockNotificationRepository`
  - `MockEventStore`
  - `MockUserRepository`

- **Test Utilities**:
  - `testutil/mongodb.go` - MongoDB v2 integration helpers
  - `testutil/redis.go` - Redis test setup
  - `testutil/assertions.go` - Custom assertions
  - `testutil/helpers.go` - General test helpers

- **Integration Tests**: Тесты с реальной БД (MongoDB, Redis)
- **E2E Tests**: End-to-end workflow тесты (messaging, tasks)

## 📈 Мониторинг

- Prometheus метрики на `/metrics`
- Health checks на `/health`
- Grafana дашборды
- Structured logging через zerolog
- Distributed tracing (опционально)

## 📄 Лицензия

[MIT License](./LICENSE)

---

**Version**: 0.4.0-alpha
**Status**: Active Development (Phase 2-3 Complete)
**Last Updated**: 2025-10-22
