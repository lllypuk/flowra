package tag_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lllypuk/flowra/internal/domain/tag"
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
		// bazovye primery
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
			input: "#task Implement authorization function #priority High",
			wantTags: []tag.ParsedTag{
				{Key: "task", Value: "Implement authorization function"},
				{Key: "priority", Value: "High"},
			},
			wantText: "",
		},
		{
			name:  "text then tags on separate line",
			input: "Finished work\n#status Done",
			wantTags: []tag.ParsedTag{
				{Key: "status", Value: "Done"},
			},
			wantText: "Finished work",
		},
		{
			name:  "tags at start then text on New line",
			input: "#status Done #assignee @alex\nFinished work",
			wantTags: []tag.ParsedTag{
				{Key: "status", Value: "Done"},
				{Key: "assignee", Value: "@alex"},
			},
			wantText: "Finished work",
		},
		{
			name:  "tag from bug example",
			input: "#bug Login error\nReproduced on Chrome",
			wantTags: []tag.ParsedTag{
				{Key: "bug", Value: "Login error"},
			},
			wantText: "Reproduced on Chrome",
		},

		// Edge cases
		{
			name:     "tags in middle of line - ignored",
			input:    "Finished work #status Done sending",
			wantTags: []tag.ParsedTag{},
			wantText: "Finished work #status Done sending",
		},
		{
			name:  "mixed tags and text on same line",
			input: "#status Done some text #assignee @alex",
			wantTags: []tag.ParsedTag{
				{Key: "status", Value: "Done some text"},
				{Key: "assignee", Value: "@alex"},
			},
			wantText: "",
		},
		{
			name:     "unknown tag - ignored",
			input:    "Support #hashtags in text",
			wantTags: []tag.ParsedTag{},
			wantText: "Support #hashtags in text",
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
			input: "#task Fix bug in authorization module 🐛",
			wantTags: []tag.ParsedTag{
				{Key: "task", Value: "Fix bug in authorization module 🐛"},
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
			input: "First paragraph\nSecond paragraph\n#status Done\n#priority High",
			wantTags: []tag.ParsedTag{
				{Key: "status", Value: "Done"},
				{Key: "priority", Value: "High"},
			},
			wantText: "First paragraph\nSecond paragraph",
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
