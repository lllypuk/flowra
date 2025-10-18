# Task 03: Entity Creation Tags Implementation

**Статус:** Pending
**Приоритет:** High
**Зависимости:** Task 01, Task 02
**Оценка:** 2-3 дня

## Описание

Реализовать обработку тегов создания сущностей: `#task`, `#bug`, `#epic`. Эти теги создают новый чат-сущность или переопределяют тип существующего чата.

## Цели

1. Реализовать команды создания сущностей
2. Интегрировать с domain model (Chat aggregate)
3. Обработать edge cases (пустой title, повторное создание)
4. Реализовать валидацию

## Технические требования

### Entity Creation Tags

```go
#task <title>       — Создать Task
#bug <title>        — Создать Bug
#epic <title>       — Создать Epic
```

### Поведение

**Если написать в обычном чате (без типа):**
- Создаётся новая задача-чат
- Чат превращается в typed chat (Task/Bug/Epic)
- Все предыдущие сообщения остаются в истории
- Чат появляется на канбане

**Если написать в typed чате:**
- Переопределяет тип (Task → Bug, например)
- Title может быть изменён

**Title:**
- Обязателен (валидация: не пустой после trim)
- Может содержать любые символы, включая спецсимволы
- Может быть многословным

### Структура команд

```go
// internal/tag/commands.go
package tag

type CreateTaskCommand struct {
    ChatID uuid.UUID
    Title  string
}

type CreateBugCommand struct {
    ChatID uuid.UUID
    Title  string
}

type CreateEpicCommand struct {
    ChatID uuid.UUID
    Title  string
}
```

### Валидация

```go
// internal/tag/validator.go

func ValidateEntityCreation(tagKey, title string) error {
    trimmed := strings.TrimSpace(title)

    if trimmed == "" {
        return fmt.Errorf("❌ %s title is required. Usage: #%s <title>",
            strings.Title(tagKey), tagKey)
    }

    return nil
}
```

### Интеграция с парсером

```go
// internal/tag/processor.go
package tag

type TagProcessor struct {
    parser *TagParser
}

func (tp *TagProcessor) ProcessTags(chatID uuid.UUID, parsedTags []ParsedTag) ([]Command, []error) {
    var commands []Command
    var errors []error

    for _, tag := range parsedTags {
        switch tag.Key {
        case "task":
            if err := ValidateEntityCreation("task", tag.Value); err != nil {
                errors = append(errors, err)
                continue
            }
            commands = append(commands, CreateTaskCommand{
                ChatID: chatID,
                Title:  strings.TrimSpace(tag.Value),
            })

        case "bug":
            if err := ValidateEntityCreation("bug", tag.Value); err != nil {
                errors = append(errors, err)
                continue
            }
            commands = append(commands, CreateBugCommand{
                ChatID: chatID,
                Title:  strings.TrimSpace(tag.Value),
            })

        case "epic":
            if err := ValidateEntityCreation("epic", tag.Value); err != nil {
                errors = append(errors, err)
                continue
            }
            commands = append(commands, CreateEpicCommand{
                ChatID: chatID,
                Title:  strings.TrimSpace(tag.Value),
            })
        }
    }

    return commands, errors
}
```

## Acceptance Criteria

- [ ] Реализованы команды `CreateTaskCommand`, `CreateBugCommand`, `CreateEpicCommand`
- [ ] Реализована валидация `ValidateEntityCreation()`
- [ ] Реализован `TagProcessor` с методом `ProcessTags()`
- [ ] Пустой title возвращает ошибку с правильным сообщением
- [ ] Title корректно обрабатывается (trim whitespace)
- [ ] Многословные title обрабатываются корректно
- [ ] Код покрыт unit-тестами

## Примеры использования

### Пример 1: Создание Task
```
Input: "#task Реализовать авторизацию"
Output:
  Command: CreateTaskCommand{Title: "Реализовать авторизацию"}
  Errors: []
```

### Пример 2: Создание Bug с атрибутами
```
Input: "#bug Ошибка при логине #severity Critical"
Output:
  Commands: [
    CreateBugCommand{Title: "Ошибка при логине"},
    SetSeverityCommand{Severity: "Critical"}
  ]
  Errors: []
```

### Пример 3: Пустой title
```
Input: "#task"
Output:
  Commands: []
  Errors: ["❌ Task title is required. Usage: #task <title>"]
```

### Пример 4: Title с пробелами
```
Input: "#task    Много пробелов    "
Output:
  Command: CreateTaskCommand{Title: "Много пробелов"}
  Errors: []
```

## Тесты

```go
func TestValidateEntityCreation(t *testing.T) {
    tests := []struct {
        name    string
        tagKey  string
        title   string
        wantErr bool
    }{
        {
            name:    "valid task title",
            tagKey:  "task",
            title:   "Реализовать авторизацию",
            wantErr: false,
        },
        {
            name:    "empty title",
            tagKey:  "task",
            title:   "",
            wantErr: true,
        },
        {
            name:    "whitespace only",
            tagKey:  "task",
            title:   "   ",
            wantErr: true,
        },
        {
            name:    "title with special chars",
            tagKey:  "bug",
            title:   "Fix issue #123 (critical!)",
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEntityCreation(tt.tagKey, tt.title)
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), "title is required")
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

func TestProcessEntityCreationTags(t *testing.T) {
    processor := NewTagProcessor()
    chatID := uuid.New()

    tests := []struct {
        name         string
        tags         []ParsedTag
        wantCommands int
        wantErrors   int
    }{
        {
            name: "create task",
            tags: []ParsedTag{
                {Key: "task", Value: "Реализовать авторизацию"},
            },
            wantCommands: 1,
            wantErrors:   0,
        },
        {
            name: "create bug with severity",
            tags: []ParsedTag{
                {Key: "bug", Value: "Ошибка при логине"},
                {Key: "severity", Value: "Critical"},
            },
            wantCommands: 2,
            wantErrors:   0,
        },
        {
            name: "empty task title",
            tags: []ParsedTag{
                {Key: "task", Value: ""},
            },
            wantCommands: 0,
            wantErrors:   1,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            commands, errors := processor.ProcessTags(chatID, tt.tags)
            assert.Len(t, commands, tt.wantCommands)
            assert.Len(t, errors, tt.wantErrors)
        })
    }
}
```

## Файловая структура

```
internal/tag/
├── commands.go         # Определения команд
├── processor.go        # TagProcessor
├── validator.go        # Валидация
├── processor_test.go   # Тесты
└── validator_test.go   # Тесты валидации
```

## Ссылки

- Entity Creation Tags: `docs/03-tag-grammar.md` (строки 132-155)
- Валидация пустого title: `docs/03-tag-grammar.md` (строки 418-426)
- Примеры: `docs/03-tag-grammar.md` (строки 757-777, 811-827)
