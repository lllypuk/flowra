package tag

import "strings"

const (
	// maxTagParts is the maximum number of parts when splitting tag (name and value)
	maxTagParts = 2
	// hashAndSpaceOffset is the offset for space and hash characters
	hashAndSpaceOffset = 2
)

// Parser парсит теги из текста сообщения
type Parser struct {
	knownTags map[string]Definition
}

// NewParser создает новый парсер с зарегистрированными системными тегами
func NewParser() *Parser {
	parser := &Parser{
		knownTags: make(map[string]Definition),
	}

	// Entity Creation Tags
	parser.registerTag(Definition{
		Name:          "task",
		RequiresValue: true,
		ValueType:     ValueTypeString,
		Validator:     noValidation,
	})

	parser.registerTag(Definition{
		Name:          "bug",
		RequiresValue: true,
		ValueType:     ValueTypeString,
		Validator:     noValidation,
	})

	parser.registerTag(Definition{
		Name:          "epic",
		RequiresValue: true,
		ValueType:     ValueTypeString,
		Validator:     noValidation,
	})

	// Entity Management Tags
	parser.registerTag(Definition{
		Name:          "status",
		RequiresValue: true,
		ValueType:     ValueTypeEnum,
		AllowedValues: nil, // Зависит от типа сущности, будет валидироваться отдельно
		Validator:     noValidation,
	})

	parser.registerTag(Definition{
		Name:          "assignee",
		RequiresValue: false, // Может быть пустым (снять assignee)
		ValueType:     ValueTypeUsername,
		Validator:     validateUsername,
	})

	parser.registerTag(Definition{
		Name:          "priority",
		RequiresValue: true,
		ValueType:     ValueTypeEnum,
		AllowedValues: []string{"High", "Medium", "Low"},
		Validator:     validatePriority,
	})

	parser.registerTag(Definition{
		Name:          "due",
		RequiresValue: false, // Может быть пустым (снять due_date)
		ValueType:     ValueTypeDate,
		Validator:     validateISODate,
	})

	parser.registerTag(Definition{
		Name:          "title",
		RequiresValue: true,
		ValueType:     ValueTypeString,
		Validator:     noValidation,
	})

	// Bug-Specific Tags
	parser.registerTag(Definition{
		Name:          "severity",
		RequiresValue: true,
		ValueType:     ValueTypeEnum,
		AllowedValues: []string{"Critical", "Major", "Minor", "Trivial"},
		Validator:     validateSeverity,
	})

	return parser
}

// registerTag регистрирует определение тега
func (p *Parser) registerTag(def Definition) {
	p.knownTags[def.Name] = def
}

// isKnownTag проверяет, является ли тег известным
func (p *Parser) isKnownTag(name string) bool {
	_, exists := p.knownTags[name]
	return exists
}

// GetTagDefinition возвращает определение тега по имени
func (p *Parser) GetTagDefinition(name string) (Definition, bool) {
	def, exists := p.knownTags[name]
	return def, exists
}

// Parse парсит текст сообщения и извлекает теги
// Правила парсинга:
// 1. Строка начинается с # → парсить теги до конца строки или до текста
// 2. Строка не начинается с # → обычный текст, теги не парсятся
// 3. После обычного текста новая строка с # → парсить теги
// 4. Пустые строки игнорируются
func (p *Parser) Parse(content string) ParseResult {
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

// parseTagsFromLine парсит все теги на одной строке
// Возвращает список распарсенных тегов и оставшийся текст
func (p *Parser) parseTagsFromLine(line string) ([]ParsedTag, string) {
	tags := []ParsedTag{}
	remaining := line

	for strings.HasPrefix(remaining, "#") {
		tag, rest := p.parseOneTag(remaining)

		if tag != nil {
			// Проверяем, является ли тег известным
			if p.isKnownTag(tag.Key) {
				tags = append(tags, *tag)
			}
			// Неизвестные теги игнорируются (MVP), но тоже парсятся
		}

		remaining = strings.TrimSpace(rest)

		// Если остаток не начинается с #, это текст - прекращаем парсинг тегов
		// Remaining будет возвращен как plaintext
		if remaining != "" && !strings.HasPrefix(remaining, "#") {
			break
		}
	}

	return tags, remaining
}

// parseOneTag парсит один тег из начала строки
// Возвращает ParsedTag и оставшуюся часть строки
func (p *Parser) parseOneTag(s string) (*ParsedTag, string) {
	// s начинается с #
	if !strings.HasPrefix(s, "#") {
		return nil, s
	}

	// Убираем # в начале
	withoutHash := s[1:]

	// Находим конец имени тега (до пробела)
	parts := strings.SplitN(withoutHash, " ", maxTagParts)
	tagName := parts[0]

	// Если тег в конце строки или следующий символ не пробел
	if len(parts) == 1 {
		return &ParsedTag{Key: tagName, Value: ""}, ""
	}

	rest := parts[1]

	// Если сразу после пробела идёт #, то значение пустое
	if strings.HasPrefix(rest, "#") {
		return &ParsedTag{Key: tagName, Value: ""}, rest
	}

	// Value тега - это всё между именем тега и следующим валидным тегом (или концом строки)
	// Spec строка 114: "Значение тега — это всё между именем тега и следующим #"
	// Валидный тег: пробел + # + lowercase буква (a-z)

	var value string
	var remaining string

	// Ищем следующий валидный тег
	searchStart := 0
	for {
		nextHashIndex := strings.Index(rest[searchStart:], " #")
		if nextHashIndex == -1 {
			// Нет следующего #, всё остальное — значение
			value = strings.TrimSpace(rest)
			remaining = ""
			break
		}

		// Проверяем, что после # идет lowercase буква a-z (валидное имя тега)
		actualIndex := searchStart + nextHashIndex + hashAndSpaceOffset // +2 для пробела и #
		if actualIndex < len(rest) {
			nextChar := rest[actualIndex]
			// Проверяем что первый символ - латинская буква a-z
			if nextChar >= 'a' && nextChar <= 'z' {
				// Это валидный тег
				value = strings.TrimSpace(rest[:searchStart+nextHashIndex])
				remaining = strings.TrimSpace(rest[searchStart+nextHashIndex:])
				break
			}
		}

		// Это не валидный тег (например #123), продолжаем поиск
		searchStart = actualIndex
		if searchStart >= len(rest) {
			// Достигли конца строки
			value = strings.TrimSpace(rest)
			remaining = ""
			break
		}
	}

	return &ParsedTag{Key: tagName, Value: value}, remaining
}
