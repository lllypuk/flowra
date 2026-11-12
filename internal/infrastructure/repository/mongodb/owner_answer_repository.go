package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
	mongodbinfra "github.com/lllypuk/flowra/internal/infrastructure/mongodb"
)

// MongoOwnerAnswerRepository stores owner answers; the feed import never replaces this collection.
type MongoOwnerAnswerRepository struct {
	answers *mongo.Collection
}

var _ leadfeed.AnswerStore = (*MongoOwnerAnswerRepository)(nil)

// NewMongoOwnerAnswerRepository creates an owner answer repository.
func NewMongoOwnerAnswerRepository(db *mongo.Database) *MongoOwnerAnswerRepository {
	return &MongoOwnerAnswerRepository{answers: db.Collection(mongodbinfra.CollectionOwnerAnswers)}
}

type ownerAnswerDoc struct {
	ID          string    `bson:"_id"`
	Source      string    `bson:"source"`
	Epoch       string    `bson:"epoch"`
	QID         string    `bson:"qid"`
	RecordID    string    `bson:"record_id"`
	QuestionFP  string    `bson:"question_fp"`
	Task        string    `bson:"task"`
	AnswerID    string    `bson:"answer_id"`
	OptionID    *string   `bson:"option_id"`
	OptionLabel *string   `bson:"option_label"`
	Text        string    `bson:"text"`
	Scope       string    `bson:"scope"`
	AnsweredAt  time.Time `bson:"answered_at"`
	CommandID   string    `bson:"command_id"`
	CommandFP   string    `bson:"command_fp"`
}

func ownerAnswerID(source, epoch, qid string) string {
	return source + ":" + epoch + ":" + qid
}

// SaveAnswer relies on the unique _id for one answer per question and on the unique command_id for the key.
func (r *MongoOwnerAnswerRepository) SaveAnswer(
	ctx context.Context,
	answer leadfeed.OwnerAnswer,
) (leadfeed.SaveAnswerResult, error) {
	if res, found, err := r.byKey(ctx, &answer); found || err != nil {
		return res, err
	}

	doc := ownerAnswerToDoc(&answer)
	_, err := r.answers.InsertOne(ctx, doc)
	if err == nil {
		return leadfeed.SaveAnswerResult{Outcome: leadfeed.AnswerCreated, Answer: ownerAnswerFromDoc(&doc)}, nil
	}
	if !mongo.IsDuplicateKeyError(err) {
		return leadfeed.SaveAnswerResult{}, fmt.Errorf("insert owner answer: %w", err)
	}

	// A concurrent request with the same key may have won the insert; it is a repeat, not a conflict.
	if res, found, keyErr := r.byKey(ctx, &answer); found || keyErr != nil {
		return res, keyErr
	}
	return leadfeed.SaveAnswerResult{}, leadfeed.ErrAlreadyAnswered
}

func (r *MongoOwnerAnswerRepository) byKey(
	ctx context.Context,
	answer *leadfeed.OwnerAnswer,
) (leadfeed.SaveAnswerResult, bool, error) {
	var existing ownerAnswerDoc
	err := r.answers.FindOne(ctx, bson.M{"command_id": answer.CommandID}).Decode(&existing)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return leadfeed.SaveAnswerResult{}, false, nil
	}
	if err != nil {
		return leadfeed.SaveAnswerResult{}, false, fmt.Errorf("find owner answer by key: %w", err)
	}
	if existing.CommandFP != answer.CommandFP {
		return leadfeed.SaveAnswerResult{}, true, leadfeed.ErrIdempotencyKeyReused
	}
	return leadfeed.SaveAnswerResult{Outcome: leadfeed.AnswerRepeated, Answer: ownerAnswerFromDoc(&existing)}, true, nil
}

// FindAnswerByKey returns the answer stored under an idempotency key.
func (r *MongoOwnerAnswerRepository) FindAnswerByKey(
	ctx context.Context,
	commandID string,
) (*leadfeed.OwnerAnswer, error) {
	var doc ownerAnswerDoc
	err := r.answers.FindOne(ctx, bson.M{"command_id": commandID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, leadfeed.ErrAnswerNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find owner answer by key: %w", err)
	}
	answer := ownerAnswerFromDoc(&doc)
	return &answer, nil
}

// FindAnswer returns the answer of one question.
func (r *MongoOwnerAnswerRepository) FindAnswer(
	ctx context.Context,
	source, epoch, qid string,
) (*leadfeed.OwnerAnswer, error) {
	var doc ownerAnswerDoc
	err := r.answers.FindOne(ctx, bson.M{"_id": ownerAnswerID(source, epoch, qid)}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, leadfeed.ErrAnswerNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find owner answer: %w", err)
	}
	answer := ownerAnswerFromDoc(&doc)
	return &answer, nil
}

// ListAnswers returns the answers of one epoch.
func (r *MongoOwnerAnswerRepository) ListAnswers(
	ctx context.Context,
	source, epoch string,
) ([]leadfeed.OwnerAnswer, error) {
	return r.list(ctx, bson.M{"source": source, "epoch": epoch})
}

// ListAnswersOtherEpochs returns the answers of the source given under another epoch.
func (r *MongoOwnerAnswerRepository) ListAnswersOtherEpochs(
	ctx context.Context,
	source, epoch string,
) ([]leadfeed.OwnerAnswer, error) {
	return r.list(ctx, bson.M{"source": source, "epoch": bson.M{"$ne": epoch}})
}

func (r *MongoOwnerAnswerRepository) list(ctx context.Context, filter bson.M) ([]leadfeed.OwnerAnswer, error) {
	order := bson.D{{Key: "answered_at", Value: 1}, {Key: "_id", Value: 1}}
	cur, err := r.answers.Find(ctx, filter, options.Find().SetSort(order))
	if err != nil {
		return nil, fmt.Errorf("list owner answers: %w", err)
	}
	var docs []ownerAnswerDoc
	if err = cur.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode owner answers: %w", err)
	}
	out := make([]leadfeed.OwnerAnswer, 0, len(docs))
	for i := range docs {
		out = append(out, ownerAnswerFromDoc(&docs[i]))
	}
	return out, nil
}

func ownerAnswerToDoc(a *leadfeed.OwnerAnswer) ownerAnswerDoc {
	return ownerAnswerDoc{
		ID:          ownerAnswerID(a.Source, a.Epoch, a.QID),
		Source:      a.Source,
		Epoch:       a.Epoch,
		QID:         a.QID,
		RecordID:    a.RecordID,
		QuestionFP:  a.QuestionFP,
		Task:        a.Task,
		AnswerID:    a.AnswerID,
		OptionID:    a.OptionID,
		OptionLabel: a.OptionLabel,
		Text:        a.Text,
		Scope:       a.Scope,
		AnsweredAt:  mongoTime(a.AnsweredAt),
		CommandID:   a.CommandID,
		CommandFP:   a.CommandFP,
	}
}

func ownerAnswerFromDoc(d *ownerAnswerDoc) leadfeed.OwnerAnswer {
	return leadfeed.OwnerAnswer{
		Source:      d.Source,
		Epoch:       d.Epoch,
		QID:         d.QID,
		RecordID:    d.RecordID,
		QuestionFP:  d.QuestionFP,
		Task:        d.Task,
		AnswerID:    d.AnswerID,
		OptionID:    d.OptionID,
		OptionLabel: d.OptionLabel,
		Text:        d.Text,
		Scope:       d.Scope,
		AnsweredAt:  d.AnsweredAt.UTC(),
		CommandID:   d.CommandID,
		CommandFP:   d.CommandFP,
	}
}
