package mongodb_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
	"github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	"github.com/lllypuk/flowra/tests/fixtures"
	"github.com/lllypuk/flowra/tests/testutil"
)

func feedBase() time.Time { return time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC) }

func setupLeadFeedRepository(t *testing.T) *mongodb.MongoLeadFeedRepository {
	t.Helper()
	client, db := testutil.SetupTestMongoDBWithClient(t)
	return mongodb.NewMongoLeadFeedRepository(client, db)
}

// feedSnapshot builds a snapshot whose every task title carries the revision, so a mixed read is visible.
func feedSnapshot(revision string, asOfSeconds int, taskIDs ...string) *leadfeed.Snapshot {
	snap := &leadfeed.Snapshot{Header: leadfeed.Header{
		Schema:   leadfeed.SchemaBase,
		Source:   "lead-task",
		Epoch:    "epoch-1",
		Revision: revision,
		AsOf:     feedBase().Add(time.Duration(asOfSeconds) * time.Second),
		Complete: true,
	}}

	owner := "owner"
	for i, id := range taskIDs {
		created := feedBase().Add(time.Duration(i) * time.Minute)
		task := leadfeed.Task{
			ID:          id,
			Title:       revision,
			Project:     "alpha",
			Agent:       "claude",
			CreatedAt:   created,
			Status:      "done",
			StatusSince: created,
			Spent:       leadfeed.Spent{Work: 60},
			Transitions: []leadfeed.Transition{{At: created, Status: "done"}},
		}
		if i == 0 {
			task.Status = "wait_user"
			task.WaitingOn = &owner
		}
		snap.Tasks = append(snap.Tasks, task)

		body := "letter for " + id
		missing := "decision without a note"
		snap.Records = append(snap.Records,
			leadfeed.Record{
				ID: revision + ":" + id + ":letter", Task: id, Kind: "letter", At: created, Order: 0,
				Actor: "lead", Source: leadfeed.Source{Kind: "journal", Ref: "1"}, Body: &body, Format: "text",
			},
			leadfeed.Record{
				ID:          revision + ":" + id + ":decision",
				Task:        id,
				Kind:        "decision",
				At:          created,
				Order:       1,
				Actor:       "lead",
				Source:      leadfeed.Source{Kind: "journal", Ref: "2"},
				BodyMissing: &missing,
				Format:      "text",
			},
		)
	}
	return snap
}

type feedRead struct {
	state   *leadfeed.State
	attempt *leadfeed.Attempt
	tasks   []leadfeed.Task
}

func readFeed(t *testing.T, repo *mongodb.MongoLeadFeedRepository) feedRead {
	t.Helper()
	var out feedRead
	err := repo.Read(context.Background(), func(ctx context.Context, view leadfeed.View) error {
		out = feedRead{}
		var err error
		if out.state, err = view.State(ctx); err != nil {
			return err
		}
		if out.attempt, err = view.LastAttempt(ctx); err != nil {
			return err
		}
		out.tasks, err = view.Tasks(ctx, leadfeed.TaskFilter{}, 0, 1000)
		return err
	})
	require.NoError(t, err)
	return out
}

func taskIDs(tasks []leadfeed.Task) []string {
	ids := make([]string, 0, len(tasks))
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}
	return ids
}

func TestLeadFeedRepository_EmptyFeed(t *testing.T) {
	repo := setupLeadFeedRepository(t)

	got := readFeed(t, repo)

	assert.Nil(t, got.state)
	assert.Nil(t, got.attempt)
	assert.Empty(t, got.tasks)
}

func TestLeadFeedRepository_PublishAndRead(t *testing.T) {
	repo := setupLeadFeedRepository(t)
	ctx := context.Background()
	now := feedBase().Add(time.Hour)

	result, err := repo.Publish(ctx, feedSnapshot("rev-a", 10, "T-0001", "T-0002", "T-0003"), now)
	require.NoError(t, err)
	assert.Equal(t, leadfeed.OutcomePublished, result.Outcome)
	assert.Empty(t, result.PreviousEpoch)

	got := readFeed(t, repo)
	require.NotNil(t, got.state)
	assert.Equal(t, "rev-a", got.state.Revision)
	assert.Equal(t, 3, got.state.Tasks)
	assert.Equal(t, 6, got.state.Records)
	assert.True(t, got.state.AsOf.Equal(feedBase().Add(10*time.Second)))
	assert.True(t, got.state.PublishedAt.Equal(now))
	assert.Equal(t, []string{"T-0003", "T-0002", "T-0001"}, taskIDs(got.tasks), "newest first")

	err = repo.Read(ctx, func(ctx context.Context, view leadfeed.View) error {
		task, taskErr := view.Task(ctx, "T-0001")
		require.NoError(t, taskErr)
		assert.Equal(t, "wait_user", task.Status)
		require.NotNil(t, task.WaitingOn)
		assert.Equal(t, "owner", *task.WaitingOn)
		assert.Equal(t, int64(60), task.Spent.Work)
		require.Len(t, task.Transitions, 1)

		records, recErr := view.Records(ctx, "T-0001", 0, 10)
		require.NoError(t, recErr)
		require.Len(t, records, 2)
		assert.Equal(t, "letter", records[0].Kind)
		require.NotNil(t, records[0].Body)
		assert.Equal(t, "letter for T-0001", *records[0].Body)
		assert.Nil(t, records[1].Body)
		require.NotNil(t, records[1].BodyMissing)

		second, pageErr := view.Records(ctx, "T-0001", 1, 10)
		require.NoError(t, pageErr)
		require.Len(t, second, 1)
		assert.Equal(t, "decision", second[0].Kind)

		_, missingErr := view.Task(ctx, "T-0999")
		require.ErrorIs(t, missingErr, leadfeed.ErrTaskNotFound)
		return nil
	})
	require.NoError(t, err)
}

func TestLeadFeedRepository_TaskFilters(t *testing.T) {
	repo := setupLeadFeedRepository(t)
	ctx := context.Background()
	_, err := repo.Publish(ctx, feedSnapshot("rev-a", 10, "T-0001", "T-0002", "T-0003"), feedBase())
	require.NoError(t, err)

	owner, nobody, done := "owner", leadfeed.WaitingOnNobody, "done"
	tests := []struct {
		name   string
		filter leadfeed.TaskFilter
		offset int
		limit  int
		want   []string
	}{
		{name: "waiting on owner", filter: leadfeed.TaskFilter{WaitingOn: &owner}, limit: 10, want: []string{"T-0001"}},
		{name: "waiting on nobody", filter: leadfeed.TaskFilter{WaitingOn: &nobody}, limit: 10,
			want: []string{"T-0003", "T-0002"}},
		{name: "by status", filter: leadfeed.TaskFilter{Status: &done}, limit: 10, want: []string{"T-0003", "T-0002"}},
		{name: "second page", offset: 2, limit: 2, want: []string{"T-0001"}},
		{name: "limit", limit: 1, want: []string{"T-0003"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = repo.Read(ctx, func(ctx context.Context, view leadfeed.View) error {
				tasks, listErr := view.Tasks(ctx, tt.filter, tt.offset, tt.limit)
				require.NoError(t, listErr)
				assert.Equal(t, tt.want, taskIDs(tasks))
				return nil
			})
			require.NoError(t, err)
		})
	}
}

func TestLeadFeedRepository_ReplaceDropsMissingTasks(t *testing.T) {
	repo := setupLeadFeedRepository(t)
	ctx := context.Background()

	_, err := repo.Publish(ctx, feedSnapshot("rev-a", 10, "T-0001", "T-0002"), feedBase())
	require.NoError(t, err)
	_, err = repo.Publish(ctx, feedSnapshot("rev-b", 20, "T-0002", "T-0003"), feedBase())
	require.NoError(t, err)

	got := readFeed(t, repo)
	assert.Equal(t, "rev-b", got.state.Revision)
	assert.Equal(t, []string{"T-0003", "T-0002"}, taskIDs(got.tasks))
	for _, task := range got.tasks {
		assert.Equal(t, "rev-b", task.Title)
	}

	err = repo.Read(ctx, func(ctx context.Context, view leadfeed.View) error {
		records, recErr := view.Records(ctx, "T-0001", 0, 10)
		require.NoError(t, recErr)
		assert.Empty(t, records, "records of a dropped task are gone")
		return nil
	})
	require.NoError(t, err)
}

func TestLeadFeedRepository_RepeatAdvancesWatermark(t *testing.T) {
	repo := setupLeadFeedRepository(t)
	ctx := context.Background()

	first, err := repo.Publish(ctx, feedSnapshot("rev-x", 10, "T-0001"), feedBase())
	require.NoError(t, err)
	assert.Equal(t, leadfeed.OutcomePublished, first.Outcome)

	repeat, err := repo.Publish(ctx, feedSnapshot("rev-x", 20, "T-0001"), feedBase().Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, leadfeed.OutcomeRefreshed, repeat.Outcome)

	_, err = repo.Publish(ctx, feedSnapshot("rev-y", 15, "T-0001", "T-0002"), feedBase().Add(2*time.Minute))
	require.ErrorIs(t, err, leadfeed.ErrSnapshotStale, "a snapshot older than the confirmed one must lose")

	got := readFeed(t, repo)
	assert.Equal(t, "rev-x", got.state.Revision)
	assert.True(t, got.state.AsOf.Equal(feedBase().Add(20*time.Second)))
	assert.True(t, got.state.PublishedAt.Equal(feedBase().Add(time.Minute)))
	assert.Equal(t, []string{"T-0001"}, taskIDs(got.tasks))
}

func TestLeadFeedRepository_SameAsOf(t *testing.T) {
	repo := setupLeadFeedRepository(t)
	ctx := context.Background()
	_, err := repo.Publish(ctx, feedSnapshot("rev-x", 10, "T-0001"), feedBase())
	require.NoError(t, err)

	t.Run("same content is a harmless repeat", func(t *testing.T) {
		result, repeatErr := repo.Publish(ctx, feedSnapshot("rev-x", 10, "T-0001"), feedBase())
		require.NoError(t, repeatErr)
		assert.Equal(t, leadfeed.OutcomeRefreshed, result.Outcome)
	})

	t.Run("different content is a conflict", func(t *testing.T) {
		_, conflictErr := repo.Publish(ctx, feedSnapshot("rev-y", 10, "T-0002"), feedBase())
		require.ErrorIs(t, conflictErr, leadfeed.ErrSnapshotConflict)

		got := readFeed(t, repo)
		assert.Equal(t, "rev-x", got.state.Revision)
		assert.Equal(t, []string{"T-0001"}, taskIDs(got.tasks))
	})
}

func TestLeadFeedRepository_EpochChange(t *testing.T) {
	repo := setupLeadFeedRepository(t)
	ctx := context.Background()
	_, err := repo.Publish(ctx, feedSnapshot("rev-x", 10, "T-0001"), feedBase())
	require.NoError(t, err)

	t.Run("older snapshot of another epoch is refused", func(t *testing.T) {
		other := feedSnapshot("rev-x", 5, "T-0001")
		other.Header.Epoch = "epoch-2"
		_, staleErr := repo.Publish(ctx, other, feedBase())
		require.ErrorIs(t, staleErr, leadfeed.ErrSnapshotStale)
	})

	t.Run("newer snapshot of another epoch replaces the feed", func(t *testing.T) {
		other := feedSnapshot("rev-x", 20, "T-0009")
		other.Header.Epoch = "epoch-2"
		result, pubErr := repo.Publish(ctx, other, feedBase())
		require.NoError(t, pubErr)
		assert.Equal(t, leadfeed.OutcomePublished, result.Outcome)
		assert.Equal(t, "epoch-1", result.PreviousEpoch)

		got := readFeed(t, repo)
		assert.Equal(t, "epoch-2", got.state.Epoch)
		assert.Equal(t, []string{"T-0009"}, taskIDs(got.tasks))
	})
}

func TestLeadFeedRepository_FailedWriteKeepsPreviousSnapshot(t *testing.T) {
	repo := setupLeadFeedRepository(t)
	ctx := context.Background()
	_, err := repo.Publish(ctx, feedSnapshot("rev-a", 10, "T-0001", "T-0002"), feedBase())
	require.NoError(t, err)

	// The duplicate id fails the insert after both DeleteMany calls have already run.
	broken := feedSnapshot("rev-b", 20, "T-0003")
	broken.Records = append(broken.Records, broken.Records[0])

	_, err = repo.Publish(ctx, broken, feedBase())
	require.Error(t, err)
	require.NotErrorIs(t, err, leadfeed.ErrCommitUnknown)

	got := readFeed(t, repo)
	assert.Equal(t, "rev-a", got.state.Revision)
	assert.True(t, got.state.AsOf.Equal(feedBase().Add(10*time.Second)))
	assert.Equal(t, []string{"T-0002", "T-0001"}, taskIDs(got.tasks))
}

func TestLeadFeedRepository_AttemptDoesNotTouchState(t *testing.T) {
	repo := setupLeadFeedRepository(t)
	ctx := context.Background()
	_, err := repo.Publish(ctx, feedSnapshot("rev-a", 10, "T-0001"), feedBase())
	require.NoError(t, err)

	attemptAt := feedBase().Add(time.Hour)
	require.NoError(t, repo.RecordAttempt(ctx, leadfeed.Attempt{
		At:       attemptAt,
		Outcome:  leadfeed.AttemptRejected,
		Error:    "snapshot has no footer",
		Revision: "rev-b",
	}))

	got := readFeed(t, repo)
	assert.Equal(t, "rev-a", got.state.Revision)
	assert.True(t, got.state.AsOf.Equal(feedBase().Add(10*time.Second)))
	require.NotNil(t, got.attempt)
	assert.Equal(t, leadfeed.AttemptRejected, got.attempt.Outcome)
	assert.Equal(t, "snapshot has no footer", got.attempt.Error)
	assert.True(t, got.attempt.At.Equal(attemptAt))
}

func TestLeadFeedRepository_ConcurrentPublishersKeepNewest(t *testing.T) {
	repo := setupLeadFeedRepository(t)
	ctx := context.Background()
	const publishers = 8

	var wg sync.WaitGroup
	errs := make([]error, publishers)
	for i := range publishers {
		wg.Go(func() {
			ids := make([]string, 0, i+1)
			for n := range i + 1 {
				ids = append(ids, fmt.Sprintf("T-%04d", n))
			}
			_, errs[i] = repo.Publish(ctx, feedSnapshot(fmt.Sprintf("rev-%d", i), 10+i, ids...), feedBase())
		})
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			require.ErrorIs(t, err, leadfeed.ErrSnapshotStale, "publisher %d", i)
		}
	}
	require.NoError(t, errs[publishers-1], "the newest snapshot always wins")

	got := readFeed(t, repo)
	assert.Equal(t, fmt.Sprintf("rev-%d", publishers-1), got.state.Revision)
	assert.Len(t, got.tasks, publishers)
	assert.Equal(t, publishers, got.state.Tasks)
}

func TestLeadFeedRepository_ReadDuringReplaceSeesOneSnapshot(t *testing.T) {
	repo := setupLeadFeedRepository(t)
	ctx := context.Background()
	const rounds = 40

	small := []string{"T-0001", "T-0002", "T-0003"}
	large := []string{"T-0001", "T-0002", "T-0003", "T-0004", "T-0005", "T-0006", "T-0007"}
	_, err := repo.Publish(ctx, feedSnapshot("rev-0", 0, small...), feedBase())
	require.NoError(t, err)

	done := make(chan struct{})
	var writeErr error
	go func() {
		defer close(done)
		for i := 1; i <= rounds; i++ {
			ids := small
			if i%2 == 1 {
				ids = large
			}
			if _, writeErr = repo.Publish(
				ctx,
				feedSnapshot(fmt.Sprintf("rev-%d", i), i, ids...),
				feedBase(),
			); writeErr != nil {
				return
			}
		}
	}()

	reads := 0
	for reading := true; reading; {
		select {
		case <-done:
			reading = false
		default:
		}

		got := readFeed(t, repo)
		reads++
		require.NotNil(t, got.state)
		require.Len(t, got.tasks, got.state.Tasks, "task count must match the state read with it")
		for _, task := range got.tasks {
			require.Equal(t, got.state.Revision, task.Title, "task from another snapshot")
		}
	}

	require.NoError(t, writeErr)
	assert.Positive(t, reads)
	assert.Equal(t, fmt.Sprintf("rev-%d", rounds), readFeed(t, repo).state.Revision)
}

func TestLeadFeedRepository_QuestionSurvivesPublication(t *testing.T) {
	repo := setupLeadFeedRepository(t)
	ctx := context.Background()

	f, err := os.Open("../../../../tests/fixtures/leadfeed/questions.jsonl")
	require.NoError(t, err)
	defer f.Close()
	snap, err := leadfeed.ParseSnapshot(f)
	require.NoError(t, err)

	_, err = repo.Publish(ctx, snap, feedBase())
	require.NoError(t, err)

	asked := 0
	err = repo.Read(ctx, func(ctx context.Context, view leadfeed.View) error {
		for i := range snap.Records {
			want := snap.Records[i]
			if want.Question == nil {
				continue
			}
			asked++
			got, qErr := view.Question(ctx, want.Question.QID)
			require.NoError(t, qErr)
			assert.Equal(t, want, *got, "question %s", want.Question.QID)
		}

		_, missingErr := view.Question(ctx, "Q-9999")
		require.ErrorIs(t, missingErr, leadfeed.ErrQuestionNotFound)

		letter, letterErr := view.Records(ctx, "T-0001", 0, 1)
		require.NoError(t, letterErr)
		require.Len(t, letter, 1)
		assert.Nil(t, letter[0].Question)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 7, asked)
}

func TestLeadFeedRepository_QuestionFollowsReplacement(t *testing.T) {
	repo := setupLeadFeedRepository(t)
	ctx := context.Background()

	withQuestion := feedSnapshot("rev-a", 10, "T-0001")
	withQuestion.Header.Schema = leadfeed.SchemaQuestions
	withQuestion.Records = append(withQuestion.Records, fixtures.LeadFeedQuestion("j:q", "T-0001", "Q-0001", 5))
	_, err := repo.Publish(ctx, withQuestion, feedBase())
	require.NoError(t, err)

	_, err = repo.Publish(ctx, feedSnapshot("rev-b", 20, "T-0001"), feedBase())
	require.NoError(t, err)

	err = repo.Read(ctx, func(ctx context.Context, view leadfeed.View) error {
		_, qErr := view.Question(ctx, "Q-0001")
		return qErr
	})
	require.ErrorIs(t, err, leadfeed.ErrQuestionNotFound)
}
