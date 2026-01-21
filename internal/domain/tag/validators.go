package tag

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

// ErrNoActiveEntity vozvraschaetsya when Entity Management Tag used bez aktivnoy entity
var ErrNoActiveEntity = errors.New("No active entity to modify. Create an entity first with #task, #bug, or #epic")

// usernameRegex defines dopustimyy format username: @[a-zA-Z0-9._-]+
var usernameRegex = regexp.MustCompile(`^@[a-zA-Z0-9._-]+$`)

// ====== Task 04: Status Validation Constants ======

//nolint:gochecknoglobals // Domain constants for entity statuses
var (
	// TaskStatuses - dopustimye statusy for Task (CASE-SENSITIVE)
	TaskStatuses = []string{"To Do", "In Progress", "Done"}

	// BugStatuses - dopustimye statusy for Bug (CASE-SENSITIVE)
	BugStatuses = []string{"New", "Investigating", "Fixed", "Verified"}

	// EpicStatuses - dopustimye statusy for Epic (CASE-SENSITIVE)
	EpicStatuses = []string{"Planned", "In Progress", "Completed"}
)

// validateUsername checks format username (@username)
func validateUsername(value string) error {
	// pustoe value dopustimo (snyatie assignee)
	if value == "" || value == "@none" {
		return nil
	}

	// check nalichiya @
	if !strings.HasPrefix(value, "@") {
		return errors.New("invalid assignee format. Use @username")
	}

	// check that after @ est imya
	if len(value) == 1 {
		return errors.New("invalid assignee format. Use @username")
	}

	// check formata username
	if !usernameRegex.MatchString(value) {
		return errors.New("invalid assignee format. Use @username")
	}

	return nil
}

// validateISODate checks format daty ISO 8601
func validateISODate(value string) error {
	// pustoe value dopustimo (snyatie due date)
	if value == "" {
		return nil
	}

	// podderzhivaemye formaty (MVP)
	formats := []string{
		"2006-01-02",                // YYYY-MM-DD
		"2006-01-02T15:04",          // YYYY-MM-DDTHH:MM
		"2006-01-02T15:04:05",       // YYYY-MM-DDTHH:MM:SS
		time.RFC3339,                // YYYY-MM-DDTHH:MM:SSZ or s timezone
		"2006-01-02T15:04:05Z07:00", // with explicit timezone
	}

	// pytaemsya rasparsit datu in odnom from formatov
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

// validateSeverity checks value sereznosti baga
func validateSeverity(value string) error {
	allowedValues := []string{"Critical", "Major", "Minor", "Trivial"}

	if slices.Contains(allowedValues, value) {
		return nil
	}

	return fmt.Errorf("invalid severity '%s'. Available: %s",
		value, strings.Join(allowedValues, ", "))
}

// noValidation - valid-zaglushka for tegov bez dopolnitelnoy valid
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

// ValidateStatus validates status for konkretnogo type entity
// entityType dolzhen byt "Task", "Bug" or "Epic"
// statusy CASE-SENSITIVE
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

	return fmt.Errorf("Invalid status '%s' for %s. Available: %s",
		status, entityType, strings.Join(allowedStatuses, ", "))
}

// ValidateDueDate parsit datu and returns *time.Time
// pustoe value returns nil (snyatie due date)
//
//nolint:nilnil // Returning (nil, nil) is intentional for empty date (remove due date)
func ValidateDueDate(dateStr string) (*time.Time, error) {
	// pustoe value dopustimo (snyatie due date)
	if dateStr == "" {
		return nil, nil
	}

	// podderzhivaemye formaty (MVP)
	formats := []string{
		"2006-01-02",                // YYYY-MM-DD
		"2006-01-02T15:04",          // YYYY-MM-DDTHH:MM
		"2006-01-02T15:04:05",       // YYYY-MM-DDTHH:MM:SS
		time.RFC3339,                // YYYY-MM-DDTHH:MM:SSZ or s timezone
		"2006-01-02T15:04:05Z07:00", // with explicit timezone
	}

	// pytaemsya rasparsit datu in odnom from formatov
	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return &t, nil
		}
	}

	return nil, errors.New("Invalid date format. Use ISO 8601: YYYY-MM-DD")
}

// ValidateTitle checks that title not empty
func ValidateTitle(title string) error {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return errors.New("Title cannot be empty")
	}
	return nil
}
