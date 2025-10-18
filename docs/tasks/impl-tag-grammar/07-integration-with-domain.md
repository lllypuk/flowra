# Task 07: Integration with Domain Model

**Статус:** Pending
**Приоритет:** High
**Зависимости:** Task 03, Task 04, Task 05, Task 06
**Оценка:** 3-4 дня

## Описание

Интегрировать систему тегов с domain model. Создать команды для Chat aggregate, реализовать CommandExecutor, обеспечить применение тегов через event sourcing.

## Цели

1. Создать domain commands для всех тегов
2. Реализовать CommandExecutor для Chat aggregate
3. Интегрировать с event sourcing
4. Обработать специальные случаи (создание typed чата, превращение чата в задачу)

## Технические требования

### Domain Commands

```go
// internal/domain/chat/commands.go
package chat

// Entity Creation
type ConvertToTaskCommand struct {
    ChatID  uuid.UUID
    Title   string
    UserID  uuid.UUID // кто выполнил команду
}

type ConvertToBugCommand struct {
    ChatID   uuid.UUID
    Title    string
    UserID   uuid.UUID
}

type ConvertToEpicCommand struct {
    ChatID  uuid.UUID
    Title   string
    UserID  uuid.UUID
}

// Entity Management
type ChangeStatusCommand struct {
    ChatID  uuid.UUID
    Status  string
    UserID  uuid.UUID
}

type AssignUserCommand struct {
    ChatID     uuid.UUID
    AssigneeID *uuid.UUID // nil = remove assignee
    UserID     uuid.UUID
}

type SetPriorityCommand struct {
    ChatID   uuid.UUID
    Priority string
    UserID   uuid.UUID
}

type SetDueDateCommand struct {
    ChatID  uuid.UUID
    DueDate *time.Time // nil = remove due date
    UserID  uuid.UUID
}

type RenameChatCommand struct {
    ChatID  uuid.UUID
    NewTitle string
    UserID  uuid.UUID
}

type SetSeverityCommand struct {
    ChatID   uuid.UUID
    Severity string
    UserID   uuid.UUID
}
```

### Domain Events

```go
// internal/domain/chat/events.go
package chat

// Entity Creation Events
type ChatConvertedToTaskEvent struct {
    ChatID    uuid.UUID
    Title     string
    UserID    uuid.UUID
    Timestamp time.Time
}

type ChatConvertedToBugEvent struct {
    ChatID    uuid.UUID
    Title     string
    UserID    uuid.UUID
    Timestamp time.Time
}

// Entity Management Events
type StatusChangedEvent struct {
    ChatID    uuid.UUID
    OldStatus string
    NewStatus string
    UserID    uuid.UUID
    Timestamp time.Time
}

type UserAssignedEvent struct {
    ChatID     uuid.UUID
    AssigneeID uuid.UUID
    UserID     uuid.UUID
    Timestamp  time.Time
}

type AssigneeRemovedEvent struct {
    ChatID    uuid.UUID
    UserID    uuid.UUID
    Timestamp time.Time
}

// ... другие события
```

### Chat Aggregate Methods

```go
// internal/domain/chat/aggregate.go
package chat

type Chat struct {
    ID          uuid.UUID
    Type        ChatType  // "discussion", "task", "bug", "epic"
    Title       string
    Status      string
    AssigneeID  *uuid.UUID
    Priority    string
    DueDate     *time.Time
    Severity    string // только для Bug
    // ... другие поля
}

func (c *Chat) ConvertToTask(title string, userID uuid.UUID) error {
    // Валидация
    if c.Type == TypeTask {
        // Уже Task - можно обновить title
        if c.Title != title {
            return c.Rename(title, userID)
        }
        return nil
    }

    // Создание события
    event := ChatConvertedToTaskEvent{
        ChatID:    c.ID,
        Title:     title,
        UserID:    userID,
        Timestamp: time.Now(),
    }

    // Применение события
    c.Apply(event)
    return nil
}

func (c *Chat) ChangeStatus(newStatus string, userID uuid.UUID) error {
    // Валидация
    if err := c.validateStatus(newStatus); err != nil {
        return err
    }

    if c.Status == newStatus {
        return nil // Нет изменений
    }

    // Создание события
    event := StatusChangedEvent{
        ChatID:    c.ID,
        OldStatus: c.Status,
        NewStatus: newStatus,
        UserID:    userID,
        Timestamp: time.Now(),
    }

    c.Apply(event)
    return nil
}

func (c *Chat) validateStatus(status string) error {
    var validStatuses []string

    switch c.Type {
    case TypeTask:
        validStatuses = []string{"To Do", "In Progress", "Done"}
    case TypeBug:
        validStatuses = []string{"New", "Investigating", "Fixed", "Verified"}
    case TypeEpic:
        validStatuses = []string{"Planned", "In Progress", "Completed"}
    default:
        return fmt.Errorf("cannot set status on discussion chat")
    }

    for _, valid := range validStatuses {
        if status == valid {
            return nil
        }
    }

    return fmt.Errorf("invalid status '%s' for %s. Available: %s",
        status, c.Type, strings.Join(validStatuses, ", "))
}

func (c *Chat) AssignUser(assigneeID *uuid.UUID, userID uuid.UUID) error {
    // Снятие assignee
    if assigneeID == nil {
        if c.AssigneeID == nil {
            return nil // Уже нет assignee
        }

        event := AssigneeRemovedEvent{
            ChatID:    c.ID,
            UserID:    userID,
            Timestamp: time.Now(),
        }
        c.Apply(event)
        return nil
    }

    // Назначение assignee
    event := UserAssignedEvent{
        ChatID:     c.ID,
        AssigneeID: *assigneeID,
        UserID:     userID,
        Timestamp:  time.Now(),
    }
    c.Apply(event)
    return nil
}

// ... остальные методы
```

### CommandExecutor

```go
// internal/tag/executor.go
package tag

type CommandExecutor struct {
    chatRepo repository.ChatRepository
    userRepo repository.UserRepository
    eventBus event.EventBus
}

func (e *CommandExecutor) Execute(cmd Command) error {
    switch c := cmd.(type) {
    case CreateTaskCommand:
        return e.executeCreateTask(c)
    case CreateBugCommand:
        return e.executeCreateBug(c)
    case ChangeStatusCommand:
        return e.executeChangeStatus(c)
    case AssignUserCommand:
        return e.executeAssignUser(c)
    // ... другие команды
    default:
        return fmt.Errorf("unknown command type: %T", cmd)
    }
}

func (e *CommandExecutor) executeCreateTask(cmd CreateTaskCommand) error {
    // Загрузка чата
    chat, err := e.chatRepo.FindByID(cmd.ChatID)
    if err != nil {
        return err
    }

    // Выполнение команды на aggregate
    if err := chat.ConvertToTask(cmd.Title, cmd.UserID); err != nil {
        return err
    }

    // Сохранение событий
    events := chat.GetUncommittedEvents()
    for _, event := range events {
        if err := e.eventBus.Publish(event); err != nil {
            return err
        }
    }

    // Сохранение в репозиторий
    if err := e.chatRepo.Save(chat); err != nil {
        return err
    }

    return nil
}

func (e *CommandExecutor) executeAssignUser(cmd AssignUserCommand) error {
    chat, err := e.chatRepo.FindByID(cmd.ChatID)
    if err != nil {
        return err
    }

    // Резолвинг пользователя (если не снятие assignee)
    var assigneeID *uuid.UUID
    if cmd.Username != "" && cmd.Username != "@none" {
        username := strings.TrimPrefix(cmd.Username, "@")
        user, err := e.userRepo.FindByUsername(username)
        if err != nil {
            return fmt.Errorf("user %s not found", cmd.Username)
        }
        assigneeID = &user.ID
    }

    // Выполнение команды
    if err := chat.AssignUser(assigneeID, cmd.UserID); err != nil {
        return err
    }

    // Публикация событий и сохранение
    events := chat.GetUncommittedEvents()
    for _, event := range events {
        if err := e.eventBus.Publish(event); err != nil {
            return err
        }
    }

    if err := e.chatRepo.Save(chat); err != nil {
        return err
    }

    return nil
}
```

### Интеграция TagProcessor и CommandExecutor

```go
// internal/tag/handler.go
package tag

func (h *TagHandler) HandleMessageWithTags(chatID uuid.UUID, authorID uuid.UUID, content string) error {
    // 1. Парсинг тегов
    parseResult := h.parser.Parse(content)

    // 2. Валидация
    ctx := h.getValidationContext(chatID)
    validTags, validationErrors := h.validator.ValidateTags(parseResult.Tags, ctx)

    // 3. Сохранение сообщения
    message := domain.Message{
        ID:        uuid.New(),
        ChatID:    chatID,
        AuthorID:  authorID,
        Content:   content,
        CreatedAt: time.Now(),
    }
    if err := h.messageRepo.Save(message); err != nil {
        return err
    }

    // 4. Преобразование тегов в команды
    commands := h.processor.ConvertToCommands(chatID, authorID, validTags)

    // 5. Выполнение команд через CommandExecutor
    var executionErrors []error
    for _, cmd := range commands {
        if err := h.executor.Execute(cmd); err != nil {
            executionErrors = append(executionErrors, err)
        }
    }

    // 6. Генерация bot response
    allErrors := append(validationErrors, executionErrors...)
    if botResponse := h.generateBotResponse(validTags, allErrors); botResponse != "" {
        h.sendBotResponse(chatID, botResponse)
    }

    return nil
}
```

## Acceptance Criteria

- [ ] Созданы все domain commands для тегов
- [ ] Созданы все domain events
- [ ] Реализованы методы Chat aggregate для всех операций
- [ ] Реализован `CommandExecutor` с методом `Execute()`
- [ ] Реализованы все execute-методы для команд
- [ ] Интегрирован с event sourcing (публикация событий)
- [ ] Обработано превращение обычного чата в typed (Task/Bug/Epic)
- [ ] Обработано изменение типа чата (Task → Bug)
- [ ] Валидация в aggregate согласуется с валидацией тегов
- [ ] Код покрыт unit-тестами

## Примеры использования

### Пример 1: Превращение чата в Task
```go
User в обычном discussion-чате:
"#task Разобраться с производительностью"

Обработка:
1. Парсинг: tag{key: "task", value: "Разобраться с производительностью"}
2. Валидация: ✅
3. Команда: CreateTaskCommand{ChatID: ..., Title: "..."}
4. Aggregate: chat.ConvertToTask(title, userID)
5. Event: ChatConvertedToTaskEvent
6. Проекция обновляется: chat.Type = "task", chat.Title = "..."
7. Bot response: "✅ Task created: Разобраться с производительностью"
```

### Пример 2: Изменение статуса
```go
User в Task-чате:
"#status In Progress"

Обработка:
1. Парсинг: tag{key: "status", value: "In Progress"}
2. Валидация: ✅ (для Task допустимо)
3. Команда: ChangeStatusCommand{ChatID: ..., Status: "In Progress"}
4. Aggregate: chat.ChangeStatus("In Progress", userID)
5. Event: StatusChangedEvent{OldStatus: "To Do", NewStatus: "In Progress"}
6. Проекция обновляется: chat.Status = "In Progress"
7. Bot response: "✅ Status changed to In Progress"
```

### Пример 3: Назначение assignee
```go
User: "#assignee @alex"

Обработка:
1. Парсинг: tag{key: "assignee", value: "@alex"}
2. Валидация: ✅ формат
3. Резолвинг: userRepo.FindByUsername("alex") → UUID
4. Команда: AssignUserCommand{Username: "@alex"}
5. Executor: резолвит @alex в userID
6. Aggregate: chat.AssignUser(userID, actorID)
7. Event: UserAssignedEvent
8. Bot response: "✅ Assigned to: @alex"
```

## Тесты

```go
func TestConvertToTask(t *testing.T) {
    chat := NewChat(uuid.New(), TypeDiscussion)

    err := chat.ConvertToTask("Test Task", userID)

    assert.NoError(t, err)
    assert.Equal(t, TypeTask, chat.Type)
    assert.Equal(t, "Test Task", chat.Title)
    assert.Equal(t, "To Do", chat.Status) // default status

    events := chat.GetUncommittedEvents()
    assert.Len(t, events, 1)
    assert.IsType(t, ChatConvertedToTaskEvent{}, events[0])
}

func TestChangeStatus(t *testing.T) {
    chat := NewChat(uuid.New(), TypeTask)
    chat.Status = "To Do"

    err := chat.ChangeStatus("In Progress", userID)

    assert.NoError(t, err)
    assert.Equal(t, "In Progress", chat.Status)

    events := chat.GetUncommittedEvents()
    assert.Len(t, events, 1)

    event := events[0].(StatusChangedEvent)
    assert.Equal(t, "To Do", event.OldStatus)
    assert.Equal(t, "In Progress", event.NewStatus)
}

func TestChangeStatusInvalidForType(t *testing.T) {
    chat := NewChat(uuid.New(), TypeTask)

    err := chat.ChangeStatus("Fixed", userID) // Bug status

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "Invalid status 'Fixed' for Task")
}

func TestExecuteAssignUser(t *testing.T) {
    executor := NewCommandExecutor(chatRepo, userRepo, eventBus)

    // Setup mock
    userRepo.AddUser("alex", alexID)

    cmd := AssignUserCommand{
        ChatID:   chatID,
        Username: "@alex",
        UserID:   actorID,
    }

    err := executor.Execute(cmd)

    assert.NoError(t, err)

    // Проверяем, что событие опубликовано
    assert.True(t, eventBus.WasPublished(UserAssignedEvent{}))

    // Проверяем, что чат сохранён
    chat := chatRepo.FindByID(chatID)
    assert.Equal(t, &alexID, chat.AssigneeID)
}
```

## Файловая структура

```
internal/
├── domain/
│   └── chat/
│       ├── aggregate.go       # Chat aggregate
│       ├── commands.go        # Domain commands
│       ├── events.go          # Domain events
│       └── aggregate_test.go  # Тесты aggregate
└── tag/
    ├── executor.go            # CommandExecutor
    ├── handler.go             # Интеграция всего pipeline
    └── executor_test.go       # Тесты executor
```

## Ссылки

- Event Sourcing: уже реализовано в Phase 1
- Chat Aggregate: `internal/domain/chat/aggregate.go`
- Сценарии превращения чата: `docs/03-tag-grammar.md` (строки 811-848)
