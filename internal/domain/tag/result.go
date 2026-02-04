package tag

// ErrorSeverity defines error severity level
type ErrorSeverity int

const (
	// ErrorSeverityError - error, applying is not possible (❌)
	ErrorSeverityError ErrorSeverity = iota
	// ErrorSeverityWarning - warning, applied with a note (⚠️)
	ErrorSeverityWarning
)

// ProcessingResult contains tag processing result for a message
type ProcessingResult struct {
	OriginalMessage string
	PlainText       string           // text without tags
	AppliedTags     []TagApplication // successfully applied tags
	Errors          []TagError       // validation and application errors
}

// TagApplication represents a successfully applied tag
//
//nolint:revive // TagApplication is intentional - represents tag processing application
type TagApplication struct {
	TagKey   string
	TagValue string
	Command  Command
	Success  bool
}

// TagError represents a validation or application error for a tag
//
//nolint:revive // TagError is intentional - represents tag processing error
type TagError struct {
	TagKey   string
	TagValue string
	Error    error
	Severity ErrorSeverity
}

// HasTags returns true if any tags were processed
func (pr *ProcessingResult) HasTags() bool {
	return len(pr.AppliedTags) > 0 || len(pr.Errors) > 0
}

// HasErrors returns true if there are errors
func (pr *ProcessingResult) HasErrors() bool {
	return len(pr.Errors) > 0
}

// SuccessCount returns count of successfully applied tags
func (pr *ProcessingResult) SuccessCount() int {
	count := 0
	for _, applied := range pr.AppliedTags {
		if applied.Success {
			count++
		}
	}
	return count
}
