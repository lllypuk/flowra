package leadfeed_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
)

type fakePublisher struct {
	published  []*leadfeed.Snapshot
	attempts   []leadfeed.Attempt
	result     leadfeed.PublishResult
	publishErr error
	attemptErr error
}

func (f *fakePublisher) Publish(
	_ context.Context,
	snap *leadfeed.Snapshot,
	_ time.Time,
) (leadfeed.PublishResult, error) {
	if f.publishErr != nil {
		return leadfeed.PublishResult{}, f.publishErr
	}
	f.published = append(f.published, snap)
	return f.result, nil
}

func (f *fakePublisher) RecordAttempt(_ context.Context, attempt leadfeed.Attempt) error {
	f.attempts = append(f.attempts, attempt)
	return f.attemptErr
}

func importClock() time.Time { return time.Date(2026, 1, 10, 13, 0, 0, 0, time.UTC) }

func newTestImporter(store *fakePublisher, opts ...leadfeed.ImporterOption) *leadfeed.Importer {
	opts = append([]leadfeed.ImporterOption{leadfeed.WithClock(importClock)}, opts...)
	return leadfeed.NewImporter(store, opts...)
}

func fixtureText(t *testing.T) string {
	t.Helper()
	return strings.Join(fixtureLines(t), "\n") + "\n"
}

func TestImporter_PublishesValidSnapshot(t *testing.T) {
	store := &fakePublisher{result: leadfeed.PublishResult{Outcome: leadfeed.OutcomePublished, PreviousEpoch: "old"}}

	result, err := newTestImporter(store).Import(context.Background(), strings.NewReader(fixtureText(t)))

	require.NoError(t, err)
	assert.Equal(t, leadfeed.OutcomePublished, result.Outcome)
	assert.Equal(t, "389227faca902622", result.Revision)
	assert.Equal(t, 4, result.Tasks)
	assert.Equal(t, 9, result.Records)
	assert.Equal(t, "old", result.PreviousEpoch)

	require.Len(t, store.published, 1)
	require.Len(t, store.attempts, 1)
	assert.Equal(t, string(leadfeed.OutcomePublished), store.attempts[0].Outcome)
	assert.Empty(t, store.attempts[0].Error)
	assert.Equal(t, "old", store.attempts[0].PreviousEpoch)
	assert.True(t, store.attempts[0].At.Equal(importClock()))
}

func TestImporter_RejectsBeforePublishing(t *testing.T) {
	tests := []struct {
		name    string
		input   func(t *testing.T) string
		opts    []leadfeed.ImporterOption
		wantErr error
	}{
		{
			name:    "truncated snapshot",
			input:   func(t *testing.T) string { return strings.Join(fixtureLines(t)[:5], "\n") },
			wantErr: leadfeed.ErrSnapshotTruncated,
		},
		{
			name: "incomplete snapshot",
			input: func(t *testing.T) string {
				lines := fixtureLines(t)
				lines[0] = edit(t, lines[0], func(h map[string]any) { h["complete"] = false })
				return strings.Join(lines, "\n")
			},
			wantErr: leadfeed.ErrSnapshotIncomplete,
		},
		{
			name:    "input over the size limit",
			input:   fixtureText,
			opts:    []leadfeed.ImporterOption{leadfeed.WithMaxSnapshotBytes(1024)},
			wantErr: leadfeed.ErrSnapshotTooLarge,
		},
		{
			name:    "empty input",
			input:   func(_ *testing.T) string { return "" },
			wantErr: leadfeed.ErrSnapshotTruncated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakePublisher{}

			_, err := newTestImporter(store, tt.opts...).Import(context.Background(), strings.NewReader(tt.input(t)))

			require.ErrorIs(t, err, tt.wantErr)
			assert.Empty(t, store.published, "nothing may reach the store")
			require.Len(t, store.attempts, 1)
			assert.Equal(t, leadfeed.AttemptRejected, store.attempts[0].Outcome)
			assert.NotEmpty(t, store.attempts[0].Error)
		})
	}
}

func TestImporter_InputExactlyAtLimitIsAccepted(t *testing.T) {
	text := fixtureText(t)
	store := &fakePublisher{result: leadfeed.PublishResult{Outcome: leadfeed.OutcomePublished}}

	_, err := newTestImporter(store, leadfeed.WithMaxSnapshotBytes(int64(len(text)))).
		Import(context.Background(), strings.NewReader(text))

	require.NoError(t, err)
}

func TestImporter_RecordsPublicationFailures(t *testing.T) {
	tests := []struct {
		name        string
		publishErr  error
		wantOutcome string
	}{
		{name: "stale snapshot", publishErr: leadfeed.ErrSnapshotStale, wantOutcome: leadfeed.AttemptRejected},
		{name: "conflicting snapshot", publishErr: leadfeed.ErrSnapshotConflict, wantOutcome: leadfeed.AttemptRejected},
		{name: "unknown commit", publishErr: leadfeed.ErrCommitUnknown, wantOutcome: leadfeed.AttemptUnknown},
		{name: "store failure", publishErr: errors.New("connection reset"), wantOutcome: leadfeed.AttemptFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakePublisher{publishErr: tt.publishErr}

			_, err := newTestImporter(store).Import(context.Background(), strings.NewReader(fixtureText(t)))

			require.ErrorIs(t, err, tt.publishErr)
			require.Len(t, store.attempts, 1)
			assert.Equal(t, tt.wantOutcome, store.attempts[0].Outcome)
			assert.Equal(t, "389227faca902622", store.attempts[0].Revision)
		})
	}
}

func TestImporter_AttemptWriteFailureDoesNotChangeTheOutcome(t *testing.T) {
	store := &fakePublisher{
		result:     leadfeed.PublishResult{Outcome: leadfeed.OutcomeRefreshed},
		attemptErr: errors.New("write concern timeout"),
	}

	result, err := newTestImporter(store).Import(context.Background(), strings.NewReader(fixtureText(t)))

	require.NoError(t, err)
	assert.Equal(t, leadfeed.OutcomeRefreshed, result.Outcome)
}

func TestImporter_RecordsAttemptAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	store := &fakePublisher{publishErr: context.Canceled}

	_, err := newTestImporter(store).Import(ctx, strings.NewReader(fixtureText(t)))

	require.ErrorIs(t, err, context.Canceled)
	require.Len(t, store.attempts, 1)
}

type failingReader struct{}

func (failingReader) Read(_ []byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestImporter_ReadFailure(t *testing.T) {
	store := &fakePublisher{}

	_, err := newTestImporter(store).Import(context.Background(), failingReader{})

	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.Empty(t, store.published)
	require.Len(t, store.attempts, 1)
	assert.Equal(t, leadfeed.AttemptRejected, store.attempts[0].Outcome)
}
