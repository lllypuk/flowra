package leadfeed_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
)

func questionLines(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(fixtureDir, "questions.jsonl"))
	require.NoError(t, err)
	return strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
}

// editQuestion rewrites the question object of the record with the given id and reseals the snapshot.
func editQuestion(t *testing.T, lines []string, id string, change func(rec, q map[string]any)) []string {
	t.Helper()
	i := lineIndex(t, lines, `"id": "`+id+`"`)
	lines[i] = edit(t, lines[i], func(rec map[string]any) {
		q, _ := rec["question"].(map[string]any)
		change(rec, q)
	})
	return reseal(t, lines)
}

func TestParseSnapshot_Questions(t *testing.T) {
	snap, err := parse(questionLines(t))
	require.NoError(t, err)
	assert.Equal(t, leadfeed.SchemaQuestions, snap.Header.Schema)

	byQID := map[string]*leadfeed.Question{}
	for i := range snap.Records {
		if q := snap.Records[i].Question; q != nil {
			assert.Equal(t, leadfeed.KindQuestion, snap.Records[i].Kind)
			byQID[q.QID] = q
		}
	}
	require.Len(t, byQID, 7)

	t.Run("waiting", func(t *testing.T) {
		q := byQID["Q-0001"]
		assert.Nil(t, q.Answer)
		assert.Nil(t, q.Resolved)
		require.Len(t, q.Options, 2)
		require.NotNil(t, q.Recommendation)
		assert.Equal(t, "a", *q.Recommendation.OptionID)
		assert.Len(t, q.FP, 64)
	})

	t.Run("answered and not yet delivered", func(t *testing.T) {
		a := byQID["Q-0002"].Answer
		require.NotNil(t, a)
		assert.False(t, a.Delivered)
		assert.Contains(t, a.Text, "<b>please</b>")
		assert.Equal(t, "b", *a.OptionID)
	})

	t.Run("answered and delivered", func(t *testing.T) {
		assert.True(t, byQID["Q-0003"].Answer.Delivered)
	})

	t.Run("no options and no recommendation", func(t *testing.T) {
		q := byQID["Q-0004"]
		assert.Empty(t, q.Options)
		assert.Nil(t, q.Recommendation)
	})

	t.Run("closed with and without an answer", func(t *testing.T) {
		assert.NotNil(t, byQID["Q-0005"].Answer)
		assert.NotNil(t, byQID["Q-0005"].Resolved)
		assert.Nil(t, byQID["Q-0006"].Answer)
		assert.False(t, byQID["Q-0006"].Resolved.Elsewhere)
	})

	t.Run("late answer on a question closed elsewhere", func(t *testing.T) {
		q := byQID["Q-0007"]
		assert.True(t, q.Answer.Late)
		assert.Nil(t, q.Answer.OptionID)
		assert.True(t, q.Resolved.Elsewhere)
	})
}

func TestParseSnapshot_QuestionRejects(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, lines []string) []string
	}{
		{
			name: "question in schema 1",
			mutate: func(t *testing.T, l []string) []string {
				l[0] = edit(t, l[0], func(h map[string]any) { h["schema"] = 1 })
				return l
			},
		},
		{
			name: "question record without a question object",
			mutate: func(t *testing.T, l []string) []string {
				return editQuestion(t, l, "j:2", func(rec, _ map[string]any) { delete(rec, "question") })
			},
		},
		{
			name: "question object on a letter",
			mutate: func(t *testing.T, l []string) []string {
				l = editQuestion(t, l, "j:2", func(rec, _ map[string]any) { rec["kind"] = "letter" })
				return l
			},
		},
		{
			name: "empty body",
			mutate: func(t *testing.T, l []string) []string {
				return editQuestion(t, l, "j:2", func(rec, _ map[string]any) { rec["body"] = "" })
			},
		},
		{
			name: "missing body",
			mutate: func(t *testing.T, l []string) []string {
				return editQuestion(t, l, "j:2", func(rec, _ map[string]any) {
					rec["body"] = nil
					rec["body_missing"] = "lost"
				})
			},
		},
		{
			name: "fp too short",
			mutate: func(t *testing.T, l []string) []string {
				return editQuestion(t, l, "j:2", func(_, q map[string]any) { q["fp"] = "abcd" })
			},
		},
		{
			name: "fp not hex",
			mutate: func(t *testing.T, l []string) []string {
				return editQuestion(t, l, "j:2", func(_, q map[string]any) { q["fp"] = strings.Repeat("z", 64) })
			},
		},
		{
			name: "empty scope",
			mutate: func(t *testing.T, l []string) []string {
				return editQuestion(t, l, "j:2", func(_, q map[string]any) { q["scope"] = "" })
			},
		},
		{
			name: "empty qid",
			mutate: func(t *testing.T, l []string) []string {
				return editQuestion(t, l, "j:2", func(_, q map[string]any) { q["qid"] = "" })
			},
		},
		{
			name: "duplicate qid",
			mutate: func(t *testing.T, l []string) []string {
				return editQuestion(t, l, "j:3", func(_, q map[string]any) { q["qid"] = "Q-0001" })
			},
		},
		{
			name: "duplicate option id",
			mutate: func(t *testing.T, l []string) []string {
				return editQuestion(t, l, "j:2", func(_, q map[string]any) {
					q["options"] = []map[string]any{{"id": "a", "label": "A"}, {"id": "a", "label": "B"}}
				})
			},
		},
		{
			name: "recommendation names an unknown option",
			mutate: func(t *testing.T, l []string) []string {
				return editQuestion(t, l, "j:2", func(_, q map[string]any) {
					q["recommendation"] = map[string]any{"option_id": "zzz", "why": "x"}
				})
			},
		},
		{
			name: "recommendation names an option of an empty list",
			mutate: func(t *testing.T, l []string) []string {
				return editQuestion(t, l, "j:5", func(_, q map[string]any) {
					q["recommendation"] = map[string]any{"option_id": "a", "why": "x"}
				})
			},
		},
		{
			name: "answer names an unknown option",
			mutate: func(t *testing.T, l []string) []string {
				return editQuestion(t, l, "j:3", func(_, q map[string]any) {
					q["answer"].(map[string]any)["option_id"] = "zzz"
				})
			},
		},
		{
			name: "answer without answer_id",
			mutate: func(t *testing.T, l []string) []string {
				return editQuestion(t, l, "j:3", func(_, q map[string]any) {
					q["answer"].(map[string]any)["answer_id"] = ""
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snap, err := parse(tt.mutate(t, questionLines(t)))
			require.ErrorIs(t, err, leadfeed.ErrSnapshotMalformed)
			assert.Nil(t, snap)
		})
	}
}

// TestParseSnapshot_LeadSampleQuestions pins the questions of the lead's real sample, which stays outside the repository.
func TestParseSnapshot_LeadSampleQuestions(t *testing.T) {
	dir := os.Getenv("LEADFEED_SAMPLE_DIR")
	if dir == "" {
		t.Skip("LEADFEED_SAMPLE_DIR is not set")
	}

	f, err := os.Open(filepath.Join(dir, "snapshot-3-questions.jsonl"))
	require.NoError(t, err)
	defer f.Close()

	snap, err := leadfeed.ParseSnapshot(f)
	require.NoError(t, err)
	assert.Equal(t, leadfeed.SchemaQuestions, snap.Header.Schema)

	questions := 0
	for i := range snap.Records {
		if q := snap.Records[i].Question; q != nil {
			questions++
			assert.NotEmpty(t, q.QID)
		}
	}
	assert.Equal(t, 2, questions)
}
