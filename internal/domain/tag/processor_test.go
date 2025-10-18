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
		wantCommands int
		wantErrors   int
		checkCommand func(t *testing.T, cmd tag.Command)
	}{
		{
			name: "create task",
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
			name: "create bug",
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
			name: "create epic",
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
			name: "task with leading/trailing spaces",
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
			name: "task with special characters",
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
			name: "empty task title - error",
			tags: []tag.ParsedTag{
				{Key: "task", Value: ""},
			},
			wantCommands: 0,
			wantErrors:   1,
		},
		{
			name: "whitespace only title - error",
			tags: []tag.ParsedTag{
				{Key: "bug", Value: "   "},
			},
			wantCommands: 0,
			wantErrors:   1,
		},
		{
			name: "multiple entity creation commands",
			tags: []tag.ParsedTag{
				{Key: "task", Value: "Task 1"},
				{Key: "bug", Value: "Bug 1"},
				{Key: "epic", Value: "Epic 1"},
			},
			wantCommands: 3,
			wantErrors:   0,
		},
		{
			name: "mix of valid and invalid",
			tags: []tag.ParsedTag{
				{Key: "task", Value: "Valid task"},
				{Key: "bug", Value: ""},
				{Key: "epic", Value: "Valid epic"},
			},
			wantCommands: 2,
			wantErrors:   1,
		},
		{
			name: "unknown tags ignored",
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
			commands, errors := processor.ProcessTags(chatID, tt.tags)

			assert.Len(t, commands, tt.wantCommands, "unexpected number of commands")
			assert.Len(t, errors, tt.wantErrors, "unexpected number of errors")

			if tt.checkCommand != nil && len(commands) > 0 {
				tt.checkCommand(t, commands[0])
			}

			// Check error messages
			if tt.wantErrors > 0 {
				for _, err := range errors {
					assert.Contains(t, err.Error(), "title is required")
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
