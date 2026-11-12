package httphandler

import (
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
)

// FeedOptionResponse is one answer choice.
type FeedOptionResponse struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// FeedRecommendationResponse is the lead's own pick.
type FeedRecommendationResponse struct {
	OptionID *string `json:"option_id"`
	Why      string  `json:"why"`
}

// FeedAnswerResponse is the answer shown with a question.
type FeedAnswerResponse struct {
	AnswerID    string   `json:"answer_id"`
	OptionID    *string  `json:"option_id"`
	OptionLabel *string  `json:"option_label"`
	Text        string   `json:"text"`
	AnsweredAt  FeedTime `json:"answered_at"`
	Late        bool     `json:"late"`
	Delivered   bool     `json:"delivered"`
}

// FeedResolvedResponse marks a question the lead closed.
type FeedResolvedResponse struct {
	At        FeedTime `json:"at"`
	Note      string   `json:"note"`
	Elsewhere bool     `json:"elsewhere"`
}

// FeedQuestionResponse is a question with the owner's answer and its state. State is one of
// waiting, answered, delivered, handled; labels are left to the client.
type FeedQuestionResponse struct {
	QID             string                      `json:"qid"`
	RecordID        string                      `json:"record_id"`
	Task            string                      `json:"task"`
	TaskTitle       string                      `json:"task_title"`
	AskedAt         FeedTime                    `json:"asked_at"`
	Body            *string                     `json:"body"`
	Format          string                      `json:"format"`
	Options         []FeedOptionResponse        `json:"options"`
	Recommendation  *FeedRecommendationResponse `json:"recommendation"`
	Scope           string                      `json:"scope"`
	State           string                      `json:"state"`
	ClosedInSession bool                        `json:"closed_in_session"`
	Answer          *FeedAnswerResponse         `json:"answer"`
	Resolved        *FeedResolvedResponse       `json:"resolved"`
}

// FeedQuestionListResponse lists the questions of one snapshot, those waiting first.
type FeedQuestionListResponse struct {
	FeedMetaResponse

	Waiting int                    `json:"waiting"`
	Items   []FeedQuestionResponse `json:"items"`
}

// FeedQuestionDetailResponse is one question.
type FeedQuestionDetailResponse struct {
	FeedMetaResponse

	Question FeedQuestionResponse `json:"question"`
}

// FeedAnswerRequest is the body of an owner answer.
type FeedAnswerRequest struct {
	OptionID *string `json:"option_id" form:"option_id"`
	Text     string  `json:"text"      form:"text"`
}

// FeedAnswerResultResponse reports an accepted answer; Replayed marks a repeat of an earlier request.
type FeedAnswerResultResponse struct {
	QID      string             `json:"qid"`
	Answer   FeedAnswerResponse `json:"answer"`
	Replayed bool               `json:"replayed"`
}

// ListQuestions handles GET /api/v1/feed/questions.
func (h *FeedHandler) ListQuestions(c echo.Context) error {
	list, err := h.questions.List(c.Request().Context())
	if err != nil {
		return h.respondFeedError(c, err)
	}
	items := make([]FeedQuestionResponse, 0, len(list.Items))
	for i := range list.Items {
		items = append(items, toFeedQuestionResponse(&list.Items[i]))
	}
	return httpserver.RespondOK(c, FeedQuestionListResponse{
		FeedMetaResponse: toFeedMeta(list.Meta),
		Waiting:          list.Waiting,
		Items:            items,
	})
}

// GetQuestion handles GET /api/v1/feed/questions/:qid.
func (h *FeedHandler) GetQuestion(c echo.Context) error {
	detail, err := h.questions.Question(c.Request().Context(), c.Param("qid"))
	if err != nil {
		return h.respondFeedError(c, err)
	}
	return httpserver.RespondOK(c, FeedQuestionDetailResponse{
		FeedMetaResponse: toFeedMeta(detail.Meta),
		Question:         toFeedQuestionResponse(&detail.Question),
	})
}

// AnswerQuestion handles POST /api/v1/feed/questions/:qid/answer.
func (h *FeedHandler) AnswerQuestion(c echo.Context) error {
	key, ok, err := requiredIdempotencyKey(c)
	if !ok {
		return err
	}
	var req FeedAnswerRequest
	if bindErr := c.Bind(&req); bindErr != nil {
		return httpserver.RespondErrorWithCode(c, http.StatusUnprocessableEntity, feedCodeInvalidAnswer,
			"the body must be JSON with option_id and text")
	}

	res, err := h.questions.Answer(c.Request().Context(), leadfeed.AnswerCommand{
		QID: c.Param("qid"), OptionID: req.OptionID, Text: req.Text, Key: key,
	})
	if err != nil {
		return h.respondFeedError(c, err)
	}

	body := FeedAnswerResultResponse{
		QID:      res.Answer.QID,
		Answer:   toFeedStoredAnswerResponse(&res.Answer),
		Replayed: res.Outcome == leadfeed.AnswerRepeated,
	}
	if body.Replayed {
		return httpserver.RespondOK(c, body)
	}
	return httpserver.RespondCreated(c, body)
}

// RequireSameOriginForCookie rejects a cookie-authenticated write whose Origin (or Referer) is not the request Host.
// A request with an Authorization header is not driven by a browser's ambient cookie and passes.
func RequireSameOriginForCookie(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		if req.Header.Get(echo.HeaderAuthorization) != "" {
			return next(c)
		}
		origin := req.Header.Get(echo.HeaderOrigin)
		if origin == "" {
			origin = req.Header.Get("Referer")
		}
		if u, err := url.Parse(origin); origin == "" || err != nil || u.Host == "" || u.Host != req.Host {
			return httpserver.RespondErrorWithCode(c, http.StatusForbidden, feedCodeOriginMismatch,
				"the Origin of the request does not match the server")
		}
		return next(c)
	}
}

func toFeedQuestionResponse(v *leadfeed.QuestionView) FeedQuestionResponse {
	rec := &v.Record
	q := rec.Question
	options := make([]FeedOptionResponse, 0, len(q.Options))
	for _, opt := range q.Options {
		options = append(options, FeedOptionResponse{ID: opt.ID, Label: opt.Label})
	}
	out := FeedQuestionResponse{
		QID:             q.QID,
		RecordID:        rec.ID,
		Task:            rec.Task,
		TaskTitle:       v.TaskTitle,
		AskedAt:         feedTime(rec.At),
		Body:            rec.Body,
		Format:          rec.Format,
		Options:         options,
		Scope:           q.Scope,
		State:           string(v.State),
		ClosedInSession: v.ClosedInSession,
	}
	if q.Recommendation != nil {
		out.Recommendation = &FeedRecommendationResponse{OptionID: q.Recommendation.OptionID, Why: q.Recommendation.Why}
	}
	if v.Answer != nil {
		out.Answer = &FeedAnswerResponse{
			AnswerID: v.Answer.AnswerID, OptionID: v.Answer.OptionID, OptionLabel: v.Answer.OptionLabel,
			Text: v.Answer.Text, AnsweredAt: feedTime(v.Answer.AnsweredAt), Late: v.Answer.Late,
			Delivered: v.Answer.Delivered,
		}
	}
	if q.Resolved != nil {
		out.Resolved = &FeedResolvedResponse{
			At: feedTime(q.Resolved.At), Note: q.Resolved.Note, Elsewhere: q.Resolved.Elsewhere,
		}
	}
	return out
}

func toFeedStoredAnswerResponse(a *leadfeed.OwnerAnswer) FeedAnswerResponse {
	return FeedAnswerResponse{
		AnswerID: a.AnswerID, OptionID: a.OptionID, OptionLabel: a.OptionLabel, Text: a.Text,
		AnsweredAt: feedTime(a.AnsweredAt),
	}
}
