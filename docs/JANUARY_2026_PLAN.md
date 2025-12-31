# Детальный план разработки — Январь 2026

**Дата создания:** 2025-12-31  
**Период:** 1 января - 31 января 2026  
**Цель:** Завершить Infrastructure Layer и начать Interface Layer  
**Владелец:** Project Lead

---

## 📊 Общий обзор

### Цели на январь

1. ✅ Завершить Infrastructure Layer (MongoDB repositories, indexes)
2. ✅ Реализовать базовый Event Bus
3. ✅ Начать Interface Layer (HTTP handlers, middleware)
4. ✅ Подготовить к запуску первого работающего API

### Ожидаемые результаты к концу января

- ✅ Все MongoDB repositories работают (включая Task)
- ✅ MongoDB indexes созданы для всех коллекций
- ✅ Event Bus публикует события асинхронно
- ✅ HTTP Infrastructure настроена (Echo, middleware)
- ✅ Первые HTTP handlers реализованы
- ⚠️ Entry points в разработке (cmd/api/main.go)

**Прогресс к концу месяца:** ~75% от MVP

---

## 🗓️ Недельный план

### Неделя 1: 1-7 января — Infrastructure Completion

**Приоритет:** 🔴 КРИТИЧЕСКИЙ  
**Цель:** Завершить все MongoDB repositories и indexes

#### День 1-2 (1-2 января): Task Repository

**Задача:** Реализовать Task Repository с Event Sourcing

**Файлы:**
```
internal/infrastructure/repository/mongodb/
├── task_repository.go           (новый)
└── task_repository_test.go      (новый)
```

**Детали реализации:**

1. **Task Repository структура** (~300 LOC)
   ```go
   type MongoTaskRepository struct {
       eventStore *eventstore.MongoEventStore
       db         *mongo.Database
       collection *mongo.Collection  // read model: "tasks"
   }
   ```

2. **Методы (аналогично ChatRepository):**
   - `Save(ctx, task)` — сохранить события + обновить read model
   - `FindByID(ctx, id)` — загрузить из событий
   - `List(ctx, filters)` — запрос из read model
   - `Delete(ctx, id)` — soft delete

3. **Read Model проекции:**
   ```go
   type taskDocument struct {
       ID          string    `bson:"_id"`
       WorkspaceID string    `bson:"workspace_id"`
       ChatID      string    `bson:"chat_id"`
       Title       string    `bson:"title"`
       Status      string    `bson:"status"`
       Priority    string    `bson:"priority"`
       AssignedTo  *string   `bson:"assigned_to"`
       DueDate     *time.Time `bson:"due_date"`
       CreatedAt   time.Time `bson:"created_at"`
       UpdatedAt   time.Time `bson:"updated_at"`
       Version     int       `bson:"version"`
   }
   ```

4. **Integration tests** (~200 LOC)
   - Save/Load task lifecycle
   - Event replay восстанавливает состояние
   - Optimistic locking работает
   - Фильтрация по workspace/status/assignee

**Референс:** `chat_repository.go` (аналогичная структура)

**Критерии приёмки:**
- ✅ Task Repository реализован
- ✅ Event Sourcing работает корректно
- ✅ Read Model обновляется при Save
- ✅ Все integration tests проходят
- ✅ Coverage > 85%

**Оценка:** 2 дня (16 часов)

---

#### День 3 (3 января): MongoDB Indexes

**Задача:** Создать production-ready индексы для всех коллекций

**Файлы:**
```
internal/infrastructure/mongodb/
├── indexes.go           (новый)
└── indexes_test.go      (новый)
```

**Детали реализации:**

1. **Index Manager** (~150 LOC)
   ```go
   type IndexManager struct {
       client *mongo.Client
       db     *mongo.Database
   }
   
   func (m *IndexManager) CreateAllIndexes(ctx context.Context) error
   func (m *IndexManager) DropAllIndexes(ctx context.Context) error
   ```

2. **Индексы по коллекциям:**

   **events:**
   ```go
   // Unique для optimistic locking
   {aggregate_id: 1, version: 1} - unique
   
   // Для загрузки событий агрегата
   {aggregate_id: 1, created_at: 1}
   
   // Для поиска по типу
   {event_type: 1, created_at: -1}
   ```

   **chats (read model):**
   ```go
   {workspace_id: 1, type: 1, created_at: -1}
   {workspace_id: 1, status: 1}
   {parent_id: 1, created_at: 1}
   {participants: 1}
   ```

   **tasks (read model):**
   ```go
   {workspace_id: 1, status: 1, created_at: -1}
   {workspace_id: 1, assigned_to: 1}
   {chat_id: 1, created_at: -1}
   {due_date: 1, status: 1}
   ```

   **messages:**
   ```go
   {chat_id: 1, created_at: -1}
   {chat_id: 1, user_id: 1}
   {parent_id: 1, created_at: 1}  // threads
   ```

   **users:**
   ```go
   {email: 1} - unique
   {username: 1} - unique
   {keycloak_id: 1} - unique, sparse
   ```

   **workspaces:**
   ```go
   {keycloak_group_id: 1} - unique
   ```

   **notifications:**
   ```go
   {user_id: 1, read_at: 1, created_at: -1}
   {workspace_id: 1, created_at: -1}
   ```

3. **Migration скрипт:**
   ```go
   // cmd/migrator/main.go
   func runIndexMigration(ctx context.Context, db *mongo.Database) error
   ```

**Критерии приёмки:**
- ✅ Все индексы созданы
- ✅ Unique constraints защищают от дубликатов
- ✅ Compound indexes покрывают частые запросы
- ✅ Migration скрипт работает идемпотентно
- ✅ Tests проверяют создание/удаление индексов

**Оценка:** 1 день (8 часов)

---

#### День 4-6 (4-6 января): Event Bus Basic Implementation

**Задача:** Реализовать Redis Pub/Sub Event Bus для асинхронной обработки событий

**Файлы:**
```
internal/infrastructure/eventbus/
├── redis_eventbus.go           (новый, ~300 LOC)
├── redis_eventbus_test.go      (новый, ~200 LOC)
├── handlers.go                 (новый, ~200 LOC)
└── handlers_test.go            (новый, ~150 LOC)
```

**Детали реализации:**

1. **RedisEventBus** (~300 LOC)
   ```go
   type RedisEventBus struct {
       client     *redis.Client
       pubsub     *redis.PubSub
       handlers   map[string][]EventHandler
       running    bool
       shutdown   chan struct{}
       wg         sync.WaitGroup
   }
   
   func (b *RedisEventBus) Publish(ctx, event) error
   func (b *RedisEventBus) Subscribe(eventType, handler) error
   func (b *RedisEventBus) Start(ctx) error
   func (b *RedisEventBus) Shutdown() error
   ```

2. **Event Serialization:**
   - JSON serialization для событий
   - Поддержка всех domain events
   - Обработка ошибок десериализации

3. **Event Handlers:**
   
   **NotificationHandler:**
   ```go
   type NotificationHandler struct {
       createNotifUC *notification.CreateNotificationUseCase
   }
   
   func (h *NotificationHandler) Handle(ctx, event) error {
       // ChatCreated → создать уведомление участникам
       // MessageSent → уведомить упомянутых пользователей
       // TaskAssigned → уведомить assignee
   }
   ```

   **LoggingHandler:**
   ```go
   type LoggingHandler struct {
       logger *log.Logger
   }
   
   func (h *LoggingHandler) Handle(ctx, event) error {
       // Логировать все события для audit trail
   }
   ```

4. **Error Handling:**
   - Retry logic с exponential backoff
   - Dead Letter Queue для failed events
   - Graceful shutdown без потери событий

**Критерии приёмки:**
- ✅ Redis Pub/Sub работает
- ✅ События публикуются асинхронно
- ✅ NotificationHandler создаёт уведомления
- ✅ LoggingHandler пишет audit log
- ✅ Graceful shutdown корректен
- ✅ Integration tests проходят

**Оценка:** 3 дня (24 часа)

---

#### День 7 (7 января): Code Review & Documentation

**Задачи:**
- Code review всех изменений Week 1
- Обновить документацию
- Smoke tests для всего Infrastructure Layer
- Подготовить демо для stakeholders

**Deliverables Week 1:**
- ✅ Task Repository готов
- ✅ MongoDB Indexes созданы
- ✅ Event Bus работает
- ✅ Infrastructure Layer: 90% complete

---

### Неделя 2: 8-14 января — HTTP Infrastructure

**Приоритет:** 🔴 КРИТИЧЕСКИЙ  
**Цель:** Настроить Echo router и middleware

#### День 8-10 (8-10 января): Echo Router & Middleware

**Задача:** Создать HTTP infrastructure с Echo v4

**Файлы:**
```
internal/infrastructure/http/
├── router.go               (новый, ~400 LOC)
├── server.go               (новый, ~150 LOC)
└── response.go             (новый, ~100 LOC)

internal/middleware/
├── auth.go                 (новый, ~200 LOC)
├── workspace.go            (новый, ~150 LOC)
├── cors.go                 (новый, ~50 LOC)
├── logging.go              (новый, ~100 LOC)
├── rate_limit.go           (новый, ~150 LOC)
└── recovery.go             (новый, ~80 LOC)
```

**Детали реализации:**

1. **Echo Server Setup:**
   ```go
   func NewServer(config *Config) *echo.Echo {
       e := echo.New()
       e.Use(middleware.Logger())
       e.Use(middleware.Recover())
       e.Use(middleware.CORS())
       
       // Custom middleware
       e.Use(middlewares.RequestID())
       e.Use(middlewares.Logging())
       
       return e
   }
   ```

2. **Router Groups:**
   ```go
   // Public routes
   public := e.Group("/api/v1")
   
   // Authenticated routes
   auth := public.Group("", middlewares.Auth())
   
   // Workspace-scoped routes
   workspace := auth.Group("/workspaces/:workspace_id",
       middlewares.WorkspaceAccess())
   ```

3. **Middleware:**

   **Auth Middleware:**
   - JWT validation
   - User extraction
   - Permission checks

   **Workspace Middleware:**
   - Проверка доступа к workspace
   - Извлечение workspace_id из пути
   - Проверка членства пользователя

   **Rate Limiting:**
   - Redis-based rate limiter
   - Per-user limits
   - Per-endpoint limits

   **Logging:**
   - Request/response logging
   - Performance metrics
   - Error tracking

4. **Response Helpers:**
   ```go
   func RespondJSON(c echo.Context, code int, data interface{}) error
   func RespondError(c echo.Context, err error) error
   func RespondValidationError(c echo.Context, err error) error
   ```

**Критерии приёмки:**
- ✅ Echo server запускается
- ✅ Middleware chain работает
- ✅ CORS настроен
- ✅ Rate limiting работает
- ✅ Logging пишет в stdout
- ✅ Unit tests для middleware

**Оценка:** 3 дня (24 часа)

---

#### День 11-14 (11-14 января): Basic HTTP Handlers

**Задача:** Реализовать первые HTTP handlers

**Файлы:**
```
internal/handler/http/
├── auth_handler.go         (новый, ~200 LOC)
├── workspace_handler.go    (новый, ~300 LOC)
├── chat_handler.go         (новый, ~400 LOC)
└── message_handler.go      (новый, ~300 LOC)
```

**Endpoints для реализации:**

**Auth Handler:**
- `POST /api/v1/auth/login` — OAuth callback
- `POST /api/v1/auth/logout` — logout
- `GET /api/v1/auth/me` — current user info

**Workspace Handler:**
- `POST /api/v1/workspaces` — create workspace
- `GET /api/v1/workspaces` — list user's workspaces
- `GET /api/v1/workspaces/:id` — get workspace
- `PUT /api/v1/workspaces/:id` — update workspace
- `DELETE /api/v1/workspaces/:id` — delete workspace

**Chat Handler:**
- `POST /api/v1/workspaces/:workspace_id/chats` — create chat
- `GET /api/v1/workspaces/:workspace_id/chats` — list chats
- `GET /api/v1/chats/:id` — get chat
- `PUT /api/v1/chats/:id` — update chat
- `DELETE /api/v1/chats/:id` — delete chat
- `POST /api/v1/chats/:id/participants` — add participant
- `DELETE /api/v1/chats/:id/participants/:user_id` — remove participant

**Message Handler:**
- `POST /api/v1/chats/:chat_id/messages` — send message
- `GET /api/v1/chats/:chat_id/messages` — list messages
- `PUT /api/v1/messages/:id` — edit message
- `DELETE /api/v1/messages/:id` — delete message

**Критерии приёмки:**
- ✅ 20+ endpoints реализованы
- ✅ Request validation работает
- ✅ Authorization checks на месте
- ✅ Use cases вызываются корректно
- ✅ Error handling корректен
- ✅ Integration tests проходят

**Оценка:** 4 дня (32 часа)

**Deliverables Week 2:**
- ✅ HTTP Infrastructure настроена
- ✅ 20+ API endpoints работают
- ✅ Можно тестировать через curl/Postman

---

### Неделя 3: 15-21 января — More Handlers & WebSocket

**Приоритет:** 🟡 ВЫСОКИЙ  
**Цель:** Завершить основные handlers и добавить WebSocket

#### День 15-17 (15-17 января): Task & Notification Handlers

**Файлы:**
```
internal/handler/http/
├── task_handler.go         (новый, ~400 LOC)
├── notification_handler.go (новый, ~250 LOC)
└── user_handler.go         (новый, ~200 LOC)
```

**Task Handler endpoints:**
- `POST /api/v1/workspaces/:workspace_id/tasks`
- `GET /api/v1/workspaces/:workspace_id/tasks`
- `GET /api/v1/tasks/:id`
- `PUT /api/v1/tasks/:id/status`
- `PUT /api/v1/tasks/:id/assign`
- `PUT /api/v1/tasks/:id/priority`
- `PUT /api/v1/tasks/:id/due-date`
- `DELETE /api/v1/tasks/:id`

**Notification Handler endpoints:**
- `GET /api/v1/notifications`
- `GET /api/v1/notifications/unread/count`
- `PUT /api/v1/notifications/:id/read`
- `PUT /api/v1/notifications/mark-all-read`
- `DELETE /api/v1/notifications/:id`

**User Handler endpoints:**
- `GET /api/v1/users/me`
- `PUT /api/v1/users/me`
- `GET /api/v1/users/:id`

**Оценка:** 3 дня (24 часа)

---

#### День 18-21 (18-21 января): WebSocket Server

**Задача:** Реализовать WebSocket для real-time updates

**Файлы:**
```
internal/infrastructure/websocket/
├── hub.go                  (новый, ~300 LOC)
├── client.go               (новый, ~250 LOC)
└── broadcaster.go          (новый, ~200 LOC)

internal/handler/websocket/
└── handler.go              (новый, ~150 LOC)
```

**Детали реализации:**

1. **Hub (connection manager):**
   ```go
   type Hub struct {
       clients    map[*Client]bool
       chatRooms  map[uuid.UUID]map[*Client]bool
       register   chan *Client
       unregister chan *Client
       broadcast  chan *Message
   }
   ```

2. **Client (WebSocket connection):**
   ```go
   type Client struct {
       hub    *Hub
       conn   *websocket.Conn
       send   chan []byte
       userID uuid.UUID
       chatIDs []uuid.UUID
   }
   ```

3. **Event Broadcaster:**
   - Слушает Event Bus
   - Отправляет события через WebSocket
   - Фильтрует по chat membership

4. **Message types:**
   - `message.new` — новое сообщение
   - `chat.updated` — изменение чата
   - `task.status_changed` — изменение статуса
   - `notification.new` — новое уведомление

**Критерии приёмки:**
- ✅ WebSocket connections работают
- ✅ Hub управляет клиентами
- ✅ Events broadcast через WS
- ✅ Graceful disconnect
- ✅ Integration tests

**Оценка:** 4 дня (32 часа)

**Deliverables Week 3:**
- ✅ Все основные handlers готовы
- ✅ WebSocket real-time updates работают
- ✅ 40+ API endpoints функциональны

---

### Неделя 4: 22-31 января — Entry Points & Integration

**Приоритет:** 🔴 КРИТИЧЕСКИЙ  
**Цель:** Собрать приложение, запустить первый раз

#### День 22-24 (22-24 января): Entry Points

**Задача:** Создать cmd/api/main.go и dependency injection

**Файлы:**
```
cmd/api/
├── main.go                 (новый, ~500 LOC)
├── container.go            (новый, ~400 LOC)
└── routes.go               (новый, ~300 LOC)

internal/config/
├── config.go               (новый, ~200 LOC)
└── loader.go               (новый, ~150 LOC)
```

**Детали реализации:**

1. **main.go:**
   ```go
   func main() {
       // Load configuration
       cfg := config.Load()
       
       // Build DI container
       container := buildContainer(cfg)
       
       // Setup router
       router := setupRoutes(container)
       
       // Start server
       router.Start(cfg.Server.Address)
   }
   ```

2. **Dependency Injection Container:**
   ```go
   type Container struct {
       // Infrastructure
       MongoDB      *mongo.Client
       Redis        *redis.Client
       EventStore   appcore.EventStore
       EventBus     appcore.EventBus
       
       // Repositories
       ChatRepo     chat.Repository
       TaskRepo     task.Repository
       // ... все остальные
       
       // Use Cases
       CreateChatUC *chat.CreateChatUseCase
       // ... все остальные
       
       // Handlers
       ChatHandler  *http.ChatHandler
       // ... все остальные
   }
   ```

3. **Configuration Loading:**
   - Читать из `configs/config.yaml`
   - Override через ENV variables
   - Validation конфигурации

4. **Graceful Shutdown:**
   ```go
   func gracefulShutdown(server *echo.Echo, eventBus EventBus) {
       quit := make(chan os.Signal, 1)
       signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
       <-quit
       
       ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
       defer cancel()
       
       eventBus.Shutdown()
       server.Shutdown(ctx)
   }
   ```

**Критерии приёмки:**
- ✅ `./api` запускает сервер
- ✅ Все dependencies корректно инициализированы
- ✅ Configuration загружается
- ✅ Graceful shutdown работает
- ✅ Health check endpoint работает

**Оценка:** 3 дня (24 часа)

---

#### День 25-27 (25-27 января): Integration Testing

**Задача:** E2E тесты для основных сценариев

**Файлы:**
```
tests/e2e/
├── auth_test.go
├── workspace_test.go
├── chat_test.go
├── message_test.go
└── task_test.go
```

**Сценарии для тестирования:**

1. **Complete User Journey:**
   - Login → Create Workspace → Create Chat → Send Message → Create Task

2. **Chat Flow:**
   - Create chat → Add participants → Send messages → Real-time delivery

3. **Task Management:**
   - Create task → Change status → Assign → Set due date → Complete

4. **WebSocket Events:**
   - Connect → Subscribe to chat → Receive messages in real-time

**Критерии приёмки:**
- ✅ 5+ E2E tests проходят
- ✅ Все основные flows покрыты
- ✅ WebSocket events тестируются
- ✅ Performance tests baseline

**Оценка:** 3 дня (24 часа)

---

#### День 28-31 (28-31 января): Bug Fixing & Documentation

**Задачи:**

1. **Bug Fixing:**
   - Исправить найденные баги из E2E тестов
   - Performance tuning
   - Memory leaks проверка

2. **Documentation:**
   - API documentation (Swagger/OpenAPI)
   - Deployment guide
   - Developer guide обновление

3. **Demo Preparation:**
   - Подготовить демо для stakeholders
   - Записать видео демонстрацию
   - Создать Postman collection

**Deliverables Week 4:**
- ✅ Приложение запускается: `./api`
- ✅ E2E tests проходят
- ✅ API документация готова
- ✅ Demo готово

---

## 📈 Метрики успеха на конец января

### Code Metrics
- **Lines of Code:** ~35,000+ (было 25,000)
- **Infrastructure Layer:** 100% complete
- **Interface Layer:** 70% complete
- **Entry Points:** 80% complete

### Test Coverage
- **Domain:** 90%+ (unchanged)
- **Application:** 80%+ (was 79%)
- **Infrastructure:** 85%+ (all components)
- **Interface:** 70%+ (new)

### Functionality
- ✅ Все MongoDB repositories работают
- ✅ Event Bus публикует события
- ✅ 40+ HTTP endpoints функциональны
- ✅ WebSocket real-time updates
- ✅ Приложение можно запустить
- ✅ E2E tests проходят

### Documentation
- ✅ API documentation (OpenAPI)
- ✅ Deployment guide
- ✅ Updated README
- ✅ Code examples

---

## 🚨 Риски и митигация

| Риск | Вероятность | Влияние | Митигация |
|------|-------------|---------|-----------|
| Task Repository занимает > 2 дней | Средняя | Среднее | Использовать ChatRepository как референс |
| Event Bus интеграция проблемы | Низкая | Среднее | In-memory fallback готов |
| HTTP Handlers complexity underestimated | Средняя | Высокое | Начать с минимальных endpoints |
| WebSocket сложнее ожидаемого | Средняя | Среднее | Упростить до базового broadcast |
| DI wiring занимает много времени | Средняя | Среднее | Manual DI вместо wire |

---

## ✅ Definition of Done (конец января)

### Must Have:
- ✅ Task Repository с Event Sourcing
- ✅ MongoDB Indexes для всех коллекций
- ✅ Event Bus публикует события
- ✅ 40+ HTTP endpoints работают
- ✅ WebSocket real-time updates
- ✅ cmd/api/main.go запускает приложение
- ✅ E2E tests для core flows
- ✅ API documentation

### Nice to Have:
- Keycloak OAuth integration (можно отложить)
- Advanced rate limiting
- Metrics/monitoring
- Advanced error handling

---

## 📞 Контрольные точки

### Еженедельные check-ins:
- **Понедельник:** Planning, task breakdown
- **Среда:** Mid-week review, blocker resolution
- **Пятница:** Week review, demo, retro

### Milestone reviews:
- **7 января:** Infrastructure Layer complete
- **14 января:** HTTP Infrastructure ready
- **21 января:** All handlers + WebSocket done
- **31 января:** Application can start, E2E tests pass

---

## 📚 Ресурсы

### Documentation References
- [DEVELOPMENT_ROADMAP_2025.md](DEVELOPMENT_ROADMAP_2025.md)
- [STATUS.md](STATUS.md)
- [MongoDB Repositories Plan](tasks/05-impl-mongodb-repositories/README.md)

### External Resources
- [Echo v4 Guide](https://echo.labstack.com/guide/)
- [MongoDB Go Driver v2](https://pkg.go.dev/go.mongodb.org/mongo-driver/v2)
- [Redis Go Client](https://redis.uptrace.dev/)
- [WebSocket Protocol](https://datatracker.ietf.org/doc/html/rfc6455)

### Team Communication
- Daily standups: 10:00 UTC
- Slack channel: #new-teams-up-dev
- Code reviews: GitHub PRs

---

## 🎯 Следующие шаги после января

### Февраль 2025: Frontend & Polish
1. HTMX templates (2-3 недели)
2. Pico CSS customization
3. JavaScript utilities
4. Keycloak OAuth integration
5. Advanced features (file upload, search)

### Март 2025: Production Readiness
1. Performance optimization
2. Security hardening
3. Monitoring & alerting
4. CI/CD pipeline
5. Production deployment

---

**Успехов в разработке! 🚀**

*Plan owner: Project Lead*  
*Last updated: 2024-12-31*  
*Next review: 2025-01-07*
