package httphandler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
)

const feedTimeLayout = "02.01.2006 15:04"

// FeedPageService reads the lead feed for the HTML pages; each call answers from one snapshot.
type FeedPageService interface {
	Board(ctx context.Context) (leadfeed.Board, error)
	Thread(ctx context.Context, taskID string, limit int, cursor string) (leadfeed.Thread, error)
}

// FeedTemplateHandler renders the read-only lead feed pages.
// Statuses and times come from the query service; templates only print them.
type FeedTemplateHandler struct {
	renderer  *TemplateRenderer
	service   FeedPageService
	questions FeedQuestionService
	logger    *slog.Logger
	now       func() time.Time
	location  *time.Location
}

// NewFeedTemplateHandler creates the feed page handler.
func NewFeedTemplateHandler(
	renderer *TemplateRenderer,
	service FeedPageService,
	logger *slog.Logger,
) *FeedTemplateHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &FeedTemplateHandler{
		renderer: renderer,
		service:  service,
		logger:   logger,
		now:      time.Now,
		location: time.Local,
	}
}

// WithQuestions enables the question pages and the answer form.
func (h *FeedTemplateHandler) WithQuestions(questions FeedQuestionService) *FeedTemplateHandler {
	h.questions = questions
	return h
}

// SetClock fixes the current time and the display time zone.
func (h *FeedTemplateHandler) SetClock(now func() time.Time, location *time.Location) {
	h.now, h.location = now, location
}

// SetupFeedRoutes registers the feed pages; the only write route is the owner's answer form.
func (h *FeedTemplateHandler) SetupFeedRoutes(e *echo.Echo) {
	e.GET("/feed", h.FeedList, RequireAuth)
	e.GET("/feed/tasks/:id", h.FeedThread, RequireAuth)
	if h.questions != nil {
		e.GET("/feed/questions", h.QuestionList, RequireAuth)
		e.GET("/feed/questions/:qid", h.QuestionPage, RequireAuth)
		e.POST("/feed/questions/:qid/answer", h.AnswerForm, RequireAuth, RequireSameOriginForCookie)
	}
}

// FeedBannerView is the state strip shown on every feed page.
type FeedBannerView struct {
	Published bool
	Revision  string
	AsOf      string
	AsOfAge   string
	// Stale is set when the snapshot is older than feedStaleAfter: the sync is not keeping up.
	Stale       bool
	PublishedAt string
	Healthy     bool
	AttemptAt   string
	Outcome     string
	LastError   string
	// ShowError is set when the last import attempt did not confirm the feed.
	ShowError bool
}

// FeedTaskRowView is one task in the list.
type FeedTaskRowView struct {
	ID          string
	Project     string
	Title       string
	Status      string
	StatusLabel string
	StatusAge   string
	Note        string
}

// FeedGroupView is one non-empty group of the list.
type FeedGroupView struct {
	Key   string
	Title string
	Count int
	Tasks []FeedTaskRowView
}

// FeedListView is the data of the list page.
type FeedListView struct {
	Banner    FeedBannerView
	Groups    []FeedGroupView
	Total     int
	Truncated bool
	// Questions is set when the question pages are on; Waiting counts those without an answer.
	Questions bool
	Waiting   int
}

// FeedTransitionView is one status change in the task header.
type FeedTransitionView struct {
	At          string
	Status      string
	StatusLabel string
	Derived     bool
	Cause       string
}

// FeedRecordView is one thread entry.
type FeedRecordView struct {
	ID           string
	Kind         string
	KindLabel    string
	At           string
	ActorLabel   string
	ActorDerived bool
	Body         string
	HasBody      bool
	BodyDerived  bool
	BodyMissing  string
	SourceKind   string
	// QID is set on a question record and links to its answer page.
	QID string
}

// FeedThreadView is the data of the thread page.
type FeedThreadView struct {
	Banner      FeedBannerView
	Task        FeedTaskRowView
	Agent       string
	WaitingOn   string
	CreatedAt   string
	StatusSince string
	Returns     int
	Spent       []FeedSpentView
	Transitions []FeedTransitionView
	Records     []FeedRecordView
	NextURL     string
}

// FeedSpentView is time spent in one waiting band.
type FeedSpentView struct {
	Label string
	Value string
}

// FeedList handles GET /feed.
func (h *FeedTemplateHandler) FeedList(c echo.Context) error {
	board, err := h.service.Board(c.Request().Context())
	if err != nil {
		return h.unavailable(c, err)
	}

	byGroup := map[string][]FeedTaskRowView{}
	for i := range board.Tasks {
		task := &board.Tasks[i]
		group := leadfeed.StatusGroup(task.Status)
		byGroup[group] = append(byGroup[group], h.taskRow(task, board.AsOf))
	}

	view := FeedListView{
		Banner:    h.banner(board.Status),
		Total:     len(board.Tasks),
		Truncated: board.Truncated,
	}
	if h.questions != nil {
		view.Questions = true
		list, listErr := h.questions.List(c.Request().Context())
		if listErr != nil {
			h.logger.WarnContext(c.Request().Context(), "feed question count failed",
				slog.String("error", listErr.Error()))
			view.Questions = false
		}
		view.Waiting = list.Waiting
	}
	for _, group := range feedGroupOrder() {
		if tasks := byGroup[group.Key]; len(tasks) > 0 {
			view.Groups = append(view.Groups, FeedGroupView{
				Key: group.Key, Title: group.Title, Count: len(tasks), Tasks: tasks,
			})
		}
	}

	return h.render(c, "feed/list.html", "Лента лида", "feed-list-content", view)
}

// FeedThread handles GET /feed/tasks/:id.
func (h *FeedTemplateHandler) FeedThread(c echo.Context) error {
	taskID := c.Param("id")
	thread, err := h.service.Thread(c.Request().Context(), taskID, leadfeed.MaxPageSize, c.QueryParam("cursor"))
	switch {
	case errors.Is(err, leadfeed.ErrTaskNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "feed task not found")
	case errors.Is(err, leadfeed.ErrRevisionChanged), errors.Is(err, leadfeed.ErrInvalidCursor):
		// The page link is stale after a republish: start the thread again.
		return c.Redirect(http.StatusFound, "/feed/tasks/"+url.PathEscape(taskID))
	case err != nil:
		return h.unavailable(c, err)
	}

	task := &thread.Task
	view := FeedThreadView{
		Banner:      h.banner(thread.Status),
		Task:        h.taskRow(task, thread.AsOf),
		Agent:       task.Agent,
		WaitingOn:   feedWaitingOnLabel(task.WaitingOn),
		CreatedAt:   h.formatTime(task.CreatedAt),
		StatusSince: h.formatTime(task.StatusSince),
		Returns:     task.Returns,
		Spent: []FeedSpentView{
			{Label: "очередь", Value: feedSpan(time.Duration(task.Spent.Queued) * time.Second)},
			{Label: "работа", Value: feedSpan(time.Duration(task.Spent.Work) * time.Second)},
			{Label: "ждёт лида", Value: feedSpan(time.Duration(task.Spent.Lead) * time.Second)},
			{Label: "ждёт владельца", Value: feedSpan(time.Duration(task.Spent.User) * time.Second)},
		},
	}
	for _, tr := range task.Transitions {
		item := FeedTransitionView{
			At:          h.formatTime(tr.At),
			Status:      tr.Status,
			StatusLabel: leadfeed.StatusLabel(tr.Status),
			Derived:     tr.Derived,
		}
		if tr.Cause != nil {
			item.Cause = *tr.Cause
		}
		view.Transitions = append(view.Transitions, item)
	}
	for i := range thread.Records {
		view.Records = append(view.Records, h.recordView(&thread.Records[i]))
	}
	if thread.NextCursor != "" {
		view.NextURL = "/feed/tasks/" + url.PathEscape(taskID) + "?cursor=" + url.QueryEscape(thread.NextCursor)
	}

	return h.render(c, "feed/thread.html", task.ID+" — лента лида", "feed-thread-content", view)
}

func (h *FeedTemplateHandler) taskRow(task *leadfeed.Task, asOf time.Time) FeedTaskRowView {
	return FeedTaskRowView{
		ID:          task.ID,
		Project:     task.Project,
		Title:       task.Title,
		Status:      task.Status,
		StatusLabel: leadfeed.StatusLabel(task.Status),
		// Age is measured to the snapshot's as_of: the status is known only as of that moment.
		StatusAge: feedSpan(asOf.Sub(task.StatusSince)),
		Note:      task.Note,
	}
}

func (h *FeedTemplateHandler) recordView(rec *leadfeed.Record) FeedRecordView {
	view := FeedRecordView{
		ID:           rec.ID,
		Kind:         rec.Kind,
		KindLabel:    feedKindLabel(rec.Kind),
		At:           h.formatTime(rec.At),
		ActorLabel:   feedActorLabel(rec.Actor),
		ActorDerived: rec.ActorDerived,
		BodyDerived:  rec.BodyDerived,
		SourceKind:   rec.Source.Kind,
	}
	if rec.Question != nil && h.questions != nil {
		view.QID = rec.Question.QID
	}
	if rec.Body != nil {
		view.Body, view.HasBody = *rec.Body, true
	}
	if rec.BodyMissing != nil {
		view.BodyMissing = *rec.BodyMissing
	}
	return view
}

func (h *FeedTemplateHandler) banner(status leadfeed.FeedStatus) FeedBannerView {
	banner := FeedBannerView{Healthy: status.Healthy()}
	if state := status.State; state != nil {
		banner.Published = true
		banner.Revision = state.Revision
		banner.AsOf = h.formatTime(state.AsOf)
		age := h.now().Sub(state.AsOf)
		banner.AsOfAge = feedSpan(age)
		banner.Stale = age > feedStaleAfter
		banner.PublishedAt = h.formatTime(state.PublishedAt)
	}
	if attempt := status.LastAttempt; attempt != nil {
		banner.AttemptAt = h.formatTime(attempt.At)
		banner.Outcome = attempt.Outcome
		banner.LastError = attempt.Error
		banner.ShowError = !banner.Healthy
	}
	return banner
}

func (h *FeedTemplateHandler) formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(h.location).Format(feedTimeLayout)
}

func (h *FeedTemplateHandler) render(c echo.Context, templateName, title, content string, data any) error {
	return h.renderStatus(c, http.StatusOK, templateName, title, content, data)
}

func (h *FeedTemplateHandler) renderStatus(
	c echo.Context, status int, templateName, title, content string, data any,
) error {
	pageData := PageData{
		Title:           title,
		User:            getUserView(c),
		Data:            data,
		ContentTemplate: content,
	}
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	c.Response().WriteHeader(status)
	return h.renderer.Render(c.Response().Writer, templateName, pageData, c)
}

func (h *FeedTemplateHandler) unavailable(c echo.Context, err error) error {
	h.logger.ErrorContext(c.Request().Context(), "feed page read failed", slog.String("error", err.Error()))
	return echo.NewHTTPError(http.StatusServiceUnavailable, "the feed could not be read; reload the page")
}

type feedGroupDef struct {
	Key   string
	Title string
}

func feedGroupOrder() []feedGroupDef {
	return []feedGroupDef{
		{Key: leadfeed.GroupOwner, Title: "Ждут владельца"},
		{Key: leadfeed.GroupLead, Title: "Ждут лида"},
		{Key: leadfeed.GroupAgent, Title: "У агента"},
		{Key: leadfeed.GroupUnknown, Title: "Неизвестно"},
		{Key: leadfeed.GroupClosed, Title: "Закрытые"},
	}
}

func feedKindLabel(kind string) string {
	switch kind {
	case "letter":
		return "письмо"
	case leadfeed.KindQuestion:
		return "вопрос"
	case "report":
		return "отчёт"
	case "decision":
		return "решение"
	case "observation":
		return "наблюдение"
	default:
		return kind
	}
}

func feedActorLabel(actor string) string {
	switch actor {
	case "lead":
		return "лид"
	case "agent":
		return "агент"
	case "owner":
		return "владелец"
	default:
		return actor
	}
}

func feedWaitingOnLabel(waitingOn *string) string {
	if waitingOn == nil {
		return "никого"
	}
	return feedActorLabel(*waitingOn)
}

const feedStaleAfter = 5 * time.Minute

// feedSpan prints a duration the way the lead's board does.
func feedSpan(d time.Duration) string {
	const (
		minute = 60
		hour   = 60 * minute
		day    = 24 * hour
	)
	sec := int64(d.Round(time.Second) / time.Second)
	if sec < 0 {
		sec = 0
	}
	switch {
	case sec < minute:
		return fmt.Sprintf("%d с", sec)
	case sec < hour:
		return fmt.Sprintf("%d мин", sec/minute)
	case sec < day:
		return fmt.Sprintf("%d ч %02d мин", sec/hour, sec%hour/minute)
	default:
		return fmt.Sprintf("%d д %d ч", sec/day, sec%day/hour)
	}
}
