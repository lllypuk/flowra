# План развития проекта new-teams-up (2025)

**Дата составления:** 2025-11-11
**Версия:** 1.0
**Текущий статус:** Active Development (Phase 2-3, 82% Complete)
**Горизонт планирования:** 6 месяцев

---

## 📊 Executive Summary

### Текущее состояние проекта

**Версия:** 0.4.0-alpha
**Прогресс:** 82% от Phase 4 (UseCase Implementation)
**Строк кода:** ~23,000 LOC
**Test Coverage:** Domain 90%+, Application 64.7% (критическая проблема: Chat 0%)

#### ✅ Что реализовано (сильные стороны)

1. **Domain Layer (90%+)** - полностью функционален
   - 6 Event-Sourced агрегатов (Chat, Message, Task, Notification, User, Workspace)
   - 30+ типов domain events
   - Tag Processing System для команд из чата
   - Comprehensive business logic

2. **Application Layer (64.7%)** - частично готов
   - 40+ use cases реализовано
   - Message/User/Workspace/Notification: 78-86% coverage ✅
   - Task: 84.9% coverage ✅
   - **КРИТИЧНО:** Chat: 0% coverage ❌

3. **Infrastructure (30%)**
   - In-memory Event Store ✅
   - MongoDB v2 connection setup ✅
   - Redis client setup ✅
   - Остальное в разработке

4. **Testing Infrastructure (85%)** - отличная база
   - Mocks, Fixtures, Test Utilities
   - Integration test helpers
   - MongoDB v2/Redis test setup

#### ❌ Критические проблемы

1. **Chat UseCases Testing Gap** 🔴 БЛОКЕР
   - 12 command use cases без единого теста
   - 3 query use cases не реализованы
   - Риск: нестабильность ключевой функциональности

2. **Infrastructure Layer отсутствует** 🟡 HIGH
   - MongoDB/Redis repositories не реализованы
   - Event Bus отсутствует
   - HTTP/WebSocket handlers не созданы

3. **No Entry Points** 🟡 HIGH
   - Невозможно запустить приложение
   - cmd/api/main.go, cmd/worker/main.go отсутствуют

4. **Frontend отсутствует** 🟡 MEDIUM
   - HTMX templates не созданы
   - UI нельзя протестировать

### Рекомендуемая стратегия

**Принцип:** Завершить текущую фазу → Минимальный рабочий MVP → Итеративное развитие

1. **Неделя 1-2:** Завершить Application Layer (Chat tests + queries)
2. **Неделя 3-6:** Infrastructure Layer (repositories, handlers)
3. **Неделя 7-10:** Entry Points + Базовый Frontend
4. **Неделя 11-12:** Testing, Bugfixing, MVP Release
5. **Месяц 4-6:** Оптимизация, расширение функциональности

---

## 🎯 Фаза 0: КРИТИЧЕСКИЕ ИСПРАВЛЕНИЯ (0-2 недели)

### Приоритет: 🔴 КРИТИЧЕСКИЙ
### Цель: Устранить блокеры, завершить Application Layer
### Оценка: 6-8 часов работы

---

### Task 0.1: Chat UseCases Testing (БЛОКЕР) 🔴

**Проблема:**
Chat domain имеет 0% test coverage при 12 реализованных use cases. Это наибольший риск проекта - ключевая функциональность может содержать критические баги.

**Решение:**
Создать comprehensive test suite для всех Chat use cases.

**Детали реализации:**

```
Файлы для создания:
├── internal/application/chat/
│   ├── create_chat_test.go          (8 тестов)
│   ├── participants_test.go         (12 тестов: Add/Remove)
│   ├── convert_test.go              (12 тестов: Task/Bug/Epic)
│   ├── management_test.go           (15 тестов: Status/Assign/Priority/DueDate)
│   ├── rename_severity_test.go      (10 тестов)
│   └── test_setup.go                (mocks setup)

Итого: ~60 unit tests
```

**Тестовое покрытие:**
- Happy path для всех операций
- Error cases (validation, authorization, not found)
- Edge cases (duplicate participants, invalid status transitions)
- Event publishing verification

**Критерии успеха:**
- ✅ Coverage Chat domain: 0% → 85%+
- ✅ Application Layer overall: 64.7% → 75%+
- ✅ Все тесты проходят
- ✅ No regressions в других доменах

**Время:** 3-4 часа
**Референс:** `internal/application/message/*_test.go` (аналогичная структура)

---

### Task 0.2: Chat Query UseCases Implementation 🔴

**Проблема:**
Query use cases для Chat не реализованы. Невозможно получить данные чата для UI.

**Решение:**
Реализовать 3 query use cases с полным тестированием.

**Use Cases:**

1. **GetChatUseCase**
   ```go
   type GetChatQuery struct {
       ChatID      uuid.UUID
       RequestedBy uuid.UUID  // для проверки доступа
   }

   type GetChatResult struct {
       Chat        *ChatDTO
       Permissions ChatPermissions  // read/write/admin
   }
   ```

   Тесты (4):
   - ✅ Success case
   - ❌ Chat not found
   - ❌ User not participant (no access)
   - ✅ Public chat access

2. **ListChatsUseCase**
   ```go
   type ListChatsQuery struct {
       WorkspaceID uuid.UUID
       Type        *ChatType      // optional filter
       Limit       int
       Offset      int
       RequestedBy uuid.UUID
   }

   type ListChatsResult struct {
       Chats      []ChatDTO
       Total      int
       HasMore    bool
   }
   ```

   Тесты (6):
   - ✅ List all chats
   - ✅ Filter by type (Task/Bug/Epic)
   - ✅ Pagination works
   - ✅ Only user's chats returned
   - ✅ Public chats included
   - ❌ Invalid workspace

3. **ListParticipantsUseCase**
   ```go
   type ListParticipantsQuery struct {
       ChatID      uuid.UUID
       RequestedBy uuid.UUID
   }

   type ListParticipantsResult struct {
       Participants []ParticipantDTO
   }
   ```

   Тесты (5):
   - ✅ Success case
   - ❌ Chat not found
   - ❌ Not a participant
   - ✅ Includes roles and join dates
   - ✅ Sorted by join date

**Файлы:**
```
internal/application/chat/
├── queries.go           (new - query definitions)
├── get_chat.go          (new)
├── list_chats.go        (new)
├── list_participants.go (new)
├── get_chat_test.go     (new)
├── list_chats_test.go   (new)
└── list_participants_test.go (new)
```

**Критерии успеха:**
- ✅ 3 query use cases реализованы
- ✅ 15 unit tests покрывают все сценарии
- ✅ Coverage >85%
- ✅ Pagination протестирована
- ✅ Authorization checks на месте

**Время:** 1.5-2 часа
**Референс:** `internal/application/message/query*.go`

---

### Task 0.3: Documentation Sync 🟡

**Проблема:**
README и архитектурная документация устарели.

**Решение:**
Синхронизировать документацию с текущим состоянием кода.

**Что обновить:**

1. **README.md**
   - Обновить метрики (23,000 LOC, 40+ use cases)
   - Добавить секцию "Current Status" с прогрессом
   - Обновить Quick Start (добавить примеры тестов)

2. **docs/01-architecture.md**
   - Добавить актуальную диаграмму слоев
   - Документировать Tag Processing integration
   - Обновить Event Flow примеры

3. **Создать API_USAGE.md**
   - Примеры использования каждого use case
   - Code snippets для типичных сценариев
   - Integration примеры (Tag + Chat + Message)

**Критерии успеха:**
- ✅ Документация отражает реальность
- ✅ Новый разработчик может разобраться за 30 минут
- ✅ Примеры кода работают

**Время:** 1 час

---

### Итоговый результат Phase 0:

**После завершения:**
- ✅ Application Layer: 100% реализован и протестирован
- ✅ Test Coverage Application: 75%+ overall
- ✅ Нет критических блокеров
- ✅ Готовность к Infrastructure Layer: 100%

**Оценка времени:** 6-8 часов активной работы
**Календарное время:** 1-2 дня (учитывая code review)

**Следующий шаг:** → Фаза 1 (Infrastructure Layer)

---

## 🏗️ Фаза 1: INFRASTRUCTURE LAYER (Недели 3-6)

### Приоритет: 🟡 HIGH
### Цель: Реализовать persistence, event bus, keycloak integration
### Оценка: 3-4 недели (80-100 часов)

---

### Milestone 1.1: Repository Implementations (2 недели)

#### Task 1.1.1: MongoDB Event Store 🔴

**Текущее состояние:**
In-memory event store работает, но не персистентен.

**Задача:**
Реализовать production-ready MongoDB Event Store.

**Компоненты:**

1. **Event Store Interface** (уже есть в shared)
   ```go
   type EventStore interface {
       SaveEvents(ctx, aggregateID, events, expectedVersion) error
       LoadEvents(ctx, aggregateID) ([]DomainEvent, error)
       LoadEventsAfter(ctx, aggregateID, version) ([]DomainEvent, error)
   }
   ```

2. **MongoDB Implementation**
   ```go
   // internal/infrastructure/eventstore/mongodb_store.go

   type MongoEventStore struct {
       client     *mongo.Client
       database   *mongo.Database
       collection *mongo.Collection  // "events"
   }

   // Schema:
   // {
   //   _id: ObjectId,
   //   aggregate_id: UUID,
   //   aggregate_type: string,
   //   event_type: string,
   //   version: int,
   //   data: BSON,
   //   metadata: {
   //     timestamp: Date,
   //     user_id: UUID,
   //     correlation_id: UUID
   //   },
   //   created_at: Date
   // }
   ```

3. **Features:**
   - ✅ Optimistic concurrency control (version check)
   - ✅ Event serialization/deserialization
   - ✅ Indexes: `{aggregate_id: 1, version: 1}` (unique)
   - ✅ Bulk event append (single transaction)
   - ✅ Event replay capability
   - ⚠️ Snapshots (optional for MVP, рекомендуется для v2)

4. **Error Handling:**
   - ConcurrencyError при конфликте версий
   - Retry logic с exponential backoff
   - Idempotency через `correlation_id`

**Тестирование:**
- Unit tests с mock Mongo client
- Integration tests с real MongoDB (testcontainers)
- Concurrency tests (multiple writers)
- Performance tests (1000 events append/load)

**Файлы:**
```
internal/infrastructure/eventstore/
├── eventstore.go              (interface - уже есть)
├── mongodb_store.go           (implementation)
├── mongodb_store_test.go      (unit tests)
├── serializer.go              (event serialization)
├── serializer_test.go
└── integration_test.go        (with MongoDB)
```

**Критерии успеха:**
- ✅ Append 100 events < 50ms
- ✅ Load 1000 events < 100ms
- ✅ Concurrency control работает
- ✅ No data loss
- ✅ Test coverage >85%

**Время:** 3-4 дня

---

#### Task 1.1.2: MongoDB Repositories 🔴

**Задача:**
Реализовать repository interfaces для всех доменов.

**Repositories:**

1. **ChatRepository**
   ```go
   // Load/Save через Event Store (event sourcing)
   Load(ctx, chatID) (*Chat, error)
   Save(ctx, *Chat) error

   // Query methods через read model
   FindByID(ctx, chatID) (*ChatReadModel, error)
   FindByWorkspace(ctx, workspaceID, filters) ([]ChatReadModel, error)
   FindByParticipant(ctx, userID) ([]ChatReadModel, error)
   ```

   Collections:
   - `events` (event sourcing)
   - `chat_read_model` (denormalized для queries)

2. **MessageRepository**
   ```go
   FindByID(ctx, messageID) (*Message, error)
   FindByChatID(ctx, chatID, pagination) ([]Message, error)
   FindThread(ctx, parentID) ([]Message, error)
   Save(ctx, *Message) error
   Update(ctx, *Message) error
   SoftDelete(ctx, messageID) error
   ```

   Collection: `messages`

3. **UserRepository**
   ```go
   FindByID(ctx, userID) (*User, error)
   FindByUsername(ctx, username) (*User, error)
   FindByEmail(ctx, email) (*User, error)
   List(ctx, pagination) ([]User, error)
   Save(ctx, *User) error
   Update(ctx, *User) error
   ```

   Collection: `users`

4. **WorkspaceRepository**
   ```go
   FindByID(ctx, workspaceID) (*Workspace, error)
   FindByKeycloakGroupID(ctx, groupID) (*Workspace, error)
   FindByUser(ctx, userID) ([]Workspace, error)
   Save(ctx, *Workspace) error
   Update(ctx, *Workspace) error
   ```

   Collections: `workspaces`, `workspace_members`

5. **NotificationRepository**
   ```go
   FindByID(ctx, notificationID) (*Notification, error)
   FindByUser(ctx, userID, unreadOnly bool, pagination) ([]Notification, error)
   CountUnread(ctx, userID) (int, error)
   Save(ctx, *Notification) error
   MarkAsRead(ctx, notificationID) error
   MarkAllAsRead(ctx, userID) error
   Delete(ctx, notificationID) error
   ```

   Collection: `notifications`

**Общие требования:**
- Implements domain repository interfaces
- Error handling (NotFoundError, ConflictError)
- Pagination support (limit/offset)
- Filtering support
- Sorting support
- Indexes для всех query methods

**Indexes:**
```javascript
// messages
db.messages.createIndex({ chat_id: 1, created_at: -1 })
db.messages.createIndex({ parent_id: 1, created_at: 1 })

// notifications
db.notifications.createIndex({ user_id: 1, read_at: 1, created_at: -1 })

// chat_read_model
db.chat_read_model.createIndex({ workspace_id: 1, type: 1 })
db.chat_read_model.createIndex({ participants: 1 })

// workspace_members
db.workspace_members.createIndex({ workspace_id: 1, user_id: 1 }, { unique: true })
```

**Тестирование:**
- Unit tests с mock MongoDB
- Integration tests с real MongoDB
- Transaction tests (где применимо)
- Index usage verification

**Файлы:**
```
internal/infrastructure/repository/mongodb/
├── chat_repository.go
├── chat_repository_test.go
├── message_repository.go
├── message_repository_test.go
├── user_repository.go
├── user_repository_test.go
├── workspace_repository.go
├── workspace_repository_test.go
├── notification_repository.go
├── notification_repository_test.go
└── common.go  (shared utilities)
```

**Критерии успеха:**
- ✅ Все repository interfaces реализованы
- ✅ Test coverage >80%
- ✅ Query performance < 50ms (95th percentile)
- ✅ Indexes созданы и используются

**Время:** 5-6 дней

---

#### Task 1.1.3: Redis Repositories 🟡

**Задача:**
Реализовать Redis-based repositories для cache и sessions.

**Repositories:**

1. **SessionRepository**
   ```go
   type SessionRepository interface {
       Save(ctx, sessionID, data, ttl) error
       Load(ctx, sessionID) (*SessionData, error)
       Delete(ctx, sessionID) error
       Extend(ctx, sessionID, ttl) error
   }
   ```

   Keys: `session:{sessionID}`
   TTL: 24 hours

2. **IdempotencyRepository**
   ```go
   type IdempotencyRepository interface {
       IsProcessed(ctx, eventID) (bool, error)
       MarkAsProcessed(ctx, eventID, ttl) error
   }
   ```

   Keys: `idempotency:{eventID}`
   TTL: 7 days

3. **CacheRepository** (optional для MVP)
   ```go
   type CacheRepository interface {
       Get(ctx, key) (interface{}, error)
       Set(ctx, key, value, ttl) error
       Delete(ctx, key) error
       DeletePattern(ctx, pattern) error
   }
   ```

**Тестирование:**
- Integration tests с Redis
- TTL verification
- Concurrency tests

**Файлы:**
```
internal/infrastructure/repository/redis/
├── session_repository.go
├── session_repository_test.go
├── idempotency_repository.go
└── idempotency_repository_test.go
```

**Критерии успеха:**
- ✅ Session management работает
- ✅ Idempotency защита активна
- ✅ Test coverage >80%

**Время:** 2 дня

---

### Milestone 1.2: Event Bus Implementation (1 неделя)

#### Task 1.2.1: Redis Event Bus 🔴

**Задача:**
Реализовать Event Bus для pub/sub через Redis.

**Компоненты:**

1. **EventBus Interface**
   ```go
   type EventBus interface {
       Publish(ctx, event DomainEvent) error
       Subscribe(eventType string, handler EventHandler) error
       Unsubscribe(eventType string) error
       Shutdown() error
   }

   type EventHandler interface {
       Handle(ctx, event DomainEvent) error
   }
   ```

2. **Redis Implementation**
   ```go
   type RedisEventBus struct {
       client     *redis.Client
       handlers   map[string][]EventHandler
       shutdown   chan struct{}
   }

   // Channels:
   // events.MessagePosted
   // events.ChatCreated
   // events.StatusChanged
   // etc.
   ```

3. **Features:**
   - ✅ Multiple subscribers per event type
   - ✅ Error handling и retry
   - ✅ Dead Letter Queue для failed events
   - ✅ Graceful shutdown
   - ✅ Event serialization (JSON)
   - ⚠️ Partitioning по aggregate ID (optional, для ordering)

4. **Error Handling:**
   - Retry с exponential backoff (3 attempts)
   - DLQ после failures
   - Logging всех errors

**Event Handlers:**

Реализовать handlers для:
1. **TagParserHandler** - уже интегрирован в SendMessageUseCase
2. **NotificationHandler** - создание notifications при событиях:
   ```go
   type NotificationHandler struct {
       createNotifUseCase CreateNotificationUseCase
   }

   func (h *NotificationHandler) Handle(ctx, event) error {
       switch e := event.(type) {
       case *ChatCreated:
           // Notify participants
       case *StatusChanged:
           // Notify assignee
       case *UserAssigned:
           // Notify user
       case *MessagePosted:
           // Notify chat participants (except author)
       }
   }
   ```

3. **ProjectionHandler** (optional для MVP)
   - Обновление read models (chat_read_model)

**Тестирование:**
- Unit tests с mock Redis
- Integration tests с real Redis
- Handler tests (mock use cases)
- Concurrency tests
- DLQ tests

**Файлы:**
```
internal/infrastructure/eventbus/
├── eventbus.go              (interface)
├── redis_bus.go             (implementation)
├── redis_bus_test.go
├── handler.go               (base handler)
└── dlq.go                   (dead letter queue)

internal/application/eventhandler/
├── notification_handler.go
├── notification_handler_test.go
└── projection_handler.go    (optional)
```

**Критерии успеха:**
- ✅ Events публикуются и доставляются
- ✅ Multiple handlers работают
- ✅ DLQ ловит failed events
- ✅ Graceful shutdown корректен
- ✅ Test coverage >80%

**Время:** 4-5 дней

---

### Milestone 1.3: Keycloak Integration (1 неделя)

#### Task 1.3.1: Keycloak Client 🟡

**Задача:**
Реализовать Keycloak integration для OAuth2/OIDC и group management.

**Компоненты:**

1. **KeycloakClient Interface**
   ```go
   type KeycloakClient interface {
       // OAuth2/OIDC
       ExchangeCode(ctx, code string) (*TokenResponse, error)
       RefreshToken(ctx, refreshToken string) (*TokenResponse, error)
       ValidateToken(ctx, token string) (*Claims, error)
       RevokeToken(ctx, token string) error

       // Group Management
       CreateGroup(ctx, name string) (string, error)  // returns group ID
       AddUserToGroup(ctx, userID, groupID string) error
       RemoveUserFromGroup(ctx, userID, groupID string) error
       ListGroupMembers(ctx, groupID string) ([]string, error)
   }
   ```

2. **HTTP Client Implementation**
   ```go
   type HTTPKeycloakClient struct {
       baseURL      string
       realm        string
       clientID     string
       clientSecret string
       httpClient   *http.Client
   }
   ```

3. **Token Validator**
   ```go
   type TokenValidator struct {
       jwksURL  string
       issuer   string
       audience string
   }

   func (v *TokenValidator) Validate(token string) (*Claims, error) {
       // JWT signature verification via JWKS
       // Claims extraction (userID, roles, groups)
   }
   ```

**OAuth2 Flow:**
```
1. User → GET /auth/login → Redirect to Keycloak
2. Keycloak → User login → Redirect to /auth/callback?code=...
3. App → Exchange code for tokens (access + refresh)
4. App → Set session cookie
5. Subsequent requests → Validate access token (JWT)
6. Token expired → Refresh via refresh token
```

**Тестирование:**
- Unit tests с mock HTTP client
- Integration tests с real Keycloak (testcontainers)
- OAuth2 flow E2E test
- Group management tests

**Файлы:**
```
internal/infrastructure/keycloak/
├── client.go                (interface)
├── http_client.go           (implementation)
├── http_client_test.go
├── token_validator.go
├── token_validator_test.go
└── integration_test.go
```

**Критерии успеха:**
- ✅ OAuth2 flow работает end-to-end
- ✅ Token validation корректна
- ✅ Group sync работает
- ✅ Test coverage >75%

**Время:** 4-5 дней

---

### Итоговый результат Фазы 1:

**После завершения:**
- ✅ MongoDB Event Store + Repositories работают
- ✅ Redis repositories для cache/sessions
- ✅ Event Bus публикует и доставляет events
- ✅ Keycloak OAuth2 интегрирован
- ✅ Notification handlers создают уведомления
- ✅ Полное integration testing

**Оценка времени:** 3-4 недели (80-100 часов)
**Следующий шаг:** → Фаза 2 (Interface Layer)

---

## 🌐 Фаза 2: INTERFACE LAYER (Недели 7-10)

### Приоритет: 🟡 HIGH
### Цель: HTTP API, WebSocket, Middleware
### Оценка: 3-4 недели (80-100 часов)

---

### Milestone 2.1: HTTP Infrastructure (1 неделя)

#### Task 2.1.1: Echo Framework Setup 🔴

**Задача:**
Настроить Echo v4 router с middleware.

**Компоненты:**

1. **Router Setup**
   ```go
   // internal/handler/http/router.go

   func NewRouter(
       authHandler *AuthHandler,
       chatHandler *ChatHandler,
       messageHandler *MessageHandler,
       // ... other handlers
   ) *echo.Echo {
       e := echo.New()

       // Middleware
       e.Use(middleware.Logger())
       e.Use(middleware.Recover())
       e.Use(middleware.CORS())
       e.Use(customMiddleware.RequestID())

       // Public routes
       e.GET("/health", healthHandler)
       e.GET("/metrics", metricsHandler)

       // Auth routes
       auth := e.Group("/auth")
       auth.GET("/login", authHandler.Login)
       auth.GET("/callback", authHandler.Callback)
       auth.POST("/logout", authHandler.Logout)

       // Protected routes
       api := e.Group("/api/v1")
       api.Use(authMiddleware.Authenticate())

       // Workspace routes
       workspaces := api.Group("/workspaces")
       workspaces.POST("", workspaceHandler.Create)
       workspaces.GET("/:id", workspaceHandler.Get)
       workspaces.Use("/:id/*", workspaceMiddleware.CheckAccess())

       // Chat routes (workspace-scoped)
       chats := api.Group("/workspaces/:workspaceId/chats")
       chats.POST("", chatHandler.Create)
       chats.GET("", chatHandler.List)
       chats.GET("/:chatId", chatHandler.Get)
       chats.POST("/:chatId/participants", chatHandler.AddParticipant)

       // Message routes
       messages := api.Group("/chats/:chatId/messages")
       messages.POST("", messageHandler.Send)
       messages.GET("", messageHandler.List)
       messages.PUT("/:messageId", messageHandler.Edit)
       messages.DELETE("/:messageId", messageHandler.Delete)

       // ... other routes

       return e
   }
   ```

2. **Response Helpers**
   ```go
   // internal/handler/http/response.go

   func respondJSON(c echo.Context, status int, data interface{}) error
   func respondError(c echo.Context, err error) error
   func respondValidationError(c echo.Context, err error) error
   ```

3. **Request Helpers**
   ```go
   // internal/handler/http/request.go

   func getUserID(c echo.Context) uuid.UUID
   func getWorkspaceID(c echo.Context) uuid.UUID
   func getCorrelationID(c echo.Context) string
   func bindAndValidate(c echo.Context, req interface{}) error
   ```

**Файлы:**
```
internal/handler/http/
├── router.go
├── response.go
├── request.go
└── router_test.go
```

**Критерии успеха:**
- ✅ Router настроен и работает
- ✅ Middleware chain корректен
- ✅ Helper functions покрыты тестами

**Время:** 1-2 дня

---

#### Task 2.1.2: Middleware Implementation 🔴

**Задача:**
Реализовать middleware для auth, authorization, rate limiting, logging.

**Middleware:**

1. **Auth Middleware**
   ```go
   // internal/middleware/auth.go

   type AuthMiddleware struct {
       tokenValidator TokenValidator
       userRepo       UserRepository
   }

   func (m *AuthMiddleware) Authenticate() echo.MiddlewareFunc {
       return func(next echo.HandlerFunc) echo.HandlerFunc {
           return func(c echo.Context) error {
               // Extract token from Authorization header or cookie
               token := extractToken(c)

               // Validate token
               claims, err := m.tokenValidator.Validate(token)
               if err != nil {
                   return echo.NewHTTPError(401, "Unauthorized")
               }

               // Load user
               user, err := m.userRepo.FindByID(claims.UserID)
               if err != nil {
                   return echo.NewHTTPError(401, "User not found")
               }

               // Set context
               ctx := shared.WithUserID(c.Request().Context(), user.ID)
               c.SetRequest(c.Request().WithContext(ctx))

               return next(c)
           }
       }
   }
   ```

2. **Workspace Access Middleware**
   ```go
   // internal/middleware/workspace.go

   func (m *WorkspaceMiddleware) CheckAccess() echo.MiddlewareFunc {
       return func(next echo.HandlerFunc) echo.HandlerFunc {
           return func(c echo.Context) error {
               workspaceID := parseUUID(c.Param("workspaceId"))
               userID := getUserID(c)

               // Check membership
               isMember, err := m.workspaceRepo.IsMember(c.Request().Context(), workspaceID, userID)
               if err != nil || !isMember {
                   return echo.NewHTTPError(403, "Access denied")
               }

               // Set workspace in context
               ctx := shared.WithWorkspaceID(c.Request().Context(), workspaceID)
               c.SetRequest(c.Request().WithContext(ctx))

               return next(c)
           }
       }
   }
   ```

3. **Chat Access Middleware**
   ```go
   // internal/middleware/chat.go

   func (m *ChatMiddleware) CheckAccess() echo.MiddlewareFunc {
       // Similar to workspace, check participant status
   }
   ```

4. **Rate Limiting Middleware**
   ```go
   // internal/middleware/ratelimit.go

   type RateLimiter struct {
       redis *redis.Client
   }

   func (m *RateLimiter) Limit(max int, window time.Duration) echo.MiddlewareFunc {
       // Token bucket or sliding window algorithm
       // Per-user rate limiting
   }
   ```

5. **Request ID Middleware**
   ```go
   // internal/middleware/requestid.go

   func RequestID() echo.MiddlewareFunc {
       return func(next echo.HandlerFunc) echo.HandlerFunc {
           return func(c echo.Context) error {
               requestID := c.Request().Header.Get("X-Request-ID")
               if requestID == "" {
                   requestID = uuid.New().String()
               }

               c.Response().Header().Set("X-Request-ID", requestID)
               ctx := shared.WithCorrelationID(c.Request().Context(), requestID)
               c.SetRequest(c.Request().WithContext(ctx))

               return next(c)
           }
       }
   }
   ```

6. **Logging Middleware**
   ```go
   // internal/middleware/logging.go

   func Logging(logger *slog.Logger) echo.MiddlewareFunc {
       return func(next echo.HandlerFunc) echo.HandlerFunc {
           return func(c echo.Context) error {
               start := time.Now()

               err := next(c)

               logger.Info("HTTP request",
                   "method", c.Request().Method,
                   "path", c.Request().URL.Path,
                   "status", c.Response().Status,
                   "duration_ms", time.Since(start).Milliseconds(),
                   "request_id", c.Response().Header().Get("X-Request-ID"),
               )

               return err
           }
       }
   }
   ```

**Файлы:**
```
internal/middleware/
├── auth.go
├── auth_test.go
├── workspace.go
├── workspace_test.go
├── chat.go
├── chat_test.go
├── ratelimit.go
├── ratelimit_test.go
├── requestid.go
├── logging.go
└── cors.go
```

**Критерии успеха:**
- ✅ Auth middleware проверяет JWT
- ✅ Authorization middleware защищает ресурсы
- ✅ Rate limiting предотвращает abuse
- ✅ Logging логирует все запросы
- ✅ Test coverage >80%

**Время:** 3-4 дня

---

### Milestone 2.2: HTTP Handlers (2 недели)

#### Task 2.2.1-7: Handler Implementation 🔴

**Задача:**
Реализовать HTTP handlers для всех use cases.

**Handlers:**

1. **AuthHandler** (`internal/handler/http/auth_handler.go`)
   ```
   POST /auth/login       → Redirect to Keycloak
   GET  /auth/callback    → Exchange code, set session
   POST /auth/logout      → Revoke token, clear session
   GET  /auth/me          → Get current user info
   ```

2. **WorkspaceHandler** (`internal/handler/http/workspace_handler.go`)
   ```
   POST   /workspaces                → CreateWorkspace
   GET    /workspaces                → ListUserWorkspaces
   GET    /workspaces/:id            → GetWorkspace
   PUT    /workspaces/:id            → UpdateWorkspace
   POST   /workspaces/:id/invites    → CreateInvite
   POST   /invites/:token/accept     → AcceptInvite
   DELETE /invites/:id               → RevokeInvite
   ```

3. **ChatHandler** (`internal/handler/http/chat_handler.go`)
   ```
   POST   /workspaces/:wid/chats          → CreateChat
   GET    /workspaces/:wid/chats          → ListChats (with filters)
   GET    /chats/:id                      → GetChat
   POST   /chats/:id/participants         → AddParticipant
   DELETE /chats/:id/participants/:userId → RemoveParticipant
   PUT    /chats/:id/name                 → RenameChat
   POST   /chats/:id/convert              → ConvertToTask/Bug/Epic
   PUT    /chats/:id/status               → ChangeStatus
   PUT    /chats/:id/assignee             → AssignUser
   PUT    /chats/:id/priority             → SetPriority
   PUT    /chats/:id/due-date             → SetDueDate
   PUT    /chats/:id/severity             → SetSeverity
   ```

4. **MessageHandler** (`internal/handler/http/message_handler.go`)
   ```
   POST   /chats/:chatId/messages         → SendMessage
   GET    /chats/:chatId/messages         → ListMessages (pagination)
   GET    /messages/:id                   → GetMessage
   GET    /messages/:id/thread            → GetThread
   PUT    /messages/:id                   → EditMessage
   DELETE /messages/:id                   → DeleteMessage
   POST   /messages/:id/reactions         → AddReaction
   DELETE /messages/:id/reactions/:emoji  → RemoveReaction
   POST   /messages/:id/attachments       → AddAttachment
   ```

5. **TaskHandler** (`internal/handler/http/task_handler.go`)
   ```
   GET /workspaces/:wid/tasks       → ListTasks (filters)
   GET /workspaces/:wid/board       → GetKanbanBoard
   GET /tasks/:id                   → GetTask (via GetChat)
   ```

6. **NotificationHandler** (`internal/handler/http/notification_handler.go`)
   ```
   GET    /notifications          → ListNotifications
   GET    /notifications/unread   → CountUnread
   PUT    /notifications/:id/read → MarkAsRead
   PUT    /notifications/read-all → MarkAllAsRead
   DELETE /notifications/:id      → DeleteNotification
   ```

7. **HealthHandler** (`internal/handler/http/health_handler.go`)
   ```
   GET /health   → Health check (MongoDB, Redis, Keycloak)
   GET /metrics  → Prometheus metrics
   ```

**Паттерн реализации:**

```go
type ChatHandler struct {
    createChatUC    CreateChatUseCase
    getChatUC       GetChatUseCase
    listChatsUC     ListChatsUseCase
    addParticipantUC AddParticipantUseCase
    // ... other use cases
}

func (h *ChatHandler) Create(c echo.Context) error {
    // 1. Parse request
    var req CreateChatRequest
    if err := bindAndValidate(c, &req); err != nil {
        return respondValidationError(c, err)
    }

    // 2. Build command from request
    cmd := chat.CreateChatCommand{
        WorkspaceID: getWorkspaceID(c),
        Type:        req.Type,
        Title:       req.Title,
        IsPublic:    req.IsPublic,
        CreatedBy:   getUserID(c),
    }

    // 3. Execute use case
    result, err := h.createChatUC.Execute(c.Request().Context(), cmd)
    if err != nil {
        return respondError(c, err)
    }

    // 4. Convert result to response DTO
    response := CreateChatResponse{
        ChatID:    result.ChatID,
        Type:      result.Type,
        CreatedAt: result.CreatedAt,
    }

    return respondJSON(c, http.StatusCreated, response)
}
```

**DTOs:**
```
internal/handler/http/dto/
├── auth_dto.go
├── workspace_dto.go
├── chat_dto.go
├── message_dto.go
├── notification_dto.go
└── common_dto.go  (pagination, errors)
```

**Тестирование:**
- Unit tests с mock use cases
- Integration tests с real dependencies
- E2E tests через HTTP client

**Файлы:**
```
internal/handler/http/
├── auth_handler.go
├── auth_handler_test.go
├── workspace_handler.go
├── workspace_handler_test.go
├── chat_handler.go
├── chat_handler_test.go
├── message_handler.go
├── message_handler_test.go
├── task_handler.go
├── task_handler_test.go
├── notification_handler.go
├── notification_handler_test.go
├── health_handler.go
└── dto/
```

**Критерии успеха:**
- ✅ Все endpoints реализованы
- ✅ Request/Response validation работает
- ✅ Error handling корректен
- ✅ Test coverage >75%
- ✅ OpenAPI spec актуален (optional)

**Время:** 8-10 дней

---

### Milestone 2.3: WebSocket Implementation (1 неделя)

#### Task 2.3.1: WebSocket Server 🟡

**Задача:**
Реализовать WebSocket для real-time обновлений.

**Компоненты:**

1. **WebSocket Hub**
   ```go
   // internal/infrastructure/websocket/hub.go

   type Hub struct {
       clients    map[*Client]bool
       chatRooms  map[uuid.UUID]map[*Client]bool  // chatID → clients
       register   chan *Client
       unregister chan *Client
       broadcast  chan *Message
   }

   func (h *Hub) Run() {
       for {
           select {
           case client := <-h.register:
               h.clients[client] = true
           case client := <-h.unregister:
               delete(h.clients, client)
               close(client.send)
           case message := <-h.broadcast:
               h.broadcastToChat(message)
           }
       }
   }

   func (h *Hub) BroadcastToChat(chatID uuid.UUID, message interface{}) {
       // Send to all clients in chat room
   }

   func (h *Hub) SendToUser(userID uuid.UUID, message interface{}) {
       // Send to specific user's connections
   }
   ```

2. **WebSocket Client**
   ```go
   // internal/infrastructure/websocket/client.go

   type Client struct {
       hub      *Hub
       conn     *websocket.Conn
       send     chan []byte
       userID   uuid.UUID
       chatIDs  []uuid.UUID  // subscribed chats
   }

   func (c *Client) readPump() {
       // Read messages from WebSocket
       // Handle: subscribe.chat, chat.typing, ping
   }

   func (c *Client) writePump() {
       // Write messages to WebSocket
   }
   ```

3. **WebSocket Handler**
   ```go
   // internal/handler/websocket/handler.go

   type Handler struct {
       hub            *Hub
       tokenValidator TokenValidator
   }

   func (h *Handler) ServeWS(c echo.Context) error {
       // Upgrade HTTP to WebSocket
       conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)

       // Authenticate via token query param
       token := c.QueryParam("token")
       claims, err := h.tokenValidator.Validate(token)

       // Create client
       client := &Client{
           hub:    h.hub,
           conn:   conn,
           send:   make(chan []byte, 256),
           userID: claims.UserID,
       }

       // Register and start pumps
       h.hub.register <- client
       go client.writePump()
       go client.readPump()

       return nil
   }
   ```

4. **Message Router**
   ```go
   // internal/handler/websocket/message_handler.go

   type MessageHandler struct {
       hub *Hub
   }

   func (h *MessageHandler) Handle(client *Client, msg *WSMessage) error {
       switch msg.Type {
       case "subscribe.chat":
           // Add client to chat room
       case "unsubscribe.chat":
           // Remove client from chat room
       case "chat.typing":
           // Broadcast typing indicator
       case "ping":
           // Respond with pong
       }
   }
   ```

5. **Event Broadcaster**
   ```go
   // Subscribe to Event Bus and broadcast to WebSocket clients

   func (h *EventBroadcaster) Start() {
       h.eventBus.Subscribe("events.MessagePosted", h)
       h.eventBus.Subscribe("events.StatusChanged", h)
       h.eventBus.Subscribe("events.NotificationCreated", h)
   }

   func (h *EventBroadcaster) Handle(ctx, event DomainEvent) error {
       switch e := event.(type) {
       case *MessagePosted:
           h.hub.BroadcastToChat(e.ChatID, WSMessage{
               Type: "chat.message.posted",
               Data: e,
           })
       case *StatusChanged:
           h.hub.BroadcastToChat(e.ChatID, WSMessage{
               Type: "task.updated",
               Data: e,
           })
       case *NotificationCreated:
           h.hub.SendToUser(e.UserID, WSMessage{
               Type: "notification.new",
               Data: e,
           })
       }
   }
   ```

**WebSocket Message Types:**

Client → Server:
- `subscribe.chat` - join chat room
- `unsubscribe.chat` - leave chat room
- `chat.typing` - typing indicator
- `ping` - keepalive

Server → Client:
- `chat.message.posted` - new message
- `chat.message.edited` - message edited
- `chat.message.deleted` - message deleted
- `chat.reaction.added` - reaction added
- `task.updated` - task status/priority changed
- `notification.new` - new notification
- `pong` - keepalive response

**Тестирование:**
- Unit tests для Hub, Client
- Integration tests с WebSocket client
- Load tests (100+ concurrent connections)
- Reconnection tests

**Файлы:**
```
internal/infrastructure/websocket/
├── hub.go
├── hub_test.go
├── client.go
├── client_test.go
└── message.go

internal/handler/websocket/
├── handler.go
├── handler_test.go
├── message_handler.go
└── event_broadcaster.go
```

**Критерии успеха:**
- ✅ WebSocket connections работают
- ✅ Real-time broadcasts доставляются
- ✅ Auth через token работает
- ✅ Graceful disconnect handling
- ✅ Support 100+ concurrent connections

**Время:** 5-6 дней

---

### Итоговый результат Фазы 2:

**После завершения:**
- ✅ REST API полностью функционален
- ✅ WebSocket real-time обновления работают
- ✅ Middleware защищают endpoints
- ✅ Rate limiting активен
- ✅ Structured logging настроен

**Оценка времени:** 3-4 недели (80-100 часов)
**Следующий шаг:** → Фаза 3 (Entry Points & DI)

---

## 🚀 Фаза 3: ENTRY POINTS & DEPENDENCY INJECTION (Недели 11-12)

### Приоритет: 🔴 CRITICAL
### Цель: Собрать приложение воедино, запустить
### Оценка: 1-2 недели (30-40 часов)

---

### Milestone 3.1: Application Entry Points

#### Task 3.1.1: API Server (cmd/api/main.go) 🔴

**Задача:**
Собрать все компоненты и запустить HTTP/WebSocket server.

**Компоненты:**

```go
// cmd/api/main.go

func main() {
    // 1. Load configuration
    cfg, err := config.Load("configs/config.yaml")
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }

    // 2. Initialize logger
    logger := logger.New(cfg.Log.Level)

    // 3. Connect to MongoDB
    mongoClient, err := mongodb.Connect(cfg.MongoDB)
    if err != nil {
        logger.Error("Failed to connect to MongoDB", "error", err)
        os.Exit(1)
    }
    defer mongoClient.Disconnect()

    // 4. Connect to Redis
    redisClient := redis.NewClient(&redis.Options{
        Addr: cfg.Redis.Addr,
    })
    defer redisClient.Close()

    // 5. Initialize Keycloak client
    keycloakClient := keycloak.NewHTTPClient(cfg.Keycloak)

    // 6. Initialize infrastructure
    eventStore := eventstore.NewMongoEventStore(mongoClient)
    eventBus := eventbus.NewRedisEventBus(redisClient)

    // 7. Initialize repositories
    chatRepo := mongodb.NewChatRepository(mongoClient, eventStore)
    messageRepo := mongodb.NewMessageRepository(mongoClient)
    userRepo := mongodb.NewUserRepository(mongoClient)
    workspaceRepo := mongodb.NewWorkspaceRepository(mongoClient)
    notificationRepo := mongodb.NewNotificationRepository(mongoClient)

    sessionRepo := redisrepo.NewSessionRepository(redisClient)

    // 8. Initialize use cases (Chat domain example)
    createChatUC := chat.NewCreateChatUseCase(chatRepo, eventStore, userRepo)
    getChatUC := chat.NewGetChatUseCase(chatRepo)
    listChatsUC := chat.NewListChatsUseCase(chatRepo)
    // ... all other use cases

    // 9. Initialize event handlers
    notificationHandler := eventhandler.NewNotificationHandler(
        notification.NewCreateNotificationUseCase(notificationRepo, eventStore),
    )
    eventBus.Subscribe("events.ChatCreated", notificationHandler)
    eventBus.Subscribe("events.StatusChanged", notificationHandler)
    // ... other subscriptions

    // 10. Initialize WebSocket hub
    wsHub := websocket.NewHub()
    go wsHub.Run()

    eventBroadcaster := websocket.NewEventBroadcaster(wsHub, eventBus)
    go eventBroadcaster.Start()

    // 11. Initialize HTTP handlers
    authHandler := httphandler.NewAuthHandler(keycloakClient, sessionRepo)
    chatHandler := httphandler.NewChatHandler(
        createChatUC, getChatUC, listChatsUC, /* ... */
    )
    messageHandler := httphandler.NewMessageHandler(/* ... */)
    workspaceHandler := httphandler.NewWorkspaceHandler(/* ... */)
    notificationHandler := httphandler.NewNotificationHandler(/* ... */)

    wsHandler := wshandler.NewHandler(wsHub, keycloakClient)

    // 12. Initialize middleware
    authMiddleware := middleware.NewAuthMiddleware(keycloakClient, userRepo)
    workspaceMiddleware := middleware.NewWorkspaceMiddleware(workspaceRepo)
    rateLimiter := middleware.NewRateLimiter(redisClient)

    // 13. Setup router
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

    // 14. Start HTTP server
    logger.Info("Starting server", "port", cfg.Server.Port)
    if err := router.Start(":" + cfg.Server.Port); err != nil {
        logger.Error("Server error", "error", err)
    }

    // 15. Graceful shutdown
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

**Критерии успеха:**
- ✅ Приложение запускается
- ✅ Все dependencies корректно инжектятся
- ✅ Health check проходит
- ✅ Graceful shutdown работает

**Время:** 3-4 дня

---

#### Task 3.1.2: Worker Service (cmd/worker/main.go) 🟡

**Задача:**
Background worker для обработки событий и фоновых задач.

**Компоненты:**

```go
// cmd/worker/main.go

func main() {
    // Similar setup as API server
    // but instead of HTTP:

    // Initialize event handlers
    tagProcessorHandler := eventhandler.NewTagProcessorHandler(/* ... */)
    notificationHandler := eventhandler.NewNotificationHandler(/* ... */)
    projectionHandler := eventhandler.NewProjectionHandler(/* ... */)

    // Subscribe to events
    eventBus.Subscribe("events.MessagePosted", tagProcessorHandler)
    eventBus.Subscribe("events.ChatCreated", notificationHandler)
    eventBus.Subscribe("events.StatusChanged", projectionHandler)
    // ... more subscriptions

    // Run worker loop
    logger.Info("Worker started")

    // Wait for shutdown signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    <-quit

    // Graceful shutdown
    logger.Info("Worker shutting down...")
    eventBus.Shutdown()
    logger.Info("Worker stopped")
}
```

**Критерии успеха:**
- ✅ Worker обрабатывает events
- ✅ Handlers выполняются корректно
- ✅ Graceful shutdown работает

**Время:** 1-2 дня

---

#### Task 3.1.3: Database Migrator (cmd/migrator/main.go) 🟡

**Задача:**
Утилита для применения MongoDB миграций.

**Компоненты:**

```go
// cmd/migrator/main.go

func main() {
    // Load config
    // Connect to MongoDB

    // Apply migrations
    migrator := migration.NewMigrator(mongoClient)

    if err := migrator.Up(); err != nil {
        log.Fatal("Migration failed:", err)
    }

    log.Println("Migrations applied successfully")
}
```

**Миграции:**
```javascript
// migrations/mongodb/001_initial_schema.js

db.createCollection("events");
db.events.createIndex({ aggregate_id: 1, version: 1 }, { unique: true });

db.createCollection("chat_read_model");
db.chat_read_model.createIndex({ workspace_id: 1, type: 1 });

db.createCollection("messages");
db.messages.createIndex({ chat_id: 1, created_at: -1 });

// ... all collections and indexes
```

**Критерии успеха:**
- ✅ Миграции применяются
- ✅ Indexes создаются
- ✅ Rollback поддерживается (optional)

**Время:** 1-2 дня

---

### Milestone 3.2: Configuration Management

#### Task 3.2.1: Config Loader 🟡

**Задача:**
Загрузка конфигурации из yaml + env variables.

**Компоненты:**

```go
// internal/config/config.go

type Config struct {
    Server   ServerConfig
    MongoDB  MongoDBConfig
    Redis    RedisConfig
    Keycloak KeycloakConfig
    Auth     AuthConfig
    EventBus EventBusConfig
    Log      LogConfig
    WebSocket WebSocketConfig
}

func Load(path string) (*Config, error) {
    // Load from yaml
    // Override with env variables
    // Validate
}
```

**Env variables:**
```bash
APP_SERVER_PORT=8080
APP_MONGODB_URI=mongodb://localhost:27017
APP_REDIS_ADDR=localhost:6379
APP_KEYCLOAK_URL=http://localhost:8090
APP_LOG_LEVEL=info
```

**Критерии успеха:**
- ✅ Yaml config загружается
- ✅ Env variables переопределяют yaml
- ✅ Validation работает

**Время:** 1 день

---

### Итоговый результат Фазы 3:

**После завершения:**
- ✅ Приложение запускается командой `./api`
- ✅ Worker обрабатывает events
- ✅ Миграции применяются автоматически
- ✅ Configuration управляема
- ✅ Graceful shutdown работает

**Оценка времени:** 1-2 недели (30-40 часов)
**Следующий шаг:** → Фаза 4 (Minimal Frontend)

---

## 🎨 Фаза 4: MINIMAL FRONTEND (Недели 13-16)

### Приоритет: 🟡 MEDIUM
### Цель: HTMX + Pico CSS для базового UI
### Оценка: 2-3 недели (50-60 часов)

---

### Milestone 4.1: Base Templates (1 неделя)

#### Task 4.1.1: Layout & Components 🟡

**Задача:**
Создать base layout и переиспользуемые компоненты.

**Файлы:**

```html
<!-- web/templates/layout.html -->
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{ .Title }} - Flowra</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css">
    <script src="https://unpkg.com/htmx.org@2.0.0"></script>
    <script src="https://unpkg.com/htmx-ext-ws@2.0.0/ws.js"></script>
</head>
<body>
    {{ template "navbar" . }}

    <main class="container">
        {{ template "content" . }}
    </main>

    {{ template "footer" . }}
</body>
</html>
```

**Components:**
- `web/components/navbar.html` - Navigation bar
- `web/components/chat_list.html` - Chat list sidebar
- `web/components/message.html` - Single message card
- `web/components/task_card.html` - Kanban card
- `web/components/notification_dropdown.html` - Notifications

**Критерии успеха:**
- ✅ Pico CSS загружается
- ✅ HTMX работает
- ✅ Layout рендерится корректно

**Время:** 2-3 дня

---

### Milestone 4.2: Core Pages (2 недели)

#### Task 4.2.1: Authentication Pages 🟡

```html
<!-- web/templates/auth/login.html -->
<div class="login-page">
    <h1>Welcome to Flowra</h1>
    <a href="/auth/login" role="button">Login with Keycloak</a>
</div>
```

**Время:** 1 день

---

#### Task 4.2.2: Workspace Pages 🟡

```html
<!-- web/templates/workspace/list.html -->
<div class="workspace-list">
    <h1>Your Workspaces</h1>

    {{ range .Workspaces }}
    <article>
        <h3><a href="/workspaces/{{ .ID }}">{{ .Name }}</a></h3>
    </article>
    {{ end }}

    <button hx-get="/workspaces/new" hx-target="#modal">
        Create Workspace
    </button>
</div>
```

**Время:** 2-3 дня

---

#### Task 4.2.3: Chat View 🟡

```html
<!-- web/templates/chat/view.html -->
<div class="chat-view" hx-ext="ws" ws-connect="/ws?token={{ .Token }}">
    <!-- Sidebar: chat list -->
    <aside class="chat-list">
        {{ template "chat_list" . }}
    </aside>

    <!-- Main: messages -->
    <section class="messages" id="messages">
        {{ range .Messages }}
            {{ template "message" . }}
        {{ end }}
    </section>

    <!-- Message input -->
    <form hx-post="/chats/{{ .ChatID }}/messages" hx-target="#messages" hx-swap="beforeend">
        <textarea name="content" placeholder="Type a message... Use #createTask to create tasks"></textarea>
        <button type="submit">Send</button>
    </form>
</div>

<script>
// WebSocket listener
document.body.addEventListener("chat.message.posted", function(e) {
    htmx.ajax("GET", "/chats/{{ .ChatID }}/messages/" + e.detail.messageID, {
        target: "#messages",
        swap: "beforeend"
    });
});
</script>
```

**Время:** 4-5 дней

---

#### Task 4.2.4: Kanban Board 🟡

```html
<!-- web/templates/board/index.html -->
<div class="kanban-board">
    <h1>{{ .WorkspaceName }} - Board</h1>

    {{ range .Columns }}
    <div class="column" data-status="{{ .Status }}">
        <h3>{{ .Title }} ({{ .Count }})</h3>

        <div class="cards"
             hx-post="/tasks/move"
             hx-trigger="drop"
             hx-vals='js:{status: "{{ .Status }}"}'>
            {{ range .Tasks }}
                {{ template "task_card" . }}
            {{ end }}
        </div>
    </div>
    {{ end }}
</div>

<script>
// Drag-n-drop с HTMX
htmx.on("htmx:afterSwap", function(e) {
    if (e.detail.target.classList.contains("cards")) {
        initDragDrop();
    }
});
</script>
```

**Время:** 3-4 дня

---

#### Task 4.2.5: Notifications 🟡

```html
<!-- web/components/notification_dropdown.html -->
<div class="notifications-dropdown">
    <button hx-get="/notifications" hx-target="#notification-list" hx-trigger="click">
        🔔 <span class="badge">{{ .UnreadCount }}</span>
    </button>

    <div id="notification-list" class="dropdown" hidden>
        <!-- Notifications loaded via HTMX -->
    </div>
</div>

<script>
// Real-time notification updates
document.body.addEventListener("notification.new", function(e) {
    htmx.ajax("GET", "/notifications/" + e.detail.notificationID, {
        target: "#notification-list",
        swap: "afterbegin"
    });

    // Update badge count
    updateBadgeCount();
});
</script>
```

**Время:** 2 дня

---

### Milestone 4.3: Static Assets (3 дня)

#### Task 4.3.1: CSS Customization 🟡

```css
/* web/static/css/custom.css */

:root {
    --primary-color: #0066cc;
    --secondary-color: #6c757d;
}

.kanban-board {
    display: flex;
    gap: 1rem;
    overflow-x: auto;
}

.column {
    min-width: 300px;
    background: var(--card-background);
    padding: 1rem;
    border-radius: 8px;
}

.task-card {
    background: white;
    padding: 1rem;
    margin-bottom: 0.5rem;
    border-radius: 4px;
    cursor: grab;
}

.task-card.dragging {
    opacity: 0.5;
}
```

**Время:** 1 день

---

#### Task 4.3.2: JavaScript Utilities 🟡

```javascript
// web/static/js/app.js

// Tag autocomplete
function initTagAutocomplete() {
    const textarea = document.querySelector('textarea[name="content"]');

    textarea.addEventListener('input', function(e) {
        const cursorPos = e.target.selectionStart;
        const text = e.target.value;

        // Detect # at cursor
        if (text[cursorPos - 1] === '#') {
            showTagSuggestions(cursorPos);
        }
    });
}

// Drag-n-drop for kanban
function initDragDrop() {
    const cards = document.querySelectorAll('.task-card');

    cards.forEach(card => {
        card.addEventListener('dragstart', handleDragStart);
        card.addEventListener('dragend', handleDragEnd);
    });

    const columns = document.querySelectorAll('.cards');
    columns.forEach(column => {
        column.addEventListener('dragover', handleDragOver);
        column.addEventListener('drop', handleDrop);
    });
}

// WebSocket connection management
function initWebSocket(token) {
    // HTMX ws extension handles this
    // but we can add custom reconnection logic
}

initTagAutocomplete();
initDragDrop();
```

**Время:** 2 дня

---

### Итоговый результат Фазы 4:

**После завершения:**
- ✅ Работающий UI для всех основных сценариев
- ✅ HTMX обеспечивает динамику без JS фреймворков
- ✅ Pico CSS делает UI чистым и минималистичным
- ✅ WebSocket обновления работают
- ✅ Drag-n-drop на канбане функционален
- ✅ Tag autocomplete помогает пользователям

**Оценка времени:** 2-3 недели (50-60 часов)
**Следующий шаг:** → Фаза 5 (Testing & QA)

---

## 🧪 Фаза 5: COMPREHENSIVE TESTING & QA (Недели 17-18)

### Приоритет: 🔴 CRITICAL
### Цель: Достичь >80% coverage, E2E tests, bugfixing
### Оценка: 2 недели (40-50 часов)

---

### Milestone 5.1: Test Coverage Improvement

#### Task 5.1.1: Unit Test Coverage 🔴

**Текущее состояние:**
- Domain: 90%+ ✅
- Application: 75% (после Phase 0)
- Infrastructure: ~60% (нужно улучшить)
- Handlers: ~50% (нужно улучшить)

**Задача:**
Довести coverage до >80% везде.

**План:**
1. Infrastructure tests (eventstore, repositories)
2. Handler tests (HTTP, WebSocket)
3. Middleware tests
4. Edge cases и error paths

**Критерии успеха:**
- ✅ Overall coverage >80%
- ✅ Critical paths coverage >90%
- ✅ No untested error handlers

**Время:** 3-4 дня

---

#### Task 5.1.2: Integration Tests 🔴

**Задача:**
E2E тесты для critical user flows.

**Test Scenarios:**

1. **Complete Task Creation Workflow**
   ```
   1. User creates workspace
   2. User creates chat (Discussion)
   3. User sends message with "#createTask Implement feature X"
   4. Tag processor parses tag
   5. CommandExecutor converts chat to Task
   6. Verify chat type changed
   7. Verify events published
   8. Verify notification created
   9. Verify WebSocket broadcast sent
   ```

2. **Messaging Workflow**
   ```
   1. User A creates chat
   2. User A invites User B
   3. User B joins chat
   4. User A sends message
   5. Verify WebSocket delivery to User B
   6. User B adds reaction
   7. Verify reaction persisted
   8. User A edits message
   9. Verify edit reflected
   ```

3. **Workspace Invitation Workflow**
   ```
   1. User A creates workspace
   2. User A creates invite link
   3. User B uses invite link
   4. Verify Keycloak group membership
   5. Verify workspace access granted
   6. User A revokes invite
   7. Verify new users cannot join
   ```

4. **Kanban Board Workflow**
   ```
   1. Create multiple tasks
   2. Drag task to different column
   3. Verify status change persisted
   4. Verify events published
   5. Verify WebSocket broadcast
   ```

**Файлы:**
```
tests/e2e/
├── task_creation_test.go
├── messaging_test.go
├── workspace_invitation_test.go
├── kanban_board_test.go
└── helpers.go
```

**Критерии успеха:**
- ✅ All critical paths покрыты E2E tests
- ✅ Tests проходят на CI/CD
- ✅ WebSocket scenarios протестированы

**Время:** 4-5 дней

---

#### Task 5.1.3: Load & Performance Testing 🟡

**Задача:**
Проверить производительность под нагрузкой.

**Scenarios:**

1. **API Load Test**
   - 100 concurrent users
   - 1000 requests/second
   - Measure: p50, p95, p99 latency

2. **WebSocket Load Test**
   - 100 concurrent WebSocket connections
   - 50 messages/second broadcast
   - Measure: delivery latency

3. **Database Performance**
   - 10000 events append
   - 1000 events load
   - Measure: throughput

**Tools:**
- `k6` для HTTP load testing
- `artillery` для WebSocket testing
- Custom scripts для DB benchmarks

**Критерии успеха:**
- ✅ API p95 latency < 200ms
- ✅ WebSocket latency < 100ms
- ✅ Support 100 concurrent users
- ✅ No memory leaks

**Время:** 2-3 дня

---

### Milestone 5.2: Bug Fixing & Stabilization

#### Task 5.2.1: Bug Triage & Fixing 🔴

**Процесс:**
1. Run all tests, collect failures
2. Categorize bugs (critical, high, medium, low)
3. Fix critical and high priority bugs
4. Re-run tests, verify fixes
5. Regression testing

**Время:** 3-4 дня

---

#### Task 5.2.2: UX Improvements 🟡

**Задача:**
Улучшить user experience на основе testing feedback.

**Improvements:**
- Error messages более информативны
- Loading states для всех async операций
- Empty states для пустых списков
- Keyboard shortcuts (optional)
- Confirmation dialogs для destructive actions

**Время:** 2-3 дня

---

### Итоговый результат Фазы 5:

**После завершения:**
- ✅ Test coverage >80% overall
- ✅ E2E tests покрывают все critical paths
- ✅ Performance requirements выполнены
- ✅ Critical bugs исправлены
- ✅ UX улучшен
- ✅ Приложение стабильно

**Оценка времени:** 2 недели (40-50 часов)
**Следующий шаг:** → Фаза 6 (Deployment & DevOps)

---

## 🚢 Фаза 6: DEPLOYMENT & DEVOPS (Недели 19-20)

### Приоритет: 🟡 HIGH
### Цель: Production-ready deployment
### Оценка: 1-2 недели (30-40 часов)

---

### Milestone 6.1: Docker & CI/CD

#### Task 6.1.1: Dockerfile 🔴

```dockerfile
# Dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o api cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o worker cmd/worker/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o migrator cmd/migrator/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/api .
COPY --from=builder /app/worker .
COPY --from=builder /app/migrator .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/web ./web

EXPOSE 8080
CMD ["./api"]
```

**Время:** 1 день

---

#### Task 6.1.2: Docker Compose для Production 🔴

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - APP_MONGODB_URI=mongodb://mongodb:27017
      - APP_REDIS_ADDR=redis:6379
    depends_on:
      - mongodb
      - redis
      - keycloak
    restart: unless-stopped

  worker:
    build: .
    command: ./worker
    environment:
      - APP_MONGODB_URI=mongodb://mongodb:27017
      - APP_REDIS_ADDR=redis:6379
    depends_on:
      - mongodb
      - redis
    restart: unless-stopped

  mongodb:
    image: mongo:8
    volumes:
      - mongo_data:/data/db
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    restart: unless-stopped

  keycloak:
    image: quay.io/keycloak/keycloak:23
    environment:
      - KEYCLOAK_ADMIN=admin
      - KEYCLOAK_ADMIN_PASSWORD=admin123
    ports:
      - "8090:8080"
    restart: unless-stopped

volumes:
  mongo_data:
```

**Время:** 1 день

---

#### Task 6.1.3: GitHub Actions CI/CD 🔴

```yaml
# .github/workflows/ci.yml
name: CI/CD

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      mongodb:
        image: mongo:8
        ports:
          - 27017:27017
      redis:
        image: redis:7
        ports:
          - 6379:6379

    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.25'

      - name: Cache Go modules
        uses: actions/cache@v3
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}

      - name: Download dependencies
        run: go mod download

      - name: Run linter
        run: golangci-lint run

      - name: Run tests
        run: go test -v -coverprofile=coverage.out ./...

      - name: Check coverage
        run: |
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Coverage: $coverage%"
          if (( $(echo "$coverage < 80" | bc -l) )); then
            echo "Coverage is below 80%"
            exit 1
          fi

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out

  build:
    needs: test
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'

    steps:
      - uses: actions/checkout@v3

      - name: Build Docker image
        run: docker build -t flowra:latest .

      - name: Push to registry
        # ... push to Docker Hub or GitHub Container Registry

  deploy:
    needs: build
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'

    steps:
      - name: Deploy to production
        # ... deploy script
```

**Время:** 2-3 дня

---

### Milestone 6.2: Monitoring & Observability

#### Task 6.2.1: Prometheus Metrics 🟡

```go
// internal/infrastructure/metrics/metrics.go

var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "path", "status"},
    )

    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "http_request_duration_seconds",
            Help: "HTTP request duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path"},
    )

    eventProcessingDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "event_processing_duration_seconds",
            Help: "Event processing duration",
        },
        []string{"event_type"},
    )

    websocketConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "websocket_connections_active",
            Help: "Number of active WebSocket connections",
        },
    )
)

func init() {
    prometheus.MustRegister(
        httpRequestsTotal,
        httpRequestDuration,
        eventProcessingDuration,
        websocketConnections,
    )
}
```

**Endpoint:** `GET /metrics`

**Время:** 2 дня

---

#### Task 6.2.2: Health Checks 🟡

```go
// internal/handler/http/health_handler.go

type HealthHandler struct {
    mongoClient *mongo.Client
    redisClient *redis.Client
    keycloakClient KeycloakClient
}

func (h *HealthHandler) Health(c echo.Context) error {
    ctx := c.Request().Context()

    health := HealthResponse{
        Status: "healthy",
        Checks: make(map[string]CheckResult),
    }

    // MongoDB check
    if err := h.mongoClient.Ping(ctx, nil); err != nil {
        health.Checks["mongodb"] = CheckResult{
            Status: "unhealthy",
            Error: err.Error(),
        }
        health.Status = "degraded"
    } else {
        health.Checks["mongodb"] = CheckResult{Status: "healthy"}
    }

    // Redis check
    if err := h.redisClient.Ping(ctx).Err(); err != nil {
        health.Checks["redis"] = CheckResult{
            Status: "unhealthy",
            Error: err.Error(),
        }
        health.Status = "degraded"
    } else {
        health.Checks["redis"] = CheckResult{Status: "healthy"}
    }

    // Keycloak check
    if err := h.keycloakClient.Health(ctx); err != nil {
        health.Checks["keycloak"] = CheckResult{
            Status: "unhealthy",
            Error: err.Error(),
        }
        health.Status = "degraded"
    } else {
        health.Checks["keycloak"] = CheckResult{Status: "healthy"}
    }

    status := http.StatusOK
    if health.Status == "degraded" {
        status = http.StatusServiceUnavailable
    }

    return c.JSON(status, health)
}
```

**Endpoint:** `GET /health`

**Время:** 1 день

---

#### Task 6.2.3: Structured Logging 🟡

```go
// pkg/logger/logger.go

type Logger struct {
    *slog.Logger
}

func New(level string) *Logger {
    var logLevel slog.Level
    switch level {
    case "debug":
        logLevel = slog.LevelDebug
    case "info":
        logLevel = slog.LevelInfo
    case "warn":
        logLevel = slog.LevelWarn
    case "error":
        logLevel = slog.LevelError
    default:
        logLevel = slog.LevelInfo
    }

    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: logLevel,
    })

    return &Logger{
        Logger: slog.New(handler),
    }
}

// Usage:
logger.Info("HTTP request",
    "method", "POST",
    "path", "/chats/123/messages",
    "duration_ms", 45,
    "user_id", userID,
    "request_id", requestID,
)
```

**Время:** 1 день

---

### Milestone 6.3: Documentation

#### Task 6.3.1: Deployment Guide 🟡

**Создать:**
- `docs/deployment/docker.md` - Docker deployment
- `docs/deployment/kubernetes.md` - K8s deployment (optional)
- `docs/deployment/monitoring.md` - Monitoring setup

**Время:** 1-2 дня

---

#### Task 6.3.2: API Documentation 🟡

**Создать:**
- OpenAPI spec (optional, но рекомендуется)
- Postman collection
- curl examples

**Время:** 1-2 дня

---

### Итоговый результат Фазы 6:

**После завершения:**
- ✅ Docker images собираются
- ✅ CI/CD pipeline работает
- ✅ Automated tests на каждый commit
- ✅ Prometheus metrics экспортируются
- ✅ Health checks работают
- ✅ Structured logging настроен
- ✅ Deployment docs готовы

**Оценка времени:** 1-2 недели (30-40 часов)
**Следующий шаг:** → MVP RELEASE 🎉

---

## 🎉 MVP RELEASE

### Критерии готовности

**Functional Requirements:**
- ✅ Workspace creation and management
- ✅ User invitations via links
- ✅ Chat/Task/Bug creation (UI + tags)
- ✅ Message sending with tag parsing
- ✅ Auto-apply tags (status, assignee, priority)
- ✅ Kanban board with drag-n-drop
- ✅ Real-time updates via WebSocket
- ✅ Notifications on changes

**Non-Functional Requirements:**
- ✅ Response time < 200ms (95th percentile)
- ✅ WebSocket latency < 100ms
- ✅ Support 100 concurrent users
- ✅ Test coverage > 80%
- ✅ Zero downtime deployment capability
- ✅ Data encrypted at rest and in transit

**Technical Requirements:**
- ✅ Domain-driven design implemented
- ✅ Event sourcing works for key aggregates
- ✅ All dependencies через interfaces
- ✅ Comprehensive test suite
- ✅ CI/CD pipeline functional
- ✅ Monitoring and health checks
- ✅ Graceful shutdown

### Release Checklist

- [ ] All tests pass on CI
- [ ] Code review completed
- [ ] Security audit completed
- [ ] Performance benchmarks met
- [ ] Documentation complete
- [ ] Deployment guide tested
- [ ] Backup/restore procedures documented
- [ ] Rollback plan prepared
- [ ] Monitoring dashboards configured
- [ ] User acceptance testing completed

### Post-Release Support

**Week 1-2:**
- Monitor metrics closely
- Fix critical bugs within 24h
- Collect user feedback
- Performance tuning

**Month 1:**
- Regular bug fixes
- UX improvements based on feedback
- Documentation updates
- Feature requests collection

---

## 📈 Фаза 7: POST-MVP OPTIMIZATION (Месяцы 4-6)

### Приоритет: 🟡 MEDIUM
### Цель: Стабилизация, оптимизация, расширение
### Оценка: 2-3 месяца

---

### Milestone 7.1: Performance Optimization

#### Task 7.1.1: Database Optimization 🟡

**Задачи:**
- Query optimization (identify slow queries)
- Index tuning (analyze index usage)
- Connection pooling optimization
- Caching strategy (Redis для read-heavy queries)

**Метрики:**
- Query p95 latency < 50ms
- Index hit rate > 95%
- Cache hit rate > 80%

**Время:** 2 недели

---

#### Task 7.1.2: Event Store Optimization 🟡

**Задачи:**
- Implement Snapshots для больших aggregates
- Event archiving (старые события в холодное хранилище)
- Projection optimization (materialized views)

**Метрики:**
- Load aggregate < 50ms (even with 1000 events)
- Snapshot creation time < 100ms

**Время:** 2 недели

---

#### Task 7.1.3: WebSocket Scaling 🟡

**Задачи:**
- Horizontal scaling с Redis pub/sub
- Connection pooling
- Load balancing strategy

**Метрики:**
- Support 1000+ concurrent connections
- Message delivery < 50ms

**Время:** 1-2 недели

---

### Milestone 7.2: Feature Enhancements

#### Task 7.2.1: Advanced Tag Features 🟡

**Features:**
- Custom tags registration
- Tag aliasing (#s → #status)
- Natural language dates (#due tomorrow)
- Tag validation rules

**Время:** 2-3 недели

---

#### Task 7.2.2: Task Relationships 🟡

**Features:**
- #parent tag для иерархии
- #blocks tag для dependencies
- #relates tag для связей
- Dependency graph visualization

**Время:** 3-4 недели

---

#### Task 7.2.3: Customizable Workflows 🟡

**Features:**
- Custom status models per workspace
- Workflow state machine configuration
- Transition rules и validation

**Время:** 3-4 недели

---

### Milestone 7.3: Analytics & Reporting

#### Task 7.3.1: Metrics Dashboard 🟡

**Metrics:**
- Lead time, cycle time
- Throughput (tasks per week)
- WIP limits tracking
- Burndown charts

**Время:** 2-3 недели

---

#### Task 7.3.2: Advanced Search 🟡

**Features:**
- Full-text search (MongoDB Atlas Search или Elasticsearch)
- Filters по custom tags
- Saved searches
- Search suggestions

**Время:** 2-3 недели

---

### Milestone 7.4: Security Enhancements

#### Task 7.4.1: Advanced RBAC 🟡

**Features:**
- Granular permissions (read/write/admin per resource)
- Role templates
- Permission inheritance
- Audit log для всех actions

**Время:** 2-3 недели

---

#### Task 7.4.2: Security Hardening 🟡

**Features:**
- Rate limiting per endpoint
- IP whitelisting (optional)
- 2FA support (через Keycloak)
- Session management improvements

**Время:** 1-2 недели

---

### Итоговый результат Фазы 7:

**После завершения:**
- ✅ Production стабилен и оптимизирован
- ✅ Расширенные фичи реализованы
- ✅ Analytics и reporting работают
- ✅ Security укреплена
- ✅ Пользовательская база растет

**Оценка времени:** 2-3 месяца (200-300 часов)

---

## 📋 Архитектурные рекомендации

### Принципы разработки

1. **Test-Driven Development**
   - Пишите тесты ДО или ВМЕСТЕ с кодом
   - Minimum coverage: 80%
   - Критические paths: 90%+

2. **Interface-First Design**
   - Declare interfaces on consumer side
   - Implementation never imports consumer
   - Easy mocking and testing

3. **Event-Driven Architecture**
   - Domain events для всех значимых изменений
   - Loose coupling через Event Bus
   - Idempotency для event handlers

4. **Clean Code Principles**
   - SOLID principles
   - DRY (но не преждевременная абстракция)
   - Meaningful naming
   - Short functions (<50 lines)

5. **Security-First**
   - Validate всё на входе
   - Authorization checks везде
   - Rate limiting
   - Input sanitization

### Code Review Guidelines

**Must Have:**
- [ ] Tests added/updated
- [ ] Test coverage не упал
- [ ] Linter проходит
- [ ] Documentation updated
- [ ] No hardcoded secrets
- [ ] Error handling корректен

**Should Have:**
- [ ] Performance не ухудшилась
- [ ] Backward compatibility сохранена
- [ ] Logging добавлен где нужно
- [ ] Metrics instrumented

### Git Workflow

**Branches:**
- `main` - production-ready code
- `develop` - integration branch
- `feature/phase-X-task-name` - feature branches
- `bugfix/issue-description` - bug fixes
- `hotfix/critical-issue` - production hotfixes

**Commits:**
```
feat: Add Chat Query UseCases implementation
^--^ ^---------------------------------^
│    │
│    └─⫸ Summary (imperative, present tense)
│
└──⫸ Type: feat, fix, refactor, test, docs, chore
```

**Pull Requests:**
- Title: `[Phase X] Task description`
- Description template:
  ```
  ## What
  Brief description of changes

  ## Why
  Motivation and context

  ## How
  Implementation approach

  ## Testing
  How to test these changes

  ## Checklist
  - [ ] Tests added
  - [ ] Documentation updated
  - [ ] Linter passes
  ```

---

## 📊 Метрики успеха и KPI

### Development Metrics

**Code Quality:**
- Test coverage: >80% overall, >90% critical paths
- Linter warnings: 0
- Code review approval time: <24h
- Build success rate: >95%

**Velocity:**
- Story points per sprint (после stabilization)
- Bug fix time: <48h для critical, <1 week для high
- Feature delivery: as per roadmap

**Technical Debt:**
- TODO count: decreasing
- Code duplication: <5%
- Cyclomatic complexity: <15 per function

### Product Metrics

**Performance:**
- API p95 latency: <200ms
- WebSocket latency: <100ms
- Database query p95: <50ms
- Error rate: <0.1%

**Reliability:**
- Uptime: >99.9%
- MTTR (Mean Time To Recovery): <1h
- Failed deployments: <5%

**User Engagement (Post-MVP):**
- Daily active users
- Messages per user per day
- Tasks created per week
- User retention (D1, D7, D30)

---

## 🎯 Приоритизация и Trade-offs

### Must Have (Critical для MVP)

1. ✅ Chat UseCases testing (Phase 0)
2. ✅ Infrastructure layer (Phase 1)
3. ✅ HTTP API handlers (Phase 2)
4. ✅ Entry points (Phase 3)
5. ✅ Minimal frontend (Phase 4)
6. ✅ Testing & QA (Phase 5)
7. ✅ Deployment (Phase 6)

### Should Have (Важно, но можно отложить)

1. WebSocket real-time updates
2. Keycloak OAuth2 integration
3. Rate limiting
4. Notification system
5. E2E tests

### Could Have (Nice to have)

1. Snapshots для Event Store
2. Custom tag features
3. Task relationships
4. Analytics dashboard
5. Advanced search

### Won't Have (V2 features)

1. Mobile app
2. Email notifications
3. Multi-tenancy
4. CQRS read replicas
5. Advanced RBAC

---

## 📅 Timeline Summary

```
┌──────────────────────────────────────────────────────────────────┐
│ PHASE 0: Critical Fixes                    │  1-2 days           │
├──────────────────────────────────────────────────────────────────┤
│ PHASE 1: Infrastructure Layer              │  3-4 weeks          │
├──────────────────────────────────────────────────────────────────┤
│ PHASE 2: Interface Layer                   │  3-4 weeks          │
├──────────────────────────────────────────────────────────────────┤
│ PHASE 3: Entry Points & DI                 │  1-2 weeks          │
├──────────────────────────────────────────────────────────────────┤
│ PHASE 4: Minimal Frontend                  │  2-3 weeks          │
├──────────────────────────────────────────────────────────────────┤
│ PHASE 5: Testing & QA                      │  2 weeks            │
├──────────────────────────────────────────────────────────────────┤
│ PHASE 6: Deployment & DevOps               │  1-2 weeks          │
├══════════════════════════════════════════════════════════════════┤
│ 🎉 MVP RELEASE                             │  Week 20            │
├──────────────────────────────────────────────────────────────────┤
│ PHASE 7: Post-MVP Optimization             │  2-3 months         │
└──────────────────────────────────────────────────────────────────┘

Total time to MVP: ~12-20 weeks (3-5 months)
Total time to Stable V1: ~6 months
```

---

## 🚨 Risks & Mitigation

### Technical Risks

1. **Event Store Performance at Scale** 🔴
   - Risk: Slow aggregate loading с большим количеством events
   - Mitigation: Implement snapshots early, monitor performance
   - Contingency: Optimize event replay, consider CQRS read models

2. **WebSocket Scalability** 🟡
   - Risk: Connection limits, memory usage
   - Mitigation: Horizontal scaling с Redis pub/sub
   - Contingency: Fallback to polling

3. **MongoDB Version Compatibility** 🟡
   - Risk: Go Driver v2 breaking changes
   - Mitigation: Comprehensive testing, version pinning
   - Contingency: Rollback to v1 if needed

4. **Keycloak Integration Complexity** 🟡
   - Risk: OAuth2 flow issues, group sync problems
   - Mitigation: Early integration tests, mock Keycloak для dev
   - Contingency: Simplified auth для MVP

### Project Risks

1. **Scope Creep** 🟡
   - Risk: Too many features, delayed release
   - Mitigation: Strict prioritization, MVP-first mindset
   - Contingency: Cut "should have" features

2. **Test Coverage Debt** 🔴
   - Risk: Rushed implementation без тестов
   - Mitigation: TDD approach, coverage gates на CI
   - Contingency: Testing sprint перед release

3. **Performance Issues in Production** 🟡
   - Risk: Не обнаруженные bottlenecks
   - Mitigation: Load testing перед release
   - Contingency: Quick performance patches, scaling

---

## ✅ Definition of Done

### For Each Task

- [ ] Code написан и работает
- [ ] Unit tests покрывают логику (>80%)
- [ ] Integration tests (где применимо)
- [ ] Code review approved
- [ ] Documentation updated
- [ ] Linter проходит (0 warnings)
- [ ] No critical security issues
- [ ] Tested locally

### For Each Phase

- [ ] All tasks completed
- [ ] Phase objectives met
- [ ] Test coverage target достигнут
- [ ] Documentation complete
- [ ] Demo готова
- [ ] Sign-off от stakeholder

### For MVP Release

- [ ] All functional requirements met
- [ ] All non-functional requirements met
- [ ] Test coverage >80%
- [ ] Security audit completed
- [ ] Performance benchmarks met
- [ ] Deployment tested
- [ ] Documentation complete
- [ ] User acceptance testing passed

---

## 📞 Support & Resources

### Documentation References

- **Project Docs:** `/docs/`
- **Architecture:** `/docs/01-architecture.md`
- **MVP Roadmap:** `/docs/08-mvp-roadmap.md`
- **Progress Tracker:** `/docs/tasks/04-impl-usecase/PROGRESS_TRACKER.md`
- **Completion Plan:** `/docs/tasks/04-impl-usecase/COMPLETION_PLAN.md`

### Code References

- **Domain Layer:** `internal/domain/`
- **Application Layer:** `internal/application/`
- **Infrastructure:** `internal/infrastructure/`
- **Test Examples:** `tests/`

### Useful Commands

```bash
# Development
make dev                     # Start dev server
make docker-up               # Start infrastructure
make test                    # Run all tests
make lint                    # Run linter

# Testing
make test-unit               # Unit tests only
make test-integration        # Integration tests
make test-coverage           # Coverage report
make test-coverage-check     # Verify >80%

# Build
make build                   # Build all binaries
make clean                   # Clean artifacts

# Database
./migrator                   # Run migrations
```

---

## 🎓 Learning & Best Practices

### Recommended Reading

1. **Domain-Driven Design** - Eric Evans
2. **Implementing Domain-Driven Design** - Vaughn Vernon
3. **Building Microservices** - Sam Newman
4. **Release It!** - Michael Nygard

### Go Best Practices

- [Effective Go](https://golang.org/doc/effective_go)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### Architecture Patterns

- Event Sourcing: [Martin Fowler](https://martinfowler.com/eaaDev/EventSourcing.html)
- CQRS: [Greg Young](https://cqrs.files.wordpress.com/2010/11/cqrs_documents.pdf)
- DDD: [Domain-Driven Design Reference](https://www.domainlanguage.com/ddd/reference/)

---

## 📝 Change Log

| Date       | Version | Author  | Changes                              |
|------------|---------|---------|--------------------------------------|
| 2025-11-11 | 1.0     | Claude  | Initial roadmap creation             |

---

## 🏁 Conclusion

Этот план развития представляет собой **детальную дорожную карту** от текущего состояния (82% Application Layer) до полностью функционального MVP и далее.

**Ключевые принципы:**
1. ✅ **Завершить текущее** перед началом нового (Phase 0 критичен)
2. 🏗️ **Infrastructure → Interface → Entry Points** (inside-out approach)
3. 🎨 **Minimal viable frontend** для быстрого feedback
4. 🧪 **Testing throughout** (не оставлять на конец)
5. 🚀 **Iterative delivery** (каждая фаза даёт ценность)

**Timeline:**
- MVP за **12-20 недель** при фокусированной работе
- Stable V1 за **6 месяцев** с пост-релиз оптимизацией

**Следующий шаг:**
🔴 **Начать с Phase 0 (Critical Fixes)** - Chat UseCases testing и Query implementation.

---

**Good luck! 🚀**
