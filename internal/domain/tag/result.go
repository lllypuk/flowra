package tag

// ErrorSeverity определяет серьезность ошибки
type ErrorSeverity int

const (
	// ErrorSeverityError - ошибка, применение невозможно (❌)
	ErrorSeverityError ErrorSeverity = iota
	// ErrorSeverityWarning - предупреждение, применено с замечанием (⚠️)
	ErrorSeverityWarning
)

// ProcessingResult содержит результат обработки тегов в сообщении
type ProcessingResult struct {
	OriginalMessage string
	PlainText       string           // Текст без тегов
	AppliedTags     []TagApplication // Успешно примененные теги
	Errors          []TagError       // Ошибки валидации и применения
}

// TagApplication представляет успешно примененный тег
//
//nolint:revive // TagApplication is intentional - represents tag processing application
type TagApplication struct {
	TagKey   string
	TagValue string
	Command  Command
	Success  bool
}

// TagError представляет ошибку валидации или применения тега
//
//nolint:revive // TagError is intentional - represents tag processing error
type TagError struct {
	TagKey   string
	TagValue string
	Error    error
	Severity ErrorSeverity
}

// HasTags возвращает true если были обработаны какие-либо теги
func (pr *ProcessingResult) HasTags() bool {
	return len(pr.AppliedTags) > 0 || len(pr.Errors) > 0
}

// HasErrors возвращает true если есть ошибки
func (pr *ProcessingResult) HasErrors() bool {
	return len(pr.Errors) > 0
}

// SuccessCount возвращает количество успешно примененных тегов
func (pr *ProcessingResult) SuccessCount() int {
	count := 0
	for _, applied := range pr.AppliedTags {
		if applied.Success {
			count++
		}
	}
	return count
}
