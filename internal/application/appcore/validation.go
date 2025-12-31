package appcore

import (
	"fmt"
	"slices"
	"time"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

const (
	// MaxTitleLength максимальная длина заголовка чата/задачи
	MaxTitleLength = 200
)

// ValidateRequired проверяет, что строка не пустая
func ValidateRequired(field, value string) error {
	if value == "" {
		return NewValidationError(field, "is required")
	}
	return nil
}

// ValidateUUID проверяет, что UUID валиден и не пустой
func ValidateUUID(field string, id uuid.UUID) error {
	if id.IsZero() {
		return NewValidationError(field, "must be a valid UUID")
	}
	return nil
}

// ValidateMaxLength проверяет максимальную длину строки
func ValidateMaxLength(field, value string, maxLength int) error {
	if len(value) > maxLength {
		return NewValidationError(field, fmt.Sprintf("must be at most %d characters", maxLength))
	}
	return nil
}

// ValidateMinLength проверяет минимальную длину строки
func ValidateMinLength(field, value string, minLength int) error {
	if len(value) < minLength {
		return NewValidationError(field, fmt.Sprintf("must be at least %d characters", minLength))
	}
	return nil
}

// ValidateEnum проверяет, что значение находится в списке допустимых
func ValidateEnum(field, value string, allowedValues []string) error {
	if slices.Contains(allowedValues, value) {
		return nil
	}
	return NewValidationError(field, fmt.Sprintf("must be one of: %v", allowedValues))
}

// ValidateDateNotPast проверяет, что дата не в прошлом
func ValidateDateNotPast(field string, date *time.Time) error {
	if date != nil && date.Before(time.Now()) {
		return NewValidationError(field, "cannot be in the past")
	}
	return nil
}

// ValidateDateRange проверяет, что дата находится в допустимом диапазоне
func ValidateDateRange(field string, date *time.Time, minDate, maxDate time.Time) error {
	if date == nil {
		return nil
	}
	if date.Before(minDate) || date.After(maxDate) {
		return NewValidationError(
			field,
			fmt.Sprintf(
				"must be between %s and %s",
				minDate.Format(time.RFC3339),
				maxDate.Format(time.RFC3339),
			),
		)
	}
	return nil
}

// ValidatePositive проверяет, что число положительное
func ValidatePositive(field string, value int) error {
	if value <= 0 {
		return NewValidationError(field, "must be positive")
	}
	return nil
}

// ValidateNonNegative проверяет, что число неотрицательное
func ValidateNonNegative(field string, value int) error {
	if value < 0 {
		return NewValidationError(field, "must be non-negative")
	}
	return nil
}

// ValidateRange проверяет, что значение находится в заданном диапазоне
func ValidateRange(field string, value, minValue, maxValue int) error {
	if value < minValue || value > maxValue {
		return NewValidationError(field, fmt.Sprintf("must be between %d and %d", minValue, maxValue))
	}
	return nil
}

// ValidateEmail проверяет базовый формат email (упрощенная проверка)
func ValidateEmail(field, value string) error {
	if value == "" {
		return NewValidationError(field, "email is required")
	}
	// Простая проверка наличия @ и точки
	hasAt := false
	hasDot := false
	for i, ch := range value {
		if ch == '@' {
			hasAt = true
		}
		if hasAt && ch == '.' && i > 0 && i < len(value)-1 {
			hasDot = true
		}
	}
	if !hasAt || !hasDot {
		return NewValidationError(field, "must be a valid email address")
	}
	return nil
}
