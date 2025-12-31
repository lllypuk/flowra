# Задача 04: Echo Router & Middleware

**Приоритет:** 🔴 Critical  
**Статус:** ⏳ Не начато  
**Дни:** 8-10 января  
**Зависит от:** [03-http-server.md](03-http-server.md)

---

## Описание

Создать HTTP infrastructure с Echo v4 framework: роутер, middleware chain и response helpers.

---

## Файлы для создания

```
internal/infrastructure/http/
├── router.go               (~400 LOC)
├── server.go               (~150 LOC)
└── response.go             (~100 LOC)

internal/middleware/
├── auth.go                 (~200 LOC)
├── workspace.go            (~150 LOC)
├── cors.go                 (~50 LOC)
├── logging.go              (~100 LOC)
├── rate_limit.go           (~150 LOC)
└── recovery.go             (~80 LOC)
```

---

## Детали реализации

### 1. Echo Server Setup (`server.go`)

```go
func NewServer(config *Config) *echo.Echo {
    e := echo.New()
    e.Use(middleware.Logger())
    e.Use(middleware.Recover())
    e.Use(middleware.CORS())
    
    // Custom middleware
    e.Use(middlewares.RequestID())
    e.Use(middlewares.Logging())
    
    return e
}
```

### 2. Router Groups (`router.go`)

```go
// Public routes
public := e.Group("/api/v1")

// Authenticated routes
auth := public.Group("", middlewares.Auth())

// Workspace-scoped routes
workspace := auth.Group("/workspaces/:workspace_id",
    middlewares.WorkspaceAccess())
```

### 3. Middleware

#### Auth Middleware (`auth.go`)
- JWT validation
- User extraction из токена
- Permission checks
- Context enrichment с UserID

#### Workspace Middleware (`workspace.go`)
- Проверка доступа к workspace
- Извлечение workspace_id из пути
- Проверка членства пользователя
- Context enrichment с WorkspaceID

#### Rate Limiting (`rate_limit.go`)
- Redis-based rate limiter
- Per-user limits
- Per-endpoint limits
- Configurable windows и limits

#### Logging (`logging.go`)
- Request/response logging
- Performance metrics (latency)
- Error tracking
- Request ID propagation

#### CORS (`cors.go`)
- Configurable origins
- Allowed methods и headers
- Credentials support

#### Recovery (`recovery.go`)
- Panic recovery
- Stack trace logging
- Graceful error response

### 4. Response Helpers (`response.go`)

```go
func RespondJSON(c echo.Context, code int, data interface{}) error
func RespondError(c echo.Context, err error) error
func RespondValidationError(c echo.Context, err error) error
func RespondCreated(c echo.Context, data interface{}) error
func RespondNoContent(c echo.Context) error
```

---

## Критерии приёмки

- [ ] Echo server запускается корректно
- [ ] Middleware chain работает в правильном порядке
- [ ] CORS настроен для development и production
- [ ] Rate limiting работает с Redis backend
- [ ] Logging пишет structured logs в stdout
- [ ] Auth middleware валидирует JWT
- [ ] Workspace middleware проверяет доступ
- [ ] Recovery middleware ловит panics
- [ ] Response helpers упрощают работу с ответами
- [ ] Unit tests для каждого middleware
- [ ] Integration test для middleware chain

---

## Зависимости

### Входящие
- [03-http-server.md](03-http-server.md) — базовый HTTP server setup

### Исходящие
- [05-handlers-auth-workspace.md](05-handlers-auth-workspace.md) — использует middleware
- [06-handlers-chat-message.md](06-handlers-chat-message.md) — использует middleware
- [07-handlers-task-notification.md](07-handlers-task-notification.md) — использует middleware

---

## Заметки

- Используем Echo v4 built-in middleware где возможно
- Rate limiter хранит счётчики в Redis для распределённости
- JWT validation использует публичный ключ из Keycloak
- Logging использует structured JSON формат
- Recovery middleware не должен падать сам

---

*Создано: 2026-01-01*