package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
	mongodbinfra "github.com/lllypuk/flowra/internal/infrastructure/mongodb"
)

const (
	leadFeedStateID   = "state"
	leadFeedAttemptID = "attempt"

	defaultLeadFeedTxTimeout = 30 * time.Second

	labelUnknownCommit = "UnknownTransactionCommitResult"
)

// MongoLeadFeedRepository stores the one published lead feed snapshot.
// Publication and every read run in a MongoDB transaction, so readers never see a mix of two snapshots.
type MongoLeadFeedRepository struct {
	client    *mongo.Client
	tasks     *mongo.Collection
	records   *mongo.Collection
	state     *mongo.Collection
	txTimeout time.Duration
}

// LeadFeedRepoOption configures MongoLeadFeedRepository.
type LeadFeedRepoOption func(*MongoLeadFeedRepository)

// WithLeadFeedTxTimeout bounds one publication; a snapshot that cannot be written in time is not published.
func WithLeadFeedTxTimeout(timeout time.Duration) LeadFeedRepoOption {
	return func(r *MongoLeadFeedRepository) {
		r.txTimeout = timeout
	}
}

// NewMongoLeadFeedRepository creates a lead feed repository.
func NewMongoLeadFeedRepository(
	client *mongo.Client,
	db *mongo.Database,
	opts ...LeadFeedRepoOption,
) *MongoLeadFeedRepository {
	r := &MongoLeadFeedRepository{
		client:    client,
		tasks:     db.Collection(mongodbinfra.CollectionLeadFeedTasks),
		records:   db.Collection(mongodbinfra.CollectionLeadFeedRecords),
		state:     db.Collection(mongodbinfra.CollectionLeadFeedState),
		txTimeout: defaultLeadFeedTxTimeout,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

type leadFeedTransitionDoc struct {
	At      time.Time `bson:"at"`
	Status  string    `bson:"status"`
	Cause   *string   `bson:"cause"`
	Derived bool      `bson:"derived"`
}

type leadFeedTaskDoc struct {
	ID          string                  `bson:"_id"`
	Title       string                  `bson:"title"`
	Project     string                  `bson:"project"`
	Agent       string                  `bson:"agent"`
	CreatedAt   time.Time               `bson:"created_at"`
	Status      string                  `bson:"status"`
	WaitingOn   *string                 `bson:"waiting_on"`
	StatusSince time.Time               `bson:"status_since"`
	Note        string                  `bson:"note"`
	Returns     int                     `bson:"returns"`
	SpentQueued int64                   `bson:"spent_queued"`
	SpentWork   int64                   `bson:"spent_work"`
	SpentLead   int64                   `bson:"spent_lead"`
	SpentUser   int64                   `bson:"spent_user"`
	Transitions []leadFeedTransitionDoc `bson:"transitions"`
}

type leadFeedRecordDoc struct {
	ID           string    `bson:"_id"`
	Task         string    `bson:"task"`
	Kind         string    `bson:"kind"`
	At           time.Time `bson:"at"`
	Order        int       `bson:"order"`
	Actor        string    `bson:"actor"`
	ActorDerived bool      `bson:"actor_derived"`
	SourceKind   string    `bson:"source_kind"`
	SourceRef    string    `bson:"source_ref"`
	Body         *string   `bson:"body"`
	BodyDerived  bool      `bson:"body_derived"`
	BodyMissing  *string   `bson:"body_missing"`
	Format       string    `bson:"format"`
	// Every field of leadfeed.Record is copied by hand: one added there and missed here is dropped on publication.
	Question *leadFeedQuestionDoc `bson:"question,omitempty"`
}

type leadFeedOptionDoc struct {
	ID    string `bson:"id"`
	Label string `bson:"label"`
}

type leadFeedRecommendationDoc struct {
	OptionID *string `bson:"option_id"`
	Why      string  `bson:"why"`
}

type leadFeedAnswerDoc struct {
	AnswerID   string    `bson:"answer_id"`
	OptionID   *string   `bson:"option_id"`
	Text       string    `bson:"text"`
	AnsweredAt time.Time `bson:"answered_at"`
	Late       bool      `bson:"late"`
	Delivered  bool      `bson:"delivered"`
}

type leadFeedResolvedDoc struct {
	At        time.Time `bson:"at"`
	Note      string    `bson:"note"`
	Elsewhere bool      `bson:"elsewhere"`
}

type leadFeedQuestionDoc struct {
	QID            string                     `bson:"qid"`
	FP             string                     `bson:"fp"`
	Options        []leadFeedOptionDoc        `bson:"options"`
	Recommendation *leadFeedRecommendationDoc `bson:"recommendation"`
	Scope          string                     `bson:"scope"`
	Answer         *leadFeedAnswerDoc         `bson:"answer"`
	Resolved       *leadFeedResolvedDoc       `bson:"resolved"`
}

type leadFeedStateDoc struct {
	ID          string    `bson:"_id"`
	Source      string    `bson:"source"`
	Epoch       string    `bson:"epoch"`
	Revision    string    `bson:"revision"`
	AsOf        time.Time `bson:"as_of"`
	Tasks       int       `bson:"tasks"`
	Records     int       `bson:"records"`
	PublishedAt time.Time `bson:"published_at"`
}

type leadFeedAttemptDoc struct {
	ID            string    `bson:"_id"`
	At            time.Time `bson:"at"`
	Outcome       string    `bson:"outcome"`
	Error         string    `bson:"error"`
	Revision      string    `bson:"revision"`
	PreviousEpoch string    `bson:"previous_epoch"`
}

// Publish replaces the feed with snap in one transaction.
// The freshness check is inside the transaction because the driver may run it more than once.
func (r *MongoLeadFeedRepository) Publish(
	ctx context.Context,
	snap *leadfeed.Snapshot,
	now time.Time,
) (leadfeed.PublishResult, error) {
	taskDocs := make([]leadFeedTaskDoc, 0, len(snap.Tasks))
	for i := range snap.Tasks {
		taskDocs = append(taskDocs, leadFeedTaskToDoc(&snap.Tasks[i]))
	}
	recordDocs := make([]leadFeedRecordDoc, 0, len(snap.Records))
	for i := range snap.Records {
		recordDocs = append(recordDocs, leadFeedRecordToDoc(&snap.Records[i]))
	}
	next := leadFeedStateDoc{
		ID:          leadFeedStateID,
		Source:      snap.Header.Source,
		Epoch:       snap.Header.Epoch,
		Revision:    snap.Header.Revision,
		AsOf:        mongoTime(snap.Header.AsOf),
		Tasks:       len(taskDocs),
		Records:     len(recordDocs),
		PublishedAt: mongoTime(now),
	}

	session, err := r.client.StartSession()
	if err != nil {
		return leadfeed.PublishResult{}, fmt.Errorf("start session: %w", err)
	}
	defer session.EndSession(ctx)

	txCtx, cancel := context.WithTimeout(ctx, r.txTimeout)
	defer cancel()

	result, err := session.WithTransaction(txCtx, func(sc context.Context) (any, error) {
		return r.publishInTx(sc, next, taskDocs, recordDocs)
	}, options.Transaction().
		SetReadConcern(readconcern.Snapshot()).
		SetWriteConcern(writeconcern.Majority()))
	if err != nil {
		var labeled mongo.LabeledError
		if errors.As(err, &labeled) && labeled.HasErrorLabel(labelUnknownCommit) {
			return leadfeed.PublishResult{}, fmt.Errorf("%w: %w", leadfeed.ErrCommitUnknown, err)
		}
		return leadfeed.PublishResult{}, err
	}

	published, ok := result.(leadfeed.PublishResult)
	if !ok {
		return leadfeed.PublishResult{}, fmt.Errorf("unexpected publication result %T", result)
	}
	return published, nil
}

func (r *MongoLeadFeedRepository) publishInTx(
	ctx context.Context,
	next leadFeedStateDoc,
	taskDocs []leadFeedTaskDoc,
	recordDocs []leadFeedRecordDoc,
) (leadfeed.PublishResult, error) {
	current, err := r.loadState(ctx)
	if err != nil {
		return leadfeed.PublishResult{}, err
	}

	result := leadfeed.PublishResult{Outcome: leadfeed.OutcomePublished}
	if current != nil {
		sameContent := current.Source == next.Source && current.Epoch == next.Epoch &&
			current.Revision == next.Revision

		switch {
		case next.AsOf.Before(current.AsOf):
			return leadfeed.PublishResult{}, leadfeed.ErrSnapshotStale
		case sameContent:
			// The watermark must move even though the content stays: a late older snapshot is judged by it.
			_, err = r.state.UpdateOne(ctx, bson.M{"_id": leadFeedStateID}, bson.M{"$set": bson.M{
				"as_of":        next.AsOf,
				"published_at": next.PublishedAt,
			}})
			if err != nil {
				return leadfeed.PublishResult{}, fmt.Errorf("advance feed state: %w", err)
			}
			return leadfeed.PublishResult{Outcome: leadfeed.OutcomeRefreshed}, nil
		case next.AsOf.Equal(current.AsOf):
			return leadfeed.PublishResult{}, leadfeed.ErrSnapshotConflict
		}

		if current.Epoch != next.Epoch {
			result.PreviousEpoch = current.Epoch
		}
	}

	if _, err = r.tasks.DeleteMany(ctx, bson.M{}); err != nil {
		return leadfeed.PublishResult{}, fmt.Errorf("clear feed tasks: %w", err)
	}
	if _, err = r.records.DeleteMany(ctx, bson.M{}); err != nil {
		return leadfeed.PublishResult{}, fmt.Errorf("clear feed records: %w", err)
	}
	if len(taskDocs) > 0 {
		if _, err = r.tasks.InsertMany(ctx, taskDocs); err != nil {
			return leadfeed.PublishResult{}, fmt.Errorf("insert feed tasks: %w", err)
		}
	}
	if len(recordDocs) > 0 {
		if _, err = r.records.InsertMany(ctx, recordDocs); err != nil {
			return leadfeed.PublishResult{}, fmt.Errorf("insert feed records: %w", err)
		}
	}
	_, err = r.state.ReplaceOne(ctx, bson.M{"_id": leadFeedStateID}, next, options.Replace().SetUpsert(true))
	if err != nil {
		return leadfeed.PublishResult{}, fmt.Errorf("write feed state: %w", err)
	}

	return result, nil
}

// RecordAttempt stores the last import attempt in its own document, outside any publication.
func (r *MongoLeadFeedRepository) RecordAttempt(ctx context.Context, attempt leadfeed.Attempt) error {
	doc := leadFeedAttemptDoc{
		ID:            leadFeedAttemptID,
		At:            mongoTime(attempt.At),
		Outcome:       attempt.Outcome,
		Error:         attempt.Error,
		Revision:      attempt.Revision,
		PreviousEpoch: attempt.PreviousEpoch,
	}
	_, err := r.state.ReplaceOne(ctx, bson.M{"_id": leadFeedAttemptID}, doc, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("record feed import attempt: %w", err)
	}
	return nil
}

// Read runs fn against one snapshot of the feed.
// fn may run more than once and must rebuild its result from scratch each time.
func (r *MongoLeadFeedRepository) Read(
	ctx context.Context,
	fn func(ctx context.Context, view leadfeed.View) error,
) error {
	session, err := r.client.StartSession()
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sc context.Context) (any, error) {
		return nil, fn(sc, leadFeedView{repo: r})
	}, options.Transaction().
		SetReadConcern(readconcern.Snapshot()).
		SetReadPreference(readpref.Primary()))
	return err
}

func (r *MongoLeadFeedRepository) loadState(ctx context.Context) (*leadFeedStateDoc, error) {
	var doc leadFeedStateDoc
	err := r.state.FindOne(ctx, bson.M{"_id": leadFeedStateID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil //nolint:nilnil // no state is a valid answer before the first publication
	}
	if err != nil {
		return nil, fmt.Errorf("load feed state: %w", err)
	}
	return &doc, nil
}

// leadFeedView reads through the session carried by the context it is given.
type leadFeedView struct {
	repo *MongoLeadFeedRepository
}

func (v leadFeedView) State(ctx context.Context) (*leadfeed.State, error) {
	doc, err := v.repo.loadState(ctx)
	if err != nil || doc == nil {
		return nil, err
	}
	return &leadfeed.State{
		Source:      doc.Source,
		Epoch:       doc.Epoch,
		Revision:    doc.Revision,
		AsOf:        doc.AsOf.UTC(),
		Tasks:       doc.Tasks,
		Records:     doc.Records,
		PublishedAt: doc.PublishedAt.UTC(),
	}, nil
}

func (v leadFeedView) LastAttempt(ctx context.Context) (*leadfeed.Attempt, error) {
	var doc leadFeedAttemptDoc
	err := v.repo.state.FindOne(ctx, bson.M{"_id": leadFeedAttemptID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil //nolint:nilnil // no attempt is a valid answer before the first import
	}
	if err != nil {
		return nil, fmt.Errorf("load feed import attempt: %w", err)
	}
	return &leadfeed.Attempt{
		At:            doc.At.UTC(),
		Outcome:       doc.Outcome,
		Error:         doc.Error,
		Revision:      doc.Revision,
		PreviousEpoch: doc.PreviousEpoch,
	}, nil
}

func (v leadFeedView) Tasks(
	ctx context.Context,
	filter leadfeed.TaskFilter,
	offset, limit int,
) ([]leadfeed.Task, error) {
	query := bson.M{}
	if filter.Status != nil {
		query["status"] = *filter.Status
	}
	if filter.WaitingOn != nil {
		if *filter.WaitingOn == leadfeed.WaitingOnNobody {
			query["waiting_on"] = nil
		} else {
			query["waiting_on"] = *filter.WaitingOn
		}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}).
		SetSkip(int64(offset)).
		SetLimit(int64(limit))
	cursor, err := v.repo.tasks.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("find feed tasks: %w", err)
	}

	var docs []leadFeedTaskDoc
	if err = cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode feed tasks: %w", err)
	}

	tasks := make([]leadfeed.Task, 0, len(docs))
	for i := range docs {
		tasks = append(tasks, leadFeedTaskFromDoc(&docs[i]))
	}
	return tasks, nil
}

func (v leadFeedView) Task(ctx context.Context, id string) (*leadfeed.Task, error) {
	var doc leadFeedTaskDoc
	err := v.repo.tasks.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, leadfeed.ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find feed task: %w", err)
	}
	task := leadFeedTaskFromDoc(&doc)
	return &task, nil
}

func (v leadFeedView) Records(
	ctx context.Context,
	taskID string,
	offset, limit int,
) ([]leadfeed.Record, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "order", Value: 1}, {Key: "_id", Value: 1}}).
		SetSkip(int64(offset)).
		SetLimit(int64(limit))
	cursor, err := v.repo.records.Find(ctx, bson.M{"task": taskID}, opts)
	if err != nil {
		return nil, fmt.Errorf("find feed records: %w", err)
	}

	var docs []leadFeedRecordDoc
	if err = cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode feed records: %w", err)
	}

	records := make([]leadfeed.Record, 0, len(docs))
	for i := range docs {
		records = append(records, leadFeedRecordFromDoc(&docs[i]))
	}
	return records, nil
}

func (v leadFeedView) Question(ctx context.Context, qid string) (*leadfeed.Record, error) {
	var doc leadFeedRecordDoc
	err := v.repo.records.FindOne(ctx, bson.M{"question.qid": qid}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, leadfeed.ErrQuestionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find feed question: %w", err)
	}
	rec := leadFeedRecordFromDoc(&doc)
	return &rec, nil
}

// mongoTime drops sub-millisecond precision so that stored and compared times agree.
func mongoTime(t time.Time) time.Time {
	return t.UTC().Truncate(time.Millisecond)
}

func leadFeedTaskToDoc(t *leadfeed.Task) leadFeedTaskDoc {
	transitions := make([]leadFeedTransitionDoc, 0, len(t.Transitions))
	for _, tr := range t.Transitions {
		transitions = append(transitions, leadFeedTransitionDoc{
			At:      mongoTime(tr.At),
			Status:  tr.Status,
			Cause:   tr.Cause,
			Derived: tr.Derived,
		})
	}
	return leadFeedTaskDoc{
		ID:          t.ID,
		Title:       t.Title,
		Project:     t.Project,
		Agent:       t.Agent,
		CreatedAt:   mongoTime(t.CreatedAt),
		Status:      t.Status,
		WaitingOn:   t.WaitingOn,
		StatusSince: mongoTime(t.StatusSince),
		Note:        t.Note,
		Returns:     t.Returns,
		SpentQueued: t.Spent.Queued,
		SpentWork:   t.Spent.Work,
		SpentLead:   t.Spent.Lead,
		SpentUser:   t.Spent.User,
		Transitions: transitions,
	}
}

func leadFeedTaskFromDoc(doc *leadFeedTaskDoc) leadfeed.Task {
	transitions := make([]leadfeed.Transition, 0, len(doc.Transitions))
	for _, tr := range doc.Transitions {
		transitions = append(transitions, leadfeed.Transition{
			At:      tr.At.UTC(),
			Status:  tr.Status,
			Cause:   tr.Cause,
			Derived: tr.Derived,
		})
	}
	return leadfeed.Task{
		ID:          doc.ID,
		Title:       doc.Title,
		Project:     doc.Project,
		Agent:       doc.Agent,
		CreatedAt:   doc.CreatedAt.UTC(),
		Status:      doc.Status,
		WaitingOn:   doc.WaitingOn,
		StatusSince: doc.StatusSince.UTC(),
		Note:        doc.Note,
		Returns:     doc.Returns,
		Spent: leadfeed.Spent{
			Queued: doc.SpentQueued,
			Work:   doc.SpentWork,
			Lead:   doc.SpentLead,
			User:   doc.SpentUser,
		},
		Transitions: transitions,
	}
}

func leadFeedRecordToDoc(rec *leadfeed.Record) leadFeedRecordDoc {
	return leadFeedRecordDoc{
		ID:           rec.ID,
		Task:         rec.Task,
		Kind:         rec.Kind,
		At:           mongoTime(rec.At),
		Order:        rec.Order,
		Actor:        rec.Actor,
		ActorDerived: rec.ActorDerived,
		SourceKind:   rec.Source.Kind,
		SourceRef:    rec.Source.Ref,
		Body:         rec.Body,
		BodyDerived:  rec.BodyDerived,
		BodyMissing:  rec.BodyMissing,
		Format:       rec.Format,
		Question:     leadFeedQuestionToDoc(rec.Question),
	}
}

func leadFeedRecordFromDoc(doc *leadFeedRecordDoc) leadfeed.Record {
	return leadfeed.Record{
		ID:           doc.ID,
		Task:         doc.Task,
		Kind:         doc.Kind,
		At:           doc.At.UTC(),
		Order:        doc.Order,
		Actor:        doc.Actor,
		ActorDerived: doc.ActorDerived,
		Source:       leadfeed.Source{Kind: doc.SourceKind, Ref: doc.SourceRef},
		Body:         doc.Body,
		BodyDerived:  doc.BodyDerived,
		BodyMissing:  doc.BodyMissing,
		Format:       doc.Format,
		Question:     leadFeedQuestionFromDoc(doc.Question),
	}
}

func leadFeedQuestionToDoc(q *leadfeed.Question) *leadFeedQuestionDoc {
	if q == nil {
		return nil
	}
	doc := &leadFeedQuestionDoc{
		QID: q.QID, FP: q.FP, Scope: q.Scope, Options: make([]leadFeedOptionDoc, 0, len(q.Options)),
	}
	for _, opt := range q.Options {
		doc.Options = append(doc.Options, leadFeedOptionDoc{ID: opt.ID, Label: opt.Label})
	}
	if q.Recommendation != nil {
		doc.Recommendation = &leadFeedRecommendationDoc{OptionID: q.Recommendation.OptionID, Why: q.Recommendation.Why}
	}
	if a := q.Answer; a != nil {
		doc.Answer = &leadFeedAnswerDoc{
			AnswerID: a.AnswerID, OptionID: a.OptionID, Text: a.Text,
			AnsweredAt: mongoTime(a.AnsweredAt), Late: a.Late, Delivered: a.Delivered,
		}
	}
	if r := q.Resolved; r != nil {
		doc.Resolved = &leadFeedResolvedDoc{At: mongoTime(r.At), Note: r.Note, Elsewhere: r.Elsewhere}
	}
	return doc
}

func leadFeedQuestionFromDoc(doc *leadFeedQuestionDoc) *leadfeed.Question {
	if doc == nil {
		return nil
	}
	q := &leadfeed.Question{
		QID: doc.QID, FP: doc.FP, Scope: doc.Scope, Options: make([]leadfeed.Option, 0, len(doc.Options)),
	}
	for _, opt := range doc.Options {
		q.Options = append(q.Options, leadfeed.Option{ID: opt.ID, Label: opt.Label})
	}
	if doc.Recommendation != nil {
		q.Recommendation = &leadfeed.Recommendation{OptionID: doc.Recommendation.OptionID, Why: doc.Recommendation.Why}
	}
	if a := doc.Answer; a != nil {
		q.Answer = &leadfeed.Answer{
			AnswerID: a.AnswerID, OptionID: a.OptionID, Text: a.Text,
			AnsweredAt: a.AnsweredAt.UTC(), Late: a.Late, Delivered: a.Delivered,
		}
	}
	if r := doc.Resolved; r != nil {
		q.Resolved = &leadfeed.Resolved{At: r.At.UTC(), Note: r.Note, Elsewhere: r.Elsewhere}
	}
	return q
}
