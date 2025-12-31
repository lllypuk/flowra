# 06: Chat & Message Handlers

**Приоритет:** 🔴 Critical  
**Статус:** ⏳ Не начато  
**Дни:** 13-14 января  
**Зависит от:** [04-middleware.md](04-middleware.md), [05-handlers-auth-workspace.md](05-handlers-auth-workspace.md)

---

## Описание

Реализовать HTTP handlers для работы с чатами и сообщениями. Это ключевые endpoints для основной функциональности системы — общения пользователей.

---

## Файлы

```
internal/handler/http/
├── chat_handler.go         (~400 LOC)
├── chat_handler_test.go    (~300 LOC)
├── message_handler.go      (~300 LOC)
└── message_handler_test.go (~250 LOC)
```

---

## API Endpoints

### Chat Handler

| Method | Endpoint | Описание |
|--------|----------|----------|
| `POST` | `/api/v1/workspaces/:workspace_id/chats` | Создать чат |
| `GET` | `/api/v1/workspaces/:workspace_id/chats` | Список чатов workspace |
| `GET` | `/api/v1/chats/:id` | Получить чат |
| `PUT` | `/api/v1/chats/:id` | Обновить чат |
| `DELETE` | `/api/v1/chats/:id` | Удалить чат |
| `POST` | `/api/v1/chats/:id/participants` | Добавить участника |
| `DELETE` | `/api/v1/chats/:id/participants/:user_id` | Удалить участника |

### Message Handler

| Method | Endpoint | Описание |
|--------|----------|----------|
| `POST` | `/api/v1/chats/:chat_id/messages` | Отправить сообщение |
| `GET` | `/api/v1/chats/:chat_id/messages` | Список сообщений |
| `PUT` | `/api/v1/messages/:id` | Редактировать сообщение |
| `DELETE` | `/api/v1/messages/:id` | Удалить сообщение |

---

## Детали реализации

### ChatHandler

```go
type ChatHandler struct {
    createChatUC     *chat.CreateChatUseCase
    getChatUC        *chat.GetChatUseCase
    listChatsUC      *chat.ListChatsUseCase
    updateChatUC     *chat.UpdateChatUseCase
    deleteChatUC     *chat.DeleteChatUseCase
    addParticipantUC *chat.AddParticipantUseCase
    removeParticipantUC *chat.RemoveParticipantUseCase
}

func NewChatHandler(/* dependencies */) *ChatHandler

func (h *ChatHandler) Create(c echo.Context) error
func (h *ChatHandler) Get(c echo.Context) error
func (h *ChatHandler) List(c echo.Context) error
func (h *ChatHandler) Update(c echo.Context) error
func (h *ChatHandler) Delete(c echo.Context) error
func (h *ChatHandler) AddParticipant(c echo.Context) error
func (h *ChatHandler) RemoveParticipant(c echo.Context) error
```

### MessageHandler

```go
type MessageHandler struct {
    sendMessageUC   *message.SendMessageUseCase
    listMessagesUC  *message.ListMessagesUseCase
    editMessageUC   *message.EditMessageUseCase
    deleteMessageUC *message.DeleteMessageUseCase
}

func NewMessageHandler(/* dependencies */) *MessageHandler

func (h *MessageHandler) Send(c echo.Context) error
func (h *MessageHandler) List(c echo.Context) error
func (h *MessageHandler) Edit(c echo.Context) error
func (h *MessageHandler) Delete(c echo.Context) error
```

---

## Request/Response DTOs

### Create Chat

**Request:**
```json
{
    "name": "Project Discussion",
    "type": "group",
    "participant_ids": ["uuid-1", "uuid-2"]
}
```

**Response:**
```json
{
    "id": "chat-uuid",
    "name": "Project Discussion",
    "type": "group",
    "participants": [...],
    "created_at": "2026-01-13T10:00:00Z"
}
```

### Send Message

**Request:**
```json
{
    "content": "Hello, team!",
    "reply_to_id": null
}
```

**Response:**
```json
{
    "id": "message-uuid",
    "chat_id": "chat-uuid",
    "sender_id": "user-uuid",
    "content": "Hello, team!",
    "created_at": "2026-01-13T10:05:00Z"
}
```

### List Messages

**Query Parameters:**
- `limit` — количество (default: 50, max: 100)
- `before` — cursor для pagination (message ID)
- `after` — cursor для pagination (message ID)

**Response:**
```json
{
    "messages": [...],
    "has_more": true,
    "next_cursor": "message-uuid-last"
}
```

---

## Валидация

### Chat Validation

```go
type CreateChatRequest struct {
    Name           string      `json:"name" validate:"required,min=1,max=100"`
    Type           string      `json:"type" validate:"required,oneof=direct group channel"`
    ParticipantIDs []uuid.UUID `json:"participant_ids" validate:"required,min=1,max=100"`
}

type UpdateChatRequest struct {
    Name string `json:"name" validate:"omitempty,min=1,max=100"`
}
```

### Message Validation

```go
type SendMessageRequest struct {
    Content   string     `json:"content" validate:"required,min=1,max=10000"`
    ReplyToID *uuid.UUID `json:"reply_to_id" validate:"omitempty,uuid"`
}

type EditMessageRequest struct {
    Content string `json:"content" validate:"required,min=1,max=10000"`
}
```

---

## Authorization

### Chat Authorization

- **Create Chat:** пользователь должен быть членом workspace
- **Get Chat:** пользователь должен быть участником чата
- **Update Chat:** только owner или admin чата
- **Delete Chat:** только owner чата
- **Add Participant:** owner или admin чата
- **Remove Participant:** owner, admin, или сам пользователь (leave)

### Message Authorization

- **Send Message:** участник чата
- **List Messages:** участник чата
- **Edit Message:** автор сообщения
- **Delete Message:** автор сообщения или admin чата

---

## Error Handling

```go
var (
    ErrChatNotFound      = echo.NewHTTPError(404, "chat not found")
    ErrNotChatMember     = echo.NewHTTPError(403, "not a member of this chat")
    ErrNotChatAdmin      = echo.NewHTTPError(403, "admin access required")
    ErrMessageNotFound   = echo.NewHTTPError(404, "message not found")
    ErrNotMessageAuthor  = echo.NewHTTPError(403, "only message author can edit")
    ErrCannotRemoveSelf  = echo.NewHTTPError(400, "owner cannot leave chat")
)
```

---

## Чеклист

### ChatHandler
- [ ] `Create` — создание чата
- [ ] `Get` — получение чата
- [ ] `List` — список чатов workspace
- [ ] `Update` — обновление чата
- [ ] `Delete` — удаление чата
- [ ] `AddParticipant` — добавление участника
- [ ] `RemoveParticipant` — удаление участника

### MessageHandler
- [ ] `Send` — отправка сообщения
- [ ] `List` — список сообщений с pagination
- [ ] `Edit` — редактирование сообщения
- [ ] `Delete` — удаление сообщения

### Общее
- [ ] Request validation
- [ ] Authorization checks
- [ ] Error handling
- [ ] Unit tests для каждого endpoint
- [ ] Integration tests

---

## Критерии приёмки

- [ ] Все endpoints функциональны
- [ ] Request validation работает корректно
- [ ] Authorization проверяется для каждого действия
- [ ] Pagination для messages работает
- [ ] Events публикуются при изменениях
- [ ] Error responses консистентны
- [ ] Unit tests покрывают happy path и edge cases
- [ ] Integration tests проходят

---

## Зависимости

### Входящие
- [04-middleware.md](04-middleware.md) — middleware для auth и workspace
- [05-handlers-auth-workspace.md](05-handlers-auth-workspace.md) — базовые patterns

### Исходящие
- [08-websocket.md](08-websocket.md) — real-time updates
- [10-e2e-tests.md](10-e2e-tests.md) — E2E testing

---

## Заметки

- Message list использует cursor-based pagination для эффективности
- При отправке сообщения публикуется `MessageSent` event
- При добавлении участника — `ParticipantAdded` event
- Direct chats (1:1) не могут иметь более 2 участников
- Deleted messages помечаются флагом, а не удаляются физически

---

*Создано: 2026-01-01*