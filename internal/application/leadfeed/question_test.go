package leadfeed_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
	"github.com/lllypuk/flowra/tests/fixtures"
)

type memAnswers struct {
	byQuestion map[string]leadfeed.OwnerAnswer
}

func newMemAnswers() *memAnswers { return &memAnswers{byQuestion: map[string]leadfeed.OwnerAnswer{}} }

func (m *memAnswers) id(source, epoch, qid string) string { return source + ":" + epoch + ":" + qid }

func (m *memAnswers) SaveAnswer(_ context.Context, a leadfeed.OwnerAnswer) (leadfeed.SaveAnswerResult, error) {
	if prior, err := m.FindAnswerByKey(context.Background(), a.CommandID); err == nil {
		if prior.CommandFP != a.CommandFP {
			return leadfeed.SaveAnswerResult{}, leadfeed.ErrIdempotencyKeyReused
		}
		return leadfeed.SaveAnswerResult{Outcome: leadfeed.AnswerRepeated, Answer: *prior}, nil
	}
	if _, ok := m.byQuestion[m.id(a.Source, a.Epoch, a.QID)]; ok {
		return leadfeed.SaveAnswerResult{}, leadfeed.ErrAlreadyAnswered
	}
	m.byQuestion[m.id(a.Source, a.Epoch, a.QID)] = a
	return leadfeed.SaveAnswerResult{Outcome: leadfeed.AnswerCreated, Answer: a}, nil
}

func (m *memAnswers) FindAnswerByKey(_ context.Context, key string) (*leadfeed.OwnerAnswer, error) {
	for _, a := range m.byQuestion {
		if a.CommandID == key {
			return &a, nil
		}
	}
	return nil, leadfeed.ErrAnswerNotFound
}

func (m *memAnswers) FindAnswer(_ context.Context, source, epoch, qid string) (*leadfeed.OwnerAnswer, error) {
	if a, ok := m.byQuestion[m.id(source, epoch, qid)]; ok {
		return &a, nil
	}
	return nil, leadfeed.ErrAnswerNotFound
}

func (m *memAnswers) ListAnswers(_ context.Context, source, epoch string) ([]leadfeed.OwnerAnswer, error) {
	var out []leadfeed.OwnerAnswer
	for _, a := range m.byQuestion {
		if a.Source == source && a.Epoch == epoch {
			out = append(out, a)
		}
	}
	return out, nil
}

func (m *memAnswers) ListAnswersOtherEpochs(context.Context, string, string) ([]leadfeed.OwnerAnswer, error) {
	return nil, nil
}

func questionFixture(t *testing.T) (*memFeed, *memAnswers, *leadfeed.QuestionService) {
	t.Helper()
	feed := newMemFeed(2)
	feed.tasks[0].Title = "Synthetic task"
	feed.records["T-0001"] = append(feed.records["T-0001"],
		fixtures.LeadFeedQuestion("j:q1", "T-0001", "Q-0001", 10),
		fixtures.LeadFeedQuestion("j:q2", "T-0001", "Q-0002", 11),
	)
	answers := newMemAnswers()
	now := time.Date(2026, 1, 10, 12, 5, 0, 0, time.UTC)
	clock := leadfeed.WithQuestionClock(func() time.Time { return now })
	return feed, answers, leadfeed.NewQuestionService(feed, answers, clock)
}

func q1(feed *memFeed) *leadfeed.Question {
	for _, recs := range feed.records {
		for i := range recs {
			if recs[i].Question != nil && recs[i].Question.QID == "Q-0001" {
				return recs[i].Question
			}
		}
	}
	return nil
}

func optionA() *string { a := "a"; return &a }

func TestQuestionService_AnswerAccepted(t *testing.T) {
	feed, answers, svc := questionFixture(t)

	res, err := svc.Answer(
		context.Background(),
		leadfeed.AnswerCommand{QID: "Q-0001", OptionID: optionA(), Text: "go", Key: "k1"},
	)
	require.NoError(t, err)

	assert.Equal(t, leadfeed.AnswerCreated, res.Outcome)
	got := res.Answer
	assert.Equal(t, "lead-task", got.Source)
	assert.Equal(t, "e1", got.Epoch)
	assert.Equal(t, "j:q1", got.RecordID)
	assert.Equal(t, q1(feed).FP, got.QuestionFP)
	assert.Equal(t, "T-0001", got.Task)
	assert.Equal(t, "Option A", *got.OptionLabel)
	assert.Equal(t, q1(feed).Scope, got.Scope)
	assert.NotEmpty(t, got.AnswerID)
	assert.Equal(t, "k1", got.CommandID)
	assert.Equal(t, leadfeed.AnswerFingerprint("Q-0001", optionA(), "go"), got.CommandFP)
	assert.Len(t, answers.byQuestion, 1)
}

func TestQuestionService_AnswerRejected(t *testing.T) {
	missing := "zzz"
	tests := []struct {
		name  string
		setup func(feed *memFeed, answers *memAnswers)
		cmd   leadfeed.AnswerCommand
		want  error
	}{
		{
			name: "no key",
			cmd:  leadfeed.AnswerCommand{QID: "Q-0001", Text: "x"},
			want: leadfeed.ErrIdempotencyKeyRequired,
		},
		{
			name: "empty body",
			cmd:  leadfeed.AnswerCommand{QID: "Q-0001", Text: "  ", Key: "k"},
			want: leadfeed.ErrInvalidAnswer,
		},
		{
			name: "text too long",
			cmd: leadfeed.AnswerCommand{
				QID:  "Q-0001",
				Text: strings.Repeat("я", leadfeed.MaxAnswerTextRunes+1),
				Key:  "k",
			},
			want: leadfeed.ErrInvalidAnswer,
		},
		{
			name: "unknown option",
			cmd:  leadfeed.AnswerCommand{QID: "Q-0001", OptionID: &missing, Key: "k"},
			want: leadfeed.ErrInvalidAnswer,
		},
		{
			name: "no such question",
			cmd:  leadfeed.AnswerCommand{QID: "Q-9999", Text: "x", Key: "k"},
			want: leadfeed.ErrQuestionNotFound,
		},
		{
			name: "answer in the snapshot only",
			setup: func(feed *memFeed, _ *memAnswers) {
				q1(feed).Answer = &leadfeed.Answer{AnswerID: "ext"}
			},
			cmd:  leadfeed.AnswerCommand{QID: "Q-0001", Text: "x", Key: "k"},
			want: leadfeed.ErrAlreadyAnswered,
		},
		{
			name: "answered in Flowra under another key",
			setup: func(_ *memFeed, answers *memAnswers) {
				answers.byQuestion["lead-task:e1:Q-0001"] = leadfeed.OwnerAnswer{
					Source: "lead-task", Epoch: "e1", QID: "Q-0001", RecordID: "j:q1", CommandID: "other",
				}
			},
			cmd:  leadfeed.AnswerCommand{QID: "Q-0001", Text: "x", Key: "k"},
			want: leadfeed.ErrAlreadyAnswered,
		},
		{
			name: "closed without an answer",
			setup: func(feed *memFeed, _ *memAnswers) {
				q1(feed).Resolved = &leadfeed.Resolved{Elsewhere: true}
			},
			cmd:  leadfeed.AnswerCommand{QID: "Q-0001", Text: "x", Key: "k"},
			want: leadfeed.ErrQuestionClosed,
		},
		{
			name: "record id changed under the answer",
			setup: func(_ *memFeed, answers *memAnswers) {
				answers.byQuestion["lead-task:e1:Q-0001"] = leadfeed.OwnerAnswer{
					Source: "lead-task", Epoch: "e1", QID: "Q-0001", RecordID: "j:old", CommandID: "other",
				}
			},
			cmd:  leadfeed.AnswerCommand{QID: "Q-0001", Text: "x", Key: "k"},
			want: leadfeed.ErrQuestionNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feed, answers, svc := questionFixture(t)
			if tt.setup != nil {
				tt.setup(feed, answers)
			}
			_, err := svc.Answer(context.Background(), tt.cmd)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestQuestionService_NothingPublished(t *testing.T) {
	feed, _, svc := questionFixture(t)
	feed.state = nil

	_, err := svc.Answer(context.Background(), leadfeed.AnswerCommand{QID: "Q-0001", Text: "x", Key: "k"})
	require.ErrorIs(t, err, leadfeed.ErrQuestionNotFound)

	list, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, list.Items)
	assert.Empty(t, list.Revision)
}

func TestQuestionService_RepeatReturnsFirstResult(t *testing.T) {
	feed, _, svc := questionFixture(t)
	ctx := context.Background()
	cmd := leadfeed.AnswerCommand{QID: "Q-0001", OptionID: optionA(), Text: "go", Key: "k1"}

	first, err := svc.Answer(ctx, cmd)
	require.NoError(t, err)

	q1(feed).Resolved = &leadfeed.Resolved{}
	q1(feed).Answer = &leadfeed.Answer{AnswerID: first.Answer.AnswerID, Delivered: true}
	again, err := svc.Answer(ctx, cmd)
	require.NoError(t, err)
	assert.Equal(t, leadfeed.AnswerRepeated, again.Outcome)
	assert.Equal(t, first.Answer, again.Answer)

	delete(feed.records, "T-0001")
	again, err = svc.Answer(ctx, cmd)
	require.NoError(t, err, "repeat does not need the question in the snapshot")
	assert.Equal(t, first.Answer, again.Answer)

	cmd.Text = "other"
	_, err = svc.Answer(ctx, cmd)
	require.ErrorIs(t, err, leadfeed.ErrIdempotencyKeyReused)

	cmd.QID, cmd.Text = "Q-0002", "go"
	_, err = svc.Answer(ctx, cmd)
	require.ErrorIs(t, err, leadfeed.ErrIdempotencyKeyReused, "the same key on another question")
}

func TestQuestionService_States(t *testing.T) {
	stored := func(id string) *leadfeed.OwnerAnswer {
		return &leadfeed.OwnerAnswer{
			Source: "lead-task", Epoch: "e1", QID: "Q-0001", RecordID: "j:q1", AnswerID: id,
			Text: "go", CommandID: "k",
		}
	}
	snapAnswer := func(id string, delivered bool) *leadfeed.Answer {
		return &leadfeed.Answer{AnswerID: id, Delivered: delivered, Text: "ext"}
	}
	closed := &leadfeed.Resolved{Note: "done"}

	tests := []struct {
		name       string
		own        *leadfeed.OwnerAnswer
		answer     *leadfeed.Answer
		resolved   *leadfeed.Resolved
		state      leadfeed.QuestionState
		closedFlag bool
		delivered  bool
	}{
		{name: "waiting", state: leadfeed.QuestionWaiting},
		{name: "closed without an answer", resolved: closed, state: leadfeed.QuestionHandled},
		{name: "answered, not in the snapshot", own: stored("a1"), state: leadfeed.QuestionAnswered},
		{
			name: "answered, snapshot has it undelivered", own: stored("a1"), answer: snapAnswer("a1", false),
			state: leadfeed.QuestionAnswered,
		},
		{
			name: "foreign answer id does not confirm", own: stored("a1"), answer: snapAnswer("zzz", true),
			state: leadfeed.QuestionAnswered,
		},
		{
			name: "answered, closed in the session", own: stored("a1"), resolved: closed,
			state: leadfeed.QuestionAnswered, closedFlag: true,
		},
		{
			name: "delivered", own: stored("a1"), answer: snapAnswer("a1", true), state: leadfeed.QuestionDelivered,
			delivered: true,
		},
		{
			name: "handled", own: stored("a1"), answer: snapAnswer("a1", true), resolved: closed,
			state: leadfeed.QuestionHandled, delivered: true,
		},
		{
			name: "answer only in the snapshot", answer: snapAnswer("ext", true), state: leadfeed.QuestionDelivered,
			delivered: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feed, answers, svc := questionFixture(t)
			q := q1(feed)
			q.Answer, q.Resolved = tt.answer, tt.resolved
			if tt.own != nil {
				answers.byQuestion["lead-task:e1:Q-0001"] = *tt.own
			}

			detail, err := svc.Question(context.Background(), "Q-0001")
			require.NoError(t, err)
			got := detail.Question
			assert.Equal(t, tt.state, got.State)
			assert.Equal(t, tt.closedFlag, got.ClosedInSession)
			assert.Equal(t, "Synthetic task", got.TaskTitle)
			if tt.own == nil && tt.answer == nil {
				assert.Nil(t, got.Answer)
				return
			}
			require.NotNil(t, got.Answer)
			assert.Equal(t, tt.delivered, got.Answer.Delivered)
		})
	}
}

func TestQuestionService_ListPutsWaitingFirst(t *testing.T) {
	feed, answers, svc := questionFixture(t)
	answers.byQuestion["lead-task:e1:Q-0002"] = leadfeed.OwnerAnswer{
		Source: "lead-task", Epoch: "e1", QID: "Q-0002", RecordID: "j:q2", AnswerID: "a2", CommandID: "k",
	}
	feed.records["T-0002"] = append(feed.records["T-0002"], fixtures.LeadFeedQuestion("j:q3", "T-0002", "Q-0003", 1))

	list, err := svc.List(context.Background())
	require.NoError(t, err)

	require.Len(t, list.Items, 3)
	assert.Equal(t, 2, list.Waiting)
	var order []string
	for _, item := range list.Items {
		order = append(order, item.Record.Question.QID)
	}
	assert.Equal(t, []string{"Q-0001", "Q-0003", "Q-0002"}, order, "waiting first, newest first within a group")
	assert.Equal(t, leadfeed.QuestionAnswered, list.Items[2].State)
	assert.Equal(t, "rev-a", list.Revision)
}

func TestQuestionService_ListHidesQuestionWithForeignAnswerRecord(t *testing.T) {
	_, answers, svc := questionFixture(t)
	answers.byQuestion["lead-task:e1:Q-0001"] = leadfeed.OwnerAnswer{
		Source: "lead-task", Epoch: "e1", QID: "Q-0001", RecordID: "j:old", CommandID: "k",
	}

	list, err := svc.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	assert.Equal(t, "Q-0002", list.Items[0].Record.Question.QID)

	_, err = svc.Question(context.Background(), "Q-0001")
	require.ErrorIs(t, err, leadfeed.ErrQuestionNotFound)
}
