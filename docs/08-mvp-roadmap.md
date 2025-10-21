# MVP Roadmap — Implementation Plan

**Дата:** 2025-09-30
**Статус:** План разработки

## Обзор

Roadmap для реализации MVP chat-based task tracker. Разработка следует принципам **Domain-Driven Design**: сначала доменная модель и бизнес-логика, затем инфраструктура. Все компоненты используют **интерфейсы** для слабой связанности и тестируемости.

## Принципы реализации

1. **Domain-first approach** — начинаем с доменной модели, независимой от инфраструктуры
2. **Interface-driven design** — все зависимости через интерфейсы
3. **TDD где возможно** — unit-тесты для domain layer
4. **Incremental delivery** — каждая фаза даёт работающий функционал
5. **Vertical slices** — полные use cases от UI до БД

---

## Phase 0: Project Setup ✅ **COMPLETED**

**Цель:** Подготовка окружения разработки и базовой структуры проекта.

**Дата завершения:** 2025-10-04

### Tasks

- [x] **0.1 Initialize Go module** ✅
  - `go mod init github.com/flowra/flowra`
  - Добавить базовые зависимости (echo v4.13.4, mongo-driver v1.17.4, redis v9.14.0, uuid v1.6.0, viper v1.21.0, testify v1.11.1)

- [x] **0.2 Setup project structure** ✅
  - Создать директории согласно `docs/07-code-structure.md`
  - `internal/domain/`, `internal/application/`, `internal/infrastructure/`, etc.
  - `.gitignore` и `.gitkeep` файлы

- [x] **0.3 Configure development environment** ✅
  - `docker-compose.yml` — MongoDB 7, Redis 7, Keycloak 23
  - `configs/config.yaml` — базовая конфигурация
  - `configs/config.dev.yaml` и `configs/config.prod.yaml`
  - `Makefile` — команды для разработки
  - `.env.example`

- [x] **0.4 Setup linting and formatting** ✅
  - `.golangci.yml` — правила линтинга (обновлен local-prefixes)
  - Команды форматирования в Makefile

- [x] **0.5 Initialize testing framework** ✅
  - Настроить `testify` для assertions
  - Создать helpers для integration tests (mongodb.go, redis.go, helpers.go)
  - Пример теста проходит успешно

**Deliverable:** ✅ Проект с правильной структурой, Docker services настроены, все проверки пройдены.

---

## Phase 1: Domain Layer — Core Aggregates

**Цель:** Реализовать доменную модель без зависимости от инфраструктуры. Чистая бизнес-логика с интерфейсами репозиториев.

📋 **Детальная задача:** [docs/tasks/02-domain-layer.md](./tasks/02-domain-layer.md)

### 1.1 Base Domain Infrastructure

- [ ] **1.1.1 Domain events infrastructure**
  - `internal/domain/event/event.go` — `DomainEvent` interface
  - `internal/domain/event/metadata.go` — `EventMetadata` struct
  - `internal/domain/event/base_event.go` — `BaseEvent` implementation

- [ ] **1.1.2 Common value objects**
  - `internal/domain/common/uuid.go` — UUID type alias
  - `internal/domain/common/errors.go` — domain errors

### 1.2 User Aggregate

- [ ] **1.2.1 User aggregate**
  - `internal/domain/user/user.go` — User aggregate root
  - Fields: ID, Username, Email, DisplayName, IsSystemAdmin
  - Methods: UpdateProfile(), SetAdmin()

- [ ] **1.2.2 User repository interface**
  - `internal/domain/user/repository.go` — Repository interface
  - Methods: FindByID(), FindByEmail(), Save()

- [ ] **1.2.3 User domain events**
  - `internal/domain/user/events.go`
  - Events: UserCreated, UserUpdated

- [ ] **1.2.4 User unit tests**
  - `internal/domain/user/user_test.go`
  - Test all business logic methods

### 1.3 Workspace Aggregate

- [ ] **1.3.1 Workspace aggregate**
  - `internal/domain/workspace/workspace.go`
  - Fields: ID, Name, KeycloakGroupID, CreatedBy
  - Methods: UpdateName(), Delete()

- [ ] **1.3.2 Invite entity**
  - `internal/domain/workspace/invite.go`
  - Fields: Token, ExpiresAt, MaxUses, UsedCount, IsActive
  - Methods: Use(), Deactivate(), IsValid()

- [ ] **1.3.3 Workspace repository interface**
  - `internal/domain/workspace/repository.go`
  - Methods: FindByID(), FindByKeycloakGroup(), Save()

- [ ] **1.3.4 Workspace events**
  - `internal/domain/workspace/events.go`
  - Events: WorkspaceCreated, WorkspaceUpdated, InviteCreated

- [ ] **1.3.5 Workspace unit tests**

### 1.4 Chat Aggregate

- [ ] **1.4.1 Chat aggregate root**
  - `internal/domain/chat/chat.go`
  - Fields: ID, WorkspaceID, Type, IsPublic, Participants
  - Methods: PostMessage(), AddParticipant(), RemoveParticipant(), ConvertToTask()

- [ ] **1.4.2 Message entity**
  - `internal/domain/chat/message.go`
  - Fields: ID, ChatID, AuthorID, Content, Tags, CreatedAt
  - Methods: Edit(), Delete()

- [ ] **1.4.3 Participant value object**
  - `internal/domain/chat/participant.go`
  - Fields: UserID, Role, JoinedAt
  - ParticipantRole enum: admin, member

- [ ] **1.4.4 Chat repository interface**
  - `internal/domain/chat/repository.go`
  - Methods: Load(chatID) (*Chat, error) — from event store
  - Methods: Save(chat *Chat) error — save events
  - Methods: FindByID(chatID) (*ChatReadModel, error) — read model
  - Methods: FindByWorkspace(workspaceID, filters) ([]ChatReadModel, error)

- [ ] **1.4.5 Chat domain events**
  - `internal/domain/chat/events.go`
  - Events: ChatCreated, MessagePosted, ParticipantJoined, ChatTypeChanged

- [ ] **1.4.6 Event sourcing support**
  - Method `Apply(event DomainEvent) error` для восстановления из событий
  - Method `GetUncommittedEvents()` для получения новых событий
  - Method `MarkEventsAsCommitted()` после сохранения

- [ ] **1.4.7 Chat unit tests**
  - Test PostMessage()
  - Test AddParticipant() / RemoveParticipant()
  - Test ConvertToTask()
  - Test event sourcing (Apply events)

### 1.5 Task Aggregate

- [ ] **1.5.1 TaskEntity aggregate**
  - `internal/domain/task/task.go`
  - Fields: ID (== ChatID), Title, State (EntityState)
  - Methods: ChangeStatus(), Assign(), SetPriority(), SetDueDate()

- [ ] **1.5.2 EntityState value object**
  - `internal/domain/task/entity_state.go`
  - Fields: Status, Assignee, Priority, DueDate, CustomFields

- [ ] **1.5.3 Status validation**
  - `internal/domain/task/status.go`
  - ValidStatuses(type) []string — для Task/Bug/Epic
  - IsValidStatus(type, status) bool
  - Hardcoded статусы для MVP

- [ ] **1.5.4 Task repository interface**
  - `internal/domain/task/repository.go`
  - Methods: FindByID(), FindByChatID(), Save()
  - Methods: FindByWorkspace(workspaceID, filters) ([]Task, error)

- [ ] **1.5.5 Task domain events**
  - `internal/domain/task/events.go`
  - Events: TaskCreated, StatusChanged, AssigneeChanged, PriorityChanged

- [ ] **1.5.6 Task unit tests**
  - Test ChangeStatus() with validation
  - Test Assign()
  - Test SetPriority(), SetDueDate()

### 1.6 Notification Aggregate

- [ ] **1.6.1 Notification aggregate**
  - `internal/domain/notification/notification.go`
  - Fields: ID, UserID, Type, Title, Message, ResourceID, ReadAt
  - Methods: MarkAsRead(), Delete()

- [ ] **1.6.2 Notification repository interface**
  - `internal/domain/notification/repository.go`
  - Methods: FindByUserID(), Save(), Delete()

- [ ] **1.6.3 Notification events**
  - Events: NotificationCreated, NotificationRead

- [ ] **1.6.4 Notification unit tests**

**Deliverable Phase 1:** Полная доменная модель с unit-тестами, все зависимости через интерфейсы. Нет привязки к БД или фреймворкам.

---

## Phase 2: Application Layer — Use Cases

**Цель:** Реализовать application services (use cases) с использованием domain aggregates через интерфейсы репозиториев.

### 2.1 Application Infrastructure

- [ ] **2.1.1 Command/Query interfaces**
  - `internal/application/command.go` — Command interface
  - `internal/application/query.go` — Query interface

- [ ] **2.1.2 DTO definitions**
  - Общие DTO структуры для всех services

### 2.2 Chat Application Service

- [ ] **2.2.1 ChatService implementation**
  - `internal/application/chat/service.go`
  - Constructor принимает интерфейсы: ChatRepository, EventStore, EventBus
  - Methods: CreateChat(), PostMessage(), GetChat(), ListChats()
  - Methods: JoinChat(), LeaveChat(), AddParticipant()

- [ ] **2.2.2 Chat commands**
  - `internal/application/chat/commands.go`
  - CreateChatCommand, PostMessageCommand, AddParticipantCommand

- [ ] **2.2.3 Chat DTOs**
  - `internal/application/chat/dto.go`
  - ChatDTO, MessageDTO для responses

- [ ] **2.2.4 ChatService unit tests**
  - Mock repositories
  - Test use cases без БД

### 2.3 Workspace Application Service

- [ ] **2.3.1 WorkspaceService implementation**
  - `internal/application/workspace/service.go`
  - Constructor: WorkspaceRepository, KeycloakClient (interface), EventStore, EventBus
  - Methods: CreateWorkspace(), GetWorkspace(), UpdateWorkspace(), DeleteWorkspace()
  - Methods: CreateInvite(), AcceptInvite(), ListMembers(), RemoveMember()

- [ ] **2.3.2 Workspace commands and DTOs**

- [ ] **2.3.3 WorkspaceService unit tests**

### 2.4 Task Application Service

- [ ] **2.4.1 TaskService implementation**
  - `internal/application/task/service.go`
  - Constructor: TaskRepository, EventStore, EventBus
  - Methods: GetTask(), ListTasks(), GetBoard()
  - Примечание: изменения статуса через теги (CommandExecutor)

- [ ] **2.4.2 CommandExecutor**
  - `internal/application/task/command_executor.go`
  - Применяет команды из тегов к TaskEntity
  - Methods: ExecuteCommand(taskID, command) error

- [ ] **2.4.3 Task commands and DTOs**

- [ ] **2.4.4 TaskService unit tests**

### 2.5 Auth Application Service

- [ ] **2.5.1 AuthService implementation**
  - `internal/application/auth/service.go`
  - Constructor: UserRepository, KeycloakClient (interface)
  - Methods: Login(), Callback(), Logout(), GetCurrentUser()
  - Methods: ValidateToken(), GetUserWorkspaces()

- [ ] **2.5.2 AuthService unit tests**

### 2.6 Event Handlers (Subscribers)

- [ ] **2.6.1 TagParserHandler**
  - `internal/application/eventhandler/tag_parser_handler.go`
  - Implements EventHandler interface
  - Subscribes to: MessagePosted
  - Publishes: TagsParsed

- [ ] **2.6.2 CommandExecutorHandler**
  - `internal/application/eventhandler/command_executor_handler.go`
  - Subscribes to: TagsParsed
  - Publishes: StatusChanged, AssigneeChanged, etc.

- [ ] **2.6.3 NotificationHandler**
  - `internal/application/eventhandler/notification_handler.go`
  - Subscribes to: StatusChanged, AssigneeChanged, TaskCreated
  - Creates notifications

- [ ] **2.6.4 ProjectionHandler (опционально для MVP)**
  - Обновляет read models (chat_read_model, task_projection)

**Deliverable Phase 2:** Application services с бизнес-логикой use cases. Всё через интерфейсы, unit-тесты с моками.

---

## Phase 3: Infrastructure Layer — Implementations

**Цель:** Реализовать интерфейсы репозиториев, event store, event bus и внешние интеграции.

### 3.1 Database Infrastructure

- [ ] **3.1.1 MongoDB connection**
  - `internal/infrastructure/mongodb/connection.go`
  - Connection pool, health check

- [ ] **3.1.2 Database migrations**
  - `migrations/mongodb/001_initial_schema.js`
  - Create collections: events, chats, tasks, users, workspaces, notifications
  - Create indexes

### 3.2 Event Store Implementation

- [ ] **3.2.1 EventStore interface**
  - `internal/infrastructure/eventstore/eventstore.go`
  - Methods: Append(), Load(), LoadAfter()

- [ ] **3.2.2 MongoDB EventStore**
  - `internal/infrastructure/eventstore/mongodb_store.go`
  - Реализация EventStore interface
  - Collection: `events`
  - Optimistic concurrency control (version check)

- [ ] **3.2.3 Snapshot Store (опционально для MVP)**
  - `internal/infrastructure/eventstore/snapshot_store.go`
  - Collection: `snapshots`

- [ ] **3.2.4 Event serialization/deserialization**
  - Маппинг событий в BSON
  - Type registry для десериализации

### 3.3 Event Bus Implementation

- [ ] **3.3.1 EventBus interface**
  - `internal/infrastructure/eventbus/eventbus.go`
  - Methods: Publish(), Subscribe(), Shutdown()

- [ ] **3.3.2 Redis EventBus**
  - `internal/infrastructure/eventbus/redis_bus.go`
  - Реализация EventBus через Redis Pub/Sub
  - Channels по типу события: `events.MessagePosted`

- [ ] **3.3.3 Partitioned EventBus**
  - `internal/infrastructure/eventbus/partitioned_bus.go`
  - Партиционирование по aggregateId для ordering

- [ ] **3.3.4 Event processor with retry**
  - Exponential backoff
  - Dead Letter Queue integration

### 3.4 Repository Implementations

- [ ] **3.4.1 MongoDB ChatRepository**
  - `internal/infrastructure/repository/mongodb/chat_repository.go`
  - Implements domain/chat/Repository interface
  - Methods: Load() — из event store, Save() — сохранить события
  - Methods: FindByID(), FindByWorkspace() — из read model (collection `chats`)

- [ ] **3.4.2 MongoDB TaskRepository**
  - `internal/infrastructure/repository/mongodb/task_repository.go`
  - Implements domain/task/Repository interface
  - Collection: `tasks` (read model)

- [ ] **3.4.3 MongoDB UserRepository**
  - `internal/infrastructure/repository/mongodb/user_repository.go`
  - Implements domain/user/Repository interface
  - Collection: `users`

- [ ] **3.4.4 MongoDB WorkspaceRepository**
  - `internal/infrastructure/repository/mongodb/workspace_repository.go`
  - Implements domain/workspace/Repository interface
  - Collections: `workspaces`, `invites`

- [ ] **3.4.5 MongoDB NotificationRepository**
  - `internal/infrastructure/repository/mongodb/notification_repository.go`
  - Collection: `notifications`

### 3.5 Redis Infrastructure

- [ ] **3.5.1 Redis connection**
  - `internal/infrastructure/redis/connection.go`
  - Connection pool

- [ ] **3.5.2 Session Repository**
  - `internal/infrastructure/repository/redis/session_repository.go`
  - Store/retrieve sessions
  - TTL management

- [ ] **3.5.3 Idempotency Repository**
  - `internal/infrastructure/repository/redis/idempotency_repository.go`
  - Track processed events
  - Collection: `processed_events` (MongoDB) или Redis keys
  - TTL: 7 days

### 3.6 Keycloak Integration

- [ ] **3.6.1 KeycloakClient interface**
  - `internal/infrastructure/keycloak/client.go`
  - Methods: ExchangeCode(), RefreshToken(), ValidateToken()
  - Methods: CreateGroup(), AddUserToGroup(), RemoveUserFromGroup()

- [ ] **3.6.2 KeycloakClient implementation**
  - `internal/infrastructure/keycloak/keycloak_client.go`
  - HTTP client для Keycloak Admin API
  - Service account authentication

- [ ] **3.6.3 Token validator**
  - `internal/infrastructure/keycloak/token_validator.go`
  - Validate JWT signature (JWKS)
  - Extract claims

### 3.7 WebSocket Infrastructure

- [ ] **3.7.1 WebSocket Hub**
  - `internal/infrastructure/websocket/hub.go`
  - Connection manager
  - Methods: Register(), Unregister(), BroadcastToChat(), SendToUser()

- [ ] **3.7.2 WebSocket Client**
  - `internal/infrastructure/websocket/client.go`
  - Client connection wrapper
  - Read/write pumps

- [ ] **3.7.3 Message types**
  - `internal/infrastructure/websocket/message.go`
  - WebSocket message structs

**Deliverable Phase 3:** Все инфраструктурные компоненты реализованы. Интеграционные тесты с реальной БД.

---

## Phase 4: Interface Layer — HTTP & WebSocket Handlers

**Цель:** Реализовать HTTP REST API и WebSocket endpoints.

### 4.1 HTTP Infrastructure

- [ ] **4.1.1 Router setup**
  - `internal/handler/http/router.go`
  - Chi router configuration
  - Route groups

- [ ] **4.1.2 Response helpers**
  - `internal/handler/http/response.go`
  - respondJSON(), respondError()
  - Error formatting

- [ ] **4.1.3 Request helpers**
  - Context helpers: getUserIDFromContext(), getWorkspaceIDFromContext()
  - JSON decoding

### 4.2 Middleware

- [ ] **4.2.1 Auth middleware**
  - `internal/middleware/auth.go`
  - JWT validation
  - User context injection

- [ ] **4.2.2 Workspace access middleware**
  - `internal/middleware/workspace.go`
  - Check workspace membership

- [ ] **4.2.3 Chat access middleware**
  - `internal/middleware/chat.go`
  - Check chat access level (read/write/admin)

- [ ] **4.2.4 CORS middleware**
  - `internal/middleware/cors.go`

- [ ] **4.2.5 Rate limiting middleware**
  - `internal/middleware/ratelimit.go`
  - Per-user and per-endpoint limits

- [ ] **4.2.6 Logging middleware**
  - `internal/middleware/logging.go`
  - Structured request/response logging

### 4.3 HTTP Handlers

- [ ] **4.3.1 AuthHandler**
  - `internal/handler/http/auth_handler.go`
  - POST /auth/login, GET /auth/callback, POST /auth/logout, GET /auth/me

- [ ] **4.3.2 WorkspaceHandler**
  - `internal/handler/http/workspace_handler.go`
  - CRUD workspaces, members, invites

- [ ] **4.3.3 ChatHandler**
  - `internal/handler/http/chat_handler.go`
  - CRUD chats, join/leave, participants

- [ ] **4.3.4 MessageHandler**
  - `internal/handler/http/message_handler.go`
  - GET /chats/{id}/messages, POST /chats/{id}/messages
  - PUT /messages/{id}, DELETE /messages/{id}

- [ ] **4.3.5 TaskHandler**
  - `internal/handler/http/task_handler.go`
  - GET /tasks, GET /tasks/{id}, GET /board

- [ ] **4.3.6 NotificationHandler**
  - `internal/handler/http/notification_handler.go`
  - GET /notifications, PUT /notifications/{id}/read

- [ ] **4.3.7 AdminHandler (DLQ)**
  - `internal/handler/http/admin_handler.go`
  - GET /admin/dlq, POST /admin/dlq/{id}/replay

### 4.4 WebSocket Handlers

- [ ] **4.4.1 WebSocket connection handler**
  - `internal/handler/websocket/handler.go`
  - Upgrade HTTP → WebSocket
  - Authentication via token query param

- [ ] **4.4.2 Message router**
  - `internal/handler/websocket/message_handler.go`
  - Route incoming messages: subscribe.chat, chat.typing, ping

- [ ] **4.4.3 Event broadcaster**
  - Subscribe to EventBus events
  - Broadcast to WebSocket clients: chat.message.posted, task.updated, notification.new

**Deliverable Phase 4:** REST API и WebSocket endpoints работают. Можно тестировать через curl/Postman.

---

## Phase 5: Entry Points & Configuration

**Цель:** Собрать всё воедино в исполняемые приложения.

### 5.1 Configuration

- [ ] **5.1.1 Config loader**
  - `internal/config/config.go`
  - Viper setup: yaml + env variables
  - Validation

- [ ] **5.1.2 Config files**
  - `configs/config.yaml` — defaults
  - `configs/config.dev.yaml` — development
  - `configs/config.prod.yaml` — production

### 5.2 Logging

- [ ] **5.2.1 Logger wrapper**
  - `pkg/logger/logger.go`
  - Structured logging with slog
  - Log levels

### 5.3 API Server

- [ ] **5.3.1 Main entrypoint**
  - `cmd/api/main.go`
  - Load config
  - Initialize dependencies (repositories, services, handlers)
  - Manual DI (constructor injection)
  - Setup router
  - Start HTTP server
  - Graceful shutdown

- [ ] **5.3.2 Dependency wiring**
  - Явная композиция всех зависимостей
  - Передача интерфейсов конструкторам

### 5.4 Worker

- [ ] **5.4.1 Worker entrypoint**
  - `cmd/worker/main.go`
  - Load config
  - Initialize event handlers
  - Subscribe to events
  - Listen and process events
  - Graceful shutdown

### 5.5 Migrator

- [ ] **5.5.1 Migration runner**
  - `cmd/migrator/main.go`
  - Apply MongoDB migrations
  - Create indexes

**Deliverable Phase 5:** Приложение запускается и работает end-to-end.

---

## Phase 6: Testing & Quality Assurance

**Цель:** Comprehensive testing coverage.

### 6.1 Unit Tests

- [ ] **6.1.1 Domain layer tests** (уже сделаны в Phase 1)
- [ ] **6.1.2 Application layer tests** (с моками, Phase 2)
- [ ] **6.1.3 Handler tests** (с mock services)

### 6.2 Integration Tests

- [ ] **6.2.1 Repository integration tests**
  - `tests/integration/repository_test.go`
  - Test с реальной MongoDB (testcontainers или docker-compose)

- [ ] **6.2.2 Event flow integration tests**
  - Test полный event flow: MessagePosted → TagsParsed → StatusChanged

- [ ] **6.2.3 API integration tests**
  - Test HTTP endpoints с реальной БД

### 6.3 End-to-End Tests

- [ ] **6.3.1 User flows**
  - `tests/e2e/create_task_flow_test.go`
  - Create workspace → Create chat → Post message with tags → Verify task created

- [ ] **6.3.2 Chat flows**
  - Join chat → Post messages → Verify broadcast via WebSocket

### 6.4 Performance Tests

- [ ] **6.4.1 Load testing** (опционально для MVP)
  - Simulate 100+ concurrent users
  - Measure response times

**Deliverable Phase 6:** Test coverage > 80%, все critical paths покрыты.

---

## Phase 7: Frontend (HTMX + Pico CSS)

**Цель:** Минимальный UI для взаимодействия с системой.

### 7.1 Base Templates

- [ ] **7.1.1 Layout template**
  - `web/templates/layout.html`
  - Base HTML structure, Pico CSS, HTMX scripts

- [ ] **7.1.2 Navigation**
  - Navbar: workspace selector, notifications, user menu

### 7.2 Pages

- [ ] **7.2.1 Login page**
  - `web/templates/auth/login.html`
  - Redirect to Keycloak

- [ ] **7.2.2 Workspace list**
  - `web/templates/workspace/list.html`
  - List user's workspaces
  - Create new workspace button

- [ ] **7.2.3 Kanban board**
  - `web/templates/board/index.html`
  - Columns: To Do, In Progress, Done
  - Drag-n-drop (HTMX + Alpine.js или vanilla JS)

- [ ] **7.2.4 Chat view**
  - `web/templates/chat/view.html`
  - Message list
  - Message input with tag autocomplete
  - Participants sidebar

- [ ] **7.2.5 Task details**
  - `web/templates/task/view.html`
  - Task metadata (status, assignee, priority)
  - Chat messages below

### 7.3 HTMX Components

- [ ] **7.3.1 Message list component**
  - `web/components/message_list.html`
  - HTMX polling for updates (или WebSocket via htmx-ext-ws)

- [ ] **7.3.2 Task card component**
  - `web/components/task_card.html`
  - Draggable card

- [ ] **7.3.3 Notification dropdown**
  - `web/components/notifications.html`
  - HTMX updates on new notifications

### 7.4 Static Assets

- [ ] **7.4.1 CSS customization**
  - `web/static/css/custom.css`
  - Pico CSS overrides

- [ ] **7.4.2 JavaScript**
  - `web/static/js/app.js`
  - WebSocket connection
  - Drag-n-drop for kanban
  - Tag autocomplete

**Deliverable Phase 7:** Работающий UI для основных use cases.

---

## Phase 8: Deployment & DevOps

**Цель:** Подготовка к продакшену.

### 8.1 Docker

- [ ] **8.1.1 Application Dockerfile**
  - Multi-stage build
  - Minimal runtime image

- [ ] **8.1.2 Docker Compose for production**
  - All services: api, worker, mongo, redis, keycloak

### 8.2 CI/CD

- [ ] **8.2.1 GitHub Actions**
  - `.github/workflows/ci.yml`
  - Run tests, linting
  - Build Docker image

- [ ] **8.2.2 Deployment script**
  - `scripts/deploy.sh`
  - Deploy to VPS or cloud

### 8.3 Monitoring

- [ ] **8.3.1 Prometheus metrics**
  - Instrument handlers, event processors
  - Metrics: request count, duration, event processing

- [ ] **8.3.2 Health checks**
  - `/health` endpoint
  - Check MongoDB, Redis, Keycloak connectivity

- [ ] **8.3.3 Logging setup**
  - Structured logs to stdout
  - Aggregation (ELK или Loki)

**Deliverable Phase 8:** Application готово к продакшен deployment.

---

## Phase 9: Documentation & Polish

**Цель:** Финализация MVP.

### 9.1 Documentation

- [ ] **9.1.1 API documentation**
  - OpenAPI/Swagger spec (опционально)
  - Postman collection

- [ ] **9.1.2 User guide**
  - How to create workspace
  - How to use tags
  - Keyboard shortcuts

- [ ] **9.1.3 Developer guide**
  - How to run locally
  - How to add new features
  - Architecture overview

### 9.2 Bug Fixes & Polish

- [ ] **9.2.1 Bug triage**
  - Fix critical bugs from testing

- [ ] **9.2.2 UX improvements**
  - Error messages
  - Loading states
  - Empty states

### 9.3 Performance Optimization

- [ ] **9.3.1 Query optimization**
  - Add missing indexes
  - Optimize slow queries

- [ ] **9.3.2 Caching** (если нужно)
  - Redis cache for read-heavy queries

**Deliverable Phase 9:** Polished MVP ready for users.

---

## Post-MVP: V2 Features (Future)

Эти фичи не входят в MVP, но запланированы на будущее:

### Advanced Features

- [ ] **Кастомизация статусных моделей**
  - UI для настройки статусов workspace
  - Валидация переходов (state machine)

- [ ] **Связи между задачами**
  - #parent, #blocks, #relates теги
  - Визуализация зависимостей

- [ ] **Custom tags**
  - Регистрация кастомных тегов
  - Типизация и валидация

- [ ] **Естественный язык для дат**
  - #due tomorrow, #due next friday

- [ ] **Алиасы тегов**
  - #s → #status, #p → #priority

- [ ] **Metrics и analytics**
  - Dashboard с метриками
  - Lead time, cycle time
  - Burndown charts

- [ ] **Advanced search**
  - Full-text search по сообщениям
  - Фильтры по custom tags

- [ ] **Email notifications**
  - Configurable notification preferences

- [ ] **Mobile app**
  - React Native или PWA

### Technical Improvements

- [ ] **Transactional Outbox**
  - At-least-once delivery гарантия

- [ ] **CQRS read replicas**
  - Оптимизированные проекции для разных use cases

- [ ] **Event Store optimization**
  - Snapshots для больших aggregates
  - Event archiving

- [ ] **Multi-tenancy**
  - Database per workspace (опционально)

---

## Success Criteria для MVP

### Functional Requirements

- ✅ Пользователь может создать workspace
- ✅ Пользователь может пригласить других через invite link
- ✅ Пользователь может создать chat/task/bug через UI или теги
- ✅ Пользователь может отправлять сообщения с тегами
- ✅ Теги автоматически применяются (статус, assignee, priority)
- ✅ Канбан-доска отображает задачи по статусам
- ✅ Drag-n-drop на канбане меняет статус
- ✅ Real-time обновления через WebSocket
- ✅ Уведомления о изменениях задач

### Non-Functional Requirements

- ✅ Response time < 200ms для API endpoints (95th percentile)
- ✅ WebSocket latency < 100ms
- ✅ Support 100 concurrent users
- ✅ Test coverage > 80%
- ✅ Zero downtime deployment
- ✅ All data encrypted at rest and in transit

### Technical Requirements

- ✅ Domain-driven design реализован
- ✅ Event sourcing работает для chat/task aggregates
- ✅ Все зависимости через интерфейсы
- ✅ Unit tests для domain layer
- ✅ Integration tests для repositories
- ✅ E2E tests для critical paths
- ✅ Graceful shutdown для всех services
- ✅ Health checks и monitoring

---

## Development Guidelines

### Code Quality

- **Linting:** Использовать golangci-lint перед каждым commit
- **Testing:** Писать unit tests одновременно с кодом (TDD где возможно)
- **Code review:** Все изменения через Pull Request
- **Documentation:** Godoc для всех публичных функций

### Git Workflow

- **Branches:** `feature/phase-X-task-name`, `bugfix/issue-description`
- **Commits:** Conventional commits (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`)
- **PR naming:** `[Phase X] Task description`

### Definition of Done

Для каждой задачи:
- [ ] Код написан и работает
- [ ] Unit tests покрывают логику
- [ ] Integration tests (если применимо)
- [ ] Code review пройден
- [ ] Documentation обновлена
- [ ] Нет критических lint warnings
- [ ] Изменения протестированы локально

---

## Приоритеты и зависимости

### Critical Path

```
Phase 0 (Setup)
    ↓
Phase 1 (Domain) ← MUST be complete before Phase 2
    ↓
Phase 2 (Application) ← MUST be complete before Phase 3
    ↓
Phase 3 (Infrastructure) ← Can partially overlap with Phase 4
    ↓
Phase 4 (Handlers) ← Depends on Phase 3
    ↓
Phase 5 (Entry Points) ← Integrates everything
    ↓
Phase 6 (Testing) ← Throughout all phases
    ↓
Phase 7 (Frontend) ← Can start after Phase 4
    ↓
Phase 8 (Deployment)
    ↓
Phase 9 (Polish)
```

### Parallel Work Opportunities

После Phase 1:
- **Developer A:** Phase 2 (Application services)
- **Developer B:** Phase 3 (Infrastructure, repositories)

После Phase 3:
- **Developer A:** Phase 4 (HTTP handlers)
- **Developer B:** Phase 7 (Frontend)

### Incremental Milestones

- **Milestone 1:** После Phase 2 — domain и application слои готовы, можно тестировать с моками
- **Milestone 2:** После Phase 4 — API работает, можно тестировать через curl/Postman
- **Milestone 3:** После Phase 7 — MVP с UI, готово для internal testing
- **Milestone 4:** После Phase 9 — Production-ready MVP

---

## Summary

Roadmap организован по принципу **inside-out**: начинаем с чистой доменной модели (Phase 1), затем use cases (Phase 2), затем инфраструктура (Phase 3), и наконец интерфейсы (Phase 4+).

Все компоненты используют **интерфейсы** для зависимостей, что позволяет:
- Тестировать каждый слой изолированно
- Менять реализации без изменения бизнес-логики
- Параллельную разработку разных слоёв

Каждая фаза даёт **deliverable** — работающий компонент или функционал, что позволяет инкрементально двигаться к MVP.
