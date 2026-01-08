package tag

// ErrorSeverity defines sereznost error
type ErrorSeverity int

const (
	// ErrorSeverityError - error, applying nevozmozhno (❌)
	ErrorSeverityError ErrorSeverity = iota
	// ErrorSeverityWarning - warning, primeneno s zamechaniem (⚠️)
	ErrorSeverityWarning
)

// ProcessingResult contains result work tegov in soobschenii
type ProcessingResult struct {
	OriginalMessage string
	PlainText       string           // tekst bez tegov
	AppliedTags     []TagApplication // successfully primenennye tags
	Errors          []TagError       // oshibki valid and primeneniya
}

// TagApplication represents successfully primenennyy teg
//
//nolint:revive // TagApplication is intentional - represents tag processing application
type TagApplication struct {
	TagKey   string
	TagValue string
	Command  Command
	Success  bool
}

// TagError represents error valid or primeneniya tega
//
//nolint:revive // TagError is intentional - represents tag processing error
type TagError struct {
	TagKey   string
	TagValue string
	Error    error
	Severity ErrorSeverity
}

// HasTags returns true if byli work as-libo tags
func (pr *ProcessingResult) HasTags() bool {
	return len(pr.AppliedTags) > 0 || len(pr.Errors) > 0
}

// HasErrors returns true if est error
func (pr *ProcessingResult) HasErrors() bool {
	return len(pr.Errors) > 0
}

// SuccessCount returns count successfully primenennyh tegov
func (pr *ProcessingResult) SuccessCount() int {
	count := 0
	for _, applied := range pr.AppliedTags {
		if applied.Success {
			count++
		}
	}
	return count
}
