# Flowra

Комплексная система чата с интегрированным таск-трекером, help desk функциональностью и поддержкой команд.

## 📊 Текущее состояние проекта

**Версия:** 0.4.0-alpha  
**Дата обновления:** 2024-12-31  
**Общий прогресс:** ~62% к MVP  
**Статус:** Active Development (Phase 1-2)

### Прогресс по слоям

| Слой | Статус | Прогресс | Файлов | Coverage |
|------|--------|----------|--------|----------|
| **Domain** | ✅ Complete | 95% | 48 | 90%+ |
| **Application** | ✅ Strong | 85% | 139 | 79% |
| **Infrastructure** | ⚠️ In Progress | 45% | 21 | 85%+ |
| **Interface** | ❌ Not Started | 0% | 0 | N/A |
| **Entry Points** | ❌ Not Started | 0% | 0 | N/A |

### Что работает ✅

- ✅ **Domain Layer:** 6 Event-Sourced агрегатов, 30+ domain events
- ✅ **Application Layer:** 40+ use cases с 79% average coverage
- ✅ **MongoDB Repositories:** Chat, User, Workspace, Message, Notification (5 из 6)
- ✅ **Event Store:** MongoDB Event Store с optimistic locking
- ✅ **Testing Infrastructure:** testcontainers-go, mocks, fixtures

### Что требуется ❌

- 🔴 **Task Repository** (последний недостающий репозиторий)
- 🔴 **MongoDB Indexes** (критично для production)
- 🔴 **Interface Layer** (HTTP handlers, WebSocket)
- 🔴 **Entry Points** (cmd/api/main.go отсутствует)
- 🟡 **Event Bus** (Redis Pub/Sub)
- 🟡 **Frontend** (HTMX templates)

### Следующие шаги

См. детальный план: [docs/JANUARY_2025_PLAN.md](./docs/JANUARY_2025_PLAN.md)

**ETA к MVP:** Середина февраля 2025 (6-8 недель)

**Документация:**
- [Текущий статус](./docs/STATUS.md) - живой статус проекта
- [Roadmap 2025](./docs/DEVELOPMENT_ROADMAP_2025.md) - детальный план развития
- [План на январь 2025](./docs/JANUARY_2025_PLAN.md) - недельный breakdown

---

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

## 📋 Документация

### Статус и планирование
- [Текущий статус](./docs/STATUS.md) - живой статус проекта (обновлен 2025-12-31)
- [Roadmap 2025](./docs/DEVELOPMENT_ROADMAP_2025.md) - детальный план на 6 месяцев
- [План на январь 2025](./docs/JANUARY_2025_PLAN.md) - недельный breakdown задач
- [Архитектурные изменения](./docs/ARCHITECTURE_FIX.md) - миграция интерфейсов
- [Лог рефакторинга](./docs/REFACTORING_LOG.md) - история изменений

### Архитектура и дизайн
- [01-architecture.md](./docs/01-architecture.md) - общая архитектура системы
- [02-domain-model.md](./docs/02-domain-model.md) - доменная модель
- [03-tag-grammar.md](./docs/03-tag-grammar.md) - грамматика команд через теги
- [04-security-model.md](./docs/04-security-model.md) - модель безопасности
- [05-event-flow.md](./docs/05-event-flow.md) - потоки событий
- [06-api-contracts.md](./docs/06-api-contracts.md) - API контракты
- [07-code-structure.md](./docs/07-code-structure.md) - структура кода
- [08-mvp-roadmap.md](./docs/08-mvp-roadmap.md) - MVP roadmap

### Разработка
- [development/setup.md](./docs/development/setup.md) - настройка окружения
- [development/coding-standards.md](./docs/development/coding-standards.md) - стандарты кода
- [development/testing.md](./docs/development/testing.md) - тестирование

## 🛠 Технологический стек

### Backend
- **Go 1.25+** - основной язык
- **Echo v4.13+** - веб-фреймворк
- **MongoDB 6+** с **Go Driver v2** - основная БД (event sourcing + read models)
- **Redis 7+** - кеш, pub/sub, session store
- **Keycloak 23+** - SSO и управление пользователями

### Frontend
- **HTMX 2+** - динамические обновления без JavaScript
- **Pico CSS v2** - минималистичный CSS фреймворк
- **Alpine.js** (опционально) - минимальный JS для интерактивности

### Development & Testing
- **testcontainers-go** - интеграционное тестирование
- **testify** - assertions и mocks
- **golangci-lint** - комплексный линтинг

## 📁 Структура проекта

```
new-teams-up/
├── cmd/                         # Точки входа приложений
│   ├── api/                    # ❌ HTTP API сервер (требуется)
│   ├── worker/                 # ❌ Background workers (требуется)
│   └── migrator/               # ❌ DB миграции (требуется)
├── internal/                    # Внутренний код приложения
│   ├── application/            # ✅ Application layer (40+ use cases, 79% coverage)
│   │   ├── appcore/           # Shared interfaces (EventStore, EventBus)
│   │   ├── auth/              # Аутентификация
│   │   ├── chat/              # Управление чатами (15 use cases, 81% coverage)
│   │   ├── message/           # Операции с сообщениями (7 use cases, 64% coverage)
│   │   ├── notification/      # Уведомления (8 use cases, 85% coverage)
│   │   ├── task/              # Управление задачами (5 use cases, 85% coverage)
│   │   ├── user/              # Управление пользователями (7 use cases, 86% coverage)
│   │   ├── workspace/         # Управление workspace (7 use cases, 86% coverage)
│   │   └── eventhandler/      # Event handling (planned)
│   ├── domain/                 # ✅ Domain layer (95% complete, 90%+ coverage)
│   │   ├── chat/              # Chat aggregate (Event Sourcing)
│   │   ├── message/           # Message aggregate
│   │   ├── task/              # Task aggregate (Event Sourcing)
│   │   ├── notification/      # Notification aggregate
│   │   ├── user/              # User entity
│   │   ├── workspace/         # Workspace entity
│   │   ├── tag/               # Tag processing system
│   │   ├── event/             # Domain events infrastructure
│   │   ├── errs/              # Domain errors
│   │   └── uuid/              # UUID utilities
│   ├── infrastructure/         # ⚠️ Infrastructure layer (45% complete)
│   │   ├── eventstore/        # ✅ MongoDB Event Store (production ready)
│   │   ├── mongodb/           # ✅ MongoDB connection setup
│   │   ├── repository/        # ⚠️ MongoDB repositories (5 из 6 готовы)
│   │   │   └── mongodb/       # Chat, User, Workspace, Message, Notification
│   │   ├── redis/             # ✅ Redis client setup
│   │   ├── eventbus/          # ❌ Event Bus (требуется)
│   │   ├── keycloak/          # ❌ Keycloak client (требуется)
│   │   └── websocket/         # ❌ WebSocket server (требуется)
│   ├── handler/                # ❌ Interface layer (требуется)
│   │   ├── http/              # HTTP handlers (planned)
│   │   └── websocket/         # WebSocket handlers (planned)
│   ├── middleware/             # ❌ Middleware (требуется)
│   └── config/                 # ✅ Configuration management
├── tests/                       # ✅ Testing infrastructure (90% complete)
│   ├── testutil/              # MongoDB/Redis test helpers
│   ├── mocks/                 # Generated mocks
│   └── fixtures/              # Test data
├── web/                         # ❌ Frontend (требуется)
│   ├── templates/             # HTMX templates (planned)
│   └── static/                # CSS, JS (planned)
├── configs/                     # ✅ Configuration files
│   ├── config.yaml            # Main config
│   ├── config.dev.yaml        # Development overrides
│   └── config.prod.yaml       # Production overrides
├── docs/                        # ✅ Documentation (обновлена 2024-12-31)
│   ├── STATUS.md              # Текущий статус проекта
│   ├── DEVELOPMENT_ROADMAP_2025.md  # Детальный roadmap
│   ├── JANUARY_2025_PLAN.md   # План на январь
│   ├── roadmap/               # Фазы разработки по неделям
│   └── tasks/                 # Детальные задачи
└── docker-compose.yml          # ✅ Development infrastructure
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
