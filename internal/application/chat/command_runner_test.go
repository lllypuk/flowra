package chat_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/appcore"
	chatapp "github.com/lllypuk/flowra/internal/application/chat"
	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

type fakeCommandLog struct {
	mu      sync.Mutex
	records map[string]appcore.CommandRecord
}

func (f *fakeCommandLog) FindCommand(_ context.Context, aggregateID, commandID string) (appcore.CommandRecord, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	rec, ok := f.records[aggregateID+"/"+commandID]
	return rec, ok, nil
}

func (f *fakeCommandLog) put(aggregateID, commandID string, rec appcore.CommandRecord) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.records == nil {
		f.records = map[string]appcore.CommandRecord{}
	}
	f.records[aggregateID+"/"+commandID] = rec
}

func applied(version int) chatapp.Result {
	var r chatapp.Result
	r.Version = version
	return r
}

func TestCommandRunner_Run(t *testing.T) {
	chatID := uuid.NewUUID()
	key := appcore.CommandKey{ID: "k", Fingerprint: "due:2026-01-01"}
	errDomain := errors.New("domain says no")

	tests := []struct {
		name         string
		stored       *appcore.CommandRecord
		results      []error
		wantErr      error
		wantVersion  int
		wantReplayed bool
		wantCalls    int
	}{
		{name: "applied first time", results: []error{nil}, wantVersion: 7, wantCalls: 1},
		{
			name:        "conflict resolved by retry",
			results:     []error{errs.ErrConcurrentModification, appcore.ErrConcurrentUpdate, nil},
			wantVersion: 7, wantCalls: 3,
		},
		{
			name: "conflict exhausted is not applied",
			results: []error{errs.ErrConcurrentModification, errs.ErrConcurrentModification,
				errs.ErrConcurrentModification},
			wantErr: errs.ErrConcurrentModification, wantCalls: 3,
		},
		{name: "domain error is returned as is", results: []error{errDomain}, wantErr: errDomain, wantCalls: 1},
		{
			name:    "unknown outcome is not retried",
			results: []error{errs.ErrOutcomeUnknown},
			wantErr: errs.ErrOutcomeUnknown, wantCalls: 1,
		},
		{
			name:        "stored key replays without running",
			stored:      &appcore.CommandRecord{Version: 4, Fingerprint: key.Fingerprint},
			wantVersion: 4, wantReplayed: true,
		},
		{
			name:    "stored key with another request",
			stored:  &appcore.CommandRecord{Version: 4, Fingerprint: "due:2030-01-01"},
			wantErr: errs.ErrIdempotencyKeyReused,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := &fakeCommandLog{}
			if tt.stored != nil {
				log.put(chatID.String(), key.ID, *tt.stored)
			}
			runner := chatapp.NewCommandRunner(log, chatapp.WithCommandAttempts(3))

			calls := 0
			out, err := runner.Run(context.Background(), chatID, key, func(ctx context.Context) (chatapp.Result, error) {
				got, ok := appcore.CommandFrom(ctx)
				require.True(t, ok)
				assert.Equal(t, key, got)
				res := tt.results[calls]
				calls++
				if res != nil {
					return chatapp.Result{}, res
				}
				return applied(7), nil
			})

			assert.Equal(t, tt.wantCalls, calls)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantVersion, out.Version)
			assert.Equal(t, tt.wantReplayed, out.Replayed)
		})
	}
}

func TestCommandRunner_DuplicateFromAnotherWriterReplays(t *testing.T) {
	chatID := uuid.NewUUID()
	key := appcore.CommandKey{ID: "k", Fingerprint: "f"}
	log := &fakeCommandLog{}
	runner := chatapp.NewCommandRunner(log)

	out, err := runner.Run(context.Background(), chatID, key, func(context.Context) (chatapp.Result, error) {
		log.put(chatID.String(), key.ID, appcore.CommandRecord{Version: 9, Fingerprint: "f"})
		return chatapp.Result{}, appcore.ErrDuplicateCommand
	})
	require.NoError(t, err)
	assert.Equal(t, chatapp.Outcome{Version: 9, Replayed: true}, out)
}

func TestCommandRunner_SerializesOneChat(t *testing.T) {
	runner := chatapp.NewCommandRunner(nil)
	chatID := uuid.NewUUID()

	var inside, maxInside atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			_, err := runner.Run(context.Background(), chatID, appcore.CommandKey{},
				func(context.Context) (chatapp.Result, error) {
					n := inside.Add(1)
					if n > maxInside.Load() {
						maxInside.Store(n)
					}
					time.Sleep(time.Millisecond)
					inside.Add(-1)
					return applied(1), nil
				})
			assert.NoError(t, err)
		})
	}
	wg.Wait()
	assert.Equal(t, int32(1), maxInside.Load())
}

func TestCommandRunner_LockWaitIsBounded(t *testing.T) {
	runner := chatapp.NewCommandRunner(nil, chatapp.WithLockWait(20*time.Millisecond))
	chatID := uuid.NewUUID()

	release := make(chan struct{})
	started := make(chan struct{})
	go func() {
		_, _ = runner.Run(context.Background(), chatID, appcore.CommandKey{},
			func(context.Context) (chatapp.Result, error) {
				close(started)
				<-release
				return applied(1), nil
			})
	}()
	<-started

	calls := 0
	_, err := runner.Run(context.Background(), chatID, appcore.CommandKey{},
		func(context.Context) (chatapp.Result, error) {
			calls++
			return applied(2), nil
		})
	close(release)
	require.ErrorIs(t, err, errs.ErrConcurrentModification)
	assert.Zero(t, calls, "a command that never got the lock was not applied")

	_, err = runner.Run(context.Background(), uuid.NewUUID(), appcore.CommandKey{},
		func(context.Context) (chatapp.Result, error) { return applied(1), nil })
	require.NoError(t, err, "other chats do not wait")
}
