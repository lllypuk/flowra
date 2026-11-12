package leadfeed

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"
)

// DefaultMaxSnapshotBytes bounds one import: the whole snapshot is held in memory and written in one transaction.
const DefaultMaxSnapshotBytes = 32 << 20

const attemptRecordTimeout = 5 * time.Second

// ErrSnapshotTooLarge means the input was refused before parsing.
var ErrSnapshotTooLarge = errors.New("snapshot is larger than the import limit")

// Publisher is what the importer needs from the feed store.
type Publisher interface {
	Publish(ctx context.Context, snap *Snapshot, now time.Time) (PublishResult, error)
	RecordAttempt(ctx context.Context, attempt Attempt) error
}

// ImportResult describes a snapshot that was published or confirmed.
type ImportResult struct {
	Outcome       Outcome
	Revision      string
	AsOf          time.Time
	Tasks         int
	Records       int
	PreviousEpoch string
}

// Importer turns one full snapshot into the published feed.
type Importer struct {
	store    Publisher
	maxBytes int64
	now      func() time.Time
	logger   *slog.Logger
}

// ImporterOption configures Importer.
type ImporterOption func(*Importer)

// WithMaxSnapshotBytes overrides the input size limit.
func WithMaxSnapshotBytes(limit int64) ImporterOption {
	return func(i *Importer) { i.maxBytes = limit }
}

// WithClock overrides the time source.
func WithClock(now func() time.Time) ImporterOption {
	return func(i *Importer) { i.now = now }
}

// WithImporterLogger sets the logger.
func WithImporterLogger(logger *slog.Logger) ImporterOption {
	return func(i *Importer) { i.logger = logger }
}

// NewImporter creates an importer.
func NewImporter(store Publisher, opts ...ImporterOption) *Importer {
	i := &Importer{
		store:    store,
		maxBytes: DefaultMaxSnapshotBytes,
		now:      time.Now,
		logger:   slog.Default(),
	}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

// Import reads the whole input, validates it and only then publishes it.
// Any error leaves the published feed as it was, except ErrCommitUnknown, where it is old or new in full.
func (i *Importer) Import(ctx context.Context, r io.Reader) (ImportResult, error) {
	snap, err := i.read(r)
	if err != nil {
		i.record(ctx, Attempt{Outcome: AttemptRejected, Error: err.Error()})
		return ImportResult{}, err
	}

	published, err := i.store.Publish(ctx, snap, i.now())
	if err != nil {
		i.record(ctx, Attempt{Outcome: failureOutcome(err), Error: err.Error(), Revision: snap.Header.Revision})
		return ImportResult{}, err
	}

	i.record(ctx, Attempt{
		Outcome:       string(published.Outcome),
		Revision:      snap.Header.Revision,
		PreviousEpoch: published.PreviousEpoch,
	})

	return ImportResult{
		Outcome:       published.Outcome,
		Revision:      snap.Header.Revision,
		AsOf:          snap.Header.AsOf,
		Tasks:         len(snap.Tasks),
		Records:       len(snap.Records),
		PreviousEpoch: published.PreviousEpoch,
	}, nil
}

func (i *Importer) read(r io.Reader) (*Snapshot, error) {
	raw, err := io.ReadAll(io.LimitReader(r, i.maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read snapshot: %w", err)
	}
	if int64(len(raw)) > i.maxBytes {
		return nil, fmt.Errorf("%w: more than %d bytes", ErrSnapshotTooLarge, i.maxBytes)
	}
	return ParseSnapshot(bytes.NewReader(raw))
}

func failureOutcome(err error) string {
	switch {
	case errors.Is(err, ErrCommitUnknown):
		return AttemptUnknown
	case errors.Is(err, ErrSnapshotStale), errors.Is(err, ErrSnapshotConflict):
		return AttemptRejected
	default:
		return AttemptFailed
	}
}

func (i *Importer) record(ctx context.Context, attempt Attempt) {
	attempt.At = i.now()

	// Detached from ctx: an interrupted import must still leave its attempt on record.
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), attemptRecordTimeout)
	defer cancel()

	if err := i.store.RecordAttempt(recordCtx, attempt); err != nil {
		i.logger.WarnContext(ctx, "failed to record feed import attempt", slog.String("error", err.Error()))
	}
}
