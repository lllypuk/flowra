# Task 06: Notification Domain Use Cases

**Дата:** 2025-10-19
**Статус:** 🟡 Partial (UseCases complete, Event Handlers pending)
**Зависимости:** Task 01 (Architecture)
**Оценка:** 3-4 часа

## Цель

Реализовать Use Cases для Notification entity. Notifications создаются автоматически event handlers'ами в ответ на события из других доменов.

## Контекст

**Notification entity:**
- 7 типов: TaskStatusChanged, TaskAssigned, TaskCreated, ChatMention, ChatMessage, WorkspaceInvite, System
- Read status tracking
- Resource ID linking (к task, chat, workspace и т.д.)
- Простая CRUD модель

## Use Cases для реализации

### Command Use Cases

| UseCase | Операция | Приоритет | Оценка |
|---------|----------|-----------|--------|
| CreateNotificationUseCase | Создание notification | Критичный | 1 ч |
| MarkAsReadUseCase | Пометка как прочитанное | Критичный | 0.5 ч |
| MarkAllAsReadUseCase | Пометка всех как прочитанные | Средний | 0.5 ч |
| DeleteNotificationUseCase | Удаление notification | Низкий | 0.5 ч |

### Query Use Cases

| UseCase | Операция | Приоритет | Оценка |
|---------|----------|-----------|--------|
| GetNotificationUseCase | Получение по ID | Средний | 0.5 ч |
| ListNotificationsUseCase | Список notifications пользователя | Критичный | 1.5 ч |
| CountUnreadUseCase | Количество непрочитанных | Высокий | 0.5 ч |

## Структура файлов

```
internal/application/notification/
├── commands.go
├── queries.go
├── results.go
├── errors.go
│
├── create_notification.go
├── mark_as_read.go
├── mark_all_as_read.go
├── delete_notification.go
│
├── get_notification.go
├── list_notifications.go
├── count_unread.go
│
└── *_test.go
```

## Commands

```go
package notification

import (
    "github.com/google/uuid"
    "github.com/lllypuk/flowra/internal/domain/notification"
)

// CreateNotificationCommand - создание notification
type CreateNotificationCommand struct {
    UserID       uuid.UUID
    Type         notification.Type
    ResourceID   *uuid.UUID        // ID задачи/чата/workspace
    Message      string
}

func (c CreateNotificationCommand) CommandName() string { return "CreateNotification" }

// MarkAsReadCommand - пометка как прочитанное
type MarkAsReadCommand struct {
    NotificationID uuid.UUID
    UserID         uuid.UUID       // проверка, что notification принадлежит пользователю
}

func (c MarkAsReadCommand) CommandName() string { return "MarkAsRead" }

// MarkAllAsReadCommand - пометка всех как прочитанные
type MarkAllAsReadCommand struct {
    UserID uuid.UUID
}

func (c MarkAllAsReadCommand) CommandName() string { return "MarkAllAsRead" }

// DeleteNotificationCommand - удаление notification
type DeleteNotificationCommand struct {
    NotificationID uuid.UUID
    UserID         uuid.UUID
}

func (c DeleteNotificationCommand) CommandName() string { return "DeleteNotification" }
```

## CreateNotificationUseCase (пример)

```go
package notification

import (
    "context"
    "fmt"

    "github.com/lllypuk/flowra/internal/application/shared"
    "github.com/lllypuk/flowra/internal/domain/notification"
    domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
)

type CreateNotificationUseCase struct {
    notificationRepo notification.Repository
}

func NewCreateNotificationUseCase(
    notificationRepo notification.Repository,
) *CreateNotificationUseCase {
    return &CreateNotificationUseCase{
        notificationRepo: notificationRepo,
    }
}

func (uc *CreateNotificationUseCase) Execute(
    ctx context.Context,
    cmd CreateNotificationCommand,
) (NotificationResult, error) {
    // Валидация
    if err := uc.validate(cmd); err != nil {
        return NotificationResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // Конвертация UUID
    userID := domainUUID.FromGoogleUUID(cmd.UserID)

    var resourceID *domainUUID.UUID
    if cmd.ResourceID != nil {
        rid := domainUUID.FromGoogleUUID(*cmd.ResourceID)
        resourceID = &rid
    }

    // Создание notification
    notif := notification.NewNotification(
        userID,
        cmd.Type,
        resourceID,
        cmd.Message,
    )

    // Сохранение
    if err := uc.notificationRepo.Save(ctx, notif); err != nil {
        return NotificationResult{}, fmt.Errorf("failed to save notification: %w", err)
    }

    return NotificationResult{
        Result: shared.Result[*notification.Notification]{
            Value: notif,
        },
    }, nil
}

func (uc *CreateNotificationUseCase) validate(cmd CreateNotificationCommand) error {
    if err := shared.ValidateUUID("userID", cmd.UserID); err != nil {
        return err
    }
    if err := shared.ValidateEnum("type", string(cmd.Type), []string{
        string(notification.TypeTaskStatusChanged),
        string(notification.TypeTaskAssigned),
        string(notification.TypeTaskCreated),
        string(notification.TypeChatMention),
        string(notification.TypeChatMessage),
        string(notification.TypeWorkspaceInvite),
        string(notification.TypeSystem),
    }); err != nil {
        return err
    }
    if err := shared.ValidateRequired("message", cmd.Message); err != nil {
        return err
    }
    return nil
}
```

## ListNotificationsUseCase (с пагинацией)

```go
package notification

import (
    "context"
    "fmt"

    "github.com/lllypuk/flowra/internal/application/shared"
    "github.com/lllypuk/flowra/internal/domain/notification"
    domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
)

type ListNotificationsUseCase struct {
    notificationRepo notification.Repository
}

func NewListNotificationsUseCase(
    notificationRepo notification.Repository,
) *ListNotificationsUseCase {
    return &ListNotificationsUseCase{
        notificationRepo: notificationRepo,
    }
}

func (uc *ListNotificationsUseCase) Execute(
    ctx context.Context,
    query ListNotificationsQuery,
) (NotificationsResult, error) {
    // Валидация
    if err := uc.validate(query); err != nil {
        return NotificationsResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // Дефолтные значения
    limit := query.Limit
    if limit == 0 || limit > 100 {
        limit = 50
    }

    // Получение notifications
    userID := domainUUID.FromGoogleUUID(query.UserID)

    var notifications []*notification.Notification
    var err error

    if query.UnreadOnly {
        notifications, err = uc.notificationRepo.FindUnreadByUserID(
            ctx,
            userID,
            limit,
            query.Offset,
        )
    } else {
        notifications, err = uc.notificationRepo.FindByUserID(
            ctx,
            userID,
            limit,
            query.Offset,
        )
    }

    if err != nil {
        return NotificationsResult{}, fmt.Errorf("failed to fetch notifications: %w", err)
    }

    return NotificationsResult{
        Result: shared.Result[[]*notification.Notification]{
            Value: notifications,
        },
    }, nil
}

func (uc *ListNotificationsUseCase) validate(query ListNotificationsQuery) error {
    if err := shared.ValidateUUID("userID", query.UserID); err != nil {
        return err
    }
    return nil
}
```

## Event Handlers Integration

Notifications создаются event handlers'ами:

```go
// internal/application/eventhandlers/notification_handler.go
package eventhandlers

import (
    "context"

    "github.com/lllypuk/flowra/internal/application/notification"
    "github.com/lllypuk/flowra/internal/domain/chat"
    "github.com/lllypuk/flowra/internal/domain/event"
    domainNotification "github.com/lllypuk/flowra/internal/domain/notification"
)

type NotificationEventHandler struct {
    createNotificationUseCase *notification.CreateNotificationUseCase
}

func NewNotificationEventHandler(
    createNotificationUseCase *notification.CreateNotificationUseCase,
) *NotificationEventHandler {
    return &NotificationEventHandler{
        createNotificationUseCase: createNotificationUseCase,
    }
}

// HandleChatCreated обрабатывает событие создания чата
func (h *NotificationEventHandler) HandleChatCreated(ctx context.Context, evt event.Event) error {
    chatCreatedEvt, ok := evt.(chat.ChatCreatedEvent)
    if !ok {
        return fmt.Errorf("invalid event type")
    }

    // Создание notification для участников (кроме создателя)
    // TODO: получить участников из Chat aggregate

    cmd := notification.CreateNotificationCommand{
        UserID:     chatCreatedEvt.CreatedBy.ToGoogleUUID(), // для примера
        Type:       domainNotification.TypeChatMessage,
        ResourceID: &chatCreatedEvt.ChatID.ToGoogleUUID(),
        Message:    fmt.Sprintf("New chat created: %s", chatCreatedEvt.Title),
    }

    _, err := h.createNotificationUseCase.Execute(ctx, cmd)
    return err
}

// HandleUserAssigned обрабатывает назначение пользователя на задачу
func (h *NotificationEventHandler) HandleUserAssigned(ctx context.Context, evt event.Event) error {
    assignedEvt, ok := evt.(chat.UserAssignedEvent)
    if !ok {
        return fmt.Errorf("invalid event type")
    }

    if assignedEvt.AssigneeID == nil {
        return nil // снятие assignee, не нотифицируем
    }

    cmd := notification.CreateNotificationCommand{
        UserID:     assignedEvt.AssigneeID.ToGoogleUUID(),
        Type:       domainNotification.TypeTaskAssigned,
        ResourceID: &assignedEvt.ChatID.ToGoogleUUID(),
        Message:    "You have been assigned to a task",
    }

    _, err := h.createNotificationUseCase.Execute(ctx, cmd)
    return err
}
```

## Subscription Setup

```go
// internal/application/setup.go
func SetupEventBus(
    eventBus event.Bus,
    notificationHandler *eventhandlers.NotificationEventHandler,
) {
    // Chat events
    eventBus.Subscribe(chat.EventTypeChatCreated, notificationHandler.HandleChatCreated)
    eventBus.Subscribe(chat.EventTypeUserAssigned, notificationHandler.HandleUserAssigned)
    eventBus.Subscribe(chat.EventTypeStatusChanged, notificationHandler.HandleStatusChanged)

    // Message events
    eventBus.Subscribe(message.EventTypeMessageSent, notificationHandler.HandleMessageSent)

    // И т.д.
}
```

## Tests

```go
func TestCreateNotificationUseCase_Success(t *testing.T) {
    notificationRepo := mocks.NewNotificationRepository()
    useCase := NewCreateNotificationUseCase(notificationRepo)

    cmd := CreateNotificationCommand{
        UserID:     uuid.New(),
        Type:       notification.TypeTaskAssigned,
        ResourceID: ptr(uuid.New()),
        Message:    "You have been assigned to a task",
    }

    result, err := useCase.Execute(context.Background(), cmd)

    assert.NoError(t, err)
    assert.NotNil(t, result.Value)
    assert.Equal(t, cmd.Type, result.Value.Type())
    assert.False(t, result.Value.IsRead())
}

func TestListNotificationsUseCase_UnreadOnly(t *testing.T) {
    notificationRepo := mocks.NewNotificationRepository()
    userID := uuid.New()

    // Setup: 3 unread, 2 read notifications
    notificationRepo.AddNotifications(userID, 3, false)
    notificationRepo.AddNotifications(userID, 2, true)

    useCase := NewListNotificationsUseCase(notificationRepo)

    query := ListNotificationsQuery{
        UserID:     userID,
        UnreadOnly: true,
        Limit:      10,
    }

    result, err := useCase.Execute(context.Background(), query)

    assert.NoError(t, err)
    assert.Len(t, result.Value, 3) // только unread
}

func ptr[T any](v T) *T { return &v }
```

## Checklist

- [x] Создать `commands.go`, `queries.go`, `results.go`, `errors.go`
- [x] CreateNotificationUseCase + tests
- [x] MarkAsReadUseCase + tests
- [x] MarkAllAsReadUseCase + tests
- [x] DeleteNotificationUseCase + tests
- [x] GetNotificationUseCase + tests
- [x] ListNotificationsUseCase + tests (с pagination и unread filter)
- [x] CountUnreadUseCase + tests
- [ ] Event handlers (NotificationEventHandler) ❌ NOT IMPLEMENTED
- [ ] Event bus subscription setup ❌ NOT IMPLEMENTED
- [ ] Integration tests (event → notification workflow) ❌ NOT IMPLEMENTED

## Следующие шаги

- **Task 07**: Integration Testing
- Event bus implementation (Redis)
- WebSocket integration для real-time notifications
