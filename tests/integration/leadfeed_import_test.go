//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
	mongoinfra "github.com/lllypuk/flowra/internal/infrastructure/mongodb"
	mongorepo "github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	"github.com/lllypuk/flowra/tests/fixtures"
	"github.com/lllypuk/flowra/tests/testutil"
)

const leadFeedFixture = "../fixtures/leadfeed/full.jsonl"

type leadFeedEnv struct {
	db       *mongo.Database
	repo     *mongorepo.MongoLeadFeedRepository
	importer *leadfeed.Importer
}

func setupLeadFeedEnv(t *testing.T, opts ...leadfeed.ImporterOption) leadFeedEnv {
	t.Helper()
	client, db := testutil.SetupTestMongoDBWithClient(t)
	repo := mongorepo.NewMongoLeadFeedRepository(client, db)
	return leadFeedEnv{db: db, repo: repo, importer: leadfeed.NewImporter(repo, opts...)}
}

func (e leadFeedEnv) importJSONL(t *testing.T, asOfSeconds int, tasks []leadfeed.Task, records []leadfeed.Record) (leadfeed.ImportResult, error) {
	t.Helper()
	raw, err := fixtures.LeadFeedSnapshotJSONL(fixtures.LeadFeedBase.Add(time.Duration(asOfSeconds)*time.Second), tasks, records)
	require.NoError(t, err)
	return e.importer.Import(context.Background(), bytes.NewReader(raw))
}

type leadFeedContent struct {
	state   *leadfeed.State
	attempt *leadfeed.Attempt
	tasks   map[string]leadfeed.Task
	records map[string][]leadfeed.Record
}

func (e leadFeedEnv) content(t *testing.T) leadFeedContent {
	t.Helper()
	var out leadFeedContent
	err := e.repo.Read(context.Background(), func(ctx context.Context, view leadfeed.View) error {
		out = leadFeedContent{tasks: map[string]leadfeed.Task{}, records: map[string][]leadfeed.Record{}}
		var err error
		if out.state, err = view.State(ctx); err != nil {
			return err
		}
		if out.attempt, err = view.LastAttempt(ctx); err != nil {
			return err
		}
		tasks, err := view.Tasks(ctx, leadfeed.TaskFilter{}, 0, 1000)
		if err != nil {
			return err
		}
		for _, task := range tasks {
			out.tasks[task.ID] = task
			if out.records[task.ID], err = view.Records(ctx, task.ID, 0, 1000); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)
	return out
}

func TestLeadFeedImport_FixtureIntoEmptyDatabase(t *testing.T) {
	env := setupLeadFeedEnv(t)
	raw, err := os.ReadFile(leadFeedFixture)
	require.NoError(t, err)

	result, err := env.importer.Import(context.Background(), bytes.NewReader(raw))
	require.NoError(t, err)
	assert.Equal(t, leadfeed.OutcomePublished, result.Outcome)

	got := env.content(t)
	require.NotNil(t, got.state)
	assert.Equal(t, "389227faca902622", got.state.Revision)
	assert.Len(t, got.tasks, 4)
	assert.Equal(t, "paused", got.tasks["T-0004"].Status, "unknown status is stored verbatim")
	assert.Nil(t, got.tasks["T-0001"].WaitingOn)

	thread := got.records["T-0002"]
	require.Len(t, thread, 3)
	assert.Equal(t, []string{"j:3", "j:4", "u:toolu_beta1"}, []string{thread[0].ID, thread[1].ID, thread[2].ID})
	assert.Nil(t, thread[0].Body)
	require.NotNil(t, thread[0].BodyMissing)

	t.Run("text that looks like tag commands changes no chat entity", func(t *testing.T) {
		require.NotNil(t, thread[1].Body)
		assert.Contains(t, *thread[1].Body, "#status Done\n#assignee @bob")

		for _, coll := range []string{
			mongoinfra.CollectionEvents, mongoinfra.CollectionChatReadModel, mongoinfra.CollectionTaskReadModel,
			mongoinfra.CollectionMessages, mongoinfra.CollectionOutbox, mongoinfra.CollectionNotifications,
		} {
			count, countErr := env.db.Collection(coll).CountDocuments(context.Background(), bson.M{})
			require.NoError(t, countErr)
			assert.Zero(t, count, "collection %s", coll)
		}
	})

	t.Run("the same snapshot again only confirms it", func(t *testing.T) {
		again, againErr := env.importer.Import(context.Background(), bytes.NewReader(raw))
		require.NoError(t, againErr)
		assert.Equal(t, leadfeed.OutcomeRefreshed, again.Outcome)
		assert.Equal(t, got.tasks, env.content(t).tasks)
	})
}

func TestLeadFeedImport_RejectedInputKeepsPublishedFeed(t *testing.T) {
	env := setupLeadFeedEnv(t, leadfeed.WithMaxSnapshotBytes(64<<10))
	tasks := []leadfeed.Task{fixtures.LeadFeedTask("T-0001", "work", "agent", 0)}
	records := []leadfeed.Record{fixtures.LeadFeedLetter("j:1", "T-0001", "first letter", 0)}
	_, err := env.importJSONL(t, 10, tasks, records)
	require.NoError(t, err)
	before := env.content(t)

	valid, err := fixtures.LeadFeedSnapshotJSONL(fixtures.LeadFeedBase.Add(20*time.Second),
		[]leadfeed.Task{fixtures.LeadFeedTask("T-0002", "work", "agent", 1)}, nil)
	require.NoError(t, err)
	lines := bytes.Split(bytes.TrimRight(valid, "\n"), []byte("\n"))

	huge := fixtures.LeadFeedLetter("j:9", "T-0001", string(bytes.Repeat([]byte("x"), 128<<10)), 1)
	oversized, err := fixtures.LeadFeedSnapshotJSONL(fixtures.LeadFeedBase.Add(20*time.Second), tasks, []leadfeed.Record{huge})
	require.NoError(t, err)

	tests := []struct {
		name    string
		input   []byte
		wantErr error
	}{
		{name: "truncated", input: bytes.Join(lines[:len(lines)-1], []byte("\n")), wantErr: leadfeed.ErrSnapshotTruncated},
		{name: "marked incomplete", input: bytes.Replace(valid, []byte(`"complete":true`), []byte(`"complete":false`), 1),
			wantErr: leadfeed.ErrSnapshotIncomplete},
		{name: "unknown schema", input: bytes.Replace(valid, []byte(`"schema":1`), []byte(`"schema":3`), 1),
			wantErr: leadfeed.ErrSnapshotSchema},
		{name: "content edited under the same revision", input: bytes.Replace(valid, []byte("T-0002"), []byte("T-0003"), 1),
			wantErr: leadfeed.ErrSnapshotRevision},
		{name: "over the size limit", input: oversized, wantErr: leadfeed.ErrSnapshotTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, importErr := env.importer.Import(context.Background(), bytes.NewReader(tt.input))
			require.ErrorIs(t, importErr, tt.wantErr)

			after := env.content(t)
			assert.Equal(t, before.state, after.state)
			assert.Equal(t, before.tasks, after.tasks)
			assert.Equal(t, before.records, after.records)
			require.NotNil(t, after.attempt)
			assert.Equal(t, leadfeed.AttemptRejected, after.attempt.Outcome)
			assert.NotEmpty(t, after.attempt.Error)
		})
	}
}

func TestLeadFeedImport_LaterSnapshotRewritesHistory(t *testing.T) {
	env := setupLeadFeedEnv(t)

	// First export: the agent's transcript was not found, so T-0002 looks queued and its letter has no text.
	queued := fixtures.LeadFeedTask("T-0002", "queued", "agent", 1)
	missing := "letter not found in the agent transcript"
	letter := fixtures.LeadFeedLetter("j:2", "T-0002", "", 0)
	letter.Body, letter.BodyMissing = nil, &missing
	_, err := env.importJSONL(t, 10,
		[]leadfeed.Task{fixtures.LeadFeedTask("T-0001", "done", "", 0), queued},
		[]leadfeed.Record{fixtures.LeadFeedLetter("j:1", "T-0001", "old task", 0), letter})
	require.NoError(t, err)

	// Second export: the transcript turned up. Past status changes, a late record appears, T-0001 is gone.
	review := fixtures.LeadFeedTask("T-0002", "review", "lead", 1)
	cause := "u:toolu_late"
	review.Transitions = []leadfeed.Transition{
		{At: review.CreatedAt, Status: "queued"},
		{At: review.CreatedAt.Add(time.Second), Status: "work", Derived: true},
		{At: review.CreatedAt.Add(time.Minute), Status: "review", Cause: &cause, Derived: true},
	}
	found := fixtures.LeadFeedLetter("j:2", "T-0002", "the delivered letter", 0)
	late := fixtures.LeadFeedLetter("u:toolu_late", "T-0002", "[T-0002: done] report found later", 1)
	late.Kind, late.Actor, late.ActorDerived = "report", "agent", true
	late.At = fixtures.LeadFeedBase.Add(5 * time.Second) // earlier than the first snapshot's as_of

	result, err := env.importJSONL(t, 20, []leadfeed.Task{review}, []leadfeed.Record{found, late})
	require.NoError(t, err)
	assert.Equal(t, leadfeed.OutcomePublished, result.Outcome)

	got := env.content(t)
	assert.NotContains(t, got.tasks, "T-0001", "a task absent from a complete snapshot disappears")
	task := got.tasks["T-0002"]
	assert.Equal(t, "review", task.Status)
	require.NotNil(t, task.WaitingOn)
	assert.Equal(t, "lead", *task.WaitingOn)
	require.Len(t, task.Transitions, 3)
	assert.True(t, task.Transitions[1].Derived)

	thread := got.records["T-0002"]
	require.Len(t, thread, 2)
	require.NotNil(t, thread[0].Body)
	assert.Equal(t, "the delivered letter", *thread[0].Body)
	assert.Nil(t, thread[0].BodyMissing)
	assert.Equal(t, "u:toolu_late", thread[1].ID)

	t.Run("an older snapshot arriving afterwards is refused", func(t *testing.T) {
		_, staleErr := env.importJSONL(t, 15, []leadfeed.Task{fixtures.LeadFeedTask("T-0009", "work", "agent", 2)}, nil)
		require.ErrorIs(t, staleErr, leadfeed.ErrSnapshotStale)

		after := env.content(t)
		assert.Equal(t, got.state, after.state)
		assert.Equal(t, got.tasks, after.tasks)
		assert.Equal(t, leadfeed.AttemptRejected, after.attempt.Outcome)
	})
}
