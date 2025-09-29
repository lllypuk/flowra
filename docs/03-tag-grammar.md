# Tag Grammar Specification

**Дата:** 2025-09-30
**Статус:** Архитектурное решение

## Обзор

Теги — основной механизм управления задачами через чат. Все управляющие воздействия (изменение статуса, назначение исполнителя, установка приоритета) происходят через теги в сообщениях.

## Принципы дизайна

- **Простота** — легко запомнить базовый синтаксис
- **Однозначность** — чёткий парсинг без неоднозначностей
- **Регистрозависимость** — `#status` ≠ `#Status` (исключает случайные срабатывания)
- **Частичное применение** — валидные теги применяются даже при ошибках в других
- **Известные теги** — парсятся только зарегистрированные теги (# в обычном тексте игнорируется)

## Позиция тегов в сообщении

Теги могут находиться:
1. **В начале сообщения** (первая строка)
2. **На отдельной строке** после обычного текста

### ✅ Валидные примеры

```
Пример 1: Только теги
#status Done

Пример 2: Теги в начале + текст
#status Done #assignee @alex
Закончил работу, готово к проверке

Пример 3: Текст + теги на отдельной строке
Закончил работу над задачей
#status Done
#assignee @alex

Пример 4: Множественные теги в начале
#task Реализовать авторизацию #priority High
```

### ❌ Невалидные примеры

```
Пример 1: Теги в середине строки (не парсятся)
Закончил работу #status Done — отправляю на проверку
→ "#status Done" не распознается как тег

Пример 2: Теги перемешаны с текстом
#status Done какой-то текст #assignee @alex
→ Только "#status Done" распознается, "#assignee" игнорируется
```

### Правила парсинга позиции

```
1. Строка начинается с # → парсить теги до конца строки или до текста
2. Строка не начинается с # → обычный текст, теги не парсятся
3. После обычного текста новая строка с # → парсить теги
4. Пустые строки игнорируются
```

## Формальная грамматика (EBNF-подобная)

```ebnf
message         = tag_line* text_line*
tag_line        = tag+ [text]
text_line       = любой текст не начинающийся с #

tag             = "#" tag_name [whitespace tag_value]
tag_name        = lowercase_identifier
tag_value       = value_text | user_mention | iso_date

lowercase_identifier = [a-z][a-z0-9_]*
user_mention    = "@" username
username        = [a-zA-Z0-9._-]+
iso_date        = YYYY-MM-DD ["T" HH:MM[:SS][timezone]]
value_text      = текст до следующего # или конца строки (trimmed)

whitespace      = пробел или табуляция (один или более)
```

### Примеры парсинга

```
Вход: "#status Done #assignee @alex"
→ Tags: [
    {key: "status", value: "Done"},
    {key: "assignee", value: "@alex"}
  ]
→ PlainText: ""

Вход: "#task Реализовать авторизацию #priority High"
→ Tags: [
    {key: "task", value: "Реализовать авторизацию"},
    {key: "priority", value: "High"}
  ]
→ PlainText: ""

Вход: "Закончил работу
       #status Done"
→ Tags: [{key: "status", value: "Done"}]
→ PlainText: "Закончил работу"

Вход: "#bug Ошибка в логине
       Воспроизводится на Chrome"
→ Tags: [{key: "bug", value: "Ошибка в логине"}]
→ PlainText: "Воспроизводится на Chrome"
```

### Парсинг значений тегов

**Правило:** Значение тега — это всё между именем тега и следующим # (или концом строки), с обрезанными пробелами.

```
"#status In Progress #assignee @alex"
            ↑________↑
            value = "In Progress"

"#task Реализовать функцию авторизации"
       ↑______________________________↑
       value = "Реализовать функцию авторизации"

"#priority High"
           ↑__↑
           value = "High"
```

## Системные теги (Reserved Keywords)

### Entity Creation Tags

```
#task <title>       — Создать Task
#bug <title>        — Создать Bug
#epic <title>       — Создать Epic
```

**Поведение:**
- Если написать в обычном чате → создаётся новая задача-чат
- Если написать в typed чате (Task/Bug/Epic) → переопределяет тип
- Title обязателен (валидация: не пустой после trim)

**Примеры:**
```
#task Реализовать авторизацию
→ Создаёт Task с title="Реализовать авторизацию"

#bug Ошибка при логине #severity Critical
→ Создаёт Bug с title="Ошибка при логине", severity="Critical"

#task
→ ❌ Ошибка: "Task title is required. Usage: #task <title>"
```

### Entity Management Tags

```
#status <value>     — Изменить статус
#assignee @user     — Назначить исполнителя
#priority <value>   — Установить приоритет
#due <date>         — Установить deadline
#title <text>       — Изменить название задачи
```

**Детали:**

#### #status

**Допустимые значения (зависят от типа сущности):**

```
Task:  "To Do", "In Progress", "Done"
Bug:   "New", "Investigating", "Fixed", "Verified"
Epic:  "Planned", "In Progress", "Completed"
```

**Регистрозависимость:** Да, значения CASE-SENSITIVE
```
✅ #status Done
❌ #status done       → ошибка: "Invalid status 'done'. Available: To Do, In Progress, Done"
❌ #status DONE       → ошибка
```

**Валидация:** Значение должно быть в списке допустимых для типа сущности.

**Примеры:**
```
#status In Progress
→ ✅ Меняет статус на "In Progress"

#status Completed
→ ❌ Для Task недоступен. Ошибка: "Invalid status 'Completed' for Task. Available: To Do, In Progress, Done"
```

#### #assignee

**Формат:** `@username`

**Поведение:**
- Username резолвится в UserID через UserRepository
- Если пользователь не найден → ошибка, assignee не меняется
- Сообщение с тегом всё равно сохраняется в чат

**Примеры:**
```
#assignee @alex
→ ✅ Резолвит @alex → UUID, назначает исполнителем

#assignee @nonexistent
→ ❌ Ошибка: "User @nonexistent not found"
→ Сообщение сохраняется в истории чата
→ Assignee не меняется

#assignee alex
→ ❌ Ошибка: "Invalid assignee format. Use @username"
```

**Снятие assignee:**
```
#assignee @none
→ Убирает assignee (assignee = null)

или просто:
#assignee
→ Убирает assignee (пустое значение = снять)
```

#### #priority

**Допустимые значения:** `High`, `Medium`, `Low`

**Регистрозависимость:** Да (CASE-SENSITIVE)

**Примеры:**
```
#priority High
→ ✅ Устанавливает priority="High"

#priority high
→ ❌ Ошибка: "Invalid priority 'high'. Available: High, Medium, Low"

#priority Urgent
→ ❌ Ошибка: "Invalid priority 'Urgent'. Available: High, Medium, Low"
```

#### #due

**Формат:** ISO 8601 date/datetime

**Допустимые форматы (MVP):**
```
YYYY-MM-DD              → 2025-10-20
YYYY-MM-DDTHH:MM        → 2025-10-20T15:30
YYYY-MM-DDTHH:MM:SS     → 2025-10-20T15:30:00
YYYY-MM-DDTHH:MM:SSZ    → 2025-10-20T15:30:00Z (UTC)
YYYY-MM-DDTHH:MM:SS+03:00 → с таймзоной
```

**Timezone:**
- Если не указан → используется локальная таймзона пользователя
- Z → UTC
- +HH:MM / -HH:MM → explicit timezone

**Валидация:**
- Дата должна парситься корректно
- Дата в прошлом — допустима (может быть желаемый дедлайн был вчера, задача просрочена)

**Примеры:**
```
#due 2025-10-20
→ ✅ Устанавливает due_date=2025-10-20T00:00:00 (локальная таймзона)

#due 2025-10-20T15:30
→ ✅ Устанавливает due_date=2025-10-20T15:30:00

#due tomorrow
→ ❌ Ошибка (V2): "Natural language dates not supported yet. Use ISO format: YYYY-MM-DD"

#due 20-10-2025
→ ❌ Ошибка: "Invalid date format. Use ISO 8601: YYYY-MM-DD"

#due
→ Убирает due_date (пустое значение = снять)
```

#### #title

**Формат:** Произвольный текст

**Поведение:** Изменяет название существующей задачи

**Примеры:**
```
#title Новое название задачи
→ ✅ Меняет title на "Новое название задачи"

#title
→ ❌ Ошибка: "Title cannot be empty"
```

### Bug-Specific Tags

```
#severity <value>   — Серьёзность бага
```

**Допустимые значения:** `Critical`, `Major`, `Minor`, `Trivial`

**Применимость:** Только для Bug (для Task/Epic игнорируется с предупреждением)

**Примеры:**
```
В баге:
#severity Critical
→ ✅ Устанавливает severity="Critical"

В таске:
#severity Critical
→ ⚠️ Предупреждение: "Severity is only applicable to Bugs"
```

### V2 Tags (отложено)

```
#parent #<id>       — Связать с родительской задачей
#blocks #<id>       — Блокирует другую задачу
#relates #<id>      — Связано с другой задачей
#sprint <name>      — Назначить в спринт
#estimate <time>    — Оценка времени (5d, 8h, 2w)
#label <value>      — Метка/категория
```

## Custom Tags (V2)

**Статус:** Отложено на V2

**Концепция:** Любой тег, не являющийся системным, может быть зарегистрирован как custom.

**Примеры:**
```
#component Auth
#environment Production
#customer ACME-Corp
#version 2.1.0
```

**Хранение:** CustomFields map[string]string в EntityState

**Регистрация:** Админ проекта/команды может зарегистрировать custom тег через UI

**MVP:** Custom tags не поддерживаются, неизвестные теги игнорируются

## Валидация и обработка ошибок

### Стратегия валидации

**Подход:** Частичное применение (partial application)

При наличии нескольких тегов в одном сообщении, каждый валидируется независимо:
- ✅ Валидные теги применяются
- ❌ Невалидные теги игнорируются с сообщением об ошибке

**Пример:**
```
User: "#status Done #assignee @nonexistent #priority High"

Результат:
✅ status → "Done" (применён)
❌ assignee → ошибка "User @nonexistent not found" (не применён)
✅ priority → "High" (применён)

Bot ответ в чат:
"✅ Status changed to Done
 ✅ Priority changed to High
 ❌ User @nonexistent not found"
```

### Типы ошибок валидации

#### 1. Синтаксическая ошибка

**Причина:** Неправильный формат тега

```
#assignee alex (без @)
→ "❌ Invalid assignee format. Use @username"

#due 20-10-2025
→ "❌ Invalid date format. Use ISO 8601: YYYY-MM-DD"
```

#### 2. Семантическая ошибка

**Причина:** Значение не из допустимых

```
#status Completed
→ "❌ Invalid status 'Completed' for Task. Available: To Do, In Progress, Done"

#priority high
→ "❌ Invalid priority 'high'. Available: High, Medium, Low"
```

#### 3. Бизнес-ошибка

**Причина:** Нарушение бизнес-правил

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

### Формат сообщений об ошибках

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

**Причины:**
- История обсуждений не теряется
- Пользователь видит, что написал
- Можно исправить и отправить заново

**Пример:**
```
User: "#status Dne #assignee @nobody"
→ Сообщение сохранено в Messages
→ Теги не применены
→ Bot отвечает с ошибками
→ Пользователь видит своё сообщение в истории чата
```

## Парсинг — алгоритм

### Псевдокод парсера

```go
type TagParser struct {
    knownTags map[string]TagDefinition
}

type TagDefinition struct {
    Name            string
    RequiresValue   bool
    ValueType       ValueType // String, Username, Date, Enum
    AllowedValues   []string  // для Enum
    Validator       func(value string) error
}

type ParseResult struct {
    Tags      []ParsedTag
    PlainText string
}

type ParsedTag struct {
    Key   string
    Value string
}

func (p *TagParser) Parse(content string) ParseResult {
    lines := strings.Split(content, "\n")
    result := ParseResult{Tags: []ParsedTag{}}

    inTagMode := true

    for i, line := range lines {
        trimmed := strings.TrimSpace(line)

        // Пустые строки пропускаем
        if trimmed == "" {
            continue
        }

        // Проверяем, начинается ли строка с #
        if strings.HasPrefix(trimmed, "#") && (inTagMode || i > 0) {
            // Парсим теги на этой строке
            tags, remaining := p.parseTagsFromLine(trimmed)
            result.Tags = append(result.Tags, tags...)

            // Если есть текст после тегов на той же строке
            if remaining != "" {
                result.PlainText += remaining + "\n"
                inTagMode = false
            }
        } else {
            // Обычный текст
            result.PlainText += line + "\n"
            inTagMode = false
        }
    }

    result.PlainText = strings.TrimSpace(result.PlainText)
    return result
}

func (p *TagParser) parseTagsFromLine(line string) ([]ParsedTag, string) {
    tags := []ParsedTag{}
    remaining := line

    for strings.HasPrefix(remaining, "#") {
        tag, rest := p.parseOneTag(remaining)

        if tag != nil {
            // Проверяем, является ли тег известным
            if p.isKnownTag(tag.Key) {
                tags = append(tags, *tag)
            }
            // Неизвестные теги игнорируются (MVP)
        }

        remaining = strings.TrimSpace(rest)

        // Если остаток не начинается с #, это текст
        if remaining != "" && !strings.HasPrefix(remaining, "#") {
            break
        }
    }

    return tags, remaining
}

func (p *TagParser) parseOneTag(s string) (*ParsedTag, string) {
    // s начинается с #
    // Находим конец имени тега (до пробела)

    withoutHash := s[1:] // убираем #

    parts := strings.SplitN(withoutHash, " ", 2)
    tagName := parts[0]

    // Если тег в конце строки или следующий символ #
    if len(parts) == 1 {
        return &ParsedTag{Key: tagName, Value: ""}, ""
    }

    rest := parts[1]

    // Если сразу после пробела идёт #, то значение пустое
    if strings.HasPrefix(rest, "#") {
        return &ParsedTag{Key: tagName, Value: ""}, rest
    }

    // Ищем следующий тег
    nextHashIndex := strings.Index(rest, " #")

    var value string
    var remaining string

    if nextHashIndex == -1 {
        // Нет следующего тега, всё остальное — значение
        value = strings.TrimSpace(rest)
        remaining = ""
    } else {
        // Значение до следующего тега
        value = strings.TrimSpace(rest[:nextHashIndex])
        remaining = strings.TrimSpace(rest[nextHashIndex+1:]) // +1 чтобы убрать пробел
    }

    return &ParsedTag{Key: tagName, Value: value}, remaining
}

func (p *TagParser) isKnownTag(name string) bool {
    _, exists := p.knownTags[name]
    return exists
}
```

### Регистрация системных тегов

```go
func NewTagParser() *TagParser {
    return &TagParser{
        knownTags: map[string]TagDefinition{
            // Entity creation
            "task": {
                Name:          "task",
                RequiresValue: true,
                ValueType:     String,
            },
            "bug": {
                Name:          "bug",
                RequiresValue: true,
                ValueType:     String,
            },
            "epic": {
                Name:          "epic",
                RequiresValue: true,
                ValueType:     String,
            },

            // Entity management
            "status": {
                Name:          "status",
                RequiresValue: true,
                ValueType:     Enum,
                AllowedValues: []string{"To Do", "In Progress", "Done"}, // динамически зависит от типа
            },
            "assignee": {
                Name:          "assignee",
                RequiresValue: false, // может быть пустым (снять assignee)
                ValueType:     Username,
                Validator:     validateUsername,
            },
            "priority": {
                Name:          "priority",
                RequiresValue: true,
                ValueType:     Enum,
                AllowedValues: []string{"High", "Medium", "Low"},
            },
            "due": {
                Name:          "due",
                RequiresValue: false, // может быть пустым (снять due_date)
                ValueType:     Date,
                Validator:     validateISODate,
            },
            "title": {
                Name:          "title",
                RequiresValue: true,
                ValueType:     String,
            },

            // Bug-specific
            "severity": {
                Name:          "severity",
                RequiresValue: true,
                ValueType:     Enum,
                AllowedValues: []string{"Critical", "Major", "Minor", "Trivial"},
            },
        },
    }
}
```

## UX Enhancements

### Автокомплит в текстовом поле

```
User печатает: "#"
→ Показать всплывающий список:
  #task <title>
  #bug <title>
  #epic <title>
  #status <value>
  #assignee @user
  #priority <value>
  #due <date>
  #title <text>

User печатает: "#sta"
→ Фильтрованный список:
  #status <value>

User печатает: "#status "
→ Показать допустимые значения:
  To Do
  In Progress
  Done

User печатает: "#assignee @"
→ Показать список пользователей:
  @alex
  @bob
  @charlie

User печатает: "#assignee @a"
→ Фильтрованный список:
  @alex
  @anna
  @andrew
```

### UI Shortcuts (альтернатива ручному вводу)

**Кнопки в интерфейсе чата:**

```
[Change Status ▼]
  → Выпадающий список статусов
  → При выборе: вставляет "#status In Progress" в поле ввода
  → User может добавить комментарий и отправить

[Assign ▼]
  → Список участников чата
  → При выборе: вставляет "#assignee @alex"

[Set Priority ▼]
  → High / Medium / Low
  → При выборе: вставляет "#priority High"

[Set Due Date 📅]
  → Date picker
  → При выборе: вставляет "#due 2025-10-20"
```

**Поведение:**
- Кнопки вставляют тег в текстовое поле (не отправляют сразу)
- Пользователь может отредактировать, добавить комментарий, потом отправить
- Это обучает синтаксису — пользователь видит, как формируется тег

### Подсветка тегов в реальном времени

```
User печатает в поле ввода:
"#status Done #assignee @alex"

Поле показывает:
[#status Done] [#assignee @alex]
  ↑ зелёная     ↑ зелёная
  подсветка     подсветка

User печатает:
"#status Dne #assignee @alex"

Поле показывает:
[#status Dne] [#assignee @alex]
  ↑ красная    ↑ зелёная
  подсветка    подсветка

Подсказка под полем:
"❌ Invalid status 'Dne'. Available: To Do, In Progress, Done"
```

**Преимущества:**
- Немедленная обратная связь
- Меньше ошибок при отправке
- Обучает правильному синтаксису

## Примеры полных сценариев

### Сценарий 1: Создание задачи с атрибутами

```
User в общем чате проекта:
"#task Реализовать OAuth авторизацию #priority High #assignee @alex"

Результат:
✅ Создан новый чат-задача
✅ type = Task
✅ title = "Реализовать OAuth авторизацию"
✅ priority = "High"
✅ assignee = @alex (резолвлен в UserID)
✅ status = "To Do" (дефолтный)

Bot ответ:
"✅ Task created: Реализовать OAuth авторизацию
 📋 Priority: High
 👤 Assigned to: @alex
 🔗 /tasks/uuid-here"
```

### Сценарий 2: Обсуждение + изменение статуса

```
User в чате задачи:
"Закончил первую итерацию, отправляю на код-ревью
#status In Progress"

Результат:
✅ Сообщение сохранено с текстом
✅ status → "In Progress"
✅ Карточка на канбане переместилась в колонку "In Progress"

Bot ответ:
"✅ Status changed to In Progress"
```

### Сценарий 3: Несколько тегов с ошибкой

```
User:
"#status Done #assignee @unknown #priority High"

Результат:
✅ status → "Done"
❌ assignee → ошибка (пользователь не найден)
✅ priority → "High"

Bot ответ:
"✅ Status changed to Done
 ✅ Priority changed to High
 ❌ User @unknown not found"
```

### Сценарий 4: Превращение чата в задачу

```
User в обычном discussion-чате:
"Давайте сделаем это задачей
#task Разобраться с проблемой производительности"

Результат:
✅ Чат превращается в Task
✅ title = "Разобраться с проблемой производительности"
✅ Чат появляется на канбане
✅ Все предыдущие сообщения остаются в истории

Bot ответ:
"✅ Chat converted to Task: Разобраться с проблемой производительности
 📋 This chat is now visible on the board"
```

### Сценарий 5: Drag-n-drop на канбане

```
User перетаскивает карточку "Task #123" из "In Progress" в "Done"

Backend:
1. Определяет: user_id, task_id, new_status = "Done"
2. Создаёт сообщение в чате задачи:
   author: user_id
   content: "#status Done"
3. Обрабатывает как обычное сообщение с тегом

Результат в чате задачи:
@alex: "#status Done"

Bot ответ:
"✅ Status changed to Done"

Все участники чата видят изменение
```

### Сценарий 6: Обычное обсуждение с # в тексте

```
User:
"Нужно добавить поддержку #hashtags в комментариях"

Результат:
✅ Сообщение сохранено
❌ "hashtags" не является известным тегом → игнорируется
✅ Ничего не меняется в задаче

(Никаких ошибок, обычное сообщение)
```

## Расширяемость (V2)

### Алиасы тегов

```
#s → #status
#p → #priority
#a → #assignee

Пример:
"#s Done #p High"
→ эквивалентно "#status Done #priority High"
```

### Естественный язык для дат

```
#due tomorrow
#due next friday
#due +3d (через 3 дня)
#due end of week
```

### Custom tags с регистрацией

```
Админ регистрирует тег #component:
- name: "component"
- type: enum
- values: ["Frontend", "Backend", "Database", "DevOps"]

User:
"#component Backend"
→ Сохраняется в CustomFields["component"] = "Backend"
```

### Bulk operations

```
#assignee @alex @bob @charlie
→ Создаёт 3 participant с ролью "assignee"

#label bug #label urgent #label security
→ Множественные лейблы
```

### Условные теги

```
#if status=Done then assignee=@qa-team
→ Автоматизация workflow
```

## Миграция и обратная совместимость

### Изменение грамматики в будущем

**Проблема:** Если изменим парсинг, старые сообщения могут распарситься иначе.

**Решение:**
```go
type Message struct {
    // ...
    ParsedTags      []ParsedTag
    ParserVersion   int  // версия парсера, которым обработано
}
```

При изменении грамматики:
- Инкрементируем ParserVersion
- Старые сообщения не перепарсиваем
- Новые сообщения парсятся новым парсером
- Read model (Task Projection) показывает актуальное состояние

## Тестирование грамматики

### Unit-тесты парсера

```go
func TestTagParser(t *testing.T) {
    tests := []struct {
        input    string
        expected ParseResult
    }{
        {
            input: "#status Done",
            expected: ParseResult{
                Tags: []ParsedTag{{Key: "status", Value: "Done"}},
            },
        },
        {
            input: "#task Название задачи #priority High",
            expected: ParseResult{
                Tags: []ParsedTag{
                    {Key: "task", Value: "Название задачи"},
                    {Key: "priority", Value: "High"},
                },
            },
        },
        {
            input: "Обсуждение\n#status Done",
            expected: ParseResult{
                Tags:      []ParsedTag{{Key: "status", Value: "Done"}},
                PlainText: "Обсуждение",
            },
        },
        {
            input: "Поддержка #hashtags в тексте",
            expected: ParseResult{
                Tags:      []ParsedTag{},
                PlainText: "Поддержка #hashtags в тексте",
            },
        },
    }

    parser := NewTagParser()
    for _, tt := range tests {
        result := parser.Parse(tt.input)
        assert.Equal(t, tt.expected, result)
    }
}
```

### Integration-тесты валидации

```go
func TestTagValidation(t *testing.T) {
    // Тест: несуществующий статус
    result := executor.Execute(taskID, ChangeStatusCommand{Status: "Invalid"})
    assert.Error(t, result)
    assert.Contains(t, result.Error(), "Invalid status")

    // Тест: несуществующий пользователь
    result := executor.Execute(taskID, AssignCommand{User: "@nobody"})
    assert.Error(t, result)
    assert.Contains(t, result.Error(), "User @nobody not found")

    // Тест: частичное применение
    results := executor.ExecuteBatch(taskID, []Command{
        ChangeStatusCommand{Status: "Done"},
        AssignCommand{User: "@nobody"},
        ChangePriorityCommand{Priority: "High"},
    })
    assert.True(t, results[0].Success)
    assert.False(t, results[1].Success)
    assert.True(t, results[2].Success)
}
```

## Резюме архитектурных решений

| Аспект | Решение | Обоснование |
|--------|---------|-------------|
| **Позиция тегов** | В начале сообщения или отдельной строкой | Баланс гибкости и безопасности |
| **Регистр** | CASE-SENSITIVE | Исключает случайные срабатывания |
| **Пользователи** | @username с резолвингом | Естественный синтаксис, валидация |
| **Даты** | Только ISO 8601 (MVP) | Однозначность, простота парсинга |
| **Ошибки** | Частичное применение | User-friendly, не теряем валидные изменения |
| **Custom tags** | Регистрация в V2 | MVP: только известные теги |
| **# в тексте** | Парсятся только известные теги | Не ломает обычное общение |
| **Алиасы** | V2 | Упрощает использование для опытных |
| **Сохранение сообщений** | Всегда, даже с ошибками | Не теряем историю обсуждений |

## Следующие шаги

1. ✅ Core use cases определены
2. ✅ Domain model разработана
3. ✅ Детальная грамматика тегов
4. **TODO:** Права доступа и security model
5. **TODO:** API контракты (HTTP + WebSocket)
6. **TODO:** Структура кода (внутри internal/)
7. **TODO:** План реализации MVP (roadmap)
