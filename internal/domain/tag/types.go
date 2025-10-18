package tag

// ValueType определяет тип значения тега
type ValueType int

const (
	// ValueTypeString - произвольная строка
	ValueTypeString ValueType = iota
	// ValueTypeUsername - имя пользователя в формате @username
	ValueTypeUsername
	// ValueTypeDate - дата в формате ISO 8601
	ValueTypeDate
	// ValueTypeEnum - значение из предопределенного списка
	ValueTypeEnum
)

// String возвращает строковое представление типа значения
func (vt ValueType) String() string {
	switch vt {
	case ValueTypeString:
		return "String"
	case ValueTypeUsername:
		return "Username"
	case ValueTypeDate:
		return "Date"
	case ValueTypeEnum:
		return "Enum"
	default:
		return "Unknown"
	}
}

// ParsedTag представляет распарсенный тег
type ParsedTag struct {
	Key   string
	Value string
}

// ParseResult представляет результат парсинга сообщения
type ParseResult struct {
	Tags      []ParsedTag
	PlainText string
}

// Definition определяет метаинформацию о теге
type Definition struct {
	Name          string
	RequiresValue bool
	ValueType     ValueType
	AllowedValues []string // для Enum
	Validator     func(value string) error
}
