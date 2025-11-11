# Task 2.2: HTTP Handlers Implementation

**Приоритет:** 🔴 КРИТИЧЕСКИЙ
**Время:** 8-10 дней
**Зависимости:** Task 2.1 (HTTP Infrastructure)

---

## Цель

Реализовать HTTP handlers для всех use cases (7 handlers, 40+ endpoints).

---

## Handlers to Implement

### 1. AuthHandler (4 endpoints)
```go
POST /auth/login       → Redirect to Keycloak
GET  /auth/callback    → Exchange code, set session
POST /auth/logout      → Revoke token, clear session
GET  /auth/me          → Get current user info
```

### 2. WorkspaceHandler (7 endpoints)
```go
POST   /workspaces                → CreateWorkspace
GET    /workspaces                → ListUserWorkspaces
GET    /workspaces/:id            → GetWorkspace
PUT    /workspaces/:id            → UpdateWorkspace
POST   /workspaces/:id/invites    → CreateInvite
POST   /invites/:token/accept     → AcceptInvite
DELETE /invites/:id               → RevokeInvite
```

### 3. ChatHandler (12 endpoints)
```go
POST   /workspaces/:wid/chats          → CreateChat
GET    /workspaces/:wid/chats          → ListChats
GET    /chats/:id                      → GetChat
POST   /chats/:id/participants         → AddParticipant
DELETE /chats/:id/participants/:userId → RemoveParticipant
PUT    /chats/:id/status               → ChangeStatus
PUT    /chats/:id/assignee             → AssignUser
PUT    /chats/:id/priority             → SetPriority
PUT    /chats/:id/due-date             → SetDueDate
```

### 4. MessageHandler (8 endpoints)
```go
POST   /chats/:chatId/messages         → SendMessage
GET    /chats/:chatId/messages         → ListMessages
GET    /messages/:id                   → GetMessage
PUT    /messages/:id                   → EditMessage
DELETE /messages/:id                   → DeleteMessage
POST   /messages/:id/reactions         → AddReaction
```

### 5. NotificationHandler (5 endpoints)
```go
GET    /notifications          → ListNotifications
GET    /notifications/unread   → CountUnread
PUT    /notifications/:id/read → MarkAsRead
PUT    /notifications/read-all → MarkAllAsRead
DELETE /notifications/:id      → DeleteNotification
```

---

## Implementation Pattern

```go
type ChatHandler struct {
    createChatUC    *chat.CreateChatUseCase
    getChatUC       *chat.GetChatUseCase
    listChatsUC     *chat.ListChatsUseCase
    addParticipantUC *chat.AddParticipantUseCase
    // ... all use cases
}

func (h *ChatHandler) Create(c echo.Context) error {
    // 1. Parse request
    var req CreateChatRequest
    if err := BindAndValidate(c, &req); err != nil {
        return RespondError(c, err)
    }

    // 2. Build command
    cmd := chat.CreateChatCommand{
        WorkspaceID: GetWorkspaceID(c),
        Type:        req.Type,
        Title:       req.Title,
        IsPublic:    req.IsPublic,
        CreatedBy:   GetUserID(c),
    }

    // 3. Execute use case
    result, err := h.createChatUC.Execute(c.Request().Context(), cmd)
    if err != nil {
        return RespondError(c, err)
    }

    // 4. Return response
    return RespondJSON(c, http.StatusCreated, CreateChatResponse{
        ChatID:    result.ChatID,
        Type:      result.Type,
        CreatedAt: result.CreatedAt,
    })
}
```

---

## DTOs

```
internal/handler/http/dto/
├── auth_dto.go
├── workspace_dto.go
├── chat_dto.go
├── message_dto.go
├── notification_dto.go
└── common_dto.go  (pagination, errors)
```

Example:
```go
type CreateChatRequest struct {
    Type     string `json:"type" validate:"required"`
    Title    string `json:"title" validate:"required,max=255"`
    IsPublic bool   `json:"is_public"`
}

type CreateChatResponse struct {
    ChatID    uuid.UUID `json:"chat_id"`
    Type      string    `json:"type"`
    CreatedAt time.Time `json:"created_at"`
}

type PaginationResponse struct {
    Total   int  `json:"total"`
    Limit   int  `json:"limit"`
    Offset  int  `json:"offset"`
    HasMore bool `json:"has_more"`
}
```

---

## Testing

```go
func TestChatHandler_Create_Success(t *testing.T) {
    // Setup
    e := echo.New()
    req := httptest.NewRequest(http.MethodPost, "/chats", strings.NewReader(`{"type":"Discussion","title":"Test"}`))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    // Mock use case
    useCase := &MockCreateChatUseCase{}
    useCase.On("Execute", mock.Anything, mock.Anything).Return(&chat.CreateChatResult{
        ChatID: uuid.New(),
    }, nil)

    handler := &ChatHandler{createChatUC: useCase}

    // Execute
    err := handler.Create(c)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, http.StatusCreated, rec.Code)
}
```

---

## Критерии успеха

- ✅ **40+ endpoints реализованы**
- ✅ **Request/Response validation работает**
- ✅ **Error handling корректен**
- ✅ **DTOs корректно маппятся**
- ✅ **Test coverage >75%**

---

## Следующий шаг

→ **Task 2.3: WebSocket Implementation**
