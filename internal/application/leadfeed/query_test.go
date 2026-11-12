package leadfeed_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
)

// memFeed is an in-memory feed; it runs every read twice to prove results are rebuilt, not appended.
type memFeed struct {
	state    *leadfeed.State
	attempt  *leadfeed.Attempt
	delivery *leadfeed.Delivery
	tasks    []leadfeed.Task
	records  map[string][]leadfeed.Record
	filters  []leadfeed.TaskFilter
	readErr  error
}

func (m *memFeed) Read(ctx context.Context, fn func(ctx context.Context, view leadfeed.View) error) error {
	if m.readErr != nil {
		return m.readErr
	}
	if err := fn(ctx, memView{m}); err != nil {
		return err
	}
	return fn(ctx, memView{m})
}

type memView struct{ feed *memFeed }

func (v memView) State(context.Context) (*leadfeed.State, error) { return v.feed.state, nil }

func (v memView) LastAttempt(context.Context) (*leadfeed.Attempt, error) { return v.feed.attempt, nil }

func (v memView) Delivery(context.Context) (*leadfeed.Delivery, error) { return v.feed.delivery, nil }

func (v memView) Tasks(_ context.Context, filter leadfeed.TaskFilter, offset, limit int) ([]leadfeed.Task, error) {
	v.feed.filters = append(v.feed.filters, filter)
	return window(v.feed.tasks, offset, limit), nil
}

func (v memView) Task(_ context.Context, id string) (*leadfeed.Task, error) {
	for i := range v.feed.tasks {
		if v.feed.tasks[i].ID == id {
			return &v.feed.tasks[i], nil
		}
	}
	return nil, leadfeed.ErrTaskNotFound
}

func (v memView) Records(_ context.Context, taskID string, offset, limit int) ([]leadfeed.Record, error) {
	return window(v.feed.records[taskID], offset, limit), nil
}

func (v memView) Question(_ context.Context, qid string) (*leadfeed.Record, error) {
	for _, recs := range v.feed.records {
		for i := range recs {
			if recs[i].Question != nil && recs[i].Question.QID == qid {
				return &recs[i], nil
			}
		}
	}
	return nil, leadfeed.ErrQuestionNotFound
}

func (v memView) Questions(context.Context) ([]leadfeed.Record, error) {
	var out []leadfeed.Record
	for _, recs := range v.feed.records {
		for i := range recs {
			if recs[i].Question != nil {
				out = append(out, recs[i])
			}
		}
	}
	return out, nil
}

func window[T any](items []T, offset, limit int) []T {
	if offset >= len(items) {
		return nil
	}
	return items[offset:min(offset+limit, len(items))]
}

func queryAsOf() time.Time { return time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC) }

func newMemFeed(tasks int) *memFeed {
	feed := &memFeed{
		state:   &leadfeed.State{Source: "lead-task", Epoch: "e1", Revision: "rev-a", AsOf: queryAsOf(), Tasks: tasks},
		records: map[string][]leadfeed.Record{},
	}
	for i := range tasks {
		id := fmt.Sprintf("T-%04d", i+1)
		feed.tasks = append(feed.tasks, leadfeed.Task{ID: id, Status: "work"})
		for n := range 3 {
			feed.records[id] = append(
				feed.records[id],
				leadfeed.Record{ID: fmt.Sprintf("%s:%d", id, n), Task: id, Order: n},
			)
		}
	}
	return feed
}

func TestStatusDisplay(t *testing.T) {
	tests := []struct {
		status string
		label  string
		group  string
	}{
		{"queued", "в очереди", leadfeed.GroupAgent},
		{"work", "в работе", leadfeed.GroupAgent},
		{"question", "вопрос к лиду", leadfeed.GroupLead},
		{"review", "на проверке у лида", leadfeed.GroupLead},
		{"wait_user", "ждёт тебя", leadfeed.GroupOwner},
		{"done", "принята", leadfeed.GroupClosed},
		{"cancelled", "отменена", leadfeed.GroupClosed},
		{"paused", "paused", leadfeed.GroupUnknown},
		{"", "", leadfeed.GroupUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			assert.Equal(t, tt.label, leadfeed.StatusLabel(tt.status))
			assert.Equal(t, tt.group, leadfeed.StatusGroup(tt.status))
		})
	}
}

func TestQueryService_ListTasks_PagesCoverEveryTaskOnce(t *testing.T) {
	svc := leadfeed.NewQueryService(newMemFeed(5))
	ctx := context.Background()

	var seen []string
	cursor := ""
	for page := 0; ; page++ {
		got, err := svc.ListTasks(ctx, leadfeed.TaskListQuery{Limit: 2, Cursor: cursor})
		require.NoError(t, err)
		assert.Equal(t, "rev-a", got.Revision)
		assert.True(t, got.AsOf.Equal(queryAsOf()))
		assert.LessOrEqual(t, len(got.Tasks), 2)
		for _, task := range got.Tasks {
			seen = append(seen, task.ID)
		}
		if got.NextCursor == "" {
			assert.Equal(t, 2, page, "five tasks by two take three pages")
			break
		}
		cursor = got.NextCursor
	}

	assert.Equal(t, []string{"T-0001", "T-0002", "T-0003", "T-0004", "T-0005"}, seen)
}

func TestQueryService_ListTasks_Limits(t *testing.T) {
	feed := newMemFeed(leadfeed.MaxPageSize + 10)
	svc := leadfeed.NewQueryService(feed)
	ctx := context.Background()

	t.Run("default page size", func(t *testing.T) {
		got, err := svc.ListTasks(ctx, leadfeed.TaskListQuery{})
		require.NoError(t, err)
		assert.Len(t, got.Tasks, leadfeed.DefaultPageSize)
		assert.NotEmpty(t, got.NextCursor)
	})

	t.Run("limit above the maximum is clamped", func(t *testing.T) {
		got, err := svc.ListTasks(ctx, leadfeed.TaskListQuery{Limit: 100000})
		require.NoError(t, err)
		assert.Len(t, got.Tasks, leadfeed.MaxPageSize)
	})

	t.Run("negative limit", func(t *testing.T) {
		_, err := svc.ListTasks(ctx, leadfeed.TaskListQuery{Limit: -1})
		require.ErrorIs(t, err, leadfeed.ErrInvalidLimit)
	})

	t.Run("exact fit has no next cursor", func(t *testing.T) {
		exact := leadfeed.NewQueryService(newMemFeed(2))
		got, err := exact.ListTasks(ctx, leadfeed.TaskListQuery{Limit: 2})
		require.NoError(t, err)
		assert.Len(t, got.Tasks, 2)
		assert.Empty(t, got.NextCursor)
	})
}

func TestQueryService_ListTasks_PassesFilter(t *testing.T) {
	feed := newMemFeed(1)
	owner := "owner"

	_, err := leadfeed.NewQueryService(feed).ListTasks(context.Background(),
		leadfeed.TaskListQuery{Filter: leadfeed.TaskFilter{WaitingOn: &owner}})

	require.NoError(t, err)
	require.NotEmpty(t, feed.filters)
	require.NotNil(t, feed.filters[0].WaitingOn)
	assert.Equal(t, "owner", *feed.filters[0].WaitingOn)
}

func TestQueryService_CursorOfAnotherRevision(t *testing.T) {
	feed := newMemFeed(5)
	svc := leadfeed.NewQueryService(feed)
	ctx := context.Background()

	first, err := svc.ListTasks(ctx, leadfeed.TaskListQuery{Limit: 2})
	require.NoError(t, err)
	records, err := svc.Records(ctx, "T-0001", 1, "")
	require.NoError(t, err)

	feed.state.Revision = "rev-b"

	_, err = svc.ListTasks(ctx, leadfeed.TaskListQuery{Limit: 2, Cursor: first.NextCursor})
	require.ErrorIs(t, err, leadfeed.ErrRevisionChanged)

	_, err = svc.Records(ctx, "T-0001", 1, records.NextCursor)
	require.ErrorIs(t, err, leadfeed.ErrRevisionChanged)

	restart, err := svc.ListTasks(ctx, leadfeed.TaskListQuery{Limit: 2})
	require.NoError(t, err, "paging restarts without a cursor")
	assert.Equal(t, "rev-b", restart.Revision)
}

func TestQueryService_InvalidCursor(t *testing.T) {
	svc := leadfeed.NewQueryService(newMemFeed(5))

	for _, cursor := range []string{"%%%", "bm90LWEtY3Vyc29y", "djF8cmV2LWF8LTE", "djJ8cmV2LWF8MA", "djF8fDA"} {
		t.Run(cursor, func(t *testing.T) {
			_, err := svc.ListTasks(context.Background(), leadfeed.TaskListQuery{Cursor: cursor})
			require.ErrorIs(t, err, leadfeed.ErrInvalidCursor)
		})
	}
}

func TestQueryService_TaskAndRecords(t *testing.T) {
	svc := leadfeed.NewQueryService(newMemFeed(2))
	ctx := context.Background()

	detail, err := svc.Task(ctx, "T-0002")
	require.NoError(t, err)
	assert.Equal(t, "T-0002", detail.Task.ID)
	assert.Equal(t, "rev-a", detail.Revision)

	_, err = svc.Task(ctx, "T-0999")
	require.ErrorIs(t, err, leadfeed.ErrTaskNotFound)

	_, err = svc.Records(ctx, "T-0999", 0, "")
	require.ErrorIs(t, err, leadfeed.ErrTaskNotFound)

	first, err := svc.Records(ctx, "T-0001", 2, "")
	require.NoError(t, err)
	assert.Equal(t, "T-0001", first.Task.ID)
	require.Len(t, first.Records, 2)
	require.NotEmpty(t, first.NextCursor)

	second, err := svc.Records(ctx, "T-0001", 2, first.NextCursor)
	require.NoError(t, err)
	require.Len(t, second.Records, 1)
	assert.Equal(t, "T-0001:2", second.Records[0].ID)
	assert.Empty(t, second.NextCursor)
}

func TestQueryService_EmptyFeed(t *testing.T) {
	svc := leadfeed.NewQueryService(&memFeed{})
	ctx := context.Background()

	page, err := svc.ListTasks(ctx, leadfeed.TaskListQuery{})
	require.NoError(t, err)
	assert.Empty(t, page.Revision)
	assert.Empty(t, page.Tasks)

	status, err := svc.Status(ctx)
	require.NoError(t, err)
	assert.Nil(t, status.State)
	assert.False(t, status.Healthy())

	_, err = svc.Task(ctx, "T-0001")
	require.ErrorIs(t, err, leadfeed.ErrTaskNotFound)

	board, err := svc.Board(ctx)
	require.NoError(t, err)
	assert.Empty(t, board.Tasks)
}

func TestFeedStatus_Healthy(t *testing.T) {
	state := &leadfeed.State{Revision: "rev-a"}
	tests := []struct {
		name    string
		status  leadfeed.FeedStatus
		healthy bool
	}{
		{name: "published, no attempt on record", status: leadfeed.FeedStatus{State: state}, healthy: true},
		{
			name: "last attempt published",
			status: leadfeed.FeedStatus{
				State:       state,
				LastAttempt: &leadfeed.Attempt{Outcome: "published"},
			},
			healthy: true,
		},
		{
			name: "last attempt refreshed",
			status: leadfeed.FeedStatus{
				State:       state,
				LastAttempt: &leadfeed.Attempt{Outcome: "refreshed"},
			},
			healthy: true,
		},
		{
			name: "last attempt rejected",
			status: leadfeed.FeedStatus{
				State:       state,
				LastAttempt: &leadfeed.Attempt{Outcome: leadfeed.AttemptRejected},
			},
		},
		{
			name: "last attempt unknown",
			status: leadfeed.FeedStatus{
				State:       state,
				LastAttempt: &leadfeed.Attempt{Outcome: leadfeed.AttemptUnknown},
			},
		},
		{name: "nothing published", status: leadfeed.FeedStatus{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.healthy, tt.status.Healthy())
		})
	}
}

func TestQueryService_Board(t *testing.T) {
	feed := newMemFeed(3)
	feed.attempt = &leadfeed.Attempt{Outcome: leadfeed.AttemptRejected, Error: "snapshot has no footer"}

	board, err := leadfeed.NewQueryService(feed).Board(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "rev-a", board.Revision)
	assert.Len(t, board.Tasks, 3)
	assert.False(t, board.Truncated)
	assert.False(t, board.Status.Healthy())
	assert.Equal(t, "snapshot has no footer", board.Status.LastAttempt.Error)
}

func TestQueryService_Board_Truncated(t *testing.T) {
	board, err := leadfeed.NewQueryService(newMemFeed(1001)).Board(context.Background())

	require.NoError(t, err)
	assert.Len(t, board.Tasks, 1000)
	assert.True(t, board.Truncated)
}

func TestQueryService_ReadError(t *testing.T) {
	svc := leadfeed.NewQueryService(&memFeed{readErr: context.DeadlineExceeded})

	_, err := svc.ListTasks(context.Background(), leadfeed.TaskListQuery{})

	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestQueryService_Thread(t *testing.T) {
	feed := newMemFeed(2)
	feed.attempt = &leadfeed.Attempt{Outcome: string(leadfeed.OutcomePublished)}
	svc := leadfeed.NewQueryService(feed)
	ctx := context.Background()

	first, err := svc.Thread(ctx, "T-0001", 2, "")
	require.NoError(t, err)
	assert.Equal(t, "T-0001", first.Task.ID)
	assert.Equal(t, "rev-a", first.Revision)
	assert.True(t, first.Status.Healthy())
	require.Len(t, first.Records, 2)
	require.NotEmpty(t, first.NextCursor)

	rest, err := svc.Thread(ctx, "T-0001", 2, first.NextCursor)
	require.NoError(t, err)
	require.Len(t, rest.Records, 1)
	assert.Empty(t, rest.NextCursor)

	_, err = svc.Thread(ctx, "T-0999", 0, "")
	require.ErrorIs(t, err, leadfeed.ErrTaskNotFound)

	_, err = svc.Thread(ctx, "T-0001", -1, "")
	require.ErrorIs(t, err, leadfeed.ErrInvalidLimit)
}
