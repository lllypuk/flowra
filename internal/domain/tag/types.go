package tag

// ValueType defines type values tega
type ValueType int

const (
	// ValueTypeString - proizvolnaya stroka
	ValueTypeString ValueType = iota
	// ValueTypeUsername - imya user in formate @username
	ValueTypeUsername
	// ValueTypeDate - date in formate ISO 8601
	ValueTypeDate
	// ValueTypeEnum - value from predopredelennogo list
	ValueTypeEnum
)

// String returns strokovoe view type values
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

// ParsedTag represents rasparsennyy teg
type ParsedTag struct {
	Key   string
	Value string
}

// ParseResult represents result parsinga messages
type ParseResult struct {
	Tags      []ParsedTag
	PlainText string
}

// Definition defines metainformatsiyu o tege
type Definition struct {
	Name          string
	RequiresValue bool
	ValueType     ValueType
	AllowedValues []string // for Enum
	Validator     func(value string) error
}
