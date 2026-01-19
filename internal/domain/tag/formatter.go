package tag

import (
	"fmt"
	"strings"
)

// GenerateBotResponse generiruet response bota s rezultatami primeneniya tegov
// returns pustuyu stroku if no tegov for work
func (pr *ProcessingResult) GenerateBotResponse() string {
	if !pr.HasTags() {
		return "" // no tegov - no response
	}

	var lines []string

	// successfully primenennye tags
	for _, applied := range pr.AppliedTags {
		if applied.Success {
			lines = append(lines, formatSuccess(applied))
		}
	}

	// oshibki
	for _, err := range pr.Errors {
		lines = append(lines, formatError(err))
	}

	return strings.Join(lines, "\n")
}

// formatSuccess formatiruet message ob uspeshnom primenenii tega
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
		return fmt.Sprintf("✅ Severity set to %s", applied.TagValue)

	case InviteUserCommand:
		return fmt.Sprintf("✅ Invited %s to the chat", applied.TagValue)

	case RemoveUserCommand:
		return fmt.Sprintf("✅ Removed %s from the chat", applied.TagValue)

	case CloseChatCommand:
		return "✅ Chat closed"

	case ReopenChatCommand:
		return "✅ Chat reopened"

	case DeleteChatCommand:
		return "✅ Chat deleted"

	default:
		return "✅ Applied"
	}
}

// formatError formatiruet message ob error
func formatError(err TagError) string {
	prefix := "❌"
	if err.Severity == ErrorSeverityWarning {
		prefix = "⚠️"
	}

	return fmt.Sprintf("%s %s", prefix, err.Error.Error())
}
