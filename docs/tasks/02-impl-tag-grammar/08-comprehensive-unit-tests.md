# Task 08: Comprehensive Unit Tests

**Статус:** Pending
**Приоритет:** High
**Зависимости:** Task 01, Task 02, Task 03, Task 04, Task 05
**Оценка:** 2-3 дня

## Описание

Создать исчерпывающий набор unit-тестов для всей системы тегов, покрывающий парсинг, валидацию, edge cases и примеры из спецификации.

## Цели

1. Покрыть тестами все компоненты системы тегов
2. Протестировать все примеры из спецификации
3. Добавить edge case тесты
4. Обеспечить высокое покрытие кода (>80%)

## Технические требования

### Структура тестов

```
internal/tag/
├── parser_test.go              # Тесты парсера
├── validator_test.go           # Тесты валидации
├── processor_test.go           # Тесты процессора
├── formatter_test.go           # Тесты форматирования
└── testdata/
    ├── valid_tags.json         # Примеры валидных тегов
    ├── invalid_tags.json       # Примеры невалидных тегов
    └── complex_scenarios.json  # Комплексные сценарии
```

### Тесты парсера (parser_test.go)

```go
package tag

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestTagParser_Parse(t *testing.T) {
    parser := NewTagParser()

    tests := []struct {
        name     string
        input    string
        wantTags []ParsedTag
        wantText string
    }{
        // Базовые примеры из спецификации (строки 84-110)
        {
            name:  "single tag",
            input: "#status Done",
            wantTags: []ParsedTag{
                {Key: "status", Value: "Done"},
            },
            wantText: "",
        },
        {
            name:  "multiple tags on one line",
            input: "#status Done #assignee @alex",
            wantTags: []ParsedTag{
                {Key: "status", Value: "Done"},
                {Key: "assignee", Value: "@alex"},
            },
            wantText: "",
        },
        {
            name:  "tag with multi-word value",
            input: "#task Реализовать авторизацию #priority High",
            wantTags: []ParsedTag{
                {Key: "task", Value: "Реализовать авторизацию"},
                {Key: "priority", Value: "High"},
            },
            wantText: "",
        },
        {
            name:  "text then tags on separate line",
            input: "Закончил работу\n#status Done",
            wantTags: []ParsedTag{
                {Key: "status", Value: "Done"},
            },
            wantText: "Закончил работу",
        },
        {
            name:  "tag from bug example",
            input: "#bug Ошибка в логине\nВоспроизводится на Chrome",
            wantTags: []ParsedTag{
                {Key: "bug", Value: "Ошибка в логине"},
            },
            wantText: "Воспроизводится на Chrome",
        },

        // Edge cases
        {
            name:     "tags in middle of line - ignored",
            input:    "Закончил работу #status Done — отправляю на проверку",
            wantTags: []ParsedTag{},
            wantText: "Закончил работу #status Done — отправляю на проверку",
        },
        {
            name:  "mixed tags and text",
            input: "#status Done какой-то текст #assignee @alex",
            wantTags: []ParsedTag{
                {Key: "status", Value: "Done какой-то текст"}, // #assignee игнорируется
            },
            wantText: "",
        },
        {
            name:     "unknown tag - ignored",
            input:    "Поддержка #hashtags в тексте",
            wantTags: []ParsedTag{},
            wantText: "Поддержка #hashtags в тексте",
        },
        {
            name:  "empty lines ignored",
            input: "#status Done\n\n\n#priority High",
            wantTags: []ParsedTag{
                {Key: "status", Value: "Done"},
                {Key: "priority", Value: "High"},
            },
            wantText: "",
        },
        {
            name:  "tag without value",
            input: "#assignee",
            wantTags: []ParsedTag{
                {Key: "assignee", Value: ""},
            },
            wantText: "",
        },
        {
            name:  "unicode in tag value",
            input: "#task Исправить баг в модуле авторизации 🐛",
            wantTags: []ParsedTag{
                {Key: "task", Value: "Исправить баг в модуле авторизации 🐛"},
            },
            wantText: "",
        },
        {
            name:  "tag value with special characters",
            input: "#task Fix issue #123 (critical!)",
            wantTags: []ParsedTag{
                {Key: "task", Value: "Fix issue #123 (critical!)"},
            },
            wantText: "",
        },
        {
            name:  "multiple tags on separate lines",
            input: "#status Done\n#priority High\n#assignee @alex",
            wantTags: []ParsedTag{
                {Key: "status", Value: "Done"},
                {Key: "priority", Value: "High"},
                {Key: "assignee", Value: "@alex"},
            },
            wantText: "",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := parser.Parse(tt.input)
            assert.Equal(t, tt.wantTags, result.Tags, "tags mismatch")
            assert.Equal(t, tt.wantText, result.PlainText, "text mismatch")
        })
    }
}

func TestTagParser_ParseValueExtraction(t *testing.T) {
    parser := NewTagParser()

    tests := []struct {
        name      string
        input     string
        wantKey   string
        wantValue string
    }{
        {
            name:      "multi-word value",
            input:     "#status In Progress #assignee @alex",
            wantKey:   "status",
            wantValue: "In Progress",
        },
        {
            name:      "long value",
            input:     "#task Реализовать функцию авторизации",
            wantKey:   "task",
            wantValue: "Реализовать функцию авторизации",
        },
        {
            name:      "single word value",
            input:     "#priority High",
            wantKey:   "priority",
            wantValue: "High",
        },
        {
            name:      "empty value",
            input:     "#assignee",
            wantKey:   "assignee",
            wantValue: "",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := parser.Parse(tt.input)
            assert.NotEmpty(t, result.Tags)
            tag := result.Tags[0]
            assert.Equal(t, tt.wantKey, tag.Key)
            assert.Equal(t, tt.wantValue, tag.Value)
        })
    }
}
```

### Тесты валидации (validator_test.go)

```go
func TestValidateStatus(t *testing.T) {
    tests := []struct {
        name       string
        entityType string
        status     string
        wantErr    bool
        errSubstr  string
    }{
        // Task statuses
        {name: "task valid status", entityType: "Task", status: "Done", wantErr: false},
        {name: "task invalid case", entityType: "Task", status: "done", wantErr: true, errSubstr: "Invalid status"},
        {name: "task wrong status", entityType: "Task", status: "Fixed", wantErr: true, errSubstr: "Invalid status"},

        // Bug statuses
        {name: "bug valid status", entityType: "Bug", status: "Fixed", wantErr: false},
        {name: "bug invalid case", entityType: "Bug", status: "fixed", wantErr: true, errSubstr: "Invalid status"},
        {name: "bug wrong status", entityType: "Bug", status: "Done", wantErr: true, errSubstr: "Invalid status"},

        // Epic statuses
        {name: "epic valid status", entityType: "Epic", status: "Completed", wantErr: false},
        {name: "epic invalid case", entityType: "Epic", status: "completed", wantErr: true, errSubstr: "Invalid status"},
        {name: "epic wrong status", entityType: "Epic", status: "Done", wantErr: true, errSubstr: "Invalid status"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateStatus(tt.entityType, tt.status)
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errSubstr)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

func TestValidateAssignee(t *testing.T) {
    tests := []struct {
        name      string
        value     string
        wantErr   bool
        errSubstr string
    }{
        {name: "valid assignee", value: "@alex", wantErr: false},
        {name: "remove assignee - @none", value: "@none", wantErr: false},
        {name: "remove assignee - empty", value: "", wantErr: false},
        {name: "missing @", value: "alex", wantErr: true, errSubstr: "Invalid assignee format"},
        {name: "only @", value: "@", wantErr: true, errSubstr: "Invalid assignee format"},
        {name: "username with dots", value: "@user.name", wantErr: false},
        {name: "username with dashes", value: "@user-name", wantErr: false},
        {name: "username with underscores", value: "@user_name", wantErr: false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateAssignee(tt.value)
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errSubstr)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

func TestValidatePriority(t *testing.T) {
    tests := []struct {
        name      string
        priority  string
        wantErr   bool
        errSubstr string
    }{
        {name: "valid High", priority: "High", wantErr: false},
        {name: "valid Medium", priority: "Medium", wantErr: false},
        {name: "valid Low", priority: "Low", wantErr: false},
        {name: "invalid case - high", priority: "high", wantErr: true, errSubstr: "Invalid priority"},
        {name: "invalid value - Urgent", priority: "Urgent", wantErr: true, errSubstr: "Invalid priority"},
        {name: "invalid value - Critical", priority: "Critical", wantErr: true, errSubstr: "Invalid priority"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePriority(tt.priority)
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errSubstr)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

func TestValidateDueDate(t *testing.T) {
    tests := []struct {
        name      string
        dateStr   string
        wantNil   bool
        wantErr   bool
        errSubstr string
    }{
        {name: "valid YYYY-MM-DD", dateStr: "2025-10-20", wantNil: false, wantErr: false},
        {name: "valid YYYY-MM-DDTHH:MM", dateStr: "2025-10-20T15:30", wantNil: false, wantErr: false},
        {name: "valid YYYY-MM-DDTHH:MM:SS", dateStr: "2025-10-20T15:30:00", wantNil: false, wantErr: false},
        {name: "valid with UTC", dateStr: "2025-10-20T15:30:00Z", wantNil: false, wantErr: false},
        {name: "valid with timezone", dateStr: "2025-10-20T15:30:00+03:00", wantNil: false, wantErr: false},
        {name: "empty - remove", dateStr: "", wantNil: true, wantErr: false},
        {name: "invalid format DD-MM-YYYY", dateStr: "20-10-2025", wantNil: false, wantErr: true, errSubstr: "Invalid date format"},
        {name: "invalid format - natural", dateStr: "tomorrow", wantNil: false, wantErr: true, errSubstr: "Invalid date format"},
        {name: "invalid format - random", dateStr: "not a date", wantNil: false, wantErr: true, errSubstr: "Invalid date format"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ValidateDueDate(tt.dateStr)
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errSubstr)
            } else {
                assert.NoError(t, err)
                if tt.wantNil {
                    assert.Nil(t, result)
                } else {
                    assert.NotNil(t, result)
                }
            }
        })
    }
}

func TestValidateSeverity(t *testing.T) {
    tests := []struct {
        name      string
        severity  string
        wantErr   bool
        errSubstr string
    }{
        {name: "valid Critical", severity: "Critical", wantErr: false},
        {name: "valid Major", severity: "Major", wantErr: false},
        {name: "valid Minor", severity: "Minor", wantErr: false},
        {name: "valid Trivial", severity: "Trivial", wantErr: false},
        {name: "invalid case", severity: "critical", wantErr: true, errSubstr: "Invalid severity"},
        {name: "invalid value", severity: "High", wantErr: true, errSubstr: "Invalid severity"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateSeverity(tt.severity)
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errSubstr)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Тесты сценариев из спецификации

```go
func TestSpecificationExamples(t *testing.T) {
    // Примеры из docs/03-tag-grammar.md строки 757-863

    t.Run("Scenario 1: Create task with attributes", func(t *testing.T) {
        input := "#task Реализовать OAuth авторизацию #priority High #assignee @alex"

        parser := NewTagParser()
        result := parser.Parse(input)

        assert.Len(t, result.Tags, 3)
        assert.Equal(t, "task", result.Tags[0].Key)
        assert.Equal(t, "Реализовать OAuth авторизацию", result.Tags[0].Value)
        assert.Equal(t, "priority", result.Tags[1].Key)
        assert.Equal(t, "High", result.Tags[1].Value)
        assert.Equal(t, "assignee", result.Tags[2].Key)
        assert.Equal(t, "@alex", result.Tags[2].Value)
    })

    t.Run("Scenario 2: Discussion + status change", func(t *testing.T) {
        input := "Закончил первую итерацию, отправляю на код-ревью\n#status In Progress"

        parser := NewTagParser()
        result := parser.Parse(input)

        assert.Len(t, result.Tags, 1)
        assert.Equal(t, "status", result.Tags[0].Key)
        assert.Equal(t, "In Progress", result.Tags[0].Value)
        assert.Equal(t, "Закончил первую итерацию, отправляю на код-ревью", result.PlainText)
    })

    t.Run("Scenario 3: Multiple tags with error", func(t *testing.T) {
        input := "#status Done #assignee @unknown #priority High"

        parser := NewTagParser()
        validator := NewTagValidationSystem(mockUserRepo)
        ctx := ValidationContext{EntityType: "Task"}

        parseResult := parser.Parse(input)
        validTags, errors := validator.ValidateTags(parseResult.Tags, ctx)

        // status и priority валидны, assignee невалиден
        assert.Len(t, validTags, 2)
        assert.Len(t, errors, 1)
        assert.Contains(t, errors[0].Error(), "not found")
    })
}
```

## Acceptance Criteria

- [ ] Созданы тесты для всех компонентов парсера
- [ ] Созданы тесты для всех валидаторов
- [ ] Протестированы все примеры из спецификации
- [ ] Добавлены edge case тесты
- [ ] Покрытие кода >80%
- [ ] Все тесты проходят
- [ ] Тесты документируют поведение системы

## Запуск тестов

```bash
# Все тесты
go test ./internal/tag/...

# С покрытием
go test -cover ./internal/tag/...

# Детальное покрытие
go test -coverprofile=coverage.out ./internal/tag/...
go tool cover -html=coverage.out
```

## Ссылки

- Примеры парсинга: `docs/03-tag-grammar.md` (строки 84-110)
- Примеры валидации: `docs/03-tag-grammar.md` (строки 146-301)
- Сценарии: `docs/03-tag-grammar.md` (строки 755-863)
- Тестирование грамматики: `docs/03-tag-grammar.md` (строки 938-1011)
