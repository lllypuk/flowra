package tag

// TagParser парсит теги из текста сообщения
type TagParser struct {
	knownTags map[string]TagDefinition
}

// NewTagParser создает новый парсер с зарегистрированными системными тегами
func NewTagParser() *TagParser {
	parser := &TagParser{
		knownTags: make(map[string]TagDefinition),
	}

	// Entity Creation Tags
	parser.registerTag(TagDefinition{
		Name:          "task",
		RequiresValue: true,
		ValueType:     ValueTypeString,
		Validator:     noValidation,
	})

	parser.registerTag(TagDefinition{
		Name:          "bug",
		RequiresValue: true,
		ValueType:     ValueTypeString,
		Validator:     noValidation,
	})

	parser.registerTag(TagDefinition{
		Name:          "epic",
		RequiresValue: true,
		ValueType:     ValueTypeString,
		Validator:     noValidation,
	})

	// Entity Management Tags
	parser.registerTag(TagDefinition{
		Name:          "status",
		RequiresValue: true,
		ValueType:     ValueTypeEnum,
		AllowedValues: nil, // Зависит от типа сущности, будет валидироваться отдельно
		Validator:     noValidation,
	})

	parser.registerTag(TagDefinition{
		Name:          "assignee",
		RequiresValue: false, // Может быть пустым (снять assignee)
		ValueType:     ValueTypeUsername,
		Validator:     validateUsername,
	})

	parser.registerTag(TagDefinition{
		Name:          "priority",
		RequiresValue: true,
		ValueType:     ValueTypeEnum,
		AllowedValues: []string{"High", "Medium", "Low"},
		Validator:     validatePriority,
	})

	parser.registerTag(TagDefinition{
		Name:          "due",
		RequiresValue: false, // Может быть пустым (снять due_date)
		ValueType:     ValueTypeDate,
		Validator:     validateISODate,
	})

	parser.registerTag(TagDefinition{
		Name:          "title",
		RequiresValue: true,
		ValueType:     ValueTypeString,
		Validator:     noValidation,
	})

	// Bug-Specific Tags
	parser.registerTag(TagDefinition{
		Name:          "severity",
		RequiresValue: true,
		ValueType:     ValueTypeEnum,
		AllowedValues: []string{"Critical", "Major", "Minor", "Trivial"},
		Validator:     validateSeverity,
	})

	return parser
}

// registerTag регистрирует определение тега
func (p *TagParser) registerTag(def TagDefinition) {
	p.knownTags[def.Name] = def
}

// isKnownTag проверяет, является ли тег известным
func (p *TagParser) isKnownTag(name string) bool {
	_, exists := p.knownTags[name]
	return exists
}

// GetTagDefinition возвращает определение тега по имени
func (p *TagParser) GetTagDefinition(name string) (TagDefinition, bool) {
	def, exists := p.knownTags[name]
	return def, exists
}

// Parse парсит текст сообщения и извлекает теги
// Примечание: это базовая версия, полная реализация будет в Task 02
func (p *TagParser) Parse(content string) ParseResult {
	// Заглушка - будет реализовано в Task 02
	return ParseResult{
		Tags:      []ParsedTag{},
		PlainText: content,
	}
}
