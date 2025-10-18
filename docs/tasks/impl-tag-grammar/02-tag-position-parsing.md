# Task 02: Tag Position Parsing Logic

**Статус:** Pending
**Приоритет:** High
**Зависимости:** Task 01
**Оценка:** 3-4 дня

## Описание

Реализовать логику парсинга тегов с учетом их позиции в сообщении. Теги могут находиться в начале сообщения или на отдельной строке после обычного текста.

## Цели

1. Реализовать главный метод `Parse(content string) ParseResult`
2. Реализовать парсинг одной строки с тегами
3. Реализовать парсинг отдельного тега
4. Обработать корректно все варианты позиционирования

## Технические требования

### Правила парсинга позиции

Согласно спецификации (строки 55-62):
1. Строка начинается с `#` → парсить теги до конца строки или до текста
2. Строка не начинается с `#` → обычный текст, теги не парсятся
3. После обычного текста новая строка с `#` → парсить теги
4. Пустые строки игнорируются

### Реализация методов

```go
// internal/tag/parser.go

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
            tags, remaining := p.parseTagsFromLine(trimmed)
            result.Tags = append(result.Tags, tags...)

            if remaining != "" {
                result.PlainText += remaining + "\n"
                inTagMode = false
            }
        } else {
            result.PlainText += line + "\n"
            inTagMode = false
        }
    }

    result.PlainText = strings.TrimSpace(result.PlainText)
    return result
}

func (p *TagParser) parseTagsFromLine(line string) ([]ParsedTag, string) {
    // Парсинг всех тегов на одной строке
    // Возвращает список тегов и оставшийся текст
}

func (p *TagParser) parseOneTag(s string) (*ParsedTag, string) {
    // Парсинг одного тега
    // Возвращает тег и оставшуюся строку
}
```

### Парсинг значений тегов

**Правило:** Значение тега — это всё между именем тега и следующим `#` (или концом строки), с обрезанными пробелами.

```
"#status In Progress #assignee @alex"
            ↑________↑
            value = "In Progress"
```

## Acceptance Criteria

- [ ] Реализован метод `Parse(content string) ParseResult`
- [ ] Реализован метод `parseTagsFromLine(line string) ([]ParsedTag, string)`
- [ ] Реализован метод `parseOneTag(s string) (*ParsedTag, string)`
- [ ] Корректно обрабатываются теги в начале сообщения
- [ ] Корректно обрабатываются теги на отдельной строке после текста
- [ ] Игнорируются теги в середине строки (невалидная позиция)
- [ ] Неизвестные теги игнорируются (но парсятся для будущего использования)
- [ ] Пустые строки корректно обрабатываются
- [ ] Значения тегов извлекаются правильно (с учетом многословных значений)

## Примеры парсинга

### Пример 1: Только теги
```
Input: "#status Done"
Output:
  Tags: [{Key: "status", Value: "Done"}]
  PlainText: ""
```

### Пример 2: Теги в начале + текст
```
Input: "#status Done #assignee @alex\nЗакончил работу"
Output:
  Tags: [
    {Key: "status", Value: "Done"},
    {Key: "assignee", Value: "@alex"}
  ]
  PlainText: "Закончил работу"
```

### Пример 3: Текст + теги на отдельной строке
```
Input: "Закончил работу\n#status Done"
Output:
  Tags: [{Key: "status", Value: "Done"}]
  PlainText: "Закончил работу"
```

### Пример 4: Невалидная позиция (игнорируется)
```
Input: "Закончил работу #status Done отправляю"
Output:
  Tags: []
  PlainText: "Закончил работу #status Done отправляю"
```

### Пример 5: Неизвестный тег (игнорируется)
```
Input: "Поддержка #hashtags в тексте"
Output:
  Tags: []
  PlainText: "Поддержка #hashtags в тексте"
```

### Пример 6: Множественные значения
```
Input: "#task Реализовать функцию авторизации #priority High"
Output:
  Tags: [
    {Key: "task", Value: "Реализовать функцию авторизации"},
    {Key: "priority", Value: "High"}
  ]
  PlainText: ""
```

## Тесты

```go
func TestParse(t *testing.T) {
    parser := NewTagParser()

    tests := []struct {
        name     string
        input    string
        wantTags []ParsedTag
        wantText string
    }{
        {
            name:  "single tag",
            input: "#status Done",
            wantTags: []ParsedTag{
                {Key: "status", Value: "Done"},
            },
            wantText: "",
        },
        {
            name:  "multiple tags",
            input: "#status Done #assignee @alex",
            wantTags: []ParsedTag{
                {Key: "status", Value: "Done"},
                {Key: "assignee", Value: "@alex"},
            },
            wantText: "",
        },
        {
            name:  "tags with multi-word value",
            input: "#task Реализовать функцию авторизации #priority High",
            wantTags: []ParsedTag{
                {Key: "task", Value: "Реализовать функцию авторизации"},
                {Key: "priority", Value: "High"},
            },
            wantText: "",
        },
        {
            name:  "text then tags on new line",
            input: "Закончил работу\n#status Done",
            wantTags: []ParsedTag{
                {Key: "status", Value: "Done"},
            },
            wantText: "Закончил работу",
        },
        {
            name:  "tags in middle of line - ignored",
            input: "Закончил работу #status Done",
            wantTags: []ParsedTag{},
            wantText: "Закончил работу #status Done",
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
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := parser.Parse(tt.input)
            assert.Equal(t, tt.wantTags, result.Tags)
            assert.Equal(t, tt.wantText, result.PlainText)
        })
    }
}
```

## Ссылки

- Позиция тегов: `docs/03-tag-grammar.md` (строки 18-63)
- Формальная грамматика: `docs/03-tag-grammar.md` (строки 64-111)
- Парсинг значений: `docs/03-tag-grammar.md` (строки 112-128)
- Псевдокод: `docs/03-tag-grammar.md` (строки 486-592)
