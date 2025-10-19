# Task 05: Tag Validation System

**Статус:** Pending
**Приоритет:** High
**Зависимости:** Task 01, Task 02, Task 03, Task 04
**Оценка:** 2-3 дня

## Описание

Создать централизованную систему валидации тегов с поддержкой всех типов валидации, форматирования ошибок и частичного применения валидных тегов.

## Цели

1. Централизовать всю логику валидации
2. Обеспечить консистентное форматирование ошибок
3. Реализовать частичное применение (partial application)
4. Добавить контекстную валидацию (в зависимости от entity type)

## Технические требования

### Архитектура валидатора

```go
// internal/tag/validation/validator.go
package validation

type ValidationContext struct {
    EntityType string      // "Task", "Bug", "Epic", или "" для нового чата
    EntityID   uuid.UUID   // ID сущности (если существует)
}

type ValidationResult struct {
    Valid   bool
    Error   error
}

type Validator interface {
    Validate(tag ParsedTag, ctx ValidationContext) ValidationResult
}

type TagValidationSystem struct {
    validators map[string]Validator
    userRepo   repository.UserRepository  // для резолвинга @username
}
```

### Типы валидации

#### 1. Синтаксическая валидация

Проверка формата значения тега без обращения к БД.

```go
type SyntaxValidator struct {
    pattern *regexp.Regexp
    message string
}

func (v *SyntaxValidator) Validate(tag ParsedTag, ctx ValidationContext) ValidationResult {
    if !v.pattern.MatchString(tag.Value) {
        return ValidationResult{
            Valid: false,
            Error: fmt.Errorf("❌ %s", v.message),
        }
    }
    return ValidationResult{Valid: true}
}
```

**Примеры:**
- `#assignee` должен быть формата `@username`
- `#due` должен быть в ISO 8601 формате

#### 2. Семантическая валидация

Проверка значения из допустимого списка (enum).

```go
type EnumValidator struct {
    allowedValues []string
    caseSensitive bool
    contextDependent bool  // зависит ли список от EntityType
}

func (v *EnumValidator) Validate(tag ParsedTag, ctx ValidationContext) ValidationResult {
    allowedValues := v.getAllowedValues(ctx)

    for _, allowed := range allowedValues {
        if v.caseSensitive && tag.Value == allowed {
            return ValidationResult{Valid: true}
        }
        if !v.caseSensitive && strings.EqualFold(tag.Value, allowed) {
            return ValidationResult{Valid: true}
        }
    }

    return ValidationResult{
        Valid: false,
        Error: fmt.Errorf("❌ Invalid %s '%s'. Available: %s",
            tag.Key, tag.Value, strings.Join(allowedValues, ", ")),
    }
}

func (v *EnumValidator) getAllowedValues(ctx ValidationContext) []string {
    if !v.contextDependent {
        return v.allowedValues
    }

    // Для #status значения зависят от EntityType
    switch ctx.EntityType {
    case "Task":
        return TaskStatuses
    case "Bug":
        return BugStatuses
    case "Epic":
        return EpicStatuses
    default:
        return v.allowedValues
    }
}
```

**Примеры:**
- `#status` - context-dependent, case-sensitive
- `#priority` - статический список, case-sensitive
- `#severity` - статический список, case-sensitive

#### 3. Бизнес-валидация

Проверка с обращением к БД или бизнес-правилам.

```go
type BusinessValidator struct {
    userRepo repository.UserRepository
}

func (v *BusinessValidator) ValidateAssignee(username string) ValidationResult {
    if username == "" || username == "@none" {
        return ValidationResult{Valid: true}
    }

    // Резолвинг пользователя
    user, err := v.userRepo.FindByUsername(strings.TrimPrefix(username, "@"))
    if err != nil {
        return ValidationResult{
            Valid: false,
            Error: fmt.Errorf("❌ User %s not found", username),
        }
    }

    return ValidationResult{Valid: true}
}

func (v *BusinessValidator) ValidateSeverityApplicability(ctx ValidationContext) ValidationResult {
    if ctx.EntityType != "Bug" && ctx.EntityType != "" {
        return ValidationResult{
            Valid: false,
            Error: fmt.Errorf("⚠️ Severity is only applicable to Bugs"),
        }
    }
    return ValidationResult{Valid: true}
}
```

**Примеры:**
- `#assignee` - проверка существования пользователя
- `#severity` - проверка применимости к Bug

#### 4. Валидация пустых значений

```go
type RequiredValueValidator struct {
    tagName string
}

func (v *RequiredValueValidator) Validate(tag ParsedTag, ctx ValidationContext) ValidationResult {
    if strings.TrimSpace(tag.Value) == "" {
        return ValidationResult{
            Valid: false,
            Error: fmt.Errorf("❌ %s is required. Usage: #%s <value>",
                strings.Title(v.tagName), v.tagName),
        }
    }
    return ValidationResult{Valid: true}
}
```

### Регистрация валидаторов

```go
func NewTagValidationSystem(userRepo repository.UserRepository) *TagValidationSystem {
    system := &TagValidationSystem{
        validators: make(map[string]Validator),
        userRepo:   userRepo,
    }

    // Entity Creation Tags
    system.RegisterValidator("task", &RequiredValueValidator{tagName: "task"})
    system.RegisterValidator("bug", &RequiredValueValidator{tagName: "bug"})
    system.RegisterValidator("epic", &RequiredValueValidator{tagName: "epic"})

    // Status (context-dependent, case-sensitive)
    system.RegisterValidator("status", &EnumValidator{
        contextDependent: true,
        caseSensitive:    true,
    })

    // Priority (case-sensitive)
    system.RegisterValidator("priority", &EnumValidator{
        allowedValues: []string{"High", "Medium", "Low"},
        caseSensitive: true,
    })

    // Assignee (syntax + business)
    system.RegisterValidator("assignee", &CompositeValidator{
        validators: []Validator{
            &SyntaxValidator{
                pattern: regexp.MustCompile(`^@[\w.-]+$|^$|^@none$`),
                message: "Invalid assignee format. Use @username",
            },
            &AssigneeBusinessValidator{userRepo: userRepo},
        },
    })

    // Due date
    system.RegisterValidator("due", &DateValidator{})

    // Title
    system.RegisterValidator("title", &RequiredValueValidator{tagName: "title"})

    // Severity (enum + applicability)
    system.RegisterValidator("severity", &CompositeValidator{
        validators: []Validator{
            &EnumValidator{
                allowedValues: []string{"Critical", "Major", "Minor", "Trivial"},
                caseSensitive: true,
            },
            &SeverityApplicabilityValidator{},
        },
    })

    return system
}
```

### Composite Validator (цепочка валидаторов)

```go
type CompositeValidator struct {
    validators []Validator
}

func (cv *CompositeValidator) Validate(tag ParsedTag, ctx ValidationContext) ValidationResult {
    for _, validator := range cv.validators {
        result := validator.Validate(tag, ctx)
        if !result.Valid {
            return result
        }
    }
    return ValidationResult{Valid: true}
}
```

### Частичное применение (Partial Application)

```go
func (system *TagValidationSystem) ValidateTags(tags []ParsedTag, ctx ValidationContext) (valid []ParsedTag, errors []error) {
    for _, tag := range tags {
        validator, exists := system.validators[tag.Key]
        if !exists {
            // Неизвестный тег - игнорируется (MVP)
            continue
        }

        result := validator.Validate(tag, ctx)
        if result.Valid {
            valid = append(valid, tag)
        } else {
            errors = append(errors, result.Error)
        }
    }

    return valid, errors
}
```

**Поведение:**
- Валидные теги добавляются в результат
- Невалидные теги пропускаются с добавлением ошибки
- Все теги обрабатываются независимо

## Acceptance Criteria

- [ ] Реализован `ValidationContext` с entity type и ID
- [ ] Реализованы все типы валидаторов (Syntax, Enum, Business, Required)
- [ ] Реализован `CompositeValidator` для цепочки проверок
- [ ] Зарегистрированы все системные теги с их валидаторами
- [ ] Реализован метод `ValidateTags()` с частичным применением
- [ ] Ошибки форматируются консистентно
- [ ] Context-dependent валидация работает (например, для #status)
- [ ] Код покрыт unit-тестами

## Примеры использования

### Пример 1: Частичное применение
```go
tags := []ParsedTag{
    {Key: "status", Value: "Done"},
    {Key: "assignee", Value: "@nonexistent"},
    {Key: "priority", Value: "High"},
}

ctx := ValidationContext{EntityType: "Task"}
valid, errors := system.ValidateTags(tags, ctx)

// valid = [{Key: "status", Value: "Done"}, {Key: "priority", Value: "High"}]
// errors = ["❌ User @nonexistent not found"]
```

### Пример 2: Context-dependent валидация
```go
tag := ParsedTag{Key: "status", Value: "Fixed"}

// В контексте Task
ctx1 := ValidationContext{EntityType: "Task"}
result1 := system.validators["status"].Validate(tag, ctx1)
// result1.Valid = false
// result1.Error = "❌ Invalid status 'Fixed' for Task. Available: To Do, In Progress, Done"

// В контексте Bug
ctx2 := ValidationContext{EntityType: "Bug"}
result2 := system.validators["status"].Validate(tag, ctx2)
// result2.Valid = true
```

## Тесты

```go
func TestValidationSystem(t *testing.T) {
    userRepo := mock.NewUserRepository()
    userRepo.AddUser("alex", uuid.New())

    system := NewTagValidationSystem(userRepo)

    t.Run("partial application", func(t *testing.T) {
        tags := []ParsedTag{
            {Key: "status", Value: "Done"},
            {Key: "assignee", Value: "@nonexistent"},
            {Key: "priority", Value: "High"},
        }

        ctx := ValidationContext{EntityType: "Task"}
        valid, errors := system.ValidateTags(tags, ctx)

        assert.Len(t, valid, 2)
        assert.Len(t, errors, 1)
        assert.Contains(t, errors[0].Error(), "not found")
    })

    t.Run("context-dependent status", func(t *testing.T) {
        tag := ParsedTag{Key: "status", Value: "Fixed"}

        // Task context
        ctx1 := ValidationContext{EntityType: "Task"}
        result1 := system.validators["status"].Validate(tag, ctx1)
        assert.False(t, result1.Valid)

        // Bug context
        ctx2 := ValidationContext{EntityType: "Bug"}
        result2 := system.validators["status"].Validate(tag, ctx2)
        assert.True(t, result2.Valid)
    })
}
```

## Файловая структура

```
internal/tag/validation/
├── validator.go              # Интерфейсы и ValidationSystem
├── syntax_validator.go       # Синтаксическая валидация
├── enum_validator.go         # Enum валидация
├── business_validator.go     # Бизнес-валидация
├── required_validator.go     # Проверка обязательных полей
├── composite_validator.go    # Цепочка валидаторов
├── context.go                # ValidationContext
└── validator_test.go         # Тесты
```

## Ссылки

- Стратегия валидации: `docs/03-tag-grammar.md` (строки 355-378)
- Типы ошибок: `docs/03-tag-grammar.md` (строки 380-427)
- Частичное применение: `docs/03-tag-grammar.md` (строки 359-378, 794-809)
