package tag

import (
	"fmt"
	"strings"
)

// GenerateBotResponse генерирует response бота с результатами применения тегов
// returns пустую строку if no тегов for обworkки
func (pr *ProcessingResult) GenerateBotResponse() string {
	if !pr.HasTags() {
		return "" // no тегов - no responseа
	}

	var lines []string

	// successfully примененные tags
	for _, applied := range pr.AppliedTags {
		if applied.Success {
			lines = append(lines, formatSuccess(applied))
		}
	}

	// Ошибки
	for _, err := range pr.Errors {
		lines = append(lines, formatError(err))
	}

	return strings.Join(lines, "\n")
}

// formatSuccess форматирует message об успешном применении тега
func formatSuccess(applied TagApplication) string {
	switch applied.Command.(type) {
	case CreateTaskCommand:
		return fmt.Sprintf("✅ Task created: %s", applied.TagValue)

	case CreateBugCommand:
		return fmt.Sprintf("✅ Bug created: %s", applied.TagValue)

	case CreateEpicCommand:
		return fmt.Sprintf("✅ Epic created: %s", applied.TagValue)

	case ChangeStatusCommand:
		return fmt.Sprintf("✅ Status changed to %s", applied.TagValue)

	case AssignUserCommand:
		if applied.TagValue == "" || applied.TagValue == "@none" {
			return "✅ Assignee removed"
		}
		return fmt.Sprintf("✅ Assigned to: %s", applied.TagValue)

	case ChangePriorityCommand:
		return fmt.Sprintf("✅ Priority changed to %s", applied.TagValue)

	case SetDueDateCommand:
		if applied.TagValue == "" {
			return "✅ Due date removed"
		}
		return fmt.Sprintf("✅ Due date set to %s", applied.TagValue)

	case ChangeTitleCommand:
		return fmt.Sprintf("✅ Title changed to: %s", applied.TagValue)

	case SetSeverityCommand:
		return fmt.Sprintf("✅ severity set to %s", applied.TagValue)

	default:
		return "✅ Applied"
	}
}

// formatError форматирует message об error
func formatError(err TagError) string {
	prefix := "❌"
	if err.severity == ErrorSeverityWarning {
		prefix = "⚠️"
	}

	return fmt.Sprintf("%s %s", prefix, err.Error.Error())
}
