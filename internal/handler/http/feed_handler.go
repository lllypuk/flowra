package httphandler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
	"github.com/lllypuk/flowra/internal/infrastructure/httpserver"
)

// Feed API error codes.
const (
	feedCodeTaskNotFound    = "FEED_TASK_NOT_FOUND"
	feedCodeRevisionChanged = "FEED_REVISION_CHANGED"
	feedCodeInvalidCursor   = "INVALID_CURSOR"
	feedCodeInvalidLimit    = "INVALID_LIMIT"
	feedCodeUnavailable     = "FEED_UNAVAILABLE"
)

// FeedService reads the published lead feed; every call answers from one snapshot.
type FeedService interface {
	Status(ctx context.Context) (leadfeed.FeedStatus, error)
	ListTasks(ctx context.Context, query leadfeed.TaskListQuery) (leadfeed.TaskPage, error)
	Task(ctx context.Context, id string) (leadfeed.TaskDetail, error)
	Records(ctx context.Context, taskID string, limit int, cursor string) (leadfeed.RecordPage, error)
}

// FeedHandler serves the read-only lead feed API. It has no write routes by design.
type FeedHandler struct {
	service FeedService
	logger  *slog.Logger
}

// NewFeedHandler creates a feed handler.
func NewFeedHandler(service FeedService, logger *slog.Logger) *FeedHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &FeedHandler{service: service, logger: logger}
}

// RegisterRoutes registers the feed routes; only GET is served.
func (h *FeedHandler) RegisterRoutes(r *httpserver.Router) {
	r.Auth().GET("/feed/status", h.Status)
	r.Auth().GET("/feed/tasks", h.ListTasks)
	r.Auth().GET("/feed/tasks/:id", h.GetTask)
	r.Auth().GET("/feed/tasks/:id/records", h.ListRecords)
}

// FeedMetaResponse names the snapshot a response was read from; revision is empty before the first import.
type FeedMetaResponse struct {
	Revision string     `json:"revision"`
	AsOf     *time.Time `json:"as_of"`
}

// FeedAttemptResponse is the last import attempt.
type FeedAttemptResponse struct {
	At            time.Time `json:"at"`
	Outcome       string    `json:"outcome"`
	Error         string    `json:"error,omitempty"`
	Revision      string    `json:"revision,omitempty"`
	PreviousEpoch string    `json:"previous_epoch,omitempty"`
}

// FeedStatusResponse is the health of the projection.
type FeedStatusResponse struct {
	FeedMetaResponse

	Published   bool                 `json:"published"`
	Healthy     bool                 `json:"healthy"`
	Source      string               `json:"source,omitempty"`
	Epoch       string               `json:"epoch,omitempty"`
	Tasks       int                  `json:"tasks"`
	Records     int                  `json:"records"`
	PublishedAt *time.Time           `json:"published_at"`
	LastAttempt *FeedAttemptResponse `json:"last_attempt"`
}

// FeedSpentResponse is time per waiting band in whole seconds, as computed by the source at as_of.
type FeedSpentResponse struct {
	Queued int64 `json:"queued"`
	Work   int64 `json:"work"`
	Lead   int64 `json:"lead"`
	User   int64 `json:"user"`
}

// FeedTransitionResponse is one status change of a task.
type FeedTransitionResponse struct {
	At      time.Time `json:"at"`
	Status  string    `json:"status"`
	Cause   *string   `json:"cause"`
	Derived bool      `json:"derived"`
}

// FeedTaskResponse is a lead task. Status is the source value; StatusLabel and Group are for display only.
type FeedTaskResponse struct {
	ID          string                   `json:"id"`
	Title       string                   `json:"title"`
	Project     string                   `json:"project"`
	Agent       string                   `json:"agent"`
	CreatedAt   time.Time                `json:"created_at"`
	Status      string                   `json:"status"`
	StatusLabel string                   `json:"status_label"`
	Group       string                   `json:"group"`
	WaitingOn   *string                  `json:"waiting_on"`
	StatusSince time.Time                `json:"status_since"`
	Note        string                   `json:"note"`
	Returns     int                      `json:"returns"`
	Spent       FeedSpentResponse        `json:"spent"`
	Transitions []FeedTransitionResponse `json:"transitions"`
}

// FeedSourceResponse points at where the lead tool found a record.
type FeedSourceResponse struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}

// FeedRecordResponse is one thread entry; exactly one of Body and BodyMissing is set.
type FeedRecordResponse struct {
	ID           string             `json:"id"`
	Task         string             `json:"task"`
	Kind         string             `json:"kind"`
	At           time.Time          `json:"at"`
	Order        int                `json:"order"`
	Actor        string             `json:"actor"`
	ActorDerived bool               `json:"actor_derived"`
	Source       FeedSourceResponse `json:"source"`
	Body         *string            `json:"body"`
	BodyDerived  bool               `json:"body_derived"`
	BodyMissing  *string            `json:"body_missing"`
	Format       string             `json:"format"`
}

// FeedTaskListResponse is a page of tasks.
type FeedTaskListResponse struct {
	FeedMetaResponse

	Items      []FeedTaskResponse `json:"items"`
	NextCursor string             `json:"next_cursor"`
}

// FeedTaskDetailResponse is one task.
type FeedTaskDetailResponse struct {
	FeedMetaResponse

	Task FeedTaskResponse `json:"task"`
}

// FeedRecordListResponse is a page of a task thread.
type FeedRecordListResponse struct {
	FeedMetaResponse

	TaskID     string               `json:"task_id"`
	Items      []FeedRecordResponse `json:"items"`
	NextCursor string               `json:"next_cursor"`
}

// Status handles GET /api/v1/feed/status.
func (h *FeedHandler) Status(c echo.Context) error {
	status, err := h.service.Status(c.Request().Context())
	if err != nil {
		return h.respondFeedError(c, err)
	}

	resp := FeedStatusResponse{Healthy: status.Healthy()}
	if state := status.State; state != nil {
		resp.FeedMetaResponse = FeedMetaResponse{Revision: state.Revision, AsOf: &state.AsOf}
		resp.Published = true
		resp.Source, resp.Epoch = state.Source, state.Epoch
		resp.Tasks, resp.Records = state.Tasks, state.Records
		resp.PublishedAt = &state.PublishedAt
	}
	if attempt := status.LastAttempt; attempt != nil {
		resp.LastAttempt = &FeedAttemptResponse{
			At:            attempt.At,
			Outcome:       attempt.Outcome,
			Error:         attempt.Error,
			Revision:      attempt.Revision,
			PreviousEpoch: attempt.PreviousEpoch,
		}
	}
	return httpserver.RespondOK(c, resp)
}

// ListTasks handles GET /api/v1/feed/tasks.
func (h *FeedHandler) ListTasks(c echo.Context) error {
	limit, ok := parseFeedLimit(c)
	if !ok {
		return respondFeedInvalidLimit(c)
	}

	query := leadfeed.TaskListQuery{Limit: limit, Cursor: c.QueryParam("cursor")}
	if status := c.QueryParam("status"); status != "" {
		query.Filter.Status = &status
	}
	if waitingOn := c.QueryParam("waiting_on"); waitingOn != "" {
		query.Filter.WaitingOn = &waitingOn
	}

	page, err := h.service.ListTasks(c.Request().Context(), query)
	if err != nil {
		return h.respondFeedError(c, err)
	}

	items := make([]FeedTaskResponse, 0, len(page.Tasks))
	for i := range page.Tasks {
		items = append(items, toFeedTaskResponse(&page.Tasks[i]))
	}
	return httpserver.RespondOK(c, FeedTaskListResponse{
		FeedMetaResponse: toFeedMeta(page.Meta),
		Items:            items,
		NextCursor:       page.NextCursor,
	})
}

// GetTask handles GET /api/v1/feed/tasks/:id.
func (h *FeedHandler) GetTask(c echo.Context) error {
	detail, err := h.service.Task(c.Request().Context(), c.Param("id"))
	if err != nil {
		return h.respondFeedError(c, err)
	}
	return httpserver.RespondOK(c, FeedTaskDetailResponse{
		FeedMetaResponse: toFeedMeta(detail.Meta),
		Task:             toFeedTaskResponse(&detail.Task),
	})
}

// ListRecords handles GET /api/v1/feed/tasks/:id/records.
func (h *FeedHandler) ListRecords(c echo.Context) error {
	limit, ok := parseFeedLimit(c)
	if !ok {
		return respondFeedInvalidLimit(c)
	}

	page, err := h.service.Records(c.Request().Context(), c.Param("id"), limit, c.QueryParam("cursor"))
	if err != nil {
		return h.respondFeedError(c, err)
	}

	items := make([]FeedRecordResponse, 0, len(page.Records))
	for i := range page.Records {
		items = append(items, toFeedRecordResponse(&page.Records[i]))
	}
	return httpserver.RespondOK(c, FeedRecordListResponse{
		FeedMetaResponse: toFeedMeta(page.Meta),
		TaskID:           page.Task.ID,
		Items:            items,
		NextCursor:       page.NextCursor,
	})
}

// parseFeedLimit returns 0 for an absent limit, which means the default page size.
func parseFeedLimit(c echo.Context) (int, bool) {
	raw := c.QueryParam("limit")
	if raw == "" {
		return 0, true
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 {
		return 0, false
	}
	return limit, true
}

func respondFeedInvalidLimit(c echo.Context) error {
	return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, feedCodeInvalidLimit,
		"limit must be a positive integer")
}

func (h *FeedHandler) respondFeedError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, leadfeed.ErrTaskNotFound):
		return httpserver.RespondErrorWithCode(c, http.StatusNotFound, feedCodeTaskNotFound, "feed task not found")
	case errors.Is(err, leadfeed.ErrRevisionChanged):
		return httpserver.RespondErrorWithCode(c, http.StatusConflict, feedCodeRevisionChanged,
			"the feed was republished; restart paging without a cursor")
	case errors.Is(err, leadfeed.ErrInvalidCursor):
		return httpserver.RespondErrorWithCode(c, http.StatusBadRequest, feedCodeInvalidCursor, "invalid cursor")
	case errors.Is(err, leadfeed.ErrInvalidLimit):
		return respondFeedInvalidLimit(c)
	default:
		h.logger.ErrorContext(c.Request().Context(), "feed read failed", slog.String("error", err.Error()))
		return httpserver.RespondErrorWithCode(c, http.StatusServiceUnavailable, feedCodeUnavailable,
			"the feed could not be read; retry")
	}
}

func toFeedMeta(meta leadfeed.Meta) FeedMetaResponse {
	if meta.Revision == "" {
		return FeedMetaResponse{}
	}
	return FeedMetaResponse{Revision: meta.Revision, AsOf: &meta.AsOf}
}

func toFeedTaskResponse(task *leadfeed.Task) FeedTaskResponse {
	transitions := make([]FeedTransitionResponse, 0, len(task.Transitions))
	for _, tr := range task.Transitions {
		transitions = append(transitions, FeedTransitionResponse{
			At:      tr.At,
			Status:  tr.Status,
			Cause:   tr.Cause,
			Derived: tr.Derived,
		})
	}
	return FeedTaskResponse{
		ID:          task.ID,
		Title:       task.Title,
		Project:     task.Project,
		Agent:       task.Agent,
		CreatedAt:   task.CreatedAt,
		Status:      task.Status,
		StatusLabel: leadfeed.StatusLabel(task.Status),
		Group:       leadfeed.StatusGroup(task.Status),
		WaitingOn:   task.WaitingOn,
		StatusSince: task.StatusSince,
		Note:        task.Note,
		Returns:     task.Returns,
		Spent: FeedSpentResponse{
			Queued: task.Spent.Queued,
			Work:   task.Spent.Work,
			Lead:   task.Spent.Lead,
			User:   task.Spent.User,
		},
		Transitions: transitions,
	}
}

func toFeedRecordResponse(rec *leadfeed.Record) FeedRecordResponse {
	return FeedRecordResponse{
		ID:           rec.ID,
		Task:         rec.Task,
		Kind:         rec.Kind,
		At:           rec.At,
		Order:        rec.Order,
		Actor:        rec.Actor,
		ActorDerived: rec.ActorDerived,
		Source:       FeedSourceResponse{Kind: rec.Source.Kind, Ref: rec.Source.Ref},
		Body:         rec.Body,
		BodyDerived:  rec.BodyDerived,
		BodyMissing:  rec.BodyMissing,
		Format:       rec.Format,
	}
}
