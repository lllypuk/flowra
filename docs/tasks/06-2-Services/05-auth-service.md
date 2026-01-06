# Task 05: AuthService

**Приоритет:** 🟡 High
**Статус:** Complete
**Зависит от:** Keycloak client (частично готов)

---

## Описание

Реализовать `AuthService` с полной интеграцией Keycloak для OAuth2 аутентификации. Сервис должен реализовать интерфейс `httphandler.AuthService` и заменить `MockAuthService`.

**Примечание:** Это наиболее сложная задача из всех сервисов, так как требует интеграции с внешней системой (Keycloak).

---

## Текущее состояние

### Mock реализация (internal/handler/http/auth_handler.go)

```go
type MockAuthService struct{}

func NewMockAuthService() *MockAuthService
func (m *MockAuthService) Login(ctx echo.Context, code, redirectURI string) (*LoginResult, error)
func (m *MockAuthService) Logout(ctx echo.Context, userID uuid.UUID) error
func (m *MockAuthService) RefreshToken(ctx echo.Context, refreshToken string) (*RefreshResult, error)
```

### Использование в container.go

```go
// container.go:421-423
c.Logger.Warn("AuthHandler: using mock implementation (real auth service not yet available)")
mockAuthService := httphandler.NewMockAuthService()
mockUserRepo := httphandler.NewMockUserRepository()
c.AuthHandler = httphandler.NewAuthHandler(mockAuthService, mockUserRepo)
```

### Существующий Keycloak client

Частичная реализация в `internal/application/workspace/keycloak_client.go`:
- Создание групп
- Добавление пользователей в группы
- Базовая OAuth2 конфигурация

---

## Интерфейс (internal/handler/http/auth_handler.go)

```go
type AuthService interface {
    Login(ctx echo.Context, code, redirectURI string) (*LoginResult, error)
    Logout(ctx echo.Context, userID uuid.UUID) error
    RefreshToken(ctx echo.Context, refreshToken string) (*RefreshResult, error)
}

type LoginResult struct {
    AccessToken  string
    RefreshToken string
    ExpiresIn    int
    User         *user.User
}

type RefreshResult struct {
    AccessToken  string
    RefreshToken string
    ExpiresIn    int
}
```

---

## Компоненты реализации

### 1. Keycloak OAuth2 Client

```go
// internal/infrastructure/keycloak/oauth_client.go

type OAuthClient struct {
    config       *oauth2.Config
    keycloakURL  string
    realm        string
    httpClient   *http.Client
}

func NewOAuthClient(cfg KeycloakConfig) *OAuthClient

// ExchangeCode обменивает authorization code на tokens
func (c *OAuthClient) ExchangeCode(ctx context.Context, code, redirectURI string) (*TokenResponse, error)

// RefreshToken обновляет access token
func (c *OAuthClient) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)

// RevokeToken отзывает refresh token
func (c *OAuthClient) RevokeToken(ctx context.Context, refreshToken string) error

// GetUserInfo получает информацию о пользователе
func (c *OAuthClient) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error)
```

### 2. Token Storage (Redis)

```go
// internal/infrastructure/auth/token_store.go

type TokenStore struct {
    redis *redis.Client
}

func NewTokenStore(redis *redis.Client) *TokenStore

// StoreRefreshToken сохраняет refresh token с привязкой к user
func (s *TokenStore) StoreRefreshToken(ctx context.Context, userID uuid.UUID, refreshToken string, ttl time.Duration) error

// GetRefreshToken получает refresh token
func (s *TokenStore) GetRefreshToken(ctx context.Context, userID uuid.UUID) (string, error)

// DeleteRefreshToken удаляет refresh token (logout)
func (s *TokenStore) DeleteRefreshToken(ctx context.Context, userID uuid.UUID) error

// IsTokenValid проверяет, что token не в blacklist
func (s *TokenStore) IsTokenValid(ctx context.Context, tokenID string) (bool, error)
```

### 3. AuthService реализация

```go
// internal/service/auth_service.go

package service

import (
    "context"
    "errors"

    "github.com/google/uuid"
    "github.com/labstack/echo/v4"
    httphandler "github.com/lllypuk/flowra/internal/handler/http"
    "github.com/lllypuk/flowra/internal/infrastructure/keycloak"
    userdomain "github.com/lllypuk/flowra/internal/domain/user"
)

type AuthService struct {
    oauthClient  *keycloak.OAuthClient
    tokenStore   *TokenStore
    userRepo     UserRepository
    logger       *slog.Logger
}

type AuthServiceConfig struct {
    OAuthClient *keycloak.OAuthClient
    TokenStore  *TokenStore
    UserRepo    UserRepository
    Logger      *slog.Logger
}

func NewAuthService(cfg AuthServiceConfig) *AuthService {
    return &AuthService{
        oauthClient: cfg.OAuthClient,
        tokenStore:  cfg.TokenStore,
        userRepo:    cfg.UserRepo,
        logger:      cfg.Logger,
    }
}

// Login выполняет OAuth2 authorization code flow.
func (s *AuthService) Login(
    ctx echo.Context,
    code, redirectURI string,
) (*httphandler.LoginResult, error) {
    // 1. Exchange code for tokens
    tokens, err := s.oauthClient.ExchangeCode(ctx.Request().Context(), code, redirectURI)
    if err != nil {
        return nil, fmt.Errorf("failed to exchange code: %w", err)
    }

    // 2. Get user info from Keycloak
    userInfo, err := s.oauthClient.GetUserInfo(ctx.Request().Context(), tokens.AccessToken)
    if err != nil {
        return nil, fmt.Errorf("failed to get user info: %w", err)
    }

    // 3. Find or create user in local DB
    user, err := s.findOrCreateUser(ctx.Request().Context(), userInfo)
    if err != nil {
        return nil, fmt.Errorf("failed to sync user: %w", err)
    }

    // 4. Store refresh token in Redis
    err = s.tokenStore.StoreRefreshToken(
        ctx.Request().Context(),
        user.ID(),
        tokens.RefreshToken,
        time.Duration(tokens.RefreshExpiresIn)*time.Second,
    )
    if err != nil {
        s.logger.Warn("failed to store refresh token", slog.String("error", err.Error()))
    }

    return &httphandler.LoginResult{
        AccessToken:  tokens.AccessToken,
        RefreshToken: tokens.RefreshToken,
        ExpiresIn:    tokens.ExpiresIn,
        User:         user,
    }, nil
}

// Logout инвалидирует сессию пользователя.
func (s *AuthService) Logout(
    ctx echo.Context,
    userID uuid.UUID,
) error {
    // 1. Get stored refresh token
    refreshToken, err := s.tokenStore.GetRefreshToken(ctx.Request().Context(), userID)
    if err != nil && !errors.Is(err, redis.Nil) {
        return fmt.Errorf("failed to get refresh token: %w", err)
    }

    // 2. Revoke token in Keycloak
    if refreshToken != "" {
        if err := s.oauthClient.RevokeToken(ctx.Request().Context(), refreshToken); err != nil {
            s.logger.Warn("failed to revoke token in Keycloak", slog.String("error", err.Error()))
        }
    }

    // 3. Delete from Redis
    if err := s.tokenStore.DeleteRefreshToken(ctx.Request().Context(), userID); err != nil {
        return fmt.Errorf("failed to delete refresh token: %w", err)
    }

    return nil
}

// RefreshToken обновляет access token.
func (s *AuthService) RefreshToken(
    ctx echo.Context,
    refreshToken string,
) (*httphandler.RefreshResult, error) {
    // 1. Refresh tokens in Keycloak
    tokens, err := s.oauthClient.RefreshToken(ctx.Request().Context(), refreshToken)
    if err != nil {
        return nil, fmt.Errorf("failed to refresh token: %w", err)
    }

    return &httphandler.RefreshResult{
        AccessToken:  tokens.AccessToken,
        RefreshToken: tokens.RefreshToken,
        ExpiresIn:    tokens.ExpiresIn,
    }, nil
}

// findOrCreateUser синхронизирует пользователя из Keycloak в локальную БД.
func (s *AuthService) findOrCreateUser(
    ctx context.Context,
    info *keycloak.UserInfo,
) (*userdomain.User, error) {
    // Попробовать найти по external ID (Keycloak sub)
    user, err := s.userRepo.FindByExternalID(ctx, info.Sub)
    if err == nil && user != nil {
        // Обновить данные если изменились
        if user.Email() != info.Email || user.DisplayName() != info.Name {
            user.Update(info.Email, info.Name)
            if err := s.userRepo.Save(ctx, user); err != nil {
                return nil, err
            }
        }
        return user, nil
    }

    // Создать нового пользователя
    user = userdomain.NewUser(
        info.Sub,           // externalID
        info.PreferredUsername,
        info.Email,
        info.Name,          // displayName
    )

    if err := s.userRepo.Save(ctx, user); err != nil {
        return nil, err
    }

    return user, nil
}
```

---

## Keycloak Configuration

```go
// internal/config/config.go

type KeycloakConfig struct {
    URL          string `env:"KEYCLOAK_URL" default:"http://localhost:8090"`
    Realm        string `env:"KEYCLOAK_REALM" default:"flowra"`
    ClientID     string `env:"KEYCLOAK_CLIENT_ID" default:"flowra-api"`
    ClientSecret string `env:"KEYCLOAK_CLIENT_SECRET"`
    AdminUser    string `env:"KEYCLOAK_ADMIN_USER" default:"admin"`
    AdminPass    string `env:"KEYCLOAK_ADMIN_PASSWORD"`
}
```

---

## OAuth2 Flow

```
┌─────────┐     ┌─────────┐     ┌──────────┐     ┌─────────┐
│ Browser │     │ Frontend│     │ API      │     │Keycloak │
└────┬────┘     └────┬────┘     └────┬─────┘     └────┬────┘
     │               │               │                │
     │  1. Click Login               │                │
     ├──────────────►│               │                │
     │               │               │                │
     │  2. Redirect to Keycloak      │                │
     │◄──────────────┤               │                │
     │               │               │                │
     │  3. Login Form                │                │
     ├───────────────────────────────────────────────►│
     │               │               │                │
     │  4. Authorization Code        │                │
     │◄───────────────────────────────────────────────┤
     │               │               │                │
     │  5. Callback with code        │                │
     ├──────────────►│               │                │
     │               │               │                │
     │               │ 6. POST /auth/login            │
     │               ├──────────────►│                │
     │               │               │                │
     │               │               │ 7. Exchange code
     │               │               ├───────────────►│
     │               │               │                │
     │               │               │ 8. Tokens      │
     │               │               │◄───────────────┤
     │               │               │                │
     │               │ 9. User + Tokens               │
     │               │◄──────────────┤                │
     │               │               │                │
     │ 10. Store tokens, redirect    │                │
     │◄──────────────┤               │                │
```

---

## Зависимости

### Внешние
- Keycloak server
- Redis для token storage

### Внутренние
- `UserRepository` для синхронизации пользователей
- `config.KeycloakConfig` для конфигурации

---

## Тестирование

### Unit tests (с mocked Keycloak)

```go
func TestAuthService_Login(t *testing.T) {
    // 1. Successful login → creates user, stores token
    // 2. Invalid code → error
    // 3. Keycloak unavailable → error
    // 4. Existing user → updates and returns
}

func TestAuthService_Logout(t *testing.T) {
    // 1. Successfully logout
    // 2. Token not found → still succeeds (idempotent)
    // 3. Keycloak revoke fails → logs warning, deletes locally
}

func TestAuthService_RefreshToken(t *testing.T) {
    // 1. Successful refresh
    // 2. Invalid/expired refresh token → error
}
```

### Integration tests

Использовать testcontainers с Keycloak image.

---

## Чеклист

### Keycloak OAuth Client
- [x] Создать `internal/infrastructure/keycloak/oauth_client.go`
- [x] Реализовать `ExchangeCode()`
- [x] Реализовать `RefreshToken()`
- [x] Реализовать `RevokeToken()`
- [x] Реализовать `GetUserInfo()`

### Token Store
- [x] Создать `internal/infrastructure/auth/token_store.go`
- [x] Реализовать `StoreRefreshToken()`
- [x] Реализовать `GetRefreshToken()`
- [x] Реализовать `DeleteRefreshToken()`

### AuthService
- [x] Создать `internal/service/auth_service.go`
- [x] Реализовать `Login()`
- [x] Реализовать `Logout()`
- [x] Реализовать `RefreshToken()`
- [x] Реализовать `findOrCreateUser()`

### Интеграция
- [x] Добавить `KeycloakConfig` в config (уже существует)
- [ ] Обновить `container.go` (Task 06)
- [x] Написать unit tests
- [x] Написать integration tests

---

## Критерии приёмки

- [x] OAuth2 authorization code flow работает
- [x] Пользователи синхронизируются из Keycloak в MongoDB
- [x] Refresh token сохраняется в Redis
- [x] Logout корректно инвалидирует сессию
- [x] Unit test coverage > 80%

---

## Альтернативный подход (упрощённый)

Если полная Keycloak интеграция слишком сложна на первом этапе:

1. **JWT-only mode:** Доверять JWT токенам от Keycloak, не хранить refresh tokens
2. **Stateless auth:** Не синхронизировать пользователей, брать данные из JWT claims
3. **Mock Keycloak:** Использовать mock для development, real для production

```go
// Упрощённый AuthService без token storage
type StatelessAuthService struct {
    jwtValidator *middleware.JWTValidator
    userRepo     UserRepository
}

func (s *StatelessAuthService) Login(ctx echo.Context, code, redirectURI string) (*LoginResult, error) {
    // Просто валидировать токен и вернуть user info из claims
}
```

---

## Заметки

- Keycloak должен быть настроен с правильным client (confidential, authorization code flow)
- PKCE рекомендуется для дополнительной безопасности
- Рассмотреть добавление rate limiting для auth endpoints
- Для production: использовать HTTPS, secure cookies, proper CORS

---

*Создано: 2026-01-06*
