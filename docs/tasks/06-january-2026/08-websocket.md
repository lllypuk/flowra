# Задача 08: WebSocket Server

**Приоритет:** 🟡 High  
**Статус:** ⏳ Не начато  
**Дни:** 18-21 января  
**Зависит от:** [01-event-bus.md](01-event-bus.md), [04-middleware.md](04-middleware.md)

---

## Описание

Реализовать WebSocket server для real-time updates. Клиенты подключаются через WebSocket и получают события в реальном времени: новые сообщения, изменения статусов задач, уведомления.

---

## Файлы для создания

```
internal/infrastructure/websocket/
├── hub.go                  (~300 LOC)
├── client.go               (~250 LOC)
├── broadcaster.go          (~200 LOC)
├── hub_test.go             (~200 LOC)
└── client_test.go          (~150 LOC)

internal/handler/websocket/
├── handler.go              (~150 LOC)
└── handler_test.go         (~100 LOC)
```

---

## Детали реализации

### 1. Hub (Connection Manager)

Управляет всеми WebSocket подключениями:

```go
type Hub struct {
    clients    map[*Client]bool
    chatRooms  map[uuid.UUID]map[*Client]bool
    register   chan *Client
    unregister chan *Client
    broadcast  chan *Message
    mu         sync.RWMutex
}

func NewHub() *Hub
func (h *Hub) Run(ctx context.Context)
func (h *Hub) Register(client *Client)
func (h *Hub) Unregister(client *Client)
func (h *Hub) JoinChat(client *Client, chatID uuid.UUID)
func (h *Hub) LeaveChat(client *Client, chatID uuid.UUID)
func (h *Hub) BroadcastToChat(chatID uuid.UUID, message []byte)
func (h *Hub) SendToUser(userID uuid.UUID, message []byte)
```

### 2. Client (WebSocket Connection)

Представляет одно WebSocket подключение:

```go
type Client struct {
    hub     *Hub
    conn    *websocket.Conn
    send    chan []byte
    userID  uuid.UUID
    chatIDs []uuid.UUID
    mu      sync.Mutex
}

func NewClient(hub *Hub, conn *websocket.Conn, userID uuid.UUID) *Client
func (c *Client) ReadPump()   // Читает сообщения от клиента
func (c *Client) WritePump()  // Отправляет сообщения клиенту
func (c *Client) Close()
```

### 3. Event Broadcaster

Слушает Event Bus и отправляет события через WebSocket:

```go
type Broadcaster struct {
    hub       *Hub
    eventBus  EventBus
}

func NewBroadcaster(hub *Hub, eventBus EventBus) *Broadcaster
func (b *Broadcaster) Start(ctx context.Context) error
func (b *Broadcaster) handleEvent(ctx context.Context, event domain.Event) error
```

**Обрабатываемые события:**
- `message.sent` → `{ "type": "message.new", "data": {...} }`
- `chat.updated` → `{ "type": "chat.updated", "data": {...} }`
- `task.status_changed` → `{ "type": "task.updated", "data": {...} }`
- `notification.created` → `{ "type": "notification.new", "data": {...} }`

### 4. WebSocket Handler

HTTP handler для upgrade к WebSocket:

```go
type Handler struct {
    hub      *Hub
    upgrader websocket.Upgrader
}

func NewHandler(hub *Hub) *Handler
func (h *Handler) HandleWebSocket(c echo.Context) error
```

---

## Message Protocol

### Client → Server

```json
{
    "type": "subscribe",
    "chat_id": "uuid"
}

{
    "type": "unsubscribe", 
    "chat_id": "uuid"
}

{
    "type": "ping"
}
```

### Server → Client

```json
{
    "type": "message.new",
    "chat_id": "uuid",
    "data": {
        "id": "uuid",
        "content": "Hello!",
        "sender_id": "uuid",
        "created_at": "2026-01-15T10:30:00Z"
    }
}

{
    "type": "chat.updated",
    "chat_id": "uuid",
    "data": {
        "name": "New Chat Name"
    }
}

{
    "type": "notification.new",
    "data": {
        "id": "uuid",
        "title": "New task assigned",
        "body": "You have been assigned to task #123"
    }
}

{
    "type": "pong"
}
```

---

## Connection Lifecycle

```
1. Client connects to /ws
2. Server validates JWT from query param or header
3. Server upgrades connection to WebSocket
4. Client is registered in Hub
5. Client sends "subscribe" for each chat
6. Server adds client to chat rooms
7. Server broadcasts events to relevant rooms
8. On disconnect, client is unregistered and removed from rooms
```

---

## Критерии приёмки

- [ ] WebSocket connections работают
- [ ] Hub корректно управляет клиентами
- [ ] Subscribe/unsubscribe на чаты работает
- [ ] Events broadcast через WebSocket
- [ ] Broadcaster слушает Event Bus
- [ ] Фильтрация событий по chat membership
- [ ] Graceful disconnect
- [ ] Heartbeat/ping-pong для keepalive
- [ ] Integration tests проходят

---

## Чеклист

### Hub
- [ ] Создать `hub.go`
- [ ] Реализовать регистрацию/дерегистрацию клиентов
- [ ] Реализовать chat rooms
- [ ] Реализовать broadcast по чатам
- [ ] Реализовать отправку конкретному пользователю
- [ ] Thread-safe операции с mutex

### Client
- [ ] Создать `client.go`
- [ ] Реализовать ReadPump с parsing команд
- [ ] Реализовать WritePump с buffered channel
- [ ] Ping/pong для keepalive
- [ ] Graceful close

### Broadcaster
- [ ] Создать `broadcaster.go`
- [ ] Подписаться на события из Event Bus
- [ ] Преобразование domain events → WebSocket messages
- [ ] Роутинг сообщений в правильные chat rooms

### Handler
- [ ] Создать `handler.go`
- [ ] WebSocket upgrade
- [ ] JWT validation
- [ ] Регистрация клиента в Hub

### Тестирование
- [ ] Unit tests для Hub
- [ ] Unit tests для Client
- [ ] Integration test для WebSocket flow
- [ ] Test multiple clients in same chat
- [ ] Test broadcast delivery

---

## Зависимости

### Требуется
- [01-event-bus.md](01-event-bus.md) — для Broadcaster
- [04-middleware.md](04-middleware.md) — для auth validation

### Внешние пакеты
- `github.com/gorilla/websocket` — WebSocket implementation

### Блокирует
- [10-e2e-tests.md](10-e2e-tests.md) — E2E tests с WebSocket

---

## Заметки

- Используем gorilla/websocket — стандартный выбор для Go
- Каждый Client запускает 2 goroutines (read + write)
- Hub работает в своей goroutine
- Broadcaster подписывается как EventHandler на Event Bus
- JWT можно передавать через query param `?token=xxx` или header
- При reconnect клиент должен повторно подписаться на чаты
- Рассмотреть Redis adapter для horizontal scaling (в будущем)

---

## Конфигурация

```yaml
websocket:
  read_buffer_size: 1024
  write_buffer_size: 1024
  ping_interval: 30s
  pong_wait: 60s
  write_wait: 10s
  max_message_size: 65536
```

---

*Создано: 2026-01-01*