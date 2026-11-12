package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// UpsertNewer applies update to the document matched by idFilter unless it already holds a later version.
// Concurrent rebuilds of one aggregate otherwise let an older state overwrite a newer one.
func UpsertNewer(ctx context.Context, coll *mongo.Collection, idFilter bson.M, version int, update bson.M) error {
	filter := bson.M{"$or": bson.A{
		bson.M{"version": bson.M{"$lte": version}},
		bson.M{"version": bson.M{"$exists": false}},
	}}
	for k, v := range idFilter {
		filter[k] = v
	}
	_, err := coll.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	if mongo.IsDuplicateKeyError(err) {
		// The document holds a newer version; the upsert's insert hit its unique id.
		return nil
	}
	return err
}
