package tag

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// usernameRegex определяет допустимый формат username: @[a-zA-Z0-9._-]+
var usernameRegex = regexp.MustCompile(`^@[a-zA-Z0-9._-]+$`)

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
