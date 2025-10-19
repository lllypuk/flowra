# Task 07: Integration with Domain Model - Subtasks

**Родительская задача:** `docs/tasks/impl-tag-grammar/07-integration-with-domain.md`

## Обзор

Task 07 слишком большая для выполнения за один раз. Разбита на 6 подзадач для последовательной реализации.

## Текущее состояние проекта

### ✅ Уже реализовано

1. **Tag система** (`internal/domain/tag/`):
   - Parser - парсинг тегов из сообщений
   - Validators - валидация тегов
   - Processor - обработка тегов и генерация команд
   - Commands - структуры команд (CreateTaskCommand, ChangeStatusCommand и т.д.)
   - Formatter - форматирование bot responses

2. **Domain модели**:
   - `Chat aggregate` - базовый функционал, Event Sourcing, ConvertToTask()
   - `Task aggregate` - полный функционал с Event Sourcing, все операции
   - Event infrastructure - BaseEvent, Metadata

### ❌ Нужно реализовать

1. Domain Commands для Chat aggregate
2. Domain Events для Chat aggregate (расширение существующих)
3. Методы Chat aggregate для всех tag операций
4. CommandExecutor - связь между tag commands и domain methods
5. Integration Handler - полный pipeline обработки
6. Unit тесты

## Подзадачи (в порядке выполнения)

### ✅ [07.1 - Chat Domain Commands](./07.1-chat-domain-commands.md)
**Оценка:** 0.5 дня
**Зависимости:** нет
**Статус:** Completed

Создать domain commands для Chat aggregate. Эти команды будут использоваться CommandExecutor'ом для выполнения операций на aggregate.

**Файлы:**
- `internal/domain/chat/commands.go` (170 строк) ✅

---

### ✅ [07.2 - Chat Domain Events](./07.2-chat-domain-events.md)
**Оценка:** 0.5 дня
**Зависимости:** 07.1
**Статус:** Completed

Расширить события Chat aggregate для всех tag операций. События публикуются при изменении состояния.

**Файлы:**
- `internal/domain/chat/events.go` (362 строки, +249) ✅

---

### ✅ [07.3 - Chat Aggregate Methods](./07.3-chat-aggregate-methods.md)
**Оценка:** 1 день
**Зависимости:** 07.1, 07.2
**Статус:** Completed

Реализовать методы Chat aggregate для всех операций: ChangeStatus, AssignUser, SetPriority, SetDueDate, Rename, SetSeverity. Включает валидацию и event sourcing.

**Файлы:**
- `internal/domain/chat/chat.go` (697 строк, +443) ✅

---

### ✅ [07.4 - CommandExecutor](./07.4-command-executor.md)
**Оценка:** 1 день
**Зависимости:** 07.3
**Статус:** Completed

Реализовать CommandExecutor - компонент, который выполняет tag commands на Chat aggregate. Включает резолвинг пользователей и публикацию событий.

**Файлы:**
- `internal/domain/tag/executor.go` (252 строки) ✅

---

### ✅ [07.5 - Integration Handler](./07.5-integration-handler.md)
**Оценка:** 0.5 дня
**Зависимости:** 07.4
**Статус:** Completed

Создать handler для полного pipeline: парсинг → валидация → обработка → выполнение → bot response.

**Файлы:**
- `internal/domain/tag/handler.go` (154 строки) ✅

---

### ✅ [07.6 - Unit Tests](./07.6-unit-tests.md)
**Оценка:** 1 день
**Зависимости:** 07.3, 07.4, 07.5
**Статус:** Completed

Написать unit тесты для всех новых компонентов.

**Файлы:**
- `internal/domain/chat/chat_test.go` (680 строк, +354) ✅
- Coverage: 84.8% (Chat), 57.3% (Tag)

---

## Общая оценка

**Итого:** 4.5 дня (вместо 3-4 дней в оригинальной задаче)

## Порядок выполнения

1. Начать с 07.1 (Commands)
2. Затем 07.2 (Events)
3. Затем 07.3 (Aggregate Methods) - **самая большая подзадача**
4. Затем 07.4 (CommandExecutor)
5. Затем 07.5 (Handler)
6. Завершить 07.6 (Tests)

## Acceptance Criteria всего Task 07

- [ ] Созданы все domain commands для тегов
- [ ] Созданы все domain events
- [ ] Реализованы методы Chat aggregate для всех операций
- [ ] Реализован CommandExecutor с методом Execute()
- [ ] Реализованы все execute-методы для команд
- [ ] Интегрирован с event sourcing (публикация событий)
- [ ] Обработано превращение обычного чата в typed (Task/Bug/Epic)
- [ ] Обработано изменение типа чата (Task → Bug)
- [ ] Валидация в aggregate согласуется с валидацией тегов
- [ ] Код покрыт unit-тестами

## Связанные документы

- Родительская задача: `docs/tasks/impl-tag-grammar/07-integration-with-domain.md`
- Tag Grammar Spec: `docs/03-tag-grammar.md`
- Event Sourcing: реализовано в Phase 1
