package tag

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

// ErrNoActiveEntity is returned when Entity Management Tag is used without active entity
var ErrNoActiveEntity = errors.New("no active entity to modify. Create an entity first with #task, #bug, or #epic")

// usernameRegex defines allowed username format: @[a-zA-Z0-9._-]+
var usernameRegex = regexp.MustCompile(`^@[a-zA-Z0-9._-]+$`)

// ====== Task 04: Status Validation Constants ======

//nolint:gochecknoglobals // Domain constants for entity statuses
var (
	// TaskStatuses - allowed statuses for Task (CASE-SENSITIVE)
	TaskStatuses = []string{"To Do", "In Progress", "Done"}

	// BugStatuses - allowed statuses for Bug (CASE-SENSITIVE)
	BugStatuses = []string{"New", "Investigating", "Fixed", "Verified"}

	// EpicStatuses - allowed statuses for Epic (CASE-SENSITIVE)
	EpicStatuses = []string{"Planned", "In Progress", "Completed"}
)

// validateUsername checks username format (@username)
func validateUsername(value string) error {
	// empty value is allowed (removes assignee)
	if value == "" || value == "@none" {
		return nil
	}

	// check for @ presence
	if !strings.HasPrefix(value, "@") {
		return errors.New("invalid assignee format. Use @username")
	}

	// check that there's a name after @
	if len(value) == 1 {
		return errors.New("invalid assignee format. Use @username")
	}

	// check username format
	if !usernameRegex.MatchString(value) {
		return errors.New("invalid assignee format. Use @username")
	}

	return nil
}

// validateISODate validates ISO 8601 date format
func validateISODate(value string) error {
	// empty value is allowed (removes due date)
	if value == "" {
		return nil
	}

	// supported formats (MVP)
	formats := []string{
		"2006-01-02",                // YYYY-MM-DD
		"2006-01-02T15:04",          // YYYY-MM-DDTHH:MM
		"2006-01-02T15:04:05",       // YYYY-MM-DDTHH:MM:SS
		time.RFC3339,                // YYYY-MM-DDTHH:MM:SSZ or with timezone
		"2006-01-02T15:04:05Z07:00", // with explicit timezone
	}

	// try to parse date in one of the formats
	for _, format := range formats {
		if _, err := time.Parse(format, value); err == nil {
			return nil
		}
	}

	return errors.New("invalid date format. Use ISO 8601: YYYY-MM-DD")
}

// validatePriority checks value priority
func validatePriority(value string) error {
	allowedValues := []string{"High", "Medium", "Low"}

	if slices.Contains(allowedValues, value) {
		return nil
	}

	return fmt.Errorf("invalid priority '%s'. Available: %s",
		value, strings.Join(allowedValues, ", "))
}

// validateSeverity validates bug severity value
func validateSeverity(value string) error {
	allowedValues := []string{"Critical", "Major", "Minor", "Trivial"}

	if slices.Contains(allowedValues, value) {
		return nil
	}

	return fmt.Errorf("invalid severity '%s'. Available: %s",
		value, strings.Join(allowedValues, ", "))
}

// noValidation is a no-op validator for tags without additional validation
func noValidation(_ string) error {
	return nil
}

// ValidateEntityCreation validates title for creating entity
func ValidateEntityCreation(tagKey, title string) error {
	trimmed := strings.TrimSpace(title)

	if trimmed == "" {
		// Capitalize first letter of tagKey for error message
		capitalizedKey := strings.ToUpper(string(tagKey[0])) + tagKey[1:]
		return fmt.Errorf("%s title is required. Usage: #%s <title>",
			capitalizedKey, tagKey)
	}

	return nil
}

// ====== Task 04: Entity Management Validators ======

// ValidateStatus validates status for specific entity type
// entityType must be "Task", "Bug" or "Epic"
// statuses are CASE-SENSITIVE
func ValidateStatus(entityType, status string) error {
	var allowedStatuses []string

	switch entityType {
	case "Task":
		allowedStatuses = TaskStatuses
	case "Bug":
		allowedStatuses = BugStatuses
	case "Epic":
		allowedStatuses = EpicStatuses
	default:
		return fmt.Errorf("unknown entity type: %s", entityType)
	}

	if slices.Contains(allowedStatuses, status) {
		return nil
	}

	return fmt.Errorf("invalid status '%s' for %s. Available: %s",
		status, entityType, strings.Join(allowedStatuses, ", "))
}

// ValidateDueDate parses date and returns *time.Time
// empty value returns nil (removes due date)
//
//nolint:nilnil // Returning (nil, nil) is intentional for empty date (remove due date)
func ValidateDueDate(dateStr string) (*time.Time, error) {
	// empty value is allowed (removes due date)
	if dateStr == "" {
		return nil, nil
	}

	// supported formats (MVP)
	formats := []string{
		"2006-01-02",                // YYYY-MM-DD
		"2006-01-02T15:04",          // YYYY-MM-DDTHH:MM
		"2006-01-02T15:04:05",       // YYYY-MM-DDTHH:MM:SS
		time.RFC3339,                // YYYY-MM-DDTHH:MM:SSZ or with timezone
		"2006-01-02T15:04:05Z07:00", // with explicit timezone
	}

	// try to parse date in one of the formats
	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return &t, nil
		}
	}

	return nil, errors.New("invalid date format. Use ISO 8601: YYYY-MM-DD")
}

// ValidateTitle checks that title not empty
func ValidateTitle(title string) error {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return errors.New("title cannot be empty")
	}
	return nil
}
