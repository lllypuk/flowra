package leadfeed

import (
	"context"
	"errors"
	"time"
)

// Outcome says what a publication did to the feed.
type Outcome string

const (
	// OutcomePublished means the feed content was replaced.
	OutcomePublished Outcome = "published"
	// OutcomeRefreshed means the same revision arrived again and only its as_of moved forward.
	OutcomeRefreshed Outcome = "refreshed"
)

// Publication and lookup errors.
var (
	ErrSnapshotStale    = errors.New("snapshot is older than the published one")
	ErrSnapshotConflict = errors.New("snapshot has the published as_of but different content")
	// ErrCommitUnknown means the feed holds either the old or the new snapshot in full; which one is not known.
	ErrCommitUnknown = errors.New("publication commit result is unknown")
	ErrTaskNotFound  = errors.New("feed task not found")
	// ErrQuestionNotFound means the published feed has no question with that qid.
	ErrQuestionNotFound = errors.New("feed question not found")
)

// State describes the published snapshot.
type State struct {
	Source      string
	Epoch       string
	Revision    string
	AsOf        time.Time
	Tasks       int
	Records     int
	PublishedAt time.Time
}

// PublishResult reports a successful publication.
type PublishResult struct {
	Outcome Outcome
	// PreviousEpoch is set when the snapshot came from another journal installation.
	PreviousEpoch string
}

// Attempt outcomes beyond the publication ones.
const (
	AttemptRejected = "rejected"
	AttemptFailed   = "failed"
	AttemptUnknown  = "unknown"
)

// Attempt is the last import attempt; it never changes the published revision or as_of.
type Attempt struct {
	At            time.Time
	Outcome       string
	Error         string
	Revision      string
	PreviousEpoch string
}

// TaskFilter narrows a task list; nil fields do not filter.
type TaskFilter struct {
	Status *string
	// WaitingOn matches the source value; WaitingOnNobody selects tasks that wait on no one.
	WaitingOn *string
}

// WaitingOnNobody is the filter value for tasks whose waiting_on is null.
const WaitingOnNobody = "none"

// View is one consistent read of the published feed.
type View interface {
	// State returns nil when nothing has been published yet.
	State(ctx context.Context) (*State, error)
	// LastAttempt returns nil when no import has run yet.
	LastAttempt(ctx context.Context) (*Attempt, error)
	Tasks(ctx context.Context, filter TaskFilter, offset, limit int) ([]Task, error)
	// Task returns ErrTaskNotFound for an unknown id.
	Task(ctx context.Context, id string) (*Task, error)
	Records(ctx context.Context, taskID string, offset, limit int) ([]Record, error)
	// Question returns the record of kind question with that qid, or ErrQuestionNotFound.
	Question(ctx context.Context, qid string) (*Record, error)
	// Questions returns every question record of the feed, oldest first.
	Questions(ctx context.Context) ([]Record, error)
}
