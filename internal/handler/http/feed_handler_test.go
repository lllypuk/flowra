package httphandler_test

import (
	"context"
	"encoding/json"
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
)

type fakeFeedService struct {
	status     leadfeed.FeedStatus
	taskPage   leadfeed.TaskPage
	detail     leadfeed.TaskDetail
	recordPage leadfeed.RecordPage
	err        error

	gotQuery  leadfeed.TaskListQuery
	gotTaskID string
	gotLimit  int
	gotCursor string
}

func (f *fakeFeedService) Status(context.Context) (leadfeed.FeedStatus, error) {
	return f.status, f.err
}

func (f *fakeFeedService) ListTasks(_ context.Context, query leadfeed.TaskListQuery) (leadfeed.TaskPage, error) {
	f.gotQuery = query
	return f.taskPage, f.err
}

func (f *fakeFeedService) Task(_ context.Context, id string) (leadfeed.TaskDetail, error) {
	f.gotTaskID = id
	return f.detail, f.err
}

func (f *fakeFeedService) Records(
	_ context.Context,
	taskID string,
	limit int,
	cursor string,
) (leadfeed.RecordPage, error) {
	f.gotTaskID, f.gotLimit, f.gotCursor = taskID, limit, cursor
	return f.recordPage, f.err
}

func feedAsOf() time.Time { return time.Date(2026, 1, 10, 12, 0, 0, 123_000_000, time.UTC) }

func feedMeta() leadfeed.Meta { return leadfeed.Meta{Revision: "rev-a", AsOf: feedAsOf()} }

func newFeedServer(svc httphandler.FeedService) *echo.Echo {
	h := httphandler.NewFeedHandler(svc, nil)
	e := echo.New()
	e.GET("/api/v1/feed/status", h.Status)
	e.GET("/api/v1/feed/tasks", h.ListTasks)
	e.GET("/api/v1/feed/tasks/:id", h.GetTask)
	e.GET("/api/v1/feed/tasks/:id/records", h.ListRecords)
	return e
}

type feedEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func getFeed(t *testing.T, e *echo.Echo, target string) (int, feedEnvelope) {
	t.Helper()
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(stdhttp.MethodGet, target, nil))

	var env feedEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), rec.Body.String())
	return rec.Code, env
}

func decodeFeed[T any](t *testing.T, env feedEnvelope) T {
	t.Helper()
	var out T
	require.NoError(t, json.Unmarshal(env.Data, &out))
	return out
}

func TestFeedHandler_Status(t *testing.T) {
	t.Run("nothing published", func(t *testing.T) {
		code, env := getFeed(t, newFeedServer(&fakeFeedService{}), "/api/v1/feed/status")

		require.Equal(t, stdhttp.StatusOK, code)
		got := decodeFeed[map[string]any](t, env)
		assert.Equal(t, false, got["published"])
		assert.Equal(t, false, got["healthy"])
		assert.Empty(t, got["revision"])
		assert.Nil(t, got["as_of"])
		assert.Nil(t, got["last_attempt"])
	})

	t.Run("published with a failed last attempt", func(t *testing.T) {
		svc := &fakeFeedService{status: leadfeed.FeedStatus{
			State: &leadfeed.State{Source: "lead-task", Epoch: "e1", Revision: "rev-a", AsOf: feedAsOf(),
				Tasks: 8, Records: 21, PublishedAt: feedAsOf().Add(time.Second)},
			LastAttempt: &leadfeed.Attempt{At: feedAsOf().Add(time.Minute), Outcome: leadfeed.AttemptRejected,
				Error: "snapshot has no footer"},
		}}

		code, env := getFeed(t, newFeedServer(svc), "/api/v1/feed/status")

		require.Equal(t, stdhttp.StatusOK, code)
		got := decodeFeed[httphandler.FeedStatusResponse](t, env)
		assert.True(t, got.Published)
		assert.False(t, got.Healthy)
		assert.Equal(t, "rev-a", got.Revision)
		assert.Equal(t, 8, got.Tasks)
		assert.Equal(t, 21, got.Records)
		require.NotNil(t, got.LastAttempt)
		assert.Equal(t, "snapshot has no footer", got.LastAttempt.Error)

		raw := decodeFeed[map[string]any](t, env)
		assert.Equal(t, "2026-01-10T12:00:00.123Z", raw["as_of"], "time is RFC 3339 UTC")
	})
}

func TestFeedHandler_ListTasks(t *testing.T) {
	owner := "owner"
	svc := &fakeFeedService{taskPage: leadfeed.TaskPage{
		Meta: feedMeta(),
		Tasks: []leadfeed.Task{
			{ID: "T-0003", Title: "Rollout", Status: "wait_user", WaitingOn: &owner, Spent: leadfeed.Spent{User: 3300},
				Transitions: []leadfeed.Transition{{At: feedAsOf(), Status: "wait_user", Derived: true}}},
			{ID: "T-0001", Status: "done"},
			{ID: "T-0004", Status: "paused"},
		},
		NextCursor: "next",
	}}

	code, env := getFeed(t, newFeedServer(svc),
		"/api/v1/feed/tasks?waiting_on=owner&status=wait_user&limit=3&cursor=abc")

	require.Equal(t, stdhttp.StatusOK, code)
	assert.True(t, env.Success)

	require.NotNil(t, svc.gotQuery.Filter.WaitingOn)
	assert.Equal(t, "owner", *svc.gotQuery.Filter.WaitingOn)
	require.NotNil(t, svc.gotQuery.Filter.Status)
	assert.Equal(t, "wait_user", *svc.gotQuery.Filter.Status)
	assert.Equal(t, 3, svc.gotQuery.Limit)
	assert.Equal(t, "abc", svc.gotQuery.Cursor)

	got := decodeFeed[httphandler.FeedTaskListResponse](t, env)
	assert.Equal(t, "rev-a", got.Revision)
	require.NotNil(t, got.AsOf)
	assert.Equal(t, "next", got.NextCursor)
	require.Len(t, got.Items, 3)

	assert.Equal(t, "wait_user", got.Items[0].Status, "source status is returned as is")
	assert.Equal(t, "ждёт тебя", got.Items[0].StatusLabel)
	assert.Equal(t, leadfeed.GroupOwner, got.Items[0].Group)
	assert.Equal(t, int64(3300), got.Items[0].Spent.User)
	require.Len(t, got.Items[0].Transitions, 1)
	assert.True(t, got.Items[0].Transitions[0].Derived)

	assert.Nil(t, got.Items[1].WaitingOn)
	assert.Equal(t, leadfeed.GroupClosed, got.Items[1].Group)

	assert.Equal(t, "paused", got.Items[2].StatusLabel, "unknown status is shown raw")
	assert.Equal(t, leadfeed.GroupUnknown, got.Items[2].Group)
}

func TestFeedHandler_ListTasks_NoFiltersAndEmptyFeed(t *testing.T) {
	svc := &fakeFeedService{}

	code, env := getFeed(t, newFeedServer(svc), "/api/v1/feed/tasks")

	require.Equal(t, stdhttp.StatusOK, code)
	assert.Nil(t, svc.gotQuery.Filter.Status)
	assert.Nil(t, svc.gotQuery.Filter.WaitingOn)
	assert.Zero(t, svc.gotQuery.Limit)

	raw := decodeFeed[map[string]any](t, env)
	assert.Empty(t, raw["revision"])
	assert.Nil(t, raw["as_of"])
	assert.Equal(t, []any{}, raw["items"], "an empty page is an empty array, not null")
}

func TestFeedHandler_GetTask(t *testing.T) {
	svc := &fakeFeedService{
		detail: leadfeed.TaskDetail{Meta: feedMeta(), Task: leadfeed.Task{ID: "T-0007", Status: "review"}},
	}

	code, env := getFeed(t, newFeedServer(svc), "/api/v1/feed/tasks/T-0007")

	require.Equal(t, stdhttp.StatusOK, code)
	assert.Equal(t, "T-0007", svc.gotTaskID)
	got := decodeFeed[httphandler.FeedTaskDetailResponse](t, env)
	assert.Equal(t, "T-0007", got.Task.ID)
	assert.Equal(t, "на проверке у лида", got.Task.StatusLabel)
	assert.Equal(t, leadfeed.GroupLead, got.Task.Group)
}

func TestFeedHandler_ListRecords(t *testing.T) {
	body, missing := "#status Done\n<script>alert(1)</script>", "letter not found in the agent transcript"
	svc := &fakeFeedService{recordPage: leadfeed.RecordPage{
		Meta: feedMeta(),
		Task: leadfeed.Task{ID: "T-0002"},
		Records: []leadfeed.Record{
			{ID: "j:3", Task: "T-0002", Kind: "letter", At: feedAsOf(), Actor: "lead", BodyDerived: true,
				Source: leadfeed.Source{Kind: "journal", Ref: "3"}, BodyMissing: &missing, Format: "text"},
			{
				ID:     "j:4",
				Task:   "T-0002",
				Kind:   "letter",
				At:     feedAsOf(),
				Order:  1,
				Actor:  "lead",
				Body:   &body,
				Format: "text",
			},
		},
	}}

	code, env := getFeed(t, newFeedServer(svc), "/api/v1/feed/tasks/T-0002/records?limit=2&cursor=xyz")

	require.Equal(t, stdhttp.StatusOK, code)
	assert.Equal(t, "T-0002", svc.gotTaskID)
	assert.Equal(t, 2, svc.gotLimit)
	assert.Equal(t, "xyz", svc.gotCursor)

	got := decodeFeed[httphandler.FeedRecordListResponse](t, env)
	assert.Equal(t, "T-0002", got.TaskID)
	require.Len(t, got.Items, 2)
	assert.Nil(t, got.Items[0].Body)
	require.NotNil(t, got.Items[0].BodyMissing)
	assert.Equal(t, "journal", got.Items[0].Source.Kind)
	require.NotNil(t, got.Items[1].Body)
	assert.Equal(t, body, *got.Items[1].Body, "text is returned verbatim")
	assert.Nil(t, got.Items[1].BodyMissing)
}

func TestFeedHandler_Errors(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "unknown task", target: "/api/v1/feed/tasks/T-0999", err: leadfeed.ErrTaskNotFound,
			wantStatus: stdhttp.StatusNotFound, wantCode: "FEED_TASK_NOT_FOUND"},
		{name: "records of an unknown task", target: "/api/v1/feed/tasks/T-0999/records", err: leadfeed.ErrTaskNotFound,
			wantStatus: stdhttp.StatusNotFound, wantCode: "FEED_TASK_NOT_FOUND"},
		{
			name:       "revision changed during paging",
			target:     "/api/v1/feed/tasks?cursor=old",
			err:        leadfeed.ErrRevisionChanged,
			wantStatus: stdhttp.StatusConflict,
			wantCode:   "FEED_REVISION_CHANGED",
		},
		{name: "invalid cursor", target: "/api/v1/feed/tasks?cursor=%25", err: leadfeed.ErrInvalidCursor,
			wantStatus: stdhttp.StatusBadRequest, wantCode: "INVALID_CURSOR"},
		{name: "limit is not a number", target: "/api/v1/feed/tasks?limit=many",
			wantStatus: stdhttp.StatusBadRequest, wantCode: "INVALID_LIMIT"},
		{name: "zero limit", target: "/api/v1/feed/tasks/T-0001/records?limit=0",
			wantStatus: stdhttp.StatusBadRequest, wantCode: "INVALID_LIMIT"},
		{name: "negative limit", target: "/api/v1/feed/tasks?limit=-5",
			wantStatus: stdhttp.StatusBadRequest, wantCode: "INVALID_LIMIT"},
		{name: "store failure on status", target: "/api/v1/feed/status", err: errors.New("no primary"),
			wantStatus: stdhttp.StatusServiceUnavailable, wantCode: "FEED_UNAVAILABLE"},
		{name: "store failure on list", target: "/api/v1/feed/tasks", err: errors.New("no primary"),
			wantStatus: stdhttp.StatusServiceUnavailable, wantCode: "FEED_UNAVAILABLE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, env := getFeed(t, newFeedServer(&fakeFeedService{err: tt.err}), tt.target)

			assert.Equal(t, tt.wantStatus, code)
			assert.False(t, env.Success)
			require.NotNil(t, env.Error)
			assert.Equal(t, tt.wantCode, env.Error.Code)
			assert.NotContains(t, env.Error.Message, "no primary", "internal errors are not leaked")
		})
	}
}

func TestFeedTime_FixedMilliseconds(t *testing.T) {
	tests := []struct {
		name string
		at   time.Time
		want string
	}{
		{
			name: "trailing zero is kept",
			at:   time.Date(2026, 11, 12, 4, 0, 17, 920_000_000, time.UTC),
			want: `"2026-11-12T04:00:17.920Z"`,
		},
		{name: "whole second", at: time.Date(2026, 11, 12, 4, 0, 17, 0, time.UTC), want: `"2026-11-12T04:00:17.000Z"`},
		{
			name: "other zone is converted to UTC",
			at: time.Date(
				2026,
				11,
				12,
				9,
				0,
				17,
				5_000_000,
				time.FixedZone("YEKT", 5*3600),
			),
			want: `"2026-11-12T04:00:17.005Z"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(httphandler.FeedTime{Time: tt.at})
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(raw))

			var back httphandler.FeedTime
			require.NoError(t, json.Unmarshal(raw, &back))
			assert.True(t, back.Equal(tt.at))
		})
	}
}
