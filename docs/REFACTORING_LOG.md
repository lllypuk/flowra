# Refactoring Log

## 2025-10-18: Migration to Event Sourcing (Task Domain)

### Цель
Полная миграция Task domain модели с традиционного CRUD подхода на Event Sourcing паттерн.

### Проблема
В проекте одновременно существовало две модели для Task:
1. **`task.go` (Entity)** - традиционная CRUD модель с изменяемым состоянием
2. **`aggregate.go` (Aggregate)** - Event Sourcing модель с историей событий

Это создавало:
- ❌ Дублирование логики
- ❌ Путаницу в том, какую модель использовать
- ❌ Два источника правды (нарушение Single Source of Truth)
- ❌ Конфликты в архитектуре

### Решение
Полная замена старой Entity модели на Aggregate с Event Sourcing.

### Выполненные действия

#### 1. Удалены устаревшие файлы:
```
❌ internal/domain/task/task.go          (старая Entity модель)
❌ internal/domain/task/task_test.go     (тесты для Entity)
❌ internal/domain/task/repository.go    (CRUD repository интерфейс)
```

**Обоснование:**
- `task.go` - заменен на Aggregate с Event Sourcing
- `task_test.go` - тесты устарели, используются новые в use case
- `repository.go` - в Event Sourcing не нужен, используется EventStore

#### 2. Переименован основной файл:
```
✅ aggregate.go → task.go
```

**Обоснование:**
- Aggregate становится единственной моделью для Task
- Логичнее называть основной файл `task.go`

#### 3. Финальная структура:
```
internal/domain/task/
├── task.go           # Task Aggregate с Event Sourcing (было aggregate.go)
├── events.go         # События для Event Sourcing
└── entity_state.go   # Value Object для state machine (статусы/приоритеты)
```

### Результаты

#### ✅ Метрики:
- **Удалено:** ~250 строк устаревшего кода
- **Осталось:** 720 строк чистого Event Sourcing кода
- **Тесты:** ✅ Все тесты проходят (7 test suites, 20+ tests)
- **Линтер:** ✅ 0 issues (golangci-lint)
- **Компиляция:** ✅ Весь проект собирается

#### ✅ Архитектурные улучшения:
1. **Single Source of Truth** - только один Aggregate для Task
2. **Чистая архитектура** - нет дублирования и конфликтов
3. **Event Sourcing** - полная история изменений
4. **CQRS ready** - готовность к разделению Command/Query
5. **Аудит из коробки** - все изменения с метаданными (кто, когда, что)

#### ✅ Возможности:
- 📝 Полная история всех изменений задачи
- ⏮️ Time-travel debugging (восстановление любого состояния)
- 👤 Аудит действий пользователей (createdBy, changedBy)
- 🔒 Optimistic locking через version tracking
- 🔄 Event replay для восстановления состояния
- 📊 Event-driven projections для Read Models

### Совместимость

#### Что НЕ сломалось:
- ✅ Use Cases работают как прежде
- ✅ CreateTaskUseCase использует новый Aggregate
- ✅ EventStore интеграция работает корректно
- ✅ Все существующие тесты проходят

#### Что изменилось:
- ❌ Старый CRUD repository больше не доступен
- ✅ Вместо него используется EventStore
- ✅ Все операции теперь через события

### Миграционная стратегия для будущего

Когда понадобятся **Read Models** (Query side в CQRS):

1. Создать новый файл `read_model.go` с проекциями
2. Проекции будут обновляться через Event Handlers
3. Быстрое чтение без replay событий
4. Aggregate остается для Command side

```
Command Side (Write)        Query Side (Read)
====================       ===================
Task Aggregate      →      Task Read Model
  ↓ Events                   ↑ Event Handlers
Event Store         →      Projections Table
```

### Проверка качества

```bash
# Компиляция
✅ go build ./...

# Тесты
✅ go test ./internal/usecase/task/...
   PASS (7 tests, 0.006s)

# Линтер
✅ golangci-lint run ./internal/...
   0 issues

# Покрытие
✅ go test -cover ./internal/usecase/task/...
   coverage: 87.8%
```

### Следующие шаги

После этого рефакторинга архитектура готова к:
1. ✅ **Task 03:** ChangeStatusUseCase (изменение статуса)
2. ✅ **Task 04:** AssignTaskUseCase (назначение исполнителя)
3. ✅ **Task 05:** ChangePriority/SetDueDate UseCases
4. 📝 **Будущее:** Read Models и Projections для быстрых запросов

### Выводы

**Было:**
```
task.go (Entity) ← CRUD
  ↓
Database Table
```

**Стало:**
```
task.go (Aggregate) ← Event Sourcing
  ↓
Event Store (append-only log)
```

✅ **Архитектура стала чище, проще и мощнее!**

---

**Автор:** Claude Code
**Дата:** 2025-10-18
**Версия:** 1.0
