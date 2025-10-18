# Task Use Cases Implementation Plan

Этот каталог содержит пошаговый план реализации use cases для работы с Task агрегатом.

## Обзор

Цель: Создать полнофункциональный слой application logic (use cases) для управления задачами с использованием Event Sourcing и CQRS.

## Структура задач

### ✅ Phase 1: Архитектура и планирование

| Задача | Файл | Статус | Описание |
|--------|------|--------|----------|
| **Task 01** | [01-usecase-architecture.md](01-usecase-architecture.md) | ✅ Completed | Архитектура use case слоя, паттерны, интерфейсы |

### ✅ Phase 2: Основные операции

| Задача | Файл | Статус | Описание |
|--------|------|--------|----------|
| **Task 02** | [02-create-task-usecase.md](02-create-task-usecase.md) | ✅ Completed | Создание задачи (CreateTask) |
| **Task 03** | [03-change-status-usecase.md](03-change-status-usecase.md) | ✅ Completed | Изменение статуса (ChangeStatus) |
| **Task 04** | [04-assign-task-usecase.md](04-assign-task-usecase.md) | ✅ Completed | Назначение исполнителя (AssignTask) |
| **Task 05** | [05-simple-attribute-usecases.md](05-simple-attribute-usecases.md) | ✅ Completed | Изменение приоритета и дедлайна |

### ✅ Phase 3: Тестирование

| Задача | Файл | Статус | Описание |
|--------|------|--------|----------|
| **Task 06** | [06-testing-strategy.md](06-testing-strategy.md) | 📝 Pending | Стратегия тестирования, инструменты, helpers |

## Порядок выполнения

Задачи должны выполняться **последовательно**, так как каждая зависит от предыдущей:

```
01 → 02 → 03 → 04 → 05 → 06
```

### 1. Task 01: Use Case Architecture (2-3 часа)

Определяет фундамент для всех остальных use cases:
- Структуру директорий
- Паттерн Command/Result
- Обработку ошибок
- Интерфейсы

**Результат**: Готовые шаблоны и общие компоненты

### 2. Task 02: CreateTask (3-4 часа)

Первый полноценный use case — эталон для остальных:
- Создание задачи
- Валидация полей
- Работа с Event Store
- Полное покрытие тестами

**Результат**: Рабочий CreateTaskUseCase с тестами

### 3. Task 03: ChangeStatus (2-3 часа)

Первый use case, работающий с существующим агрегатом:
- Загрузка из Event Store
- Восстановление состояния из событий
- Optimistic locking
- Идемпотентность

**Результат**: Рабочий ChangeStatusUseCase с тестами

### 4. Task 04: AssignTask (2-3 часа)

Первый use case с внешней зависимостью:
- Dependency Injection
- UserRepository interface
- Mock для тестов
- Валидация существования пользователя

**Результат**: Рабочий AssignTaskUseCase с тестами и моками

### 5. Task 05: Simple Attributes (2-3 часа)

Два простых use case для закрепления паттерна:
- ChangePriority
- SetDueDate
- Схожая структура
- Простая валидация

**Результат**: Два рабочих use case с тестами

### 6. Task 06: Testing Strategy (2 часа)

Финальная задача — систематизация тестирования:
- Unit, Integration, E2E тесты
- Test helpers и fixtures
- Coverage checking
- CI/CD интеграция

**Результат**: Полная тестовая инфраструктура

## Общая оценка времени

- **Task 01**: 2-3 часа
- **Task 02**: 3-4 часа
- **Task 03**: 2-3 часа
- **Task 04**: 2-3 часа
- **Task 05**: 2-3 часа
- **Task 06**: 2 часа

**Итого**: ~13-18 часов работы

## Результаты после завершения

После выполнения всех задач у вас будет:

### 1. Реализованные Use Cases

```
internal/usecase/task/
├── commands.go              # Все команды
├── results.go               # Результаты
├── errors.go                # Ошибки
├── create_task.go           # ✅
├── change_status.go         # ✅
├── assign_task.go           # ✅
├── change_priority.go       # ✅
├── set_due_date.go          # ✅
└── *_test.go                # Полное покрытие тестами
```

### 2. Тестовая инфраструктура

```
tests/
├── mocks/
│   └── user_repository.go   # Mock для UserRepository
├── testutil/
│   ├── db.go                # Database helpers
│   └── fixtures.go          # Test data builders
└── integration/
    └── usecase/
        └── *.go             # Integration tests
```

### 3. Инфраструктура

```
internal/infrastructure/eventstore/
├── eventstore.go            # Интерфейс
├── inmemory.go             # In-memory для тестов
└── mongodb.go              # MongoDB реализация (будущее)
```

### 4. Готовность к интеграции

- ✅ Use cases готовы к использованию в HTTP handlers
- ✅ Готовы к интеграции с Tag Parser
- ✅ Готовы к работе с WebSocket
- ✅ Полное покрытие тестами (>80%)

## Следующие шаги после завершения

1. **Tag Parser реализация** (из `docs/03-tag-grammar.md`)
   - Парсинг тегов из сообщений
   - Конвертация тегов в команды
   - Интеграция с use cases

2. **HTTP Handlers**
   - REST API endpoints
   - HTMX integration
   - Request validation

3. **WebSocket Integration**
   - Real-time updates
   - Broadcasting events
   - Presence tracking

4. **Projections (Read Models)**
   - TaskProjection для списков задач
   - Board view для канбана
   - Activity log

## Принципы разработки

### 1. Test-Driven Development (TDD)

Для каждого use case:
1. Пишем тесты
2. Реализуем код
3. Рефакторим

### 2. Single Responsibility

Каждый use case отвечает за одну операцию.

### 3. Fail Fast

Валидация в начале, явная обработка ошибок.

### 4. Idempotency

Повторный вызов с теми же параметрами не создает дублирующих событий.

### 5. Event Sourcing

Все изменения через события, агрегаты восстанавливаются из истории.

## Ресурсы

- [Domain Model](../02-domain-model.md) — модель домена Task
- [Tag Grammar](../03-tag-grammar.md) — грамматика тегов
- [Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html)
- [Use Case Pattern](https://martinfowler.com/eaaCatalog/applicationFacade.html)

## Вопросы и проблемы

Если возникают вопросы по реализации, обращайтесь к:
1. Документации конкретной задачи
2. Уже реализованным use cases (как примеру)
3. Domain model для понимания бизнес-логики

## Статус обновления

**Последнее обновление**: 2025-10-17
**Версия плана**: 1.0
**Статус**: Все задачи документированы, готовы к реализации
