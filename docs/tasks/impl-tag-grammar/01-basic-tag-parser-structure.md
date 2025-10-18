# Task 01: Basic Tag Parser Structure

**Статус:** Pending
**Приоритет:** High
**Зависимости:** None
**Оценка:** 2-3 дня

## Описание

Реализовать базовую структуру парсера тегов с регистрацией известных тегов и основными типами данных.

## Цели

1. Создать основные типы данных для парсера
2. Реализовать регистрацию известных тегов
3. Подготовить инфраструктуру для расширения

## Технические требования

### Структуры данных

```go
// internal/tag/parser.go
package tag

type TagParser struct {
    knownTags map[string]TagDefinition
}

type TagDefinition struct {
    Name            string
    RequiresValue   bool
    ValueType       ValueType
    AllowedValues   []string  // для Enum
    Validator       func(value string) error
}

type ValueType int

const (
    ValueTypeString ValueType = iota
    ValueTypeUsername
    ValueTypeDate
    ValueTypeEnum
)

type ParseResult struct {
    Tags      []ParsedTag
    PlainText string
}

type ParsedTag struct {
    Key   string
    Value string
}
```

### Регистрация системных тегов

Реализовать функцию `NewTagParser()` с регистрацией всех системных тегов:

**Entity Creation Tags:**
- `task` - требует значение, тип String
- `bug` - требует значение, тип String
- `epic` - требует значение, тип String

**Entity Management Tags:**
- `status` - требует значение, тип Enum (значения зависят от типа сущности)
- `assignee` - опциональное значение, тип Username
- `priority` - требует значение, тип Enum (High, Medium, Low)
- `due` - опциональное значение, тип Date
- `title` - требует значение, тип String

**Bug-Specific Tags:**
- `severity` - требует значение, тип Enum (Critical, Major, Minor, Trivial)

### Валидаторы

Реализовать базовые валидаторы:

```go
func validateUsername(value string) error {
    // Проверка формата @username
}

func validateISODate(value string) error {
    // Проверка ISO 8601 формата
}
```

## Acceptance Criteria

- [ ] Созданы все необходимые структуры данных
- [ ] Реализована функция `NewTagParser()` с регистрацией всех тегов
- [ ] Реализованы базовые валидаторы для username и date
- [ ] Метод `isKnownTag(name string) bool` работает корректно
- [ ] Код покрыт базовыми unit-тестами

## Файловая структура

```
internal/tag/
├── parser.go           # Основные структуры и NewTagParser()
├── types.go            # ValueType и константы
├── validators.go       # Валидаторы
└── parser_test.go      # Unit-тесты
```

## Тесты

```go
func TestIsKnownTag(t *testing.T) {
    parser := NewTagParser()

    assert.True(t, parser.isKnownTag("task"))
    assert.True(t, parser.isKnownTag("status"))
    assert.False(t, parser.isKnownTag("unknown"))
    assert.False(t, parser.isKnownTag("hashtag"))
}

func TestValidateUsername(t *testing.T) {
    assert.NoError(t, validateUsername("@alex"))
    assert.Error(t, validateUsername("alex"))
    assert.Error(t, validateUsername("@"))
}

func TestValidateISODate(t *testing.T) {
    assert.NoError(t, validateISODate("2025-10-20"))
    assert.NoError(t, validateISODate("2025-10-20T15:30:00"))
    assert.Error(t, validateISODate("20-10-2025"))
    assert.Error(t, validateISODate("tomorrow"))
}
```

## Ссылки

- Спецификация: `docs/03-tag-grammar.md` (строки 464-658)
- Псевдокод парсера: `docs/03-tag-grammar.md` (строки 463-592)
