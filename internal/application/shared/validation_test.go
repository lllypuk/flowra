package shared_test

import (
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/application/shared"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		wantError bool
	}{
		{
			name:      "valid non-empty string",
			field:     "title",
			value:     "Test Title",
			wantError: false,
		},
		{
			name:      "empty string",
			field:     "title",
			value:     "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := shared.ValidateRequired(tt.field, tt.value)
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.field)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		id        uuid.UUID
		wantError bool
	}{
		{
			name:      "valid UUID",
			field:     "userID",
			id:        uuid.NewUUID(),
			wantError: false,
		},
		{
			name:      "empty UUID",
			field:     "userID",
			id:        "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := shared.ValidateUUID(tt.field, tt.id)
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.field)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateMaxLength(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		maxLength int
		wantError bool
	}{
		{
			name:      "within max length",
			field:     "title",
			value:     "Short",
			maxLength: 10,
			wantError: false,
		},
		{
			name:      "exactly max length",
			field:     "title",
			value:     "Exactly10!",
			maxLength: 10,
			wantError: false,
		},
		{
			name:      "exceeds max length",
			field:     "title",
			value:     "This is too long",
			maxLength: 10,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := shared.ValidateMaxLength(tt.field, tt.value, tt.maxLength)
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.field)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateMinLength(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		minLength int
		wantError bool
	}{
		{
			name:      "meets min length",
			field:     "password",
			value:     "longpassword",
			minLength: 8,
			wantError: false,
		},
		{
			name:      "exactly min length",
			field:     "password",
			value:     "password",
			minLength: 8,
			wantError: false,
		},
		{
			name:      "below min length",
			field:     "password",
			value:     "short",
			minLength: 8,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := shared.ValidateMinLength(tt.field, tt.value, tt.minLength)
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.field)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEnum(t *testing.T) {
	allowedValues := []string{"pending", "active", "completed"}

	tests := []struct {
		name      string
		field     string
		value     string
		wantError bool
	}{
		{
			name:      "valid enum value",
			field:     "status",
			value:     "active",
			wantError: false,
		},
		{
			name:      "invalid enum value",
			field:     "status",
			value:     "invalid",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := shared.ValidateEnum(tt.field, tt.value, allowedValues)
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.field)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateDateNotPast(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour)
	future := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name      string
		field     string
		date      *time.Time
		wantError bool
	}{
		{
			name:      "nil date",
			field:     "dueDate",
			date:      nil,
			wantError: false,
		},
		{
			name:      "future date",
			field:     "dueDate",
			date:      &future,
			wantError: false,
		},
		{
			name:      "past date",
			field:     "dueDate",
			date:      &past,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := shared.ValidateDateNotPast(tt.field, tt.date)
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.field)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePositive(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     int
		wantError bool
	}{
		{
			name:      "positive value",
			field:     "count",
			value:     10,
			wantError: false,
		},
		{
			name:      "zero value",
			field:     "count",
			value:     0,
			wantError: true,
		},
		{
			name:      "negative value",
			field:     "count",
			value:     -5,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := shared.ValidatePositive(tt.field, tt.value)
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.field)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateNonNegative(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     int
		wantError bool
	}{
		{
			name:      "positive value",
			field:     "count",
			value:     10,
			wantError: false,
		},
		{
			name:      "zero value",
			field:     "count",
			value:     0,
			wantError: false,
		},
		{
			name:      "negative value",
			field:     "count",
			value:     -5,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := shared.ValidateNonNegative(tt.field, tt.value)
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.field)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		value     string
		wantError bool
	}{
		{
			name:      "valid email",
			field:     "email",
			value:     "user@example.com",
			wantError: false,
		},
		{
			name:      "empty email",
			field:     "email",
			value:     "",
			wantError: true,
		},
		{
			name:      "missing @",
			field:     "email",
			value:     "userexample.com",
			wantError: true,
		},
		{
			name:      "missing domain",
			field:     "email",
			value:     "user@",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := shared.ValidateEmail(tt.field, tt.value)
			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.field)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
