package tag

// ValueType defines type values тега
type ValueType int

const (
	// ValueTypeString - произвольная строка
	ValueTypeString ValueType = iota
	// ValueTypeUsername - имя user in формате @username
	ValueTypeUsername
	// ValueTypeDate - date in формате ISO 8601
	ValueTypeDate
	// ValueTypeEnum - value from предопределенного list
	ValueTypeEnum
)

// String returns строковое view type values
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

// ParsedTag represents распарсенный тег
type ParsedTag struct {
	Key   string
	Value string
}

// ParseResult represents result парсинга messages
type ParseResult struct {
	Tags      []ParsedTag
	PlainText string
}

// Definition defines метаинформацию о теге
type Definition struct {
	Name          string
	RequiresValue bool
	ValueType     ValueType
	AllowedValues []string // for Enum
	Validator     func(value string) error
}
