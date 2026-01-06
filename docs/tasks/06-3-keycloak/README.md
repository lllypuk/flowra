# Keycloak Integration

**Цель:** Полная интеграция с Keycloak для SSO, управления группами и авторизации
**Статус:** ✅ Завершено (100%)

---

## Текущее состояние

### Реализовано

| Компонент | Файл | Статус |
|-----------|------|--------|
| OAuth2 Client | `internal/infrastructure/keycloak/oauth_client.go` | ✅ Готово |
| Token Store (Redis) | `internal/infrastructure/auth/token_store.go` | ✅ Готово |
| AuthService | `internal/service/auth_service.go` | ✅ Готово |
| Auth Handler | `internal/handler/http/auth_handler.go` | ✅ Готово |
| Docker Compose | `docker-compose.yml` (Keycloak v23) | ✅ Готово |
| NoOp Keycloak Client | `internal/application/workspace/noop_keycloak_client.go` | ✅ Готово |
| Realm Setup | `configs/keycloak/realm-export.json` | ✅ Готово |
| JWT Validator | `internal/infrastructure/keycloak/jwt_validator.go` | ✅ Готово |
| Admin Token Manager | `internal/infrastructure/keycloak/admin_token.go` | ✅ Готово |
| Group Client | `internal/infrastructure/keycloak/group_client.go` | ✅ Готово |
| User Client | `internal/infrastructure/keycloak/user_client.go` | ✅ Готово |
| Auth Middleware | `internal/handler/http/auth_middleware.go` | ✅ Готово |

---

## Структура задач

| № | Задача | Файл | Приоритет | Статус |
|---|--------|------|-----------|--------|
| 01 | Keycloak Realm Setup | [01-realm-setup.md](01-realm-setup.md) | 🔴 Critical | ✅ |
| 02 | JWT Validation | [02-jwt-validation.md](02-jwt-validation.md) | 🔴 Critical | ✅ |
| 03 | Token Middleware | [03-token-middleware.md](03-token-middleware.md) | 🔴 Critical | ✅ |
| 04 | Group Management | [04-group-management.md](04-group-management.md) | 🟡 High | ✅ |
| 05 | User Sync | [05-user-sync.md](05-user-sync.md) | 🟢 Medium | ✅ |
| 06 | Integration Tests | [06-integration-tests.md](06-integration-tests.md) | 🟡 High | ✅ |

---

## Зависимости между задачами

```
[01 Realm Setup] ✅
       │
       ├──> [02 JWT Validation] ✅
       │           │
       │           v
       └──> [03 Token Middleware] ✅
                   │
       ┌───────────┴───────────┐
       v                       v
[04 Group Management] ✅  [05 User Sync] ✅
       │                       │
       └───────────┬───────────┘
                   v
         [06 Integration Tests] ✅
```

---

## Архитектура

### Текущая (реализованная) архитектура

```
┌─────────────────────────────────────────────────────────┐
│                    HTTP Requests                         │
│              (Authorization: Bearer <token>)             │
└──────────────────────┬──────────────────────────────────┘
                       │
                       v
        ┌──────────────────────────────┐
        │    Auth Middleware           │ ◄── JWT Validator
        │  Extract & Validate Bearer   │         │
        └──────────────┬───────────────┘         │
                       │                         v
                       │              ┌──────────────────┐
                       │              │ JWKS Cache       │
                       │              │ (keyfunc/jwkset) │
                       │              └──────────────────┘
                       v
        ┌──────────────────────────────┐
        │    AuthHandler / Protected   │
        │    Handlers                  │
        └──────────────┬───────────────┘
                       │
         ┌─────────────┴─────────────────┬───────────────────┐
         │                               │                   │
         v                               v                   v
┌─────────────────────┐    ┌─────────────────────────┐  ┌────────────┐
│   AuthService       │    │ GroupClient             │  │ UserClient │
│   (OAuth flow)      │    │ Keycloak Admin API      │  │ Admin API  │
└─────────────────────┘    └─────────────────────────┘  └────────────┘
         │                               │
         v                               v
┌─────────────────────┐    ┌─────────────────────────┐
│ TokenStore (Redis)  │    │ AdminTokenManager       │
└─────────────────────┘    │ (token caching)         │
                           └─────────────────────────┘
```

---

## Компоненты

### JWT Validator

Офлайн валидация токенов через JWKS:
- Кэширование ключей с автообновлением
- Проверка signature, issuer, audience, expiry
- Извлечение claims (roles, groups, user info)
- Latency ~63μs на валидацию

### Group Client

Управление группами через Keycloak Admin API:
- CreateGroup, DeleteGroup
- AddUserToGroup, RemoveUserFromGroup
- GetGroup, GetUserGroups

### User Client

Работа с пользователями через Admin API:
- ListUsers (с пагинацией)
- GetUser
- CountUsers

### Admin Token Manager

Управление admin токенами:
- Автоматическое получение токена
- Кэширование с auto-refresh
- Поддержка password grant и client credentials

---

## Конфигурация

### `configs/config.yaml`

```yaml
keycloak:
  url: "http://localhost:8090"
  realm: "flowra"
  client_id: "flowra-backend"
  client_secret: "${KEYCLOAK_CLIENT_SECRET}"
  admin_username: "admin"
  admin_password: "${KEYCLOAK_ADMIN_PASSWORD}"
  jwt:
    leeway: "30s"              # Допуск при проверке exp/iat
    refresh_interval: "1h"     # Интервал обновления JWKS

sync:
  users:
    enabled: true
    interval: "15m"            # Интервал синхронизации пользователей
    batch_size: 100
```

---

## Docker Compose

```yaml
keycloak:
  image: quay.io/keycloak/keycloak:23.0
  ports:
    - "8090:8080"
  environment:
    - KEYCLOAK_ADMIN=admin
    - KEYCLOAK_ADMIN_PASSWORD=admin123
    - KC_DB=dev-file
  volumes:
    - ./configs/keycloak:/opt/keycloak/data/import
  command: start-dev --import-realm
```

---

## Тестирование

### Unit Tests

| Файл | Тестов | Описание |
|------|--------|----------|
| `oauth_client_test.go` | 13 | OAuth flow с mock HTTP server |
| `token_store_test.go` | 8 | Redis storage |
| `auth_service_test.go` | 14 | Service layer с mocks |
| `jwt_validator_test.go` | 10+ | JWT validation |
| `admin_token_test.go` | 21 | Admin token management |
| `group_client_test.go` | 18 | Group operations |
| `user_client_test.go` | 12 | User operations |
| `auth_middleware_test.go` | 30+ | Middleware и helpers |

### Integration Tests

`tests/integration/keycloak_integration_test.go`:
- JWT Validator — валидация токенов, извлечение claims
- OAuth Client — refresh, revoke, userinfo
- Admin Token — получение, кэширование, инвалидация
- Group Client — CRUD операции с группами
- User Client — listing, pagination, get user
- Full Auth Flow — полный цикл аутентификации

### Запуск тестов

```bash
# Unit tests
go test ./internal/infrastructure/keycloak/...

# Integration tests (требует Docker)
make test-integration-keycloak
```

---

## Ресурсы

### Документация Keycloak
- [Keycloak Admin REST API](https://www.keycloak.org/docs-api/23.0.0/rest-api/)
- [OIDC Endpoints](https://www.keycloak.org/docs/latest/server_admin/#openid-connect-1)

### Go Libraries
- [MicahParks/keyfunc](https://github.com/MicahParks/keyfunc) — JWKS validation
- [golang-jwt/jwt](https://github.com/golang-jwt/jwt) — JWT parsing

### Внутренние документы
- [Phase 1.3.1 Keycloak Integration](../../roadmap/phase-1/task-1.3.1-keycloak-integration.md)
- [Auth Service](../06-2-Services/05-auth-service.md)

---

## Application Access

- **Main App**: http://localhost:8080
- **Keycloak Admin Console**: http://localhost:8090 (admin/admin123)
- **MongoDB**: localhost:27017 (admin/admin123)
- **Redis**: localhost:6379

### Test Users

| Username | Email | Password | Roles |
|----------|-------|----------|-------|
| `testuser` | testuser@example.com | password123 | user |
| `admin` | admin@example.com | admin123 | user, admin |
| `alice` | alice@example.com | password123 | user, workspace_owner |
| `bob` | bob@example.com | password123 | user |

---

*Обновлено: 2026-01-06*
