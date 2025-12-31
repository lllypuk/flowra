# Task 08: Tag Integration Refactoring

**Дата:** 2025-10-19
**Статус:** ✅ Complete
**Дата завершения:** 2025-10-22
**Зависимости:** Task 02 (Chat UseCases), Task 03 (Message UseCases)
**Оценка:** 2-3 часа

## Цель

Рефакторинг `internal/domain/tag/executor.go` для использования Chat UseCases вместо прямой работы с Chat aggregate. Это обеспечит единообразие обработки команд и использование всей бизнес-логики UseCases.

## Контекст

**Текущая реализация:**
```go
// internal/domain/tag/executor.go
func (e *CommandExecutor) executeCreateTask(ctx context.Context, cmd CreateTaskCommand, actorID uuid.UUID) error {
    chatID := domainUUID.FromGoogleUUID(cmd.ChatID)
    userID := domainUUID.FromGoogleUUID(actorID)

    // Прямая работа с repository
    c, err := e.chatRepo.Load(ctx, chatID)
    if err != nil {
        return fmt.Errorf("failed to load chat: %w", err)
    }

    // Прямой вызов метода aggregate
    if err = c.ConvertToTask(cmd.Title, userID); err != nil {
        return fmt.Errorf("failed to convert to task: %w", err)
    }

    // Ручное сохранение и публикация событий
    return e.publishAndSave(ctx, c)
}
```

**Проблемы:**
- Дублирование логики сохранения/публикации
- Нет валидации из UseCases
- Нет авторизации из UseCases
- Сложность поддержки (логика в двух местах)

## Целевая архитектура

```
Message (с тегами)
    ↓
Tag Parser → TagProcessor
    ↓
Tag CommandExecutor
    ↓
Chat/Message UseCases ← Единственная точка входа
    ↓
Domain Aggregates
```

## Новая реализация

### 1. Рефакторинг CommandExecutor

```go
// internal/domain/tag/executor.go
package tag

import (
    "context"
    "fmt"
    "strings"

    "github.com/google/uuid"
    chatApp "github.com/lllypuk/flowra/internal/application/chat"
    "github.com/lllypuk/flowra/internal/domain/user"
)

// CommandExecutor выполняет tag команды через UseCases
type CommandExecutor struct {
    chatUseCases *chatApp.ChatUseCases  // Группа всех Chat UseCases
    userRepo     user.Repository        // Для резолвинга @mentions
}

// ChatUseCases группирует все Chat UseCases
type ChatUseCases struct {
    ConvertToTask  *chatApp.ConvertToTaskUseCase
    ConvertToBug   *chatApp.ConvertToBugUseCase
    ConvertToEpic  *chatApp.ConvertToEpicUseCase
    ChangeStatus   *chatApp.ChangeStatusUseCase
    AssignUser     *chatApp.AssignUserUseCase
    SetPriority    *chatApp.SetPriorityUseCase
    SetDueDate     *chatApp.SetDueDateUseCase
    Rename         *chatApp.RenameChatUseCase
    SetSeverity    *chatApp.SetSeverityUseCase
}

// NewCommandExecutor создает новый CommandExecutor
func NewCommandExecutor(
    chatUseCases *ChatUseCases,
    userRepo user.Repository,
) *CommandExecutor {
    return &CommandExecutor{
        chatUseCases: chatUseCases,
        userRepo:     userRepo,
    }
}

// Execute выполняет команду
func (e *CommandExecutor) Execute(ctx context.Context, cmd Command, actorID uuid.UUID) error {
    switch c := cmd.(type) {
    case CreateTaskCommand:
        return e.executeCreateTask(ctx, c, actorID)
    case CreateBugCommand:
        return e.executeCreateBug(ctx, c, actorID)
    case CreateEpicCommand:
        return e.executeCreateEpic(ctx, c, actorID)
    case ChangeStatusCommand:
        return e.executeChangeStatus(ctx, c, actorID)
    case AssignUserCommand:
        return e.executeAssignUser(ctx, c, actorID)
    case ChangePriorityCommand:
        return e.executeChangePriority(ctx, c, actorID)
    case SetDueDateCommand:
        return e.executeSetDueDate(ctx, c, actorID)
    case ChangeTitleCommand:
        return e.executeChangeTitle(ctx, c, actorID)
    case SetSeverityCommand:
        return e.executeSetSeverity(ctx, c, actorID)
    default:
        return fmt.Errorf("unknown command type: %T", cmd)
    }
}

// executeCreateTask выполняет команду создания Task через UseCase
func (e *CommandExecutor) executeCreateTask(
    ctx context.Context,
    cmd CreateTaskCommand,
    actorID uuid.UUID,
) error {
    usecaseCmd := chatApp.ConvertToTaskCommand{
        ChatID:      cmd.ChatID,
        Title:       cmd.Title,
        ConvertedBy: actorID,
    }

    _, err := e.chatUseCases.ConvertToTask.Execute(ctx, usecaseCmd)
    if err != nil {
        return fmt.Errorf("failed to convert to task: %w", err)
    }

    return nil
}

// executeCreateBug выполняет команду создания Bug через UseCase
func (e *CommandExecutor) executeCreateBug(
    ctx context.Context,
    cmd CreateBugCommand,
    actorID uuid.UUID,
) error {
    usecaseCmd := chatApp.ConvertToBugCommand{
        ChatID:      cmd.ChatID,
        Title:       cmd.Title,
        ConvertedBy: actorID,
    }

    _, err := e.chatUseCases.ConvertToBug.Execute(ctx, usecaseCmd)
    if err != nil {
        return fmt.Errorf("failed to convert to bug: %w", err)
    }

    return nil
}

// executeCreateEpic выполняет команду создания Epic через UseCase
func (e *CommandExecutor) executeCreateEpic(
    ctx context.Context,
    cmd CreateEpicCommand,
    actorID uuid.UUID,
) error {
    usecaseCmd := chatApp.ConvertToEpicCommand{
        ChatID:      cmd.ChatID,
        Title:       cmd.Title,
        ConvertedBy: actorID,
    }

    _, err := e.chatUseCases.ConvertToEpic.Execute(ctx, usecaseCmd)
    if err != nil {
        return fmt.Errorf("failed to convert to epic: %w", err)
    }

    return nil
}

// executeChangeStatus выполняет команду изменения статуса через UseCase
func (e *CommandExecutor) executeChangeStatus(
    ctx context.Context,
    cmd ChangeStatusCommand,
    actorID uuid.UUID,
) error {
    usecaseCmd := chatApp.ChangeStatusCommand{
        ChatID:    cmd.ChatID,
        Status:    cmd.Status,
        ChangedBy: actorID,
    }

    _, err := e.chatUseCases.ChangeStatus.Execute(ctx, usecaseCmd)
    if err != nil {
        return fmt.Errorf("failed to change status: %w", err)
    }

    return nil
}

// executeAssignUser выполняет команду назначения пользователя через UseCase
func (e *CommandExecutor) executeAssignUser(
    ctx context.Context,
    cmd AssignUserCommand,
    actorID uuid.UUID,
) error {
    // Резолвинг username → UUID
    var assigneeID *uuid.UUID
    if cmd.Username != "" && cmd.Username != "@none" {
        username := strings.TrimPrefix(cmd.Username, "@")
        u, err := e.userRepo.FindByUsername(ctx, username)
        if err != nil {
            return fmt.Errorf("user %s not found: %w", cmd.Username, err)
        }
        uid := u.ID().ToGoogleUUID()
        assigneeID = &uid
    }

    usecaseCmd := chatApp.AssignUserCommand{
        ChatID:     cmd.ChatID,
        AssigneeID: assigneeID,
        AssignedBy: actorID,
    }

    _, err := e.chatUseCases.AssignUser.Execute(ctx, usecaseCmd)
    if err != nil {
        return fmt.Errorf("failed to assign user: %w", err)
    }

    return nil
}

// executeChangePriority выполняет команду изменения приоритета через UseCase
func (e *CommandExecutor) executeChangePriority(
    ctx context.Context,
    cmd ChangePriorityCommand,
    actorID uuid.UUID,
) error {
    usecaseCmd := chatApp.SetPriorityCommand{
        ChatID:   cmd.ChatID,
        Priority: cmd.Priority,
        SetBy:    actorID,
    }

    _, err := e.chatUseCases.SetPriority.Execute(ctx, usecaseCmd)
    if err != nil {
        return fmt.Errorf("failed to set priority: %w", err)
    }

    return nil
}

// executeSetDueDate выполняет команду установки дедлайна через UseCase
func (e *CommandExecutor) executeSetDueDate(
    ctx context.Context,
    cmd SetDueDateCommand,
    actorID uuid.UUID,
) error {
    usecaseCmd := chatApp.SetDueDateCommand{
        ChatID:  cmd.ChatID,
        DueDate: cmd.DueDate,
        SetBy:   actorID,
    }

    _, err := e.chatUseCases.SetDueDate.Execute(ctx, usecaseCmd)
    if err != nil {
        return fmt.Errorf("failed to set due date: %w", err)
    }

    return nil
}

// executeChangeTitle выполняет команду изменения названия через UseCase
func (e *CommandExecutor) executeChangeTitle(
    ctx context.Context,
    cmd ChangeTitleCommand,
    actorID uuid.UUID,
) error {
    usecaseCmd := chatApp.RenameChatCommand{
        ChatID:    cmd.ChatID,
        NewTitle:  cmd.Title,
        RenamedBy: actorID,
    }

    _, err := e.chatUseCases.Rename.Execute(ctx, usecaseCmd)
    if err != nil {
        return fmt.Errorf("failed to rename: %w", err)
    }

    return nil
}

// executeSetSeverity выполняет команду установки severity через UseCase
func (e *CommandExecutor) executeSetSeverity(
    ctx context.Context,
    cmd SetSeverityCommand,
    actorID uuid.UUID,
) error {
    usecaseCmd := chatApp.SetSeverityCommand{
        ChatID:   cmd.ChatID,
        Severity: cmd.Severity,
        SetBy:    actorID,
    }

    _, err := e.chatUseCases.SetSeverity.Execute(ctx, usecaseCmd)
    if err != nil {
        return fmt.Errorf("failed to set severity: %w", err)
    }

    return nil
}
```

### 2. Integration с Message UseCase

Tag processing должен происходить в SendMessageUseCase:

```go
// internal/application/message/send_message.go
package message

import (
    "context"
    "fmt"

    "github.com/lllypuk/flowra/internal/application/shared"
    "github.com/lllypuk/flowra/internal/domain/chat"
    "github.com/lllypuk/flowra/internal/domain/event"
    "github.com/lllypuk/flowra/internal/domain/message"
    "github.com/lllypuk/flowra/internal/domain/tag"
    domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
)

type SendMessageUseCase struct {
    messageRepo    message.Repository
    chatRepo       chat.ReadModelRepository
    eventBus       event.Bus
    tagProcessor   *tag.Processor           // Tag processor
    tagExecutor    *tag.CommandExecutor     // Tag executor
}

func NewSendMessageUseCase(
    messageRepo message.Repository,
    chatRepo chat.ReadModelRepository,
    eventBus event.Bus,
    tagProcessor *tag.Processor,
    tagExecutor *tag.CommandExecutor,
) *SendMessageUseCase {
    return &SendMessageUseCase{
        messageRepo:  messageRepo,
        chatRepo:     chatRepo,
        eventBus:     eventBus,
        tagProcessor: tagProcessor,
        tagExecutor:  tagExecutor,
    }
}

func (uc *SendMessageUseCase) Execute(
    ctx context.Context,
    cmd SendMessageCommand,
) (MessageResult, error) {
    // ... существующая логика создания сообщения ...

    // Сохранение сообщения
    if err := uc.messageRepo.Save(ctx, msg); err != nil {
        return MessageResult{}, fmt.Errorf("failed to save message: %w", err)
    }

    // Обработка тегов (асинхронно, чтобы не блокировать ответ)
    go uc.processTagsAsync(ctx, msg, cmd.AuthorID)

    return MessageResult{
        Result: shared.Result[*message.Message]{
            Value: msg,
        },
    }, nil
}

func (uc *SendMessageUseCase) processTagsAsync(
    ctx context.Context,
    msg *message.Message,
    authorID uuid.UUID,
) {
    // Парсинг тегов
    tags, err := tag.Parse(msg.Content())
    if err != nil || len(tags) == 0 {
        return
    }

    // Обработка тегов через Processor
    commands, errors := uc.tagProcessor.Process(msg.ChatID().ToGoogleUUID(), tags)

    // Выполнение команд через Executor
    for _, cmd := range commands {
        if err := uc.tagExecutor.Execute(ctx, cmd, authorID); err != nil {
            // TODO: отправить notification об ошибке
            // или создать reply с ботом
        }
    }

    // TODO: форматирование результатов через tag.Formatter
}
```

### 3. Dependency Injection Setup

```go
// cmd/api/main.go или internal/application/setup.go
package main

import (
    chatApp "github.com/lllypuk/flowra/internal/application/chat"
    messageApp "github.com/lllypuk/flowra/internal/application/message"
    "github.com/lllypuk/flowra/internal/domain/tag"
)

func setupApplication() {
    // Repositories
    chatRepo := // ...
    messageRepo := // ...
    userRepo := // ...
    eventBus := // ...

    // Chat UseCases
    chatUseCases := &tag.ChatUseCases{
        ConvertToTask:  chatApp.NewConvertToTaskUseCase(chatRepo, eventBus),
        ConvertToBug:   chatApp.NewConvertToBugUseCase(chatRepo, eventBus),
        ConvertToEpic:  chatApp.NewConvertToEpicUseCase(chatRepo, eventBus),
        ChangeStatus:   chatApp.NewChangeStatusUseCase(chatRepo, eventBus),
        AssignUser:     chatApp.NewAssignUserUseCase(chatRepo, eventBus),
        SetPriority:    chatApp.NewSetPriorityUseCase(chatRepo, eventBus),
        SetDueDate:     chatApp.NewSetDueDateUseCase(chatRepo, eventBus),
        Rename:         chatApp.NewRenameChatUseCase(chatRepo, eventBus),
        SetSeverity:    chatApp.NewSetSeverityUseCase(chatRepo, eventBus),
    }

    // Tag components
    tagProcessor := tag.NewProcessor(/* validators */)
    tagExecutor := tag.NewCommandExecutor(chatUseCases, userRepo)

    // Message UseCase с tag integration
    sendMessageUseCase := messageApp.NewSendMessageUseCase(
        messageRepo,
        chatRepo,
        eventBus,
        tagProcessor,
        tagExecutor,
    )
}
```

## Преимущества новой архитектуры

### 1. Единая точка входа
Вся бизнес-логика в UseCases:
- Валидация
- Авторизация
- Обработка ошибок
- Event publishing

### 2. Упрощение Tag Executor
Превратился в thin adapter:
- Маппинг tag команд → usecase команды
- Резолвинг usernames
- Обработка результатов

### 3. Тестируемость
```go
func TestTagExecutor_CreateTask(t *testing.T) {
    mockConvertToTaskUseCase := mocks.NewConvertToTaskUseCase()
    chatUseCases := &tag.ChatUseCases{
        ConvertToTask: mockConvertToTaskUseCase,
    }

    executor := tag.NewCommandExecutor(chatUseCases, userRepo)

    cmd := tag.CreateTaskCommand{
        ChatID: uuid.New(),
        Title:  "Test Task",
    }

    err := executor.Execute(context.Background(), cmd, uuid.New())

    assert.NoError(t, err)
    mockConvertToTaskUseCase.AssertCalled(t, "Execute", mock.Anything, mock.Anything)
}
```

## Migration Plan

### Step 1: Создать Chat UseCases wrapper
```go
// internal/domain/tag/chat_usecases.go
package tag

import chatApp "github.com/lllypuk/flowra/internal/application/chat"

type ChatUseCases struct {
    ConvertToTask  *chatApp.ConvertToTaskUseCase
    ConvertToBug   *chatApp.ConvertToBugUseCase
    ConvertToEpic  *chatApp.ConvertToEpicUseCase
    ChangeStatus   *chatApp.ChangeStatusUseCase
    AssignUser     *chatApp.AssignUserUseCase
    SetPriority    *chatApp.SetPriorityUseCase
    SetDueDate     *chatApp.SetDueDateUseCase
    Rename         *chatApp.RenameChatUseCase
    SetSeverity    *chatApp.SetSeverityUseCase
}
```

### Step 2: Рефакторинг executor методов по одному
Начать с `executeCreateTask`, затем остальные

### Step 3: Удалить старую логику
Удалить `publishAndSave` метод и прямые обращения к `chatRepo`

### Step 4: Обновить тесты
Использовать mock UseCases вместо mock repositories

## Tests

```go
// internal/domain/tag/executor_test.go
package tag_test

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"

    "github.com/lllypuk/flowra/internal/domain/tag"
    "github.com/lllypuk/flowra/tests/mocks"
)

func TestCommandExecutor_CreateTask_CallsUseCase(t *testing.T) {
    // Arrange
    mockConvertToTaskUseCase := mocks.NewConvertToTaskUseCase()
    mockUserRepo := mocks.NewUserRepository()

    chatUseCases := &tag.ChatUseCases{
        ConvertToTask: mockConvertToTaskUseCase,
    }

    executor := tag.NewCommandExecutor(chatUseCases, mockUserRepo)

    cmd := tag.CreateTaskCommand{
        ChatID: uuid.New(),
        Title:  "Test Task",
    }
    actorID := uuid.New()

    // Expect
    mockConvertToTaskUseCase.On("Execute", mock.Anything, mock.MatchedBy(func(usecaseCmd interface{}) bool {
        // Verify mapping
        c, ok := usecaseCmd.(chatApp.ConvertToTaskCommand)
        return ok && c.ChatID == cmd.ChatID && c.Title == cmd.Title
    })).Return(chatApp.ChatResult{}, nil)

    // Act
    err := executor.Execute(context.Background(), cmd, actorID)

    // Assert
    assert.NoError(t, err)
    mockConvertToTaskUseCase.AssertExpectations(t)
}
```

## Checklist

- [x] Создать `ChatUseCases` wrapper struct
- [x] Рефакторинг `executeCreateTask` для использования UseCase
- [x] Рефакторинг `executeCreateBug` для использования UseCase
- [x] Рефакторинг `executeCreateEpic` для использования UseCase
- [x] Рефакторинг `executeChangeStatus` для использования UseCase
- [x] Рефакторинг `executeAssignUser` для использования UseCase (с резолвингом username)
- [x] Рефакторинг `executeChangePriority` для использования UseCase
- [x] Рефакторинг `executeSetDueDate` для использования UseCase
- [x] Рефакторинг `executeChangeTitle` для использования UseCase
- [x] Рефакторинг `executeSetSeverity` для использования UseCase
- [x] Удалить старый метод `publishAndSave`
- [x] Удалить зависимость от `chat.Repository` в executor
- [x] Обновить все тесты для использования mock UseCases
- [x] Интеграция tag processing в `SendMessageUseCase`
- [ ] Обновить DI setup в main.go *(будет выполнено на этапе инфраструктуры)*
- [x] Integration tests (end-to-end tag workflow)

## Следующие шаги

После завершения рефакторинга:
- Repository implementations (MongoDB)
- HTTP handlers
- WebSocket integration
