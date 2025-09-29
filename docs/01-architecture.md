# Архитектура системы

## Общий обзор

Система построена на микросервисной архитектуре с использованием Go для backend и HTMX для frontend. Основные принципы:

- **Event-driven architecture** для слабой связанности компонентов
- **Domain-Driven Design** для организации бизнес-логики
- **CQRS** для разделения команд и запросов
- **Repository pattern** для абстракции доступа к данным

## Компоненты системы

### 1. API Gateway (Fiber)

Основной HTTP сервер, обрабатывающий:
- REST API endpoints
- HTMX запросы
- Статические файлы
- WebSocket upgrade

### 2. WebSocket Server

Отдельный сервер для real-time коммуникаций:
- Управление подключениями
- Broadcasting сообщений
- Presence tracking
- Heartbeat/ping-pong

### 3. Worker Service

Background обработка задач:
- SLA мониторинг
- Email уведомления
- Очистка старых данных
- Аналитика и отчеты

### 4. Command Processor

Обработка команд из чата:
- Парсинг команд
- Валидация
- Выполнение действий
- Event generation

## Слои приложения

```
┌─────────────────────────────────────────┐
│           Presentation Layer             │
│         (HTMX Templates, API)           │
├─────────────────────────────────────────┤
│           Application Layer              │
│      (Handlers, Commands, Events)       │
├─────────────────────────────────────────┤
│            Domain Layer                  │
│    (Business Logic, Entities, Rules)    │
├─────────────────────────────────────────┤
│         Infrastructure Layer             │
│    (Repository, External Services)      │
├─────────────────────────────────────────┤
│            Data Layer                    │
│      (PostgreSQL, Redis, S3)            │
└─────────────────────────────────────────┘
```

## Потоки данных

### 1. Синхронный поток (HTTP Request)

```mermaid
sequenceDiagram
    participant U as User
    participant H as HTMX
    participant A as API
    participant S as Service
    participant D as Database

    U->>H: Interaction
    H->>A: HTTP Request
    A->>S: Business Logic
    S->>D: Query/Command
    D-->>S: Result
    S-->>A: Response
    A-->>H: HTML Fragment
    H-->>U: DOM Update
```

### 2. Асинхронный поток (WebSocket)

```mermaid
sequenceDiagram
    participant U as User
    participant W as WebSocket
    participant H as Hub
    participant E as Event Bus
    participant C as Clients

    U->>W: Send Message
    W->>H: Broadcast
    H->>E: Publish Event
    E->>C: Notify Subscribers
    C-->>U: Real-time Update
```

### 3. Command Processing

```mermaid
flowchart LR
    A[Chat Message] --> B{Contains Command?}
    B -->|Yes| C[Parse Command]
    B -->|No| D[Store Message]
    C --> E[Validate Command]
    E --> F[Execute Handler]
    F --> G[Update State]
    G --> H[Emit Events]
    H --> I[Notify Clients]
```

## База данных

### Основные сущности

1. **Users** - пользователи системы
2. **Chats** - чаты (direct, group, support)
3. **Messages** - сообщения в чатах
4. **Tasks** - задачи с state machine
5. **Chat_members** - участники чатов
6. **Audit_log** - аудит действий

### Стратегия хранения

- **PostgreSQL** - основные данные, транзакции
- **Redis** - кеш, сессии, pub/sub
- **MongoDB** (опционально) - архив сообщений
- **S3** (опционально) - файлы и медиа

## Безопасность

### Аутентификация и авторизация

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Browser    │────▶│   Keycloak   │────▶│     API      │
└──────────────┘     └──────────────┘     └──────────────┘
        │                    │                     │
        │   1. Login        │                     │
        │──────────────────▶│                     │
        │                    │                     │
        │   2. Auth Code    │                     │
        │◀──────────────────│                     │
        │                    │                     │
        │   3. Exchange Code │                     │
        │──────────────────▶│                     │
        │                    │                     │
        │   4. JWT Token    │                     │
        │◀──────────────────│                     │
        │                    │                     │
        │   5. API Request with JWT               │
        │─────────────────────────────────────────▶│
        │                    │                     │
        │                    │  6. Validate JWT   │
        │                    │◀────────────────────│
        │                    │                     │
        │   7. Response      │                     │
        │◀─────────────────────────────────────────│
```

### RBAC модель

```yaml
roles:
  admin:
    - all_permissions
  manager:
    - manage_team
    - view_reports
    - manage_tasks
  agent:
    - handle_tickets
    - manage_own_tasks
    - chat_access
  user:
    - chat_access
    - create_tasks
    - view_own_data
```

## Масштабирование

### Вертикальное масштабирование

- Увеличение ресурсов сервера
- Оптимизация запросов БД
- Индексирование
- Connection pooling

### Горизонтальное масштабирование

```
                    ┌──────────────┐
                    │ Load Balancer│
                    └──────┬───────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
   ┌────▼────┐       ┌────▼────┐       ┌────▼────┐
   │  API-1  │       │  API-2  │       │  API-3  │
   └────┬────┘       └────┬────┘       └────┬────┘
        │                  │                  │
        └──────────┬───────┴──────────────────┘
                   │
           ┌───────▼──────┐
           │   Redis      │
           │   Pub/Sub    │
           └───────┬──────┘
                   │
        ┌──────────┼──────────┐
        │                     │
   ┌────▼────┐          ┌────▼────┐
   │ Worker-1│          │ Worker-2│
   └─────────┘          └─────────┘
```

## Event-Driven Architecture

### События системы

```go
// Основные события
type Events struct {
    // Chat events
    MessageCreated
    MessageUpdated
    MessageDeleted

    // Task events
    TaskCreated
    TaskAssigned
    TaskStatusChanged
    TaskCompleted

    // User events
    UserJoined
    UserLeft
    UserTyping

    // System events
    SLABreached
    NotificationSent
}
```

### Event Bus

```go
type EventBus interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(eventType string, handler EventHandler) error
    Unsubscribe(subscriptionID string) error
}
```

## Кеширование

### Стратегии кеширования

1. **Cache-aside** - для редко меняющихся данных
2. **Write-through** - для критичных данных
3. **Write-behind** - для массовых операций

### Уровни кеша

```
Application Cache (in-memory)
        ↓
   Redis Cache
        ↓
   Database
```

## Производительность

### Целевые метрики

- **Response Time P50**: < 50ms
- **Response Time P95**: < 200ms
- **Response Time P99**: < 500ms
- **WebSocket Connections**: 10,000+ per instance
- **Messages/sec**: 1,000+
- **Uptime**: 99.9%

### Оптимизации

1. **Database**
   - Connection pooling
   - Prepared statements
   - Proper indexing
   - Query optimization

2. **Caching**
   - Redis для hot data
   - In-memory cache для static data
   - CDN для static assets

3. **Code**
   - Goroutine pools
   - Efficient serialization
   - Minimal allocations
   - Profiling и benchmarking

## Мониторинг и наблюдаемость

### Метрики (Prometheus)

- Business metrics (messages, tasks, SLA)
- System metrics (CPU, memory, disk)
- Application metrics (goroutines, GC)
- Custom metrics

### Логирование (Zerolog)

```go
log.Info().
    Str("user_id", userID).
    Str("action", "message_sent").
    Dur("duration", duration).
    Msg("Message processed successfully")
```

### Трассировка (OpenTelemetry)

- Distributed tracing
- Request correlation
- Performance bottlenecks
- Error tracking

## Disaster Recovery

### Backup стратегия

1. **Database**: Daily snapshots + WAL архивы
2. **Files**: S3 с версионированием
3. **Configuration**: Git repository
4. **Secrets**: Encrypted backups

### RTO/RPO

- **RTO (Recovery Time Objective)**: < 1 час
- **RPO (Recovery Point Objective)**: < 15 минут

## Технологические решения

### Почему Go?

- Высокая производительность
- Отличная поддержка concurrency
- Простота deployment (single binary)
- Сильная типизация
- Богатая экосистема

### Почему HTMX?

- Минимальный JavaScript
- Server-side rendering
- Простота разработки
- Progressive enhancement
- SEO friendly

### Почему PostgreSQL?

- ACID compliance
- JSON support
- Full-text search
- Расширяемость
- Надежность

### Почему Keycloak?

- Enterprise-ready SSO
- Поддержка множества протоколов
- Гибкое управление ролями
- UI для администрирования
- Расширяемость
