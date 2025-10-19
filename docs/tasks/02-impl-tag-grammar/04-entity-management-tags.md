# Task 04: Entity Management Tags Implementation

**Статус:** Pending
**Приоритет:** High
**Зависимости:** Task 01, Task 02, Task 03
**Оценка:** 4-5 дней

## Описание

Реализовать обработку тегов управления сущностями: `#status`, `#assignee`, `#priority`, `#due`, `#title`, `#severity`. Эти теги изменяют состояние существующих задач.

## Цели

1. Реализовать команды управления сущностями
2. Реализовать специфичную валидацию для каждого тега
3. Обработать резолвинг пользователей для `#assignee`
4. Обработать парсинг дат для `#due`
5. Обработать CASE-SENSITIVE enum значения

## Технические требования

### Entity Management Tags

```go
#status <value>     — Изменить статус
#assignee @user     — Назначить исполнителя
#priority <value>   — Установить приоритет
#due <date>         — Установить deadline
#title <text>       — Изменить название задачи
#severity <value>   — Серьёзность бага (Bug-specific)
```

### Команды

```go
// internal/tag/commands.go

type ChangeStatusCommand struct {
    ChatID uuid.UUID
    Status string
}

type AssignUserCommand struct {
    ChatID   uuid.UUID
    Username string // @alex
    UserID   *uuid.UUID // резолвленный ID (может быть nil при снятии)
}

type ChangePriorityCommand struct {
    ChatID   uuid.UUID
    Priority string
}

type SetDueDateCommand struct {
    ChatID  uuid.UUID
    DueDate *time.Time // nil означает снять due date
}

type ChangeTitleCommand struct {
    ChatID uuid.UUID
    Title  string
}

type SetSeverityCommand struct {
    ChatID   uuid.UUID
    Severity string
}
```

### Валидация #status

```go
// internal/tag/validator.go

var (
    TaskStatuses = []string{"To Do", "In Progress", "Done"}
    BugStatuses  = []string{"New", "Investigating", "Fixed", "Verified"}
    EpicStatuses = []string{"Planned", "In Progress", "Completed"}
)

func ValidateStatus(entityType string, status string) error {
    var allowedStatuses []string

    switch entityType {
    case "Task":
        allowedStatuses = TaskStatuses
    case "Bug":
        allowedStatuses = BugStatuses
    case "Epic":
        allowedStatuses = EpicStatuses
    default:
        return fmt.Errorf("unknown entity type: %s", entityType)
    }

    for _, allowed := range allowedStatuses {
        if status == allowed {
            return nil
        }
    }

    return fmt.Errorf("❌ Invalid status '%s' for %s. Available: %s",
        status, entityType, strings.Join(allowedStatuses, ", "))
}
```

**ВАЖНО:** Статус CASE-SENSITIVE:
- ✅ `#status Done`
- ❌ `#status done` → ошибка
- ❌ `#status DONE` → ошибка

### Валидация #assignee

```go
func ValidateAssignee(value string) error {
    if value == "" || value == "@none" {
        return nil // Снятие assignee
    }

    if !strings.HasPrefix(value, "@") {
        return fmt.Errorf("❌ Invalid assignee format. Use @username")
    }

    username := value[1:]
    if username == "" {
        return fmt.Errorf("❌ Invalid assignee format. Use @username")
    }

    return nil
}
```

**Поведение:**
- `#assignee @alex` → резолвит @alex в UserID
- `#assignee @nonexistent` → ошибка "User @nonexistent not found"
- `#assignee @none` → убирает assignee (assignee = null)
- `#assignee` (пустое) → убирает assignee
- `#assignee alex` (без @) → ошибка формата

### Валидация #priority

```go
var Priorities = []string{"High", "Medium", "Low"}

func ValidatePriority(priority string) error {
    for _, allowed := range Priorities {
        if priority == allowed {
            return nil
        }
    }

    return fmt.Errorf("❌ Invalid priority '%s'. Available: %s",
        priority, strings.Join(Priorities, ", "))
}
```

**CASE-SENSITIVE:**
- ✅ `#priority High`
- ❌ `#priority high` → ошибка
- ❌ `#priority Urgent` → ошибка

### Валидация #due

```go
func ValidateDueDate(dateStr string) (*time.Time, error) {
    if dateStr == "" {
        return nil, nil // Снятие due date
    }

    // Поддерживаемые форматы (MVP)
    formats := []string{
        "2006-01-02",                 // YYYY-MM-DD
        "2006-01-02T15:04",           // YYYY-MM-DDTHH:MM
        "2006-01-02T15:04:05",        // YYYY-MM-DDTHH:MM:SS
        time.RFC3339,                 // с timezone
    }

    var lastErr error
    for _, format := range formats {
        if t, err := time.Parse(format, dateStr); err == nil {
            return &t, nil
        } else {
            lastErr = err
        }
    }

    return nil, fmt.Errorf("❌ Invalid date format. Use ISO 8601: YYYY-MM-DD")
}
```

**Примеры:**
- ✅ `#due 2025-10-20`
- ✅ `#due 2025-10-20T15:30`
- ✅ `#due 2025-10-20T15:30:00Z`
- ❌ `#due 20-10-2025` → ошибка
- ❌ `#due tomorrow` → ошибка (V2 feature)
- `#due` (пустое) → убирает due_date

### Валидация #title

```go
func ValidateTitle(title string) error {
    trimmed := strings.TrimSpace(title)
    if trimmed == "" {
        return fmt.Errorf("❌ Title cannot be empty")
    }
    return nil
}
```

### Валидация #severity (Bug-specific)

```go
var Severities = []string{"Critical", "Major", "Minor", "Trivial"}

func ValidateSeverity(severity string) error {
    for _, allowed := range Severities {
        if severity == allowed {
            return nil
        }
    }

    return fmt.Errorf("❌ Invalid severity '%s'. Available: %s",
        severity, strings.Join(Severities, ", "))
}
```

**Применимость:** Только для Bug
- В Bug: `#severity Critical` → применяется
- В Task/Epic: `#severity Critical` → предупреждение "Severity is only applicable to Bugs"

## Acceptance Criteria

- [ ] Реализованы все команды управления
- [ ] Реализована валидация для каждого тега
- [ ] #status валидирует значения в зависимости от типа сущности (Task/Bug/Epic)
- [ ] #status проверяет CASE-SENSITIVE
- [ ] #assignee проверяет формат @username
- [ ] #assignee обрабатывает снятие assignee (@none или пустое)
- [ ] #priority валидирует CASE-SENSITIVE значения
- [ ] #due парсит ISO 8601 форматы
- [ ] #due обрабатывает снятие due_date (пустое значение)
- [ ] #title проверяет на пустоту
- [ ] #severity валидирует значения
- [ ] #severity показывает предупреждение для non-Bug сущностей
- [ ] Код покрыт unit-тестами

## Примеры использования

### Пример 1: Изменение статуса
```
Input: "#status In Progress"
Entity: Task
Output:
  Command: ChangeStatusCommand{Status: "In Progress"}
  Errors: []
```

### Пример 2: Неправильный регистр
```
Input: "#status done"
Entity: Task
Output:
  Commands: []
  Errors: ["❌ Invalid status 'done' for Task. Available: To Do, In Progress, Done"]
```

### Пример 3: Назначение пользователя
```
Input: "#assignee @alex"
Output:
  Command: AssignUserCommand{Username: "@alex", UserID: <resolved-uuid>}
  Errors: []
```

### Пример 4: Несуществующий пользователь
```
Input: "#assignee @nonexistent"
Output:
  Commands: []
  Errors: ["❌ User @nonexistent not found"]
```

### Пример 5: Установка дедлайна
```
Input: "#due 2025-10-20"
Output:
  Command: SetDueDateCommand{DueDate: 2025-10-20T00:00:00}
  Errors: []
```

### Пример 6: Неправильный формат даты
```
Input: "#due 20-10-2025"
Output:
  Commands: []
  Errors: ["❌ Invalid date format. Use ISO 8601: YYYY-MM-DD"]
```

## Тесты

```go
func TestValidateStatus(t *testing.T) {
    tests := []struct {
        entityType string
        status     string
        wantErr    bool
    }{
        {"Task", "Done", false},
        {"Task", "done", true},        // case-sensitive
        {"Task", "Completed", true},   // not in list
        {"Bug", "Fixed", false},
        {"Bug", "Done", true},         // wrong entity
        {"Epic", "Completed", false},
    }

    for _, tt := range tests {
        err := ValidateStatus(tt.entityType, tt.status)
        if tt.wantErr {
            assert.Error(t, err)
        } else {
            assert.NoError(t, err)
        }
    }
}

func TestValidateAssignee(t *testing.T) {
    tests := []struct {
        value   string
        wantErr bool
    }{
        {"@alex", false},
        {"@none", false},
        {"", false},              // empty = remove
        {"alex", true},           // missing @
        {"@", true},              // missing username
        {"@user.name", false},    // valid username chars
    }

    for _, tt := range tests {
        err := ValidateAssignee(tt.value)
        if tt.wantErr {
            assert.Error(t, err)
        } else {
            assert.NoError(t, err)
        }
    }
}

func TestValidateDueDate(t *testing.T) {
    tests := []struct {
        input   string
        wantNil bool
        wantErr bool
    }{
        {"2025-10-20", false, false},
        {"2025-10-20T15:30", false, false},
        {"2025-10-20T15:30:00Z", false, false},
        {"", true, false},              // empty = remove
        {"20-10-2025", false, true},    // wrong format
        {"tomorrow", false, true},      // natural language (V2)
    }

    for _, tt := range tests {
        result, err := ValidateDueDate(tt.input)
        if tt.wantErr {
            assert.Error(t, err)
        } else {
            assert.NoError(t, err)
            if tt.wantNil {
                assert.Nil(t, result)
            } else {
                assert.NotNil(t, result)
            }
        }
    }
}
```

## Файловая структура

```
internal/tag/
├── commands.go              # Определения команд
├── validator.go             # Валидация всех тегов
├── validator_status.go      # Специфичная валидация для status
├── validator_assignee.go    # Резолвинг и валидация assignee
├── validator_date.go        # Парсинг дат
├── validator_test.go        # Тесты
└── constants.go             # Константы (TaskStatuses, Priorities, etc.)
```

## Ссылки

- Entity Management Tags: `docs/03-tag-grammar.md` (строки 157-302)
- Bug-Specific Tags: `docs/03-tag-grammar.md` (строки 303-323)
- Примеры валидации: `docs/03-tag-grammar.md` (строки 380-416)
- Сценарии использования: `docs/03-tag-grammar.md` (строки 778-809)
