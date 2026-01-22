package tag_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/lllypuk/flowra/internal/domain/tag"
	"github.com/stretchr/testify/assert"
)

// ====== Task 06: Formatter and Bot Response Tests ======

func TestGenerateBotResponse(t *testing.T) {
	tests := []struct {
		name     string
		result   tag.ProcessingResult
		expected string
	}{
		{
			name: "no tags - no response",
			result: tag.ProcessingResult{
				AppliedTags: []tag.TagApplication{},
				Errors:      []tag.TagError{},
			},
			expected: "",
		},
		{
			name: "single success - task created",
			result: tag.ProcessingResult{
				AppliedTags: []tag.TagApplication{
					{
						TagKey:   "task",
						TagValue: "Implement OAuth",
						Command:  tag.CreateTaskCommand{ChatID: uuid.New(), Title: "Implement OAuth"},
						Success:  true,
					},
				},
				Errors: []tag.TagError{},
			},
			expected: "✅ Task created: Implement OAuth",
		},
		{
			name: "partial application",
			result: tag.ProcessingResult{
				AppliedTags: []tag.TagApplication{
					{
						TagKey:   "status",
						TagValue: "Done",
						Command:  tag.ChangeStatusCommand{ChatID: uuid.New(), Status: "Done"},
						Success:  true,
					},
					{
						TagKey:   "priority",
						TagValue: "High",
						Command:  tag.ChangePriorityCommand{ChatID: uuid.New(), Priority: "High"},
						Success:  true,
					},
				},
				Errors: []tag.TagError{
					{
						TagKey:   "assignee",
						TagValue: "@nonexistent",
						Error:    errors.New("User @nonexistent not found"),
						Severity: tag.ErrorSeverityError,
					},
				},
			},
			expected: "✅ Status changed to Done\n✅ Priority changed to High\n❌ User @nonexistent not found",
		},
		{
			name: "all errors",
			result: tag.ProcessingResult{
				Errors: []tag.TagError{
					{
						TagKey:   "status",
						TagValue: "done",
						Error:    errors.New("Invalid status 'done' for Task. Available: To Do, In Progress, Done"),
						Severity: tag.ErrorSeverityError,
					},
					{
						TagKey:   "assignee",
						TagValue: "@nobody",
						Error:    errors.New("User @nobody not found"),
						Severity: tag.ErrorSeverityError,
					},
				},
			},
			expected: "❌ Invalid status 'done' for Task. Available: To Do, In Progress, Done\n❌ User @nobody not found",
		},
		{
			name: "multiple successes",
			result: tag.ProcessingResult{
				AppliedTags: []tag.TagApplication{
					{
						TagKey:   "task",
						TagValue: "New task",
						Command:  tag.CreateTaskCommand{ChatID: uuid.New(), Title: "New task"},
						Success:  true,
					},
					{
						TagKey:   "priority",
						TagValue: "High",
						Command:  tag.ChangePriorityCommand{ChatID: uuid.New(), Priority: "High"},
						Success:  true,
					},
					{
						TagKey:   "assignee",
						TagValue: "@alex",
						Command:  tag.AssignUserCommand{ChatID: uuid.New(), Username: "@alex"},
						Success:  true,
					},
				},
				Errors: []tag.TagError{},
			},
			expected: "✅ Task created: New task\n✅ Priority changed to High\n✅ Assigned to: @alex",
		},
		{
			name: "warning severity",
			result: tag.ProcessingResult{
				AppliedTags: []tag.TagApplication{},
				Errors: []tag.TagError{
					{
						TagKey:   "severity",
						TagValue: "Critical",
						Error:    errors.New("severity is only applicable to Bugs"),
						Severity: tag.ErrorSeverityWarning,
					},
				},
			},
			expected: "⚠️ severity is only applicable to Bugs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := tt.result.GenerateBotResponse()
			assert.Equal(t, tt.expected, response)
		})
	}
}

func TestFormatSuccess_AllCommandTypes(t *testing.T) {
	chatID := uuid.New()

	tests := []struct {
		name     string
		applied  tag.TagApplication
		expected string
	}{
		{
			name: "CreateTaskCommand",
			applied: tag.TagApplication{
				TagKey:   "task",
				TagValue: "Test task",
				Command:  tag.CreateTaskCommand{ChatID: chatID, Title: "Test task"},
				Success:  true,
			},
			expected: "✅ Task created: Test task",
		},
		{
			name: "CreateBugCommand",
			applied: tag.TagApplication{
				TagKey:   "bug",
				TagValue: "Test bug",
				Command:  tag.CreateBugCommand{ChatID: chatID, Title: "Test bug"},
				Success:  true,
			},
			expected: "✅ Bug created: Test bug",
		},
		{
			name: "CreateEpicCommand",
			applied: tag.TagApplication{
				TagKey:   "epic",
				TagValue: "Test epic",
				Command:  tag.CreateEpicCommand{ChatID: chatID, Title: "Test epic"},
				Success:  true,
			},
			expected: "✅ Epic created: Test epic",
		},
		{
			name: "ChangeStatusCommand",
			applied: tag.TagApplication{
				TagKey:   "status",
				TagValue: "In Progress",
				Command:  tag.ChangeStatusCommand{ChatID: chatID, Status: "In Progress"},
				Success:  true,
			},
			expected: "✅ Status changed to In Progress",
		},
		{
			name: "AssignUserCommand - assign",
			applied: tag.TagApplication{
				TagKey:   "assignee",
				TagValue: "@bob",
				Command:  tag.AssignUserCommand{ChatID: chatID, Username: "@bob"},
				Success:  true,
			},
			expected: "✅ Assigned to: @bob",
		},
		{
			name: "AssignUserCommand - remove (@none)",
			applied: tag.TagApplication{
				TagKey:   "assignee",
				TagValue: "@none",
				Command:  tag.AssignUserCommand{ChatID: chatID, Username: "@none"},
				Success:  true,
			},
			expected: "✅ Assignee removed",
		},
		{
			name: "AssignUserCommand - remove (empty)",
			applied: tag.TagApplication{
				TagKey:   "assignee",
				TagValue: "",
				Command:  tag.AssignUserCommand{ChatID: chatID, Username: ""},
				Success:  true,
			},
			expected: "✅ Assignee removed",
		},
		{
			name: "ChangePriorityCommand",
			applied: tag.TagApplication{
				TagKey:   "priority",
				TagValue: "High",
				Command:  tag.ChangePriorityCommand{ChatID: chatID, Priority: "High"},
				Success:  true,
			},
			expected: "✅ Priority changed to High",
		},
		{
			name: "SetDueDateCommand - set",
			applied: tag.TagApplication{
				TagKey:   "due",
				TagValue: "2025-10-20",
				Command:  tag.SetDueDateCommand{ChatID: chatID},
				Success:  true,
			},
			expected: "✅ Due date set to 2025-10-20",
		},
		{
			name: "SetDueDateCommand - remove",
			applied: tag.TagApplication{
				TagKey:   "due",
				TagValue: "",
				Command:  tag.SetDueDateCommand{ChatID: chatID},
				Success:  true,
			},
			expected: "✅ Due date removed",
		},
		{
			name: "ChangeTitleCommand",
			applied: tag.TagApplication{
				TagKey:   "title",
				TagValue: "New title",
				Command:  tag.ChangeTitleCommand{ChatID: chatID, Title: "New title"},
				Success:  true,
			},
			expected: "✅ Title changed to: New title",
		},
		{
			name: "SetSeverityCommand",
			applied: tag.TagApplication{
				TagKey:   "severity",
				TagValue: "Critical",
				Command:  tag.SetSeverityCommand{ChatID: chatID, Severity: "Critical"},
				Success:  true,
			},
			expected: "✅ Severity set to Critical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tag.ProcessingResult{
				AppliedTags: []tag.TagApplication{tt.applied},
			}
			response := result.GenerateBotResponse()
			assert.Equal(t, tt.expected, response)
		})
	}
}

func TestProcessingResult_HelperMethods(t *testing.T) {
	t.Run("HasTags - true with applied tags", func(t *testing.T) {
		result := tag.ProcessingResult{
			AppliedTags: []tag.TagApplication{
				{TagKey: "task", Success: true},
			},
		}
		assert.True(t, result.HasTags())
	})

	t.Run("HasTags - true with errors", func(t *testing.T) {
		result := tag.ProcessingResult{
			Errors: []tag.TagError{
				{TagKey: "status", Error: errors.New("error")},
			},
		}
		assert.True(t, result.HasTags())
	})

	t.Run("HasTags - false with no tags", func(t *testing.T) {
		result := tag.ProcessingResult{}
		assert.False(t, result.HasTags())
	})

	t.Run("HasErrors - true", func(t *testing.T) {
		result := tag.ProcessingResult{
			Errors: []tag.TagError{
				{TagKey: "status", Error: errors.New("error")},
			},
		}
		assert.True(t, result.HasErrors())
	})

	t.Run("HasErrors - false", func(t *testing.T) {
		result := tag.ProcessingResult{}
		assert.False(t, result.HasErrors())
	})

	t.Run("SuccessCount", func(t *testing.T) {
		result := tag.ProcessingResult{
			AppliedTags: []tag.TagApplication{
				{TagKey: "task", Success: true},
				{TagKey: "bug", Success: true},
				{TagKey: "epic", Success: false}, // not dolzhen schitatsya
			},
		}
		assert.Equal(t, 2, result.SuccessCount())
	})
}
