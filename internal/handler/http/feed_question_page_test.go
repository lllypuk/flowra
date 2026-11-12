package httphandler_test

import (
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
)

func newQuestionPages(t *testing.T, svc *fakeQuestionService, pages httphandler.FeedPageService) *echo.Echo {
	t.Helper()
	return newFeedPagesWithQuestions(t, pages, svc, pageNow)
}

func postForm(e *echo.Echo, form url.Values, mutate func(*stdhttp.Request)) *httptest.ResponseRecorder {
	req := httptest.NewRequest(stdhttp.MethodPost, "/feed/questions/Q-0001/answer", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.AddCookie(&stdhttp.Cookie{Name: sessionCookie, Value: ownerSecret})
	req.Header.Set(echo.HeaderOrigin, "http://"+req.Host)
	if mutate != nil {
		mutate(req)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func detailOf(view leadfeed.QuestionView, meta leadfeed.Meta) leadfeed.QuestionDetail {
	return leadfeed.QuestionDetail{Meta: meta, Question: view}
}

func pageMeta() leadfeed.Meta { return leadfeed.Meta{Revision: "rev-a", AsOf: pageAsOf()} }

func TestQuestionPages_RequireSession(t *testing.T) {
	e := newQuestionPages(t, &fakeQuestionService{detail: detailOf(questionView(leadfeed.QuestionWaiting), pageMeta())},
		&fakeFeedPages{})

	for _, target := range []string{"/feed/questions", "/feed/questions/Q-0001"} {
		rec := getPage(e, target, false)
		assert.Equal(t, stdhttp.StatusFound, rec.Code, target)
		assert.Equal(t, "/login", rec.Header().Get("Location"), target)
	}

	rec := postForm(e, url.Values{"idempotency_key": {"k"}, "text": {"x"}},
		func(r *stdhttp.Request) { r.Header.Del("Cookie") })
	assert.Equal(t, stdhttp.StatusFound, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestQuestionPage_ShowsFormForWaitingQuestion(t *testing.T) {
	svc := &fakeQuestionService{detail: detailOf(questionView(leadfeed.QuestionWaiting), pageMeta())}

	rec := getPage(newQuestionPages(t, svc, &fakeFeedPages{}), "/feed/questions/Q-0001", true)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	assert.Equal(t, "Q-0001", svc.gotQID)
	content := feedContent(t, rec.Body.String())
	assert.Contains(t, content, `method="post" action="/feed/questions/Q-0001/answer"`)
	assert.Contains(t, content, `type="radio" name="option_id" value="a"`)
	assert.Contains(t, content, "рекомендация лида")
	assert.Contains(t, content, "because")
	assert.Contains(t, content, "synthetic scope")
	assert.Contains(t, content, `<textarea name="text"`)
	assert.Regexp(t, `name="idempotency_key" value="[0-9a-f-]{36}"`, content)
	assert.Contains(t, content, "ждёт ответа")
	assert.NotContains(t, content, "<script")
}

func TestQuestionPage_NewKeyOnEveryShow(t *testing.T) {
	e := newQuestionPages(t, &fakeQuestionService{detail: detailOf(questionView(leadfeed.QuestionWaiting), pageMeta())},
		&fakeFeedPages{})
	key := regexp.MustCompile(`name="idempotency_key" value="([^"]+)"`)

	first := key.FindStringSubmatch(getPage(e, "/feed/questions/Q-0001", true).Body.String())
	second := key.FindStringSubmatch(getPage(e, "/feed/questions/Q-0001", true).Body.String())

	require.Len(t, first, 2)
	require.Len(t, second, 2)
	assert.NotEqual(t, first[1], second[1])
}

func TestQuestionPage_EscapesQuestionAndAnswer(t *testing.T) {
	view := questionView(leadfeed.QuestionAnswered)
	body := "ask <script>alert(1)</script> & co"
	view.Record.Body = &body
	view.Record.Question.Scope = "scope <i>x</i>"
	label := "Opt <b>A</b>"
	view.Answer = &leadfeed.AnswerView{AnswerID: "ans-1", OptionLabel: &label, Text: "reply <img src=x onerror=1>"}
	svc := &fakeQuestionService{detail: detailOf(view, pageMeta())}

	content := feedContent(t, getPage(newQuestionPages(t, svc, &fakeFeedPages{}), "/feed/questions/Q-0001", true).
		Body.String())

	for _, raw := range []string{"<script>alert(1)</script>", "<i>x</i>", "<b>A</b>", "<img src=x"} {
		assert.NotContains(t, content, raw)
	}
	assert.Contains(t, content, "ask &lt;script&gt;alert(1)&lt;/script&gt; &amp; co")
	assert.Contains(t, content, "reply &lt;img src=x onerror=1&gt;")
	assert.Contains(t, content, "Opt &lt;b&gt;A&lt;/b&gt;")
}

func TestQuestionPage_StateReplacesForm(t *testing.T) {
	answer := &leadfeed.AnswerView{AnswerID: "ans-1", Text: "go"}
	resolved := &leadfeed.Resolved{At: pageAsOf(), Note: "done in session"}
	tests := []struct {
		name   string
		mutate func(*leadfeed.QuestionView)
		state  leadfeed.QuestionState
		want   []string
	}{
		{name: "answered", state: leadfeed.QuestionAnswered,
			mutate: func(v *leadfeed.QuestionView) { v.Answer = answer },
			want:   []string{"отвечен, лид ещё не получил", "data-feed-answer"}},
		{name: "answered after close", state: leadfeed.QuestionAnswered,
			mutate: func(v *leadfeed.QuestionView) {
				v.Answer, v.ClosedInSession, v.Record.Question.Resolved = answer, true, resolved
			},
			want: []string{"вопрос закрыт в сессии", "data-feed-closed-in-session", "done in session"}},
		{name: "delivered", state: leadfeed.QuestionDelivered,
			mutate: func(v *leadfeed.QuestionView) { v.Answer = answer },
			want:   []string{"передан лиду", "data-feed-answer"}},
		{name: "handled", state: leadfeed.QuestionHandled,
			mutate: func(v *leadfeed.QuestionView) { v.Answer, v.Record.Question.Resolved = answer, resolved },
			want:   []string{"обработан", "data-feed-resolved"}},
		{name: "closed without answer", state: leadfeed.QuestionHandled,
			mutate: func(v *leadfeed.QuestionView) { v.Record.Question.Resolved = resolved },
			want:   []string{"обработан", "data-feed-resolved"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := questionView(tt.state)
			tt.mutate(&view)
			svc := &fakeQuestionService{detail: detailOf(view, pageMeta())}

			content := feedContent(t,
				getPage(newQuestionPages(t, svc, &fakeFeedPages{}), "/feed/questions/Q-0001", true).Body.String())

			assert.NotContains(t, content, "<form")
			assert.NotContains(t, content, "<button")
			for _, want := range tt.want {
				assert.Contains(t, content, want)
			}
		})
	}
}

func TestQuestionPage_Errors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "unknown question", err: leadfeed.ErrQuestionNotFound, want: stdhttp.StatusNotFound},
		{name: "store failure", err: errors.New("no primary"), want: stdhttp.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newQuestionPages(t, &fakeQuestionService{err: tt.err}, &fakeFeedPages{})

			rec := getPage(e, "/feed/questions/Q-0001", true)

			assert.Equal(t, tt.want, rec.Code)
			assert.NotContains(t, rec.Body.String(), "no primary")
		})
	}
}

func TestQuestionList_WaitingFirstWithCount(t *testing.T) {
	waiting := questionView(leadfeed.QuestionWaiting)
	long := strings.Repeat("я", 300) + "\nsecond line"
	waiting.Record.Body = &long
	other := questionView(leadfeed.QuestionDelivered)
	other.Record.Question = &leadfeed.Question{QID: "Q-0002", Scope: "s"}
	svc := &fakeQuestionService{list: leadfeed.QuestionList{
		Meta: pageMeta(), Waiting: 1, Items: []leadfeed.QuestionView{waiting, other},
	}}

	e := newQuestionPages(t, svc, &fakeFeedPages{})
	content := feedContent(t, getPage(e, "/feed/questions", true).Body.String())

	assert.Contains(t, content, `data-feed-waiting>1<`)
	assert.Contains(t, content, `href="/feed/questions/Q-0001"`)
	assert.Less(t, strings.Index(content, "Q-0001"), strings.Index(content, "Q-0002"))
	assert.NotContains(t, content, "second line")
	assert.Contains(t, content, "…")
	assert.NotContains(t, content, strings.Repeat("я", 300))
}

func TestFeedList_ShowsWaitingQuestionCount(t *testing.T) {
	svc := &fakeQuestionService{list: leadfeed.QuestionList{Meta: pageMeta(), Waiting: 3}}

	content := feedContent(t, getPage(newQuestionPages(t, svc, &fakeFeedPages{board: feedBoard()}), "/feed", true).
		Body.String())

	assert.Contains(t, content, `href="/feed/questions"`)
	assert.Contains(t, content, `data-feed-waiting>3<`)
}

func TestFeedList_QuestionFailureDoesNotBreakTheList(t *testing.T) {
	svc := &fakeQuestionService{err: errors.New("no primary")}

	rec := getPage(newQuestionPages(t, svc, &fakeFeedPages{board: feedBoard()}), "/feed", true)

	require.Equal(t, stdhttp.StatusOK, rec.Code)
	assert.NotContains(t, feedContent(t, rec.Body.String()), "data-feed-questions-link")
}

func TestFeedThread_QuestionRecordLinksToItsPage(t *testing.T) {
	thread := feedThread()
	body := "ask"
	thread.Records = append(thread.Records, leadfeed.Record{
		ID: "j:9", Kind: leadfeed.KindQuestion, At: pageAsOf(), Actor: "lead", Source: leadfeed.Source{Kind: "journal"},
		Body: &body, Question: &leadfeed.Question{QID: "Q-0007"},
	})
	e := newQuestionPages(t, &fakeQuestionService{}, &fakeFeedPages{thread: thread})

	content := feedContent(t, getPage(e, "/feed/tasks/T-0002", true).Body.String())

	assert.Contains(t, content, `href="/feed/questions/Q-0007"`)
	assert.Contains(t, content, "data-feed-question-card")
	assert.NotContains(t, content, "<form")
}

func TestAnswerForm_AcceptsAndRedirectsToTheQuestion(t *testing.T) {
	svc := &fakeQuestionService{result: leadfeed.SaveAnswerResult{Outcome: leadfeed.AnswerCreated}}
	e := newQuestionPages(t, svc, &fakeFeedPages{})

	rec := postForm(e,
		url.Values{"idempotency_key": {"key-1"}, "option_id": {"a"}, "text": {"go"}}, nil)

	assert.Equal(t, stdhttp.StatusSeeOther, rec.Code)
	assert.Equal(t, "/feed/questions/Q-0001", rec.Header().Get("Location"))
	assert.Equal(t, "Q-0001", svc.gotCmd.QID)
	assert.Equal(t, "key-1", svc.gotCmd.Key)
	assert.Equal(t, "go", svc.gotCmd.Text)
	require.NotNil(t, svc.gotCmd.OptionID)
	assert.Equal(t, "a", *svc.gotCmd.OptionID)
}

func TestAnswerForm_TextOnlyHasNoOption(t *testing.T) {
	svc := &fakeQuestionService{}
	e := newQuestionPages(t, svc, &fakeFeedPages{})

	rec := postForm(e, url.Values{"idempotency_key": {"k"}, "text": {"go"}}, nil)

	assert.Equal(t, stdhttp.StatusSeeOther, rec.Code)
	assert.Nil(t, svc.gotCmd.OptionID)
}

func TestAnswerForm_ResubmitWithSameKeyRedirectsAgain(t *testing.T) {
	svc := &fakeQuestionService{result: leadfeed.SaveAnswerResult{Outcome: leadfeed.AnswerRepeated}}
	e := newQuestionPages(t, svc, &fakeFeedPages{})

	for range 2 {
		rec := postForm(e,
			url.Values{"idempotency_key": {"key-1"}, "text": {"go"}}, nil)

		assert.Equal(t, stdhttp.StatusSeeOther, rec.Code)
		assert.Equal(t, "key-1", svc.gotCmd.Key)
	}
}

func TestAnswerForm_ErrorsOfTheService(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantStatus   int
		wantLocation string
	}{
		{name: "already answered shows the state", err: leadfeed.ErrAlreadyAnswered,
			wantStatus: stdhttp.StatusSeeOther, wantLocation: "/feed/questions/Q-0001"},
		{name: "closed shows the state", err: leadfeed.ErrQuestionClosed,
			wantStatus: stdhttp.StatusSeeOther, wantLocation: "/feed/questions/Q-0001"},
		{name: "unknown question", err: leadfeed.ErrQuestionNotFound, wantStatus: stdhttp.StatusNotFound},
		{name: "key of another answer", err: leadfeed.ErrIdempotencyKeyReused, wantStatus: stdhttp.StatusConflict},
		{name: "store failure", err: errors.New("no primary"), wantStatus: stdhttp.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newQuestionPages(t, &fakeQuestionService{err: tt.err}, &fakeFeedPages{})

			rec := postForm(e,
				url.Values{"idempotency_key": {"k"}, "text": {"go"}}, nil)

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantLocation, rec.Header().Get("Location"))
			assert.NotContains(t, rec.Body.String(), "no primary")
		})
	}
}

func TestAnswerForm_InvalidAnswerShowsTheFormAgainWithTheText(t *testing.T) {
	svc := &fakeQuestionService{
		detail:    detailOf(questionView(leadfeed.QuestionWaiting), pageMeta()),
		answerErr: leadfeed.ErrInvalidAnswer,
	}
	e := newQuestionPages(t, svc, &fakeFeedPages{})

	rec := postForm(e,
		url.Values{"idempotency_key": {"key-9"}, "option_id": {"a"}, "text": {"typed <b>text</b>"}}, nil)

	require.Equal(t, stdhttp.StatusUnprocessableEntity, rec.Code)
	content := feedContent(t, rec.Body.String())
	assert.Contains(t, content, "data-feed-form-error")
	assert.Contains(t, content, `value="key-9"`)
	assert.Contains(t, content, "typed &lt;b&gt;text&lt;/b&gt;")
	assert.Contains(t, content, `value="a" checked`)
}

func TestAnswerForm_RequiresAKey(t *testing.T) {
	svc := &fakeQuestionService{}
	e := newQuestionPages(t, svc, &fakeFeedPages{})

	rec := postForm(e, url.Values{"text": {"go"}}, nil)

	assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)
	assert.Empty(t, svc.gotCmd.QID, "the service is not called")
}

func TestAnswerForm_OriginIsChecked(t *testing.T) {
	form := url.Values{"idempotency_key": {"k"}, "text": {"go"}}
	tests := []struct {
		name   string
		mutate func(*stdhttp.Request)
		want   int
	}{
		{name: "same origin", mutate: nil, want: stdhttp.StatusSeeOther},
		{name: "foreign origin", want: stdhttp.StatusForbidden,
			mutate: func(r *stdhttp.Request) { r.Header.Set(echo.HeaderOrigin, "http://evil.example") }},
		{name: "no origin and no referer", want: stdhttp.StatusForbidden,
			mutate: func(r *stdhttp.Request) { r.Header.Del(echo.HeaderOrigin) }},
		{name: "same host referer", want: stdhttp.StatusSeeOther,
			mutate: func(r *stdhttp.Request) {
				r.Header.Del(echo.HeaderOrigin)
				r.Header.Set("Referer", "http://"+r.Host+"/feed/questions/Q-0001")
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeQuestionService{}
			e := newQuestionPages(t, svc, &fakeFeedPages{})

			rec := postForm(e, form, tt.mutate)

			assert.Equal(t, tt.want, rec.Code)
			if tt.want == stdhttp.StatusForbidden {
				assert.Empty(t, svc.gotCmd.QID, "a rejected write never reaches the service")
			}
		})
	}
}

func TestQuestionRoutes_AbsentWithoutQuestions(t *testing.T) {
	e := newFeedPages(t, &fakeFeedPages{board: feedBoard()})

	assert.Equal(t, stdhttp.StatusNotFound, getPage(e, "/feed/questions", true).Code)
}
