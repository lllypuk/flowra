package httphandler_test

import (
	"context"
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/web"
)

type fakeFeedPages struct {
	board     leadfeed.Board
	thread    leadfeed.Thread
	err       error
	gotCursor string
	gotLimit  int
}

func (f *fakeFeedPages) Board(context.Context) (leadfeed.Board, error) { return f.board, f.err }

func (f *fakeFeedPages) Thread(_ context.Context, _ string, limit int, cursor string) (leadfeed.Thread, error) {
	f.gotLimit, f.gotCursor = limit, cursor
	return f.thread, f.err
}

func pageAsOf() time.Time { return time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC) }

func pageNow() time.Time { return pageAsOf().Add(5 * time.Minute) }

func healthyFeedStatus() leadfeed.FeedStatus {
	return leadfeed.FeedStatus{
		State: &leadfeed.State{
			Revision:    "rev-a",
			AsOf:        pageAsOf(),
			Tasks:       4,
			PublishedAt: pageAsOf().Add(time.Second),
		},
		LastAttempt: &leadfeed.Attempt{At: pageAsOf().Add(time.Second), Outcome: string(leadfeed.OutcomePublished)},
	}
}

// newFeedPages serves the feed pages behind the real page auth with an owner token.
func newFeedPages(t *testing.T, svc httphandler.FeedPageService) *echo.Echo {
	t.Helper()
	return newFeedPagesAt(t, svc, pageNow)
}

func newFeedPagesAt(t *testing.T, svc httphandler.FeedPageService, now func() time.Time) *echo.Echo {
	t.Helper()
	return newFeedPagesWithQuestions(t, svc, nil, now)
}

func newFeedPagesWithQuestions(
	t *testing.T, svc httphandler.FeedPageService, questions httphandler.FeedQuestionService, now func() time.Time,
) *echo.Echo {
	t.Helper()
	renderer, err := httphandler.NewTemplateRenderer(httphandler.TemplateRendererConfig{FS: web.TemplatesFS})
	require.NoError(t, err)
	usePageAuth(t, &httphandler.PageAuthConfig{TokenValidator: ownerValidator(t, ownerSecret)})

	handler := httphandler.NewFeedTemplateHandler(renderer, svc, nil)
	if questions != nil {
		handler.WithQuestions(questions)
	}
	handler.SetClock(now, time.UTC)

	e := echo.New()
	handler.SetupFeedRoutes(e)
	return e
}

func getPage(e *echo.Echo, target string, authenticated bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(stdhttp.MethodGet, target, nil)
	if authenticated {
		req.AddCookie(&stdhttp.Cookie{Name: sessionCookie, Value: ownerSecret})
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// feedContent returns the feed section only, without the shared navbar and layout.
func feedContent(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, "data-feed-content")
	require.NotEqual(t, -1, start, "feed content is missing")
	end := strings.Index(body[start:], ".feed-banner {")
	require.NotEqual(t, -1, end, "feed content end is missing")
	return body[start : start+end]
}

func feedBoard() leadfeed.Board {
	owner, lead, agent := "owner", "lead", "agent"
	at := func(minutes int) time.Time { return pageAsOf().Add(-time.Duration(minutes) * time.Minute) }
	return leadfeed.Board{
		Meta:   leadfeed.Meta{Revision: "rev-a", AsOf: pageAsOf()},
		Status: healthyFeedStatus(),
		Tasks: []leadfeed.Task{
			{
				ID:          "T-0004",
				Project:     "delta",
				Title:       "Paused thing",
				Status:      "paused",
				WaitingOn:   &agent,
				StatusSince: at(30),
			},
			{ID: "T-0003", Project: "gamma", Title: "Rollout <b>plan</b>", Status: "wait_user", WaitingOn: &owner,
				StatusSince: at(125), Note: "needs <script>alert(1)</script> decision"},
			{ID: "T-0002", Project: "beta", Title: "Fix build", Status: "review", WaitingOn: &lead, StatusSince: at(2)},
			{ID: "T-0005", Project: "beta", Title: "Refactor", Status: "work", WaitingOn: &agent, StatusSince: at(0)},
			{ID: "T-0001", Project: "alpha", Title: "Old task", Status: "done", StatusSince: at(3000)},
		},
	}
}

func TestFeedList_RequiresSession(t *testing.T) {
	e := newFeedPages(t, &fakeFeedPages{board: feedBoard()})

	for _, target := range []string{"/feed", "/feed/tasks/T-0001"} {
		rec := getPage(e, target, false)

		assert.Equal(t, stdhttp.StatusFound, rec.Code, target)
		assert.Equal(t, "/login", rec.Header().Get("Location"), target)
		assert.NotContains(t, rec.Body.String(), "T-0001")
	}
}

func TestFeedList_GroupsByWhoIsWaited(t *testing.T) {
	rec := getPage(newFeedPages(t, &fakeFeedPages{board: feedBoard()}), "/feed", true)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	content := feedContent(t, rec.Body.String())

	order := []string{
		`data-feed-group="owner"`, "T-0003",
		`data-feed-group="lead"`, "T-0002",
		`data-feed-group="agent"`, "T-0005",
		`data-feed-group="unknown"`, "T-0004",
		`data-feed-group="closed"`, "T-0001",
	}
	last := -1
	for _, marker := range order {
		pos := strings.Index(content, marker)
		require.Greater(t, pos, last, "%s is out of order", marker)
		last = pos
	}

	assert.Contains(t, content, "Ждут владельца")
	assert.Contains(t, content, `У агента <span class="feed-muted">· 1</span>`, "group headings carry their task count")
	assert.Contains(t, content, "ждёт тебя")
	assert.Contains(t, content, "wait_user", "source status is shown next to its label")
	assert.Contains(t, content, "2 ч 05 мин", "status age is measured to the snapshot time")
	assert.Contains(t, content, ">paused<", "unknown status is shown raw")
	assert.Contains(t, content, `href="/feed/tasks/T-0003"`)
	assert.Contains(t, content, "Задач: 5")
}

func TestFeedList_EscapesSourceText(t *testing.T) {
	rec := getPage(newFeedPages(t, &fakeFeedPages{board: feedBoard()}), "/feed", true)

	content := feedContent(t, rec.Body.String())
	assert.NotContains(t, content, "<script>alert(1)</script>")
	assert.Contains(t, content, "&lt;script&gt;alert(1)&lt;/script&gt;")
	assert.NotContains(t, content, "<b>plan</b>")
	assert.Contains(t, content, "Rollout &lt;b&gt;plan&lt;/b&gt;")
}

func TestFeedPages_HaveNoWriteControls(t *testing.T) {
	svc := &fakeFeedPages{board: feedBoard(), thread: feedThread()}
	e := newFeedPages(t, svc)

	for _, target := range []string{"/feed", "/feed/tasks/T-0002"} {
		content := feedContent(t, getPage(e, target, true).Body.String())

		for _, forbidden := range []string{"<form", "<button", "<input", "<textarea", "<select",
			"hx-post", "hx-put", "hx-patch", "hx-delete"} {
			assert.NotContains(t, content, forbidden, "%s on %s", forbidden, target)
		}
	}
}

func TestFeedList_Banner(t *testing.T) {
	t.Run("healthy feed shows snapshot time and age", func(t *testing.T) {
		content := feedContent(
			t,
			getPage(newFeedPages(t, &fakeFeedPages{board: feedBoard()}), "/feed", true).Body.String(),
		)

		assert.Contains(t, content, "10.01.2026 12:00")
		assert.Contains(t, content, "5 мин назад")
		assert.Contains(t, content, "rev-a")
		assert.NotContains(t, content, "data-feed-error")
		assert.NotContains(t, content, "feed-banner-problem")
		assert.NotContains(t, content, "data-feed-stale", "five minutes exactly is still fresh")
	})

	t.Run("old snapshot warns on the list and on the thread", func(t *testing.T) {
		later := func() time.Time { return pageAsOf().Add(5*time.Minute + time.Second) }
		e := newFeedPagesAt(t, &fakeFeedPages{board: feedBoard(), thread: feedThread()}, later)

		for _, target := range []string{"/feed", "/feed/tasks/T-0002"} {
			content := feedContent(t, getPage(e, target, true).Body.String())
			assert.Contains(t, content, "data-feed-stale", target)
			assert.Contains(t, content, "5 мин назад", target)
			assert.Contains(t, content, "feed-banner-problem", target)
		}
	})

	t.Run("failed import is visible and the old snapshot stays", func(t *testing.T) {
		board := feedBoard()
		board.Status.LastAttempt = &leadfeed.Attempt{
			At: pageAsOf().Add(3 * time.Minute), Outcome: leadfeed.AttemptRejected, Error: "snapshot has no footer",
		}

		content := feedContent(t, getPage(newFeedPages(t, &fakeFeedPages{board: board}), "/feed", true).Body.String())

		assert.Contains(t, content, "feed-banner-problem")
		assert.Contains(t, content, "data-feed-error")
		assert.Contains(t, content, "snapshot has no footer")
		assert.Contains(t, content, "Показан прежний снимок")
		assert.Contains(t, content, "T-0003", "tasks of the previous snapshot are still listed")
	})

	t.Run("nothing imported yet", func(t *testing.T) {
		content := feedContent(t, getPage(newFeedPages(t, &fakeFeedPages{}), "/feed", true).Body.String())

		assert.Contains(t, content, "Лента пуста")
		assert.Contains(t, content, "Задач: 0")
	})
}

func feedThread() leadfeed.Thread {
	lead := "lead"
	cause := "u:toolu_beta1"
	body := "#status Done\n#assignee @bob\nReturned: handle <script>alert(1)</script> & co."
	report := "[T-0002: done] fixed"
	missing := "letter not found in the agent transcript"
	return leadfeed.Thread{
		Status: healthyFeedStatus(),
		RecordPage: leadfeed.RecordPage{
			Meta: leadfeed.Meta{Revision: "rev-a", AsOf: pageAsOf()},
			Task: leadfeed.Task{
				ID: "T-0002", Project: "beta", Agent: "claude", Title: "Fix build", Status: "review", WaitingOn: &lead,
				CreatedAt: pageAsOf().Add(-2 * time.Hour), StatusSince: pageAsOf().Add(-80 * time.Minute), Returns: 1,
				Spent: leadfeed.Spent{Work: 2400, Lead: 4800},
				Transitions: []leadfeed.Transition{
					{At: pageAsOf().Add(-2 * time.Hour), Status: "queued"},
					{At: pageAsOf().Add(-80 * time.Minute), Status: "review", Cause: &cause, Derived: true},
				},
			},
			Records: []leadfeed.Record{
				{ID: "j:3", Kind: "letter", At: pageAsOf().Add(-2 * time.Hour), Actor: "lead", BodyDerived: true,
					Source: leadfeed.Source{Kind: "journal"}, BodyMissing: &missing},
				{
					ID:          "j:4",
					Kind:        "letter",
					At:          pageAsOf().Add(-2 * time.Hour),
					Order:       1,
					Actor:       "lead",
					BodyDerived: true,
					Source:      leadfeed.Source{Kind: "journal"},
					Body:        &body,
				},
				{ID: "u:toolu_beta1", Kind: "report", At: pageAsOf().Add(-80 * time.Minute), Order: 2, Actor: "agent",
					ActorDerived: true, Source: leadfeed.Source{Kind: "transcript"}, Body: &report},
			},
		},
	}
}

func TestFeedThread_ShowsTaskAndRecords(t *testing.T) {
	svc := &fakeFeedPages{thread: feedThread()}

	rec := getPage(newFeedPages(t, svc), "/feed/tasks/T-0002", true)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	assert.Equal(t, leadfeed.MaxPageSize, svc.gotLimit)
	content := feedContent(t, rec.Body.String())

	assert.Contains(t, content, "T-0002 · Fix build")
	assert.Contains(t, content, "на проверке у лида")
	assert.Contains(t, content, "ждём <strong>лид</strong>")
	assert.Contains(t, content, "возвратов 1")
	assert.Contains(t, content, "работа: 40 мин")
	assert.Contains(t, content, "ждёт лида: 1 ч 20 мин")

	first := strings.Index(content, `id="rec-j:3"`)
	second := strings.Index(content, `id="rec-j:4"`)
	third := strings.Index(content, `id="rec-u:toolu_beta1"`)
	assert.True(t, first != -1 && first < second && second < third, "records keep source order")
}

func TestFeedThread_MarksDerivedAndMissing(t *testing.T) {
	content := feedContent(
		t,
		getPage(newFeedPages(t, &fakeFeedPages{thread: feedThread()}), "/feed/tasks/T-0002", true).Body.String(),
	)

	transitions := content[strings.Index(content, "data-feed-transitions"):strings.Index(content, "Переписка")]
	assert.Equal(t, 1, strings.Count(transitions, "feed-derived"), "only the inferred transition is marked")
	assert.Contains(t, transitions, `href="#rec-u%3atoolu_beta1"`, "the cause links to its record")

	missing := content[strings.Index(content, `id="rec-j:3"`):strings.Index(content, `id="rec-j:4"`)]
	assert.Contains(t, missing, "data-feed-missing")
	assert.Contains(t, missing, "letter not found in the agent transcript")
	assert.NotContains(t, missing, "feed-body")

	reportAt := strings.Index(content, `id="rec-u:toolu_beta1"`)
	require.NotEqual(t, -1, reportAt)
	report := content[reportAt:]
	assert.Contains(t, report, "агент <span class=\"feed-derived\"", "inferred author is marked")
}

func TestFeedThread_TextIsShownVerbatimAndInert(t *testing.T) {
	content := feedContent(
		t,
		getPage(newFeedPages(t, &fakeFeedPages{thread: feedThread()}), "/feed/tasks/T-0002", true).Body.String(),
	)

	assert.Contains(t, content, "#status Done\n#assignee @bob\n", "tag-like lines are plain text")
	assert.NotContains(t, content, "<script>alert(1)</script>")
	assert.Contains(t, content, "&lt;script&gt;alert(1)&lt;/script&gt; &amp; co.")
}

func TestFeedThread_Paging(t *testing.T) {
	thread := feedThread()
	thread.NextCursor = "next+cursor"
	svc := &fakeFeedPages{thread: thread}
	e := newFeedPages(t, svc)

	content := feedContent(t, getPage(e, "/feed/tasks/T-0002?cursor=abc", true).Body.String())

	assert.Equal(t, "abc", svc.gotCursor)
	assert.Contains(t, content, `href="/feed/tasks/T-0002?cursor=next%2Bcursor"`)
}

func TestFeedThread_Errors(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantStatus   int
		wantLocation string
	}{
		{name: "unknown task", err: leadfeed.ErrTaskNotFound, wantStatus: stdhttp.StatusNotFound},
		{name: "stale page link restarts the thread", err: leadfeed.ErrRevisionChanged,
			wantStatus: stdhttp.StatusFound, wantLocation: "/feed/tasks/T-0002"},
		{name: "broken page link restarts the thread", err: leadfeed.ErrInvalidCursor,
			wantStatus: stdhttp.StatusFound, wantLocation: "/feed/tasks/T-0002"},
		{name: "store failure", err: errors.New("no primary"), wantStatus: stdhttp.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := getPage(newFeedPages(t, &fakeFeedPages{err: tt.err}), "/feed/tasks/T-0002?cursor=old", true)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantLocation, rec.Header().Get("Location"))
			assert.NotContains(t, rec.Body.String(), "no primary")
		})
	}
}

func TestFeedList_StoreFailure(t *testing.T) {
	rec := getPage(newFeedPages(t, &fakeFeedPages{err: errors.New("no primary")}), "/feed", true)

	assert.Equal(t, stdhttp.StatusServiceUnavailable, rec.Code)
	assert.NotContains(t, rec.Body.String(), "no primary")
}
