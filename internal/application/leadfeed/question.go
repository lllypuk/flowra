package leadfeed

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// MaxAnswerTextRunes bounds the free text of an answer.
const MaxAnswerTextRunes = 4000

// Question service errors.
var (
	// ErrQuestionClosed means the lead closed the question without an answer.
	ErrQuestionClosed = errors.New("question is closed")
	// ErrInvalidAnswer means the answer body is unusable; the message says why.
	ErrInvalidAnswer = errors.New("invalid answer")
	// ErrIdempotencyKeyRequired means an answer arrived without a key.
	ErrIdempotencyKeyRequired = errors.New("idempotency key is required")
)

// QuestionState is where a question stands on its way from the lead to the owner and back.
type QuestionState string

// Question states, see the state table of the plan.
const (
	// QuestionWaiting: no answer anywhere, not closed.
	QuestionWaiting QuestionState = "waiting"
	// QuestionAnswered: Flowra holds an answer the lead's journal has not confirmed yet.
	QuestionAnswered QuestionState = "answered"
	// QuestionDelivered: the journal confirmed the answer and the lead's mailbox accepted it.
	QuestionDelivered QuestionState = "delivered"
	// QuestionHandled: the lead closed the question and any answer is confirmed.
	QuestionHandled QuestionState = "handled"
)

// AnswerView is the answer shown with a question: the stored one, else the one the snapshot carries.
type AnswerView struct {
	AnswerID    string
	OptionID    *string
	OptionLabel *string
	Text        string
	AnsweredAt  time.Time
	Late        bool
	// Delivered is true once the snapshot confirms this very answer.
	Delivered bool
}

// QuestionView is a question of the published snapshot with the owner's answer laid over it.
type QuestionView struct {
	Record    Record
	TaskTitle string
	State     QuestionState
	// ClosedInSession is set when the lead closed the question while an answer is still unconfirmed.
	ClosedInSession bool
	Answer          *AnswerView
}

// QuestionList is every question of one snapshot, those waiting for an answer first.
type QuestionList struct {
	Meta

	Items   []QuestionView
	Waiting int
}

// QuestionDetail is one question from a single snapshot.
type QuestionDetail struct {
	Meta

	Question QuestionView
}

// AnswerCommand is one owner answer to accept.
type AnswerCommand struct {
	QID      string
	OptionID *string
	Text     string
	Key      string
}

// QuestionService shows questions with their answers and accepts the owner's answer.
type QuestionService struct {
	reader  Reader
	answers AnswerStore
	now     func() time.Time
}

// QuestionOption configures a QuestionService.
type QuestionOption func(*QuestionService)

// WithQuestionClock replaces the clock that stamps accepted answers.
func WithQuestionClock(now func() time.Time) QuestionOption {
	return func(s *QuestionService) { s.now = now }
}

// NewQuestionService creates a question service.
func NewQuestionService(reader Reader, answers AnswerStore, opts ...QuestionOption) *QuestionService {
	s := &QuestionService{reader: reader, answers: answers, now: time.Now}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// List returns the questions of the published snapshot.
func (s *QuestionService) List(ctx context.Context) (QuestionList, error) {
	state, recs, tasks, err := s.readQuestions(ctx)
	if err != nil {
		return QuestionList{}, err
	}
	out := QuestionList{Meta: metaOf(state), Items: []QuestionView{}}
	if state == nil {
		return out, nil
	}

	stored, err := s.answers.ListAnswers(ctx, state.Source, state.Epoch)
	if err != nil {
		return QuestionList{}, fmt.Errorf("list owner answers: %w", err)
	}
	byQID := make(map[string]*OwnerAnswer, len(stored))
	for i := range stored {
		byQID[stored[i].QID] = &stored[i]
	}

	for i := range recs {
		own := byQID[recs[i].Question.QID]
		if own != nil && own.RecordID != recs[i].ID {
			continue
		}
		item := overlay(&recs[i], tasks[recs[i].Task], own)
		if item.State == QuestionWaiting {
			out.Waiting++
		}
		out.Items = append(out.Items, item)
	}
	slices.SortStableFunc(out.Items, compareQuestions)
	return out, nil
}

func (s *QuestionService) readQuestions(ctx context.Context) (*State, []Record, map[string]string, error) {
	var (
		state *State
		recs  []Record
		tasks map[string]string
	)
	err := s.reader.Read(ctx, func(ctx context.Context, view View) error {
		var err error
		recs, tasks = nil, map[string]string{}
		if state, err = view.State(ctx); err != nil || state == nil {
			return err
		}
		if recs, err = view.Questions(ctx); err != nil {
			return err
		}
		for i := range recs {
			task, taskErr := view.Task(ctx, recs[i].Task)
			if taskErr != nil {
				return taskErr
			}
			tasks[recs[i].Task] = task.Title
		}
		return nil
	})
	return state, recs, tasks, err
}

func compareQuestions(a, b QuestionView) int {
	aw, bw := a.State == QuestionWaiting, b.State == QuestionWaiting
	switch {
	case aw && !bw:
		return -1
	case bw && !aw:
		return 1
	default:
		return b.Record.At.Compare(a.Record.At)
	}
}

// Question returns one question or ErrQuestionNotFound.
func (s *QuestionService) Question(ctx context.Context, qid string) (QuestionDetail, error) {
	snap, err := s.readQuestion(ctx, qid)
	if err != nil {
		return QuestionDetail{}, err
	}
	own, err := s.storedAnswer(ctx, snap)
	if err != nil {
		return QuestionDetail{}, err
	}
	return QuestionDetail{Meta: metaOf(snap.state), Question: overlay(snap.record, snap.taskTitle, own)}, nil
}

// Answer accepts the owner's answer. A repeated key returns the first result whatever the question became since.
func (s *QuestionService) Answer(ctx context.Context, cmd AnswerCommand) (SaveAnswerResult, error) {
	if cmd.Key == "" {
		return SaveAnswerResult{}, ErrIdempotencyKeyRequired
	}
	if cmd.OptionID != nil && *cmd.OptionID == "" {
		cmd.OptionID = nil
	}
	if cmd.OptionID == nil && strings.TrimSpace(cmd.Text) == "" {
		return SaveAnswerResult{}, fmt.Errorf("%w: an option or a text is required", ErrInvalidAnswer)
	}
	if utf8.RuneCountInString(cmd.Text) > MaxAnswerTextRunes {
		return SaveAnswerResult{}, fmt.Errorf(
			"%w: text is longer than %d characters", ErrInvalidAnswer, MaxAnswerTextRunes)
	}

	fp := AnswerFingerprint(cmd.QID, cmd.OptionID, cmd.Text)
	prior, err := s.answers.FindAnswerByKey(ctx, cmd.Key)
	switch {
	case err == nil && prior.CommandFP == fp:
		return SaveAnswerResult{Outcome: AnswerRepeated, Answer: *prior}, nil
	case err == nil:
		return SaveAnswerResult{}, ErrIdempotencyKeyReused
	case !errors.Is(err, ErrAnswerNotFound):
		return SaveAnswerResult{}, fmt.Errorf("find answer by key: %w", err)
	}

	snap, err := s.readQuestion(ctx, cmd.QID)
	if err != nil {
		return SaveAnswerResult{}, err
	}
	q := snap.record.Question
	label, err := optionLabel(q, cmd.OptionID)
	if err != nil {
		return SaveAnswerResult{}, err
	}
	own, err := s.storedAnswer(ctx, snap)
	if err != nil {
		return SaveAnswerResult{}, err
	}
	switch {
	case own != nil || q.Answer != nil:
		return SaveAnswerResult{}, ErrAlreadyAnswered
	case q.Resolved != nil:
		return SaveAnswerResult{}, ErrQuestionClosed
	}

	return s.answers.SaveAnswer(ctx, OwnerAnswer{
		Source:      snap.state.Source,
		Epoch:       snap.state.Epoch,
		QID:         q.QID,
		RecordID:    snap.record.ID,
		QuestionFP:  q.FP,
		Task:        snap.record.Task,
		AnswerID:    uuid.NewString(),
		OptionID:    cmd.OptionID,
		OptionLabel: label,
		Text:        cmd.Text,
		Scope:       q.Scope,
		AnsweredAt:  s.now().UTC(),
		CommandID:   cmd.Key,
		CommandFP:   fp,
	})
}

type questionSnapshot struct {
	state     *State
	record    *Record
	taskTitle string
}

func (s *QuestionService) readQuestion(ctx context.Context, qid string) (questionSnapshot, error) {
	var out questionSnapshot
	err := s.reader.Read(ctx, func(ctx context.Context, view View) error {
		out = questionSnapshot{}
		state, err := view.State(ctx)
		if err != nil {
			return err
		}
		if state == nil {
			return ErrQuestionNotFound
		}
		rec, err := view.Question(ctx, qid)
		if err != nil {
			return err
		}
		task, err := view.Task(ctx, rec.Task)
		if err != nil {
			return err
		}
		out = questionSnapshot{state: state, record: rec, taskTitle: task.Title}
		return nil
	})
	return out, err
}

// storedAnswer returns the answer given in Flowra; one that names another record is not this question's.
func (s *QuestionService) storedAnswer(ctx context.Context, snap questionSnapshot) (*OwnerAnswer, error) {
	own, err := s.answers.FindAnswer(ctx, snap.state.Source, snap.state.Epoch, snap.record.Question.QID)
	if errors.Is(err, ErrAnswerNotFound) {
		return nil, nil //nolint:nilnil // no answer is a normal state
	}
	if err != nil {
		return nil, fmt.Errorf("find owner answer: %w", err)
	}
	if own.RecordID != snap.record.ID {
		return nil, ErrQuestionNotFound
	}
	return own, nil
}

func metaOf(state *State) Meta {
	if state == nil {
		return Meta{}
	}
	return Meta{Revision: state.Revision, AsOf: state.AsOf}
}

func optionLabel(q *Question, optionID *string) (*string, error) {
	if optionID == nil {
		return nil, nil //nolint:nilnil // a text-only answer has no option
	}
	for _, opt := range q.Options {
		if opt.ID == *optionID {
			label := opt.Label
			return &label, nil
		}
	}
	return nil, fmt.Errorf("%w: unknown option %q", ErrInvalidAnswer, *optionID)
}

// overlay derives the state; a snapshot answer with another answer_id never confirms the stored one.
func overlay(rec *Record, taskTitle string, own *OwnerAnswer) QuestionView {
	q := rec.Question
	view := QuestionView{Record: *rec, TaskTitle: taskTitle}
	closed := q.Resolved != nil

	switch {
	case own != nil:
		confirmed := isConfirmed(q, own.AnswerID)
		view.Answer = &AnswerView{
			AnswerID: own.AnswerID, OptionID: own.OptionID, OptionLabel: own.OptionLabel, Text: own.Text,
			AnsweredAt: own.AnsweredAt, Late: q.Answer != nil && q.Answer.AnswerID == own.AnswerID && q.Answer.Late,
			Delivered: confirmed,
		}
		view.State = stateOfAnswer(confirmed, closed)
		view.ClosedInSession = closed && !confirmed
	case q.Answer != nil:
		a := q.Answer
		view.Answer = &AnswerView{
			AnswerID: a.AnswerID, OptionID: a.OptionID, OptionLabel: labelOf(q, a.OptionID), Text: a.Text,
			AnsweredAt: a.AnsweredAt, Late: a.Late, Delivered: a.Delivered,
		}
		view.State = stateOfAnswer(a.Delivered, closed)
		view.ClosedInSession = closed && !a.Delivered
	case closed:
		view.State = QuestionHandled
	default:
		view.State = QuestionWaiting
	}
	return view
}

func stateOfAnswer(confirmed, closed bool) QuestionState {
	switch {
	case !confirmed:
		return QuestionAnswered
	case closed:
		return QuestionHandled
	default:
		return QuestionDelivered
	}
}

func labelOf(q *Question, optionID *string) *string {
	label, err := optionLabel(q, optionID)
	if err != nil {
		return nil
	}
	return label
}
