package tag

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ErrNoActiveEntity возвращается когда Entity Management Tag используется без активной сущности
var ErrNoActiveEntity = errors.New("❌ No active entity to modify. Create an entity first with #task, #bug, or #epic")

// usernameRegex определяет допустимый формат username: @[a-zA-Z0-9._-]+
var usernameRegex = regexp.MustCompile(`^@[a-zA-Z0-9._-]+$`)

// ====== Task 04: Status Validation Constants ======

//nolint:gochecknoglobals // Domain constants for entity statuses
var (
	// TaskStatuses - допустимые статусы для Task (CASE-SENSITIVE)
	TaskStatuses = []string{"To Do", "In Progress", "Done"}

	// BugStatuses - допустимые статусы для Bug (CASE-SENSITIVE)
	BugStatuses = []string{"New", "Investigating", "Fixed", "Verified"}

	// EpicStatuses - допустимые статусы для Epic (CASE-SENSITIVE)
	EpicStatuses = []string{"Planned", "In Progress", "Completed"}
)

// validateUsername проверяет формат username (@username)
func validateUsername(value string) error {
	// Пустое значение допустимо (снятие assignee)
	if value == "" || value == "@none" {
		return nil
	}

	// Проверка наличия @
	if !strings.HasPrefix(value, "@") {
		return errors.New("invalid assignee format. Use @username")
	}

	// Проверка что после @ есть имя
	if len(value) == 1 {
		return errors.New("invalid assignee format. Use @username")
	}

	// Проверка формата username
	if !usernameRegex.MatchString(value) {
		return errors.New("invalid assignee format. Use @username")
	}

	return nil
}

// validateISODate проверяет формат даты ISO 8601
func validateISODate(value string) error {
	// Пустое значение допустимо (снятие due date)
	if value == "" {
		return nil
	}

	// Поддерживаемые форматы (MVP)
	formats := []string{
		"2006-01-02",                // YYYY-MM-DD
		"2006-01-02T15:04",          // YYYY-MM-DDTHH:MM
		"2006-01-02T15:04:05",       // YYYY-MM-DDTHH:MM:SS
		time.RFC3339,                // YYYY-MM-DDTHH:MM:SSZ или с timezone
		"2006-01-02T15:04:05Z07:00", // с explicit timezone
	}

	// Пытаемся распарсить дату в одном из форматов
	for _, format := range formats {
		if _, err := time.Parse(format, value); err == nil {
			return nil
		}
	}

	return errors.New("invalid date format. Use ISO 8601: YYYY-MM-DD")
}

// validatePriority проверяет значение приоритета
func validatePriority(value string) error {
	allowedValues := []string{"High", "Medium", "Low"}

	for _, allowed := range allowedValues {
		if value == allowed {
			return nil
		}
	}

	return fmt.Errorf("invalid priority '%s'. Available: %s",
		value, strings.Join(allowedValues, ", "))
}

// validateSeverity проверяет значение серьезности бага
func validateSeverity(value string) error {
	allowedValues := []string{"Critical", "Major", "Minor", "Trivial"}

	for _, allowed := range allowedValues {
		if value == allowed {
			return nil
		}
	}

	return fmt.Errorf("invalid severity '%s'. Available: %s",
		value, strings.Join(allowedValues, ", "))
}

// noValidation - валидатор-заглушка для тегов без дополнительной валидации
func noValidation(_ string) error {
	return nil
}

// ValidateEntityCreation проверяет валидность title для создания сущности
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

// ValidateStatus проверяет валидность статуса для конкретного типа сущности
// entityType должен быть "Task", "Bug" или "Epic"
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

	for _, allowed := range allowedStatuses {
		if status == allowed {
			return nil
		}
	}

	return fmt.Errorf("❌ Invalid status '%s' for %s. Available: %s",
		status, entityType, strings.Join(allowedStatuses, ", "))
}

// ValidateDueDate парсит дату и возвращает *time.Time
// Пустое значение возвращает nil (снятие due date)
//
//nolint:nilnil // Returning (nil, nil) is intentional for empty date (remove due date)
func ValidateDueDate(dateStr string) (*time.Time, error) {
	// Пустое значение допустимо (снятие due date)
	if dateStr == "" {
		return nil, nil
	}

	// Поддерживаемые форматы (MVP)
	formats := []string{
		"2006-01-02",                // YYYY-MM-DD
		"2006-01-02T15:04",          // YYYY-MM-DDTHH:MM
		"2006-01-02T15:04:05",       // YYYY-MM-DDTHH:MM:SS
		time.RFC3339,                // YYYY-MM-DDTHH:MM:SSZ или с timezone
		"2006-01-02T15:04:05Z07:00", // с explicit timezone
	}

	// Пытаемся распарсить дату в одном из форматов
	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return &t, nil
		}
	}

	return nil, errors.New("❌ Invalid date format. Use ISO 8601: YYYY-MM-DD")
}

// ValidateTitle проверяет что title не пустой
func ValidateTitle(title string) error {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return errors.New("❌ Title cannot be empty")
	}
	return nil
}
