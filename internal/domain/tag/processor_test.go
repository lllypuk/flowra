package tag_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lllypuk/teams-up/internal/domain/tag"
	"github.com/stretchr/testify/assert"
)

// ====== Task 03: TagProcessor Tests ======

func TestNewProcessor(t *testing.T) {
	processor := tag.NewProcessor()

	assert.NotNil(t, processor)
}

func TestProcessTags_EntityCreation(t *testing.T) {
	processor := tag.NewProcessor()
	chatID := uuid.New()

	tests := []struct {
		name         string
		tags         []tag.ParsedTag
		entityType   string // current entity type
		wantCommands int
		wantErrors   int
		checkCommand func(t *testing.T, cmd tag.Command)
	}{
		{
			name:       "create task",
			entityType: "",
			tags: []tag.ParsedTag{
				{Key: "task", Value: "Реализовать авторизацию"},
			},
			wantCommands: 1,
			wantErrors:   0,
			checkCommand: func(t *testing.T, cmd tag.Command) {
				taskCmd, ok := cmd.(tag.CreateTaskCommand)
				assert.True(t, ok, "command should be CreateTaskCommand")
				assert.Equal(t, chatID, taskCmd.ChatID)
				assert.Equal(t, "Реализовать авторизацию", taskCmd.Title)
			},
		},
		{
			name:       "create bug",
			entityType: "",
			tags: []tag.ParsedTag{
				{Key: "bug", Value: "Ошибка при логине"},
			},
			wantCommands: 1,
			wantErrors:   0,
			checkCommand: func(t *testing.T, cmd tag.Command) {
				bugCmd, ok := cmd.(tag.CreateBugCommand)
				assert.True(t, ok, "command should be CreateBugCommand")
				assert.Equal(t, chatID, bugCmd.ChatID)
				assert.Equal(t, "Ошибка при логине", bugCmd.Title)
			},
		},
		{
			name:       "create epic",
			entityType: "",
			tags: []tag.ParsedTag{
				{Key: "epic", Value: "Новая фича"},
			},
			wantCommands: 1,
			wantErrors:   0,
			checkCommand: func(t *testing.T, cmd tag.Command) {
				epicCmd, ok := cmd.(tag.CreateEpicCommand)
				assert.True(t, ok, "command should be CreateEpicCommand")
				assert.Equal(t, chatID, epicCmd.ChatID)
				assert.Equal(t, "Новая фича", epicCmd.Title)
			},
		},
		{
			name:       "task with leading/trailing spaces",
			entityType: "",
			tags: []tag.ParsedTag{
				{Key: "task", Value: "   Много пробелов   "},
			},
			wantCommands: 1,
			wantErrors:   0,
			checkCommand: func(t *testing.T, cmd tag.Command) {
				taskCmd, ok := cmd.(tag.CreateTaskCommand)
				assert.True(t, ok)
				assert.Equal(t, "Много пробелов", taskCmd.Title, "spaces should be trimmed")
			},
		},
		{
			name:       "task with special characters",
			entityType: "",
			tags: []tag.ParsedTag{
				{Key: "task", Value: "Fix issue #123 (critical!)"},
			},
			wantCommands: 1,
			wantErrors:   0,
			checkCommand: func(t *testing.T, cmd tag.Command) {
				taskCmd, ok := cmd.(tag.CreateTaskCommand)
				assert.True(t, ok)
				assert.Equal(t, "Fix issue #123 (critical!)", taskCmd.Title)
			},
		},
		{
			name:       "empty task title - error",
			entityType: "",
			tags: []tag.ParsedTag{
				{Key: "task", Value: ""},
			},
			wantCommands: 0,
			wantErrors:   1,
		},
		{
			name:       "whitespace only title - error",
			entityType: "",
			tags: []tag.ParsedTag{
				{Key: "bug", Value: "   "},
			},
			wantCommands: 0,
			wantErrors:   1,
		},
		{
			name:       "multiple entity creation commands",
			entityType: "",
			tags: []tag.ParsedTag{
				{Key: "task", Value: "Task 1"},
				{Key: "bug", Value: "Bug 1"},
				{Key: "epic", Value: "Epic 1"},
			},
			wantCommands: 3,
			wantErrors:   0,
		},
		{
			name:       "mix of valid and invalid",
			entityType: "",
			tags: []tag.ParsedTag{
				{Key: "task", Value: "Valid task"},
				{Key: "bug", Value: ""},
				{Key: "epic", Value: "Valid epic"},
			},
			wantCommands: 2,
			wantErrors:   1,
		},
		{
			name:       "unknown tags ignored",
			entityType: "",
			tags: []tag.ParsedTag{
				{Key: "task", Value: "Task"},
				{Key: "unknown", Value: "Something"},
				{Key: "bug", Value: "Bug"},
			},
			wantCommands: 2,
			wantErrors:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.ProcessTags(chatID, tt.tags, tt.entityType)

			assert.Len(t, result.AppliedTags, tt.wantCommands, "unexpected number of commands")
			assert.Len(t, result.Errors, tt.wantErrors, "unexpected number of errors")

			if tt.checkCommand != nil && len(result.AppliedTags) > 0 {
				tt.checkCommand(t, result.AppliedTags[0].Command)
			}

			// Check error messages
			if tt.wantErrors > 0 {
				for _, err := range result.Errors {
					assert.Contains(t, err.Error.Error(), "title is required")
				}
			}
		})
	}
}

func TestCommandType(t *testing.T) {
	chatID := uuid.New()

	tests := []struct {
		name     string
		command  tag.Command
		wantType string
	}{
		{
			name:     "CreateTaskCommand",
			command:  tag.CreateTaskCommand{ChatID: chatID, Title: "Test"},
			wantType: "CreateTask",
		},
		{
			name:     "CreateBugCommand",
			command:  tag.CreateBugCommand{ChatID: chatID, Title: "Test"},
			wantType: "CreateBug",
		},
		{
			name:     "CreateEpicCommand",
			command:  tag.CreateEpicCommand{ChatID: chatID, Title: "Test"},
			wantType: "CreateEpic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantType, tt.command.CommandType())
		})
	}
}

// ====== Task 04: Entity Management Tests ======

func TestProcessTags_EntityManagement(t *testing.T) {
	processor := tag.NewProcessor()
	chatID := uuid.New()

	tests := []struct {
		name         string
		tags         []tag.ParsedTag
		entityType   string // current entity type
		wantCommands int
		wantErrors   int
		checkCommand func(t *testing.T, cmd tag.Command)
	}{
		// #status tests
		{
			name:       "change Task status - valid",
			entityType: "Task",
			tags: []tag.ParsedTag{
				{Key: "status", Value: "In Progress"},
			},
			wantCommands: 1,
			wantErrors:   0,
			checkCommand: func(t *testing.T, cmd tag.Command) {
				statusCmd, ok := cmd.(tag.ChangeStatusCommand)
				assert.True(t, ok)
				assert.Equal(t, "In Progress", statusCmd.Status)
			},
		},
		{
			name:       "change Bug status - valid",
			entityType: "Bug",
			tags: []tag.ParsedTag{
				{Key: "status", Value: "Fixed"},
			},
			wantCommands: 1,
			wantErrors:   0,
			checkCommand: func(t *testing.T, cmd tag.Command) {
				statusCmd, ok := cmd.(tag.ChangeStatusCommand)
				assert.True(t, ok)
				assert.Equal(t, "Fixed", statusCmd.Status)
			},
		},
		{
			name:       "change status - invalid (lowercase)",
			entityType: "Task",
			tags: []tag.ParsedTag{
				{Key: "status", Value: "done"},
			},
			wantCommands: 0,
			wantErrors:   1,
		},
		{
			name:       "change status - wrong entity type",
			entityType: "Task",
			tags: []tag.ParsedTag{
				{Key: "status", Value: "Fixed"}, // Bug status
			},
			wantCommands: 0,
			wantErrors:   1,
		},
		{
			name:       "change status - no active entity",
			entityType: "",
			tags: []tag.ParsedTag{
				{Key: "status", Value: "Done"},
			},
			wantCommands: 0,
			wantErrors:   1,
		},

		// #assignee tests
		{
			name:       "assign user - valid",
			entityType: "Task",
			tags: []tag.ParsedTag{
				{Key: "assignee", Value: "@alex"},
			},
			wantCommands: 1,
			wantErrors:   0,
			checkCommand: func(t *testing.T, cmd tag.Command) {
				assignCmd, ok := cmd.(tag.AssignUserCommand)
				assert.True(t, ok)
				assert.Equal(t, "@alex", assignCmd.Username)
				assert.Nil(t, assignCmd.UserID)
			},
		},
		{
			name:       "assign user - remove (@none)",
			entityType: "Task",
			tags: []tag.ParsedTag{
				{Key: "assignee", Value: "@none"},
			},
			wantCommands: 1,
			wantErrors:   0,
			checkCommand: func(t *testing.T, cmd tag.Command) {
				assignCmd, ok := cmd.(tag.AssignUserCommand)
				assert.True(t, ok)
				assert.Equal(t, "@none", assignCmd.Username)
			},
		},
		{
			name:       "assign user - remove (empty)",
			entityType: "Task",
			tags: []tag.ParsedTag{
				{Key: "assignee", Value: ""},
			},
			wantCommands: 1,
			wantErrors:   0,
		},
		{
			name:       "assign user - invalid format",
			entityType: "Task",
			tags: []tag.ParsedTag{
				{Key: "assignee", Value: "alex"}, // missing @
			},
			wantCommands: 0,
			wantErrors:   1,
		},

		// #priority tests
		{
			name:       "change priority - High",
			entityType: "Task",
			tags: []tag.ParsedTag{
				{Key: "priority", Value: "High"},
			},
			wantCommands: 1,
			wantErrors:   0,
			checkCommand: func(t *testing.T, cmd tag.Command) {
				priorityCmd, ok := cmd.(tag.ChangePriorityCommand)
				assert.True(t, ok)
				assert.Equal(t, "High", priorityCmd.Priority)
			},
		},
		{
			name:       "change priority - invalid (lowercase)",
			entityType: "Task",
			tags: []tag.ParsedTag{
				{Key: "priority", Value: "high"},
			},
			wantCommands: 0,
			wantErrors:   1,
		},

		// #due tests
		{
			name:       "set due date - valid",
			entityType: "Task",
			tags: []tag.ParsedTag{
				{Key: "due", Value: "2025-10-20"},
			},
			wantCommands: 1,
			wantErrors:   0,
			checkCommand: func(t *testing.T, cmd tag.Command) {
				dueCmd, ok := cmd.(tag.SetDueDateCommand)
				assert.True(t, ok)
				assert.NotNil(t, dueCmd.DueDate)
			},
		},
		{
			name:       "set due date - remove",
			entityType: "Task",
			tags: []tag.ParsedTag{
				{Key: "due", Value: ""},
			},
			wantCommands: 1,
			wantErrors:   0,
			checkCommand: func(t *testing.T, cmd tag.Command) {
				dueCmd, ok := cmd.(tag.SetDueDateCommand)
				assert.True(t, ok)
				assert.Nil(t, dueCmd.DueDate)
			},
		},
		{
			name:       "set due date - invalid format",
			entityType: "Task",
			tags: []tag.ParsedTag{
				{Key: "due", Value: "20-10-2025"},
			},
			wantCommands: 0,
			wantErrors:   1,
		},

		// #title tests
		{
			name:       "change title - valid",
			entityType: "Task",
			tags: []tag.ParsedTag{
				{Key: "title", Value: "New title"},
			},
			wantCommands: 1,
			wantErrors:   0,
			checkCommand: func(t *testing.T, cmd tag.Command) {
				titleCmd, ok := cmd.(tag.ChangeTitleCommand)
				assert.True(t, ok)
				assert.Equal(t, "New title", titleCmd.Title)
			},
		},
		{
			name:       "change title - empty",
			entityType: "Task",
			tags: []tag.ParsedTag{
				{Key: "title", Value: ""},
			},
			wantCommands: 0,
			wantErrors:   1,
		},

		// #severity tests
		{
			name:       "set severity - valid",
			entityType: "Bug",
			tags: []tag.ParsedTag{
				{Key: "severity", Value: "Critical"},
			},
			wantCommands: 1,
			wantErrors:   0,
			checkCommand: func(t *testing.T, cmd tag.Command) {
				severityCmd, ok := cmd.(tag.SetSeverityCommand)
				assert.True(t, ok)
				assert.Equal(t, "Critical", severityCmd.Severity)
			},
		},
		{
			name:       "set severity - invalid (lowercase)",
			entityType: "Bug",
			tags: []tag.ParsedTag{
				{Key: "severity", Value: "critical"},
			},
			wantCommands: 0,
			wantErrors:   1,
		},

		// Combined tests
		{
			name:       "create task and change status",
			entityType: "",
			tags: []tag.ParsedTag{
				{Key: "task", Value: "New task"},
				{Key: "status", Value: "In Progress"},
			},
			wantCommands: 2,
			wantErrors:   0,
		},
		{
			name:       "create bug and set severity and priority",
			entityType: "",
			tags: []tag.ParsedTag{
				{Key: "bug", Value: "New bug"},
				{Key: "severity", Value: "Major"},
				{Key: "priority", Value: "High"},
			},
			wantCommands: 3,
			wantErrors:   0,
		},
		{
			name:       "multiple management tags",
			entityType: "Task",
			tags: []tag.ParsedTag{
				{Key: "status", Value: "Done"},
				{Key: "assignee", Value: "@bob"},
				{Key: "priority", Value: "Low"},
			},
			wantCommands: 3,
			wantErrors:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.ProcessTags(chatID, tt.tags, tt.entityType)

			assert.Len(t, result.AppliedTags, tt.wantCommands, "unexpected number of commands")
			assert.Len(t, result.Errors, tt.wantErrors, "unexpected number of errors")

			if tt.checkCommand != nil && len(result.AppliedTags) > 0 {
				tt.checkCommand(t, result.AppliedTags[0].Command)
			}
		})
	}
}
