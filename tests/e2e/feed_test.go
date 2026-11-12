//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/tests/fixtures"
)

const feedFixture = "../fixtures/leadfeed/full.jsonl"

func importFeedFixture(t *testing.T, suite *E2ETestSuite) {
	t.Helper()
	raw, err := os.ReadFile(feedFixture)
	require.NoError(t, err)
	_, err = leadfeed.NewImporter(suite.FeedRepo).Import(context.Background(), bytes.NewReader(raw))
	require.NoError(t, err)
}

func TestFeed_RequiresAuthentication(t *testing.T) {
	suite := NewE2ETestSuite(t)
	importFeedFixture(t, suite)

	for _, path := range []string{"/feed/status", "/feed/tasks", "/feed/tasks/T-0001", "/feed/tasks/T-0001/records"} {
		t.Run(path, func(t *testing.T) {
			AssertStatus(t, suite.NewHTTPClient("").Get(path), http.StatusUnauthorized)
			AssertStatus(t, suite.NewHTTPClient("not-a-token").Get(path), http.StatusUnauthorized)
		})
	}
}

func TestFeed_EmptyBeforeFirstImport(t *testing.T) {
	suite := NewE2ETestSuite(t)
	client := suite.NewHTTPClient(suite.CreateTestUser("feedempty").Token)

	status := ParseResponse[APIResponse[httphandler.FeedStatusResponse]](t, client.Get("/feed/status"))
	assert.False(t, status.Data.Published)
	assert.False(t, status.Data.Healthy)

	list := ParseResponse[APIResponse[httphandler.FeedTaskListResponse]](t, client.Get("/feed/tasks"))
	assert.Empty(t, list.Data.Revision)
	assert.Empty(t, list.Data.Items)

	AssertStatus(t, client.Get("/feed/tasks/T-0001"), http.StatusNotFound)
}

func TestFeed_ReadsImportedSnapshot(t *testing.T) {
	suite := NewE2ETestSuite(t)
	importFeedFixture(t, suite)
	client := suite.NewHTTPClient(suite.CreateTestUser("feedreader").Token)

	t.Run("status", func(t *testing.T) {
		resp := client.Get("/feed/status")
		AssertStatus(t, resp, http.StatusOK)
		status := ParseResponse[APIResponse[httphandler.FeedStatusResponse]](t, resp)

		assert.True(t, status.Data.Published)
		assert.True(t, status.Data.Healthy)
		assert.Equal(t, "389227faca902622", status.Data.Revision)
		assert.Equal(t, 4, status.Data.Tasks)
		assert.Equal(t, 9, status.Data.Records)
		require.NotNil(t, status.Data.LastAttempt)
		assert.Equal(t, "published", status.Data.LastAttempt.Outcome)
	})

	t.Run("list with source statuses and display groups", func(t *testing.T) {
		list := ParseResponse[APIResponse[httphandler.FeedTaskListResponse]](t, client.Get("/feed/tasks"))

		require.Len(t, list.Data.Items, 4)
		byID := map[string]httphandler.FeedTaskResponse{}
		for _, item := range list.Data.Items {
			byID[item.ID] = item
		}
		assert.Equal(t, "T-0004", list.Data.Items[0].ID, "newest first")
		assert.Equal(t, leadfeed.GroupClosed, byID["T-0001"].Group)
		assert.Nil(t, byID["T-0001"].WaitingOn)
		assert.Equal(t, "review", byID["T-0002"].Status)
		assert.Equal(t, leadfeed.GroupLead, byID["T-0002"].Group)
		assert.Equal(t, "ждёт тебя", byID["T-0003"].StatusLabel)
		assert.Equal(t, "paused", byID["T-0004"].Status)
		assert.Equal(t, leadfeed.GroupUnknown, byID["T-0004"].Group)
	})

	t.Run("filters", func(t *testing.T) {
		owner := ParseResponse[APIResponse[httphandler.FeedTaskListResponse]](t, client.Get("/feed/tasks?waiting_on=owner"))
		require.Len(t, owner.Data.Items, 1)
		assert.Equal(t, "T-0003", owner.Data.Items[0].ID)

		closed := ParseResponse[APIResponse[httphandler.FeedTaskListResponse]](t, client.Get("/feed/tasks?waiting_on=none"))
		require.Len(t, closed.Data.Items, 1)
		assert.Equal(t, "T-0001", closed.Data.Items[0].ID)

		review := ParseResponse[APIResponse[httphandler.FeedTaskListResponse]](t, client.Get("/feed/tasks?status=review"))
		require.Len(t, review.Data.Items, 1)
		assert.Equal(t, "T-0002", review.Data.Items[0].ID)
	})

	t.Run("paging walks every task once", func(t *testing.T) {
		var seen []string
		path := "/feed/tasks?limit=3"
		for {
			page := ParseResponse[APIResponse[httphandler.FeedTaskListResponse]](t, client.Get(path))
			for _, item := range page.Data.Items {
				seen = append(seen, item.ID)
			}
			if page.Data.NextCursor == "" {
				break
			}
			path = "/feed/tasks?limit=3&cursor=" + page.Data.NextCursor
		}
		assert.Equal(t, []string{"T-0004", "T-0003", "T-0002", "T-0001"}, seen)
	})

	t.Run("task with transitions", func(t *testing.T) {
		detail := ParseResponse[APIResponse[httphandler.FeedTaskDetailResponse]](t, client.Get("/feed/tasks/T-0001"))

		assert.Equal(t, "389227faca902622", detail.Data.Revision)
		require.Len(t, detail.Data.Task.Transitions, 4)
		assert.True(t, detail.Data.Task.Transitions[1].Derived)
		require.NotNil(t, detail.Data.Task.Transitions[1].Cause)
		assert.Equal(t, "j:1", *detail.Data.Task.Transitions[1].Cause)
	})

	t.Run("thread keeps source order, missing bodies and verbatim text", func(t *testing.T) {
		thread := ParseResponse[APIResponse[httphandler.FeedRecordListResponse]](t, client.Get("/feed/tasks/T-0002/records"))

		require.Len(t, thread.Data.Items, 3)
		assert.Equal(t, "j:3", thread.Data.Items[0].ID)
		assert.Nil(t, thread.Data.Items[0].Body)
		require.NotNil(t, thread.Data.Items[0].BodyMissing)
		require.NotNil(t, thread.Data.Items[1].Body)
		assert.Contains(t, *thread.Data.Items[1].Body, "#status Done\n#assignee @bob")
		assert.Contains(t, *thread.Data.Items[1].Body, "<script>alert(1)</script>")
		assert.True(t, thread.Data.Items[2].ActorDerived)
	})

	t.Run("unknown task", func(t *testing.T) {
		AssertStatus(t, client.Get("/feed/tasks/T-0999"), http.StatusNotFound)
		AssertStatus(t, client.Get("/feed/tasks/T-0999/records"), http.StatusNotFound)
	})

	t.Run("bad paging input", func(t *testing.T) {
		AssertStatus(t, client.Get("/feed/tasks?limit=0"), http.StatusBadRequest)
		AssertStatus(t, client.Get("/feed/tasks?cursor=not-a-cursor"), http.StatusBadRequest)
	})

	t.Run("write methods are not served", func(t *testing.T) {
		for _, resp := range []*http.Response{
			client.Post("/feed/tasks", map[string]string{"title": "x"}),
			client.Put("/feed/tasks/T-0001", map[string]string{"status": "done"}),
			client.Delete("/feed/tasks/T-0001"),
			client.Post("/feed/tasks/T-0001/records", map[string]string{"body": "x"}),
		} {
			assert.Contains(t, []int{http.StatusNotFound, http.StatusMethodNotAllowed}, resp.StatusCode)
			_ = resp.Body.Close()
		}
	})
}

func TestFeed_CursorAfterRepublish(t *testing.T) {
	suite := NewE2ETestSuite(t)
	importFeedFixture(t, suite)
	client := suite.NewHTTPClient(suite.CreateTestUser("feedcursor").Token)

	first := ParseResponse[APIResponse[httphandler.FeedTaskListResponse]](t, client.Get("/feed/tasks?limit=2"))
	require.NotEmpty(t, first.Data.NextCursor)

	raw, err := fixtures.LeadFeedSnapshotJSONL(fixtures.LeadFeedBase.Add(24*time.Hour),
		[]leadfeed.Task{fixtures.LeadFeedTask("T-0100", "work", "agent", 0)}, nil)
	require.NoError(t, err)
	_, err = leadfeed.NewImporter(suite.FeedRepo).Import(context.Background(), bytes.NewReader(raw))
	require.NoError(t, err)

	resp := client.Get("/feed/tasks?limit=2&cursor=" + first.Data.NextCursor)
	AssertStatus(t, resp, http.StatusConflict)
	conflict := ParseResponse[APIResponse[any]](t, resp)
	require.NotNil(t, conflict.Error)
	assert.Equal(t, "FEED_REVISION_CHANGED", conflict.Error.Code)

	restart := ParseResponse[APIResponse[httphandler.FeedTaskListResponse]](t, client.Get("/feed/tasks?limit=2"))
	require.Len(t, restart.Data.Items, 1)
	assert.Equal(t, "T-0100", restart.Data.Items[0].ID)
}
