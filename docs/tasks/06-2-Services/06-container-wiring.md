# Task 06: Container Wiring

**Приоритет:** 🔴 Critical
**Статус:** ✅ Complete
**Зависит от:** Tasks 01-05 (все сервисы)
**Дата выполнения:** 2026-01-06

---

## Описание

Обновить `cmd/api/container.go` для использования реальных сервисов вместо mock-реализаций. Это финальная задача, которая интегрирует все созданные сервисы.

---

## Реализация

### Созданные файлы

1. **`internal/service/noop_keycloak_client.go`** - NoOp реализация KeycloakClient для случаев, когда Keycloak не настроен
2. **`internal/service/noop_keycloak_client_test.go`** - Тесты для NoOpKeycloakClient
3. **`tests/integration/container_wiring_test.go`** - Интеграционные тесты для container wiring

### Обновлённые файлы

1. **`cmd/api/container.go`** - Полностью обновлён для использования реальных сервисов
2. **`cmd/api/container_test.go`** - Добавлены unit тесты для wiring

---

## Изменения в Container

### Новые поля в struct

```go
// Repositories
ChatQueryRepo    *mongodb.MongoChatReadModelRepository

// Services (for external access if needed)
WorkspaceService *service.WorkspaceService
MemberService    *service.MemberService
ChatService      *service.ChatService
```

### Новые imports

```go
import (
    chatapp "github.com/lllypuk/flowra/internal/application/chat"
    wsapp "github.com/lllypuk/flowra/internal/application/workspace"
    "github.com/lllypuk/flowra/internal/infrastructure/auth"
    "github.com/lllypuk/flowra/internal/infrastructure/keycloak"
    "github.com/lllypuk/flowra/internal/service"
    "github.com/labstack/echo/v4"
    "github.com/lllypuk/flowra/internal/domain/user"
    "github.com/lllypuk/flowra/internal/domain/uuid"
)
```

### Новые helper методы

- `createWorkspaceService()` - создаёт WorkspaceService с use cases
- `createChatService()` - создаёт ChatService с use cases  
- `createAuthService()` - создаёт AuthService (mock если Keycloak не настроен)
- `createUserRepoAdapter()` - создаёт адаптер для UserRepository

### Обновлённый setupHTTPHandlers()

```go
func (c *Container) setupHTTPHandlers() {
    // === 1. Access Checker (Real) ===
    c.AccessChecker = service.NewRealWorkspaceAccessChecker(c.WorkspaceRepo)

    // === 2. Member Service (Real) ===
    c.MemberService = service.NewMemberService(c.WorkspaceRepo, c.WorkspaceRepo)

    // === 3. Workspace Service (Real) ===
    c.WorkspaceService = c.createWorkspaceService()

    // === 4. Workspace Handler with Real Services ===
    c.WorkspaceHandler = httphandler.NewWorkspaceHandler(c.WorkspaceService, c.MemberService)

    // === 5. Chat Service (Real) ===
    c.ChatService = c.createChatService()
    c.ChatHandler = httphandler.NewChatHandler(c.ChatService)

    // === 6. Auth Service ===
    authService := c.createAuthService()
    c.AuthHandler = httphandler.NewAuthHandler(authService, c.createUserRepoAdapter())

    // === 7. WebSocket Handler (unchanged) ===
    // === 8. Token Validator (unchanged) ===
}
```

### Обновлённый validateWiring()

Добавлена проверка на использование mock в production:

```go
// Check for mock access checker in production
if c.Config.IsProduction() {
    if _, isMock := c.AccessChecker.(*middleware.MockWorkspaceAccessChecker); isMock {
        errs = append(errs, errors.New("mock access checker is not allowed in production"))
    }
}
```

---

## Тестирование

### Unit тесты (cmd/api/container_test.go)

- ✅ `TestContainer_ValidateWiring_MockAccessCheckerInProduction`
- ✅ `TestContainer_RealWorkspaceAccessChecker_Type`
- ✅ `TestContainer_Services_NotNil`
- ✅ `TestContainer_NoOpKeycloakClient`
- ✅ `TestContainer_UserRepoAdapter`
- ✅ `TestContainer_WiringMode_Real`
- ✅ `TestContainer_WiringMode_Mock`
- ✅ `TestContainer_WiringMode_Default`

### Unit тесты (internal/service/noop_keycloak_client_test.go)

- ✅ `TestNewNoOpKeycloakClient`
- ✅ `TestNoOpKeycloakClient_CreateGroup`
- ✅ `TestNoOpKeycloakClient_CreateGroup_ReturnsUniqueIDs`
- ✅ `TestNoOpKeycloakClient_DeleteGroup`
- ✅ `TestNoOpKeycloakClient_AddUserToGroup`
- ✅ `TestNoOpKeycloakClient_RemoveUserFromGroup`
- ✅ `TestNoOpKeycloakClient_FullWorkflow`
- ✅ `TestNoOpKeycloakClient_CanceledContext`

### Integration тесты (tests/integration/container_wiring_test.go)

- ✅ `TestContainerWiring_RealAccessChecker`
- ✅ `TestContainerWiring_MemberService`
- ✅ `TestContainerWiring_MemberService_OwnerProtection`
- ✅ `TestContainerWiring_NoOpKeycloakClient`
- ✅ `TestContainerWiring_AccessChecker_WorkspaceExists`
- ✅ `TestContainerWiring_FullMembershipFlow`

---

## Чеклист

### Подготовка
- [x] Создать `internal/service/` директорию (уже существует)
- [x] Убедиться, что все сервисы из Tasks 01-05 реализованы

### Container updates
- [x] Добавить import для `internal/service`
- [x] Создать `createWorkspaceService()` method
- [x] Создать `createChatService()` method
- [x] Создать `createAuthService()` method
- [x] Обновить `setupHTTPHandlers()` для real сервисов
- [x] Обновить `validateWiring()` для проверки real сервисов

### Configuration
- [x] Feature flags уже реализованы через `config.App.Mode` (real/mock)
- [x] `.env.example` уже содержит `APP_MODE` переменную

### Testing
- [x] Написать unit тесты для real wiring
- [x] Написать integration тесты для container wiring
- [x] Проверить, что все существующие тесты проходят

### Cleanup
- [x] Удалить TODO комментарии про mock
- [x] Обновить логирование (убрать Warn про mocks для real сервисов)

---

## Критерии приёмки

- [x] Все mock-сервисы заменены на real в `setupHTTPHandlers()` (кроме Auth при отсутствии Keycloak)
- [x] Feature flags позволяют вернуться к mock для debugging (`APP_MODE=mock`)
- [x] `validateWiring()` предупреждает о mocks в production
- [x] Все существующие тесты проходят
- [ ] HTMX frontend работает с реальными данными (будет проверено в February 2026)

---

## Rollback план

Если что-то пойдёт не так:

1. Установить `APP_MODE=mock` в environment
2. Перезапустить приложение
3. Mock-сервисы будут использоваться (требуется отдельная реализация mock режима)

---

## Заметки

- NoOpKeycloakClient используется когда Keycloak не настроен
- AuthService автоматически переключается на mock если Keycloak URL не задан
- userRepoAdapter обеспечивает совместимость между context.Context и echo.Context
- Все тесты проходят: unit (0.024s) и integration (4.36s + 4.96s)

---

*Создано: 2026-01-06*
*Выполнено: 2026-01-06*