# 01: Event Bus (Redis Pub/Sub)

**Приоритет:** 🔴 Critical  
**Неделя:** 1 (1-3 января)  
**Статус:** ⏳ Не начато

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
- [ ] Создать `redis_eventbus.go`
- [ ] Реализовать `NewRedisEventBus`
- [ ] Реализовать `Publish` с JSON serialization
- [ ] Реализовать `Subscribe` для регистрации handlers
- [ ] Реализовать `Start` для запуска listener loop
- [ ] Реализовать `Shutdown` для graceful stop
- [ ] Добавить retry logic

### Тестирование
- [ ] Unit tests для serialization
- [ ] Integration tests с Redis testcontainer
- [ ] Test graceful shutdown
- [ ] Test multiple handlers

### Документация
- [ ] GoDoc комментарии
- [ ] Примеры использования в README

---

## Критерии приёмки

- [ ] Redis Pub/Sub работает
- [ ] События публикуются асинхронно
- [ ] Multiple handlers получают события
- [ ] Graceful shutdown корректен
- [ ] Integration tests проходят

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
