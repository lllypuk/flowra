# Task 01: WorkspaceAccessChecker

**Приоритет:** 🔴 Critical
**Статус:** Pending
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

    "github.com/google/uuid"
    "github.com/lllypuk/flowra/internal/application/workspace"
    "github.com/lllypuk/flowra/internal/domain/errs"
    "github.com/lllypuk/flowra/internal/middleware"
)

// RealWorkspaceAccessChecker реализует middleware.WorkspaceAccessChecker
// используя реальный репозиторий workspace.
type RealWorkspaceAccessChecker struct {
    repo workspace.QueryRepository
}

// NewRealWorkspaceAccessChecker создаёт новый access checker.
func NewRealWorkspaceAccessChecker(repo workspace.QueryRepository) *RealWorkspaceAccessChecker {
    return &RealWorkspaceAccessChecker{repo: repo}
}

// GetMembership возвращает информацию о членстве пользователя в workspace.
func (c *RealWorkspaceAccessChecker) GetMembership(
    ctx context.Context,
    workspaceID, userID uuid.UUID,
) (*middleware.WorkspaceMembership, error) {
    member, err := c.repo.GetMember(ctx, workspaceID, userID)
    if err != nil {
        if errors.Is(err, errs.ErrNotFound) {
            return nil, nil // Не член — возвращаем nil без ошибки
        }
        return nil, err
    }

    return &middleware.WorkspaceMembership{
        WorkspaceID: workspaceID,
        UserID:      userID,
        Role:        string(member.Role()),
        JoinedAt:    member.JoinedAt(),
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

```go
// internal/service/workspace_access_checker_test.go

func TestRealWorkspaceAccessChecker_GetMembership(t *testing.T) {
    // Test cases:
    // 1. User is member → returns membership
    // 2. User is not member → returns nil, nil
    // 3. Repository error → returns error
}

func TestRealWorkspaceAccessChecker_WorkspaceExists(t *testing.T) {
    // Test cases:
    // 1. Workspace exists → returns true
    // 2. Workspace not found → returns false, nil
    // 3. Repository error → returns error
}
```

### Integration tests

Использовать существующие тестовые утилиты из `tests/testutil/`.

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

- [ ] Создать файл `internal/service/workspace_access_checker.go`
- [ ] Реализовать `RealWorkspaceAccessChecker`
- [ ] Реализовать `GetMembership()` с обработкой not found
- [ ] Реализовать `WorkspaceExists()`
- [ ] Написать unit tests
- [ ] Написать integration tests
- [ ] Обновить `container.go` (Task 06)

---

## Критерии приёмки

- [ ] `RealWorkspaceAccessChecker` реализует `middleware.WorkspaceAccessChecker`
- [ ] Корректно обрабатывает случай "пользователь не член workspace"
- [ ] Корректно обрабатывает случай "workspace не существует"
- [ ] Unit test coverage > 80%
- [ ] Integration tests проходят

---

## Заметки

- Этот компонент критичен для авторизации всех workspace-scoped запросов
- Должен быть быстрым — вызывается на каждый запрос
- Рассмотреть кеширование membership в Redis (опционально, можно добавить позже)

---

*Создано: 2026-01-06*
