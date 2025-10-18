package tag_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lllypuk/teams-up/internal/domain/tag"
)

// ====== Task 02: Tag Position Parsing Tests ======

func TestParse(t *testing.T) {
	parser := tag.NewParser()

	tests := []struct {
		name     string
		input    string
		wantTags []tag.ParsedTag
		wantText string
	}{
		// Базовые примеры
		{
			name:  "single tag",
			input: "#status Done",
			wantTags: []tag.ParsedTag{
				{Key: "status", Value: "Done"},
			},
			wantText: "",
		},
		{
			name:  "multiple tags on one line",
			input: "#status Done #assignee @alex",
			wantTags: []tag.ParsedTag{
				{Key: "status", Value: "Done"},
				{Key: "assignee", Value: "@alex"},
			},
			wantText: "",
		},
		{
			name:  "tag with multi-word value",
			input: "#task Реализовать функцию авторизации #priority High",
			wantTags: []tag.ParsedTag{
				{Key: "task", Value: "Реализовать функцию авторизации"},
				{Key: "priority", Value: "High"},
			},
			wantText: "",
		},
		{
			name:  "text then tags on separate line",
			input: "Закончил работу\n#status Done",
			wantTags: []tag.ParsedTag{
				{Key: "status", Value: "Done"},
			},
			wantText: "Закончил работу",
		},
		{
			name:  "tags at start then text on new line",
			input: "#status Done #assignee @alex\nЗакончил работу",
			wantTags: []tag.ParsedTag{
				{Key: "status", Value: "Done"},
				{Key: "assignee", Value: "@alex"},
			},
			wantText: "Закончил работу",
		},
		{
			name:  "tag from bug example",
			input: "#bug Ошибка в логине\nВоспроизводится на Chrome",
			wantTags: []tag.ParsedTag{
				{Key: "bug", Value: "Ошибка в логине"},
			},
			wantText: "Воспроизводится на Chrome",
		},

		// Edge cases
		{
			name:     "tags in middle of line - ignored",
			input:    "Закончил работу #status Done отправляю",
			wantTags: []tag.ParsedTag{},
			wantText: "Закончил работу #status Done отправляю",
		},
		{
			name:  "mixed tags and text on same line",
			input: "#status Done какой-то текст #assignee @alex",
			wantTags: []tag.ParsedTag{
				{Key: "status", Value: "Done какой-то текст"},
				{Key: "assignee", Value: "@alex"},
			},
			wantText: "",
		},
		{
			name:     "unknown tag - ignored",
			input:    "Поддержка #hashtags в тексте",
			wantTags: []tag.ParsedTag{},
			wantText: "Поддержка #hashtags в тексте",
		},
		{
			name:  "empty lines ignored",
			input: "#status Done\n\n\n#priority High",
			wantTags: []tag.ParsedTag{
				{Key: "status", Value: "Done"},
				{Key: "priority", Value: "High"},
			},
			wantText: "",
		},
		{
			name:  "tag without value",
			input: "#assignee",
			wantTags: []tag.ParsedTag{
				{Key: "assignee", Value: ""},
			},
			wantText: "",
		},
		{
			name:  "unicode in tag value",
			input: "#task Исправить баг в модуле авторизации 🐛",
			wantTags: []tag.ParsedTag{
				{Key: "task", Value: "Исправить баг в модуле авторизации 🐛"},
			},
			wantText: "",
		},
		{
			name:  "tag value with special characters",
			input: "#task Fix issue #123 (critical!)",
			wantTags: []tag.ParsedTag{
				{Key: "task", Value: "Fix issue #123 (critical!)"},
			},
			wantText: "",
		},
		{
			name:  "multiple tags on separate lines",
			input: "#status Done\n#priority High\n#assignee @alex",
			wantTags: []tag.ParsedTag{
				{Key: "status", Value: "Done"},
				{Key: "priority", Value: "High"},
				{Key: "assignee", Value: "@alex"},
			},
			wantText: "",
		},
		{
			name:  "text with multiple paragraphs and tags",
			input: "Первый параграф\nВторой параграф\n#status Done\n#priority High",
			wantTags: []tag.ParsedTag{
				{Key: "status", Value: "Done"},
				{Key: "priority", Value: "High"},
			},
			wantText: "Первый параграф\nВторой параграф",
		},
		{
			name:     "only unknown tags",
			input:    "#unknown1 value1 #unknown2 value2",
			wantTags: []tag.ParsedTag{},
			wantText: "",
		},
		{
			name:  "mix of known and unknown tags",
			input: "#status Done #unknown value #priority High",
			wantTags: []tag.ParsedTag{
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
