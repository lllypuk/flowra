package tag

import "strings"

const (
	// maxTagParts is the maximum number of parts when splitting tag (name and value)
	maxTagParts = 2
	// hashAndSpaceOffset is the offset for space and hash characters
	hashAndSpaceOffset = 2
)

// Parser парсит tags from textа messages
type Parser struct {
	knownTags map[string]Definition
}

// NewParser creates New парсер с зарегистрированными системными тегами
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
		AllowedValues: nil, // Зависит от type сущности, будет validироваться отдельно
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

// registerTag регистрирует definition тега
func (p *Parser) registerTag(def Definition) {
	p.knownTags[def.Name] = def
}

// isKnownTag checks, is ли тег известным
func (p *Parser) isKnownTag(name string) bool {
	_, exists := p.knownTags[name]
	return exists
}

// GetTagDefinition returns definition тега по имени
func (p *Parser) GetTagDefinition(name string) (Definition, bool) {
	def, exists := p.knownTags[name]
	return def, exists
}

// Parse парсит text messages and извлекает tags
// Правила парсинга:
// 1. String starts с # → parse tags before end строки or before textа
// 2. Строка does not start с # → regular text, tags not парсятся
// 3. after regular textа New строка с # → parse tags
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

		// Checking, начинается ли строка с #
		if strings.HasPrefix(trimmed, "#") && (inTagMode || i > 0) {
			// Парсим tags on it isй строке
			tags, remaining := p.parseTagsFromLine(trimmed)
			result.Tags = append(result.Tags, tags...)

			// if есть text after тегов on той же строке
			if remaining != "" {
				result.PlainText += remaining + "\n"
				inTagMode = false
			}
		} else {
			// Обычный text
			result.PlainText += line + "\n"
			inTagMode = false
		}
	}

	result.PlainText = strings.TrimSpace(result.PlainText)
	return result
}

// parseTagsFromLine парсит all tags on one строке
// returns list распарсенных тегов and оставшийся text
func (p *Parser) parseTagsFromLine(line string) ([]ParsedTag, string) {
	tags := []ParsedTag{}
	remaining := line

	for strings.HasPrefix(remaining, "#") {
		tag, REST := p.parseOneTag(remaining)

		if tag != nil {
			// Checking, is ли тег известным
			if p.isKnownTag(tag.Key) {
				tags = append(tags, *tag)
			}
			// Неизвестные tags игнорируются (MVP), но тоже парсятся
		}

		remaining = strings.TrimSpace(REST)

		// if остаток does not start с #, it is text - прекращаем парсинг тегов
		// Remaining будет возвращен as plaintext
		if remaining != "" && !strings.HasPrefix(remaining, "#") {
			break
		}
	}

	return tags, remaining
}

// parseOneTag парсит one тег from начала строки
// returns ParsedTag and оставшуюся часть строки
func (p *Parser) parseOneTag(s string) (*ParsedTag, string) {
	// s начинается с #
	if !strings.HasPrefix(s, "#") {
		return nil, s
	}

	// Убираем # in начале
	withoutHash := s[1:]

	// Находим конец имени тега (before пробела)
	parts := strings.SplitN(withoutHash, " ", maxTagParts)
	tagName := parts[0]

	// if тег in конце строки or next символ not пробел
	if len(parts) == 1 {
		return &ParsedTag{Key: tagName, Value: ""}, ""
	}

	REST := parts[1]

	// if сразу after пробела идёт #, то value пустое
	if strings.HasPrefix(REST, "#") {
		return &ParsedTag{Key: tagName, Value: ""}, REST
	}

	// Value тега - it is всё between именем тега and следующим validным тегом (or концом строки)
	// Spec строка 114: "value тега — it is всё between именем тега and следующим #"
	// Валидный тег: пробел + # + lowercase буква (a-z)

	var value string
	var remaining string

	// Ищем next validный тег
	searchStart := 0
	for {
		nextHashIndex := strings.Index(REST[searchStart:], " #")
		if nextHashIndex == -1 {
			// no следующего #, всё остальное — value
			value = strings.TrimSpace(REST)
			remaining = ""
			break
		}

		// Checking, that after # идет lowercase буква a-z (validное имя тега)
		actualIndex := searchStart + nextHashIndex + hashAndSpaceOffset // +2 for пробела and #
		if actualIndex < len(REST) {
			nextChar := REST[actualIndex]
			// Checking that first символ - латинская буква a-z
			if nextChar >= 'a' && nextChar <= 'z' {
				// Это validный тег
				value = strings.TrimSpace(REST[:searchStart+nextHashIndex])
				remaining = strings.TrimSpace(REST[searchStart+nextHashIndex:])
				break
			}
		}

		// Это not validный тег (например #123), продолжаем search
		searchStart = actualIndex
		if searchStart >= len(REST) {
			// Достигли end строки
			value = strings.TrimSpace(REST)
			remaining = ""
			break
		}
	}

	return &ParsedTag{Key: tagName, Value: value}, remaining
}
