package leadfeed_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
)

const fixtureDir = "../../../tests/fixtures/leadfeed"

func fixtureLines(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(fixtureDir, "full.jsonl"))
	require.NoError(t, err)
	return strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
}

func parse(lines []string) (*leadfeed.Snapshot, error) {
	return leadfeed.ParseSnapshot(strings.NewReader(strings.Join(lines, "\n") + "\n"))
}

// edit rewrites one JSON line through a generic map.
func edit(t *testing.T, line string, change func(map[string]any)) string {
	t.Helper()
	var obj map[string]any
	require.NoError(t, json.Unmarshal([]byte(line), &obj))
	change(obj)
	out, err := json.Marshal(obj)
	require.NoError(t, err)
	return string(out)
}

// reseal recomputes revision and counts so that only the edited content is under test.
func reseal(t *testing.T, lines []string) []string {
	t.Helper()
	body := lines[1 : len(lines)-1]
	sum := sha256.Sum256([]byte(strings.Join(body, "\n")))
	revision := hex.EncodeToString(sum[:])[:16]

	tasks := 0
	for _, line := range body {
		if strings.Contains(line, `"type":"task"`) || strings.Contains(line, `"type": "task"`) {
			tasks++
		}
	}

	out := append([]string{}, lines...)
	out[0] = edit(t, lines[0], func(h map[string]any) { h["revision"] = revision })
	out[len(out)-1] = edit(t, lines[len(lines)-1], func(f map[string]any) {
		f["revision"] = revision
		f["tasks"] = tasks
		f["records"] = len(body) - tasks
	})
	return out
}

func lineIndex(t *testing.T, lines []string, needle string) int {
	t.Helper()
	for i, line := range lines {
		if strings.Contains(line, needle) {
			return i
		}
	}
	t.Fatalf("no line contains %q", needle)
	return -1
}

func TestParseSnapshot_Full(t *testing.T) {
	snap, err := parse(fixtureLines(t))
	require.NoError(t, err)

	assert.Equal(t, "lead-task", snap.Header.Source)
	assert.Equal(t, "feedfacecafe", snap.Header.Epoch)
	assert.Len(t, snap.Tasks, 4)
	assert.Len(t, snap.Records, 9)

	byID := map[string]leadfeed.Record{}
	for _, rec := range snap.Records {
		byID[rec.ID] = rec
	}

	t.Run("closed task has no waiting side", func(t *testing.T) {
		assert.Equal(t, "done", snap.Tasks[0].Status)
		assert.Nil(t, snap.Tasks[0].WaitingOn)
		assert.Equal(t, int64(1200), snap.Tasks[0].Spent.Work)
	})

	t.Run("unknown status is kept verbatim", func(t *testing.T) {
		assert.Equal(t, "paused", snap.Tasks[3].Status)
		assert.Nil(t, snap.Tasks[3].Transitions[0].Cause)
	})

	t.Run("derived transition keeps its mark and cause", func(t *testing.T) {
		tr := snap.Tasks[0].Transitions[1]
		assert.True(t, tr.Derived)
		require.NotNil(t, tr.Cause)
		assert.Equal(t, "j:1", *tr.Cause)
	})

	t.Run("missing body is a reason, not an empty text", func(t *testing.T) {
		rec := byID["j:3"]
		assert.Nil(t, rec.Body)
		require.NotNil(t, rec.BodyMissing)
		assert.NotEmpty(t, *rec.BodyMissing)
		assert.True(t, rec.BodyDerived)
	})

	t.Run("tag-like text is stored as written", func(t *testing.T) {
		rec := byID["j:4"]
		require.NotNil(t, rec.Body)
		assert.True(t, strings.HasPrefix(*rec.Body, "#status Done\n#assignee @bob\n"))
		assert.Contains(t, *rec.Body, "<script>")
	})

	t.Run("records sharing a timestamp keep source order", func(t *testing.T) {
		assert.True(t, byID["j:3"].At.Equal(byID["j:4"].At))
		assert.Less(t, byID["j:3"].Order, byID["j:4"].Order)
	})

	t.Run("observation is attributed as derived", func(t *testing.T) {
		rec := byID["t:5f0c2a9e-0000-4000-8000-000000000001"]
		assert.Equal(t, "observation", rec.Kind)
		assert.True(t, rec.ActorDerived)
	})
}

func TestParseSnapshot_Rejects(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(t *testing.T, lines []string) []string
		wantErr error
	}{
		{
			name:    "empty input",
			mutate:  func(_ *testing.T, _ []string) []string { return nil },
			wantErr: leadfeed.ErrSnapshotTruncated,
		},
		{
			name:    "no footer",
			mutate:  func(_ *testing.T, l []string) []string { return l[:len(l)-1] },
			wantErr: leadfeed.ErrSnapshotTruncated,
		},
		{
			name: "cut in the middle of a line",
			mutate: func(_ *testing.T, l []string) []string {
				return append(append([]string{}, l[:3]...), l[3][:len(l[3])/2])
			},
			wantErr: leadfeed.ErrSnapshotMalformed,
		},
		{
			name: "complete false with a valid footer",
			mutate: func(t *testing.T, l []string) []string {
				l[0] = edit(t, l[0], func(h map[string]any) { h["complete"] = false })
				return l
			},
			wantErr: leadfeed.ErrSnapshotIncomplete,
		},
		{
			name: "source errors with complete true",
			mutate: func(t *testing.T, l []string) []string {
				l[0] = edit(t, l[0], func(h map[string]any) { h["errors"] = []string{"transcript unreadable"} })
				return l
			},
			wantErr: leadfeed.ErrSnapshotIncomplete,
		},
		{
			name: "unknown schema",
			mutate: func(t *testing.T, l []string) []string {
				l[0] = edit(t, l[0], func(h map[string]any) { h["schema"] = 3 })
				return l
			},
			wantErr: leadfeed.ErrSnapshotSchema,
		},
		{
			name: "header without epoch",
			mutate: func(t *testing.T, l []string) []string {
				l[0] = edit(t, l[0], func(h map[string]any) { h["epoch"] = "" })
				return l
			},
			wantErr: leadfeed.ErrSnapshotMalformed,
		},
		{
			name:    "first line is not a header",
			mutate:  func(_ *testing.T, l []string) []string { return l[1:] },
			wantErr: leadfeed.ErrSnapshotMalformed,
		},
		{
			name:    "line after the footer",
			mutate:  func(_ *testing.T, l []string) []string { return append(l, l[1]) },
			wantErr: leadfeed.ErrSnapshotMalformed,
		},
		{
			name: "unknown line type",
			mutate: func(_ *testing.T, l []string) []string {
				return append(append(append([]string{}, l[:2]...), `{"type":"note"}`), l[2:]...)
			},
			wantErr: leadfeed.ErrSnapshotMalformed,
		},
		{
			name: "footer counts differ from the lines read",
			mutate: func(_ *testing.T, l []string) []string {
				return append(append([]string{}, l[:len(l)-2]...), l[len(l)-1])
			},
			wantErr: leadfeed.ErrSnapshotMalformed,
		},
		{
			name: "footer names another revision",
			mutate: func(t *testing.T, l []string) []string {
				l[len(l)-1] = edit(t, l[len(l)-1], func(f map[string]any) { f["revision"] = "0000000000000000" })
				return l
			},
			wantErr: leadfeed.ErrSnapshotMalformed,
		},
		{
			name: "content changed under the same revision",
			mutate: func(t *testing.T, l []string) []string {
				i := lineIndex(t, l, `"id": "j:6"`)
				l[i] = strings.Replace(l[i], "rollout", "rollback", 1)
				return l
			},
			wantErr: leadfeed.ErrSnapshotRevision,
		},
		{
			name: "duplicate record id",
			mutate: func(t *testing.T, l []string) []string {
				i := lineIndex(t, l, `"id": "j:2"`)
				l[i] = edit(t, l[i], func(r map[string]any) { r["id"] = "j:1" })
				return reseal(t, l)
			},
			wantErr: leadfeed.ErrSnapshotMalformed,
		},
		{
			name: "duplicate task id",
			mutate: func(t *testing.T, l []string) []string {
				l[2] = edit(t, l[2], func(task map[string]any) { task["id"] = "T-0001" })
				return reseal(t, l)
			},
			wantErr: leadfeed.ErrSnapshotMalformed,
		},
		{
			name: "record of an unknown task",
			mutate: func(t *testing.T, l []string) []string {
				i := lineIndex(t, l, `"id": "j:5"`)
				l[i] = edit(t, l[i], func(r map[string]any) { r["task"] = "T-0999" })
				return reseal(t, l)
			},
			wantErr: leadfeed.ErrSnapshotMalformed,
		},
		{
			name: "transition caused by an unknown record",
			mutate: func(t *testing.T, l []string) []string {
				l[4] = edit(t, l[4], func(task map[string]any) {
					task["transitions"] = []map[string]any{
						{"at": "2026-01-10T11:30:00.000Z", "status": "paused", "cause": "j:999", "derived": false},
					}
				})
				return reseal(t, l)
			},
			wantErr: leadfeed.ErrSnapshotMalformed,
		},
		{
			name: "record with both body and missing reason",
			mutate: func(t *testing.T, l []string) []string {
				i := lineIndex(t, l, `"id": "j:1"`)
				l[i] = edit(t, l[i], func(r map[string]any) { r["body_missing"] = "lost" })
				return reseal(t, l)
			},
			wantErr: leadfeed.ErrSnapshotMalformed,
		},
		{
			name: "record with neither body nor missing reason",
			mutate: func(t *testing.T, l []string) []string {
				i := lineIndex(t, l, `"id": "j:2"`)
				l[i] = edit(t, l[i], func(r map[string]any) { r["body_missing"] = nil })
				return reseal(t, l)
			},
			wantErr: leadfeed.ErrSnapshotMalformed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snap, err := parse(tt.mutate(t, fixtureLines(t)))
			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, snap)
		})
	}
}

func TestParseSnapshot_ResealedEditIsAccepted(t *testing.T) {
	lines := fixtureLines(t)
	i := lineIndex(t, lines, `"id": "j:6"`)
	lines[i] = strings.Replace(lines[i], "rollout", "rollback", 1)

	snap, err := parse(reseal(t, lines))
	require.NoError(t, err)
	assert.NotEqual(t, "389227faca902622", snap.Header.Revision)
}

// TestParseSnapshot_LeadSamples checks real exports, which stay outside the repository.
func TestParseSnapshot_LeadSamples(t *testing.T) {
	dir := os.Getenv("LEADFEED_SAMPLE_DIR")
	if dir == "" {
		t.Skip("LEADFEED_SAMPLE_DIR is not set")
	}

	paths, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	require.NoError(t, err)
	require.NotEmpty(t, paths)

	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			f, openErr := os.Open(path)
			require.NoError(t, openErr)
			defer f.Close()

			snap, parseErr := leadfeed.ParseSnapshot(f)
			require.NoError(t, parseErr)
			assert.NotEmpty(t, snap.Tasks)
		})
	}
}
