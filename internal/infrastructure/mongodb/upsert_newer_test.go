package mongodb_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lllypuk/flowra/internal/infrastructure/mongodb"
	"github.com/lllypuk/flowra/tests/testutil"
)

func TestUpsertNewer(t *testing.T) {
	db := testutil.SetupTestMongoDB(t)
	ctx := context.Background()
	coll := db.Collection("rm")
	_, err := coll.Indexes().CreateOne(ctx, mongoIndex("chat_id"))
	require.NoError(t, err)

	id := bson.M{"chat_id": "c1"}
	write := func(version int, title string) {
		t.Helper()
		require.NoError(t, mongodb.UpsertNewer(ctx, coll, id, version,
			bson.M{"$set": bson.M{"chat_id": "c1", "version": version, "title": title}}))
	}
	title := func() string {
		var doc struct {
			Title string `bson:"title"`
		}
		require.NoError(t, coll.FindOne(ctx, id).Decode(&doc))
		return doc.Title
	}

	write(3, "v3")
	assert.Equal(t, "v3", title(), "inserts a missing document")
	write(2, "v2")
	assert.Equal(t, "v3", title(), "an older version does not overwrite a newer one")
	write(3, "v3 repaired")
	assert.Equal(t, "v3 repaired", title(), "the same version may repair the document")
	write(5, "v5")
	assert.Equal(t, "v5", title())

	_, err = coll.InsertOne(ctx, bson.M{"chat_id": "legacy", "title": "old"})
	require.NoError(t, err)
	require.NoError(t, mongodb.UpsertNewer(ctx, coll, bson.M{"chat_id": "legacy"}, 1,
		bson.M{"$set": bson.M{"version": 1, "title": "new"}}))
	n, err := coll.CountDocuments(ctx, bson.M{"chat_id": "legacy", "title": "new"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), n, "a document without a version is updated")
}

func mongoIndex(field string) mongo.IndexModel {
	return mongo.IndexModel{Keys: bson.D{{Key: field, Value: 1}}, Options: options.Index().SetUnique(true)}
}
