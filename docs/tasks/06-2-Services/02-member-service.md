# Task 02: MemberService

**Приоритет:** 🔴 Critical
**Статус:** Pending
**Зависит от:** MongoDB репозитории (готовы)

---

## Описание

Реализовать `MemberService`, который управляет участниками workspace. Сервис должен реализовать интерфейс `httphandler.MemberService` и заменить `MockMemberService`.

---

## Текущее состояние

### Mock реализация (internal/handler/http/workspace_handler.go)

```go
type MockMemberService struct {
    members map[string]map[string]*workspace.Member
}

func NewMockMemberService() *MockMemberService
func (m *MockMemberService) AddMember(...) (*workspace.Member, error)
func (m *MockMemberService) RemoveMember(...) error
func (m *MockMemberService) UpdateMemberRole(...) (*workspace.Member, error)
func (m *MockMemberService) GetMember(...) (*workspace.Member, error)
func (m *MockMemberService) ListMembers(...) ([]*workspace.Member, int, error)
func (m *MockMemberService) IsOwner(...) (bool, error)
```

### Использование в container.go

```go
// container.go:428
mockMemberService := httphandler.NewMockMemberService()
c.WorkspaceHandler = httphandler.NewWorkspaceHandler(mockWorkspaceService, mockMemberService)
```

---

## Интерфейс (internal/handler/http/workspace_handler.go)

```go
type MemberService interface {
    AddMember(ctx context.Context, workspaceID, userID uuid.UUID, role workspace.Role) (*workspace.Member, error)
    RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error
    UpdateMemberRole(ctx context.Context, workspaceID, userID uuid.UUID, role workspace.Role) (*workspace.Member, error)
    GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*workspace.Member, error)
    ListMembers(ctx context.Context, workspaceID uuid.UUID, offset, limit int) ([]*workspace.Member, int, error)
    IsOwner(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error)
}
```

---

## Реализация

### Файл: internal/service/member_service.go

```go
package service

import (
    "context"
    "errors"

    "github.com/google/uuid"
    "github.com/lllypuk/flowra/internal/application/workspace"
    wsdomain "github.com/lllypuk/flowra/internal/domain/workspace"
    "github.com/lllypuk/flowra/internal/domain/errs"
)

// MemberService реализует httphandler.MemberService
type MemberService struct {
    commandRepo workspace.CommandRepository
    queryRepo   workspace.QueryRepository
}

// NewMemberService создаёт новый MemberService.
func NewMemberService(
    commandRepo workspace.CommandRepository,
    queryRepo workspace.QueryRepository,
) *MemberService {
    return &MemberService{
        commandRepo: commandRepo,
        queryRepo:   queryRepo,
    }
}

// AddMember добавляет пользователя в workspace.
func (s *MemberService) AddMember(
    ctx context.Context,
    workspaceID, userID uuid.UUID,
    role wsdomain.Role,
) (*wsdomain.Member, error) {
    // Проверить, что workspace существует
    ws, err := s.queryRepo.FindByID(ctx, workspaceID)
    if err != nil {
        return nil, err
    }
    if ws == nil {
        return nil, errs.ErrNotFound
    }

    // Проверить, что пользователь ещё не член
    existing, err := s.queryRepo.GetMember(ctx, workspaceID, userID)
    if err != nil && !errors.Is(err, errs.ErrNotFound) {
        return nil, err
    }
    if existing != nil {
        return nil, errs.ErrAlreadyExists
    }

    // Создать member
    member := wsdomain.NewMember(workspaceID, userID, role)

    if err := s.commandRepo.AddMember(ctx, member); err != nil {
        return nil, err
    }

    return member, nil
}

// RemoveMember удаляет пользователя из workspace.
func (s *MemberService) RemoveMember(
    ctx context.Context,
    workspaceID, userID uuid.UUID,
) error {
    // Проверить, что member существует
    member, err := s.queryRepo.GetMember(ctx, workspaceID, userID)
    if err != nil {
        return err
    }
    if member == nil {
        return errs.ErrNotFound
    }

    // Нельзя удалить owner
    if member.Role() == wsdomain.RoleOwner {
        return errs.ErrForbidden
    }

    return s.commandRepo.RemoveMember(ctx, workspaceID, userID)
}

// UpdateMemberRole обновляет роль участника.
func (s *MemberService) UpdateMemberRole(
    ctx context.Context,
    workspaceID, userID uuid.UUID,
    role wsdomain.Role,
) (*wsdomain.Member, error) {
    member, err := s.queryRepo.GetMember(ctx, workspaceID, userID)
    if err != nil {
        return nil, err
    }
    if member == nil {
        return nil, errs.ErrNotFound
    }

    // Нельзя изменить роль owner
    if member.Role() == wsdomain.RoleOwner {
        return nil, errs.ErrForbidden
    }

    // Нельзя назначить owner через этот метод
    if role == wsdomain.RoleOwner {
        return nil, errs.ErrForbidden
    }

    // Обновить роль
    member.SetRole(role)

    if err := s.commandRepo.UpdateMember(ctx, member); err != nil {
        return nil, err
    }

    return member, nil
}

// GetMember возвращает информацию об участнике.
func (s *MemberService) GetMember(
    ctx context.Context,
    workspaceID, userID uuid.UUID,
) (*wsdomain.Member, error) {
    return s.queryRepo.GetMember(ctx, workspaceID, userID)
}

// ListMembers возвращает список участников workspace.
func (s *MemberService) ListMembers(
    ctx context.Context,
    workspaceID uuid.UUID,
    offset, limit int,
) ([]*wsdomain.Member, int, error) {
    members, err := s.queryRepo.ListMembers(ctx, workspaceID, offset, limit)
    if err != nil {
        return nil, 0, err
    }

    total, err := s.queryRepo.CountMembers(ctx, workspaceID)
    if err != nil {
        return nil, 0, err
    }

    return members, total, nil
}

// IsOwner проверяет, является ли пользователь владельцем workspace.
func (s *MemberService) IsOwner(
    ctx context.Context,
    workspaceID, userID uuid.UUID,
) (bool, error) {
    member, err := s.queryRepo.GetMember(ctx, workspaceID, userID)
    if err != nil {
        if errors.Is(err, errs.ErrNotFound) {
            return false, nil
        }
        return false, err
    }
    if member == nil {
        return false, nil
    }

    return member.Role() == wsdomain.RoleOwner, nil
}
```

---

## Зависимости

### Входящие

Из `internal/application/workspace/repository.go`:

```go
type CommandRepository interface {
    AddMember(ctx context.Context, member *workspace.Member) error
    RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error
    UpdateMember(ctx context.Context, member *workspace.Member) error // Может потребоваться добавить
}

type QueryRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)
    GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (*workspace.Member, error)
    ListMembers(ctx context.Context, workspaceID uuid.UUID, offset, limit int) ([]*workspace.Member, error)
    CountMembers(ctx context.Context, workspaceID uuid.UUID) (int, error)
}
```

### Возможно требуется добавить

Проверить, есть ли метод `UpdateMember` в `CommandRepository`. Если нет — добавить.

---

## Бизнес-правила

1. **Owner protection:** Нельзя удалить или изменить роль owner
2. **No self-promotion to owner:** Нельзя назначить себя owner через UpdateMemberRole
3. **Duplicate check:** Нельзя добавить пользователя, который уже член workspace
4. **Workspace existence:** Все операции требуют существующий workspace

---

## Тестирование

### Unit tests

```go
// internal/service/member_service_test.go

func TestMemberService_AddMember(t *testing.T) {
    // Test cases:
    // 1. Successfully add member
    // 2. Workspace not found → error
    // 3. User already member → ErrAlreadyExists
}

func TestMemberService_RemoveMember(t *testing.T) {
    // Test cases:
    // 1. Successfully remove member
    // 2. Member not found → ErrNotFound
    // 3. Try to remove owner → ErrForbidden
}

func TestMemberService_UpdateMemberRole(t *testing.T) {
    // Test cases:
    // 1. Successfully update role
    // 2. Try to update owner role → ErrForbidden
    // 3. Try to set role to owner → ErrForbidden
}

func TestMemberService_IsOwner(t *testing.T) {
    // Test cases:
    // 1. User is owner → true
    // 2. User is member but not owner → false
    // 3. User is not member → false
}
```

---

## Чеклист

- [ ] Создать файл `internal/service/member_service.go`
- [ ] Реализовать `NewMemberService()`
- [ ] Реализовать `AddMember()` с проверками
- [ ] Реализовать `RemoveMember()` с owner protection
- [ ] Реализовать `UpdateMemberRole()` с ограничениями
- [ ] Реализовать `GetMember()`
- [ ] Реализовать `ListMembers()` с пагинацией
- [ ] Реализовать `IsOwner()`
- [ ] Проверить/добавить `UpdateMember` в CommandRepository
- [ ] Написать unit tests
- [ ] Обновить `container.go` (Task 06)

---

## Критерии приёмки

- [ ] `MemberService` реализует `httphandler.MemberService`
- [ ] Owner protection работает корректно
- [ ] Duplicate member detection работает
- [ ] Unit test coverage > 80%
- [ ] Все handler тесты проходят с real сервисом

---

## Заметки

- Keycloak integration для member management можно добавить позже
- Рассмотреть добавление событий при add/remove member (для notification system)
- Пагинация в `ListMembers` использует offset/limit

---

*Создано: 2026-01-06*
