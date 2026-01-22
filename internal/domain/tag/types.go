package tag

// ValueType defines the type of tag value
type ValueType int

const (
	// ValueTypeString - arbitrary string
	ValueTypeString ValueType = iota
	// ValueTypeUsername - username in format @username
	ValueTypeUsername
	// ValueTypeDate - date in ISO 8601 format
	ValueTypeDate
	// ValueTypeEnum - value from predefined list
	ValueTypeEnum
	// ValueTypeNone - tag without value
	ValueTypeNone
)

// String returns string representation of value type
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
	case ValueTypeNone:
		return "None"
	default:
		return "Unknown"
	}
}

// ParsedTag represents a parsed tag
type ParsedTag struct {
	Key   string
	Value string
}

// ParseResult represents the result of message parsing
type ParseResult struct {
	Tags      []ParsedTag
	PlainText string
}

// Definition defines metadata about a tag
type Definition struct {
	Name          string
	RequiresValue bool
	ValueType     ValueType
	AllowedValues []string // for Enum
	Validator     func(value string) error
}
