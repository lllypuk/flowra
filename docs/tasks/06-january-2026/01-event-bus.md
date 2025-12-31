# 01: Event Bus (Redis Pub/Sub)

**Приоритет:** 🔴 Critical  
**Неделя:** 1 (1-3 января)  
**Статус:** ✅ Завершено

---

## Описание

Реализовать Redis Pub/Sub Event Bus для асинхронной обработки событий между компонентами системы. Event Bus является ключевым элементом event-driven архитектуры и обеспечивает слабую связанность между доменами.

---

## Цели

- Асинхронная публикация и доставка domain events
- Интеграция с Redis Pub/Sub для масштабируемости
- Поддержка множественных handlers на одно событие
- Graceful shutdown без потери событий

---

## Файлы для создания

```
internal/infrastructure/eventbus/
├── redis_eventbus.go           (~300 LOC)
└── redis_eventbus_test.go      (~200 LOC)
```

---

## Детали реализации

### RedisEventBus

```go
type RedisEventBus struct {
    client     *redis.Client
    pubsub     *redis.PubSub
    handlers   map[string][]EventHandler
    running    bool
    shutdown   chan struct{}
    wg         sync.WaitGroup
}

func NewRedisEventBus(client *redis.Client) *RedisEventBus
func (b *RedisEventBus) Publish(ctx context.Context, event domain.Event) error
func (b *RedisEventBus) Subscribe(eventType string, handler EventHandler) error
func (b *RedisEventBus) Start(ctx context.Context) error
func (b *RedisEventBus) Shutdown() error
```

### Event Serialization

- JSON serialization для событий
- Envelope с metadata (event type, timestamp, correlation ID)
- Обработка ошибок десериализации

### Error Handling

- Retry logic с exponential backoff
- Dead Letter Queue для failed events (опционально)
- Logging всех ошибок

---

## Чеклист

### Реализация
- [x] Создать `redis_eventbus.go`
- [x] Реализовать `NewRedisEventBus`
- [x] Реализовать `Publish` с JSON serialization
- [x] Реализовать `Subscribe` для регистрации handlers
- [x] Реализовать `Start` для запуска listener loop
- [x] Реализовать `Shutdown` для graceful stop
- [x] Добавить retry logic

### Тестирование
- [x] Unit tests для serialization
- [x] Integration tests с Redis testcontainer
- [x] Test graceful shutdown
- [x] Test multiple handlers

### Документация
- [x] GoDoc комментарии
- [ ] Примеры использования в README

---

## Критерии приёмки

- [x] Redis Pub/Sub работает
- [x] События публикуются асинхронно
- [x] Multiple handlers получают события
- [x] Graceful shutdown корректен
- [x] Integration tests проходят

---

## Зависимости

### Требуется
- Redis client (`github.com/redis/go-redis/v9`)
- Domain events interface

### Блокирует
- [02-event-handlers.md](02-event-handlers.md)
- [08-websocket.md](08-websocket.md)

---

## Референсы

- [Redis Pub/Sub Documentation](https://redis.io/topics/pubsub)
- [go-redis Client](https://redis.uptrace.dev/)
- `internal/domain/event/event.go` — базовый интерфейс событий

---

## Дополнительные возможности реализации

Реализованные фичи сверх базовых требований:

- **Configurable Options**: `WithLogger`, `WithRetryConfig`, `WithChannelPrefix`
- **Channel Prefix**: изоляция событий между разными инстансами
- **RetryConfig**: настраиваемый exponential backoff (MaxRetries, InitialBackoff, MaxBackoff, BackoffFactor)
- **Testcontainers**: автоматический запуск Redis в Docker для тестов
- **Shared Container**: переиспользование контейнера между тестами для быстрого выполнения
