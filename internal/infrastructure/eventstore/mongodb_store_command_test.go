//go:build integration

package eventstore_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/infrastructure/eventstore"
	mongodbinfra "github.com/lllypuk/flowra/internal/infrastructure/mongodb"
	"github.com/lllypuk/flowra/tests/testutil"
)

func testEvents(aggregateID string, fromVersion, n int) []event.DomainEvent {
	events := make([]event.DomainEvent, n)
	for i := range n {
		base := event.NewBaseEvent("TestEventCreated", aggregateID, "TestAggregate", fromVersion+i,
			event.NewMetadata("user-1", "", ""))
		events[i] = &TestEvent{BaseEvent: base, TestData: "v"}
	}
	return events
}

func TestMongoEventStore_CommandKey(t *testing.T) {
	db := testutil.SetupTestMongoDB(t)
	ctx := context.Background()
	require.NoError(t, mongodbinfra.CreateAllIndexes(ctx, db))
	store := eventstore.NewMongoEventStore(db.Client(), db.Name())
	const aggregateID = "agg-cmd"

	require.NoError(t, store.SaveEvents(ctx, aggregateID, testEvents(aggregateID, 1, 1), 0))

	_, found, err := store.FindCommand(ctx, aggregateID, "k1")
	require.NoError(t, err)
	assert.False(t, found)

	cmdCtx := appcore.WithCommand(ctx, appcore.CommandKey{ID: "k1", Fingerprint: "due:2026-01-01"})
	require.NoError(t, store.SaveEvents(cmdCtx, aggregateID, testEvents(aggregateID, 2, 2), 1))

	rec, found, err := store.FindCommand(ctx, aggregateID, "k1")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, 3, rec.Version, "the record carries the version after the command's last event")
	assert.Equal(t, "due:2026-01-01", rec.Fingerprint)

	err = store.SaveEvents(cmdCtx, aggregateID, testEvents(aggregateID, 4, 1), 3)
	require.ErrorIs(t, err, appcore.ErrDuplicateCommand)

	_, found, err = store.FindCommand(ctx, "other-agg", "k1")
	require.NoError(t, err)
	assert.False(t, found, "keys are scoped to one aggregate")
	require.NoError(t, store.SaveEvents(appcore.WithCommand(ctx, appcore.CommandKey{ID: "k1"}),
		"other-agg", testEvents("other-agg", 1, 1), 0))

	version, err := store.GetVersion(ctx, aggregateID)
	require.NoError(t, err)
	assert.Equal(t, 3, version)
}
