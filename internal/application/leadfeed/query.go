package leadfeed

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Page size bounds for every list.
const (
	DefaultPageSize = 50
	MaxPageSize     = 200
)

const cursorVersion = "v1"

// Query errors.
var (
	// ErrRevisionChanged means the feed was republished after the cursor was issued; paging restarts.
	ErrRevisionChanged = errors.New("feed revision changed since the cursor was issued")
	ErrInvalidCursor   = errors.New("invalid feed cursor")
	ErrInvalidLimit    = errors.New("invalid page limit")
)

// Task groups of the feed list, by who is expected to act.
const (
	GroupOwner   = "owner"
	GroupLead    = "lead"
	GroupAgent   = "agent"
	GroupClosed  = "closed"
	GroupUnknown = "unknown"
)

// statusDisplay maps a source status to its label and list group. The source status itself is never rewritten.
//
//nolint:gochecknoglobals // immutable lookup table
var statusDisplay = map[string]struct{ label, group string }{
	"queued":    {"в очереди", GroupAgent},
	"work":      {"в работе", GroupAgent},
	"question":  {"вопрос к лиду", GroupLead},
	"review":    {"на проверке у лида", GroupLead},
	"wait_user": {"ждёт тебя", GroupOwner},
	"done":      {"принята", GroupClosed},
	"cancelled": {"отменена", GroupClosed},
}

// StatusLabel returns the display label of a source status; an unknown status is shown as is.
func StatusLabel(status string) string {
	if display, ok := statusDisplay[status]; ok {
		return display.label
	}
	return status
}

// StatusGroup returns the list group of a source status.
func StatusGroup(status string) string {
	if display, ok := statusDisplay[status]; ok {
		return display.group
	}
	return GroupUnknown
}

// Reader is what the query service needs from the feed store.
type Reader interface {
	// Read runs fn against one snapshot of the feed; fn may run more than once.
	Read(ctx context.Context, fn func(ctx context.Context, view View) error) error
}

// Meta ties a response to the snapshot it was read from. Revision is empty when nothing is published.
type Meta struct {
	Revision string
	AsOf     time.Time
}

// FeedStatus is the health of the projection.
type FeedStatus struct {
	State       *State
	LastAttempt *Attempt
}

// Healthy reports whether the last import attempt, if any, left the feed confirmed.
func (s FeedStatus) Healthy() bool {
	if s.LastAttempt == nil {
		return s.State != nil
	}
	return s.LastAttempt.Outcome == string(OutcomePublished) || s.LastAttempt.Outcome == string(OutcomeRefreshed)
}

// TaskListQuery selects a page of tasks.
type TaskListQuery struct {
	Filter TaskFilter
	Limit  int
	Cursor string
}

// TaskPage is one page of tasks from a single snapshot.
type TaskPage struct {
	Meta

	Tasks      []Task
	NextCursor string
}

// TaskDetail is one task from a single snapshot.
type TaskDetail struct {
	Meta

	Task Task
}

// RecordPage is one page of a task thread from a single snapshot.
type RecordPage struct {
	Meta

	Task       Task
	Records    []Record
	NextCursor string
}

// Thread is a page of a task thread together with the feed status, for the thread page.
type Thread struct {
	RecordPage

	Status FeedStatus
}

// Board is the whole feed, for the list page.
type Board struct {
	Meta

	Status FeedStatus
	Tasks  []Task
	// Truncated is set when the feed holds more tasks than one board shows.
	Truncated bool
}

// boardLimit caps the list page; the feed of one person is far below it.
const boardLimit = 1000

// QueryService answers reads of the published feed. Every answer comes from one snapshot.
type QueryService struct {
	reader Reader
}

// NewQueryService creates a query service.
func NewQueryService(reader Reader) *QueryService {
	return &QueryService{reader: reader}
}

// Status returns the published snapshot and the last import attempt.
func (s *QueryService) Status(ctx context.Context) (FeedStatus, error) {
	var out FeedStatus
	err := s.reader.Read(ctx, func(ctx context.Context, view View) error {
		out = FeedStatus{}
		var err error
		if out.State, err = view.State(ctx); err != nil {
			return err
		}
		out.LastAttempt, err = view.LastAttempt(ctx)
		return err
	})
	return out, err
}

// ListTasks returns a page of tasks, newest first.
func (s *QueryService) ListTasks(ctx context.Context, query TaskListQuery) (TaskPage, error) {
	limit, limitErr := pageLimit(query.Limit)
	if limitErr != nil {
		return TaskPage{}, limitErr
	}

	var out TaskPage
	err := s.reader.Read(ctx, func(ctx context.Context, view View) error {
		out = TaskPage{}
		meta, offset, err := startPage(ctx, view, query.Cursor)
		if err != nil {
			return err
		}
		out.Meta = meta

		tasks, err := view.Tasks(ctx, query.Filter, offset, limit+1)
		if err != nil {
			return err
		}
		if len(tasks) > limit {
			tasks = tasks[:limit]
			out.NextCursor = encodeCursor(meta.Revision, offset+limit)
		}
		out.Tasks = tasks
		return nil
	})
	return out, err
}

// Task returns one task with its transitions.
func (s *QueryService) Task(ctx context.Context, id string) (TaskDetail, error) {
	var out TaskDetail
	err := s.reader.Read(ctx, func(ctx context.Context, view View) error {
		out = TaskDetail{}
		meta, _, err := startPage(ctx, view, "")
		if err != nil {
			return err
		}
		task, err := view.Task(ctx, id)
		if err != nil {
			return err
		}
		out = TaskDetail{Meta: meta, Task: *task}
		return nil
	})
	return out, err
}

// Records returns a page of a task thread in source order.
func (s *QueryService) Records(ctx context.Context, taskID string, limit int, cursor string) (RecordPage, error) {
	limit, limitErr := pageLimit(limit)
	if limitErr != nil {
		return RecordPage{}, limitErr
	}

	var out RecordPage
	err := s.reader.Read(ctx, func(ctx context.Context, view View) error {
		out = RecordPage{}
		meta, offset, err := startPage(ctx, view, cursor)
		if err != nil {
			return err
		}
		task, err := view.Task(ctx, taskID)
		if err != nil {
			return err
		}
		out.Meta, out.Task = meta, *task

		records, err := view.Records(ctx, taskID, offset, limit+1)
		if err != nil {
			return err
		}
		if len(records) > limit {
			records = records[:limit]
			out.NextCursor = encodeCursor(meta.Revision, offset+limit)
		}
		out.Records = records
		return nil
	})
	return out, err
}

// Thread returns a page of a task thread and the feed status, from one snapshot.
func (s *QueryService) Thread(ctx context.Context, taskID string, limit int, cursor string) (Thread, error) {
	limit, limitErr := pageLimit(limit)
	if limitErr != nil {
		return Thread{}, limitErr
	}

	var out Thread
	err := s.reader.Read(ctx, func(ctx context.Context, view View) error {
		out = Thread{}
		meta, offset, err := startPage(ctx, view, cursor)
		if err != nil {
			return err
		}
		task, err := view.Task(ctx, taskID)
		if err != nil {
			return err
		}
		out.Meta, out.Task = meta, *task

		if out.Status.State, err = view.State(ctx); err != nil {
			return err
		}
		if out.Status.LastAttempt, err = view.LastAttempt(ctx); err != nil {
			return err
		}

		records, err := view.Records(ctx, taskID, offset, limit+1)
		if err != nil {
			return err
		}
		if len(records) > limit {
			records = records[:limit]
			out.NextCursor = encodeCursor(meta.Revision, offset+limit)
		}
		out.Records = records
		return nil
	})
	return out, err
}

// Board returns every task together with the feed status, from one snapshot.
func (s *QueryService) Board(ctx context.Context) (Board, error) {
	var out Board
	err := s.reader.Read(ctx, func(ctx context.Context, view View) error {
		out = Board{}
		meta, _, err := startPage(ctx, view, "")
		if err != nil {
			return err
		}
		out.Meta = meta

		if out.Status.State, err = view.State(ctx); err != nil {
			return err
		}
		if out.Status.LastAttempt, err = view.LastAttempt(ctx); err != nil {
			return err
		}

		tasks, err := view.Tasks(ctx, TaskFilter{}, 0, boardLimit+1)
		if err != nil {
			return err
		}
		if len(tasks) > boardLimit {
			tasks, out.Truncated = tasks[:boardLimit], true
		}
		out.Tasks = tasks
		return nil
	})
	return out, err
}

// startPage reads the snapshot identity and checks the cursor against it inside the same read.
func startPage(ctx context.Context, view View, cursor string) (Meta, int, error) {
	state, err := view.State(ctx)
	if err != nil {
		return Meta{}, 0, err
	}

	var meta Meta
	if state != nil {
		meta = Meta{Revision: state.Revision, AsOf: state.AsOf}
	}
	if cursor == "" {
		return meta, 0, nil
	}

	revision, offset, err := decodeCursor(cursor)
	if err != nil {
		return Meta{}, 0, err
	}
	if revision != meta.Revision {
		return Meta{}, 0, ErrRevisionChanged
	}
	return meta, offset, nil
}

func pageLimit(limit int) (int, error) {
	switch {
	case limit == 0:
		return DefaultPageSize, nil
	case limit < 0:
		return 0, ErrInvalidLimit
	case limit > MaxPageSize:
		return MaxPageSize, nil
	default:
		return limit, nil
	}
}

func encodeCursor(revision string, offset int) string {
	raw := strings.Join([]string{cursorVersion, revision, strconv.Itoa(offset)}, "|")
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(cursor string) (string, int, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return "", 0, ErrInvalidCursor
	}
	parts := strings.Split(string(raw), "|")
	const cursorParts = 3
	if len(parts) != cursorParts || parts[0] != cursorVersion || parts[1] == "" {
		return "", 0, ErrInvalidCursor
	}
	offset, err := strconv.Atoi(parts[2])
	if err != nil || offset < 0 {
		return "", 0, fmt.Errorf("%w: bad offset", ErrInvalidCursor)
	}
	return parts[1], offset, nil
}
