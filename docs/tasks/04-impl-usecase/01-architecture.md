# Task 01: UseCase Layer Architecture

**Дата:** 2025-10-19
**Статус:** 📝 Pending
**Зависимости:** Domain models (все)
**Оценка:** 3-4 часа

## Цель

Спроектировать и реализовать базовую архитектуру слоя Use Cases для всех доменных агрегатов. Определить shared компоненты, паттерны, интерфейсы и структуру кода, которые будут использоваться во всех use cases.

## Контекст

**У нас есть:**
- ✅ Domain models: Chat, Message, Task, User, Workspace, Notification, Tag
- ✅ Event Sourcing в Chat и Task агрегатах
- ✅ Tag.CommandExecutor (частичная UseCase логика)
- ✅ Repository интерфейсы в доменах

**Отсутствует:**
- ❌ Единая UseCase архитектура
- ❌ Shared компоненты для всех UseCases
- ❌ Application layer структура

## Архитектурные решения

### 1. Структура директорий

```
internal/application/
├── shared/
│   ├── interfaces.go          # Общие интерфейсы (UseCase, Command, Result)
│   ├── base.go                # Базовая функциональность
│   ├── errors.go              # Общие ошибки
│   ├── validation.go          # Общие валидаторы
│   └── context.go             # Context utilities
├── chat/
│   ├── commands.go            # Все команды
│   ├── results.go             # Результаты
│   ├── errors.go              # Специфичные ошибки
│   ├── create_chat.go
│   ├── add_participant.go
│   └── ... (остальные UseCases)
├── message/
│   ├── commands.go
│   ├── results.go
│   ├── errors.go
│   └── ... (UseCases)
├── user/
│   └── ...
├── workspace/
│   └── ...
└── notification/
    └── ...
```

### 2. Shared Interfaces

#### File: `internal/application/shared/interfaces.go`

```go
package shared

import (
    "context"
)

// UseCase - базовый интерфейс для всех use cases
type UseCase[TCommand any, TResult any] interface {
    Execute(ctx context.Context, cmd TCommand) (TResult, error)
}

// Command - маркер интерфейс для команд
type Command interface {
    CommandName() string
}

// Query - маркер интерфейс для запросов (CQRS)
type Query interface {
    QueryName() string
}

// Result - базовая структура результата
type Result[T any] struct {
    Value   T
    Version int
    Error   error
}

func (r Result[T]) IsSuccess() bool {
    return r.Error == nil
}

func (r Result[T]) IsFailure() bool {
    return r.Error != nil
}

// EventSourcedResult - результат для event-sourced операций
type EventSourcedResult[T any] struct {
    Result[T]
    Events []interface{} // domain events
}

// Validator - интерфейс для валидации команд
type Validator[T any] interface {
    Validate(cmd T) error
}

// UnitOfWork - интерфейс для транзакционности
type UnitOfWork interface {
    Begin(ctx context.Context) error
    Commit(ctx context.Context) error
    Rollback(ctx context.Context) error
}
```

### 3. Shared Errors

#### File: `internal/application/shared/errors.go`

```go
package shared

import (
    "errors"
    "fmt"
)

// Common application errors
var (
    // Validation errors
    ErrValidationFailed   = errors.New("validation failed")
    ErrInvalidID          = errors.New("invalid ID")
    ErrEmptyField         = errors.New("required field is empty")
    ErrInvalidFormat      = errors.New("invalid format")

    // Authorization errors
    ErrUnauthorized       = errors.New("unauthorized")
    ErrForbidden          = errors.New("forbidden")
    ErrInsufficientPermissions = errors.New("insufficient permissions")

    // Not found errors
    ErrNotFound           = errors.New("resource not found")
    ErrChatNotFound       = errors.New("chat not found")
    ErrMessageNotFound    = errors.New("message not found")
    ErrUserNotFound       = errors.New("user not found")
    ErrWorkspaceNotFound  = errors.New("workspace not found")

    // Conflict errors
    ErrConflict           = errors.New("conflict")
    ErrAlreadyExists      = errors.New("resource already exists")
    ErrConcurrentUpdate   = errors.New("concurrent update detected")

    // Infrastructure errors
    ErrDatabaseError      = errors.New("database error")
    ErrEventStoreError    = errors.New("event store error")
    ErrEventBusError      = errors.New("event bus error")
)

// ValidationError - ошибка валидации с контекстом
type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// NewValidationError создает ValidationError
func NewValidationError(field, message string) error {
    return &ValidationError{Field: field, Message: message}
}

// AuthorizationError - ошибка авторизации
type AuthorizationError struct {
    UserID   string
    Resource string
    Action   string
}

func (e AuthorizationError) Error() string {
    return fmt.Sprintf("user %s is not authorized to %s on %s", e.UserID, e.Action, e.Resource)
}

// NotFoundError - ошибка "не найдено"
type NotFoundError struct {
    Resource string
    ID       string
}

func (e NotFoundError) Error() string {
    return fmt.Sprintf("%s with ID %s not found", e.Resource, e.ID)
}
```

### 4. Context Utilities

#### File: `internal/application/shared/context.go`

```go
package shared

import (
    "context"
    "errors"

    "github.com/google/uuid"
)

// Context keys
type contextKey string

const (
    userIDKey        contextKey = "userID"
    workspaceIDKey   contextKey = "workspaceID"
    correlationIDKey contextKey = "correlationID"
    traceIDKey       contextKey = "traceID"
)

var (
    ErrUserIDNotFound        = errors.New("user ID not found in context")
    ErrWorkspaceIDNotFound   = errors.New("workspace ID not found in context")
    ErrCorrelationIDNotFound = errors.New("correlation ID not found in context")
)

// GetUserID извлекает ID пользователя из контекста
func GetUserID(ctx context.Context) (uuid.UUID, error) {
    userID, ok := ctx.Value(userIDKey).(uuid.UUID)
    if !ok {
        return uuid.Nil, ErrUserIDNotFound
    }
    return userID, nil
}

// WithUserID добавляет ID пользователя в контекст
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
    return context.WithValue(ctx, userIDKey, userID)
}

// GetWorkspaceID извлекает ID workspace из контекста
func GetWorkspaceID(ctx context.Context) (uuid.UUID, error) {
    workspaceID, ok := ctx.Value(workspaceIDKey).(uuid.UUID)
    if !ok {
        return uuid.Nil, ErrWorkspaceIDNotFound
    }
    return workspaceID, nil
}

// WithWorkspaceID добавляет ID workspace в контекст
func WithWorkspaceID(ctx context.Context, workspaceID uuid.UUID) context.Context {
    return context.WithValue(ctx, workspaceIDKey, workspaceID)
}

// GetCorrelationID извлекает correlation ID из контекста
func GetCorrelationID(ctx context.Context) (string, error) {
    correlationID, ok := ctx.Value(correlationIDKey).(string)
    if !ok {
        return "", ErrCorrelationIDNotFound
    }
    return correlationID, nil
}

// WithCorrelationID добавляет correlation ID в контекст
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
    return context.WithValue(ctx, correlationIDKey, correlationID)
}

// GetTraceID извлекает trace ID из контекста (для distributed tracing)
func GetTraceID(ctx context.Context) string {
    traceID, ok := ctx.Value(traceIDKey).(string)
    if !ok {
        return ""
    }
    return traceID
}

// WithTraceID добавляет trace ID в контекст
func WithTraceID(ctx context.Context, traceID string) context.Context {
    return context.WithValue(ctx, traceIDKey, traceID)
}
```

### 5. Validation Utilities

#### File: `internal/application/shared/validation.go`

```go
package shared

import (
    "fmt"
    "time"

    "github.com/google/uuid"
)

// ValidateRequired проверяет, что строка не пустая
func ValidateRequired(field, value string) error {
    if value == "" {
        return NewValidationError(field, "is required")
    }
    return nil
}

// ValidateUUID проверяет, что UUID валиден и не nil
func ValidateUUID(field string, id uuid.UUID) error {
    if id == uuid.Nil {
        return NewValidationError(field, "must be a valid UUID")
    }
    return nil
}

// ValidateMaxLength проверяет максимальную длину строки
func ValidateMaxLength(field, value string, maxLength int) error {
    if len(value) > maxLength {
        return NewValidationError(field, fmt.Sprintf("must be at most %d characters", maxLength))
    }
    return nil
}

// ValidateMinLength проверяет минимальную длину строки
func ValidateMinLength(field, value string, minLength int) error {
    if len(value) < minLength {
        return NewValidationError(field, fmt.Sprintf("must be at least %d characters", minLength))
    }
    return nil
}

// ValidateEnum проверяет, что значение находится в списке допустимых
func ValidateEnum(field, value string, allowedValues []string) error {
    for _, allowed := range allowedValues {
        if value == allowed {
            return nil
        }
    }
    return NewValidationError(field, fmt.Sprintf("must be one of: %v", allowedValues))
}

// ValidateDateNotPast проверяет, что дата не в прошлом
func ValidateDateNotPast(field string, date *time.Time) error {
    if date != nil && date.Before(time.Now()) {
        return NewValidationError(field, "cannot be in the past")
    }
    return nil
}

// ValidateDateRange проверяет, что дата находится в допустимом диапазоне
func ValidateDateRange(field string, date *time.Time, min, max time.Time) error {
    if date == nil {
        return nil
    }
    if date.Before(min) || date.After(max) {
        return NewValidationError(field, fmt.Sprintf("must be between %s and %s", min, max))
    }
    return nil
}
```

## Паттерн UseCase

### Базовая структура

```go
// Пример: CreateChatUseCase
package chat

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "github.com/lllypuk/flowra/internal/application/shared"
    "github.com/lllypuk/flowra/internal/domain/chat"
    "github.com/lllypuk/flowra/internal/domain/event"
)

// CreateChatUseCase обрабатывает создание нового чата
type CreateChatUseCase struct {
    chatRepo chat.Repository
    eventBus event.Bus
}

// NewCreateChatUseCase создает новый UseCase
func NewCreateChatUseCase(
    chatRepo chat.Repository,
    eventBus event.Bus,
) *CreateChatUseCase {
    return &CreateChatUseCase{
        chatRepo: chatRepo,
        eventBus: eventBus,
    }
}

// Execute выполняет UseCase
func (uc *CreateChatUseCase) Execute(
    ctx context.Context,
    cmd CreateChatCommand,
) (shared.EventSourcedResult[*chat.Chat], error) {
    // 1. Валидация
    if err := uc.validate(cmd); err != nil {
        return shared.EventSourcedResult[*chat.Chat]{},
            fmt.Errorf("validation failed: %w", err)
    }

    // 2. Авторизация (опционально)
    if err := uc.authorize(ctx, cmd); err != nil {
        return shared.EventSourcedResult[*chat.Chat]{}, err
    }

    // 3. Создание агрегата
    chatAggregate := chat.NewChat(
        cmd.WorkspaceID,
        cmd.Title,
        cmd.Type,
        cmd.CreatedBy,
    )

    // 4. Добавление создателя как участника
    if err := chatAggregate.AddParticipant(cmd.CreatedBy, chat.RoleAdmin); err != nil {
        return shared.EventSourcedResult[*chat.Chat]{},
            fmt.Errorf("failed to add creator as participant: %w", err)
    }

    // 5. Сохранение
    if err := uc.chatRepo.Save(ctx, chatAggregate); err != nil {
        return shared.EventSourcedResult[*chat.Chat]{},
            fmt.Errorf("failed to save chat: %w", err)
    }

    // 6. Публикация событий
    events := chatAggregate.GetUncommittedEvents()
    for _, evt := range events {
        if err := uc.eventBus.Publish(ctx, evt); err != nil {
            return shared.EventSourcedResult[*chat.Chat]{},
                fmt.Errorf("failed to publish event: %w", err)
    }
    }

    chatAggregate.MarkEventsAsCommitted()

    // 7. Возврат результата
    return shared.EventSourcedResult[*chat.Chat]{
        Result: shared.Result[*chat.Chat]{
            Value:   chatAggregate,
            Version: chatAggregate.Version(),
        },
        Events: events,
    }, nil
}

func (uc *CreateChatUseCase) validate(cmd CreateChatCommand) error {
    if err := shared.ValidateUUID("workspaceID", cmd.WorkspaceID); err != nil {
        return err
    }
    if err := shared.ValidateRequired("title", cmd.Title); err != nil {
        return err
    }
    if err := shared.ValidateMaxLength("title", cmd.Title, 200); err != nil {
        return err
    }
    if err := shared.ValidateEnum("type", string(cmd.Type), []string{
        string(chat.TypeDiscussion),
        string(chat.TypeTask),
        string(chat.TypeBug),
        string(chat.TypeEpic),
    }); err != nil {
        return err
    }
    if err := shared.ValidateUUID("createdBy", cmd.CreatedBy); err != nil {
        return err
    }
    return nil
}

func (uc *CreateChatUseCase) authorize(ctx context.Context, cmd CreateChatCommand) error {
    // Проверка, что пользователь имеет доступ к workspace
    userID, err := shared.GetUserID(ctx)
    if err != nil {
        return shared.ErrUnauthorized
    }

    if userID != cmd.CreatedBy {
        return shared.ErrForbidden
    }

    // TODO: Проверить membership в workspace

    return nil
}
```

## Testing Strategy

### Unit Tests

```go
func TestCreateChatUseCase_Success(t *testing.T) {
    // Arrange
    chatRepo := mocks.NewChatRepository()
    eventBus := mocks.NewEventBus()
    useCase := NewCreateChatUseCase(chatRepo, eventBus)

    cmd := CreateChatCommand{
        WorkspaceID: uuid.New(),
        Title:       "Test Chat",
        Type:        chat.TypeDiscussion,
        CreatedBy:   uuid.New(),
    }

    ctx := shared.WithUserID(context.Background(), cmd.CreatedBy)

    // Act
    result, err := useCase.Execute(ctx, cmd)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result.Value)
    assert.Equal(t, cmd.Title, result.Value.Title())
    assert.Len(t, result.Events, 2) // ChatCreated, ParticipantAdded

    // Verify repository was called
    assert.Equal(t, 1, chatRepo.SaveCallCount())

    // Verify events were published
    assert.Equal(t, 2, eventBus.PublishCallCount())
}
```

## Checklist

- [ ] Создать `internal/application/shared/interfaces.go`
- [ ] Создать `internal/application/shared/errors.go`
- [ ] Создать `internal/application/shared/context.go`
- [ ] Создать `internal/application/shared/validation.go`
- [ ] Создать структуру директорий для доменов
- [ ] Документировать паттерны в README
- [ ] Создать примеры UseCases
- [ ] Написать unit tests для shared компонентов

## Следующие шаги

После завершения архитектуры:
- **Task 02**: Chat UseCases (используя shared компоненты)
- **Task 03**: Message UseCases
- **Task 04**: User UseCases
- Остальные домены по очереди

## Референсы

- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [CQRS Pattern](https://martinfowler.com/bliki/CQRS.html)
- [Domain-Driven Design](https://martinfowler.com/tags/domain%20driven%20design.html)
