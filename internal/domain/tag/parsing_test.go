package tag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ====== Task 02: Tag Position Parsing Tests ======

func TestParse(t *testing.T) {
	parser := NewTagParser()

	tests := []struct {
		name     string
		input    string
		wantTags []ParsedTag
		wantText string
	}{
		// Базовые примеры
		{
			name:  "single tag",
			input: "#status Done",
			wantTags: []ParsedTag{
				{Key: "status", Value: "Done"},
			},
			wantText: "",
		},
		{
			name:  "multiple tags on one line",
			input: "#status Done #assignee @alex",
			wantTags: []ParsedTag{
				{Key: "status", Value: "Done"},
				{Key: "assignee", Value: "@alex"},
			},
			wantText: "",
		},
		{
			name:  "tag with multi-word value",
			input: "#task Реализовать функцию авторизации #priority High",
			wantTags: []ParsedTag{
				{Key: "task", Value: "Реализовать функцию авторизации"},
				{Key: "priority", Value: "High"},
			},
			wantText: "",
		},
		{
			name:  "text then tags on separate line",
			input: "Закончил работу\n#status Done",
			wantTags: []ParsedTag{
				{Key: "status", Value: "Done"},
			},
			wantText: "Закончил работу",
		},
		{
			name:  "tags at start then text on new line",
			input: "#status Done #assignee @alex\nЗакончил работу",
			wantTags: []ParsedTag{
				{Key: "status", Value: "Done"},
				{Key: "assignee", Value: "@alex"},
			},
			wantText: "Закончил работу",
		},
		{
			name:  "tag from bug example",
			input: "#bug Ошибка в логине\nВоспроизводится на Chrome",
			wantTags: []ParsedTag{
				{Key: "bug", Value: "Ошибка в логине"},
			},
			wantText: "Воспроизводится на Chrome",
		},

		// Edge cases
		{
			name:     "tags in middle of line - ignored",
			input:    "Закончил работу #status Done отправляю",
			wantTags: []ParsedTag{},
			wantText: "Закончил работу #status Done отправляю",
		},
		{
			name:  "mixed tags and text on same line",
			input: "#status Done какой-то текст #assignee @alex",
			wantTags: []ParsedTag{
				{Key: "status", Value: "Done какой-то текст"},
				{Key: "assignee", Value: "@alex"},
			},
			wantText: "",
		},
		{
			name:     "unknown tag - ignored",
			input:    "Поддержка #hashtags в тексте",
			wantTags: []ParsedTag{},
			wantText: "Поддержка #hashtags в тексте",
		},
		{
			name:  "empty lines ignored",
			input: "#status Done\n\n\n#priority High",
			wantTags: []ParsedTag{
				{Key: "status", Value: "Done"},
				{Key: "priority", Value: "High"},
			},
			wantText: "",
		},
		{
			name:  "tag without value",
			input: "#assignee",
			wantTags: []ParsedTag{
				{Key: "assignee", Value: ""},
			},
			wantText: "",
		},
		{
			name:  "unicode in tag value",
			input: "#task Исправить баг в модуле авторизации 🐛",
			wantTags: []ParsedTag{
				{Key: "task", Value: "Исправить баг в модуле авторизации 🐛"},
			},
			wantText: "",
		},
		{
			name:  "tag value with special characters",
			input: "#task Fix issue #123 (critical!)",
			wantTags: []ParsedTag{
				{Key: "task", Value: "Fix issue #123 (critical!)"},
			},
			wantText: "",
		},
		{
			name:  "multiple tags on separate lines",
			input: "#status Done\n#priority High\n#assignee @alex",
			wantTags: []ParsedTag{
				{Key: "status", Value: "Done"},
				{Key: "priority", Value: "High"},
				{Key: "assignee", Value: "@alex"},
			},
			wantText: "",
		},
		{
			name:  "text with multiple paragraphs and tags",
			input: "Первый параграф\nВторой параграф\n#status Done\n#priority High",
			wantTags: []ParsedTag{
				{Key: "status", Value: "Done"},
				{Key: "priority", Value: "High"},
			},
			wantText: "Первый параграф\nВторой параграф",
		},
		{
			name:     "only unknown tags",
			input:    "#unknown1 value1 #unknown2 value2",
			wantTags: []ParsedTag{},
			wantText: "",
		},
		{
			name:  "mix of known and unknown tags",
			input: "#status Done #unknown value #priority High",
			wantTags: []ParsedTag{
				{Key: "status", Value: "Done"},
				{Key: "priority", Value: "High"},
			},
			wantText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.input)
			assert.Equal(t, tt.wantTags, result.Tags, "tags mismatch")
			assert.Equal(t, tt.wantText, result.PlainText, "text mismatch")
		})
	}
}

func TestParseOneTag(t *testing.T) {
	parser := NewTagParser()

	tests := []struct {
		name          string
		input         string
		wantKey       string
		wantValue     string
		wantRemaining string
	}{
		{
			name:          "single word value",
			input:         "#status Done",
			wantKey:       "status",
			wantValue:     "Done",
			wantRemaining: "",
		},
		{
			name:          "multi-word value",
			input:         "#task Реализовать авторизацию",
			wantKey:       "task",
			wantValue:     "Реализовать авторизацию",
			wantRemaining: "",
		},
		{
			name:          "value with next tag",
			input:         "#status In Progress #assignee @alex",
			wantKey:       "status",
			wantValue:     "In Progress",
			wantRemaining: "#assignee @alex",
		},
		{
			name:          "tag without value",
			input:         "#assignee",
			wantKey:       "assignee",
			wantValue:     "",
			wantRemaining: "",
		},
		{
			name:          "tag without value with next tag",
			input:         "#assignee #priority High",
			wantKey:       "assignee",
			wantValue:     "",
			wantRemaining: "#priority High",
		},
		{
			name:          "value with special chars",
			input:         "#task Fix #123 and #456",
			wantKey:       "task",
			wantValue:     "Fix #123 and #456",
			wantRemaining: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag, remaining := parser.parseOneTag(tt.input)
			assert.NotNil(t, tag)
			assert.Equal(t, tt.wantKey, tag.Key)
			assert.Equal(t, tt.wantValue, tag.Value)
			assert.Equal(t, tt.wantRemaining, remaining)
		})
	}
}

func TestParseTagsFromLine(t *testing.T) {
	parser := NewTagParser()

	tests := []struct {
		name          string
		input         string
		wantTags      []ParsedTag
		wantRemaining string
	}{
		{
			name:  "single tag",
			input: "#status Done",
			wantTags: []ParsedTag{
				{Key: "status", Value: "Done"},
			},
			wantRemaining: "",
		},
		{
			name:  "multiple tags",
			input: "#status Done #priority High #assignee @alex",
			wantTags: []ParsedTag{
				{Key: "status", Value: "Done"},
				{Key: "priority", Value: "High"},
				{Key: "assignee", Value: "@alex"},
			},
			wantRemaining: "",
		},
		{
			name:  "tags with text after",
			input: "#status Done some text here",
			wantTags: []ParsedTag{
				{Key: "status", Value: "Done some text here"},
			},
			wantRemaining: "",
		},
		{
			name:  "unknown tags filtered out",
			input: "#status Done #unknown value #priority High",
			wantTags: []ParsedTag{
				{Key: "status", Value: "Done"},
				{Key: "priority", Value: "High"},
			},
			wantRemaining: "",
		},
		{
			name:          "only unknown tags",
			input:         "#unknown1 value1 #unknown2 value2",
			wantTags:      []ParsedTag{},
			wantRemaining: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags, remaining := parser.parseTagsFromLine(tt.input)
			assert.Equal(t, tt.wantTags, tags)
			assert.Equal(t, tt.wantRemaining, remaining)
		})
	}
}
