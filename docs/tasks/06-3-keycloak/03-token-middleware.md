# 03: Token Middleware

**Приоритет:** 🔴 Critical
**Статус:** ⏳ Не начато
**Зависит от:** [02-jwt-validation.md](02-jwt-validation.md)

---

## Описание

Реализовать Echo middleware для автоматической валидации Bearer токенов в HTTP заголовках. Middleware извлекает токен, валидирует через JWTValidator, и добавляет claims в context.

---

## Текущее состояние

Сейчас защита routes реализована вручную:

```go
// Каждый handler проверяет токен сам
func (h *Handler) SomeProtectedEndpoint(c echo.Context) error {
    // Manual token extraction
    token := extractBearerToken(c)
    // Manual validation
    user, err := h.authService.ValidateToken(ctx, token)
    // ...
}
```

**Проблемы:**
- Дублирование кода в каждом handler
- Легко забыть добавить проверку
- Нет стандартизации

---

## Решение

### Middleware Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                      HTTP Request                             │
│              Authorization: Bearer eyJhbGciOi...              │
└────────────────────────────┬─────────────────────────────────┘
                             │
                             v
                   ┌─────────────────────┐
                   │   Auth Middleware   │
                   │   ┌───────────────┐ │
                   │   │ Extract Token │ │
                   │   └───────┬───────┘ │
                   │           │         │
                   │   ┌───────v───────┐ │
                   │   │ JWT Validator │ │
                   │   └───────┬───────┘ │
                   │           │         │
                   │   ┌───────v───────┐ │
                   │   │ Set Context   │ │
                   │   └───────────────┘ │
                   └─────────┬───────────┘
                             │
                             v
                   ┌─────────────────────┐
                   │   Protected Handler │
                   │   c.Get("user")     │
                   └─────────────────────┘
```

---

## Файлы

```
internal/middleware/
├── auth.go           # Auth middleware
├── auth_test.go      # Tests
└── context_keys.go   # Context key constants
```

---

## Реализация

### Context Keys

```go
// internal/middleware/context_keys.go

package middleware

type contextKey string

const (
    // UserKey is the context key for authenticated user claims
    UserKey contextKey = "user"

    // TokenKey is the context key for raw JWT token
    TokenKey contextKey = "token"
)
```

### Auth Middleware

```go
// internal/middleware/auth.go

package middleware

import (
    "net/http"
    "strings"

    "github.com/labstack/echo/v4"
    "github.com/lllypuk/flowra/internal/infrastructure/keycloak"
)

// AuthConfig configuration for auth middleware
type AuthConfig struct {
    // Validator is the JWT validator
    Validator keycloak.JWTValidator

    // Skipper defines a function to skip middleware
    Skipper func(c echo.Context) bool

    // TokenLookup is the header to look for token
    // Default: "header:Authorization"
    TokenLookup string

    // AuthScheme is the auth scheme in header
    // Default: "Bearer"
    AuthScheme string

    // ContextKey is the key to store user in context
    // Default: "user"
    ContextKey string

    // ErrorHandler is called when auth fails
    ErrorHandler func(c echo.Context, err error) error
}

// DefaultAuthConfig default configuration
var DefaultAuthConfig = AuthConfig{
    Skipper:     func(c echo.Context) bool { return false },
    TokenLookup: "header:Authorization",
    AuthScheme:  "Bearer",
    ContextKey:  "user",
    ErrorHandler: func(c echo.Context, err error) error {
        return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
    },
}

// Auth returns auth middleware
func Auth(validator keycloak.JWTValidator) echo.MiddlewareFunc {
    config := DefaultAuthConfig
    config.Validator = validator
    return AuthWithConfig(config)
}

// AuthWithConfig returns auth middleware with config
func AuthWithConfig(config AuthConfig) echo.MiddlewareFunc {
    if config.Validator == nil {
        panic("auth middleware requires validator")
    }
    if config.Skipper == nil {
        config.Skipper = DefaultAuthConfig.Skipper
    }
    if config.TokenLookup == "" {
        config.TokenLookup = DefaultAuthConfig.TokenLookup
    }
    if config.AuthScheme == "" {
        config.AuthScheme = DefaultAuthConfig.AuthScheme
    }
    if config.ContextKey == "" {
        config.ContextKey = DefaultAuthConfig.ContextKey
    }
    if config.ErrorHandler == nil {
        config.ErrorHandler = DefaultAuthConfig.ErrorHandler
    }

    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Skip if configured
            if config.Skipper(c) {
                return next(c)
            }

            // Extract token
            token, err := extractToken(c, config)
            if err != nil {
                return config.ErrorHandler(c, err)
            }

            // Validate token
            claims, err := config.Validator.Validate(c.Request().Context(), token)
            if err != nil {
                return config.ErrorHandler(c, err)
            }

            // Store in context
            c.Set(config.ContextKey, claims)
            c.Set(string(TokenKey), token)

            return next(c)
        }
    }
}

func extractToken(c echo.Context, config AuthConfig) (string, error) {
    parts := strings.Split(config.TokenLookup, ":")
    if len(parts) != 2 {
        return "", ErrInvalidTokenLookup
    }

    switch parts[0] {
    case "header":
        return extractFromHeader(c, parts[1], config.AuthScheme)
    case "query":
        return extractFromQuery(c, parts[1])
    case "cookie":
        return extractFromCookie(c, parts[1])
    default:
        return "", ErrInvalidTokenLookup
    }
}

func extractFromHeader(c echo.Context, header, scheme string) (string, error) {
    auth := c.Request().Header.Get(header)
    if auth == "" {
        return "", ErrMissingToken
    }

    parts := strings.SplitN(auth, " ", 2)
    if len(parts) != 2 || !strings.EqualFold(parts[0], scheme) {
        return "", ErrInvalidAuthHeader
    }

    return parts[1], nil
}

func extractFromQuery(c echo.Context, param string) (string, error) {
    token := c.QueryParam(param)
    if token == "" {
        return "", ErrMissingToken
    }
    return token, nil
}

func extractFromCookie(c echo.Context, name string) (string, error) {
    cookie, err := c.Cookie(name)
    if err != nil {
        return "", ErrMissingToken
    }
    return cookie.Value, nil
}

// Errors
var (
    ErrMissingToken       = errors.New("missing token")
    ErrInvalidAuthHeader  = errors.New("invalid authorization header")
    ErrInvalidTokenLookup = errors.New("invalid token lookup")
)
```

### Helper Functions

```go
// internal/middleware/auth_helpers.go

// GetUser returns user claims from context
func GetUser(c echo.Context) *keycloak.TokenClaims {
    user, ok := c.Get("user").(*keycloak.TokenClaims)
    if !ok {
        return nil
    }
    return user
}

// GetUserID returns user ID from context
func GetUserID(c echo.Context) string {
    user := GetUser(c)
    if user == nil {
        return ""
    }
    return user.UserID
}

// HasRole checks if user has a role
func HasRole(c echo.Context, role string) bool {
    user := GetUser(c)
    if user == nil {
        return false
    }
    for _, r := range user.RealmRoles {
        if r == role {
            return true
        }
    }
    return false
}

// RequireRole returns middleware that requires a role
func RequireRole(role string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            if !HasRole(c, role) {
                return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
            }
            return next(c)
        }
    }
}

// InGroup checks if user is in a group
func InGroup(c echo.Context, group string) bool {
    user := GetUser(c)
    if user == nil {
        return false
    }
    for _, g := range user.Groups {
        if g == group || g == "/"+group {
            return true
        }
    }
    return false
}
```

---

## Использование

### Route Configuration

```go
// cmd/api/routes.go

func SetupRoutes(e *echo.Echo, h *Handlers, validator keycloak.JWTValidator) {
    // Public routes
    e.GET("/health", h.Health)
    e.POST("/auth/login", h.AuthHandler.Login)

    // Protected routes - require valid token
    api := e.Group("/api/v1")
    api.Use(middleware.Auth(validator))

    // Workspace routes
    api.GET("/workspaces", h.WorkspaceHandler.List)
    api.POST("/workspaces", h.WorkspaceHandler.Create)

    // Admin routes - require admin role
    admin := api.Group("/admin")
    admin.Use(middleware.RequireRole("admin"))
    admin.GET("/users", h.AdminHandler.ListUsers)
}
```

### In Handlers

```go
func (h *WorkspaceHandler) Create(c echo.Context) error {
    // Get authenticated user
    user := middleware.GetUser(c)
    if user == nil {
        return echo.NewHTTPError(http.StatusUnauthorized)
    }

    // Use user info
    userID, _ := uuid.Parse(user.UserID)

    result, err := h.service.CreateWorkspace(c.Request().Context(), userID, req.Name)
    // ...
}
```

---

## Чеклист

### Implementation
- [ ] `Auth` middleware реализован
- [ ] Token extraction (header, query, cookie)
- [ ] Context storage работает
- [ ] Error handling настроен
- [ ] Skipper function работает

### Helper Functions
- [ ] `GetUser` реализован
- [ ] `GetUserID` реализован
- [ ] `HasRole` реализован
- [ ] `RequireRole` middleware реализован
- [ ] `InGroup` реализован

### Testing
- [ ] Unit tests для middleware
- [ ] Unit tests для helpers
- [ ] Integration test с real validator

### Integration
- [ ] Routes используют middleware
- [ ] Handlers используют helpers
- [ ] Error responses стандартизированы

---

## Критерии приёмки

- [ ] Protected routes возвращают 401 без токена
- [ ] Protected routes возвращают 401 с invalid токеном
- [ ] Valid токен позволяет доступ
- [ ] User claims доступны в handlers
- [ ] Role-based access работает
- [ ] Custom error handler работает

---

## Зависимости

### Входящие
- [02-jwt-validation.md](02-jwt-validation.md) — JWTValidator

### Исходящие
- [04-group-management.md](04-group-management.md) — использует middleware для авторизации
- Frontend tasks — требуют защищённые endpoints

---

*Обновлено: 2026-01-06*
