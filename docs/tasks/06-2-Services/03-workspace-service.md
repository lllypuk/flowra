# Task 03: WorkspaceService

**Приоритет:** 🔴 Critical
**Статус:** Pending
**Зависит от:** Task 02 (MemberService)

---

## Описание

Реализовать `WorkspaceService` — фасад над существующими workspace юзкейсами. Сервис должен реализовать интерфейс `httphandler.WorkspaceService` и заменить `MockWorkspaceService`.

---

## Текущее состояние

### Mock реализация (internal/handler/http/workspace_handler.go)

```go
type MockWorkspaceService struct {
    workspaces map[string]*workspace.Workspace
    counter    int
}

func NewMockWorkspaceService() *MockWorkspaceService
func (m *MockWorkspaceService) CreateWorkspace(...) (*workspace.Workspace, error)
func (m *MockWorkspaceService) GetWorkspace(...) (*workspace.Workspace, error)
func (m *MockWorkspaceService) ListUserWorkspaces(...) ([]*workspace.Workspace, int, error)
func (m *MockWorkspaceService) UpdateWorkspace(...) (*workspace.Workspace, error)
func (m *MockWorkspaceService) DeleteWorkspace(...) error
func (m *MockWorkspaceService) GetMemberCount(...) (int, error)
```

### Использование в container.go

```go
// container.go:427
mockWorkspaceService := httphandler.NewMockWorkspaceService()
c.WorkspaceHandler = httphandler.NewWorkspaceHandler(mockWorkspaceService, mockMemberService)
```

---

## Интерфейс (internal/handler/http/workspace_handler.go)

```go
type WorkspaceService interface {
    CreateWorkspace(ctx context.Context, ownerID uuid.UUID, name, description string) (*workspace.Workspace, error)
    GetWorkspace(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)
    ListUserWorkspaces(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*workspace.Workspace, int, error)
    UpdateWorkspace(ctx context.Context, id uuid.UUID, name, description string) (*workspace.Workspace, error)
    DeleteWorkspace(ctx context.Context, id uuid.UUID) error
    GetMemberCount(ctx context.Context, workspaceID uuid.UUID) (int, error)
}
```

---

## Существующие юзкейсы (internal/application/workspace/)

| Юзкейс | Файл | Статус |
|--------|------|--------|
| `CreateWorkspaceUseCase` | `create_workspace.go` | ✅ Готов |
| `GetWorkspaceUseCase` | `get_workspace.go` | ✅ Готов |
| `ListUserWorkspacesUseCase` | `list_workspaces.go` | ✅ Готов |
| `UpdateWorkspaceUseCase` | `update_workspace.go` | ✅ Готов |
| `CreateInviteUseCase` | `create_invite.go` | ✅ Готов |
| `AcceptInviteUseCase` | `accept_invite.go` | ✅ Готов |
| `RevokeInviteUseCase` | `revoke_invite.go` | ✅ Готов |

---

## Реализация

### Файл: internal/service/workspace_service.go

```go
package service

import (
    "context"

    "github.com/google/uuid"
    wsapp "github.com/lllypuk/flowra/internal/application/workspace"
    wsdomain "github.com/lllypuk/flowra/internal/domain/workspace"
)

// WorkspaceService реализует httphandler.WorkspaceService
type WorkspaceService struct {
    // Use cases
    createUC *wsapp.CreateWorkspaceUseCase
    getUC    *wsapp.GetWorkspaceUseCase
    listUC   *wsapp.ListUserWorkspacesUseCase
    updateUC *wsapp.UpdateWorkspaceUseCase

    // Repositories (для операций без use case)
    commandRepo wsapp.CommandRepository
    queryRepo   wsapp.QueryRepository
}

// WorkspaceServiceConfig содержит зависимости для WorkspaceService.
type WorkspaceServiceConfig struct {
    CreateUC    *wsapp.CreateWorkspaceUseCase
    GetUC       *wsapp.GetWorkspaceUseCase
    ListUC      *wsapp.ListUserWorkspacesUseCase
    UpdateUC    *wsapp.UpdateWorkspaceUseCase
    CommandRepo wsapp.CommandRepository
    QueryRepo   wsapp.QueryRepository
}

// NewWorkspaceService создаёт новый WorkspaceService.
func NewWorkspaceService(cfg WorkspaceServiceConfig) *WorkspaceService {
    return &WorkspaceService{
        createUC:    cfg.CreateUC,
        getUC:       cfg.GetUC,
        listUC:      cfg.ListUC,
        updateUC:    cfg.UpdateUC,
        commandRepo: cfg.CommandRepo,
        queryRepo:   cfg.QueryRepo,
    }
}

// CreateWorkspace создаёт новый workspace.
func (s *WorkspaceService) CreateWorkspace(
    ctx context.Context,
    ownerID uuid.UUID,
    name, description string,
) (*wsdomain.Workspace, error) {
    result, err := s.createUC.Execute(ctx, wsapp.CreateWorkspaceCommand{
        Name:        name,
        Description: description,
        CreatedBy:   ownerID,
    })
    if err != nil {
        return nil, err
    }

    // Получить созданный workspace
    return s.queryRepo.FindByID(ctx, result.WorkspaceID)
}

// GetWorkspace возвращает workspace по ID.
func (s *WorkspaceService) GetWorkspace(
    ctx context.Context,
    id uuid.UUID,
) (*wsdomain.Workspace, error) {
    result, err := s.getUC.Execute(ctx, wsapp.GetWorkspaceQuery{
        WorkspaceID: id,
    })
    if err != nil {
        return nil, err
    }

    return result.Workspace, nil
}

// ListUserWorkspaces возвращает список workspaces пользователя.
func (s *WorkspaceService) ListUserWorkspaces(
    ctx context.Context,
    userID uuid.UUID,
    offset, limit int,
) ([]*wsdomain.Workspace, int, error) {
    result, err := s.listUC.Execute(ctx, wsapp.ListUserWorkspacesQuery{
        UserID: userID,
        Offset: offset,
        Limit:  limit,
    })
    if err != nil {
        return nil, 0, err
    }

    return result.Workspaces, result.Total, nil
}

// UpdateWorkspace обновляет workspace.
func (s *WorkspaceService) UpdateWorkspace(
    ctx context.Context,
    id uuid.UUID,
    name, description string,
) (*wsdomain.Workspace, error) {
    _, err := s.updateUC.Execute(ctx, wsapp.UpdateWorkspaceCommand{
        WorkspaceID: id,
        Name:        name,
        Description: description,
    })
    if err != nil {
        return nil, err
    }

    // Получить обновлённый workspace
    return s.queryRepo.FindByID(ctx, id)
}

// DeleteWorkspace удаляет workspace.
func (s *WorkspaceService) DeleteWorkspace(
    ctx context.Context,
    id uuid.UUID,
) error {
    // Прямое удаление через repository
    // Use case для delete пока не реализован
    return s.commandRepo.Delete(ctx, id)
}

// GetMemberCount возвращает количество участников workspace.
func (s *WorkspaceService) GetMemberCount(
    ctx context.Context,
    workspaceID uuid.UUID,
) (int, error) {
    return s.queryRepo.CountMembers(ctx, workspaceID)
}
```

---

## Инициализация юзкейсов

Юзкейсы должны быть созданы в container и переданы в сервис:

```go
// В container.go или отдельном методе setupUseCases

// Workspace use cases
createWorkspaceUC := wsapp.NewCreateWorkspaceUseCase(
    c.WorkspaceRepo,      // CommandRepository
    c.WorkspaceRepo,      // QueryRepository
    c.KeycloakClient,     // Опционально
)

getWorkspaceUC := wsapp.NewGetWorkspaceUseCase(c.WorkspaceRepo)
listWorkspacesUC := wsapp.NewListUserWorkspacesUseCase(c.WorkspaceRepo)
updateWorkspaceUC := wsapp.NewUpdateWorkspaceUseCase(c.WorkspaceRepo, c.WorkspaceRepo)

// Создание сервиса
workspaceService := service.NewWorkspaceService(service.WorkspaceServiceConfig{
    CreateUC:    createWorkspaceUC,
    GetUC:       getWorkspaceUC,
    ListUC:      listWorkspacesUC,
    UpdateUC:    updateWorkspaceUC,
    CommandRepo: c.WorkspaceRepo,
    QueryRepo:   c.WorkspaceRepo,
})
```

---

## Дополнительные методы (опционально)

Можно добавить методы для invite management:

```go
// CreateInvite создаёт приглашение в workspace.
func (s *WorkspaceService) CreateInvite(ctx context.Context, cmd wsapp.CreateInviteCommand) (*wsapp.CreateInviteResult, error)

// AcceptInvite принимает приглашение.
func (s *WorkspaceService) AcceptInvite(ctx context.Context, cmd wsapp.AcceptInviteCommand) error

// RevokeInvite отзывает приглашение.
func (s *WorkspaceService) RevokeInvite(ctx context.Context, cmd wsapp.RevokeInviteCommand) error
```

---

## Зависимости

### Входящие
- Workspace use cases из `internal/application/workspace/`
- `workspace.CommandRepository` и `workspace.QueryRepository`
- Опционально: `KeycloakClient` для создания групп

### Использует репозиторий

```go
type CommandRepository interface {
    Save(ctx context.Context, ws *workspace.Workspace) error
    Delete(ctx context.Context, id uuid.UUID) error
    AddMember(ctx context.Context, member *workspace.Member) error
    RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error
}

type QueryRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)
    ListWorkspacesByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*workspace.Workspace, error)
    CountWorkspacesByUser(ctx context.Context, userID uuid.UUID) (int, error)
    CountMembers(ctx context.Context, workspaceID uuid.UUID) (int, error)
}
```

---

## Тестирование

### Unit tests

```go
// internal/service/workspace_service_test.go

func TestWorkspaceService_CreateWorkspace(t *testing.T) {
    // Test cases:
    // 1. Successfully create workspace
    // 2. Validation error (empty name) → error from use case
    // 3. Repository error → propagated
}

func TestWorkspaceService_GetWorkspace(t *testing.T) {
    // 1. Workspace exists → returns workspace
    // 2. Workspace not found → ErrNotFound
}

func TestWorkspaceService_ListUserWorkspaces(t *testing.T) {
    // 1. User has workspaces → returns list with total
    // 2. User has no workspaces → returns empty list, 0
    // 3. Pagination works correctly
}

func TestWorkspaceService_UpdateWorkspace(t *testing.T) {
    // 1. Successfully update
    // 2. Workspace not found → ErrNotFound
    // 3. Validation error → error from use case
}

func TestWorkspaceService_DeleteWorkspace(t *testing.T) {
    // 1. Successfully delete
    // 2. Workspace not found → ErrNotFound
}
```

---

## Чеклист

- [ ] Создать файл `internal/service/workspace_service.go`
- [ ] Определить `WorkspaceServiceConfig` struct
- [ ] Реализовать `NewWorkspaceService()`
- [ ] Реализовать `CreateWorkspace()` через use case
- [ ] Реализовать `GetWorkspace()` через use case
- [ ] Реализовать `ListUserWorkspaces()` через use case
- [ ] Реализовать `UpdateWorkspace()` через use case
- [ ] Реализовать `DeleteWorkspace()` через repository
- [ ] Реализовать `GetMemberCount()` через repository
- [ ] Написать unit tests
- [ ] Обновить `container.go` для создания use cases (Task 06)

---

## Критерии приёмки

- [ ] `WorkspaceService` реализует `httphandler.WorkspaceService`
- [ ] Все методы делегируют работу юзкейсам или репозиториям
- [ ] Ошибки корректно пробрасываются
- [ ] Unit test coverage > 80%
- [ ] Handler тесты проходят с real сервисом

---

## Заметки

- Keycloak integration для создания группы workspace — опционально на первом этапе
- Delete workspace должен также удалять members (обрабатывается в repository с транзакцией)
- Рассмотреть добавление soft delete вместо hard delete

---

*Создано: 2026-01-06*
