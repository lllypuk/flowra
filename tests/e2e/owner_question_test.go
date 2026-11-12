//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	httphandler "github.com/lllypuk/flowra/internal/handler/http"
	"github.com/lllypuk/flowra/tests/testutil"
)

const (
	fakeLeadTask = "../fixtures/fake-lead-task.sh"
	syncScript   = "../../scripts/leadfeed-sync.sh"

	routeAskInput = `{"body": "Synthetic question: which option?", "options": [{"id": "a", "label": "Option A"},` +
		` {"id": "b", "label": "Option B"}], "recommendation": {"option_id": "a", "why": "Synthetic reason."},` +
		` "scope": "Synthetic scope: the demo only."}`

	stateWaiting   = "waiting"
	stateAnswered  = "answered"
	stateDelivered = "delivered"
	stateHandled   = "handled"
)

// leadRig runs the real sync script and the real leadfeed binary against the suite database, with the fake lead-task.
type leadRig struct {
	t      *testing.T
	dir    string
	bin    string
	script string
	fake   string
	uri    string
	db     string
	lock   string
	client *HTTPClient
}

func newLeadRig(t *testing.T) *leadRig {
	t.Helper()
	for _, tool := range []string{"jq", "flock", "bash", "go"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is required by the lead feed route", tool)
		}
	}
	suite := NewE2ETestSuite(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cont, err := testutil.GetSharedMongoContainer(ctx)
	require.NoError(t, err)

	bin := filepath.Join(t.TempDir(), "leadfeed")
	build := exec.CommandContext(context.Background(), "go", "build", "-o", bin, "../../cmd/leadfeed")
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))

	fake, err := filepath.Abs(fakeLeadTask)
	require.NoError(t, err)
	script, err := filepath.Abs(syncScript)
	require.NoError(t, err)

	return &leadRig{
		t: t, dir: t.TempDir(), bin: bin, script: script, fake: fake, uri: cont.URI, db: suite.MongoDB.Name(),
		lock:   filepath.Join(t.TempDir(), "sync.lock"),
		client: suite.NewHTTPClient(suite.CreateTestUser("owner").Token),
	}
}

func (r *leadRig) env(extra ...string) []string {
	return append(os.Environ(), append([]string{"LEAD_TASKS_DIR=" + r.dir}, extra...)...)
}

func exitCodeOf(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	return exitErr.ExitCode()
}

func (r *leadRig) leadTask(stdin string, args ...string) (int, string) {
	r.t.Helper()
	cmd := exec.CommandContext(context.Background(), r.fake, args...)
	cmd.Env = r.env()
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	code := exitCodeOf(r.t, cmd.Run())
	return code, stdout.String() + stderr.String()
}

func (r *leadRig) mustLeadTask(stdin string, args ...string) {
	r.t.Helper()
	code, out := r.leadTask(stdin, args...)
	require.Zero(r.t, code, out)
}

func (r *leadRig) ask() {
	r.t.Helper()
	if _, err := os.Stat(filepath.Join(r.dir, "events.jsonl")); err != nil {
		r.mustLeadTask("", "new", "Synthetic task")
	}
	r.mustLeadTask(routeAskInput, "ask", "T-0001")
}

func (r *leadRig) sync(extraEnv ...string) int {
	r.t.Helper()
	return r.syncWith(r.uri, extraEnv...)
}

func (r *leadRig) syncWith(uri string, extraEnv ...string) int {
	r.t.Helper()
	cmd := exec.CommandContext(context.Background(), "bash", r.script)
	cmd.Env = r.env(append([]string{
		"LEAD_TASK=" + r.fake, "LEADFEED=" + r.bin, "LEADFEED_SYNC_LOCK=" + r.lock,
		"MONGODB_URI=" + uri, "MONGODB_DATABASE=" + r.db,
	}, extraEnv...)...)
	out, err := cmd.CombinedOutput()
	code := exitCodeOf(r.t, err)
	r.t.Logf("sync exit=%d\n%s", code, out)
	return code
}

func (r *leadRig) journal(kind string) []map[string]any {
	r.t.Helper()
	return jsonLines(r.t, filepath.Join(r.dir, "events.jsonl"), func(e map[string]any) bool { return e["ev"] == kind })
}

func (r *leadRig) mailbox() []map[string]any {
	r.t.Helper()
	return jsonLines(r.t, filepath.Join(r.dir, "mailbox.log"), nil)
}

func jsonLines(t *testing.T, path string, keep func(map[string]any) bool) []map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	require.NoError(t, err)
	var out []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(string(raw)), "\n") {
		var item map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &item))
		if keep == nil || keep(item) {
			out = append(out, item)
		}
	}
	return out
}

func (r *leadRig) post(qid, key string, option string, text string) *http.Response {
	r.t.Helper()
	body := map[string]any{"text": text}
	if option != "" {
		body["option_id"] = option
	}
	return r.client.DoRequestWithHeaders(http.MethodPost, "/feed/questions/"+qid+"/answer", body,
		map[string]string{"Idempotency-Key": key})
}

func (r *leadRig) question(qid string) httphandler.FeedQuestionResponse {
	r.t.Helper()
	resp := r.client.Get("/feed/questions/" + qid)
	AssertStatus(r.t, resp, http.StatusOK)
	return ParseResponse[APIResponse[httphandler.FeedQuestionDetailResponse]](r.t, resp).Data.Question
}

func (r *leadRig) state(qid string) string {
	r.t.Helper()
	return r.question(qid).State
}

func errorCode(t *testing.T, resp *http.Response, status int) string {
	t.Helper()
	AssertStatus(t, resp, status)
	body := ParseResponse[APIResponse[any]](t, resp)
	require.NotNil(t, body.Error)
	return body.Error.Code
}

func TestOwnerQuestion_RouteEndToEnd(t *testing.T) {
	r := newLeadRig(t)
	r.ask()
	require.Zero(t, r.sync())
	assert.Equal(t, stateWaiting, r.state("Q-0001"))

	list := ParseResponse[APIResponse[httphandler.FeedQuestionListResponse]](t, r.client.Get("/feed/questions"))
	assert.Equal(t, 1, list.Data.Waiting)

	resp := r.post("Q-0001", "key-1", "a", "Synthetic answer")
	AssertStatus(t, resp, http.StatusCreated)
	answer := ParseResponse[APIResponse[httphandler.FeedAnswerResultResponse]](t, resp).Data.Answer
	assert.Equal(t, stateAnswered, r.state("Q-0001"))
	assert.Empty(t, r.journal("answer"), "nothing reaches the lead before the sync")

	require.Zero(t, r.sync())
	require.Len(t, r.journal("answer"), 1)
	assert.Equal(t, answer.AnswerID, r.journal("answer")[0]["answer_id"])
	require.Len(t, r.mailbox(), 1)
	assert.Equal(t, "Q-0001", r.mailbox()[0]["qid"])
	assert.Len(t, r.journal("delivered"), 1)

	require.Zero(t, r.sync())
	assert.Equal(t, stateDelivered, r.state("Q-0001"))

	r.mustLeadTask("", "resolve", "Q-0001", "-m", "handled")
	require.Zero(t, r.sync())
	handled := r.question("Q-0001")
	assert.Equal(t, stateHandled, handled.State)
	require.NotNil(t, handled.Answer)
	assert.Equal(t, answer.AnswerID, handled.Answer.AnswerID)
	assert.Len(t, r.journal("answer"), 1)
}

func TestOwnerQuestion_RepeatedDeliveryWritesOneAnswer(t *testing.T) {
	r := newLeadRig(t)
	r.ask()
	require.Zero(t, r.sync())
	AssertStatus(t, r.post("Q-0001", "key-1", "b", ""), http.StatusCreated)

	for range 3 {
		require.Zero(t, r.sync())
	}
	resp := r.post("Q-0001", "key-1", "b", "")
	AssertStatus(t, resp, http.StatusOK)
	assert.True(t, ParseResponse[APIResponse[httphandler.FeedAnswerResultResponse]](t, resp).Data.Replayed)

	assert.Len(t, r.journal("answer"), 1)
	assert.Len(t, r.journal("delivered"), 1)
	assert.Len(t, r.mailbox(), 1, "a confirmed answer is not sent again")
}

func TestOwnerQuestion_LostLetterMarkIsRepeatedWithoutAnotherAnswer(t *testing.T) {
	r := newLeadRig(t)
	r.ask()
	require.Zero(t, r.sync())
	AssertStatus(t, r.post("Q-0001", "key-1", "a", ""), http.StatusCreated)

	assert.NotZero(t, r.sync("FAKE_LEAD_TASK_MAIL=lost"), "the failed delivery fails the run")
	assert.Len(t, r.journal("answer"), 1)
	assert.Empty(t, r.journal("delivered"))

	require.Zero(t, r.sync())
	require.Zero(t, r.sync())
	assert.Len(t, r.journal("answer"), 1)
	assert.Len(t, r.journal("delivered"), 1)
	assert.Equal(t, stateDelivered, r.state("Q-0001"))
}

func TestOwnerQuestion_FlowraUnavailableAtAskLosesNothing(t *testing.T) {
	r := newLeadRig(t)
	r.ask()

	unreachable := closedPortURI(t)
	assert.NotZero(t, r.syncWith(unreachable), "import and delivery fail while the database is unreachable")
	require.Len(t, r.journal("ask"), 1, "the question is in the lead's journal")

	resp := r.client.Get("/feed/questions/Q-0001")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	_ = resp.Body.Close()

	require.Zero(t, r.sync())
	assert.Equal(t, stateWaiting, r.state("Q-0001"))

	AssertStatus(t, r.post("Q-0001", "key-1", "a", "Synthetic answer"), http.StatusCreated)
	require.Zero(t, r.sync())
	require.Zero(t, r.sync())
	assert.Equal(t, stateDelivered, r.state("Q-0001"))
	r.mustLeadTask("", "resolve", "Q-0001")
	require.Zero(t, r.sync())
	assert.Equal(t, stateHandled, r.state("Q-0001"))
	assert.Len(t, r.journal("answer"), 1)
}

func TestOwnerQuestion_ReplayAfterResolvedKeepsTheFirstResult(t *testing.T) {
	r := newLeadRig(t)
	r.ask()
	require.Zero(t, r.sync())
	first := ParseResponse[APIResponse[httphandler.FeedAnswerResultResponse]](t,
		r.post("Q-0001", "key-1", "a", "Synthetic answer"))

	require.Zero(t, r.sync())
	r.mustLeadTask("", "resolve", "Q-0001")
	require.Zero(t, r.sync())
	require.Equal(t, stateHandled, r.state("Q-0001"))

	resp := r.post("Q-0001", "key-1", "a", "Synthetic answer")
	AssertStatus(t, resp, http.StatusOK)
	again := ParseResponse[APIResponse[httphandler.FeedAnswerResultResponse]](t, resp).Data
	assert.True(t, again.Replayed)
	assert.Equal(t, first.Data.Answer.AnswerID, again.Answer.AnswerID)

	assert.Equal(t, "IDEMPOTENCY_KEY_REUSED", errorCode(t, r.post("Q-0001", "key-1", "b", "Other"), http.StatusConflict))
	assert.Equal(t, "QUESTION_ALREADY_ANSWERED",
		errorCode(t, r.post("Q-0001", "key-2", "b", "Other"), http.StatusConflict))
}

func TestOwnerQuestion_TwoDifferentAnswersAtOnceAcceptOne(t *testing.T) {
	r := newLeadRig(t)
	r.ask()
	require.Zero(t, r.sync())

	statuses := make([]int, 2)
	var wg sync.WaitGroup
	for i, option := range []string{"a", "b"} {
		wg.Go(func() {
			resp := r.post("Q-0001", "key-"+option, option, "Synthetic "+option)
			statuses[i] = resp.StatusCode
			_ = resp.Body.Close()
		})
	}
	wg.Wait()
	assert.ElementsMatch(t, []int{http.StatusCreated, http.StatusConflict}, statuses)

	require.Zero(t, r.sync())
	assert.Len(t, r.journal("answer"), 1)
}

func TestOwnerQuestion_AnswerAfterResolveElsewhereIsDeliveredAsLate(t *testing.T) {
	r := newLeadRig(t)
	r.ask()
	require.Zero(t, r.sync())
	r.mustLeadTask("", "resolve", "Q-0001", "--elsewhere")

	AssertStatus(t, r.post("Q-0001", "key-1", "a", "Synthetic late answer"), http.StatusCreated)

	require.Zero(t, r.sync())
	require.Len(t, r.journal("answer"), 1)
	assert.Equal(t, true, r.journal("answer")[0]["late"])
	require.Len(t, r.mailbox(), 1)
	assert.Equal(t, true, r.mailbox()[0]["late"])

	require.Zero(t, r.sync())
	handled := r.question("Q-0001")
	assert.Equal(t, stateHandled, handled.State)
	require.NotNil(t, handled.Answer)
	assert.True(t, handled.Answer.Late)

	assert.Equal(t, "QUESTION_ALREADY_ANSWERED",
		errorCode(t, r.post("Q-0001", "key-2", "b", ""), http.StatusConflict))
}

// closedPortURI points at a loopback port nobody listens on, with a short server selection timeout.
func closedPortURI(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())
	return "mongodb://" + addr + "/?serverSelectionTimeoutMS=500&connectTimeoutMS=500"
}
