# Task 09: Chat UseCases Testing

**Дата:** 2025-10-22
**Статус:** 🔴 Critical - Not Started
**Зависимости:** Task 02 (Chat UseCases implementation)
**Оценка:** 3-4 часа
**Приоритет:** ⚠️ КРИТИЧЕСКИЙ - Блокирует переход к следующей фазе

## Проблема

Chat UseCases имеют **0% test coverage**, несмотря на то что все 12 Command UseCases реализованы. Это создаёт серьёзный риск:
- Невозможно валидировать корректность бизнес-логики
- Рефакторинг становится опасным
- Регрессии не будут обнаружены
- Нарушается общий стандарт качества проекта (target: >85%)

## Цель

Создать полное тестовое покрытие для всех Chat Command UseCases с достижением coverage >85%.

## Текущее состояние

### Реализованные UseCases (без тестов):
1. ✅ CreateChatUseCase
2. ✅ AddParticipantUseCase
3. ✅ RemoveParticipantUseCase
4. ✅ ConvertToTaskUseCase
5. ✅ ConvertToBugUseCase
6. ✅ ConvertToEpicUseCase
7. ✅ ChangeStatusUseCase
8. ✅ AssignUserUseCase
9. ✅ SetPriorityUseCase
10. ✅ SetDueDateUseCase
11. ✅ RenameChatUseCase
12. ✅ SetSeverityUseCase

## Тестовая стратегия

### Подготовка

Создать базовую тестовую инфраструктуру для Chat UseCases:

```go
// File: internal/application/chat/test_setup.go
package chat_test

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/lllypuk/flowra/internal/application/chat"
    domainChat "github.com/lllypuk/flowra/internal/domain/chat"
    domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
    "github.com/lllypuk/flowra/tests/mocks"
)

// TestContext создает контекст для тестов
func testContext() context.Context {
    return context.Background()
}

// NewTestEventStore создает mock EventStore
func newTestEventStore() *mocks.EventStore {
    return mocks.NewEventStore()
}

// CreateTestChat создает тестовый чат агрегат
func createTestChat(workspaceID, creatorID domainUUID.UUID, chatType domainChat.Type) *domainChat.Chat {
    c, _ := domainChat.NewChat(workspaceID, chatType, true, creatorID)
    return c
}
```

### Тестовые сценарии для каждого UseCase

#### 1. CreateChatUseCase Tests

**File:** `create_chat_test.go`

```go
package chat_test

func TestCreateChatUseCase_Success_Discussion(t *testing.T) {
    // Создание Discussion чата
    // Проверка событий: ChatCreated
    // Проверка сохранения в EventStore
}

func TestCreateChatUseCase_Success_Task(t *testing.T) {
    // Создание Task чата с title
    // Проверка событий: ChatCreated, TypeChanged
    // Проверка установки title
}

func TestCreateChatUseCase_Success_Bug(t *testing.T) {
    // Создание Bug чата
}

func TestCreateChatUseCase_Success_Epic(t *testing.T) {
    // Создание Epic чата
}

func TestCreateChatUseCase_ValidationError_InvalidWorkspaceID(t *testing.T) {
    // WorkspaceID = uuid.Nil
    // Ожидаем ValidationError
}

func TestCreateChatUseCase_ValidationError_InvalidType(t *testing.T) {
    // Type = "invalid"
    // Ожидаем ValidationError
}

func TestCreateChatUseCase_ValidationError_InvalidCreatedBy(t *testing.T) {
    // CreatedBy = uuid.Nil
}

func TestCreateChatUseCase_EventStoreError(t *testing.T) {
    // EventStore возвращает ошибку
    // Проверяем обработку ошибки
}
```

**Количество тестов:** 8
**Примерное время:** 30 минут

#### 2. AddParticipantUseCase Tests

**File:** `add_participant_test.go`

```go
func TestAddParticipantUseCase_Success_AddMember(t *testing.T) {
    // Добавить участника с ролью Member
    // Проверить событие ParticipantAdded
}

func TestAddParticipantUseCase_Success_AddAdmin(t *testing.T) {
    // Добавить участника с ролью Admin
}

func TestAddParticipantUseCase_Error_AlreadyParticipant(t *testing.T) {
    // Попытка добавить существующего участника
    // Ожидаем ошибку
}

func TestAddParticipantUseCase_ValidationError_InvalidChatID(t *testing.T) {}

func TestAddParticipantUseCase_ValidationError_InvalidUserID(t *testing.T) {}

func TestAddParticipantUseCase_EventStoreError_LoadFails(t *testing.T) {}

func TestAddParticipantUseCase_EventStoreError_SaveFails(t *testing.T) {}
```

**Количество тестов:** 7
**Примерное время:** 25 минут

#### 3. RemoveParticipantUseCase Tests

**File:** `remove_participant_test.go`

```go
func TestRemoveParticipantUseCase_Success(t *testing.T) {
    // Удалить участника
    // Проверить событие ParticipantRemoved
}

func TestRemoveParticipantUseCase_Error_CannotRemoveLastAdmin(t *testing.T) {
    // Попытка удалить последнего админа
    // Ожидаем ErrCannotRemoveLastAdmin
}

func TestRemoveParticipantUseCase_Error_NotParticipant(t *testing.T) {
    // Попытка удалить несуществующего участника
}

func TestRemoveParticipantUseCase_ValidationError_InvalidChatID(t *testing.T) {}

func TestRemoveParticipantUseCase_ValidationError_InvalidUserID(t *testing.T) {}
```

**Количество тестов:** 5
**Примерное время:** 20 минут

#### 4. ConvertToTaskUseCase Tests

**File:** `convert_to_task_test.go`

```go
func TestConvertToTaskUseCase_Success_FromDiscussion(t *testing.T) {
    // Конвертировать Discussion → Task
    // Проверить событие TypeChanged
    // Проверить установку title
}

func TestConvertToTaskUseCase_Error_AlreadyTask(t *testing.T) {
    // Попытка конвертировать Task → Task
    // Ожидаем ошибку
}

func TestConvertToTaskUseCase_ValidationError_EmptyTitle(t *testing.T) {}

func TestConvertToTaskUseCase_ValidationError_TitleTooLong(t *testing.T) {}

func TestConvertToTaskUseCase_EventStoreError(t *testing.T) {}
```

**Количество тестов:** 5
**Примерное время:** 20 минут

#### 5. ConvertToBugUseCase Tests

**File:** `convert_to_bug_test.go`

```go
func TestConvertToBugUseCase_Success_FromDiscussion(t *testing.T) {}

func TestConvertToBugUseCase_Error_AlreadyBug(t *testing.T) {}

func TestConvertToBugUseCase_ValidationError_EmptyTitle(t *testing.T) {}

func TestConvertToBugUseCase_EventStoreError(t *testing.T) {}
```

**Количество тестов:** 4
**Примерное время:** 15 минут

#### 6. ConvertToEpicUseCase Tests

**File:** `convert_to_epic_test.go`

```go
func TestConvertToEpicUseCase_Success_FromDiscussion(t *testing.T) {}

func TestConvertToEpicUseCase_Error_AlreadyEpic(t *testing.T) {}

func TestConvertToEpicUseCase_ValidationError_EmptyTitle(t *testing.T) {}
```

**Количество тестов:** 3
**Примерное время:** 15 минут

#### 7. ChangeStatusUseCase Tests

**File:** `change_status_test.go`

```go
func TestChangeStatusUseCase_Success_TaskStatus(t *testing.T) {
    // Изменить статус Task: Open → InProgress → Done
    // Проверить события StatusChanged
}

func TestChangeStatusUseCase_Success_BugStatus(t *testing.T) {
    // Изменить статус Bug: Open → InProgress → Resolved
}

func TestChangeStatusUseCase_Success_EpicStatus(t *testing.T) {
    // Изменить статус Epic
}

func TestChangeStatusUseCase_Error_InvalidStatusForType(t *testing.T) {
    // Попытка установить BugStatus для Task
    // Ожидаем ошибку
}

func TestChangeStatusUseCase_ValidationError_InvalidChatID(t *testing.T) {}

func TestChangeStatusUseCase_ValidationError_EmptyStatus(t *testing.T) {}
```

**Количество тестов:** 6
**Примерное время:** 25 минут

#### 8. AssignUserUseCase Tests

**File:** `assign_user_test.go`

```go
func TestAssignUserUseCase_Success_AssignUser(t *testing.T) {
    // Назначить пользователя
    // Проверить событие UserAssigned
}

func TestAssignUserUseCase_Success_UnassignUser(t *testing.T) {
    // Снять назначение (AssigneeID = nil)
    // Проверить событие UserUnassigned
}

func TestAssignUserUseCase_Error_OnlyForTypedChats(t *testing.T) {
    // Попытка назначить для Discussion
    // Ожидаем ошибку
}

func TestAssignUserUseCase_ValidationError_InvalidChatID(t *testing.T) {}
```

**Количество тестов:** 4
**Примерное время:** 15 минут

#### 9. SetPriorityUseCase Tests

**File:** `set_priority_test.go`

```go
func TestSetPriorityUseCase_Success_Low(t *testing.T) {}

func TestSetPriorityUseCase_Success_Medium(t *testing.T) {}

func TestSetPriorityUseCase_Success_High(t *testing.T) {}

func TestSetPriorityUseCase_Success_Critical(t *testing.T) {}

func TestSetPriorityUseCase_Error_OnlyForTypedChats(t *testing.T) {}

func TestSetPriorityUseCase_ValidationError_InvalidPriority(t *testing.T) {}
```

**Количество тестов:** 6
**Примерное время:** 20 минут

#### 10. SetDueDateUseCase Tests

**File:** `set_due_date_test.go`

```go
func TestSetDueDateUseCase_Success_SetFutureDate(t *testing.T) {
    // Установить дату в будущем
    // Проверить событие DueDateSet
}

func TestSetDueDateUseCase_Success_ClearDueDate(t *testing.T) {
    // Очистить дату (DueDate = nil)
}

func TestSetDueDateUseCase_Error_DateInPast(t *testing.T) {
    // Попытка установить дату в прошлом
    // Ожидаем ValidationError
}

func TestSetDueDateUseCase_Error_OnlyForTypedChats(t *testing.T) {}

func TestSetDueDateUseCase_ValidationError_InvalidChatID(t *testing.T) {}
```

**Количество тестов:** 5
**Примерное время:** 20 минут

#### 11. RenameChatUseCase Tests

**File:** `rename_chat_test.go`

```go
func TestRenameChatUseCase_Success(t *testing.T) {
    // Переименовать чат
    // Проверить событие TitleChanged
}

func TestRenameChatUseCase_ValidationError_EmptyTitle(t *testing.T) {}

func TestRenameChatUseCase_ValidationError_TitleTooLong(t *testing.T) {}

func TestRenameChatUseCase_EventStoreError(t *testing.T) {}
```

**Количество тестов:** 4
**Примерное время:** 15 минут

#### 12. SetSeverityUseCase Tests

**File:** `set_severity_test.go`

```go
func TestSetSeverityUseCase_Success_Minor(t *testing.T) {}

func TestSetSeverityUseCase_Success_Major(t *testing.T) {}

func TestSetSeverityUseCase_Success_Critical(t *testing.T) {}

func TestSetSeverityUseCase_Success_Blocker(t *testing.T) {}

func TestSetSeverityUseCase_Error_OnlyForBugs(t *testing.T) {
    // Попытка установить severity для Task
    // Ожидаем ErrSeverityOnlyForBugs
}

func TestSetSeverityUseCase_ValidationError_InvalidSeverity(t *testing.T) {}
```

**Количество тестов:** 6
**Примерное время:** 20 минут

## Пример полного теста

```go
// File: internal/application/chat/create_chat_test.go
package chat_test

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/lllypuk/flowra/internal/application/chat"
    domainChat "github.com/lllypuk/flowra/internal/domain/chat"
    domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
    "github.com/lllypuk/flowra/tests/mocks"
)

func TestCreateChatUseCase_Success_Discussion(t *testing.T) {
    // Arrange
    eventStore := mocks.NewEventStore()
    useCase := chat.NewCreateChatUseCase(eventStore)

    workspaceID := domainUUID.New()
    creatorID := domainUUID.New()

    cmd := chat.CreateChatCommand{
        WorkspaceID: workspaceID,
        Type:        domainChat.TypeDiscussion,
        IsPublic:    true,
        CreatedBy:   creatorID,
    }

    // Act
    result, err := useCase.Execute(context.Background(), cmd)

    // Assert
    require.NoError(t, err)
    assert.NotNil(t, result.Aggregate)
    assert.Equal(t, domainChat.TypeDiscussion, result.Aggregate.Type())
    assert.Equal(t, workspaceID, result.Aggregate.WorkspaceID())

    // Проверяем события
    require.Len(t, result.Events, 1)
    chatCreatedEvent, ok := result.Events[0].(domainChat.ChatCreatedEvent)
    require.True(t, ok, "expected ChatCreatedEvent")
    assert.Equal(t, workspaceID, chatCreatedEvent.WorkspaceID)
    assert.Equal(t, creatorID, chatCreatedEvent.CreatedBy)

    // Проверяем вызов EventStore
    assert.Equal(t, 1, eventStore.SaveEventsCallCount())
}

func TestCreateChatUseCase_Success_Task(t *testing.T) {
    // Arrange
    eventStore := mocks.NewEventStore()
    useCase := chat.NewCreateChatUseCase(eventStore)

    cmd := chat.CreateChatCommand{
        WorkspaceID: domainUUID.New(),
        Type:        domainChat.TypeTask,
        IsPublic:    true,
        Title:       "Test Task",
        CreatedBy:   domainUUID.New(),
    }

    // Act
    result, err := useCase.Execute(context.Background(), cmd)

    // Assert
    require.NoError(t, err)
    assert.Equal(t, domainChat.TypeTask, result.Aggregate.Type())
    assert.Equal(t, "Test Task", result.Aggregate.Title())

    // Проверяем события: ChatCreated + TypeChanged
    require.Len(t, result.Events, 2)

    _, isChatCreated := result.Events[0].(domainChat.ChatCreatedEvent)
    assert.True(t, isChatCreated)

    typeChangedEvent, isTypeChanged := result.Events[1].(domainChat.TypeChangedEvent)
    assert.True(t, isTypeChanged)
    assert.Equal(t, domainChat.TypeTask, typeChangedEvent.NewType)
}

func TestCreateChatUseCase_ValidationError_InvalidWorkspaceID(t *testing.T) {
    // Arrange
    eventStore := mocks.NewEventStore()
    useCase := chat.NewCreateChatUseCase(eventStore)

    cmd := chat.CreateChatCommand{
        WorkspaceID: domainUUID.Nil(), // Невалидный UUID
        Type:        domainChat.TypeDiscussion,
        CreatedBy:   domainUUID.New(),
    }

    // Act
    result, err := useCase.Execute(context.Background(), cmd)

    // Assert
    require.Error(t, err)
    assert.Contains(t, err.Error(), "validation failed")
    assert.Nil(t, result.Aggregate)

    // EventStore не должен вызываться
    assert.Equal(t, 0, eventStore.SaveEventsCallCount())
}

func TestCreateChatUseCase_EventStoreError(t *testing.T) {
    // Arrange
    eventStore := mocks.NewEventStore()
    eventStore.SetSaveEventsError(errors.New("database error"))

    useCase := chat.NewCreateChatUseCase(eventStore)

    cmd := chat.CreateChatCommand{
        WorkspaceID: domainUUID.New(),
        Type:        domainChat.TypeDiscussion,
        CreatedBy:   domainUUID.New(),
    }

    // Act
    result, err := useCase.Execute(context.Background(), cmd)

    // Assert
    require.Error(t, err)
    assert.Contains(t, err.Error(), "database error")
}
```

## Checklist

### Подготовка (15 минут)
- [ ] Создать `test_setup.go` с утилитами для тестов
- [ ] Проверить работу mocks (EventStore)
- [ ] Создать примеры тестовых данных

### Реализация тестов (3 часа)
- [ ] CreateChatUseCase - 8 тестов (30 мин)
- [ ] AddParticipantUseCase - 7 тестов (25 мин)
- [ ] RemoveParticipantUseCase - 5 тестов (20 мин)
- [ ] ConvertToTaskUseCase - 5 тестов (20 мин)
- [ ] ConvertToBugUseCase - 4 теста (15 мин)
- [ ] ConvertToEpicUseCase - 3 теста (15 мин)
- [ ] ChangeStatusUseCase - 6 тестов (25 мин)
- [ ] AssignUserUseCase - 4 теста (15 мин)
- [ ] SetPriorityUseCase - 6 тестов (20 мин)
- [ ] SetDueDateUseCase - 5 тестов (20 мин)
- [ ] RenameChatUseCase - 4 теста (15 мин)
- [ ] SetSeverityUseCase - 6 тестов (20 мин)

### Проверка (15 минут)
- [ ] Запустить все тесты: `go test ./internal/application/chat/... -v`
- [ ] Проверить coverage: `go test -coverprofile=coverage.out ./internal/application/chat/...`
- [ ] Убедиться coverage >85%
- [ ] Проверить линтер: `golangci-lint run ./internal/application/chat/...`

## Метрики успеха

- ✅ **Минимум 60 unit тестов** создано
- ✅ **Coverage >85%** для chat package
- ✅ **Все тесты проходят** без ошибок
- ✅ **Нет warnings** от линтера
- ✅ **Test execution time <5 секунд**

## Оценка времени

| Этап | Время |
|------|-------|
| Подготовка | 15 минут |
| Реализация тестов | 3 часа |
| Проверка и фиксы | 15 минут |
| **ИТОГО** | **3.5 часа** |

## Следующие шаги

После завершения:
- [ ] Обновить PROGRESS_TRACKER.md (Phase 2 coverage)
- [ ] Перейти к Task 10 (Chat Query UseCases)
- [ ] Запустить полный test suite проекта

## Референсы

- Пример тестов Message UseCases: `internal/application/message/*_test.go`
- Mock EventStore: `tests/mocks/eventstore.go`
- Test utilities: `tests/testutil/`
