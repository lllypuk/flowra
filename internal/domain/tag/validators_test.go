package tag //nolint:testpackage // Чтобы тестировать unexported функции

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantErr   bool
		errSubstr string
	}{
		// Валидные значения
		{
			name:    "valid username",
			value:   "@alex",
			wantErr: false,
		},
		{
			name:    "username with dots",
			value:   "@user.name",
			wantErr: false,
		},
		{
			name:    "username with dashes",
			value:   "@user-name",
			wantErr: false,
		},
		{
			name:    "username with underscores",
			value:   "@user_name",
			wantErr: false,
		},
		{
			name:    "username with numbers",
			value:   "@user123",
			wantErr: false,
		},
		{
			name:    "empty value (remove assignee)",
			value:   "",
			wantErr: false,
		},
		{
			name:    "@none (remove assignee)",
			value:   "@none",
			wantErr: false,
		},

		// Невалидные значения
		{
			name:      "missing @",
			value:     "alex",
			wantErr:   true,
			errSubstr: "invalid assignee format",
		},
		{
			name:      "only @",
			value:     "@",
			wantErr:   true,
			errSubstr: "invalid assignee format",
		},
		{
			name:      "@ with space",
			value:     "@ alex",
			wantErr:   true,
			errSubstr: "invalid assignee format",
		},
		{
			name:      "username with spaces",
			value:     "@user name",
			wantErr:   true,
			errSubstr: "invalid assignee format",
		},
		{
			name:      "username with special chars",
			value:     "@user!name",
			wantErr:   true,
			errSubstr: "invalid assignee format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUsername(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateISODate(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantErr   bool
		errSubstr string
	}{
		// Валидные форматы
		{
			name:    "YYYY-MM-DD",
			value:   "2025-10-20",
			wantErr: false,
		},
		{
			name:    "YYYY-MM-DDTHH:MM",
			value:   "2025-10-20T15:30",
			wantErr: false,
		},
		{
			name:    "YYYY-MM-DDTHH:MM:SS",
			value:   "2025-10-20T15:30:00",
			wantErr: false,
		},
		{
			name:    "YYYY-MM-DDTHH:MM:SSZ (UTC)",
			value:   "2025-10-20T15:30:00Z",
			wantErr: false,
		},
		{
			name:    "with timezone +03:00",
			value:   "2025-10-20T15:30:00+03:00",
			wantErr: false,
		},
		{
			name:    "with timezone -05:00",
			value:   "2025-10-20T15:30:00-05:00",
			wantErr: false,
		},
		{
			name:    "empty value (remove due date)",
			value:   "",
			wantErr: false,
		},

		// Невалидные форматы
		{
			name:      "DD-MM-YYYY",
			value:     "20-10-2025",
			wantErr:   true,
			errSubstr: "invalid date format",
		},
		{
			name:      "MM/DD/YYYY",
			value:     "10/20/2025",
			wantErr:   true,
			errSubstr: "invalid date format",
		},
		{
			name:      "natural language",
			value:     "tomorrow",
			wantErr:   true,
			errSubstr: "invalid date format",
		},
		{
			name:      "random text",
			value:     "not a date",
			wantErr:   true,
			errSubstr: "invalid date format",
		},
		{
			name:      "partial date",
			value:     "2025-10",
			wantErr:   true,
			errSubstr: "invalid date format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateISODate(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePriority(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantErr   bool
		errSubstr string
	}{
		// Валидные значения (CASE-SENSITIVE)
		{
			name:    "High",
			value:   "High",
			wantErr: false,
		},
		{
			name:    "Medium",
			value:   "Medium",
			wantErr: false,
		},
		{
			name:    "Low",
			value:   "Low",
			wantErr: false,
		},

		// Невалидные значения
		{
			name:      "lowercase high",
			value:     "high",
			wantErr:   true,
			errSubstr: "invalid priority",
		},
		{
			name:      "UPPERCASE HIGH",
			value:     "HIGH",
			wantErr:   true,
			errSubstr: "invalid priority",
		},
		{
			name:      "invalid value Urgent",
			value:     "Urgent",
			wantErr:   true,
			errSubstr: "invalid priority",
		},
		{
			name:      "invalid value Critical",
			value:     "Critical",
			wantErr:   true,
			errSubstr: "invalid priority",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePriority(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSeverity(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantErr   bool
		errSubstr string
	}{
		// Валидные значения (CASE-SENSITIVE)
		{
			name:    "Critical",
			value:   "Critical",
			wantErr: false,
		},
		{
			name:    "Major",
			value:   "Major",
			wantErr: false,
		},
		{
			name:    "Minor",
			value:   "Minor",
			wantErr: false,
		},
		{
			name:    "Trivial",
			value:   "Trivial",
			wantErr: false,
		},

		// Невалидные значения
		{
			name:      "lowercase critical",
			value:     "critical",
			wantErr:   true,
			errSubstr: "invalid severity",
		},
		{
			name:      "UPPERCASE CRITICAL",
			value:     "CRITICAL",
			wantErr:   true,
			errSubstr: "invalid severity",
		},
		{
			name:      "invalid value High",
			value:     "High",
			wantErr:   true,
			errSubstr: "invalid severity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSeverity(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNoValidation(t *testing.T) {
	// noValidation всегда возвращает nil
	assert.NoError(t, noValidation(""))
	assert.NoError(t, noValidation("any value"))
	assert.NoError(t, noValidation("🎉 emoji"))
}

// ====== Task 03: Entity Creation Validation Tests ======

func TestValidateEntityCreation(t *testing.T) {
	tests := []struct {
		name      string
		tagKey    string
		title     string
		wantErr   bool
		errSubstr string
	}{
		// Валидные title
		{
			name:    "valid task title",
			tagKey:  "task",
			title:   "Реализовать авторизацию",
			wantErr: false,
		},
		{
			name:    "valid bug title",
			tagKey:  "bug",
			title:   "Ошибка при логине",
			wantErr: false,
		},
		{
			name:    "valid epic title",
			tagKey:  "epic",
			title:   "Новая фича",
			wantErr: false,
		},
		{
			name:    "title with special chars",
			tagKey:  "bug",
			title:   "Fix issue #123 (critical!)",
			wantErr: false,
		},
		{
			name:    "title with unicode",
			tagKey:  "task",
			title:   "Исправить баг 🐛",
			wantErr: false,
		},
		{
			name:    "title with leading/trailing spaces",
			tagKey:  "task",
			title:   "  Много пробелов  ",
			wantErr: false,
		},

		// Невалидные title
		{
			name:      "empty task title",
			tagKey:    "task",
			title:     "",
			wantErr:   true,
			errSubstr: "Task title is required",
		},
		{
			name:      "empty bug title",
			tagKey:    "bug",
			title:     "",
			wantErr:   true,
			errSubstr: "Bug title is required",
		},
		{
			name:      "empty epic title",
			tagKey:    "epic",
			title:     "",
			wantErr:   true,
			errSubstr: "Epic title is required",
		},
		{
			name:      "whitespace only",
			tagKey:    "task",
			title:     "   ",
			wantErr:   true,
			errSubstr: "title is required",
		},
		{
			name:      "tabs only",
			tagKey:    "task",
			title:     "\t\t",
			wantErr:   true,
			errSubstr: "title is required",
		},
		{
			name:      "newlines only",
			tagKey:    "task",
			title:     "\n\n",
			wantErr:   true,
			errSubstr: "title is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEntityCreation(tt.tagKey, tt.title)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
