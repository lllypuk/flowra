package tag //nolint:testpackage // to test unexported functions

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
		// valid values
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

		// Invalid values
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
		// valid formats
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

		// Invalid formats
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
		name          string
		value         string
		wantErr       bool
		errSubstr     string
		wantCanonical string
	}{
		// valid values (case-insensitive)
		{
			name:          "High",
			value:         "High",
			wantCanonical: "High",
		},
		{
			name:          "Medium",
			value:         "Medium",
			wantCanonical: "Medium",
		},
		{
			name:          "Low",
			value:         "Low",
			wantCanonical: "Low",
		},
		{
			name:          "lowercase high",
			value:         "high",
			wantCanonical: "High",
		},
		{
			name:          "UPPERCASE HIGH",
			value:         "HIGH",
			wantCanonical: "High",
		},

		// Invalid values
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
			canonical, err := validatePriority(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantCanonical, canonical)
			}
		})
	}
}

func TestValidateSeverity(t *testing.T) {
	tests := []struct {
		name          string
		value         string
		wantErr       bool
		errSubstr     string
		wantCanonical string
	}{
		// valid values (case-insensitive)
		{
			name:          "Critical",
			value:         "Critical",
			wantCanonical: "Critical",
		},
		{
			name:          "Major",
			value:         "Major",
			wantCanonical: "Major",
		},
		{
			name:          "Minor",
			value:         "Minor",
			wantCanonical: "Minor",
		},
		{
			name:          "Trivial",
			value:         "Trivial",
			wantCanonical: "Trivial",
		},
		{
			name:          "lowercase critical",
			value:         "critical",
			wantCanonical: "Critical",
		},
		{
			name:          "UPPERCASE CRITICAL",
			value:         "CRITICAL",
			wantCanonical: "Critical",
		},

		// Invalid values
		{
			name:      "invalid value High",
			value:     "High",
			wantErr:   true,
			errSubstr: "invalid severity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonical, err := validateSeverity(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantCanonical, canonical)
			}
		})
	}
}

func TestNoValidation(t *testing.T) {
	// noValidation always returns nil
	assert.NoError(t, noValidation(""))
	assert.NoError(t, noValidation("any value"))
	assert.NoError(t, noValidation("🎉 emoji"))
}

// ====== Task 03: Entity Creation Validation Tests ======

// ====== Task 04: Entity Management Validators Tests ======

func TestValidateStatus(t *testing.T) {
	tests := []struct {
		name          string
		entityType    string
		status        string
		wantErr       bool
		errSubstr     string
		wantCanonical string
	}{
		// Task statuses
		{
			name:          "valid Task status - To Do",
			entityType:    "Task",
			status:        "To Do",
			wantCanonical: "To Do",
		},
		{
			name:          "valid Task status - In Progress",
			entityType:    "Task",
			status:        "In Progress",
			wantCanonical: "In Progress",
		},
		{
			name:          "valid Task status - Done",
			entityType:    "Task",
			status:        "Done",
			wantCanonical: "Done",
		},
		{
			name:          "case-insensitive Task status - lowercase",
			entityType:    "Task",
			status:        "done",
			wantCanonical: "Done",
		},
		{
			name:          "case-insensitive Task status - in progress lowercase",
			entityType:    "Task",
			status:        "in progress",
			wantCanonical: "In Progress",
		},
		{
			name:       "invalid Task status - wrong value",
			entityType: "Task",
			status:     "Completed",
			wantErr:    true,
			errSubstr:  "invalid status",
		},

		// Bug statuses
		{
			name:          "valid Bug status - New",
			entityType:    "Bug",
			status:        "New",
			wantCanonical: "New",
		},
		{
			name:          "valid Bug status - Investigating",
			entityType:    "Bug",
			status:        "Investigating",
			wantCanonical: "Investigating",
		},
		{
			name:          "valid Bug status - Fixed",
			entityType:    "Bug",
			status:        "Fixed",
			wantCanonical: "Fixed",
		},
		{
			name:          "valid Bug status - Verified",
			entityType:    "Bug",
			status:        "Verified",
			wantCanonical: "Verified",
		},
		{
			name:       "invalid Bug status - Task status",
			entityType: "Bug",
			status:     "Done",
			wantErr:    true,
			errSubstr:  "invalid status",
		},

		// Epic statuses
		{
			name:          "valid Epic status - Planned",
			entityType:    "Epic",
			status:        "Planned",
			wantCanonical: "Planned",
		},
		{
			name:          "valid Epic status - In Progress",
			entityType:    "Epic",
			status:        "In Progress",
			wantCanonical: "In Progress",
		},
		{
			name:          "valid Epic status - Completed",
			entityType:    "Epic",
			status:        "Completed",
			wantCanonical: "Completed",
		},
		{
			name:          "case-insensitive Epic status",
			entityType:    "Epic",
			status:        "completed",
			wantCanonical: "Completed",
		},

		// unknown entity type
		{
			name:       "unknown entity type",
			entityType: "Story",
			status:     "In Progress",
			wantErr:    true,
			errSubstr:  "unknown entity type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonical, err := ValidateStatus(tt.entityType, tt.status)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantCanonical, canonical)
			}
		})
	}
}

func TestValidateDueDate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantNil   bool
		wantErr   bool
		errSubstr string
	}{
		// valid formats
		{
			name:    "YYYY-MM-DD",
			input:   "2025-10-20",
			wantNil: false,
			wantErr: false,
		},
		{
			name:    "YYYY-MM-DDTHH:MM",
			input:   "2025-10-20T15:30",
			wantNil: false,
			wantErr: false,
		},
		{
			name:    "YYYY-MM-DDTHH:MM:SS",
			input:   "2025-10-20T15:30:00",
			wantNil: false,
			wantErr: false,
		},
		{
			name:    "YYYY-MM-DDTHH:MM:SSZ (UTC)",
			input:   "2025-10-20T15:30:00Z",
			wantNil: false,
			wantErr: false,
		},
		{
			name:    "with timezone +03:00",
			input:   "2025-10-20T15:30:00+03:00",
			wantNil: false,
			wantErr: false,
		},
		{
			name:    "with timezone -05:00",
			input:   "2025-10-20T15:30:00-05:00",
			wantNil: false,
			wantErr: false,
		},
		{
			name:    "empty value (remove due date)",
			input:   "",
			wantNil: true,
			wantErr: false,
		},

		// Invalid formats
		{
			name:      "DD-MM-YYYY",
			input:     "20-10-2025",
			wantErr:   true,
			errSubstr: "invalid date format",
		},
		{
			name:      "MM/DD/YYYY",
			input:     "10/20/2025",
			wantErr:   true,
			errSubstr: "invalid date format",
		},
		{
			name:      "natural language",
			input:     "tomorrow",
			wantErr:   true,
			errSubstr: "invalid date format",
		},
		{
			name:      "random text",
			input:     "not a date",
			wantErr:   true,
			errSubstr: "invalid date format",
		},
		{
			name:      "partial date",
			input:     "2025-10",
			wantErr:   true,
			errSubstr: "invalid date format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateDueDate(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errSubstr != "" {
					assert.Contains(t, err.Error(), tt.errSubstr)
				}
			} else {
				assert.NoError(t, err)
				if tt.wantNil {
					assert.Nil(t, result)
				} else {
					assert.NotNil(t, result)
				}
			}
		})
	}
}

func TestValidateTitle(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{
			name:    "valid title",
			title:   "Some title",
			wantErr: false,
		},
		{
			name:    "title with leading/trailing spaces",
			title:   "  Spaces  ",
			wantErr: false,
		},
		{
			name:    "empty title",
			title:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			title:   "   ",
			wantErr: true,
		},
		{
			name:    "tabs only",
			title:   "\t\t",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTitle(tt.title)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "title cannot be empty")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEntityCreation(t *testing.T) {
	tests := []struct {
		name      string
		tagKey    string
		title     string
		wantErr   bool
		errSubstr string
	}{
		// valid titles
		{
			name:    "valid task title",
			tagKey:  "task",
			title:   "Implement authorization",
			wantErr: false,
		},
		{
			name:    "valid bug title",
			tagKey:  "bug",
			title:   "Login error",
			wantErr: false,
		},
		{
			name:    "valid epic title",
			tagKey:  "epic",
			title:   "New feature",
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
			title:   "Fix bug 🐛",
			wantErr: false,
		},
		{
			name:    "title with leading/trailing spaces",
			tagKey:  "task",
			title:   "  many spaces  ",
			wantErr: false,
		},

		// Invalid title
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
