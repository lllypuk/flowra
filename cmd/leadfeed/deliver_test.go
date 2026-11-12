package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
	mongorepo "github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
)

const (
	fakeLeadTaskPath = "../../tests/fixtures/fake-lead-task.sh"
	askInput         = `{"body": "Synthetic question: which option?", "options": [{"id": "a", "label": "Option A"},` +
		` {"id": "b", "label": "Option B"}], "recommendation": {"option_id": "a", "why": "Synthetic reason."},` +
		` "scope": "Synthetic scope: the demo only."}`
)

type fakeLead struct {
	t    *testing.T
	path string
	dir  string
}

func newFakeLead(t *testing.T) fakeLead {
	t.Helper()
	for _, tool := range []string{"jq", "flock", "bash"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is required by the fake lead-task", tool)
		}
	}
	path, err := filepath.Abs(fakeLeadTaskPath)
	require.NoError(t, err)
	dir := t.TempDir()
	t.Setenv("LEAD_TASKS_DIR", dir)
	return fakeLead{t: t, path: path, dir: dir}
}

func (f fakeLead) call(stdin string, args ...string) (int, string, string) {
	f.t.Helper()
	cmd := exec.CommandContext(context.Background(), f.path, args...)
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if err == nil {
		return 0, stdout.String(), stderr.String()
	}
	var exitErr *exec.ExitError
	require.ErrorAs(f.t, err, &exitErr)
	return exitErr.ExitCode(), stdout.String(), stderr.String()
}

func (f fakeLead) must(stdin string, args ...string) string {
	f.t.Helper()
	code, stdout, stderr := f.call(stdin, args...)
	require.Zero(f.t, code, stderr)
	return stdout
}

func (f fakeLead) withQuestion() {
	f.t.Helper()
	f.must("", "new", "Synthetic task")
	f.must(askInput, "ask", "T-0001")
}

func (f fakeLead) events(kind string) []map[string]any {
	f.t.Helper()
	raw, err := os.ReadFile(filepath.Join(f.dir, "events.jsonl"))
	require.NoError(f.t, err)
	var out []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(string(raw)), "\n") {
		var event map[string]any
		require.NoError(f.t, json.Unmarshal([]byte(line), &event))
		if event["ev"] == kind {
			out = append(out, event)
		}
	}
	return out
}

func (f fakeLead) mailbox() []map[string]any {
	f.t.Helper()
	raw, err := os.ReadFile(filepath.Join(f.dir, "mailbox.log"))
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(f.t, err)
	var out []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(string(raw)), "\n") {
		var letter map[string]any
		require.NoError(f.t, json.Unmarshal([]byte(line), &letter))
		out = append(out, letter)
	}
	return out
}

func (f fakeLead) snapshotHeader() map[string]string {
	f.t.Helper()
	first, _, _ := strings.Cut(f.must("", "export"), "\n")
	var header map[string]any
	require.NoError(f.t, json.Unmarshal([]byte(first), &header))
	return map[string]string{"epoch": header["epoch"].(string)}
}

func (f fakeLead) questionFP(qid string) string {
	f.t.Helper()
	for line := range strings.SplitSeq(f.must("", "export"), "\n") {
		var rec struct {
			Question *struct {
				QID string `json:"qid"`
				FP  string `json:"fp"`
			} `json:"question"`
		}
		if json.Unmarshal([]byte(line), &rec) == nil && rec.Question != nil && rec.Question.QID == qid {
			return rec.Question.FP
		}
	}
	require.Fail(f.t, "question not in the export", qid)
	return ""
}

func (c cli) sync(f fakeLead) {
	c.t.Helper()
	code, stdout, stderr := c.run(f.must("", "export"), "import")
	require.Equal(c.t, exitOK, code, stdout+stderr)
}

func (c cli) deliver(f fakeLead) (int, string, string) {
	c.t.Helper()
	return c.run("", "deliver", "--lead-task", f.path)
}

func (c cli) service() *leadfeed.QuestionService {
	repo := mongorepo.NewMongoLeadFeedRepository(c.client, c.db)
	return leadfeed.NewQuestionService(repo, mongorepo.NewMongoOwnerAnswerRepository(c.db))
}

func (c cli) answer() leadfeed.OwnerAnswer {
	c.t.Helper()
	option := "a"
	res, err := c.service().Answer(context.Background(), leadfeed.AnswerCommand{
		QID: "Q-0001", OptionID: &option, Text: "Synthetic answer", Key: "key-Q-0001",
	})
	require.NoError(c.t, err)
	return res.Answer
}

// storeAnswer writes an answer the service would refuse, such as one for a closed question or with a wrong fingerprint.
func (c cli) storeAnswer(qid, answerID string, mutate func(*leadfeed.OwnerAnswer)) {
	c.t.Helper()
	repo := mongorepo.NewMongoLeadFeedRepository(c.client, c.db)
	var (
		state *leadfeed.State
		rec   *leadfeed.Record
	)
	err := repo.Read(context.Background(), func(ctx context.Context, view leadfeed.View) error {
		var err error
		if state, err = view.State(ctx); err != nil {
			return err
		}
		rec, err = view.Question(ctx, qid)
		return err
	})
	require.NoError(c.t, err)

	answer := leadfeed.OwnerAnswer{
		Source: state.Source, Epoch: state.Epoch, QID: qid, RecordID: rec.ID, QuestionFP: rec.Question.FP,
		Task: rec.Task, AnswerID: answerID, Text: "Synthetic answer", Scope: rec.Question.Scope,
		AnsweredAt: state.AsOf, CommandID: "raw-" + qid, CommandFP: "raw-fp",
	}
	if mutate != nil {
		mutate(&answer)
	}
	_, err = mongorepo.NewMongoOwnerAnswerRepository(c.db).SaveAnswer(context.Background(), answer)
	require.NoError(c.t, err)
}

func (c cli) questionState() leadfeed.QuestionState {
	c.t.Helper()
	detail, err := c.service().Question(context.Background(), "Q-0001")
	require.NoError(c.t, err)
	return detail.Question.State
}

func (c cli) storedAnswers() []leadfeed.OwnerAnswer {
	c.t.Helper()
	repo := mongorepo.NewMongoLeadFeedRepository(c.client, c.db)
	var state *leadfeed.State
	require.NoError(c.t, repo.Read(context.Background(), func(ctx context.Context, view leadfeed.View) error {
		var err error
		state, err = view.State(ctx)
		return err
	}))
	answers, err := mongorepo.NewMongoOwnerAnswerRepository(c.db).
		ListAnswers(context.Background(), state.Source, state.Epoch)
	require.NoError(c.t, err)
	return answers
}

func TestDeliver_DeliveredAndConfirmedInTwoRuns(t *testing.T) {
	f, c := newFakeLead(t), newCLI(t)
	f.withQuestion()
	c.sync(f)
	answer := c.answer()
	assert.Equal(t, leadfeed.QuestionAnswered, c.questionState())

	code, stdout, stderr := c.deliver(f)

	require.Equal(t, exitOK, code, stderr)
	assert.Contains(t, stdout, "sent=1 failed=0 unconfirmed=1")
	require.Len(t, f.events("answer"), 1)
	assert.Equal(t, answer.AnswerID, f.events("answer")[0]["answer_id"])
	assert.Len(t, f.events("delivered"), 1)
	assert.Len(t, f.mailbox(), 1)

	_, status, _ := c.run("", "status")
	assert.Contains(t, status, "last delivery: ")
	assert.Contains(t, status, "sent=1 failed=0 unconfirmed=1")

	c.sync(f)
	assert.Equal(t, leadfeed.QuestionDelivered, c.questionState())
	code, stdout, stderr = c.deliver(f)
	require.Equal(t, exitOK, code, stderr)
	assert.Contains(t, stdout, "sent=0 failed=0 unconfirmed=0")
	assert.Len(t, f.mailbox(), 1, "a confirmed answer is not sent again")

	f.must("", "resolve", "Q-0001", "-m", "handled")
	c.sync(f)
	assert.Equal(t, leadfeed.QuestionHandled, c.questionState())
}

func TestDeliver_NoLeadPanelRetriesOnTheNextRun(t *testing.T) {
	f, c := newFakeLead(t), newCLI(t)
	f.withQuestion()
	c.sync(f)
	c.answer()

	t.Setenv("FAKE_LEAD_TASK_MAIL", "fail")
	code, _, stderr := c.deliver(f)

	require.Equal(t, exitFailed, code)
	assert.Contains(t, stderr, "exited with 3")
	assert.Len(t, f.events("answer"), 1, "the answer is written before the letter")
	assert.Empty(t, f.events("delivered"))
	_, status, _ := c.run("", "status")
	assert.Contains(t, status, "failed=1 unconfirmed=1")
	assert.Contains(t, status, "last delivery error: Q-0001: lead-task answer exited with 3")

	t.Setenv("FAKE_LEAD_TASK_MAIL", "ok")
	code, stdout, stderr := c.deliver(f)

	require.Equal(t, exitOK, code, stderr)
	assert.Contains(t, stdout, "sent=1 failed=0")
	assert.Len(t, f.events("answer"), 1)
	assert.Len(t, f.events("delivered"), 1)
	assert.Len(t, f.mailbox(), 1)
}

func TestDeliver_DeathBetweenLetterAndMarkRepeatsOnlyTheLetter(t *testing.T) {
	f, c := newFakeLead(t), newCLI(t)
	f.withQuestion()
	c.sync(f)
	c.answer()

	t.Setenv("FAKE_LEAD_TASK_MAIL", "lost")
	code, _, _ := c.deliver(f)
	require.Equal(t, exitFailed, code)
	assert.Len(t, f.mailbox(), 1)
	assert.Empty(t, f.events("delivered"))

	t.Setenv("FAKE_LEAD_TASK_MAIL", "ok")
	code, _, stderr := c.deliver(f)

	require.Equal(t, exitOK, code, stderr)
	assert.Len(t, f.mailbox(), 2, "at least once: the letter may arrive twice")
	assert.Equal(t, f.mailbox()[0], f.mailbox()[1], "both letters name the same qid and answer_id")
	assert.Len(t, f.events("answer"), 1)
	assert.Len(t, f.events("delivered"), 1)
}

func TestDeliver_AnotherAnswerInTheJournalIsAReconciliationError(t *testing.T) {
	f, c := newFakeLead(t), newCLI(t)
	f.withQuestion()
	c.sync(f)
	c.answer()
	header := f.snapshotHeader()
	foreign, err := json.Marshal(map[string]string{
		"source": "lead-task", "epoch": header["epoch"], "qid": "Q-0001", "answer_id": "foreign",
		"question_fp": f.questionFP("Q-0001"), "answered_at": "2026-01-10T10:00:00.000Z", "text": "elsewhere",
	})
	require.NoError(t, err)
	f.must(string(foreign), "answer", "Q-0001")

	for range 2 {
		code, _, stderr := c.deliver(f)
		require.Equal(t, exitFailed, code)
		assert.Contains(t, stderr, "exited with 4")
		c.sync(f)
	}

	_, status, _ := c.run("", "status")
	assert.Contains(t, status, "failed=1 unconfirmed=1", "the foreign answer_id does not confirm ours")
	assert.Len(t, c.storedAnswers(), 1, "the answer stays in the store")
	assert.Equal(t, leadfeed.QuestionAnswered, c.questionState())
	assert.Len(t, f.events("answer"), 1)
}

func TestDeliver_WrongQuestionFingerprintIsAReconciliationError(t *testing.T) {
	f, c := newFakeLead(t), newCLI(t)
	f.withQuestion()
	c.sync(f)
	c.storeAnswer("Q-0001", "ans-1", func(a *leadfeed.OwnerAnswer) { a.QuestionFP = "not-the-fingerprint" })

	code, _, stderr := c.deliver(f)

	require.Equal(t, exitFailed, code)
	assert.Contains(t, stderr, "exited with 5")
	assert.Empty(t, f.events("answer"))
	assert.Len(t, c.storedAnswers(), 1)
	assert.Empty(t, f.mailbox())
}

func TestDeliver_AnswerToAQuestionClosedMeanwhileIsDeliveredAsLate(t *testing.T) {
	f, c := newFakeLead(t), newCLI(t)
	f.withQuestion()
	c.sync(f)
	c.answer()
	f.must("", "resolve", "Q-0001", "--elsewhere")
	c.sync(f)

	code, _, stderr := c.deliver(f)

	require.Equal(t, exitOK, code, stderr)
	require.Len(t, f.events("answer"), 1)
	assert.Equal(t, true, f.events("answer")[0]["late"])
	require.Len(t, f.mailbox(), 1)
	assert.Equal(t, true, f.mailbox()[0]["late"])

	c.sync(f)
	assert.Equal(t, leadfeed.QuestionHandled, c.questionState())
}

func TestDeliver_UsageAndMissingExecutable(t *testing.T) {
	c := newCLI(t)

	code, _, _ := c.run("", "deliver")
	assert.Equal(t, exitUsage, code)

	code, _, _ = c.run("", "deliver", "--lead-task", "x", "extra")
	assert.Equal(t, exitUsage, code)

	raw, err := os.ReadFile("../../tests/fixtures/leadfeed/questions.jsonl")
	require.NoError(t, err)
	code, _, stderr := c.run(string(raw), "import")
	require.Equal(t, exitOK, code, stderr)
	c.storeAnswer("Q-0001", "ans-1", nil)

	code, _, stderr = c.run("", "deliver", "--lead-task", filepath.Join(t.TempDir(), "absent"))

	assert.Equal(t, exitFailed, code)
	assert.Contains(t, stderr, "not delivered")
}
