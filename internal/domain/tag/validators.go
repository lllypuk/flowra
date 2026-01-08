package tag

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

// ErrNoActiveEntity возвращается when Entity Management Tag used без активной сущности
var ErrNoActiveEntity = errors.New("❌ No active entity to modify. Create an entity first with #task, #bug, or #epic")

// usernameRegex defines допустимый формат username: @[a-zA-Z0-9._-]+
var usernameRegex = regexp.MustCompile(`^@[a-zA-Z0-9._-]+$`)

// ====== Task 04: Status Validation Constants ======

//nolint:gochecknoglobals // Domain constants for entity statuses
var (
	// TaskStatuses - допустимые статусы for Task (CASE-SENSITIVE)
	TaskStatuses = []string{"To Do", "In Progress", "Done"}

	// BugStatuses - допустимые статусы for Bug (CASE-SENSITIVE)
	BugStatuses = []string{"New", "Investigating", "Fixed", "Verified"}

	// EpicStatuses - допустимые статусы for Epic (CASE-SENSITIVE)
	EpicStatuses = []string{"Planned", "In Progress", "Completed"}
)

// validateUsername checks формат username (@username)
func validateUsername(value string) error {
	// Пустое value допустимо (снятие assignee)
	if value == "" || value == "@none" {
		return nil
	}

	// check наличия @
	if !strings.HasPrefix(value, "@") {
		return errors.New("invalid assignee format. Use @username")
	}

	// check that after @ есть имя
	if len(value) == 1 {
		return errors.New("invalid assignee format. Use @username")
	}

	// check формата username
	if !usernameRegex.MatchString(value) {
		return errors.New("invalid assignee format. Use @username")
	}

	return nil
}

// validateISODate checks формат даты ISO 8601
func validateISODate(value string) error {
	// Пустое value допустимо (снятие due date)
	if value == "" {
		return nil
	}

	// Поддерживаемые форматы (MVP)
	formats := []string{
		"2006-01-02",                // YYYY-MM-DD
		"2006-01-02T15:04",          // YYYY-MM-DDTHH:MM
		"2006-01-02T15:04:05",       // YYYY-MM-DDTHH:MM:SS
		time.RFC3339,                // YYYY-MM-DDTHH:MM:SSZ or с timezone
		"2006-01-02T15:04:05Z07:00", // with explicit timezone
	}

	// Пытаемся распарсить дату in одном from форматов
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

// validateSeverity checks value серьезности бага
func validateSeverity(value string) error {
	allowedValues := []string{"Critical", "Major", "Minor", "Trivial"}

	if slices.Contains(allowedValues, value) {
		return nil
	}

	return fmt.Errorf("invalid severity '%s'. Available: %s",
		value, strings.Join(allowedValues, ", "))
}

// noValidation - validатор-заглушка for тегов без дополнительной validации
func noValidation(_ string) error {
	return nil
}

// ValidateEntityCreation validates title for creating сущности
func ValidateEntityCreation(tagKey, title string) error {
	trimmed := strings.TrimSpace(title)

	if trimmed == "" {
		// Capitalize first letter of tagKey for error message
		capitalizedKey := strings.ToUpper(string(tagKey[0])) + tagKey[1:]
		return fmt.Errorf("❌ %s title is required. Usage: #%s <title>",
			capitalizedKey, tagKey)
	}

	return nil
}

// ====== Task 04: Entity Management Validators ======

// ValidateStatus validates status for конкретного type сущности
// entityType должен быть "Task", "Bug" or "Epic"
// Статусы CASE-SENSITIVE
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

	return fmt.Errorf("❌ Invalid status '%s' for %s. Available: %s",
		status, entityType, strings.Join(allowedStatuses, ", "))
}

// ValidateDueDate парсит дату and returns *time.Time
// Пустое value returns nil (снятие due date)
//
//nolint:nilnil // Returning (nil, nil) is intentional for empty date (remove due date)
func ValidateDueDate(dateStr string) (*time.Time, error) {
	// Пустое value допустимо (снятие due date)
	if dateStr == "" {
		return nil, nil
	}

	// Поддерживаемые форматы (MVP)
	formats := []string{
		"2006-01-02",                // YYYY-MM-DD
		"2006-01-02T15:04",          // YYYY-MM-DDTHH:MM
		"2006-01-02T15:04:05",       // YYYY-MM-DDTHH:MM:SS
		time.RFC3339,                // YYYY-MM-DDTHH:MM:SSZ or с timezone
		"2006-01-02T15:04:05Z07:00", // with explicit timezone
	}

	// Пытаемся распарсить дату in одном from форматов
	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return &t, nil
		}
	}

	return nil, errors.New("❌ Invalid date format. Use ISO 8601: YYYY-MM-DD")
}

// ValidateTitle checks that title not empty
func ValidateTitle(title string) error {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return errors.New("❌ Title cannot be empty")
	}
	return nil
}
