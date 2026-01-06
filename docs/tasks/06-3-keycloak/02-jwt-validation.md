# 02: JWT Validation

**Приоритет:** 🔴 Critical
**Статус:** ⏳ Не начато
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
└── jwks_cache.go           # JWKS кэш
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
    "github.com/MicahParks/keyfunc/v2"
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

### Implementation

```go
type jwtValidator struct {
    jwks      *keyfunc.JWKS
    config    JWTValidatorConfig
    issuerURL string
}

func NewJWTValidator(config JWTValidatorConfig) (JWTValidator, error) {
    issuerURL := fmt.Sprintf("%s/realms/%s", config.KeycloakURL, config.Realm)
    jwksURL := fmt.Sprintf("%s/protocol/openid-connect/certs", issuerURL)

    // Configure JWKS with auto-refresh
    jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{
        RefreshInterval:   config.RefreshInterval,
        RefreshRateLimit:  time.Minute * 5,
        RefreshTimeout:    time.Second * 10,
        RefreshUnknownKID: true,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create JWKS: %w", err)
    }

    return &jwtValidator{
        jwks:      jwks,
        config:    config,
        issuerURL: issuerURL,
    }, nil
}

func (v *jwtValidator) Validate(ctx context.Context, tokenString string) (*TokenClaims, error) {
    // Parse and validate token
    token, err := jwt.Parse(tokenString, v.jwks.Keyfunc,
        jwt.WithIssuer(v.issuerURL),
        jwt.WithAudience(v.config.ClientID),
        jwt.WithLeeway(v.config.Leeway),
        jwt.WithIssuedAt(),
        jwt.WithExpirationRequired(),
    )
    if err != nil {
        return nil, fmt.Errorf("invalid token: %w", err)
    }

    if !token.Valid {
        return nil, ErrInvalidToken
    }

    // Extract claims
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return nil, ErrInvalidClaims
    }

    return v.extractClaims(claims)
}

func (v *jwtValidator) extractClaims(claims jwt.MapClaims) (*TokenClaims, error) {
    tc := &TokenClaims{}

    // Required claims
    tc.UserID, _ = claims["sub"].(string)
    if tc.UserID == "" {
        return nil, ErrMissingSubject
    }

    // Optional claims
    tc.Email, _ = claims["email"].(string)
    tc.EmailVerified, _ = claims["email_verified"].(bool)
    tc.Username, _ = claims["preferred_username"].(string)
    tc.Name, _ = claims["name"].(string)
    tc.GivenName, _ = claims["given_name"].(string)
    tc.FamilyName, _ = claims["family_name"].(string)
    tc.SessionState, _ = claims["session_state"].(string)

    // Extract realm roles
    if realmAccess, ok := claims["realm_access"].(map[string]interface{}); ok {
        if roles, ok := realmAccess["roles"].([]interface{}); ok {
            for _, role := range roles {
                if r, ok := role.(string); ok {
                    tc.RealmRoles = append(tc.RealmRoles, r)
                }
            }
        }
    }

    // Extract groups
    if groups, ok := claims["groups"].([]interface{}); ok {
        for _, group := range groups {
            if g, ok := group.(string); ok {
                tc.Groups = append(tc.Groups, g)
            }
        }
    }

    // Time claims
    if iat, ok := claims["iat"].(float64); ok {
        tc.IssuedAt = time.Unix(int64(iat), 0)
    }
    if exp, ok := claims["exp"].(float64); ok {
        tc.ExpiresAt = time.Unix(int64(exp), 0)
    }

    return tc, nil
}

func (v *jwtValidator) Close() error {
    v.jwks.EndBackground()
    return nil
}
```

### Error Types

```go
var (
    ErrInvalidToken   = errors.New("invalid token")
    ErrInvalidClaims  = errors.New("invalid claims")
    ErrMissingSubject = errors.New("missing subject claim")
    ErrTokenExpired   = errors.New("token expired")
    ErrInvalidIssuer  = errors.New("invalid issuer")
    ErrInvalidAudience = errors.New("invalid audience")
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

### Container Integration

```go
// cmd/api/container.go

func (c *Container) createJWTValidator() keycloak.JWTValidator {
    validator, err := keycloak.NewJWTValidator(keycloak.JWTValidatorConfig{
        KeycloakURL:     c.Config.Keycloak.URL,
        Realm:           c.Config.Keycloak.Realm,
        ClientID:        c.Config.Keycloak.ClientID,
        Leeway:          c.Config.Keycloak.JWT.Leeway,
        RefreshInterval: c.Config.Keycloak.JWT.RefreshInterval,
    })
    if err != nil {
        c.Logger.Fatal("Failed to create JWT validator", zap.Error(err))
    }
    return validator
}
```

---

## Чеклист

### Implementation
- [ ] `JWTValidator` interface определён
- [ ] `jwtValidator` struct реализован
- [ ] JWKS кэширование работает
- [ ] Auto-refresh JWKS настроен
- [ ] Claims extraction реализован

### Error Handling
- [ ] Expired token handling
- [ ] Invalid signature handling
- [ ] Missing claims handling
- [ ] JWKS fetch failure handling

### Testing
- [ ] Unit tests с mock JWKS
- [ ] Integration test с реальным Keycloak
- [ ] Performance benchmark

### Integration
- [ ] Container создаёт validator
- [ ] Graceful shutdown (Close)
- [ ] Logging добавлен

---

## Критерии приёмки

- [ ] Токен валидируется без сетевых запросов (после initial JWKS fetch)
- [ ] JWKS обновляется автоматически
- [ ] Claims корректно извлекаются
- [ ] Истёкшие токены reject'ятся
- [ ] Invalid signature reject'ится
- [ ] Latency < 1ms на валидацию

---

## Зависимости

### Входящие
- [01-realm-setup.md](01-realm-setup.md) — JWKS endpoint доступен

### Исходящие
- [03-token-middleware.md](03-token-middleware.md) — использует validator

### Go Dependencies

```go
require (
    github.com/golang-jwt/jwt/v5 v5.2.0
    github.com/MicahParks/keyfunc/v2 v2.1.0
)
```

---

*Обновлено: 2026-01-06*
