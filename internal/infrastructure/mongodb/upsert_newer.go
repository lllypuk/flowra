package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const versionField = "version"

// UpsertNewer applies update to the document matched by idFilter unless it already holds a later version.
// Concurrent rebuilds of one aggregate otherwise let an older state overwrite a newer one.
func UpsertNewer(ctx context.Context, coll *mongo.Collection, idFilter bson.M, version int, update bson.M) error {
	filter := bson.M{"$or": bson.A{
		bson.M{versionField: bson.M{"$lte": version}},
		bson.M{versionField: bson.M{"$exists": false}},
	}}
	for k, v := range idFilter {
		filter[k] = v
	}
	_, err := coll.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	if !mongo.IsDuplicateKeyError(err) {
		return err
	}
	// The insert lost to another writer: either a newer version is stored, or an older one appeared after the
	// filter missed, so the version check runs once more against the stored document.
	_, err = coll.UpdateOne(ctx, filter, update)
	return err
}
