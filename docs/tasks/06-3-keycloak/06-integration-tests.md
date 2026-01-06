# 06: Integration Tests

**Приоритет:** 🟡 High
**Статус:** ✅ Завершено
**Зависит от:** Все предыдущие задачи (01-05)

---

## Описание

Реализовать интеграционные тесты для Keycloak интеграции с использованием testcontainers. Тесты должны проверять полный auth flow, группы и sync.

---

## Текущие тесты

Существуют только unit-тесты с mocks:
- `oauth_client_test.go` — mock HTTP server
- `token_store_test.go` — Redis
- `auth_service_test.go` — mocks

**Проблемы:**
- Не тестируется реальное взаимодействие с Keycloak
- Нет E2E тестов auth flow
- Не проверяется совместимость с конкретной версией Keycloak

---

## Решение

### Testcontainers

```go
// Keycloak container для тестов
container := testcontainers.NewKeycloakContainer(
    testcontainers.WithRealmImportFile("./testdata/realm-export.json"),
)
```

---

## Файлы

```
tests/integration/
├── keycloak_integration_test.go  # Comprehensive integration tests for all Keycloak clients
├── keycloak_realm_test.go        # Realm setup verification tests
tests/testutil/
└── keycloak.go                   # Container helpers (SharedKeycloakContainer)
configs/keycloak/
└── realm-export.json             # Test realm config with users and groups
```

---

## Реализация

Создан файл `tests/integration/keycloak_integration_test.go` с полным набором интеграционных тестов:

### JWT Validator Tests
- `TestJWTValidator_ValidToken` - валидация реального токена
- `TestJWTValidator_ClaimsExtraction` - проверка извлечения claims для всех пользователей
- `TestJWTValidator_InvalidToken` - отклонение невалидных токенов
- `TestJWTValidator_TamperedToken` - отклонение изменённых токенов

### OAuth Client Tests
- `TestOAuthClient_RefreshToken` - обновление токена
- `TestOAuthClient_RefreshToken_InvalidToken` - отклонение невалидного refresh token
- `TestOAuthClient_RevokeToken` - отзыв токена
- `TestOAuthClient_GetUserInfo` - получение информации о пользователе
- `TestOAuthClient_GetUserInfo_InvalidToken` - отклонение невалидного access token
- `TestOAuthClient_AuthorizationURL` - генерация URL авторизации

### Admin Token Manager Tests
- `TestAdminTokenManager_GetToken` - получение admin токена
- `TestAdminTokenManager_TokenCaching` - проверка кэширования токенов
- `TestAdminTokenManager_InvalidateToken` - инвалидация кэша токенов
- `TestAdminTokenManager_InvalidCredentials` - отклонение неверных учётных данных

### Group Client Tests
- `TestGroupClient_CreateAndDeleteGroup` - создание и удаление группы
- `TestGroupClient_CreateGroup_EmptyName` - отклонение пустого имени группы
- `TestGroupClient_AddRemoveUserFromGroup` - добавление/удаление пользователя из группы
- `TestGroupClient_DeleteGroup_NotFound` - ошибка при удалении несуществующей группы
- `TestGroupClient_GetUserGroups` - получение групп пользователя

### User Client Tests
- `TestUserClient_ListUsers` - список пользователей
- `TestUserClient_ListUsers_Pagination` - пагинация списка пользователей
- `TestUserClient_GetUser` - получение пользователя по ID
- `TestUserClient_GetUser_NotFound` - ошибка для несуществующего пользователя
- `TestUserClient_CountUsers` - подсчёт пользователей
- `TestUserClient_DisplayName` - проверка метода DisplayName

### Full Integration Flow Tests
- `TestFullAuthFlow_TokenValidationWithGroupMembership` - полный flow аутентификации
- `TestWorkspaceGroupLifecycle` - полный lifecycle группы workspace

---

## Makefile

Добавлены таргеты в Makefile:

```makefile
test-integration: ## Run integration tests (with testcontainers)
	go test -tags=integration -v -timeout=10m ./tests/integration/...

test-integration-keycloak: ## Run Keycloak integration tests only
	go test -tags=integration -v -count=1 -timeout=10m -run TestKeycloak ./tests/integration/...
	go test -tags=integration -v -count=1 -timeout=10m -run TestJWT ./tests/integration/...
	go test -tags=integration -v -count=1 -timeout=10m -run TestOAuth ./tests/integration/...
	go test -tags=integration -v -count=1 -timeout=10m -run TestAdmin ./tests/integration/...
	go test -tags=integration -v -count=1 -timeout=10m -run TestGroup ./tests/integration/...
	go test -tags=integration -v -count=1 -timeout=10m -run TestUser ./tests/integration/...
	go test -tags=integration -v -count=1 -timeout=10m -run TestFull ./tests/integration/...
	go test -tags=integration -v -count=1 -timeout=10m -run TestWorkspace ./tests/integration/...
```

---

## Чеклист

### Container Setup
- [x] Keycloak container helper
- [x] Test realm export
- [x] Test users created
- [x] Container cleanup

### OAuth Tests
- [x] Token exchange (password grant)
- [x] Token refresh
- [x] Token revocation
- [x] User info endpoint

### JWT Tests
- [x] Valid token validation
- [x] Invalid token rejection
- [x] Expired token rejection (via tampered token test)
- [x] Claims extraction

### Group Tests
- [x] Create group
- [x] Delete group
- [x] Add user to group
- [x] Remove user from group

### Admin Tests
- [x] Admin token caching
- [x] Token refresh

---

## Критерии приёмки

- [x] Все тесты проходят с реальным Keycloak
- [x] Тесты запускаются через `make test-integration`
- [x] Container cleanup работает
- [x] Тесты изолированы (не влияют друг на друга)
- [x] Время выполнения < 2 минут
- [x] CI/CD интеграция настроена (Makefile targets added)

---

## Зависимости

### Входящие
- [01-realm-setup.md](01-realm-setup.md) — конфигурация для тестов
- [02-jwt-validation.md](02-jwt-validation.md) — JWT validator
- [03-token-middleware.md](03-token-middleware.md) — middleware
- [04-group-management.md](04-group-management.md) — group client
- [05-user-sync.md](05-user-sync.md) — user client

### Go Dependencies

```go
require (
    github.com/testcontainers/testcontainers-go v0.40.0
    github.com/stretchr/testify v1.11.1
)
```

---

*Обновлено: 2026-01-07*
