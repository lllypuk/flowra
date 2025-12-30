# Task 03: Message Domain Use Cases

**Дата:** 2025-10-19
**Статус:** ✅ Complete
**Зависимости:** Task 01 (Architecture), Task 02 (Chat UseCases)
**Оценка:** 5-7 часов

## Цель

Реализовать все Use Cases для Message entity с полным тестовым покрытием. Message - это основная сущность для обмена сообщениями в чатах.

## Контекст

**Message entity поддерживает:**
- Создание сообщений в чатах
- Редактирование (с отслеживанием editedAt)
- Soft delete (deletedAt)
- Треды (parentMessageID для replies)
- Реакции (emoji per user)
- Вложения файлов
- Авторизация (только автор может редактировать/удалять)

**Особенность**: Message НЕ использует Event Sourcing, это простая CRUD entity.

## Use Cases для реализации

### Command Use Cases

| UseCase | Операция | Приоритет | Оценка |
|---------|----------|-----------|--------|
| SendMessageUseCase | Отправка сообщения | Критичный | 1.5 ч |
| EditMessageUseCase | Редактирование | Критичный | 1 ч |
| DeleteMessageUseCase | Удаление (soft) | Критичный | 1 ч |
| AddReactionUseCase | Добавление реакции | Высокий | 0.5 ч |
| RemoveReactionUseCase | Удаление реакции | Высокий | 0.5 ч |
| AddAttachmentUseCase | Добавление вложения | Средний | 1 ч |

### Query Use Cases

| UseCase | Операция | Приоритет | Оценка |
|---------|----------|-----------|--------|
| GetMessageUseCase | Получение по ID | Критичный | 0.5 ч |
| ListMessagesUseCase | Список сообщений чата | Критичный | 1 ч |
| GetThreadUseCase | Получение треда | Высокий | 0.5 ч |

## Структура файлов

```
internal/application/message/
├── commands.go            # Команды
├── queries.go             # Запросы
├── results.go             # Результаты
├── errors.go              # Ошибки
│
├── send_message.go        # SendMessageUseCase
├── edit_message.go        # EditMessageUseCase
├── delete_message.go      # DeleteMessageUseCase
├── add_reaction.go        # AddReactionUseCase
├── remove_reaction.go     # RemoveReactionUseCase
├── add_attachment.go      # AddAttachmentUseCase
│
├── get_message.go         # GetMessageUseCase
├── list_messages.go       # ListMessagesUseCase
├── get_thread.go          # GetThreadUseCase
│
└── *_test.go
```

## Детальное описание

### 1. Commands

```go
package message

import (
    "github.com/google/uuid"
    "github.com/lllypuk/flowra/internal/application/shared"
)

// SendMessageCommand - отправка сообщения
type SendMessageCommand struct {
    ChatID          uuid.UUID
    Content         string
    AuthorID        uuid.UUID
    ParentMessageID *uuid.UUID   // для replies
}

func (c SendMessageCommand) CommandName() string { return "SendMessage" }

// EditMessageCommand - редактирование сообщения
type EditMessageCommand struct {
    MessageID uuid.UUID
    Content   string
    EditorID  uuid.UUID        // должен совпадать с AuthorID
}

func (c EditMessageCommand) CommandName() string { return "EditMessage" }

// DeleteMessageCommand - удаление сообщения
type DeleteMessageCommand struct {
    MessageID uuid.UUID
    DeletedBy uuid.UUID        // должен совпадать с AuthorID
}

func (c DeleteMessageCommand) CommandName() string { return "DeleteMessage" }

// AddReactionCommand - добавление реакции
type AddReactionCommand struct {
    MessageID uuid.UUID
    Emoji     string
    UserID    uuid.UUID
}

func (c AddReactionCommand) CommandName() string { return "AddReaction" }

// RemoveReactionCommand - удаление реакции
type RemoveReactionCommand struct {
    MessageID uuid.UUID
    Emoji     string
    UserID    uuid.UUID
}

func (c RemoveReactionCommand) CommandName() string { return "RemoveReaction" }

// AddAttachmentCommand - добавление вложения
type AddAttachmentCommand struct {
    MessageID uuid.UUID
    FileID    string
    FileName  string
    FileSize  int64
    MimeType  string
    UserID    uuid.UUID        // должен совпадать с AuthorID
}

func (c AddAttachmentCommand) CommandName() string { return "AddAttachment" }
```

### 2. Errors

```go
package message

import (
    "errors"
)

var (
    // Validation errors
    ErrEmptyContent          = errors.New("message content cannot be empty")
    ErrContentTooLong        = errors.New("message content too long")
    ErrInvalidEmoji          = errors.New("invalid emoji")
    ErrInvalidFileSize       = errors.New("file size exceeds limit")

    // Business logic errors
    ErrMessageNotFound       = errors.New("message not found")
    ErrChatNotFound          = errors.New("chat not found")
    ErrParentNotFound        = errors.New("parent message not found")
    ErrNotAuthor             = errors.New("user is not the message author")
    ErrMessageDeleted        = errors.New("message is deleted")
    ErrReactionAlreadyExists = errors.New("reaction already exists")
    ErrReactionNotFound      = errors.New("reaction not found")

    // Authorization
    ErrNotChatParticipant    = errors.New("user is not a chat participant")
)

const (
    MaxContentLength = 10000    // 10k символов
    MaxFileSize      = 10 << 20 // 10 MB
)
```

### 3. SendMessageUseCase (пример)

```go
package message

import (
    "context"
    "fmt"

    "github.com/lllypuk/flowra/internal/application/shared"
    "github.com/lllypuk/flowra/internal/domain/chat"
    "github.com/lllypuk/flowra/internal/domain/event"
    "github.com/lllypuk/flowra/internal/domain/message"
    domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
)

type SendMessageUseCase struct {
    messageRepo message.Repository
    chatRepo    chat.ReadModelRepository  // для проверки доступа
    eventBus    event.Bus
}

func NewSendMessageUseCase(
    messageRepo message.Repository,
    chatRepo chat.ReadModelRepository,
    eventBus event.Bus,
) *SendMessageUseCase {
    return &SendMessageUseCase{
        messageRepo: messageRepo,
        chatRepo:    chatRepo,
        eventBus:    eventBus,
    }
}

func (uc *SendMessageUseCase) Execute(
    ctx context.Context,
    cmd SendMessageCommand,
) (MessageResult, error) {
    // 1. Валидация
    if err := uc.validate(cmd); err != nil {
        return MessageResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // 2. Проверка доступа к чату
    chatID := domainUUID.FromGoogleUUID(cmd.ChatID)
    chatReadModel, err := uc.chatRepo.FindByID(ctx, chatID)
    if err != nil {
        return MessageResult{}, ErrChatNotFound
    }

    authorID := domainUUID.FromGoogleUUID(cmd.AuthorID)
    if !chatReadModel.IsParticipant(authorID) {
        return MessageResult{}, ErrNotChatParticipant
    }

    // 3. Проверка parent message (если это reply)
    if cmd.ParentMessageID != nil {
        parentID := domainUUID.FromGoogleUUID(*cmd.ParentMessageID)
        parent, err := uc.messageRepo.FindByID(ctx, parentID)
        if err != nil {
            return MessageResult{}, ErrParentNotFound
        }
        // Проверка, что parent в том же чате
        if parent.ChatID() != chatID {
            return MessageResult{}, shared.NewValidationError("parentMessageID", "parent message is from different chat")
        }
    }

    // 4. Создание сообщения
    var parentMessageID *domainUUID.UUID
    if cmd.ParentMessageID != nil {
        pid := domainUUID.FromGoogleUUID(*cmd.ParentMessageID)
        parentMessageID = &pid
    }

    msg := message.NewMessage(
        chatID,
        authorID,
        cmd.Content,
        parentMessageID,
    )

    // 5. Сохранение
    if err := uc.messageRepo.Save(ctx, msg); err != nil {
        return MessageResult{}, fmt.Errorf("failed to save message: %w", err)
    }

    // 6. Публикация события (для WebSocket broadcast)
    evt := message.MessageSentEvent{
        MessageID: msg.ID(),
        ChatID:    chatID,
        AuthorID:  authorID,
        Content:   cmd.Content,
    }
    if err := uc.eventBus.Publish(ctx, evt); err != nil {
        // Не критично, сообщение уже сохранено
        // TODO: log error
    }

    return MessageResult{
        Result: shared.Result[*message.Message]{
            Value: msg,
        },
    }, nil
}

func (uc *SendMessageUseCase) validate(cmd SendMessageCommand) error {
    if err := shared.ValidateUUID("chatID", cmd.ChatID); err != nil {
        return err
    }
    if err := shared.ValidateRequired("content", cmd.Content); err != nil {
        return err
    }
    if len(cmd.Content) > MaxContentLength {
        return ErrContentTooLong
    }
    if err := shared.ValidateUUID("authorID", cmd.AuthorID); err != nil {
        return err
    }
    return nil
}
```

### 4. EditMessageUseCase (пример с авторизацией)

```go
package message

import (
    "context"
    "fmt"

    "github.com/lllypuk/flowra/internal/application/shared"
    "github.com/lllypuk/flowra/internal/domain/event"
    "github.com/lllypuk/flowra/internal/domain/message"
    domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
)

type EditMessageUseCase struct {
    messageRepo message.Repository
    eventBus    event.Bus
}

func NewEditMessageUseCase(
    messageRepo message.Repository,
    eventBus event.Bus,
) *EditMessageUseCase {
    return &EditMessageUseCase{
        messageRepo: messageRepo,
        eventBus:    eventBus,
    }
}

func (uc *EditMessageUseCase) Execute(
    ctx context.Context,
    cmd EditMessageCommand,
) (MessageResult, error) {
    // Валидация
    if err := uc.validate(cmd); err != nil {
        return MessageResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // Загрузка сообщения
    messageID := domainUUID.FromGoogleUUID(cmd.MessageID)
    msg, err := uc.messageRepo.FindByID(ctx, messageID)
    if err != nil {
        return MessageResult{}, ErrMessageNotFound
    }

    // Авторизация (только автор может редактировать)
    editorID := domainUUID.FromGoogleUUID(cmd.EditorID)
    if msg.AuthorID() != editorID {
        return MessageResult{}, ErrNotAuthor
    }

    // Редактирование
    if err := msg.EditContent(cmd.Content); err != nil {
        return MessageResult{}, err
    }

    // Сохранение
    if err := uc.messageRepo.Save(ctx, msg); err != nil {
        return MessageResult{}, fmt.Errorf("failed to save message: %w", err)
    }

    // Публикация события
    evt := message.MessageEditedEvent{
        MessageID: messageID,
        ChatID:    msg.ChatID(),
        Content:   cmd.Content,
    }
    _ = uc.eventBus.Publish(ctx, evt)

    return MessageResult{
        Result: shared.Result[*message.Message]{
            Value: msg,
        },
    }, nil
}

func (uc *EditMessageUseCase) validate(cmd EditMessageCommand) error {
    if err := shared.ValidateUUID("messageID", cmd.MessageID); err != nil {
        return err
    }
    if err := shared.ValidateRequired("content", cmd.Content); err != nil {
        return err
    }
    if len(cmd.Content) > MaxContentLength {
        return ErrContentTooLong
    }
    return nil
}
```

## Специальные требования

### 1. Tag Parsing Integration

SendMessageUseCase должен интегрироваться с Tag Parser:

```go
func (uc *SendMessageUseCase) Execute(ctx context.Context, cmd SendMessageCommand) (MessageResult, error) {
    // ... создание и сохранение сообщения ...

    // Парсинг тегов
    tags, err := uc.tagParser.Parse(cmd.Content)
    if err == nil && len(tags) > 0 {
        // Обработка тегов через Tag Processor
        go uc.processTagsAsync(ctx, msg.ID(), tags)
    }

    return result, nil
}
```

### 2. WebSocket Broadcasting

После создания/редактирования/удаления сообщения должны отправляться события для WebSocket:

```go
type MessageSentEvent struct {
    MessageID uuid.UUID
    ChatID    uuid.UUID
    AuthorID  uuid.UUID
    Content   string
    CreatedAt time.Time
}
```

### 3. Pagination для ListMessagesUseCase

```go
type ListMessagesQuery struct {
    ChatID uuid.UUID
    Limit  int           // default: 50, max: 100
    Offset int
    Before *time.Time    // для pagination по времени
}
```

## Tests

```go
func TestSendMessageUseCase_Success(t *testing.T) {
    messageRepo := mocks.NewMessageRepository()
    chatRepo := mocks.NewChatReadModelRepository()
    eventBus := mocks.NewEventBus()

    // Setup chat with participant
    chatID := uuid.New()
    authorID := uuid.New()
    chatRepo.AddChat(chatID, []uuid.UUID{authorID})

    useCase := NewSendMessageUseCase(messageRepo, chatRepo, eventBus)

    cmd := SendMessageCommand{
        ChatID:   chatID,
        Content:  "Hello, world!",
        AuthorID: authorID,
    }

    result, err := useCase.Execute(context.Background(), cmd)

    assert.NoError(t, err)
    assert.NotNil(t, result.Value)
    assert.Equal(t, cmd.Content, result.Value.Content())
}

func TestSendMessageUseCase_NotParticipant(t *testing.T) {
    messageRepo := mocks.NewMessageRepository()
    chatRepo := mocks.NewChatReadModelRepository()
    eventBus := mocks.NewEventBus()

    chatID := uuid.New()
    chatRepo.AddChat(chatID, []uuid.UUID{}) // нет участников

    useCase := NewSendMessageUseCase(messageRepo, chatRepo, eventBus)

    cmd := SendMessageCommand{
        ChatID:   chatID,
        Content:  "Hello",
        AuthorID: uuid.New(), // не участник
    }

    result, err := useCase.Execute(context.Background(), cmd)

    assert.Error(t, err)
    assert.ErrorIs(t, err, ErrNotChatParticipant)
}
```

## Checklist

- [x] Создать `commands.go`, `queries.go`, `results.go`, `errors.go`
- [x] SendMessageUseCase + tests
- [x] EditMessageUseCase + tests
- [x] DeleteMessageUseCase + tests
- [x] AddReactionUseCase + tests
- [x] RemoveReactionUseCase + tests
- [x] AddAttachmentUseCase + tests
- [x] GetMessageUseCase + tests
- [x] ListMessagesUseCase + tests (с pagination)
- [x] GetThreadUseCase + tests
- [x] Integration tests (message lifecycle)

## Следующие шаги

- **Task 04**: User UseCases
- Интеграция с Tag Parser (Task 08)
- WebSocket broadcasting (будущее)
