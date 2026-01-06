# 01: Keycloak Realm Setup

**Приоритет:** 🔴 Critical
**Статус:** ⏳ Не начато
**Зависит от:** Docker Compose с Keycloak

---

## Описание

Настроить Keycloak realm для Flowra: создать realm, OAuth2 client, роли, группы и тестовых пользователей. Экспортировать конфигурацию для воспроизводимого development environment.

---

## Задачи

### 1. Создание Realm

```
Realm: flowra
Display Name: Flowra
Enabled: true
Login Settings:
  - User registration: true
  - Email as username: false
  - Remember me: true
  - Login with email: true
```

### 2. OAuth2 Client Configuration

```
Client ID: flowra-backend
Client Type: OpenID Connect
Client Authentication: On (confidential)
Standard Flow: Enabled
Direct Access Grants: Enabled (for testing)

Valid Redirect URIs:
  - http://localhost:8080/auth/callback
  - http://localhost:3000/auth/callback (dev)

Web Origins:
  - http://localhost:8080
  - http://localhost:3000

Logout Settings:
  - Front channel logout: Off
  - Backchannel logout: On
  - Backchannel logout URL: http://localhost:8080/auth/logout/callback
```

### 3. Client Scopes

```
Default Scopes:
  - openid
  - profile
  - email

Optional Scopes:
  - roles (realm roles)
  - groups (user groups)
```

### 4. Realm Roles

| Role | Description |
|------|-------------|
| `user` | Default role for all users |
| `admin` | System administrator |
| `workspace_owner` | Can create/delete workspaces |
| `workspace_admin` | Can manage workspace members |

### 5. Default Groups

| Group | Description |
|-------|-------------|
| `users` | All registered users |
| `admins` | System administrators |

### 6. Test Users

| Username | Email | Password | Roles |
|----------|-------|----------|-------|
| `testuser` | testuser@example.com | password123 | user |
| `admin` | admin@example.com | admin123 | user, admin |
| `alice` | alice@example.com | password123 | user, workspace_owner |
| `bob` | bob@example.com | password123 | user |

---

## Файлы

### Export Configuration

```
configs/keycloak/
├── realm-export.json       # Полный экспорт realm
├── users-export.json       # Тестовые пользователи (dev only)
└── README.md               # Инструкция по импорту
```

### Docker Compose Update

```yaml
# docker-compose.yml
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

## Скрипт автоматизации

### `scripts/setup-keycloak.sh`

```bash
#!/bin/bash
set -e

KEYCLOAK_URL="${KEYCLOAK_URL:-http://localhost:8090}"
REALM="flowra"

echo "Waiting for Keycloak to start..."
until curl -s "$KEYCLOAK_URL/health/ready" > /dev/null 2>&1; do
    sleep 2
done

echo "Getting admin token..."
ADMIN_TOKEN=$(curl -s -X POST "$KEYCLOAK_URL/realms/master/protocol/openid-connect/token" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "username=admin" \
    -d "password=admin123" \
    -d "grant_type=password" \
    -d "client_id=admin-cli" | jq -r '.access_token')

echo "Importing realm..."
curl -s -X POST "$KEYCLOAK_URL/admin/realms" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d @configs/keycloak/realm-export.json

echo "Keycloak setup complete!"
```

---

## JWT Token Claims

После настройки токены будут содержать:

```json
{
  "exp": 1704067200,
  "iat": 1704063600,
  "iss": "http://localhost:8090/realms/flowra",
  "aud": "flowra-backend",
  "sub": "user-uuid",
  "typ": "Bearer",
  "azp": "flowra-backend",
  "session_state": "session-uuid",
  "scope": "openid profile email",
  "email_verified": true,
  "name": "Test User",
  "preferred_username": "testuser",
  "given_name": "Test",
  "family_name": "User",
  "email": "testuser@example.com",
  "realm_access": {
    "roles": ["user", "default-roles-flowra"]
  },
  "groups": ["/users"]
}
```

---

## Чеклист

### Realm Configuration
- [ ] Realm `flowra` создан
- [ ] Login settings настроены
- [ ] Email settings настроены (SMTP для dev)

### OAuth Client
- [ ] Client `flowra-backend` создан
- [ ] Client secret сгенерирован и сохранён
- [ ] Redirect URIs настроены
- [ ] Client scopes настроены

### Roles & Groups
- [ ] Realm roles созданы
- [ ] Default groups созданы
- [ ] Role mappings настроены

### Test Users
- [ ] Тестовые пользователи созданы
- [ ] Пароли установлены
- [ ] Роли назначены

### Export
- [ ] Realm экспортирован в JSON
- [ ] Docker volume настроен
- [ ] Auto-import при старте работает

---

## Критерии приёмки

- [ ] `docker-compose up` автоматически настраивает Keycloak
- [ ] OAuth2 login flow работает с настроенным client
- [ ] Тестовые пользователи могут авторизоваться
- [ ] JWT токены содержат roles и groups
- [ ] Конфигурация воспроизводима (fresh start работает)

---

## Зависимости

### Входящие
- Docker Compose с Keycloak ✅

### Исходящие
- [02-jwt-validation.md](02-jwt-validation.md) — требует настроенный realm
- [03-token-middleware.md](03-token-middleware.md) — требует client configuration

---

*Обновлено: 2026-01-06*
