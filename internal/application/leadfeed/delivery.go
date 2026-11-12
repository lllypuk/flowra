package leadfeed

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"
)

const (
	// answerTimeLayout is the millisecond UTC form the source uses for its own timestamps.
	answerTimeLayout = "2006-01-02T15:04:05.000Z"
	maxDeliveryError = 300
)

// Delivery is the outcome of the last delivery run; it is kept apart from the import attempt.
type Delivery struct {
	At time.Time
	// Sent counts calls the source accepted; the snapshot confirms them one import later.
	Sent   int
	Failed int
	// Unconfirmed counts the answers of the current epoch the published snapshot does not confirm.
	Unconfirmed         int
	OldestUnconfirmedAt *time.Time
	// OtherEpoch counts answers given under a journal that is no longer the source; they are never delivered.
	OtherEpoch int
	LastError  string
}

// DeliveryRecorder stores the outcome of a delivery run.
type DeliveryRecorder interface {
	RecordDelivery(ctx context.Context, delivery Delivery) error
}

// AnswerSender hands one answer payload to the source; any error leaves the answer unconfirmed.
type AnswerSender interface {
	SendAnswer(ctx context.Context, qid string, payload []byte) error
}

// Deliverer passes unconfirmed owner answers to the source, at least once.
type Deliverer struct {
	reader   Reader
	answers  AnswerStore
	sender   AnswerSender
	recorder DeliveryRecorder
	now      func() time.Time
}

// NewDeliverer creates a deliverer.
func NewDeliverer(
	reader Reader,
	answers AnswerStore,
	sender AnswerSender,
	recorder DeliveryRecorder,
	now func() time.Time,
) *Deliverer {
	if now == nil {
		now = time.Now
	}
	return &Deliverer{reader: reader, answers: answers, sender: sender, recorder: recorder, now: now}
}

type answerPayload struct {
	Source      string  `json:"source"`
	Epoch       string  `json:"epoch"`
	QID         string  `json:"qid"`
	Task        string  `json:"task"`
	AnswerID    string  `json:"answer_id"`
	QuestionFP  string  `json:"question_fp"`
	OptionID    *string `json:"option_id"`
	OptionLabel *string `json:"option_label"`
	Text        string  `json:"text"`
	AnsweredAt  string  `json:"answered_at"`
	Scope       string  `json:"scope"`
}

// Run delivers every answer the published snapshot does not confirm and records the outcome.
// The error is set only when the feed or the delivery state cannot be read or written.
func (d *Deliverer) Run(ctx context.Context) (Delivery, error) {
	state, questions, err := d.readPublished(ctx)
	if err != nil {
		return Delivery{}, err
	}
	out := Delivery{At: d.now().UTC()}
	if state == nil {
		return out, d.record(ctx, out)
	}

	other, err := d.answers.ListAnswersOtherEpochs(ctx, state.Source, state.Epoch)
	if err != nil {
		return Delivery{}, fmt.Errorf("list answers of other epochs: %w", err)
	}
	if out.OtherEpoch = len(other); out.OtherEpoch > 0 {
		out.LastError = fmt.Sprintf("%d answers belong to an earlier journal epoch and cannot be delivered", len(other))
	}
	answers, err := d.answers.ListAnswers(ctx, state.Source, state.Epoch)
	if err != nil {
		return Delivery{}, fmt.Errorf("list owner answers: %w", err)
	}

	for i := range answers {
		answer := &answers[i]
		rec := questions[answer.QID]
		if rec != nil && rec.ID == answer.RecordID && isConfirmed(rec.Question, answer.AnswerID) {
			continue
		}
		out.Unconfirmed++
		if out.OldestUnconfirmedAt == nil || answer.AnsweredAt.Before(*out.OldestUnconfirmedAt) {
			at := answer.AnsweredAt
			out.OldestUnconfirmedAt = &at
		}
		if ctx.Err() != nil {
			return Delivery{}, ctx.Err()
		}
		if rec == nil || rec.ID != answer.RecordID {
			out.Failed++
			out.LastError = clip(fmt.Sprintf("%s: the question is not in the published snapshot", answer.QID))
			continue
		}
		if sendErr := d.send(ctx, answer); sendErr != nil {
			out.Failed++
			out.LastError = clip(fmt.Sprintf("%s: %v", answer.QID, sendErr))
			continue
		}
		out.Sent++
	}
	return out, d.record(ctx, out)
}

func (d *Deliverer) record(ctx context.Context, out Delivery) error {
	if err := d.recorder.RecordDelivery(ctx, out); err != nil {
		return fmt.Errorf("record delivery state: %w", err)
	}
	return nil
}

func (d *Deliverer) send(ctx context.Context, a *OwnerAnswer) error {
	payload, err := json.Marshal(answerPayload{
		Source: a.Source, Epoch: a.Epoch, QID: a.QID, Task: a.Task, AnswerID: a.AnswerID, QuestionFP: a.QuestionFP,
		OptionID: a.OptionID, OptionLabel: a.OptionLabel, Text: a.Text,
		AnsweredAt: a.AnsweredAt.UTC().Format(answerTimeLayout), Scope: a.Scope,
	})
	if err != nil {
		return fmt.Errorf("encode answer: %w", err)
	}
	return d.sender.SendAnswer(ctx, a.QID, payload)
}

func (d *Deliverer) readPublished(ctx context.Context) (*State, map[string]*Record, error) {
	var (
		state     *State
		questions map[string]*Record
	)
	err := d.reader.Read(ctx, func(ctx context.Context, view View) error {
		var err error
		questions = map[string]*Record{}
		if state, err = view.State(ctx); err != nil || state == nil {
			return err
		}
		recs, err := view.Questions(ctx)
		if err != nil {
			return err
		}
		for i := range recs {
			questions[recs[i].Question.QID] = &recs[i]
		}
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("read published feed: %w", err)
	}
	return state, questions, nil
}

// isConfirmed: only the same answer_id marked delivered counts; a closed question or another answer does not.
func isConfirmed(q *Question, answerID string) bool {
	return q != nil && q.Answer != nil && q.Answer.AnswerID == answerID && q.Answer.Delivered
}

func clip(s string) string {
	if utf8.RuneCountInString(s) <= maxDeliveryError {
		return s
	}
	return string([]rune(s)[:maxDeliveryError]) + "…"
}
