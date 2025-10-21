# Quick Reference - Critical Tasks

> **TL;DR:** Нужно завершить 2 критические задачи за 5-6 часов, чтобы UseCase layer был полностью готов.

---

## 🎯 Что делать СЕЙЧАС

```bash
# 1️⃣ КРИТИЧНО: Добавить тесты для Chat UseCases
cd /home/sasha/Projects/new-teams-up
cat docs/tasks/04-impl-usecase/09-chat-tests.md

# Создать 12 файлов с тестами в internal/application/chat/
# - create_chat_test.go
# - add_participant_test.go
# - remove_participant_test.go
# - convert_to_task_test.go
# - convert_to_bug_test.go
# - convert_to_epic_test.go
# - change_status_test.go
# - assign_user_test.go
# - set_priority_test.go
# - set_due_date_test.go
# - rename_chat_test.go
# - set_severity_test.go

# Время: 3-4 часа
# Результат: Coverage 0% → >85%
```

```bash
# 2️⃣ ВЫСОКИЙ ПРИОРИТЕТ: Реализовать Query UseCases
cat docs/tasks/04-impl-usecase/10-chat-queries.md

# Создать 3 файла в internal/application/chat/
# - get_chat.go + get_chat_test.go
# - list_chats.go + list_chats_test.go
# - list_participants.go + list_participants_test.go

# Также создать/обновить:
# - queries.go (новые Query типы)
# - results.go (новые Result типы)

# Время: 1-2 часа
# Результат: Phase 2 полностью завершена
```

---

## 📂 Файлы для создания

### Task 09: Chat Tests (12 файлов)

```
internal/application/chat/
├── test_setup.go              ← NEW (вспомогательные функции)
├── create_chat_test.go        ← NEW (8 тестов)
├── add_participant_test.go    ← NEW (7 тестов)
├── remove_participant_test.go ← NEW (5 тестов)
├── convert_to_task_test.go    ← NEW (5 тестов)
├── convert_to_bug_test.go     ← NEW (4 теста)
├── convert_to_epic_test.go    ← NEW (3 теста)
├── change_status_test.go      ← NEW (6 тестов)
├── assign_user_test.go        ← NEW (4 теста)
├── set_priority_test.go       ← NEW (6 тестов)
├── set_due_date_test.go       ← NEW (5 тестов)
├── rename_chat_test.go        ← NEW (4 теста)
└── set_severity_test.go       ← NEW (6 тестов)

Всего: ~60 unit тестов
```

### Task 10: Query UseCases (7 файлов)

```
internal/application/chat/
├── queries.go                    ← NEW (Query типы)
├── results.go                    ← UPDATE (добавить новые Result типы)
├── get_chat.go                   ← NEW (GetChatUseCase)
├── get_chat_test.go              ← NEW (4 теста)
├── list_chats.go                 ← NEW (ListChatsUseCase)
├── list_chats_test.go            ← NEW (6 тестов)
├── list_participants.go          ← NEW (ListParticipantsUseCase)
└── list_participants_test.go     ← NEW (5 тестов)

Всего: 3 UseCases + 15 тестов
```

---

## 🏃 Быстрый старт

### Шаг 1: Подготовка (5 минут)

```bash
cd /home/sasha/Projects/new-teams-up

# Открыть детальные планы
cat docs/tasks/04-impl-usecase/09-chat-tests.md
cat docs/tasks/04-impl-usecase/10-chat-queries.md

# Проверить текущий coverage
go test -coverprofile=/tmp/coverage.out ./internal/application/...
go tool cover -func=/tmp/coverage.out | grep "chat"
# Ожидаемо: 0.0%
```

### Шаг 2: Task 09 - Тесты (3.5 часа)

```bash
cd internal/application/chat

# Создать test_setup.go (пример в 09-chat-tests.md)
vim test_setup.go

# Создать тесты по одному файлу
# Следовать шаблонам из 09-chat-tests.md

# Начать с CreateChatUseCase (самый важный)
vim create_chat_test.go
# Скопировать пример из 09-chat-tests.md
# Адаптировать под реальную реализацию

# Запускать тесты после каждого файла
go test -v -run TestCreateChat
go test -v -run TestAddParticipant
# и т.д.

# Финальная проверка
go test -v ./...
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -1
# Цель: >85%
```

### Шаг 3: Task 10 - Query UseCases (2 часа)

```bash
cd internal/application/chat

# 1. Создать queries.go
vim queries.go
# Код из 10-chat-queries.md

# 2. Обновить results.go
vim results.go
# Добавить ListQueryResult, ParticipantsQueryResult

# 3. Реализовать GetChatUseCase
vim get_chat.go
vim get_chat_test.go
go test -v -run TestGetChat

# 4. Реализовать ListChatsUseCase
vim list_chats.go
vim list_chats_test.go
go test -v -run TestListChats

# 5. Реализовать ListParticipantsUseCase
vim list_participants.go
vim list_participants_test.go
go test -v -run TestListParticipants

# Финальная проверка
go test -v -run Query ./...
```

### Шаг 4: Проверка (10 минут)

```bash
# Все тесты
go test ./internal/application/chat/... -v

# Coverage
go test -coverprofile=/tmp/chat_coverage.out ./internal/application/chat/...
go tool cover -html=/tmp/chat_coverage.out
# Должно быть >85%

# Общий coverage application layer
go test -coverprofile=/tmp/coverage.out ./internal/application/...
go tool cover -func=/tmp/coverage.out | grep total
# Ожидаемо: ~75-80% (было 64.7%)

# Линтер
golangci-lint run ./internal/application/chat/...
```

### Шаг 5: Обновить трекер (5 минут)

```bash
vim docs/tasks/04-impl-usecase/PROGRESS_TRACKER.md

# Обновить Phase 2:
# - Chat UseCases: coverage 0% → >85%
# - Query UseCases: реализованы
# - Status: 🟡 In Progress → 🟢 Complete

# Обновить Overall Progress:
# - Phase 2: [██████░░░░] → [██████████]
# - Overall: 82% → ~95%
```

---

## 📋 Шаблоны кода

### Шаблон теста (скопировать и адаптировать)

```go
package chat_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/lllypuk/flowra/internal/application/chat"
    domainChat "github.com/lllypuk/flowra/internal/domain/chat"
    domainUUID "github.com/lllypuk/flowra/internal/domain/uuid"
    "github.com/lllypuk/flowra/tests/mocks"
)

func TestXxxUseCase_Success(t *testing.T) {
    // Arrange
    eventStore := mocks.NewEventStore()
    useCase := chat.NewXxxUseCase(eventStore)

    cmd := chat.XxxCommand{
        // ... параметры
    }

    // Act
    result, err := useCase.Execute(context.Background(), cmd)

    // Assert
    require.NoError(t, err)
    assert.NotNil(t, result.Aggregate)
    // ... проверки
}

func TestXxxUseCase_ValidationError(t *testing.T) {
    // Arrange
    eventStore := mocks.NewEventStore()
    useCase := chat.NewXxxUseCase(eventStore)

    cmd := chat.XxxCommand{
        // ... невалидные параметры
    }

    // Act
    result, err := useCase.Execute(context.Background(), cmd)

    // Assert
    require.Error(t, err)
    assert.Contains(t, err.Error(), "validation failed")
}
```

### Шаблон Query UseCase

```go
package chat

import (
    "context"
    "fmt"

    "github.com/lllypuk/flowra/internal/application/shared"
    "github.com/lllypuk/flowra/internal/domain/chat"
)

type XxxUseCase struct {
    eventStore shared.EventStore
}

func NewXxxUseCase(eventStore shared.EventStore) *XxxUseCase {
    return &XxxUseCase{eventStore: eventStore}
}

func (uc *XxxUseCase) Execute(ctx context.Context, query XxxQuery) (QueryResult, error) {
    // Валидация
    if err := uc.validate(query); err != nil {
        return QueryResult{}, fmt.Errorf("validation failed: %w", err)
    }

    // Загрузка из EventStore
    events, err := uc.eventStore.LoadEvents(ctx, query.ChatID.String())
    if err != nil {
        return QueryResult{}, fmt.Errorf("failed to load events: %w", err)
    }

    if len(events) == 0 {
        return QueryResult{}, ErrChatNotFound
    }

    // Восстановление агрегата
    chatAggregate := &chat.Chat{}
    if err := chatAggregate.LoadFromHistory(events); err != nil {
        return QueryResult{}, fmt.Errorf("failed to load from history: %w", err)
    }

    return QueryResult{
        Aggregate: chatAggregate,
        Version:   chatAggregate.Version(),
    }, nil
}

func (uc *XxxUseCase) validate(query XxxQuery) error {
    // Валидация
    return nil
}
```

---

## ✅ Критерии завершения

### Task 09: Chat Tests ✅

- [x] 12+ test файлов созданы
- [x] 60+ unit тестов написаны
- [x] Все тесты проходят: `go test ./internal/application/chat/... -v`
- [x] Coverage >85%: `go test -coverprofile=coverage.out ./internal/application/chat/...`
- [x] Нет ошибок линтера: `golangci-lint run ./internal/application/chat/...`

### Task 10: Query UseCases ✅

- [x] queries.go создан
- [x] results.go обновлён
- [x] GetChatUseCase реализован с тестами
- [x] ListChatsUseCase реализован с тестами
- [x] ListParticipantsUseCase реализован с тестами
- [x] Все Query тесты проходят
- [x] Coverage >85%

### Overall ✅

- [x] Application layer coverage: >75%
- [x] Phase 2 полностью завершена
- [x] PROGRESS_TRACKER.md обновлён
- [x] Готово к переходу на infrastructure layer

---

## 🆘 Troubleshooting

### "Не знаю, как писать тесты"

→ Смотри примеры в `internal/application/message/*_test.go`
→ Копируй шаблоны из `09-chat-tests.md`
→ Используй готовые mocks из `tests/mocks/`

### "EventStore mock не работает"

→ Проверь `tests/mocks/eventstore.go`
→ Используй методы: `SetLoadEventsResult()`, `SetSaveEventsError()`
→ Смотри примеры в message tests

### "Coverage не растёт"

→ Проверь, что тесты вызывают все ветки кода
→ Добавь тесты для error cases
→ Проверь validation errors
→ Используй: `go test -coverprofile=coverage.out && go tool cover -html=coverage.out`

### "Тесты долго выполняются"

→ Используй `t.Parallel()` для параллельных тестов
→ Не используй реальную БД, только mocks
→ Цель: <5 секунд для всех Chat tests

---

## 📚 Полезные ссылки

**Детальные планы:**
- [09-chat-tests.md](./09-chat-tests.md) - полный план тестирования
- [10-chat-queries.md](./10-chat-queries.md) - полный план Query UseCases
- [COMPLETION_PLAN.md](./COMPLETION_PLAN.md) - общая стратегия
- [PRIORITIES.md](./PRIORITIES.md) - приоритеты задач

**Примеры кода:**
- `internal/application/message/` - reference implementation
- `tests/mocks/` - готовые mocks
- `tests/fixtures/` - test fixtures
- `tests/testutil/` - test utilities

**Документация:**
- [PROGRESS_TRACKER.md](./PROGRESS_TRACKER.md) - текущий прогресс
- [README.md](./README.md) - общая информация

---

## 🎯 Focus

**Единственная цель:** Завершить Task 09 и Task 10

**Всё остальное** - отвлечение. Не трать время на:
- ❌ Рефакторинг существующего кода
- ❌ Оптимизацию производительности
- ❌ Создание дополнительной документации
- ❌ Настройку CI/CD
- ❌ Event Handlers

**Фокус на:**
- ✅ Написание тестов для Chat UseCases
- ✅ Реализация Query UseCases
- ✅ Достижение coverage >85%

**Результат через 5-6 часов:**
- ✅ UseCase layer готов на 100%
- ✅ Можно переходить к infrastructure layer
- ✅ Вся бизнес-логика протестирована

---

**Удачи! 🚀**
