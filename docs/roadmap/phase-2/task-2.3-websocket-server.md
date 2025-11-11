# Task 2.3: WebSocket Server

**Приоритет:** 🟡 MEDIUM
**Время:** 5-6 дней
**Зависимости:** Task 2.1 (HTTP Infrastructure)

---

## Цель

Реализовать WebSocket для real-time обновлений: new messages, task status changes, notifications.

---

## Архитектура

```
WebSocket Client ←→ Hub ←→ Event Bus (Redis)
                     ↓
              Chat Rooms (by ChatID)
```

---

## Файлы

```
internal/infrastructure/websocket/
├── hub.go               (manages clients and rooms)
├── hub_test.go
├── client.go            (individual WebSocket connection)
├── client_test.go
└── message.go           (message types)

internal/handler/websocket/
├── handler.go           (HTTP → WebSocket upgrade)
├── message_handler.go   (handle client messages)
└── event_broadcaster.go (Event Bus → WebSocket)
```

---

## Implementation

### 1. Hub (hub.go)

```go
type Hub struct {
    clients    map[*Client]bool
    chatRooms  map[uuid.UUID]map[*Client]bool  // chatID → clients
    register   chan *Client
    unregister chan *Client
    broadcast  chan *Message
    mu         sync.RWMutex
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

func (h *Hub) BroadcastToChat(chatID uuid.UUID, msg interface{}) {
    h.mu.RLock()
    clients := h.chatRooms[chatID]
    h.mu.RUnlock()

    data, _ := json.Marshal(msg)

    for client := range clients {
        select {
        case client.send <- data:
        default:
            close(client.send)
            delete(h.clients, client)
        }
    }
}
```

### 2. Client (client.go)

```go
type Client struct {
    hub      *Hub
    conn     *websocket.Conn
    send     chan []byte
    userID   uuid.UUID
    chatIDs  []uuid.UUID
}

func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()

    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            break
        }

        // Handle message (subscribe, typing, etc.)
        c.handleMessage(message)
    }
}

func (c *Client) writePump() {
    ticker := time.NewTicker(54 * time.Second)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            c.conn.WriteMessage(websocket.TextMessage, message)

        case <-ticker.C:
            c.conn.WriteMessage(websocket.PingMessage, nil)
        }
    }
}
```

### 3. WebSocket Handler (handler.go)

```go
func (h *Handler) ServeWS(c echo.Context) error {
    // 1. Upgrade to WebSocket
    conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
    if err != nil {
        return err
    }

    // 2. Authenticate (token from query param)
    token := c.QueryParam("token")
    claims, err := h.tokenValidator.Validate(token)
    if err != nil {
        conn.Close()
        return echo.NewHTTPError(401, "Invalid token")
    }

    // 3. Create client
    client := &Client{
        hub:    h.hub,
        conn:   conn,
        send:   make(chan []byte, 256),
        userID: claims.UserID,
    }

    // 4. Register and start pumps
    h.hub.register <- client

    go client.writePump()
    go client.readPump()

    return nil
}
```

### 4. Event Broadcaster (event_broadcaster.go)

```go
type EventBroadcaster struct {
    hub      *Hub
    eventBus eventbus.EventBus
}

func (b *EventBroadcaster) Start() {
    b.eventBus.Subscribe("MessagePosted", b)
    b.eventBus.Subscribe("StatusChanged", b)
    b.eventBus.Subscribe("NotificationCreated", b)
}

func (b *EventBroadcaster) Handle(ctx context.Context, event shared.DomainEvent) error {
    switch e := event.(type) {
    case *messageevents.MessagePosted:
        b.hub.BroadcastToChat(e.ChatID, WSMessage{
            Type: "chat.message.posted",
            Data: e,
        })

    case *chatevents.StatusChanged:
        b.hub.BroadcastToChat(e.ChatID, WSMessage{
            Type: "task.updated",
            Data: e,
        })

    case *notificationevents.NotificationCreated:
        b.hub.SendToUser(e.UserID, WSMessage{
            Type: "notification.new",
            Data: e,
        })
    }

    return nil
}
```

---

## Message Types

**Client → Server:**
- `subscribe.chat` - join chat room
- `unsubscribe.chat` - leave chat room
- `chat.typing` - typing indicator
- `ping` - keepalive

**Server → Client:**
- `chat.message.posted` - new message
- `chat.message.edited` - message edited
- `task.updated` - task status changed
- `notification.new` - new notification

---

## Testing

```go
func TestHub_BroadcastToChat(t *testing.T) {
    hub := NewHub()
    go hub.Run()

    // Create mock clients
    client1 := &Client{send: make(chan []byte, 10)}
    client2 := &Client{send: make(chan []byte, 10)}

    chatID := uuid.New()
    hub.chatRooms[chatID] = map[*Client]bool{
        client1: true,
        client2: true,
    }

    // Broadcast message
    hub.BroadcastToChat(chatID, map[string]string{"type": "test"})

    // Verify both clients received
    msg1 := <-client1.send
    msg2 := <-client2.send

    assert.Contains(t, string(msg1), "test")
    assert.Contains(t, string(msg2), "test")
}
```

---

## Критерии успеха

- ✅ **WebSocket connections работают**
- ✅ **Real-time broadcasts доставляются**
- ✅ **Auth через token работает**
- ✅ **Graceful disconnect handling**
- ✅ **Support 100+ concurrent connections**

---

Phase 2 завершена → **Phase 3: Entry Points**
