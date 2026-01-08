package tag

// ErrorSeverity defines серьезность error
type ErrorSeverity int

const (
	// ErrorSeverityError - error, applying невозможно (❌)
	ErrorSeverityError ErrorSeverity = iota
	// ErrorSeverityWarning - warning, применено с замечанием (⚠️)
	ErrorSeverityWarning
)

// ProcessingResult contains result обworkки тегов in сообщении
type ProcessingResult struct {
	OriginalMessage string
	PlainText       string           // Текст без тегов
	AppliedTags     []TagApplication // successfully примененные tags
	Errors          []TagError       // Ошибки validации and применения
}

// TagApplication represents successfully примененный тег
//
//nolint:revive // TagApplication is intentional - represents tag processing application
type TagApplication struct {
	TagKey   string
	TagValue string
	Command  Command
	Success  bool
}

// TagError represents error validации or применения тега
//
//nolint:revive // TagError is intentional - represents tag processing error
type TagError struct {
	TagKey   string
	TagValue string
	Error    error
	severity ErrorSeverity
}

// HasTags returns true if были обworkаны asие-либо tags
func (pr *ProcessingResult) HasTags() bool {
	return len(pr.AppliedTags) > 0 || len(pr.Errors) > 0
}

// HasErrors returns true if есть error
func (pr *ProcessingResult) HasErrors() bool {
	return len(pr.Errors) > 0
}

// SuccessCount returns count successfully примененных тегов
func (pr *ProcessingResult) SuccessCount() int {
	count := 0
	for _, applied := range pr.AppliedTags {
		if applied.Success {
			count++
		}
	}
	return count
}
