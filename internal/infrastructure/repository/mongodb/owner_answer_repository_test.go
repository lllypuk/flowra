package mongodb_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/leadfeed"
	"github.com/lllypuk/flowra/internal/infrastructure/repository/mongodb"
	"github.com/lllypuk/flowra/tests/testutil"
)

func setupOwnerAnswers(t *testing.T) (*mongodb.MongoOwnerAnswerRepository, *mongodb.MongoLeadFeedRepository) {
	t.Helper()
	client, db := testutil.SetupTestMongoDBWithClient(t)
	return mongodb.NewMongoOwnerAnswerRepository(db), mongodb.NewMongoLeadFeedRepository(client, db)
}

func ptr(s string) *string { return &s }

func newAnswer(epoch, qid, key, text string) leadfeed.OwnerAnswer {
	option := ptr("opt-a")
	return leadfeed.OwnerAnswer{
		Source:      "lead-task",
		Epoch:       epoch,
		QID:         qid,
		RecordID:    "j:7",
		QuestionFP:  "fp-" + qid,
		Task:        "T-0001",
		AnswerID:    "answer-" + key,
		OptionID:    option,
		OptionLabel: ptr("Option A"),
		Text:        text,
		Scope:       "this task only",
		AnsweredAt:  feedBase().Add(time.Minute),
		CommandID:   key,
		CommandFP:   leadfeed.AnswerFingerprint(qid, option, text),
	}
}

func TestOwnerAnswerRepository_FirstAnswerKeepsDeliveryData(t *testing.T) {
	repo, _ := setupOwnerAnswers(t)
	ctx := context.Background()

	want := newAnswer("epoch-1", "Q-0001", "key-1", "go ahead")
	res, err := repo.SaveAnswer(ctx, want)
	require.NoError(t, err)
	assert.Equal(t, leadfeed.AnswerCreated, res.Outcome)
	assert.Equal(t, want, res.Answer)

	got, err := repo.FindAnswer(ctx, "lead-task", "epoch-1", "Q-0001")
	require.NoError(t, err)
	assert.Equal(t, want, *got)

	_, err = repo.FindAnswer(ctx, "lead-task", "epoch-1", "Q-0002")
	require.ErrorIs(t, err, leadfeed.ErrAnswerNotFound)
}

func TestOwnerAnswerRepository_FindByKey(t *testing.T) {
	repo, _ := setupOwnerAnswers(t)
	ctx := context.Background()

	want := newAnswer("epoch-1", "Q-0001", "key-1", "go ahead")
	_, err := repo.SaveAnswer(ctx, want)
	require.NoError(t, err)

	got, err := repo.FindAnswerByKey(ctx, "key-1")
	require.NoError(t, err)
	assert.Equal(t, want, *got)

	_, err = repo.FindAnswerByKey(ctx, "key-2")
	require.ErrorIs(t, err, leadfeed.ErrAnswerNotFound)
}

func TestOwnerAnswerRepository_SameKeyRepeats(t *testing.T) {
	repo, _ := setupOwnerAnswers(t)
	ctx := context.Background()

	first, err := repo.SaveAnswer(ctx, newAnswer("epoch-1", "Q-0001", "key-1", "go ahead"))
	require.NoError(t, err)

	retry := newAnswer("epoch-1", "Q-0001", "key-1", "go ahead")
	retry.AnswerID = "a-fresh-id"
	res, err := repo.SaveAnswer(ctx, retry)
	require.NoError(t, err)
	assert.Equal(t, leadfeed.AnswerRepeated, res.Outcome)
	assert.Equal(t, first.Answer, res.Answer, "the original answer, not the retry")
}

func TestOwnerAnswerRepository_SameKeyOtherBody(t *testing.T) {
	repo, _ := setupOwnerAnswers(t)
	ctx := context.Background()

	_, err := repo.SaveAnswer(ctx, newAnswer("epoch-1", "Q-0001", "key-1", "go ahead"))
	require.NoError(t, err)

	_, err = repo.SaveAnswer(ctx, newAnswer("epoch-1", "Q-0001", "key-1", "another text"))
	require.ErrorIs(t, err, leadfeed.ErrIdempotencyKeyReused)

	_, err = repo.SaveAnswer(ctx, newAnswer("epoch-1", "Q-0002", "key-1", "go ahead"))
	require.ErrorIs(t, err, leadfeed.ErrIdempotencyKeyReused, "the key is bound to the first question too")
}

func TestOwnerAnswerRepository_OtherKeyOnAnsweredQuestion(t *testing.T) {
	repo, _ := setupOwnerAnswers(t)
	ctx := context.Background()

	first, err := repo.SaveAnswer(ctx, newAnswer("epoch-1", "Q-0001", "key-1", "go ahead"))
	require.NoError(t, err)

	_, err = repo.SaveAnswer(ctx, newAnswer("epoch-1", "Q-0001", "key-2", "go ahead"))
	require.ErrorIs(t, err, leadfeed.ErrAlreadyAnswered)

	got, err := repo.FindAnswer(ctx, "lead-task", "epoch-1", "Q-0001")
	require.NoError(t, err)
	assert.Equal(t, first.Answer, *got)
}

func TestOwnerAnswerRepository_RaceStoresExactlyOne(t *testing.T) {
	repo, _ := setupOwnerAnswers(t)
	ctx := context.Background()

	const racers = 8
	var wg sync.WaitGroup
	results := make([]error, racers)
	outcomes := make([]leadfeed.AnswerOutcome, racers)
	for i := range racers {
		wg.Go(func() {
			res, err := repo.SaveAnswer(ctx, newAnswer("epoch-1", "Q-0001", fmt.Sprintf("key-%d", i), "text"))
			results[i], outcomes[i] = err, res.Outcome
		})
	}
	wg.Wait()

	created := 0
	for i := range racers {
		if results[i] == nil {
			assert.Equal(t, leadfeed.AnswerCreated, outcomes[i])
			created++
			continue
		}
		require.ErrorIs(t, results[i], leadfeed.ErrAlreadyAnswered)
	}
	assert.Equal(t, 1, created)

	list, err := repo.ListAnswers(ctx, "lead-task", "epoch-1")
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestOwnerAnswerRepository_RaceOnSameKeyIsRepeat(t *testing.T) {
	repo, _ := setupOwnerAnswers(t)
	ctx := context.Background()

	const racers = 8
	var wg sync.WaitGroup
	results := make([]error, racers)
	for i := range racers {
		wg.Go(func() {
			_, results[i] = repo.SaveAnswer(ctx, newAnswer("epoch-1", "Q-0001", "key-1", "text"))
		})
	}
	wg.Wait()

	for i := range racers {
		require.NoError(t, results[i])
	}
	list, err := repo.ListAnswers(ctx, "lead-task", "epoch-1")
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestOwnerAnswerRepository_NewEpochIsSeparate(t *testing.T) {
	repo, _ := setupOwnerAnswers(t)
	ctx := context.Background()

	_, err := repo.SaveAnswer(ctx, newAnswer("epoch-1", "Q-0001", "key-1", "old"))
	require.NoError(t, err)
	res, err := repo.SaveAnswer(ctx, newAnswer("epoch-2", "Q-0001", "key-2", "new"))
	require.NoError(t, err)
	assert.Equal(t, leadfeed.AnswerCreated, res.Outcome)

	current, err := repo.ListAnswers(ctx, "lead-task", "epoch-2")
	require.NoError(t, err)
	require.Len(t, current, 1)
	assert.Equal(t, "new", current[0].Text)

	other, err := repo.ListAnswersOtherEpochs(ctx, "lead-task", "epoch-2")
	require.NoError(t, err)
	require.Len(t, other, 1)
	assert.Equal(t, "old", other[0].Text)
}

func TestOwnerAnswerRepository_ImportLeavesAnswersAlone(t *testing.T) {
	answers, feed := setupOwnerAnswers(t)
	ctx := context.Background()

	_, err := answers.SaveAnswer(ctx, newAnswer("epoch-1", "Q-0001", "key-1", "go ahead"))
	require.NoError(t, err)

	_, err = feed.Publish(ctx, feedSnapshot("rev-a", 10, "T-0001"), feedBase().Add(time.Hour))
	require.NoError(t, err)
	_, err = feed.Publish(ctx, feedSnapshot("rev-b", 20, "T-0002"), feedBase().Add(2*time.Hour))
	require.NoError(t, err)

	got, err := answers.FindAnswer(ctx, "lead-task", "epoch-1", "Q-0001")
	require.NoError(t, err)
	assert.Equal(t, "go ahead", got.Text)
}
