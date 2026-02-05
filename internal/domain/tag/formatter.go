package tag

import (
	"fmt"
	"strings"
	"time"
)

// ActorInfo contains information about who performed an action.
type ActorInfo struct {
	ID            string
	DisplayName   string
	IsIntegration bool   // true if action was performed by an external integration
	Integration   string // integration name (e.g., "Jira sync", "GitHub webhook")
}

// GenerateBotResponse generates bot response with tag application results
// returns empty string if no tags were processed
func (pr *ProcessingResult) GenerateBotResponse() string {
	return pr.GenerateBotResponseWithActor(ActorInfo{})
}

// GenerateBotResponseWithActor generates bot response with actor information
// returns empty string if no tags were processed
func (pr *ProcessingResult) GenerateBotResponseWithActor(actor ActorInfo) string {
	if !pr.HasTags() {
		return "" // no tags - no response
	}

	var lines []string

	// successfully applied tags
	for _, applied := range pr.AppliedTags {
		if applied.Success {
			lines = append(lines, formatSuccessWithActor(applied, actor))
		}
	}

	// errors
	for _, err := range pr.Errors {
		lines = append(lines, formatError(err))
	}

	return strings.Join(lines, "\n")
}

// formatSuccessWithActor formats success message with actor information
func formatSuccessWithActor(applied TagApplication, actor ActorInfo) string {
	actorName := getActorName(actor)

	switch applied.Command.(type) {
	case CreateTaskCommand:
		return formatWithActor(actorName, "created task:", "Task created:", applied.TagValue)
	case CreateBugCommand:
		return formatWithActor(actorName, "created bug:", "Bug created:", applied.TagValue)
	case CreateEpicCommand:
		return formatWithActor(actorName, "created epic:", "Epic created:", applied.TagValue)
	case ChangeStatusCommand:
		return formatWithActor(actorName, "changed status to", "Status changed to", applied.TagValue)
	case AssignUserCommand:
		return formatAssignUser(actorName, applied.TagValue)
	case ChangePriorityCommand:
		return formatWithActor(actorName, "set priority to", "Priority changed to", applied.TagValue)
	case SetDueDateCommand:
		return formatDueDate(actorName, applied.TagValue)
	case ChangeTitleCommand:
		return formatWithActor(actorName, "changed title to:", "Title changed to:", applied.TagValue)
	case SetSeverityCommand:
		return formatWithActor(actorName, "set severity to", "Severity set to", applied.TagValue)
	case InviteUserCommand:
		invitee := strings.TrimPrefix(applied.TagValue, "@")
		return formatWithActor(actorName, "invited "+invitee+" to the chat", "Invited "+invitee+" to the chat", "")
	case RemoveUserCommand:
		removee := strings.TrimPrefix(applied.TagValue, "@")
		return formatWithActor(actorName, "removed "+removee+" from the chat", "Removed "+removee+" from the chat", "")
	case CloseChatCommand:
		return formatWithActor(actorName, "closed the chat", "Chat closed", "")
	case ReopenChatCommand:
		return formatWithActor(actorName, "reopened the chat", "Chat reopened", "")
	case DeleteChatCommand:
		return formatWithActor(actorName, "deleted the chat", "Chat deleted", "")
	default:
		return "✅ Applied"
	}
}

// formatWithActor formats a message with or without actor name
func formatWithActor(actorName, actionWithActor, actionWithoutActor, value string) string {
	if actorName != "" {
		if value != "" {
			return fmt.Sprintf("✅ %s %s %s", actorName, actionWithActor, value)
		}
		return fmt.Sprintf("✅ %s %s", actorName, actionWithActor)
	}
	if value != "" {
		return fmt.Sprintf("✅ %s %s", actionWithoutActor, value)
	}
	return fmt.Sprintf("✅ %s", actionWithoutActor)
}

// formatAssignUser formats assign/unassign user message
func formatAssignUser(actorName, tagValue string) string {
	if tagValue == "" || tagValue == "@none" {
		return formatWithActor(actorName, "removed the assignee", "Assignee removed", "")
	}
	assignee := strings.TrimPrefix(tagValue, "@")
	return formatWithActor(actorName, "assigned this to", "Assigned to:", assignee)
}

// formatDueDate formats due date set/remove message
func formatDueDate(actorName, tagValue string) string {
	if tagValue == "" {
		return formatWithActor(actorName, "removed the due date", "Due date removed", "")
	}
	formattedDate := formatHumanReadableDate(tagValue)
	return formatWithActor(actorName, "set due date to", "Due date set to", formattedDate)
}

// getActorName returns the display name for the actor
func getActorName(actor ActorInfo) string {
	if actor.IsIntegration && actor.Integration != "" {
		return actor.Integration
	}
	return actor.DisplayName
}

// formatHumanReadableDate formats a date string (YYYY-MM-DD) to human-readable format
func formatHumanReadableDate(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr // return original if parsing fails
	}
	return t.Format("January 2, 2006")
}

// formatError formats error message
func formatError(err TagError) string {
	prefix := "❌"
	if err.Severity == ErrorSeverityWarning {
		prefix = "⚠️"
	}

	return fmt.Sprintf("%s %s", prefix, err.Error.Error())
}
