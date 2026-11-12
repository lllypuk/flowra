package leadfeed_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
)

type sentCall struct {
	qid     string
	payload map[string]any
}

type fakeSender struct {
	calls []sentCall
	err   error
}

func (s *fakeSender) SendAnswer(_ context.Context, qid string, payload []byte) error {
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return err
	}
	s.calls = append(s.calls, sentCall{qid: qid, payload: decoded})
	return s.err
}

type fakeRecorder struct {
	saved []leadfeed.Delivery
	err   error
}

func (r *fakeRecorder) RecordDelivery(_ context.Context, d leadfeed.Delivery) error {
	r.saved = append(r.saved, d)
	return r.err
}

type otherEpochAnswers struct {
	*memAnswers

	other []leadfeed.OwnerAnswer
}

func (o otherEpochAnswers) ListAnswersOtherEpochs(context.Context, string, string) ([]leadfeed.OwnerAnswer, error) {
	return o.other, nil
}

func deliveryFixture(t *testing.T) (*memFeed, *memAnswers, *fakeSender, *fakeRecorder, *leadfeed.Deliverer) {
	t.Helper()
	feed, answers, _ := questionFixture(t)
	sender, recorder := &fakeSender{}, &fakeRecorder{}
	now := func() time.Time { return time.Date(2026, 1, 10, 12, 10, 0, 0, time.UTC) }
	return feed, answers, sender, recorder, leadfeed.NewDeliverer(feed, answers, sender, recorder, now)
}

func storeAnswer(
	t *testing.T,
	feed *memFeed,
	answers *memAnswers,
	qid, answerID string,
	at time.Time,
) leadfeed.OwnerAnswer {
	t.Helper()
	label := "Option A"
	var rec leadfeed.Record
	for _, recs := range feed.records {
		for i := range recs {
			if recs[i].Question != nil && recs[i].Question.QID == qid {
				rec = recs[i]
			}
		}
	}
	require.NotNil(t, rec.Question, qid)
	res, err := answers.SaveAnswer(context.Background(), leadfeed.OwnerAnswer{
		Source: "lead-task", Epoch: "e1", QID: qid, RecordID: rec.ID, QuestionFP: rec.Question.FP, Task: rec.Task,
		AnswerID: answerID, OptionID: optionA(), OptionLabel: &label, Text: "go <b>", Scope: rec.Question.Scope,
		AnsweredAt: at, CommandID: "key-" + qid, CommandFP: "fp-" + qid,
	})
	require.NoError(t, err)
	return res.Answer
}

func confirm(feed *memFeed, answerID string, delivered bool) {
	for _, recs := range feed.records {
		for i := range recs {
			if q := recs[i].Question; q != nil && q.QID == "Q-0001" {
				q.Answer = &leadfeed.Answer{AnswerID: answerID, Delivered: delivered}
			}
		}
	}
}

func TestDeliverer_SendsStoredDataNotTheSnapshot(t *testing.T) {
	feed, answers, sender, recorder, deliverer := deliveryFixture(t)
	at := time.Date(2026, 1, 10, 12, 4, 5, 123_000_000, time.UTC)
	stored := storeAnswer(t, feed, answers, "Q-0001", "ans-1", at)

	out, err := deliverer.Run(context.Background())

	require.NoError(t, err)
	require.Len(t, sender.calls, 1)
	assert.Equal(t, "Q-0001", sender.calls[0].qid)
	assert.Equal(t, map[string]any{
		"source": "lead-task", "epoch": "e1", "qid": "Q-0001", "task": "T-0001", "answer_id": "ans-1",
		"question_fp": stored.QuestionFP, "option_id": "a", "option_label": "Option A", "text": "go <b>",
		"answered_at": "2026-01-10T12:04:05.123Z", "scope": stored.Scope,
	}, sender.calls[0].payload)
	assert.Equal(t, 1, out.Sent)
	assert.Equal(t, 1, out.Unconfirmed, "the snapshot confirms it only after the next import")
	require.Len(t, recorder.saved, 1)
	assert.Equal(t, out, recorder.saved[0])
	require.NotNil(t, out.OldestUnconfirmedAt)
	assert.True(t, out.OldestUnconfirmedAt.Equal(at))
}

func TestDeliverer_ConfirmedAnswerIsNotSentAgain(t *testing.T) {
	feed, answers, sender, _, deliverer := deliveryFixture(t)
	storeAnswer(t, feed, answers, "Q-0001", "ans-1", time.Date(2026, 1, 10, 12, 4, 0, 0, time.UTC))
	confirm(feed, "ans-1", true)

	out, err := deliverer.Run(context.Background())

	require.NoError(t, err)
	assert.Empty(t, sender.calls)
	assert.Zero(t, out.Unconfirmed)
	assert.Nil(t, out.OldestUnconfirmedAt)
}

func TestDeliverer_OnlyTheSameAnswerIDConfirms(t *testing.T) {
	tests := []struct {
		name string
		mark func(feed *memFeed)
	}{
		{"snapshot holds another answer_id", func(f *memFeed) { confirm(f, "someone-else", true) }},
		{"same answer_id not yet marked delivered", func(f *memFeed) { confirm(f, "ans-1", false) }},
		{"question closed without a confirmation", func(f *memFeed) {
			for _, recs := range f.records {
				for i := range recs {
					if q := recs[i].Question; q != nil && q.QID == "Q-0001" {
						q.Resolved = &leadfeed.Resolved{At: queryAsOf()}
					}
				}
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feed, answers, sender, _, deliverer := deliveryFixture(t)
			storeAnswer(t, feed, answers, "Q-0001", "ans-1", time.Date(2026, 1, 10, 12, 4, 0, 0, time.UTC))
			tt.mark(feed)

			out, err := deliverer.Run(context.Background())

			require.NoError(t, err)
			assert.Len(t, sender.calls, 1, "the answer stays unconfirmed and is passed on again")
			assert.Equal(t, 1, out.Unconfirmed)
		})
	}
}

func TestDeliverer_FailedCallIsCountedAndTheRestGoOn(t *testing.T) {
	feed, answers, sender, recorder, deliverer := deliveryFixture(t)
	storeAnswer(t, feed, answers, "Q-0001", "ans-1", time.Date(2026, 1, 10, 12, 4, 0, 0, time.UTC))
	storeAnswer(t, feed, answers, "Q-0002", "ans-2", time.Date(2026, 1, 10, 12, 3, 0, 0, time.UTC))
	sender.err = errors.New("lead-task answer exited with 3: no panel")

	out, err := deliverer.Run(context.Background())

	require.NoError(t, err)
	assert.Len(t, sender.calls, 2)
	assert.Equal(t, 2, out.Failed)
	assert.Zero(t, out.Sent)
	assert.Equal(t, 2, out.Unconfirmed)
	assert.Contains(t, out.LastError, "exited with 3")
	require.NotNil(t, out.OldestUnconfirmedAt)
	assert.True(t, out.OldestUnconfirmedAt.Equal(time.Date(2026, 1, 10, 12, 3, 0, 0, time.UTC)))
	require.Len(t, recorder.saved, 1)
}

func TestDeliverer_QuestionMissingFromSnapshotIsAReconciliationError(t *testing.T) {
	feed, answers, sender, _, deliverer := deliveryFixture(t)
	storeAnswer(t, feed, answers, "Q-0001", "ans-1", time.Date(2026, 1, 10, 12, 4, 0, 0, time.UTC))
	feed.records["T-0001"] = nil

	out, err := deliverer.Run(context.Background())

	require.NoError(t, err)
	assert.Empty(t, sender.calls)
	assert.Equal(t, 1, out.Failed)
	assert.Equal(t, 1, out.Unconfirmed)
	assert.Contains(t, out.LastError, "not in the published snapshot")
	assert.Len(t, answers.byQuestion, 1, "the answer stays in the store")
}

func TestDeliverer_AnswersOfAnotherEpochAreReportedNotSent(t *testing.T) {
	feed, answers, sender, recorder, _ := deliveryFixture(t)
	storeAnswer(t, feed, answers, "Q-0001", "ans-1", time.Date(2026, 1, 10, 12, 4, 0, 0, time.UTC))
	confirm(feed, "ans-1", true)
	old := leadfeed.OwnerAnswer{Source: "lead-task", Epoch: "old", QID: "Q-0001", AnswerID: "ans-old"}
	deliverer := leadfeed.NewDeliverer(
		feed,
		otherEpochAnswers{answers, []leadfeed.OwnerAnswer{old}},
		sender,
		recorder,
		nil,
	)

	out, err := deliverer.Run(context.Background())

	require.NoError(t, err)
	assert.Empty(t, sender.calls)
	assert.Equal(t, 1, out.OtherEpoch)
	assert.Zero(t, out.Failed, "an answer that can never be delivered must not turn every timer run red")
	assert.Contains(t, out.LastError, "earlier journal epoch")
}

func TestDeliverer_NothingPublishedRecordsAnEmptyRun(t *testing.T) {
	feed, _, sender, recorder, deliverer := deliveryFixture(t)
	feed.state = nil

	out, err := deliverer.Run(context.Background())

	require.NoError(t, err)
	assert.Empty(t, sender.calls)
	assert.Zero(t, out.Unconfirmed)
	require.Len(t, recorder.saved, 1)
}

func TestDeliverer_RecorderFailureIsReturned(t *testing.T) {
	_, _, _, recorder, deliverer := deliveryFixture(t)
	recorder.err = errors.New("db down")

	_, err := deliverer.Run(context.Background())

	require.ErrorContains(t, err, "db down")
}

func TestDeliverer_LongErrorIsClipped(t *testing.T) {
	feed, answers, sender, _, deliverer := deliveryFixture(t)
	storeAnswer(t, feed, answers, "Q-0001", "ans-1", time.Date(2026, 1, 10, 12, 4, 0, 0, time.UTC))
	sender.err = errors.New(string(make([]rune, 2000)))

	out, err := deliverer.Run(context.Background())

	require.NoError(t, err)
	assert.LessOrEqual(t, len([]rune(out.LastError)), 301)
}
