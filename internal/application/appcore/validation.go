package appcore

import (
	"fmt"
	"slices"
	"time"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

const (
	// MaxTitleLength is the maximum length of a chat/task title
	MaxTitleLength = 200
)

// ValidateRequired checks that the string is not empty
func ValidateRequired(field, value string) error {
	if value == "" {
		return NewValidationError(field, "is required")
	}
	return nil
}

// ValidateUUID checks that the UUID is valid and not empty
func ValidateUUID(field string, id uuid.UUID) error {
	if id.IsZero() {
		return NewValidationError(field, "must be a valid UUID")
	}
	return nil
}

// ValidateMaxLength checks the maximum string length
func ValidateMaxLength(field, value string, maxLength int) error {
	if len(value) > maxLength {
		return NewValidationError(field, fmt.Sprintf("must be at most %d characters", maxLength))
	}
	return nil
}

// ValidateMinLength checks the minimum string length
func ValidateMinLength(field, value string, minLength int) error {
	if len(value) < minLength {
		return NewValidationError(field, fmt.Sprintf("must be at least %d characters", minLength))
	}
	return nil
}

// ValidateEnum checks that the value is in the list of allowed values
func ValidateEnum(field, value string, allowedValues []string) error {
	if slices.Contains(allowedValues, value) {
		return nil
	}
	return NewValidationError(field, fmt.Sprintf("must be one of: %v", allowedValues))
}

// ValidateDateNotPast checks that the date is not in the past
func ValidateDateNotPast(field string, date *time.Time) error {
	if date != nil && date.Before(time.Now()) {
		return NewValidationError(field, "cannot be in the past")
	}
	return nil
}

// ValidateDateRange checks that the date is within the allowed range
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

// ValidatePositive checks that the number is positive
func ValidatePositive(field string, value int) error {
	if value <= 0 {
		return NewValidationError(field, "must be positive")
	}
	return nil
}

// ValidateNonNegative checks that the number is non-negative
func ValidateNonNegative(field string, value int) error {
	if value < 0 {
		return NewValidationError(field, "must be non-negative")
	}
	return nil
}

// ValidateRange checks that the value is within the specified range
func ValidateRange(field string, value, minValue, maxValue int) error {
	if value < minValue || value > maxValue {
		return NewValidationError(field, fmt.Sprintf("must be between %d and %d", minValue, maxValue))
	}
	return nil
}

// ValidateEmail checks basic email format (simplified check)
func ValidateEmail(field, value string) error {
	if value == "" {
		return NewValidationError(field, "email is required")
	}
	// Simple check for @ and dot presence
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
