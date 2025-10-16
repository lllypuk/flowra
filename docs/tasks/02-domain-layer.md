# Task 02: Domain Layer — Core Aggregates (Phase 1)

**Фаза:** 1 - Domain Layer
**Приоритет:** Critical
**Статус:** ✅ **COMPLETED**
**Дата создания:** 2025-10-04
**Дата завершения:** 2025-10-16
**Предыдущая задача:** [01-init-project.md](./01-init-project.md) ✅

## Цель

Реализовать доменную модель без зависимости от инфраструктуры. Чистая бизнес-логика с интерфейсами репозиториев. Все компоненты используют **интерфейсы** для слабой связанности и тестируемости.

**Принцип:** Domain-first approach — начинаем с доменной модели, независимой от инфраструктуры.

---

## Подзадачи

### 1.1 Base Domain Infrastructure

**Описание:** Создать базовую инфраструктуру для domain events и общих value objects.

#### 1.1.1 Domain Events Infrastructure

**Файлы:**
- `internal/domain/event/event.go`
- `internal/domain/event/metadata.go`
- `internal/domain/event/base_event.go`

**Реализация:**

1. Создать `internal/domain/event/event.go`:
   ```go
   package event

   import "time"

   // DomainEvent представляет доменное событие
   type DomainEvent interface {
       // EventType возвращает тип события
       EventType() string

       // AggregateID возвращает ID агрегата
       AggregateID() string

       // AggregateType возвращает тип агрегата
       AggregateType() string

       // OccurredAt возвращает время возникновения события
       OccurredAt() time.Time

       // Version возвращает версию агрегата
       Version() int

       // Metadata возвращает метаданные события
       Metadata() EventMetadata
   }
   ```

2. Создать `internal/domain/event/metadata.go`:
   ```go
   package event

   import "time"

   // EventMetadata содержит метаданные события
   type EventMetadata struct {
       UserID        string
       CorrelationID string
       CausationID   string
       Timestamp     time.Time
       IPAddress     string
       UserAgent     string
   }

   // NewMetadata создает новые метаданные
   func NewMetadata(userID, correlationID, causationID string) EventMetadata {
       return EventMetadata{
           UserID:        userID,
           CorrelationID: correlationID,
           CausationID:   causationID,
           Timestamp:     time.Now(),
       }
   }
   ```

3. Создать `internal/domain/event/base_event.go`:
   ```go
   package event

   import (
       "time"

       "github.com/google/uuid"
   )

   // BaseEvent базовая реализация DomainEvent
   type BaseEvent struct {
       eventType     string
       aggregateID   string
       aggregateType string
       occurredAt    time.Time
       version       int
       metadata      EventMetadata
   }

   // NewBaseEvent создает новое базовое событие
   func NewBaseEvent(eventType, aggregateID, aggregateType string, version int, metadata EventMetadata) BaseEvent {
       return BaseEvent{
           eventType:     eventType,
           aggregateID:   aggregateID,
           aggregateType: aggregateType,
           occurredAt:    time.Now(),
           version:       version,
           metadata:      metadata,
       }
   }

   func (e BaseEvent) EventType() string          { return e.eventType }
   func (e BaseEvent) AggregateID() string        { return e.aggregateID }
   func (e BaseEvent) AggregateType() string      { return e.aggregateType }
   func (e BaseEvent) OccurredAt() time.Time      { return e.occurredAt }
   func (e BaseEvent) Version() int               { return e.version }
   func (e BaseEvent) Metadata() EventMetadata    { return e.metadata }
   ```

**Тесты:**
- `internal/domain/event/event_test.go` - unit tests для event infrastructure

**Критерии выполнения:**
- [x] DomainEvent interface определен
- [x] EventMetadata struct реализован (переименован в Metadata)
- [x] BaseEvent реализует DomainEvent
- [x] Unit tests покрывают функциональность (100% coverage)

---

#### 1.1.2 Common Value Objects

**Файлы:**
- `internal/domain/common/uuid.go`
- `internal/domain/common/errors.go`

**Реализация:**

1. Создать `internal/domain/common/uuid.go`:
   ```go
   package common

   import (
       "github.com/google/uuid"
   )

   // UUID type alias для UUID
   type UUID string

   // NewUUID создает новый UUID
   func NewUUID() UUID {
       return UUID(uuid.New().String())
   }

   // ParseUUID парсит строку в UUID
   func ParseUUID(s string) (UUID, error) {
       _, err := uuid.Parse(s)
       if err != nil {
           return "", err
       }
       return UUID(s), nil
   }

   // String возвращает строковое представление
   func (u UUID) String() string {
       return string(u)
   }

   // IsZero проверяет, является ли UUID нулевым
   func (u UUID) IsZero() bool {
       return u == ""
   }
   ```

2. Создать `internal/domain/common/errors.go`:
   ```go
   package common

   import "errors"

   var (
       // ErrNotFound возвращается, когда ресурс не найден
       ErrNotFound = errors.New("resource not found")

       // ErrAlreadyExists возвращается, когда ресурс уже существует
       ErrAlreadyExists = errors.New("resource already exists")

       // ErrInvalidInput возвращается при невалидных входных данных
       ErrInvalidInput = errors.New("invalid input")

       // ErrUnauthorized возвращается при отсутствии прав доступа
       ErrUnauthorized = errors.New("unauthorized")

       // ErrForbidden возвращается при запрещенном действии
       ErrForbidden = errors.New("forbidden")

       // ErrConcurrentModification возвращается при конфликте версий
       ErrConcurrentModification = errors.New("concurrent modification detected")
   )
   ```

**Тесты:**
- `internal/domain/common/uuid_test.go`
- `internal/domain/common/errors_test.go`

**Критерии выполнения:**
- [x] UUID type alias создан с методами (internal/domain/uuid/)
- [x] Domain errors определены (internal/domain/errs/)
- [x] Unit tests покрывают UUID функциональность (100% coverage)

**Примечание:** Пакет `common` был разделен на `uuid` и `errs` для лучшей организации.

---

### 1.2 User Aggregate

**Описание:** Реализовать User aggregate root с бизнес-логикой и событиями.

#### 1.2.1 User Aggregate

**Файл:** `internal/domain/user/user.go`

**Реализация:**
```go
package user

import (
    "time"

    "github.com/lllypuk/teams-up/internal/domain/common"
)

// User представляет пользователя системы
type User struct {
    id            common.UUID
    username      string
    email         string
    displayName   string
    isSystemAdmin bool
    createdAt     time.Time
    updatedAt     time.Time
}

// NewUser создает нового пользователя
func NewUser(username, email, displayName string) (*User, error) {
    if username == "" {
        return nil, common.ErrInvalidInput
    }
    if email == "" {
        return nil, common.ErrInvalidInput
    }

    return &User{
        id:          common.NewUUID(),
        username:    username,
        email:       email,
        displayName: displayName,
        createdAt:   time.Now(),
        updatedAt:   time.Now(),
    }, nil
}

// Getters
func (u *User) ID() common.UUID       { return u.id }
func (u *User) Username() string      { return u.username }
func (u *User) Email() string         { return u.email }
func (u *User) DisplayName() string   { return u.displayName }
func (u *User) IsSystemAdmin() bool   { return u.isSystemAdmin }
func (u *User) CreatedAt() time.Time  { return u.createdAt }
func (u *User) UpdatedAt() time.Time  { return u.updatedAt }

// UpdateProfile обновляет профиль пользователя
func (u *User) UpdateProfile(displayName string) error {
    if displayName == "" {
        return common.ErrInvalidInput
    }
    u.displayName = displayName
    u.updatedAt = time.Now()
    return nil
}

// SetAdmin устанавливает права администратора
func (u *User) SetAdmin(isAdmin bool) {
    u.isSystemAdmin = isAdmin
    u.updatedAt = time.Now()
}
```

**Критерии выполнения:**
- [x] User aggregate создан с полями
- [x] NewUser конструктор с валидацией
- [x] UpdateProfile() метод реализован
- [x] SetAdmin() метод реализован

---

#### 1.2.2 User Repository Interface

**Файл:** `internal/domain/user/repository.go`

**Реализация:**
```go
package user

import (
    "context"

    "github.com/lllypuk/teams-up/internal/domain/common"
)

// Repository определяет интерфейс репозитория пользователей
type Repository interface {
    // FindByID находит пользователя по ID
    FindByID(ctx context.Context, id common.UUID) (*User, error)

    // FindByEmail находит пользователя по email
    FindByEmail(ctx context.Context, email string) (*User, error)

    // FindByUsername находит пользователя по username
    FindByUsername(ctx context.Context, username string) (*User, error)

    // Save сохраняет пользователя
    Save(ctx context.Context, user *User) error

    // Delete удаляет пользователя
    Delete(ctx context.Context, id common.UUID) error
}
```

**Критерии выполнения:**
- [x] Repository interface определен
- [x] Методы FindByID, FindByEmail, FindByUsername
- [x] Методы Save и Delete

---

#### 1.2.3 User Domain Events

**Файл:** `internal/domain/user/events.go`

**Реализация:**
```go
package user

import (
    "github.com/lllypuk/teams-up/internal/domain/common"
    "github.com/lllypuk/teams-up/internal/domain/event"
)

const (
    EventTypeUserCreated = "user.created"
    EventTypeUserUpdated = "user.updated"
)

// UserCreated событие создания пользователя
type UserCreated struct {
    event.BaseEvent
    Username    string
    Email       string
    DisplayName string
}

// NewUserCreated создает событие UserCreated
func NewUserCreated(userID common.UUID, username, email, displayName string, metadata event.EventMetadata) *UserCreated {
    return &UserCreated{
        BaseEvent:   event.NewBaseEvent(EventTypeUserCreated, userID.String(), "User", 1, metadata),
        Username:    username,
        Email:       email,
        DisplayName: displayName,
    }
}

// UserUpdated событие обновления пользователя
type UserUpdated struct {
    event.BaseEvent
    DisplayName string
}

// NewUserUpdated создает событие UserUpdated
func NewUserUpdated(userID common.UUID, displayName string, version int, metadata event.EventMetadata) *UserUpdated {
    return &UserUpdated{
        BaseEvent:   event.NewBaseEvent(EventTypeUserUpdated, userID.String(), "User", version, metadata),
        DisplayName: displayName,
    }
}
```

**Критерии выполнения:**
- [x] UserCreated event определен
- [x] UserUpdated event определен (+ AdminRightsChanged, UserDeleted)
- [x] Конструкторы создают события с метаданными

---

#### 1.2.4 User Unit Tests

**Файл:** `internal/domain/user/user_test.go`

**Тесты:**
- NewUser создание с валидацией
- UpdateProfile обновление профиля
- SetAdmin установка прав
- Edge cases (пустые значения, nil)

**Критерии выполнения:**
- [x] Тесты для NewUser()
- [x] Тесты для UpdateProfile()
- [x] Тесты для SetAdmin()
- [x] Coverage > 80% (83.3% достигнуто)

---

### 1.3 Workspace Aggregate ✅

**Подзадачи:**
- [x] 1.3.1 Workspace aggregate (с Keycloak integration)
- [x] 1.3.2 Invite entity (с expiration, usage tracking, revocation)
- [x] 1.3.3 Workspace repository interface
- [x] 1.3.4 Workspace events (6 событий)
- [x] 1.3.5 Workspace unit tests (88.5% coverage, 32 теста)

---

### 1.4 Chat Aggregate ✅

**Описание:** Реализовать Chat aggregate с поддержкой Event Sourcing.

**Особенности:**
- Event Sourcing для восстановления состояния
- Методы Apply(), GetUncommittedEvents(), MarkEventsAsCommitted()
- Participant value object с ролями (admin/member)
- Chat type conversion (Discussion → Task/Bug/Epic)

**Подзадачи:**
- [x] 1.4.1 Chat aggregate root (с Event Sourcing)
- [x] 1.4.2 Message entity (упрощенная версия)
- [x] 1.4.3 Participant value object (с ролями)
- [x] 1.4.4 Chat repository interface (Event Store + Read Model)
- [x] 1.4.5 Chat domain events (4 события)
- [x] 1.4.6 Event sourcing support (Apply, GetUncommitted, MarkCommitted)
- [x] 1.4.7 Chat unit tests (96.8% coverage, 32 теста)

---

### 1.5 Task Aggregate ✅

**Подзадачи:**
- [x] 1.5.1 TaskEntity aggregate (переименован в Entity)
- [x] 1.5.2 EntityState value object (с типами Task/Bug/Epic/Discussion)
- [x] 1.5.3 Status validation (статус-машина с 6 статусами)
- [x] 1.5.4 Task repository interface (с GetBoard для канбана)
- [x] 1.5.5 Task domain events (8 событий)
- [x] 1.5.6 Task unit tests (88.6% coverage, 42 теста)

**Особенности реализации:**
- Полноценная статус-машина с валидацией переходов
- Priority system (Low/Medium/High/Critical)
- Due date tracking с IsOverdue()
- Custom fields для тегов (#sprint, #component, etc.)

---

### 1.6 Notification Aggregate ✅

**Подзадачи:**
- [x] 1.6.1 Notification aggregate (с 7 типами уведомлений)
- [x] 1.6.2 Notification repository interface (с unread tracking)
- [x] 1.6.3 Notification events (3 события)
- [x] 1.6.4 Notification unit tests (88.5% coverage, 23 теста)

---

## Deliverable

После выполнения всех подзадач должно быть готово:

✅ **Базовая Domain Infrastructure**
- DomainEvent interface и BaseEvent
- EventMetadata для трассировки
- Common value objects (UUID, errors)

✅ **5 Domain Aggregates с бизнес-логикой**
- User - управление пользователями
- Workspace - управление workspace и приглашениями
- Chat - чаты с event sourcing
- Task - задачи с валидацией статусов
- Notification - уведомления

✅ **Repository Interfaces**
- Все зависимости через интерфейсы
- Нет привязки к БД или фреймворкам

✅ **Domain Events**
- События для всех изменений состояния
- Метаданные для трассировки

✅ **Unit Tests**
- Coverage > 80% для domain layer
- Тесты изолированы от инфраструктуры
- TDD подход где возможно

---

## Порядок реализации

**Рекомендуемый порядок:**

1. **Сначала:** 1.1 Base Domain Infrastructure (event, common)
2. **Затем:** 1.2 User (самый простой aggregate)
3. **Потом:** 1.3 Workspace, 1.6 Notification (средняя сложность)
4. **Затем:** 1.5 Task (со статус-машиной)
5. **В конце:** 1.4 Chat (самый сложный, с Event Sourcing)

**Принцип:** От простого к сложному, тестируем каждый компонент перед переходом к следующему.

---

## Проверка выполнения

```bash
# 1. Проверка структуры
ls -la internal/domain/event/
ls -la internal/domain/common/
ls -la internal/domain/user/
ls -la internal/domain/workspace/
ls -la internal/domain/chat/
ls -la internal/domain/task/
ls -la internal/domain/notification/

# 2. Проверка компиляции
go build ./internal/domain/...

# 3. Запуск unit tests
go test ./internal/domain/... -v -cover

# 4. Проверка покрытия
go test ./internal/domain/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# 5. Линтинг
make lint
```

Все команды должны выполняться успешно, покрытие > 80%.

---

## Следующие шаги

После завершения Phase 1 переходим к **Phase 2: Application Layer — Use Cases**:
- Application services с бизнес-логикой
- Command/Query handlers
- Event handlers (subscribers)

См. `docs/08-mvp-roadmap.md` Phase 2 для деталей.

---

## Примечания

- **Никакой инфраструктуры** - только чистая бизнес-логика
- **Все через интерфейсы** - репозитории, event bus (пока не реализованы)
- **TDD подход** - пишем тесты вместе с кодом
- **Event Sourcing** только для Chat aggregate
- **Версии зависимостей** из Phase 0 (uuid v1.6.0)

**Важно:** Domain layer не зависит от application, infrastructure или interface layers. Направление зависимостей: наружу → внутрь (к domain).

---

## ✅ Итоговая статистика выполнения

### Реализованные компоненты:

**Domain Infrastructure:**
- ✅ Event system (DomainEvent, BaseEvent, Metadata) - 100% coverage
- ✅ UUID value object - 100% coverage
- ✅ Domain errors (errs package)

**Aggregates (5/5):**
1. ✅ **User** - 83.3% coverage, 24 теста
2. ✅ **Workspace** - 88.5% coverage, 32 теста
3. ✅ **Notification** - 88.5% coverage, 23 теста
4. ✅ **Task** - 88.6% coverage, 42 теста
5. ✅ **Chat** - 96.8% coverage, 32 теста (с Event Sourcing)

**Всего:**
- 📦 **32 файла** создано
- ✅ **161 unit test** (все проходят)
- 📊 **Средний coverage: 90.6%** (превышает требование 80%)
- 📝 **~3500 строк** production кода
- 🧪 **~2500 строк** test кода

### Коммиты:

```
7321197 Refactor domain packages: split common into errs and uuid
fab0a70 Implement Phase 1: Domain Infrastructure & User Aggregate
b580b2e Implement Phase 1.3: Workspace Aggregate
2930fe2 Implement Phase 1.6: Notification Aggregate
57dbb60 Implement Phase 1.5: Task Aggregate with Status Machine
f6309bf Implement Phase 1.4: Chat Aggregate with Event Sourcing
```

### Ключевые достижения:

- ✅ Полная изоляция от инфраструктуры
- ✅ Все зависимости через интерфейсы
- ✅ Event-driven design (23 domain events)
- ✅ Event Sourcing для Chat
- ✅ Status state machine для Task (10+ tested transitions)
- ✅ CQRS-ready (Read Models)
- ✅ Нулевые критичные ошибки линтера

**Статус:** Phase 1 полностью завершена ✅
**Готовность к Phase 2:** 100%
