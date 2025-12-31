# 10: E2E Tests

**Приоритет:** 🟡 High  
**Статус:** ⏳ Не начато  
**Дни:** 25-27 января  
**Зависит от:** [09-entry-points.md](09-entry-points.md)

---

## Описание

Реализовать End-to-End тесты для основных пользовательских сценариев. E2E тесты проверяют полный flow от HTTP запроса до базы данных и WebSocket событий.

---

## Файлы для создания

```
tests/e2e/
├── setup_test.go           (~200 LOC) — общая настройка тестов
├── auth_test.go            (~150 LOC) — тесты авторизации
├── workspace_test.go       (~200 LOC) — тесты workspace
├── chat_test.go            (~250 LOC) — тесты чатов
├── message_test.go         (~200 LOC) — тесты сообщений
├── task_test.go            (~250 LOC) — тесты задач
└── websocket_test.go       (~200 LOC) — тесты WebSocket
```

---

## Test Setup

### Testcontainers

Используем testcontainers для изолированного окружения:

```go
type TestSuite struct {
    app        *App
    httpClient *http.Client
    wsDialer   *websocket.Dialer
    
    mongoContainer testcontainers.Container
    redisContainer testcontainers.Container
}

func (s *TestSuite) SetupSuite() {
    // Start MongoDB container
    s.mongoContainer = startMongoContainer()
    
    // Start Redis container
    s.redisContainer = startRedisContainer()
    
    // Build and start app
    s.app = NewApp(testConfig())
    go s.app.Start()
    
    // Wait for app to be ready
    waitForHealthCheck(s.app.BaseURL() + "/health")
}

func (s *TestSuite) TearDownSuite() {
    s.app.Shutdown()
    s.mongoContainer.Terminate(context.Background())
    s.redisContainer.Terminate(context.Background())
}
```

### Test Fixtures

```go
func (s *TestSuite) createTestUser() *User {
    // Создать пользователя напрямую в БД
}

func (s *TestSuite) getAuthToken(userID uuid.UUID) string {
    // Получить JWT token для пользователя
}

func (s *TestSuite) createTestWorkspace(ownerID uuid.UUID) *Workspace {
    // Создать workspace через API
}
```

---

## Тестовые сценарии

### 1. Complete User Journey

**Сценарий:** Полный путь пользователя от входа до отправки сообщения

```go
func TestCompleteUserJourney(t *testing.T) {
    // 1. Login
    user := login(t, "test@example.com")
    
    // 2. Create Workspace
    workspace := createWorkspace(t, user.Token, "My Team")
    
    // 3. Create Chat
    chat := createChat(t, user.Token, workspace.ID, "General")
    
    // 4. Send Message
    message := sendMessage(t, user.Token, chat.ID, "Hello, World!")
    
    // 5. Create Task from Message
    task := createTask(t, user.Token, workspace.ID, "Review PR", chat.ID)
    
    // Assert
    assert.NotEmpty(t, user.ID)
    assert.NotEmpty(t, workspace.ID)
    assert.NotEmpty(t, chat.ID)
    assert.NotEmpty(t, message.ID)
    assert.NotEmpty(t, task.ID)
}
```

### 2. Chat Flow

**Сценарий:** Создание чата и обмен сообщениями

```go
func TestChatFlow(t *testing.T) {
    // Setup
    user1 := createTestUser(t)
    user2 := createTestUser(t)
    workspace := createWorkspace(t, user1.Token, "Team")
    
    // Add user2 to workspace
    addMember(t, user1.Token, workspace.ID, user2.ID)
    
    // Create chat with both users
    chat := createChat(t, user1.Token, workspace.ID, "Discussion", 
        []uuid.UUID{user1.ID, user2.ID})
    
    // User1 sends message
    msg1 := sendMessage(t, user1.Token, chat.ID, "Hi!")
    
    // User2 receives and replies
    messages := listMessages(t, user2.Token, chat.ID)
    assert.Len(t, messages, 1)
    
    msg2 := sendMessage(t, user2.Token, chat.ID, "Hello!")
    
    // Both messages visible
    messages = listMessages(t, user1.Token, chat.ID)
    assert.Len(t, messages, 2)
}
```

### 3. Task Management

**Сценарий:** Создание, назначение и завершение задачи

```go
func TestTaskManagement(t *testing.T) {
    // Setup
    manager := createTestUser(t)
    developer := createTestUser(t)
    workspace := createWorkspace(t, manager.Token, "Project")
    addMember(t, manager.Token, workspace.ID, developer.ID)
    
    // Create task
    task := createTask(t, manager.Token, workspace.ID, "Implement feature")
    assert.Equal(t, "open", task.Status)
    
    // Assign to developer
    assignTask(t, manager.Token, task.ID, developer.ID)
    
    // Developer changes status
    changeStatus(t, developer.Token, task.ID, "in_progress")
    
    // Complete task
    changeStatus(t, developer.Token, task.ID, "done")
    
    // Verify final state
    task = getTask(t, manager.Token, task.ID)
    assert.Equal(t, "done", task.Status)
    assert.Equal(t, developer.ID.String(), *task.AssigneeID)
}
```

### 4. WebSocket Events

**Сценарий:** Real-time события через WebSocket

```go
func TestWebSocketEvents(t *testing.T) {
    // Setup
    user1 := createTestUser(t)
    user2 := createTestUser(t)
    workspace := createWorkspace(t, user1.Token, "Team")
    addMember(t, user1.Token, workspace.ID, user2.ID)
    chat := createChat(t, user1.Token, workspace.ID, "Chat", 
        []uuid.UUID{user1.ID, user2.ID})
    
    // User2 connects to WebSocket
    ws := connectWebSocket(t, user2.Token)
    defer ws.Close()
    
    // Subscribe to chat
    subscribe(t, ws, chat.ID)
    
    // User1 sends message
    go sendMessage(t, user1.Token, chat.ID, "Real-time message!")
    
    // User2 receives event via WebSocket
    event := readWSEvent(t, ws, 5*time.Second)
    assert.Equal(t, "message.new", event.Type)
    assert.Equal(t, chat.ID.String(), event.ChatID)
    assert.Equal(t, "Real-time message!", event.Data.Content)
}
```

### 5. Notification Flow

**Сценарий:** Создание и чтение уведомлений

```go
func TestNotificationFlow(t *testing.T) {
    // Setup
    user1 := createTestUser(t)
    user2 := createTestUser(t)
    workspace := createWorkspace(t, user1.Token, "Project")
    addMember(t, user1.Token, workspace.ID, user2.ID)
    
    // Create task and assign to user2
    task := createTask(t, user1.Token, workspace.ID, "Review code")
    assignTask(t, user1.Token, task.ID, user2.ID)
    
    // Wait for async notification creation
    time.Sleep(500 * time.Millisecond)
    
    // User2 has notification
    count := getUnreadCount(t, user2.Token)
    assert.Equal(t, 1, count)
    
    // Read notification
    notifications := listNotifications(t, user2.Token)
    assert.Len(t, notifications, 1)
    assert.Contains(t, notifications[0].Title, "assigned")
    
    // Mark as read
    markRead(t, user2.Token, notifications[0].ID)
    
    count = getUnreadCount(t, user2.Token)
    assert.Equal(t, 0, count)
}
```

---

## Helper Functions

### HTTP Helpers

```go
func doRequest(t *testing.T, method, url, token string, body interface{}) *http.Response {
    var bodyReader io.Reader
    if body != nil {
        data, _ := json.Marshal(body)
        bodyReader = bytes.NewReader(data)
    }
    
    req, err := http.NewRequest(method, url, bodyReader)
    require.NoError(t, err)
    
    req.Header.Set("Content-Type", "application/json")
    if token != "" {
        req.Header.Set("Authorization", "Bearer "+token)
    }
    
    resp, err := http.DefaultClient.Do(req)
    require.NoError(t, err)
    
    return resp
}

func parseResponse[T any](t *testing.T, resp *http.Response) T {
    defer resp.Body.Close()
    
    var result T
    err := json.NewDecoder(resp.Body).Decode(&result)
    require.NoError(t, err)
    
    return result
}
```

### WebSocket Helpers

```go
func connectWebSocket(t *testing.T, token string) *websocket.Conn {
    url := fmt.Sprintf("ws://localhost:8080/ws?token=%s", token)
    conn, _, err := websocket.DefaultDialer.Dial(url, nil)
    require.NoError(t, err)
    return conn
}

func subscribe(t *testing.T, conn *websocket.Conn, chatID uuid.UUID) {
    msg := map[string]string{
        "type":    "subscribe",
        "chat_id": chatID.String(),
    }
    err := conn.WriteJSON(msg)
    require.NoError(t, err)
}

func readWSEvent(t *testing.T, conn *websocket.Conn, timeout time.Duration) WSEvent {
    conn.SetReadDeadline(time.Now().Add(timeout))
    
    var event WSEvent
    err := conn.ReadJSON(&event)
    require.NoError(t, err)
    
    return event
}
```

---

## Чеклист

### Setup
- [ ] Test suite с testcontainers
- [ ] Fixtures для users, workspaces
- [ ] Auth token generation
- [ ] HTTP client helpers
- [ ] WebSocket client helpers

### Test Cases
- [ ] Complete User Journey test
- [ ] Chat Flow test
- [ ] Message Flow test
- [ ] Task Management test
- [ ] WebSocket Events test
- [ ] Notification Flow test

### Coverage
- [ ] All main endpoints covered
- [ ] Error scenarios tested
- [ ] Edge cases covered
- [ ] Performance baseline recorded

---

## Критерии приёмки

- [ ] 5+ E2E тестов проходят
- [ ] Тесты используют testcontainers (изолированное окружение)
- [ ] Все основные flows покрыты
- [ ] WebSocket события тестируются
- [ ] Тесты стабильны (no flaky tests)
- [ ] Тесты можно запустить локально: `go test ./tests/e2e -tags=e2e`
- [ ] CI/CD интеграция готова

---

## Зависимости

### Входящие
- [09-entry-points.md](09-entry-points.md) — работающее приложение

### Внешние пакеты
- `github.com/testcontainers/testcontainers-go`
- `github.com/gorilla/websocket`
- `github.com/stretchr/testify`

---

## Заметки

- Тесты запускаются с тегом `e2e`: `go test ./tests/e2e -tags=e2e`
- Каждый тест создаёт изолированные данные
- WebSocket тесты требуют timeout для ожидания событий
- Event Bus работает асинхронно — нужны небольшие задержки или eventually assertions
- Testcontainers требуют Docker

---

*Создано: 2026-01-01*