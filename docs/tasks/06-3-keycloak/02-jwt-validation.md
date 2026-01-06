# 02: JWT Validation

**Приоритет:** 🔴 Critical
**Статус:** ✅ Завершено
**Зависит от:** [01-realm-setup.md](01-realm-setup.md)

---

## Описание

Реализовать офлайн валидацию JWT токенов через JWKS (JSON Web Key Set). Это позволит проверять токены без сетевых запросов к Keycloak на каждый request.

---

## Проблема

Текущая реализация:
```go
// AuthService.validateToken вызывает Keycloak на каждый запрос
userInfo, err := s.oauthClient.GetUserInfo(ctx, accessToken)
```

**Проблемы:**
- Latency: +50-100ms на каждый запрос
- Single point of failure: если Keycloak недоступен — все запросы fail
- Нагрузка: каждый API вызов = вызов Keycloak

---

## Решение

### JWKS-based validation

```
┌─────────────┐     ┌───────────────────┐     ┌─────────────┐
│   Client    │────>│   Token Validator │────>│ JWKS Cache  │
│  (Bearer)   │     │   (local)         │     │ (in-memory) │
└─────────────┘     └───────────────────┘     └──────┬──────┘
                                                     │
                                                     │ Refresh every 1h
                                                     v
                                              ┌─────────────┐
                                              │  Keycloak   │
                                              │  /certs     │
                                              └─────────────┘
```

---

## Файлы

```
internal/infrastructure/keycloak/
├── jwt_validator.go        # JWT валидатор
├── jwt_validator_test.go   # Тесты
└── oauth_client.go         # OAuth клиент (существующий)
```

---

## Реализация

### JWT Validator Interface

```go
// internal/infrastructure/keycloak/jwt_validator.go

package keycloak

import (
    "context"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/MicahParks/keyfunc/v3"
)

// TokenClaims represents validated JWT claims
type TokenClaims struct {
    UserID          string   `json:"sub"`
    Email           string   `json:"email"`
    EmailVerified   bool     `json:"email_verified"`
    Username        string   `json:"preferred_username"`
    Name            string   `json:"name"`
    GivenName       string   `json:"given_name"`
    FamilyName      string   `json:"family_name"`
    RealmRoles      []string // extracted from realm_access.roles
    Groups          []string `json:"groups"`
    SessionState    string   `json:"session_state"`
    IssuedAt        time.Time
    ExpiresAt       time.Time
}

// JWTValidator validates Keycloak JWT tokens
type JWTValidator interface {
    // Validate validates token and returns claims
    Validate(ctx context.Context, tokenString string) (*TokenClaims, error)

    // Close stops background JWKS refresh
    Close() error
}

// JWTValidatorConfig configuration for validator
type JWTValidatorConfig struct {
    KeycloakURL   string
    Realm         string
    ClientID      string        // Expected audience
    Leeway        time.Duration // Clock skew tolerance
    RefreshInterval time.Duration // JWKS refresh interval
}
```

### Error Types

```go
var (
    ErrInvalidToken    = errors.New("invalid token")
    ErrInvalidClaims   = errors.New("invalid claims")
    ErrMissingSubject  = errors.New("missing subject claim")
    ErrTokenExpired    = errors.New("token expired")
    ErrInvalidIssuer   = errors.New("invalid issuer")
    ErrInvalidAudience = errors.New("invalid audience")
    ErrJWKSFetchFailed = errors.New("failed to fetch JWKS")
)
```

---

## Конфигурация

### `config.yaml`

```yaml
keycloak:
  url: "http://localhost:8090"
  realm: "flowra"
  client_id: "flowra-backend"
  jwt:
    leeway: "30s"           # Допуск для clock skew
    refresh_interval: "1h"  # Интервал обновления JWKS
```

### Config Struct

```go
// internal/config/config.go

type KeycloakConfig struct {
    URL           string    `yaml:"url"`
    Realm         string    `yaml:"realm"`
    ClientID      string    `yaml:"client_id"`
    ClientSecret  string    `yaml:"client_secret"`
    AdminUsername string    `yaml:"admin_username"`
    AdminPassword string    `yaml:"admin_password"`
    JWT           JWTConfig `yaml:"jwt"`
}

type JWTConfig struct {
    Leeway          time.Duration `yaml:"leeway"`
    RefreshInterval time.Duration `yaml:"refresh_interval"`
}
```

---

## Чеклист

### Implementation
- [x] `JWTValidator` interface определён
- [x] `jwtValidator` struct реализован
- [x] JWKS кэширование работает (через jwkset.Storage)
- [x] Auto-refresh JWKS настроен (через RefreshInterval)
- [x] Claims extraction реализован

### Error Handling
- [x] Expired token handling
- [x] Invalid signature handling
- [x] Missing claims handling
- [x] JWKS fetch failure handling

### Testing
- [x] Unit tests с mock JWKS server
- [x] Performance benchmark (~63μs per validation)
- [ ] Integration test с реальным Keycloak

### Integration
- [ ] Container создаёт validator
- [x] Graceful shutdown (Close)
- [x] Logging добавлен

---

## Критерии приёмки

- [x] Токен валидируется без сетевых запросов (после initial JWKS fetch)
- [x] JWKS обновляется автоматически
- [x] Claims корректно извлекаются
- [x] Истёкшие токены reject'ятся
- [x] Invalid signature reject'ится
- [x] Latency < 1ms на валидацию (фактически ~63μs)

---

## Зависимости

### Входящие
- [01-realm-setup.md](01-realm-setup.md) — JWKS endpoint доступен

### Исходящие
- [03-token-middleware.md](03-token-middleware.md) — использует validator

### Go Dependencies

```go
require (
    github.com/golang-jwt/jwt/v5 v5.3.0
    github.com/MicahParks/keyfunc/v3 v3.7.0
    github.com/MicahParks/jwkset v0.11.0
)
```

---

*Обновлено: 2026-01-06*
