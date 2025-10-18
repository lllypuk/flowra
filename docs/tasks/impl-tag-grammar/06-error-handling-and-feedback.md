# Task 06: Error Handling and User Feedback

**Статус:** Pending
**Приоритет:** High
**Зависимости:** Task 05
**Оценка:** 2 дня

## Описание

Реализовать систему обработки ошибок валидации и формирования понятных сообщений об ошибках для пользователя. Обеспечить частичное применение валидных тегов с сохранением всех сообщений в истории чата.

## Цели

1. Создать систему форматирования ошибок
2. Реализовать генерацию bot-ответов с результатами применения тегов
3. Обеспечить сохранение сообщений даже при наличии ошибок
4. Создать понятные и консистентные сообщения для пользователя

## Технические требования

### Типы ошибок

Согласно спецификации (строки 380-427), есть 4 типа ошибок:

#### 1. Синтаксическая ошибка
```
#assignee alex (без @)
→ "❌ Invalid assignee format. Use @username"

#due 20-10-2025
→ "❌ Invalid date format. Use ISO 8601: YYYY-MM-DD"
```

#### 2. Семантическая ошибка
```
#status Completed
→ "❌ Invalid status 'Completed' for Task. Available: To Do, In Progress, Done"

#priority high
→ "❌ Invalid priority 'high'. Available: High, Medium, Low"
```

#### 3. Бизнес-ошибка
```
#assignee @nonexistent
→ "❌ User @nonexistent not found"

#severity Critical (в таске)
→ "⚠️ Severity is only applicable to Bugs"
```

#### 4. Пустое значение для обязательного поля
```
#task
→ "❌ Task title is required. Usage: #task <title>"

#title
→ "❌ Title cannot be empty"
```

### Структура результата обработки

```go
// internal/tag/processor.go
package tag

type ProcessingResult struct {
    OriginalMessage string
    AppliedTags     []TagApplication
    Errors          []TagError
    BotResponse     string
}

type TagApplication struct {
    TagKey   string
    TagValue string
    Command  Command
    Success  bool
}

type TagError struct {
    TagKey   string
    TagValue string
    Error    error
    Severity ErrorSeverity
}

type ErrorSeverity int

const (
    ErrorSeverityError   ErrorSeverity = iota  // ❌ Применение невозможно
    ErrorSeverityWarning                       // ⚠️ Применено с предупреждением
)
```

### Генерация Bot Response

```go
func (pr *ProcessingResult) GenerateBotResponse() string {
    if len(pr.AppliedTags) == 0 && len(pr.Errors) == 0 {
        return "" // Нет тегов - нет ответа
    }

    var lines []string

    // Успешно применённые теги
    for _, applied := range pr.AppliedTags {
        if applied.Success {
            lines = append(lines, formatSuccess(applied))
        }
    }

    // Ошибки
    for _, err := range pr.Errors {
        lines = append(lines, formatError(err))
    }

    return strings.Join(lines, "\n")
}

func formatSuccess(applied TagApplication) string {
    switch applied.Command.(type) {
    case CreateTaskCommand:
        return fmt.Sprintf("✅ Task created: %s", applied.TagValue)
    case ChangeStatusCommand:
        return fmt.Sprintf("✅ Status changed to %s", applied.TagValue)
    case AssignUserCommand:
        if applied.TagValue == "" || applied.TagValue == "@none" {
            return "✅ Assignee removed"
        }
        return fmt.Sprintf("✅ Assigned to: %s", applied.TagValue)
    case ChangePriorityCommand:
        return fmt.Sprintf("✅ Priority changed to %s", applied.TagValue)
    case SetDueDateCommand:
        if applied.TagValue == "" {
            return "✅ Due date removed"
        }
        return fmt.Sprintf("✅ Due date set to %s", applied.TagValue)
    case ChangeTitleCommand:
        return fmt.Sprintf("✅ Title changed to: %s", applied.TagValue)
    case SetSeverityCommand:
        return fmt.Sprintf("✅ Severity set to %s", applied.TagValue)
    default:
        return "✅ Applied"
    }
}

func formatError(err TagError) string {
    prefix := "❌"
    if err.Severity == ErrorSeverityWarning {
        prefix = "⚠️"
    }

    return fmt.Sprintf("%s %s", prefix, err.Error.Error())
}
```

### Формат сообщений об ошибках

Согласно спецификации (строки 428-439):

```
Шаблон:
"❌ <описание ошибки>. <подсказка или доступные значения>"

Примеры:
"❌ Invalid status 'done'. Available: To Do, In Progress, Done"
"❌ User @bob not found"
"❌ Invalid date format. Use ISO 8601: YYYY-MM-DD"
"❌ Task title is required. Usage: #task <title>"
```

### Сохранение сообщений с ошибками

**Правило:** Сообщение ВСЕГДА сохраняется в чат, даже если все теги невалидны.

```go
// internal/tag/handler.go
package tag

func (h *TagHandler) HandleMessageWithTags(chatID uuid.UUID, authorID uuid.UUID, content string) error {
    // 1. Парсинг тегов
    parseResult := h.parser.Parse(content)

    // 2. Валидация
    ctx := h.getValidationContext(chatID)
    validTags, validationErrors := h.validator.ValidateTags(parseResult.Tags, ctx)

    // 3. ВСЕГДА сохраняем сообщение в чат (с тегами или без)
    message := domain.Message{
        ID:        uuid.New(),
        ChatID:    chatID,
        AuthorID:  authorID,
        Content:   content,
        CreatedAt: time.Now(),
    }
    if err := h.messageRepo.Save(message); err != nil {
        return err
    }

    // 4. Применяем валидные теги
    result := h.processor.ProcessTags(chatID, validTags)

    // 5. Добавляем ошибки валидации
    result.Errors = append(result.Errors, validationErrors...)

    // 6. Генерируем и отправляем bot response (если есть что сказать)
    if botResponse := result.GenerateBotResponse(); botResponse != "" {
        h.sendBotResponse(chatID, botResponse)
    }

    return nil
}
```

## Acceptance Criteria

- [ ] Реализована структура `ProcessingResult`
- [ ] Реализован метод `GenerateBotResponse()`
- [ ] Реализованы функции форматирования успехов и ошибок
- [ ] Все типы ошибок имеют правильный формат сообщений
- [ ] Сообщения ВСЕГДА сохраняются, даже с ошибками
- [ ] Bot response генерируется только при наличии тегов
- [ ] Частичное применение работает корректно
- [ ] Код покрыт unit-тестами

## Примеры использования

### Пример 1: Частичное применение с ошибкой
```
User: "#status Done #assignee @nonexistent #priority High"

Результат:
✅ status → "Done" (применён)
❌ assignee → ошибка "User @nonexistent not found" (не применён)
✅ priority → "High" (применён)

Bot response:
"✅ Status changed to Done
 ✅ Priority changed to High
 ❌ User @nonexistent not found"
```

### Пример 2: Все теги невалидны, но сообщение сохранено
```
User: "#status Dne #assignee @nobody"

Результат:
→ Сообщение сохранено в Messages
→ Теги не применены

Bot response:
"❌ Invalid status 'Dne' for Task. Available: To Do, In Progress, Done
 ❌ User @nobody not found"

→ Пользователь видит своё сообщение в истории чата
```

### Пример 3: Создание задачи с атрибутами
```
User: "#task Реализовать OAuth #priority High #assignee @alex"

Результат:
✅ Создан Task
✅ Priority = "High"
✅ Assignee = @alex

Bot response:
"✅ Task created: Реализовать OAuth
 ✅ Priority changed to High
 ✅ Assigned to: @alex"
```

### Пример 4: Только обсуждение, без тегов
```
User: "Закончил работу над задачей"

Результат:
→ Сообщение сохранено
→ Нет тегов
→ Нет bot response (пустой ответ)
```

## Тесты

```go
func TestGenerateBotResponse(t *testing.T) {
    tests := []struct {
        name     string
        result   ProcessingResult
        expected string
    }{
        {
            name: "partial application",
            result: ProcessingResult{
                AppliedTags: []TagApplication{
                    {TagKey: "status", TagValue: "Done", Success: true},
                    {TagKey: "priority", TagValue: "High", Success: true},
                },
                Errors: []TagError{
                    {TagKey: "assignee", Error: errors.New("User @nonexistent not found")},
                },
            },
            expected: "✅ Status changed to Done\n✅ Priority changed to High\n❌ User @nonexistent not found",
        },
        {
            name: "no tags - no response",
            result: ProcessingResult{
                AppliedTags: []TagApplication{},
                Errors:      []TagError{},
            },
            expected: "",
        },
        {
            name: "all errors",
            result: ProcessingResult{
                Errors: []TagError{
                    {TagKey: "status", Error: errors.New("Invalid status 'done'. Available: To Do, In Progress, Done")},
                    {TagKey: "assignee", Error: errors.New("User @nobody not found")},
                },
            },
            expected: "❌ Invalid status 'done'. Available: To Do, In Progress, Done\n❌ User @nobody not found",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            response := tt.result.GenerateBotResponse()
            assert.Equal(t, tt.expected, response)
        })
    }
}

func TestMessageAlwaysSaved(t *testing.T) {
    handler := NewTagHandler(...)

    // Сообщение с невалидными тегами
    err := handler.HandleMessageWithTags(chatID, authorID, "#status InvalidStatus")

    assert.NoError(t, err)

    // Проверяем, что сообщение сохранено
    messages := messageRepo.FindByChatID(chatID)
    assert.Len(t, messages, 1)
    assert.Equal(t, "#status InvalidStatus", messages[0].Content)
}
```

## Файловая структура

```
internal/tag/
├── result.go           # ProcessingResult, TagApplication, TagError
├── formatter.go        # formatSuccess, formatError
├── handler.go          # HandleMessageWithTags
└── handler_test.go     # Тесты
```

## Ссылки

- Типы ошибок: `docs/03-tag-grammar.md` (строки 380-427)
- Формат сообщений: `docs/03-tag-grammar.md` (строки 428-439)
- Сохранение сообщений: `docs/03-tag-grammar.md` (строки 441-457)
- Частичное применение: `docs/03-tag-grammar.md` (строки 359-378)
- Примеры: `docs/03-tag-grammar.md` (строки 794-809)
