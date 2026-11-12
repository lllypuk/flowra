package chat

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

const (
	defaultCommandAttempts = 5
	defaultLockWait        = 5 * time.Second
	retryBaseDelay         = 5 * time.Millisecond
	retryMaxDelay          = 100 * time.Millisecond
)

// CommandLog finds the stored effect of a command key.
type CommandLog interface {
	FindCommand(ctx context.Context, aggregateID, commandID string) (appcore.CommandRecord, bool, error)
}

// Outcome is the definite result of an applied chat command.
type Outcome struct {
	// Version is the chat version after the command.
	Version int
	// Replayed is true when the command key was applied earlier and nothing was written now.
	Replayed bool
}

// CommandRunner applies chat commands one at a time per chat, retries version conflicts by reloading
// the aggregate, and applies a keyed command at most once.
type CommandRunner struct {
	log      CommandLog
	attempts int
	lockWait time.Duration

	mu    sync.Mutex
	locks map[uuid.UUID]*chatLock
}

type chatLock struct {
	ch   chan struct{}
	refs int
}

// CommandRunnerOption customizes CommandRunner.
type CommandRunnerOption func(*CommandRunner)

// WithCommandAttempts sets how many times a command runs before a version conflict is returned.
func WithCommandAttempts(n int) CommandRunnerOption {
	return func(r *CommandRunner) { r.attempts = max(1, n) }
}

// WithLockWait bounds the wait for another command on the same chat.
func WithLockWait(d time.Duration) CommandRunnerOption {
	return func(r *CommandRunner) { r.lockWait = d }
}

// NewCommandRunner creates a CommandRunner; log may be nil, then command keys are not checked.
func NewCommandRunner(log CommandLog, opts ...CommandRunnerOption) *CommandRunner {
	r := &CommandRunner{
		log:      log,
		attempts: defaultCommandAttempts,
		lockWait: defaultLockWait,
		locks:    map[uuid.UUID]*chatLock{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Run applies exec, which must load, change and save the chat itself, so that each attempt sees fresh state.
// Errors: errs.ErrConcurrentModification — not applied; errs.ErrOutcomeUnknown — retry with the same key;
// errs.ErrIdempotencyKeyReused; anything else comes from exec and was not applied.
func (r *CommandRunner) Run(
	ctx context.Context,
	chatID uuid.UUID,
	key appcore.CommandKey,
	exec func(context.Context) (Result, error),
) (Outcome, error) {
	unlock, err := r.lock(ctx, chatID)
	if err != nil {
		return Outcome{}, err
	}
	defer unlock()

	cmdCtx := appcore.WithCommand(ctx, key)
	var lastErr error
	for attempt := range r.attempts {
		if outcome, done, findErr := r.replay(ctx, chatID, key); findErr != nil || done {
			return outcome, findErr
		}

		res, execErr := exec(cmdCtx)
		switch {
		case execErr == nil:
			return Outcome{Version: res.Version}, nil
		case errors.Is(execErr, appcore.ErrDuplicateCommand):
			continue // another process applied the key: the next replay check returns its outcome
		case errors.Is(execErr, errs.ErrConcurrentModification), errors.Is(execErr, appcore.ErrConcurrentUpdate):
			lastErr = execErr
		default:
			return Outcome{}, execErr
		}

		if attempt < r.attempts-1 {
			if waitErr := sleepCtx(ctx, retryDelay(attempt)); waitErr != nil {
				return Outcome{}, fmt.Errorf("%w: %w", errs.ErrConcurrentModification, waitErr)
			}
		}
	}

	if outcome, done, findErr := r.replay(ctx, chatID, key); findErr != nil || done {
		return outcome, findErr
	}
	return Outcome{}, fmt.Errorf("%w: not applied after %d attempts: %w",
		errs.ErrConcurrentModification, r.attempts, lastErr)
}

// replay returns the stored outcome of key; done is false when the key is unset or unknown.
func (r *CommandRunner) replay(
	ctx context.Context,
	chatID uuid.UUID,
	key appcore.CommandKey,
) (Outcome, bool, error) {
	if key.ID == "" || r.log == nil {
		return Outcome{}, false, nil
	}
	rec, found, err := r.log.FindCommand(ctx, chatID.String(), key.ID)
	if err != nil {
		return Outcome{}, false, fmt.Errorf("%w: check command key: %w", errs.ErrOutcomeUnknown, err)
	}
	if !found {
		return Outcome{}, false, nil
	}
	if rec.Fingerprint != key.Fingerprint {
		return Outcome{}, false, errs.ErrIdempotencyKeyReused
	}
	return Outcome{Version: rec.Version, Replayed: true}, true, nil
}

func (r *CommandRunner) lock(ctx context.Context, chatID uuid.UUID) (func(), error) {
	r.mu.Lock()
	l := r.locks[chatID]
	if l == nil {
		l = &chatLock{ch: make(chan struct{}, 1)}
		r.locks[chatID] = l
	}
	l.refs++
	r.mu.Unlock()

	release := func() {
		r.mu.Lock()
		l.refs--
		if l.refs == 0 {
			delete(r.locks, chatID)
		}
		r.mu.Unlock()
	}

	timer := time.NewTimer(r.lockWait)
	defer timer.Stop()
	select {
	case l.ch <- struct{}{}:
		return func() { <-l.ch; release() }, nil
	case <-timer.C:
		release()
		return nil, fmt.Errorf("%w: chat busy for %s", errs.ErrConcurrentModification, r.lockWait)
	case <-ctx.Done():
		release()
		return nil, fmt.Errorf("%w: %w", errs.ErrConcurrentModification, ctx.Err())
	}
}

func retryDelay(attempt int) time.Duration {
	ceiling := min(retryMaxDelay, retryBaseDelay<<attempt)
	return ceiling/2 + rand.N(ceiling/2+1) //nolint:gosec // jitter, not security
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
