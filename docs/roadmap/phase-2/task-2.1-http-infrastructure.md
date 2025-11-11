# Task 2.1: HTTP Infrastructure (Echo + Middleware)

**Приоритет:** 🔴 КРИТИЧЕСКИЙ
**Статус:** Blocked
**Время:** 4-5 дней
**Зависимости:** Phase 1 завершена

---

## Объединяет Tasks

- Task 2.1.1: Echo Framework Setup (1-2 дня)
- Task 2.1.2: Middleware Implementation (3-4 дня)

---

## Цель

Настроить Echo v4 router со всеми необходимыми middleware: auth, authorization, rate limiting, logging, CORS.

---

## Файлы для создания

```
internal/handler/http/
├── router.go              (Echo setup + routes)
├── response.go            (response helpers)
├── request.go             (request helpers)
└── router_test.go

internal/middleware/
├── auth.go                (JWT validation)
├── auth_test.go
├── workspace.go           (workspace access check)
├── workspace_test.go
├── chat.go                (chat participant check)
├── ratelimit.go           (rate limiting)
├── requestid.go           (request ID injection)
├── logging.go             (structured logging)
└── cors.go                (CORS headers)
```

---

## 1. Router Setup (router.go)

```go
package http

import (
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"

    custommw "github.com/lllypuk/flowra/internal/middleware"
)

func NewRouter(
    authHandler *AuthHandler,
    chatHandler *ChatHandler,
    messageHandler *MessageHandler,
    workspaceHandler *WorkspaceHandler,
    notificationHandler *NotificationHandler,
    wsHandler *WebSocketHandler,
    authMiddleware *custommw.AuthMiddleware,
    workspaceMiddleware *custommw.WorkspaceMiddleware,
    rateLimiter *custommw.RateLimiter,
    logger *slog.Logger,
) *echo.Echo {
    e := echo.New()

    // Global middleware
    e.Use(middleware.Recover())
    e.Use(custommw.RequestID())
    e.Use(custommw.Logging(logger))
    e.Use(middleware.CORS())

    // Public routes
    e.GET("/health", HealthCheck)
    e.GET("/metrics", PrometheusMetrics)

    // Auth routes (no auth required)
    auth := e.Group("/auth")
    auth.GET("/login", authHandler.Login)
    auth.GET("/callback", authHandler.Callback)
    auth.POST("/logout", authHandler.Logout)

    // API routes (require auth)
    api := e.Group("/api/v1")
    api.Use(authMiddleware.Authenticate())
    api.Use(rateLimiter.Limit(100, 1*time.Minute))

    // Workspace routes
    workspaces := api.Group("/workspaces")
    workspaces.POST("", workspaceHandler.Create)
    workspaces.GET("", workspaceHandler.List)
    workspaces.GET("/:id", workspaceHandler.Get)

    // Workspace-scoped routes (check membership)
    ws := api.Group("/workspaces/:workspaceId")
    ws.Use(workspaceMiddleware.CheckAccess())

    // Chat routes
    chats := ws.Group("/chats")
    chats.POST("", chatHandler.Create)
    chats.GET("", chatHandler.List)
    chats.GET("/:chatId", chatHandler.Get)
    chats.POST("/:chatId/participants", chatHandler.AddParticipant)
    chats.DELETE("/:chatId/participants/:userId", chatHandler.RemoveParticipant)
    chats.PUT("/:chatId/status", chatHandler.ChangeStatus)
    chats.PUT("/:chatId/assignee", chatHandler.AssignUser)

    // Message routes
    messages := api.Group("/chats/:chatId/messages")
    messages.POST("", messageHandler.Send)
    messages.GET("", messageHandler.List)
    messages.PUT("/:messageId", messageHandler.Edit)
    messages.DELETE("/:messageId", messageHandler.Delete)

    // Notification routes
    notifications := api.Group("/notifications")
    notifications.GET("", notificationHandler.List)
    notifications.PUT("/:id/read", notificationHandler.MarkAsRead)

    // WebSocket
    e.GET("/ws", wsHandler.ServeWS)

    return e
}
```

---

## 2. Auth Middleware (auth.go)

```go
package middleware

import (
    "github.com/labstack/echo/v4"
    "github.com/lllypuk/flowra/internal/infrastructure/keycloak"
    "github.com/lllypuk/flowra/internal/application/shared"
)

type AuthMiddleware struct {
    tokenValidator *keycloak.TokenValidator
    userRepo       shared.UserRepository
}

func (m *AuthMiddleware) Authenticate() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // 1. Extract token
            token := extractToken(c)
            if token == "" {
                return echo.NewHTTPError(401, "Missing authorization token")
            }

            // 2. Validate token
            claims, err := m.tokenValidator.Validate(token)
            if err != nil {
                return echo.NewHTTPError(401, "Invalid token")
            }

            // 3. Load user
            user, err := m.userRepo.FindByID(c.Request().Context(), claims.UserID)
            if err != nil {
                return echo.NewHTTPError(401, "User not found")
            }

            // 4. Set user in context
            ctx := shared.WithUserID(c.Request().Context(), user.ID)
            ctx = shared.WithUser(ctx, user)
            c.SetRequest(c.Request().WithContext(ctx))

            return next(c)
        }
    }
}

func extractToken(c echo.Context) string {
    // Try Authorization header first
    auth := c.Request().Header.Get("Authorization")
    if auth != "" && strings.HasPrefix(auth, "Bearer ") {
        return auth[7:]
    }

    // Try session cookie
    cookie, err := c.Cookie("session_id")
    if err == nil {
        // Load session and get access token
        // ...
    }

    return ""
}
```

---

## 3. Workspace Access Middleware (workspace.go)

```go
type WorkspaceMiddleware struct {
    workspaceRepo WorkspaceRepository
}

func (m *WorkspaceMiddleware) CheckAccess() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            workspaceID := parseUUID(c.Param("workspaceId"))
            userID := shared.UserIDFromContext(c.Request().Context())

            // Check membership
            isMember, err := m.workspaceRepo.IsMember(c.Request().Context(), workspaceID, userID)
            if err != nil || !isMember {
                return echo.NewHTTPError(403, "Access denied to workspace")
            }

            // Set workspace in context
            ctx := shared.WithWorkspaceID(c.Request().Context(), workspaceID)
            c.SetRequest(c.Request().WithContext(ctx))

            return next(c)
        }
    }
}
```

---

## 4. Rate Limiting (ratelimit.go)

```go
type RateLimiter struct {
    redis *redis.Client
}

func (m *RateLimiter) Limit(max int, window time.Duration) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            userID := shared.UserIDFromContext(c.Request().Context())
            key := fmt.Sprintf("ratelimit:%s", userID.String())

            // Increment counter
            count, err := m.redis.Incr(c.Request().Context(), key).Result()
            if err != nil {
                return err
            }

            // Set expiry on first request
            if count == 1 {
                m.redis.Expire(c.Request().Context(), key, window)
            }

            // Check limit
            if count > int64(max) {
                return echo.NewHTTPError(429, "Rate limit exceeded")
            }

            // Add rate limit headers
            c.Response().Header().Set("X-RateLimit-Limit", fmt.Sprint(max))
            c.Response().Header().Set("X-RateLimit-Remaining", fmt.Sprint(max-int(count)))

            return next(c)
        }
    }
}
```

---

## 5. Request ID (requestid.go)

```go
func RequestID() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            requestID := c.Request().Header.Get("X-Request-ID")
            if requestID == "" {
                requestID = uuid.New().String()
            }

            c.Response().Header().Set("X-Request-ID", requestID)

            ctx := shared.WithCorrelationID(c.Request().Context(), requestID)
            c.SetRequest(c.Request().WithContext(ctx))

            return next(c)
        }
    }
}
```

---

## 6. Logging Middleware (logging.go)

```go
func Logging(logger *slog.Logger) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            start := time.Now()

            err := next(c)

            duration := time.Since(start)

            logger.Info("HTTP request",
                "method", c.Request().Method,
                "path", c.Request().URL.Path,
                "status", c.Response().Status,
                "duration_ms", duration.Milliseconds(),
                "request_id", c.Response().Header().Get("X-Request-ID"),
            )

            return err
        }
    }
}
```

---

## 7. Response/Request Helpers

```go
// response.go
func RespondJSON(c echo.Context, status int, data interface{}) error {
    return c.JSON(status, data)
}

func RespondError(c echo.Context, err error) error {
    // Handle different error types
    switch e := err.(type) {
    case *shared.NotFoundError:
        return c.JSON(404, map[string]string{"error": e.Error()})
    case *shared.ValidationError:
        return c.JSON(400, map[string]string{"error": e.Error()})
    case *shared.UnauthorizedError:
        return c.JSON(403, map[string]string{"error": e.Error()})
    default:
        return c.JSON(500, map[string]string{"error": "Internal server error"})
    }
}

// request.go
func GetUserID(c echo.Context) uuid.UUID {
    return shared.UserIDFromContext(c.Request().Context())
}

func GetWorkspaceID(c echo.Context) uuid.UUID {
    return shared.WorkspaceIDFromContext(c.Request().Context())
}

func BindAndValidate(c echo.Context, req interface{}) error {
    if err := c.Bind(req); err != nil {
        return &shared.ValidationError{Message: "Invalid request body"}
    }

    // TODO: Add validation
    return nil
}
```

---

## Тестирование

```go
func TestAuthMiddleware_ValidToken(t *testing.T) {
    // Setup
    e := echo.New()
    req := httptest.NewRequest(http.MethodGet, "/", nil)
    req.Header.Set("Authorization", "Bearer valid-token")
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    // Mock validator
    validator := &MockTokenValidator{}
    validator.On("Validate", "valid-token").Return(&Claims{UserID: uuid.New()}, nil)

    middleware := &AuthMiddleware{tokenValidator: validator}

    // Execute
    handler := middleware.Authenticate()(func(c echo.Context) error {
        return c.String(200, "OK")
    })

    err := handler(c)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, 200, rec.Code)
}
```

---

## Критерии успеха

- ✅ **Echo router configured**
- ✅ **Auth middleware validates JWT**
- ✅ **Authorization middleware protects resources**
- ✅ **Rate limiting prevents abuse**
- ✅ **Logging logs all requests**
- ✅ **CORS configured**
- ✅ **Test coverage >80%**

---

## Следующий шаг

→ **Task 2.2: HTTP Handlers Implementation**
