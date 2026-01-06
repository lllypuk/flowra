# Task 01: WorkspaceAccessChecker

**Приоритет:** 🔴 Critical
**Статус:** ✅ Complete
**Зависит от:** MongoDB репозитории (готовы)

---

## Описание

Реализовать `RealWorkspaceAccessChecker`, который заменит `MockWorkspaceAccessChecker` в middleware. Этот компонент отвечает за проверку членства пользователя в workspace и используется middleware для авторизации запросов.

---

## Текущее состояние

### Mock реализация (internal/middleware/workspace.go:324)

```go
type MockWorkspaceAccessChecker struct {
    memberships map[string]map[string]*WorkspaceMembership
    workspaces  map[string]bool
}

func NewMockWorkspaceAccessChecker() *MockWorkspaceAccessChecker
func (m *MockWorkspaceAccessChecker) AddMembership(workspaceID, userID uuid.UUID, role string)
func (m *MockWorkspaceAccessChecker) AddWorkspace(workspaceID uuid.UUID)
func (m *MockWorkspaceAccessChecker) GetMembership(...) (*WorkspaceMembership, error)
func (m *MockWorkspaceAccessChecker) WorkspaceExists(...) (bool, error)
```

### Использование в container.go

```go
// container.go:456-457
c.Logger.Warn("AccessChecker: using mock implementation (real access checker not yet available)")
c.AccessChecker = middleware.NewMockWorkspaceAccessChecker()
```

---

## Интерфейс (internal/middleware/workspace.go:55-63)

```go
type WorkspaceAccessChecker interface {
    // GetMembership возвращает информацию о членстве пользователя в workspace
    GetMembership(ctx context.Context, workspaceID, userID uuid.UUID) (*WorkspaceMembership, error)

    // WorkspaceExists проверяет существование workspace
    WorkspaceExists(ctx context.Context, workspaceID uuid.UUID) (bool, error)
}

type WorkspaceMembership struct {
    WorkspaceID uuid.UUID
    UserID      uuid.UUID
    Role        string
    JoinedAt    time.Time
}
```

---

## Реализация

### Файл: internal/service/workspace_access_checker.go

```go
package service

import (
    "context"
    "errors"

    "github.com/lllypuk/flowra/internal/domain/errs"
    "github.com/lllypuk/flowra/internal/domain/uuid"
    "github.com/lllypuk/flowra/internal/domain/workspace"
    "github.com/lllypuk/flowra/internal/middleware"
)

// WorkspaceQueryRepository определяет интерфейс репозитория, необходимый для access checker.
// Объявлен на стороне потребителя согласно принципам Go interface design.
type WorkspaceQueryRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)
    GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*workspace.Member, error)
}

// RealWorkspaceAccessChecker реализует middleware.WorkspaceAccessChecker
// используя реальный репозиторий workspace.
type RealWorkspaceAccessChecker struct {
    repo WorkspaceQueryRepository
}

// NewRealWorkspaceAccessChecker создаёт новый access checker.
func NewRealWorkspaceAccessChecker(repo WorkspaceQueryRepository) *RealWorkspaceAccessChecker {
    return &RealWorkspaceAccessChecker{repo: repo}
}

// GetMembership возвращает информацию о членстве пользователя в workspace.
// Возвращает (nil, nil) если пользователь не является членом workspace.
// Возвращает middleware.ErrWorkspaceNotFound если workspace не существует.
func (c *RealWorkspaceAccessChecker) GetMembership(
    ctx context.Context,
    workspaceID, userID uuid.UUID,
) (*middleware.WorkspaceMembership, error) {
    // Сначала проверяем, что workspace существует и получаем его данные
    ws, err := c.repo.FindByID(ctx, workspaceID)
    if err != nil {
        if errors.Is(err, errs.ErrNotFound) {
            return nil, middleware.ErrWorkspaceNotFound
        }
        return nil, err
    }

    // Получаем информацию о членстве
    member, err := c.repo.GetMember(ctx, workspaceID, userID)
    if err != nil {
        if errors.Is(err, errs.ErrNotFound) {
            return nil, nil // Пользователь не член workspace
        }
        return nil, err
    }

    return &middleware.WorkspaceMembership{
        WorkspaceID:   workspaceID,
        WorkspaceName: ws.Name(),
        UserID:        userID,
        Role:          member.Role().String(),
    }, nil
}

// WorkspaceExists проверяет существование workspace.
func (c *RealWorkspaceAccessChecker) WorkspaceExists(
    ctx context.Context,
    workspaceID uuid.UUID,
) (bool, error) {
    ws, err := c.repo.FindByID(ctx, workspaceID)
    if err != nil {
        if errors.Is(err, errs.ErrNotFound) {
            return false, nil
        }
        return false, err
    }
    return ws != nil, nil
}
```

---

## Зависимости

### Входящие
- `workspace.QueryRepository` — уже реализован в `MongoWorkspaceRepository`

### Методы репозитория

Из `internal/application/workspace/repository.go`:

```go
type QueryRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)
    GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*workspace.Member, error)
    // ... другие методы
}
```

Реализация: `internal/infrastructure/repository/mongodb/workspace_repository.go`

---

## Тестирование

### Unit tests

Файл: `internal/service/workspace_access_checker_test.go`

- ✅ `TestRealWorkspaceAccessChecker_GetMembership` - 7 тест-кейсов
- ✅ `TestRealWorkspaceAccessChecker_WorkspaceExists` - 3 тест-кейса
- ✅ `TestRealWorkspaceAccessChecker_ImplementsInterface` - compile-time check
- ✅ `TestNewRealWorkspaceAccessChecker` - constructor test

### Integration tests

Файл: `tests/integration/service/workspace_access_checker_test.go`

- ✅ `TestRealWorkspaceAccessChecker_Integration_GetMembership` - 5 тест-кейсов
- ✅ `TestRealWorkspaceAccessChecker_Integration_WorkspaceExists` - 2 тест-кейса
- ✅ `TestRealWorkspaceAccessChecker_Integration_MultipleMembers` - 1 тест-кейс

---

## Обновление container.go

После реализации, в `setupHTTPHandlers()`:

```go
// БЫЛО:
c.Logger.Warn("AccessChecker: using mock implementation")
c.AccessChecker = middleware.NewMockWorkspaceAccessChecker()

// СТАЛО:
c.AccessChecker = service.NewRealWorkspaceAccessChecker(c.WorkspaceRepo)
c.Logger.Debug("workspace access checker initialized")
```

---

## Чеклист

- [x] Создать файл `internal/service/workspace_access_checker.go`
- [x] Реализовать `RealWorkspaceAccessChecker`
- [x] Реализовать `GetMembership()` с обработкой not found
- [x] Реализовать `WorkspaceExists()`
- [x] Написать unit tests
- [x] Написать integration tests
- [ ] Обновить `container.go` (Task 06)

---

## Критерии приёмки

- [x] `RealWorkspaceAccessChecker` реализует `middleware.WorkspaceAccessChecker`
- [x] Корректно обрабатывает случай "пользователь не член workspace"
- [x] Корректно обрабатывает случай "workspace не существует"
- [x] Unit test coverage > 80% (достигнуто: 100%)
- [x] Integration tests проходят

---

## Заметки

- Этот компонент критичен для авторизации всех workspace-scoped запросов
- Должен быть быстрым — вызывается на каждый запрос
- Рассмотреть кеширование membership в Redis (опционально, можно добавить позже)
- Интерфейс `WorkspaceQueryRepository` объявлен локально в service package согласно принципам Go interface design

---

*Создано: 2026-01-06*
*Завершено: 2026-01-06*
