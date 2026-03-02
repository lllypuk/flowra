package mongodb_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	mongodbinfra "github.com/lllypuk/flowra/internal/infrastructure/mongodb"
	"github.com/lllypuk/flowra/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestWarnIfLegacyReadModelCollectionsContainData(t *testing.T) {
	t.Run("does not log warning when legacy collections are empty", func(t *testing.T) {
		db := testutil.SetupTestMongoDB(t)
		ctx := context.Background()

		var logBuf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))

		err := mongodbinfra.WarnIfLegacyReadModelCollectionsContainData(ctx, db, logger)
		require.NoError(t, err)
		assert.Empty(t, logBuf.String())
	})

	t.Run("logs warning when legacy collections contain data", func(t *testing.T) {
		db := testutil.SetupTestMongoDB(t)
		ctx := context.Background()

		_, err := db.Collection("chat_read_model").InsertOne(ctx, bson.M{"chat_id": "legacy-chat"})
		require.NoError(t, err)
		_, err = db.Collection("task_read_model").InsertOne(ctx, bson.M{"task_id": "legacy-task"})
		require.NoError(t, err)

		var logBuf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))

		err = mongodbinfra.WarnIfLegacyReadModelCollectionsContainData(ctx, db, logger)
		require.NoError(t, err)

		logOutput := logBuf.String()
		assert.Contains(t, logOutput, "legacy read model collection contains data")
		assert.Contains(t, logOutput, "collection=chat_read_model")
		assert.Contains(t, logOutput, "collection=task_read_model")
		assert.Contains(t, logOutput, "guidance=\"run make reset-data\"")
	})
}
