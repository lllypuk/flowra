package httphandler_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
	httphandler "github.com/lllypuk/flowra/internal/handler/http"
)

type fakeQuestionService struct {
	list   leadfeed.QuestionList
	detail leadfeed.QuestionDetail
	result leadfeed.SaveAnswerResult
	err    error

	gotQID string
	gotCmd leadfeed.AnswerCommand
}

func (f *fakeQuestionService) List(context.Context) (leadfeed.QuestionList, error) {
	return f.list, f.err
}

func (f *fakeQuestionService) Question(_ context.Context, qid string) (leadfeed.QuestionDetail, error) {
	f.gotQID = qid
	return f.detail, f.err
}

func (f *fakeQuestionService) Answer(_ context.Context, cmd leadfeed.AnswerCommand) (leadfeed.SaveAnswerResult, error) {
	f.gotCmd = cmd
	return f.result, f.err
}

func newQuestionServer(svc *fakeQuestionService) *echo.Echo {
	h := httphandler.NewFeedHandler(&fakeFeedService{}, nil).WithQuestions(svc)
	e := echo.New()
	e.GET("/api/v1/feed/questions", h.ListQuestions)
	e.GET("/api/v1/feed/questions/:qid", h.GetQuestion)
	e.POST("/api/v1/feed/questions/:qid/answer", h.AnswerQuestion, httphandler.RequireSameOriginForCookie)
	return e
}

func questionView(state leadfeed.QuestionState) leadfeed.QuestionView {
	body := "Synthetic body <b>x</b>"
	a := "a"
	return leadfeed.QuestionView{
		TaskTitle: "Synthetic task",
		State:     state,
		Record: leadfeed.Record{
			ID: "j:7", Task: "T-0001", Kind: leadfeed.KindQuestion, At: feedAsOf(), Body: &body, Format: "text",
			Question: &leadfeed.Question{
				QID: "Q-0001", FP: "fp",
				Options:        []leadfeed.Option{{ID: "a", Label: "Option A"}},
				Recommendation: &leadfeed.Recommendation{OptionID: &a, Why: "because"},
				Scope:          "synthetic scope",
			},
		},
	}
}

func storedAnswer() leadfeed.OwnerAnswer {
	a, label := "a", "Option A"
	return leadfeed.OwnerAnswer{
		Source: "lead-task", Epoch: "e1", QID: "Q-0001", AnswerID: "ans-1", OptionID: &a, OptionLabel: &label,
		Text: "go", AnsweredAt: feedAsOf(),
	}
}

type answerRequest struct {
	target string
	key    string
	body   string
	header map[string]string
}

func postAnswer(e *echo.Echo, r answerRequest) *httptest.ResponseRecorder {
	req := httptest.NewRequest(stdhttp.MethodPost, r.target, strings.NewReader(r.body))
	req.Host = "127.0.0.1:8080"
	req.Header.Set("Content-Type", "application/json")
	if r.key != "" {
		req.Header.Set("Idempotency-Key", r.key)
	}
	for k, v := range r.header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var env feedEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env), rec.Body.String())
	require.NotNil(t, env.Error, rec.Body.String())
	return env.Error.Code
}

const answerTarget = "/api/v1/feed/questions/Q-0001/answer"

func bearer() map[string]string { return map[string]string{"Authorization": "Bearer t"} }

func TestFeedQuestions_List(t *testing.T) {
	svc := &fakeQuestionService{list: leadfeed.QuestionList{
		Meta: feedMeta(), Waiting: 1, Items: []leadfeed.QuestionView{questionView(leadfeed.QuestionWaiting)},
	}}

	code, env := getFeed(t, newQuestionServer(svc), "/api/v1/feed/questions")

	require.Equal(t, stdhttp.StatusOK, code)
	got := decodeFeed[httphandler.FeedQuestionListResponse](t, env)
	assert.Equal(t, "rev-a", got.Revision)
	assert.Equal(t, 1, got.Waiting)
	require.Len(t, got.Items, 1)
	item := got.Items[0]
	assert.Equal(t, "Q-0001", item.QID)
	assert.Equal(t, "waiting", item.State)
	assert.Equal(t, "Synthetic task", item.TaskTitle)
	assert.Equal(t, "synthetic scope", item.Scope)
	require.NotNil(t, item.Recommendation)
	assert.Equal(t, "because", item.Recommendation.Why)
	assert.Nil(t, item.Answer)
}

func TestFeedQuestions_ListEmptyIsArray(t *testing.T) {
	code, env := getFeed(t, newQuestionServer(&fakeQuestionService{}), "/api/v1/feed/questions")

	require.Equal(t, stdhttp.StatusOK, code)
	assert.JSONEq(t, `[]`, string(decodeFeed[map[string]json.RawMessage](t, env)["items"]))
}

func TestFeedQuestions_Detail(t *testing.T) {
	view := questionView(leadfeed.QuestionAnswered)
	own := storedAnswer()
	view.Answer = &leadfeed.AnswerView{
		AnswerID: own.AnswerID, OptionID: own.OptionID, OptionLabel: own.OptionLabel, Text: own.Text,
		AnsweredAt: own.AnsweredAt,
	}
	view.ClosedInSession = true
	svc := &fakeQuestionService{detail: leadfeed.QuestionDetail{Meta: feedMeta(), Question: view}}

	code, env := getFeed(t, newQuestionServer(svc), "/api/v1/feed/questions/Q-0001")

	require.Equal(t, stdhttp.StatusOK, code)
	assert.Equal(t, "Q-0001", svc.gotQID)
	got := decodeFeed[httphandler.FeedQuestionDetailResponse](t, env).Question
	assert.Equal(t, "answered", got.State)
	assert.True(t, got.ClosedInSession)
	require.NotNil(t, got.Answer)
	assert.Equal(t, "ans-1", got.Answer.AnswerID)
	assert.False(t, got.Answer.Delivered)
}

func TestFeedQuestions_DetailNotFound(t *testing.T) {
	code, env := getFeed(t, newQuestionServer(&fakeQuestionService{err: leadfeed.ErrQuestionNotFound}),
		"/api/v1/feed/questions/Q-9")

	assert.Equal(t, stdhttp.StatusNotFound, code)
	assert.Equal(t, "FEED_QUESTION_NOT_FOUND", env.Error.Code)
}

func TestFeedQuestions_AnswerCreatedAndRepeated(t *testing.T) {
	svc := &fakeQuestionService{
		result: leadfeed.SaveAnswerResult{Outcome: leadfeed.AnswerCreated, Answer: storedAnswer()},
	}
	e := newQuestionServer(svc)
	body := `{"option_id":"a","text":"go"}`

	rec := postAnswer(e, answerRequest{target: answerTarget, key: "k1", body: body, header: bearer()})
	require.Equal(t, stdhttp.StatusCreated, rec.Code, rec.Body.String())
	assert.Equal(
		t,
		leadfeed.AnswerCommand{QID: "Q-0001", OptionID: svc.gotCmd.OptionID, Text: "go", Key: "k1"},
		svc.gotCmd,
	)
	require.NotNil(t, svc.gotCmd.OptionID)
	assert.Equal(t, "a", *svc.gotCmd.OptionID)
	var env feedEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	created := decodeFeed[httphandler.FeedAnswerResultResponse](t, env)
	assert.Equal(t, "Q-0001", created.QID)
	assert.Equal(t, "ans-1", created.Answer.AnswerID)
	assert.False(t, created.Replayed)

	svc.result.Outcome = leadfeed.AnswerRepeated
	rec = postAnswer(e, answerRequest{target: answerTarget, key: "k1", body: body, header: bearer()})
	require.Equal(t, stdhttp.StatusOK, rec.Code)
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	assert.True(t, decodeFeed[httphandler.FeedAnswerResultResponse](t, env).Replayed)
}

func TestFeedQuestions_AnswerErrors(t *testing.T) {
	long := strings.Repeat("k", 201)
	tests := []struct {
		name   string
		key    string
		body   string
		err    error
		status int
		code   string
	}{
		{name: "no key", body: `{"text":"x"}`, status: stdhttp.StatusBadRequest, code: "IDEMPOTENCY_KEY_REQUIRED"},
		{
			name:   "long key",
			key:    long,
			body:   `{"text":"x"}`,
			status: stdhttp.StatusBadRequest,
			code:   "INVALID_IDEMPOTENCY_KEY",
		},
		{name: "broken body", key: "k", body: `{`, status: stdhttp.StatusUnprocessableEntity, code: "INVALID_ANSWER"},
		{
			name:   "invalid answer",
			key:    "k",
			body:   `{}`,
			status: stdhttp.StatusUnprocessableEntity,
			code:   "INVALID_ANSWER",
			err:    fmt.Errorf("%w: an option or a text is required", leadfeed.ErrInvalidAnswer),
		},
		{
			name: "no question", key: "k", body: `{"text":"x"}`, err: leadfeed.ErrQuestionNotFound,
			status: stdhttp.StatusNotFound, code: "FEED_QUESTION_NOT_FOUND",
		},
		{
			name: "already answered", key: "k", body: `{"text":"x"}`, err: leadfeed.ErrAlreadyAnswered,
			status: stdhttp.StatusConflict, code: "QUESTION_ALREADY_ANSWERED",
		},
		{
			name: "closed", key: "k", body: `{"text":"x"}`, err: leadfeed.ErrQuestionClosed,
			status: stdhttp.StatusConflict, code: "QUESTION_CLOSED",
		},
		{
			name: "key reused", key: "k", body: `{"text":"x"}`, err: leadfeed.ErrIdempotencyKeyReused,
			status: stdhttp.StatusConflict, code: "IDEMPOTENCY_KEY_REUSED",
		},
		{
			name: "store down", key: "k", body: `{"text":"x"}`, err: errors.New("mongo down"),
			status: stdhttp.StatusServiceUnavailable, code: "FEED_UNAVAILABLE",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeQuestionService{err: tt.err}

			rec := postAnswer(
				newQuestionServer(svc),
				answerRequest{target: answerTarget, key: tt.key, body: tt.body, header: bearer()},
			)

			assert.Equal(t, tt.status, rec.Code, rec.Body.String())
			assert.Equal(t, tt.code, errorCode(t, rec))
			if tt.status < stdhttp.StatusInternalServerError && tt.err == nil {
				assert.Empty(t, svc.gotCmd.Key, "the service is not reached")
			}
		})
	}
}

func TestFeedQuestions_AnswerOriginCheck(t *testing.T) {
	cookie := map[string]string{"Cookie": "flowra_session=t"}
	with := func(extra map[string]string) map[string]string {
		out := map[string]string{}
		for k, v := range cookie {
			out[k] = v
		}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}
	tests := []struct {
		name   string
		header map[string]string
		status int
	}{
		{
			name:   "cookie, own origin",
			header: with(map[string]string{"Origin": "http://127.0.0.1:8080"}),
			status: stdhttp.StatusCreated,
		},
		{
			name:   "cookie, own referer",
			header: with(map[string]string{"Referer": "http://127.0.0.1:8080/feed/questions/Q-0001"}),
			status: stdhttp.StatusCreated,
		},
		{
			name:   "cookie, foreign origin",
			header: with(map[string]string{"Origin": "http://evil.example"}),
			status: stdhttp.StatusForbidden,
		},
		{
			name:   "cookie, same host other port",
			header: with(map[string]string{"Origin": "http://127.0.0.1:9999"}),
			status: stdhttp.StatusForbidden,
		},
		{
			name:   "cookie, origin null",
			header: with(map[string]string{"Origin": "null"}),
			status: stdhttp.StatusForbidden,
		},
		{name: "cookie, no origin", header: cookie, status: stdhttp.StatusForbidden},
		{name: "bearer, no origin", header: bearer(), status: stdhttp.StatusCreated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeQuestionService{
				result: leadfeed.SaveAnswerResult{Outcome: leadfeed.AnswerCreated, Answer: storedAnswer()},
			}

			rec := postAnswer(
				newQuestionServer(svc),
				answerRequest{target: answerTarget, key: "k", body: `{"text":"x"}`, header: tt.header},
			)

			assert.Equal(t, tt.status, rec.Code, rec.Body.String())
			if tt.status == stdhttp.StatusForbidden {
				assert.Equal(t, "ORIGIN_MISMATCH", errorCode(t, rec))
				assert.Empty(t, svc.gotCmd.Key, "nothing is recorded")
			}
		})
	}
}
