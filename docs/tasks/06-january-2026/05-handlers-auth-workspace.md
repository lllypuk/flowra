# 05: Auth & Workspace Handlers

**Приоритет:** 🔴 Critical  
**Статус:** ✅ Выполнено  
**Дни:** 11-12 января  
**Зависит от:** [04-middleware.md](04-middleware.md)

---

## Описание

Реализовать HTTP handlers для аутентификации и управления workspaces. Эти handlers обеспечивают базовую функциональность для входа пользователей и организации их работы в workspaces.

---

## Файлы

### Созданные файлы

```
internal/handler/http/
├── auth_handler.go         (386 LOC) - Auth handler с Login, Logout, Me, Refresh
├── auth_handler_test.go    (680 LOC) - Тесты для auth handler
├── workspace_handler.go    (1030 LOC) - Workspace handler с CRUD и member management
└── workspace_handler_test.go (1670 LOC) - Тесты для workspace handler
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
    authService AuthService
    userRepo    UserRepository
}

func NewAuthHandler(authService AuthService, userRepo UserRepository) *AuthHandler

func (h *AuthHandler) Login(c echo.Context) error
func (h *AuthHandler) Logout(c echo.Context) error
func (h *AuthHandler) Me(c echo.Context) error
func (h *AuthHandler) Refresh(c echo.Context) error
```

#### Login Flow

1. Получить OAuth code/token из request
2. Валидировать через AuthService
3. Создать/обновить пользователя в системе
4. Выдать JWT access + refresh tokens
5. Вернуть user info

#### Logout Flow

1. Получить user из context
2. Invalidate refresh token
3. Очистить сессию

### Workspace Handler

```go
type WorkspaceHandler struct {
    workspaceService WorkspaceService
    memberService    MemberService
}

func NewWorkspaceHandler(workspaceService WorkspaceService, memberService MemberService) *WorkspaceHandler

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
    Code        string `json:"code"`
    RedirectURI string `json:"redirect_uri"`
}

type LoginResponse struct {
    AccessToken  string  `json:"access_token"`
    RefreshToken string  `json:"refresh_token"`
    ExpiresIn    int     `json:"expires_in"`
    User         UserDTO `json:"user"`
}

type UserDTO struct {
    ID          uuid.UUID `json:"id"`
    Email       string    `json:"email"`
    Username    string    `json:"username"`
    DisplayName string    `json:"display_name,omitempty"`
    AvatarURL   string    `json:"avatar_url,omitempty"`
}
```

### Workspace DTOs

```go
type CreateWorkspaceRequest struct {
    Name        string `json:"name"`
    Description string `json:"description"`
}

type WorkspaceResponse struct {
    ID          uuid.UUID `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description,omitempty"`
    OwnerID     uuid.UUID `json:"owner_id"`
    CreatedAt   string    `json:"created_at"`
    UpdatedAt   string    `json:"updated_at"`
    MemberCount int       `json:"member_count"`
}

type AddMemberRequest struct {
    UserID uuid.UUID `json:"user_id"`
    Role   string    `json:"role"`
}
```

---

## Validation

- Ручная валидация с понятными error messages
- Константы для max lengths (100 для name, 500 для description)
- Унифицированные error responses через `httpserver.RespondErrorWithCode`

---

## Error Handling

| Ошибка | HTTP Code | Описание |
|--------|-----------|----------|
| `UNAUTHORIZED` | 401 | Invalid/missing token |
| `FORBIDDEN` | 403 | No access to resource |
| `WORKSPACE_NOT_FOUND` | 404 | Workspace doesn't exist |
| `MEMBER_ALREADY_EXISTS` | 409 | User already member |
| `VALIDATION_ERROR` | 400 | Invalid request data |

---

## Чеклист

### Auth Handler
- [x] `Login` endpoint
- [x] `Logout` endpoint
- [x] `Me` endpoint
- [x] `Refresh` endpoint
- [x] Unit tests

### Workspace Handler
- [x] `Create` endpoint
- [x] `List` endpoints
- [x] `Get` endpoint
- [x] `Update` endpoint
- [x] `Delete` endpoint
- [x] `AddMember` endpoint
- [x] `RemoveMember` endpoint
- [x] `UpdateMemberRole` endpoint
- [x] Unit tests

### Общее
- [x] Request validation
- [x] Error responses
- [x] Authorization checks
- [x] Mock implementations for testing

---

## Критерии приёмки

- [x] 12 endpoints реализованы и работают
- [x] Request validation корректна
- [x] Authorization checks на месте
- [x] Error handling унифицирован
- [x] Unit tests: coverage 91.2% (выше 80%)
- [x] Все тесты проходят

---

## Зависимости

### Входящие
- [04-middleware.md](04-middleware.md) — Auth middleware, response helpers ✅

### Использует
- `middleware.GetUserID()` — получение user ID из context
- `middleware.IsSystemAdmin()` — проверка system admin
- `httpserver.RespondOK/Created/NoContent/ErrorWithCode` — response helpers
- Domain models: `workspace.Workspace`, `workspace.Member`, `user.User`

### Исходящие
- [06-handlers-chat-message.md](06-handlers-chat-message.md) — Chat handlers зависят от workspace context
- [09-entry-points.md](09-entry-points.md) — Entry point регистрирует handlers

---

## Заметки

- Mock implementations включены для тестирования и development
- Workspace deletion через `DeleteWorkspace` (soft delete в service layer)
- Member roles: `owner`, `admin`, `member`
- Owner не может быть удалён из workspace
- Только owner может менять роли других участников
- System admin имеет доступ ко всем workspaces

---

*Создано: 2026-01-01*  
*Выполнено: 2026-01-12*