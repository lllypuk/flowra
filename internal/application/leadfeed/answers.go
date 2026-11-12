package leadfeed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"time"
)

// Answer store errors.
var (
	ErrAnswerNotFound = errors.New("owner answer not found")
	// ErrAlreadyAnswered means the question has an answer under another idempotency key.
	ErrAlreadyAnswered = errors.New("question is already answered")
	// ErrIdempotencyKeyReused means the key was used with another request body.
	ErrIdempotencyKeyReused = errors.New("idempotency key reused with another body")
)

// OwnerAnswer is the owner's answer to one question, with everything needed to deliver it without a snapshot.
// Identity is (Source, Epoch, QID); the feed revision is deliberately not part of it.
type OwnerAnswer struct {
	Source      string
	Epoch       string
	QID         string
	RecordID    string
	QuestionFP  string
	Task        string
	AnswerID    string
	OptionID    *string
	OptionLabel *string
	Text        string
	Scope       string
	AnsweredAt  time.Time
	CommandID   string
	CommandFP   string
}

// AnswerOutcome says what saving an answer did.
type AnswerOutcome string

const (
	// AnswerCreated means the answer was stored by this call.
	AnswerCreated AnswerOutcome = "created"
	// AnswerRepeated means the same key and body had already stored it.
	AnswerRepeated AnswerOutcome = "repeated"
)

// SaveAnswerResult carries the stored answer: the new one, or the original on a repeat.
type SaveAnswerResult struct {
	Outcome AnswerOutcome
	Answer  OwnerAnswer
}

// AnswerStore keeps owner answers outside the feed projection; an import never touches it.
type AnswerStore interface {
	// SaveAnswer checks the idempotency key before the question state: a repeat returns the original answer.
	// It returns ErrIdempotencyKeyReused or ErrAlreadyAnswered.
	SaveAnswer(ctx context.Context, answer OwnerAnswer) (SaveAnswerResult, error)
	// FindAnswerByKey returns the answer stored under an idempotency key, in any epoch, or ErrAnswerNotFound.
	FindAnswerByKey(ctx context.Context, commandID string) (*OwnerAnswer, error)
	// FindAnswer returns ErrAnswerNotFound when the question has no answer.
	FindAnswer(ctx context.Context, source, epoch, qid string) (*OwnerAnswer, error)
	// ListAnswers returns the answers of one epoch, oldest first.
	ListAnswers(ctx context.Context, source, epoch string) ([]OwnerAnswer, error)
	// ListAnswersOtherEpochs returns the answers of the source given under any other epoch, oldest first.
	ListAnswersOtherEpochs(ctx context.Context, source, epoch string) ([]OwnerAnswer, error)
}

// AnswerFingerprint hashes the request body that an idempotency key is bound to.
func AnswerFingerprint(qid string, optionID *string, text string) string {
	h := sha256.New()
	option := ""
	if optionID != nil {
		option = "1" + *optionID
	}
	for _, part := range []string{qid, option, text} {
		h.Write([]byte(strconv.Itoa(len(part)) + ":" + part))
	}
	return hex.EncodeToString(h.Sum(nil))
}
