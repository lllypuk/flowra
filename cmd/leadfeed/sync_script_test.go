package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const syncScriptPath = "../../scripts/leadfeed-sync.sh"

type syncRun struct {
	cmd  *exec.Cmd
	done chan error
}

func (c cli) startSync(f fakeLead, lock string) *syncRun {
	c.t.Helper()
	bin := filepath.Join(c.t.TempDir(), "leadfeed")
	build := exec.CommandContext(context.Background(), "go", "build", "-o", bin, ".")
	out, err := build.CombinedOutput()
	require.NoError(c.t, err, string(out))
	script, err := filepath.Abs(syncScriptPath)
	require.NoError(c.t, err)

	cmd := exec.CommandContext(context.Background(), "bash", script)
	cmd.Env = append(os.Environ(),
		"LEAD_TASK="+f.path, "LEADFEED="+bin, "LEADFEED_SYNC_LOCK="+lock,
		"MONGODB_URI="+c.uri, "MONGODB_DATABASE="+c.db.Name())
	require.NoError(c.t, cmd.Start())
	run := &syncRun{cmd: cmd, done: make(chan error, 1)}
	go func() { run.done <- cmd.Wait() }()
	return run
}

func (r *syncRun) wait(t *testing.T) int {
	t.Helper()
	select {
	case err := <-r.done:
		if err == nil {
			return 0
		}
		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
		return exitErr.ExitCode()
	case <-time.After(time.Minute):
		require.FailNow(t, "the sync script did not finish")
		return -1
	}
}

func TestSyncScript_FailedExportStillDeliversAndFails(t *testing.T) {
	f, c := newFakeLead(t), newCLI(t)
	f.withQuestion()
	c.sync(f)
	c.answer()

	t.Setenv("FAKE_LEAD_TASK_EXPORT", "fail")
	code := c.startSync(f, filepath.Join(t.TempDir(), "lock")).wait(t)

	assert.Equal(t, 1, code, "a failed export fails the run")
	assert.Len(t, f.events("answer"), 1, "delivery runs after a failed export")
	assert.Len(t, f.mailbox(), 1)
	_, status, _ := c.run("", "status")
	assert.Contains(t, status, "last attempt: failed")
	assert.Contains(t, status, "sent=1 failed=0")
}

func TestSyncScript_FailedDeliveryFailsButTheSnapshotIsPublished(t *testing.T) {
	f, c := newFakeLead(t), newCLI(t)
	f.withQuestion()
	c.sync(f)
	c.answer()

	t.Setenv("FAKE_LEAD_TASK_MAIL", "fail")
	code := c.startSync(f, filepath.Join(t.TempDir(), "lock")).wait(t)

	assert.Equal(t, 1, code)
	_, status, _ := c.run("", "status")
	assert.Contains(t, status, "last attempt: refreshed", "a mail failure does not spoil the publication")
	assert.Contains(t, status, "failed=1")
}

func TestSyncScript_HealthyRunPublishesDeliversAndSucceeds(t *testing.T) {
	f, c := newFakeLead(t), newCLI(t)
	f.withQuestion()

	code := c.startSync(f, filepath.Join(t.TempDir(), "lock")).wait(t)

	assert.Zero(t, code)
	_, status, _ := c.run("", "status")
	assert.Contains(t, status, "last attempt: published")
	assert.Contains(t, status, "last delivery: ")
}

func TestSyncScript_WaitsForTheRunThatHoldsTheLock(t *testing.T) {
	f, c := newFakeLead(t), newCLI(t)
	f.withQuestion()
	lock := filepath.Join(t.TempDir(), "lock")
	held, err := os.OpenFile(lock, os.O_CREATE|os.O_RDWR, 0o600)
	require.NoError(t, err)
	defer held.Close()
	require.NoError(t, syscall.Flock(int(held.Fd()), syscall.LOCK_EX))

	run := c.startSync(f, lock)

	select {
	case err = <-run.done:
		require.FailNow(t, "the script ran while the lock was held", "%v", err)
	case <-time.After(time.Second):
	}
	_, status, _ := c.run("", "status")
	assert.Contains(t, status, "published: nothing yet")

	require.NoError(t, syscall.Flock(int(held.Fd()), syscall.LOCK_UN))
	assert.Zero(t, run.wait(t))
	_, status, _ = c.run("", "status")
	assert.Contains(t, status, "published: revision=")
}

func TestSyncScript_GivesUpWhenTheLockStaysHeld(t *testing.T) {
	f, c := newFakeLead(t), newCLI(t)
	lock := filepath.Join(t.TempDir(), "lock")
	held, err := os.OpenFile(lock, os.O_CREATE|os.O_RDWR, 0o600)
	require.NoError(t, err)
	defer held.Close()
	require.NoError(t, syscall.Flock(int(held.Fd()), syscall.LOCK_EX))
	t.Setenv("LEADFEED_SYNC_LOCK_WAIT", "1")

	code := c.startSync(f, lock).wait(t)

	assert.Equal(t, 1, code)
}
