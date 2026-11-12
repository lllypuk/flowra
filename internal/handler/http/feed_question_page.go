package httphandler

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
)

const (
	questionSnippetRunes = 160
	invalidAnswerMessage = "Выберите вариант или напишите текст не длиннее 4000 символов."
)

// FeedSnapshotView is the time of the snapshot a question page was built from.
type FeedSnapshotView struct {
	Published bool
	AsOf      string
	AsOfAge   string
	Stale     bool
}

// FeedOptionView is one answer choice on the form.
type FeedOptionView struct {
	ID          string
	Label       string
	Recommended bool
	Selected    bool
}

// FeedAnswerPageView is the answer shown instead of the form.
type FeedAnswerPageView struct {
	Choice     string
	Text       string
	AnsweredAt string
	Late       bool
}

// FeedQuestionRowView is one line of the question list.
type FeedQuestionRowView struct {
	QID        string
	Task       string
	TaskTitle  string
	AskedAt    string
	Snippet    string
	State      string
	StateLabel string
}

// FeedQuestionsView is the data of the question list page.
type FeedQuestionsView struct {
	Snapshot FeedSnapshotView
	Waiting  int
	Items    []FeedQuestionRowView
}

// FeedQuestionPageView is the data of the question page.
type FeedQuestionPageView struct {
	Snapshot          FeedSnapshotView
	QID               string
	Task              string
	TaskTitle         string
	AskedAt           string
	Body              string
	Options           []FeedOptionView
	HasRecommendation bool
	RecommendedLabel  string
	RecommendedWhy    string
	Scope             string
	State             string
	StateLabel        string
	ClosedInSession   bool
	ResolvedAt        string
	ResolvedNote      string
	ResolvedElsewhere bool
	Answer            *FeedAnswerPageView
	CanAnswer         bool
	Key               string
	Text              string
	Error             string
}

// QuestionList handles GET /feed/questions.
func (h *FeedTemplateHandler) QuestionList(c echo.Context) error {
	list, err := h.questions.List(c.Request().Context())
	if err != nil {
		return h.unavailable(c, err)
	}
	view := FeedQuestionsView{Snapshot: h.snapshotView(list.Meta), Waiting: list.Waiting}
	for i := range list.Items {
		item := &list.Items[i]
		row := FeedQuestionRowView{
			QID:        item.Record.Question.QID,
			Task:       item.Record.Task,
			TaskTitle:  item.TaskTitle,
			AskedAt:    h.formatTime(item.Record.At),
			State:      string(item.State),
			StateLabel: feedQuestionStateLabel(item),
		}
		if item.Record.Body != nil {
			row.Snippet = snippet(*item.Record.Body, questionSnippetRunes)
		}
		view.Items = append(view.Items, row)
	}
	return h.render(c, "feed/questions.html", "Вопросы лида", "feed-questions-content", view)
}

// QuestionPage handles GET /feed/questions/:qid.
func (h *FeedTemplateHandler) QuestionPage(c echo.Context) error {
	detail, err := h.questions.Question(c.Request().Context(), c.Param("qid"))
	if err != nil {
		return h.questionError(c, err)
	}
	view := h.questionView(&detail)
	view.Key = uuid.NewString()
	return h.render(c, "feed/question.html", detail.Question.Record.Question.QID+" — вопрос лида",
		"feed-question-content", view)
}

// AnswerForm handles POST /feed/questions/:qid/answer; a rejected answer is shown again with the typed text.
func (h *FeedTemplateHandler) AnswerForm(c echo.Context) error {
	qid := c.Param("qid")
	key := c.FormValue("idempotency_key")
	if key == "" || len(key) > maxIdempotencyKeyLength {
		return echo.NewHTTPError(http.StatusBadRequest, "the form has no valid idempotency key; reload the page")
	}
	cmd := leadfeed.AnswerCommand{QID: qid, Text: c.FormValue("text"), Key: key}
	if option := c.FormValue("option_id"); option != "" {
		cmd.OptionID = &option
	}

	_, err := h.questions.Answer(c.Request().Context(), cmd)
	switch {
	case err == nil,
		errors.Is(err, leadfeed.ErrAlreadyAnswered),
		errors.Is(err, leadfeed.ErrQuestionClosed):
		// The page shows the state the question is in now.
		return c.Redirect(http.StatusSeeOther, "/feed/questions/"+url.PathEscape(qid))
	case errors.Is(err, leadfeed.ErrInvalidAnswer):
		return h.rejectedAnswer(c, qid, cmd, invalidAnswerMessage)
	case errors.Is(err, leadfeed.ErrIdempotencyKeyReused):
		return echo.NewHTTPError(http.StatusConflict, "the form was already sent with another answer; reload the page")
	default:
		return h.questionError(c, err)
	}
}

func (h *FeedTemplateHandler) rejectedAnswer(
	c echo.Context, qid string, cmd leadfeed.AnswerCommand, message string,
) error {
	detail, err := h.questions.Question(c.Request().Context(), qid)
	if err != nil {
		return h.questionError(c, err)
	}
	view := h.questionView(&detail)
	view.Key, view.Text, view.Error = cmd.Key, cmd.Text, message
	for i := range view.Options {
		view.Options[i].Selected = cmd.OptionID != nil && view.Options[i].ID == *cmd.OptionID
	}
	return h.renderStatus(c, http.StatusUnprocessableEntity, "feed/question.html",
		qid+" — вопрос лида", "feed-question-content", view)
}

func (h *FeedTemplateHandler) questionError(c echo.Context, err error) error {
	if errors.Is(err, leadfeed.ErrQuestionNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, "feed question not found")
	}
	return h.unavailable(c, err)
}

func (h *FeedTemplateHandler) questionView(detail *leadfeed.QuestionDetail) FeedQuestionPageView {
	item := &detail.Question
	rec := &item.Record
	q := rec.Question
	view := FeedQuestionPageView{
		Snapshot:        h.snapshotView(detail.Meta),
		QID:             q.QID,
		Task:            rec.Task,
		TaskTitle:       item.TaskTitle,
		AskedAt:         h.formatTime(rec.At),
		Scope:           q.Scope,
		State:           string(item.State),
		StateLabel:      feedQuestionStateLabel(item),
		ClosedInSession: item.ClosedInSession,
		CanAnswer:       item.State == leadfeed.QuestionWaiting,
	}
	if rec.Body != nil {
		view.Body = *rec.Body
	}
	var recommended *string
	if q.Recommendation != nil {
		view.HasRecommendation = true
		view.RecommendedWhy = q.Recommendation.Why
		recommended = q.Recommendation.OptionID
	}
	for _, opt := range q.Options {
		isRecommended := recommended != nil && *recommended == opt.ID
		if isRecommended {
			view.RecommendedLabel = opt.Label
		}
		view.Options = append(view.Options, FeedOptionView{ID: opt.ID, Label: opt.Label, Recommended: isRecommended})
	}
	if q.Resolved != nil {
		view.ResolvedAt = h.formatTime(q.Resolved.At)
		view.ResolvedNote = q.Resolved.Note
		view.ResolvedElsewhere = q.Resolved.Elsewhere
	}
	if a := item.Answer; a != nil {
		choice := ""
		switch {
		case a.OptionLabel != nil:
			choice = *a.OptionLabel
		case a.OptionID != nil:
			choice = *a.OptionID
		}
		view.Answer = &FeedAnswerPageView{
			Choice: choice, Text: a.Text, AnsweredAt: h.formatTime(a.AnsweredAt), Late: a.Late,
		}
	}
	return view
}

func (h *FeedTemplateHandler) snapshotView(meta leadfeed.Meta) FeedSnapshotView {
	if meta.Revision == "" {
		return FeedSnapshotView{}
	}
	age := h.now().Sub(meta.AsOf)
	return FeedSnapshotView{
		Published: true, AsOf: h.formatTime(meta.AsOf), AsOfAge: feedSpan(age), Stale: age > feedStaleAfter,
	}
}

func feedQuestionStateLabel(item *leadfeed.QuestionView) string {
	switch item.State {
	case leadfeed.QuestionWaiting:
		return "ждёт ответа"
	case leadfeed.QuestionAnswered:
		if item.ClosedInSession {
			return "отвечен, лид ещё не получил; вопрос закрыт в сессии"
		}
		return "отвечен, лид ещё не получил"
	case leadfeed.QuestionDelivered:
		return "передан лиду"
	case leadfeed.QuestionHandled:
		return "обработан"
	default:
		return string(item.State)
	}
}

// snippet is the first line of text cut to limit runes.
func snippet(text string, limit int) string {
	line, _, multi := strings.Cut(strings.TrimSpace(text), "\n")
	if utf8.RuneCountInString(line) <= limit {
		if multi {
			return line + "…"
		}
		return line
	}
	return string([]rune(line)[:limit]) + "…"
}
