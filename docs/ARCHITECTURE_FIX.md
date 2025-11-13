# Архитектурное исправление: Перемещение интерфейсов репозиториев

## Дата: 2025-10-21

## Проблема

В проекте была обнаружена серьезная архитектурная проблема, нарушающая принципы идиоматичного Go и правила, описанные в `CLAUDE.md`:

### Что было неправильно:

1. **Интерфейсы репозиториев были объявлены в domain слое**
   ```
   ❌ internal/domain/chat/repository.go
   ❌ internal/domain/message/repository.go
   ❌ internal/domain/user/repository.go
   ❌ internal/domain/workspace/repository.go
   ❌ internal/domain/notification/repository.go
   ```

2. **Infrastructure слой импортировал domain интерфейсы**
   - Создавалось неправильное направление зависимостей
   - Нарушался Dependency Inversion Principle

3. **Смешивались команды и запросы**
   - В одном интерфейсе были и методы изменения состояния (Save, Delete)
   - И методы чтения (FindByID, List)
   - Нарушался CQRS pattern

### Почему это плохо:

- **Нарушение Dependency Inversion Principle**: Domain слой НЕ должен знать об интерфейсах репозиториев
- **Против идиоматичного Go**: Интерфейсы должны объявляться там, где используются (consumer side), а не где реализуются (producer side)
- **Проблема зависимостей**: Infrastructure зависел от domain интерфейсов вместо того, чтобы зависеть от application
- **Затрудненное тестирование**: Моки создавались на основе domain интерфейсов

## Решение

### 1. Перемещение интерфейсов в application слой

Интерфейсы репозиториев были перемещены из `internal/domain/*/repository.go` в `internal/application/*/repository.go`:

```
✅ internal/application/chat/repository.go
✅ internal/application/message/repository.go
✅ internal/application/user/repository.go
✅ internal/application/workspace/repository.go
✅ internal/application/notification/repository.go
```

### 2. Разделение по CQRS pattern

Каждый репозиторий теперь имеет три интерфейса:

```go
// CommandRepository - для команд (изменение состояния)
type CommandRepository interface {
    Save(ctx context.Context, entity *domain.Entity) error
    Delete(ctx context.Context, id uuid.UUID) error
}

// QueryRepository - для запросов (только чтение)
type QueryRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*domain.Entity, error)
    List(ctx context.Context, offset, limit int) ([]*domain.Entity, error)
}

// Repository - объединяет оба для удобства
type Repository interface {
    CommandRepository
    QueryRepository
}
```

### 3. Перемещение DTO типов

Типы, которые использовались только для query операций, также перемещены в application слой:

- `Pagination` (message) - параметры пагинации для сообщений
- `ReadModel` (chat) - материализованное представление чата для быстрых запросов
- `Filters` (chat) - фильтры для поиска чатов

### 4. Обновление infrastructure слоя

MongoDB репозитории обновлены для реализации интерфейсов из application слоя:

```go
// internal/infrastructure/repository/mongodb/message_repository.go
import messageapp "github.com/lllypuk/flowra/internal/application/message"

// MongoMessageRepository реализует messageapp.Repository (application layer interface)
type MongoMessageRepository struct { ... }
```

### 5. Обновление use cases

Use cases теперь используют интерфейсы из того же пакета (application layer):

```go
package message

type SendMessageUseCase struct {
    messageRepo Repository  // Из того же пакета application/message
    chatRepo    ChatRepository
}
```

## Правильное направление зависимостей

### До исправления:
```
Domain Layer (repository interfaces)
    ↑
    └── Infrastructure Layer (implementations)
    ↑
    └── Application Layer (use cases)
```

### После исправления:
```
Application Layer (repository interfaces + use cases)
    ↑
    └── Infrastructure Layer (implementations)

Domain Layer (business logic only, no repository knowledge)
    ↑
    └── Application Layer
```

## Преимущества нового подхода

1. **Идиоматичный Go**: Следует принципу "accept interfaces, return structs"
2. **Правильная инверсия зависимостей**: Application определяет что ему нужно, infrastructure реализует
3. **CQRS**: Четкое разделение команд и запросов
4. **Тестируемость**: Легко создавать моки на стороне потребителя
5. **Гибкость**: Можно менять реализацию (MongoDB → PostgreSQL) без изменения application слоя
6. **Ясность владения**: Интерфейсы меняются только когда меняются требования потребителя

## Примеры правильного использования

### Объявление интерфейса (consumer side):
```go
// internal/application/message/repository.go
package message

// QueryRepository определяет интерфейс для запросов (только чтение) сообщений
// Интерфейс объявлен на стороне потребителя (application layer)
type QueryRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*domain.Message, error)
    FindByChatID(ctx context.Context, chatID uuid.UUID, pagination Pagination) ([]*domain.Message, error)
}
```

### Реализация интерфейса (producer side):
```go
// internal/infrastructure/repository/mongodb/message_repository.go
package mongodb

import messageapp "github.com/lllypuk/flowra/internal/application/message"

// MongoMessageRepository реализует messageapp.QueryRepository
type MongoMessageRepository struct { ... }

func (r *MongoMessageRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Message, error) {
    // implementation
}
```

### Использование в use case:
```go
// internal/application/message/send_message.go
package message

type SendMessageUseCase struct {
    messageRepo Repository // Интерфейс из того же пакета
}

func NewSendMessageUseCase(repo Repository) *SendMessageUseCase {
    return &SendMessageUseCase{messageRepo: repo}
}
```

## Сравнение с правильным примером

В проекте уже был правильный пример - `EventStore`:

```go
// internal/application/shared/eventstore.go
// EventStore определяет интерфейс для сохранения и загрузки событий
// Интерфейс объявлен здесь (на стороне потребителя - application layer),
// а не в infrastructure, следуя идиоматичному Go подходу.
type EventStore interface {
    SaveEvents(ctx context.Context, aggregateID string, events []event.DomainEvent, expectedVersion int) error
    LoadEvents(ctx context.Context, aggregateID string) ([]event.DomainEvent, error)
}
```

Теперь все репозитории следуют тому же паттерну.

## Что осталось сделать

1. Обновить тестовые файлы (они были пропущены для экономии времени)
2. Обновить моки в тестах для использования новых интерфейсов
3. Добавить документацию по созданию новых репозиториев

## Ссылки

- `CLAUDE.md` - правила проекта, описывающие consumer-side interface declaration
- Go Proverbs: "Accept interfaces, return concrete types"
- Dependency Inversion Principle (SOLID)
- CQRS Pattern
