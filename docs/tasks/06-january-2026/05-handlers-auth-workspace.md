# 05: Auth & Workspace Handlers

**Приоритет:** 🔴 Critical  
**Статус:** ⏳ Не начато  
**Дни:** 11-12 января  
**Зависит от:** [04-middleware.md](04-middleware.md)

---

## Описание

Реализовать HTTP handlers для аутентификации и управления workspaces. Эти handlers обеспечивают базовую функциональность для входа пользователей и организации их работы в workspaces.

---

## Файлы для создания

```
internal/handler/http/
├── auth_handler.go         (~200 LOC)
├── auth_handler_test.go    (~150 LOC)
├── workspace_handler.go    (~300 LOC)
└── workspace_handler_test.go (~200 LOC)
```

---

## Endpoints

### Auth Handler

| Метод | Endpoint | Описание |
|-------|----------|----------|
| `POST` | `/api/v1/auth/login` | OAuth callback / login |
| `POST` | `/api/v1/auth/logout` | Logout, invalidate session |
| `GET` | `/api/v1/auth/me` | Get current user info |
| `POST` | `/api/v1/auth/refresh` | Refresh access token |

### Workspace Handler

| Метод | Endpoint | Описание |
|-------|----------|----------|
| `POST` | `/api/v1/workspaces` | Create workspace |
| `GET` | `/api/v1/workspaces` | List user's workspaces |
| `GET` | `/api/v1/workspaces/:id` | Get workspace by ID |
| `PUT` | `/api/v1/workspaces/:id` | Update workspace |
| `DELETE` | `/api/v1/workspaces/:id` | Delete workspace |
| `POST` | `/api/v1/workspaces/:id/members` | Add member |
| `DELETE` | `/api/v1/workspaces/:id/members/:user_id` | Remove member |
| `PUT` | `/api/v1/workspaces/:id/members/:user_id/role` | Update member role |

---

## Детали реализации

### Auth Handler

```go
type AuthHandler struct {
    loginUC   *auth.LoginUseCase
    logoutUC  *auth.LogoutUseCase
    refreshUC *auth.RefreshTokenUseCase
    userRepo  user.Repository
}

func NewAuthHandler(
    loginUC *auth.LoginUseCase,
    logoutUC *auth.LogoutUseCase,
    refreshUC *auth.RefreshTokenUseCase,
    userRepo user.Repository,
) *AuthHandler

func (h *AuthHandler) Login(c echo.Context) error
func (h *AuthHandler) Logout(c echo.Context) error
func (h *AuthHandler) Me(c echo.Context) error
func (h *AuthHandler) Refresh(c echo.Context) error
```

#### Login Flow

1. Получить OAuth code/token из request
2. Валидировать через Keycloak
3. Создать/обновить пользователя в системе
4. Выдать JWT access + refresh tokens
5. Вернуть user info

#### Logout Flow

1. Получить user из context
2. Invalidate refresh token
3. Очистить сессию в Redis

### Workspace Handler

```go
type WorkspaceHandler struct {
    createWS   *workspace.CreateWorkspaceUseCase
    updateWS   *workspace.UpdateWorkspaceUseCase
    deleteWS   *workspace.DeleteWorkspaceUseCase
    addMember  *workspace.AddMemberUseCase
    removeMember *workspace.RemoveMemberUseCase
    wsRepo     workspace.Repository
}

func NewWorkspaceHandler(...) *WorkspaceHandler

func (h *WorkspaceHandler) Create(c echo.Context) error
func (h *WorkspaceHandler) List(c echo.Context) error
func (h *WorkspaceHandler) Get(c echo.Context) error
func (h *WorkspaceHandler) Update(c echo.Context) error
func (h *WorkspaceHandler) Delete(c echo.Context) error
func (h *WorkspaceHandler) AddMember(c echo.Context) error
func (h *WorkspaceHandler) RemoveMember(c echo.Context) error
func (h *WorkspaceHandler) UpdateMemberRole(c echo.Context) error
```

---

## Request/Response DTOs

### Auth DTOs

```go
type LoginRequest struct {
    Code        string `json:"code"`         // OAuth code
    RedirectURI string `json:"redirect_uri"`
}

type LoginResponse struct {
    AccessToken  string    `json:"access_token"`
    RefreshToken string    `json:"refresh_token"`
    ExpiresIn    int       `json:"expires_in"`
    User         UserDTO   `json:"user"`
}

type UserDTO struct {
    ID        uuid.UUID `json:"id"`
    Email     string    `json:"email"`
    Name      string    `json:"name"`
    AvatarURL string    `json:"avatar_url,omitempty"`
}
```

### Workspace DTOs

```go
type CreateWorkspaceRequest struct {
    Name        string `json:"name" validate:"required,min=1,max=100"`
    Description string `json:"description" validate:"max=500"`
}

type WorkspaceResponse struct {
    ID          uuid.UUID          `json:"id"`
    Name        string             `json:"name"`
    Description string             `json:"description"`
    OwnerID     uuid.UUID          `json:"owner_id"`
    CreatedAt   time.Time          `json:"created_at"`
    MemberCount int                `json:"member_count"`
}

type AddMemberRequest struct {
    UserID uuid.UUID `json:"user_id" validate:"required"`
    Role   string    `json:"role" validate:"required,oneof=admin member guest"`
}
```

---

## Validation

- Используем `go-playground/validator/v10`
- Custom validators для business rules
- Унифицированные error responses

```go
func (h *WorkspaceHandler) Create(c echo.Context) error {
    var req CreateWorkspaceRequest
    if err := c.Bind(&req); err != nil {
        return RespondError(c, err)
    }
    
    if err := c.Validate(&req); err != nil {
        return RespondValidationError(c, err)
    }
    
    userID := GetUserIDFromContext(c)
    ws, err := h.createWS.Execute(c.Request().Context(), userID, req.Name, req.Description)
    if err != nil {
        return RespondError(c, err)
    }
    
    return RespondCreated(c, toWorkspaceResponse(ws))
}
```

---

## Error Handling

| Ошибка | HTTP Code | Описание |
|--------|-----------|----------|
| `ErrUnauthorized` | 401 | Invalid/missing token |
| `ErrForbidden` | 403 | No access to resource |
| `ErrWorkspaceNotFound` | 404 | Workspace doesn't exist |
| `ErrMemberAlreadyExists` | 409 | User already member |
| `ErrValidationFailed` | 422 | Invalid request data |

---

## Чеклист

### Auth Handler
- [ ] `Login` endpoint
- [ ] `Logout` endpoint
- [ ] `Me` endpoint
- [ ] `Refresh` endpoint
- [ ] Unit tests

### Workspace Handler
- [ ] `Create` endpoint
- [ ] `List` endpoints
- [ ] `Get` endpoint
- [ ] `Update` endpoint
- [ ] `Delete` endpoint
- [ ] `AddMember` endpoint
- [ ] `RemoveMember` endpoint
- [ ] `UpdateMemberRole` endpoint
- [ ] Unit tests

### Общее
- [ ] Request validation
- [ ] Error responses
- [ ] Authorization checks
- [ ] Integration tests

---

## Критерии приёмки

- [ ] 12 endpoints реализованы и работают
- [ ] Request validation корректна
- [ ] Authorization checks на месте
- [ ] Use cases вызываются правильно
- [ ] Error handling унифицирован
- [ ] Unit tests: coverage 80%+
- [ ] Integration tests проходят

---

## Зависимости

### Входящие
- [04-middleware.md](04-middleware.md) — Auth middleware, response helpers

### Использует
- `auth.*UseCase` — authentication logic
- `workspace.*UseCase` — workspace operations
- `user.Repository` — user data access
- `workspace.Repository` — workspace data access

### Исходящие
- [06-handlers-chat-message.md](06-handlers-chat-message.md) — Chat handlers зависят от workspace context
- [09-entry-points.md](09-entry-points.md) — Entry point регистрирует handlers

---

## Заметки

- OAuth integration с Keycloak можно упростить на первом этапе (mock tokens)
- Workspace deletion должен быть soft delete
- Member roles: `owner`, `admin`, `member`, `guest`
- Owner не может быть удалён из workspace

---

*Создано: 2026-01-01*