# Tag Grammar Implementation Tasks

Этот каталог содержит детальные задачи по реализации системы тегов для управления задачами через чат, основанные на спецификации из `docs/03-tag-grammar.md`.

## Обзор

Система тегов позволяет пользователям управлять задачами непосредственно из чата, используя специальный синтаксис с `#` символом. Реализация разбита на 9 логических задач.

## Список задач

### Phase 1: Core Parsing (Tasks 01-02)

#### [01. Basic Tag Parser Structure](./01-basic-tag-parser-structure.md)
**Приоритет:** High | **Оценка:** 2-3 дня | **Зависимости:** None

Базовая структура парсера тегов с регистрацией известных тегов и основными типами данных.

**Ключевые deliverables:**
- `TagParser` структура
- `TagDefinition` с типами значений
- Регистрация всех системных тегов
- Базовые валидаторы (username, date)

#### [02. Tag Position Parsing Logic](./02-tag-position-parsing.md)
**Приоритет:** High | **Оценка:** 3-4 дня | **Зависимости:** Task 01

Логика парсинга тегов с учетом их позиции в сообщении.

**Ключевые deliverables:**
- Метод `Parse(content string) ParseResult`
- Парсинг тегов в начале сообщения
- Парсинг тегов на отдельной строке
- Извлечение многословных значений

---

### Phase 2: Tag Processing (Tasks 03-04)

#### [03. Entity Creation Tags Implementation](./03-entity-creation-tags.md)
**Приоритет:** High | **Оценка:** 2-3 дня | **Зависимости:** Task 01, 02

Реализация тегов создания сущностей: `#task`, `#bug`, `#epic`.

**Ключевые deliverables:**
- Команды создания сущностей
- Валидация title (не пустой)
- `TagProcessor` с методом `ProcessTags()`
- Обработка превращения чата в typed

#### [04. Entity Management Tags Implementation](./04-entity-management-tags.md)
**Приоритет:** High | **Оценка:** 4-5 дней | **Зависимости:** Task 01, 02, 03

Реализация тегов управления: `#status`, `#assignee`, `#priority`, `#due`, `#title`, `#severity`.

**Ключевые deliverables:**
- Команды управления сущностями
- CASE-SENSITIVE валидация для enum значений
- Резолвинг пользователей для `#assignee`
- Парсинг ISO 8601 дат для `#due`
- Context-dependent валидация статусов

---

### Phase 3: Validation & Error Handling (Tasks 05-06)

#### [05. Tag Validation System](./05-tag-validation-system.md)
**Приоритет:** High | **Оценка:** 2-3 дня | **Зависимости:** Task 01-04

Централизованная система валидации тегов.

**Ключевые deliverables:**
- `ValidationContext` с entity type
- Типы валидаторов (Syntax, Enum, Business, Required)
- `CompositeValidator` для цепочки проверок
- Метод `ValidateTags()` с частичным применением

#### [06. Error Handling and User Feedback](./06-error-handling-and-feedback.md)
**Приоритет:** High | **Оценка:** 2 дня | **Зависимости:** Task 05

Система обработки ошибок и генерация bot responses.

**Ключевые deliverables:**
- `ProcessingResult` структура
- Генерация bot responses
- Форматирование успехов и ошибок
- Сохранение сообщений даже при ошибках

---

### Phase 4: Domain Integration (Task 07)

#### [07. Integration with Domain Model](./07-integration-with-domain.md)
**Приоритет:** High | **Оценка:** 3-4 дня | **Зависимости:** Task 03-06

Интеграция системы тегов с domain model и event sourcing.

**Ключевые deliverables:**
- Domain commands и events
- Методы Chat aggregate
- `CommandExecutor`
- Интеграция с event sourcing
- Полный pipeline от тега до изменения в БД

---

### Phase 5: Testing (Tasks 08-09)

#### [08. Comprehensive Unit Tests](./08-comprehensive-unit-tests.md)
**Приоритет:** High | **Оценка:** 2-3 дня | **Зависимости:** Task 01-05

Исчерпывающие unit-тесты для всех компонентов.

**Ключевые deliverables:**
- Тесты парсера (все примеры из спецификации)
- Тесты валидации (все типы)
- Edge case тесты
- Покрытие кода >80%

#### [09. Integration Tests](./09-integration-tests.md)
**Приоритет:** High | **Оценка:** 3-4 дня | **Зависимости:** Task 01-08

End-to-end интеграционные тесты.

**Ключевые deliverables:**
- Полный pipeline тестирование
- Тесты всех сценариев из спецификации
- Event sourcing и проекции
- Тесты с реальной MongoDB

---

## Общая оценка

**Общее время:** ~22-30 дней (4-6 недель)

## Порядок выполнения

Задачи должны выполняться последовательно по фазам:

1. **Phase 1 (Tasks 01-02):** Базовый парсинг — ~1 неделя
2. **Phase 2 (Tasks 03-04):** Обработка тегов — ~1-1.5 недели
3. **Phase 3 (Tasks 05-06):** Валидация и ошибки — ~1 неделя
4. **Phase 4 (Task 07):** Интеграция с domain — ~1 неделя
5. **Phase 5 (Tasks 08-09):** Тестирование — ~1-1.5 недели

## Архитектура

```
internal/domain/
├── tag/
│   ├── parser.go              # Task 01, 02
│   ├── types.go               # Task 01
│   ├── validators.go          # Task 01, 04
│   ├── commands.go            # Task 03, 04
│   ├── processor.go           # Task 03, 04
│   ├── validation/
│   │   ├── validator.go       # Task 05
│   │   ├── syntax_validator.go
│   │   ├── enum_validator.go
│   │   ├── business_validator.go
│   │   └── composite_validator.go
│   ├── result.go              # Task 06
│   ├── formatter.go           # Task 06
│   ├── handler.go             # Task 06, 07
│   ├── executor.go            # Task 07
│   └── *_test.go              # Task 08
└── chat/
    ├── aggregate.go           # Task 07
    ├── commands.go            # Task 07
    └── events.go              # Task 07

tests/
└── integration/
    └── tag/
        └── *_test.go          # Task 09
```

**Примечание:** Пакет `tag` находится в `internal/domain/`, так как теги являются частью domain model и описывают ubiquitous language проекта.

## Критерии завершения

Все задачи считаются завершенными, когда:

- ✅ Все Acceptance Criteria выполнены
- ✅ Все unit-тесты проходят
- ✅ Все интеграционные тесты проходят
- ✅ Покрытие кода >80%
- ✅ Все примеры из спецификации работают
- ✅ Документация обновлена
- ✅ Code review пройден

## Ссылки

- **Спецификация:** [`docs/03-tag-grammar.md`](../../03-tag-grammar.md)
- **Domain Model:** [`docs/02-domain-model.md`](../../02-domain-model.md)
- **Core Use Cases:** [`docs/01-core-use-cases.md`](../../01-core-use-cases.md)

## Дополнительные материалы

### Примеры использования

Все примеры из спецификации (строки 755-863) должны работать после завершения всех задач:

1. **Создание задачи с атрибутами**
2. **Обсуждение + изменение статуса**
3. **Несколько тегов с ошибкой**
4. **Превращение чата в задачу**
5. **Drag-n-drop на канбане**
6. **Обычное обсуждение с # в тексте**

### UX Features (V2)

После MVP могут быть добавлены:
- Автокомплит тегов
- UI shortcuts (кнопки)
- Подсветка тегов в реальном времени
- Алиасы тегов (#s → #status)
- Natural language для дат
- Custom tags

## Контакты

Вопросы по задачам:
- Архитектура: см. `docs/03-tag-grammar.md`
- Domain model: см. `internal/domain/`
- Event Sourcing: уже реализовано в Phase 1
