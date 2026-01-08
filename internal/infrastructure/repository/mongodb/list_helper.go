package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// listDocuments performs obschuyu logiku receiv list dokumentov s paginatsiey.
// T - type dokumenta for dekodirovaniya
// R - type result (domain object)
//
// parameters:
// - ctx: text vypolneniya
// - collection: MongoDB kollektsiya
// - offset: smeschenie for paginatsii
// - limit: limit dokumentov (if 0, used DefaultPaginationLimit)
// - decoder: function conversion dokumenta in domain object
// - collectionName: nazvanie kollektsii for soobscheniy ob error
//
// returns:
// - srez domain obektov (never not nil)
// - error at problemah s zaprosom
func listDocuments[T any, R any](
	ctx context.Context,
	collection *mongo.Collection,
	offset, limit int,
	decoder func(*T) (R, error),
	collectionName string,
) ([]R, error) {
	limit = DefaultLimit(limit, DefaultPaginationLimit)

	opts := FindWithPaginationDesc(offset, limit)

	cursor, err := collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, HandleMongoError(err, collectionName)
	}
	defer cursor.Close(ctx)

	var results []R
	for cursor.Next(ctx) {
		var doc T
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			continue // propuskaem nekorrektnye dokumenty
		}

		item, docErr := decoder(&doc)
		if docErr != nil {
			continue // propuskaem dokumenty, kotorye not udalos convert
		}

		results = append(results, item)
	}

	if err = cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	// garantiruem vozvrat pustogo sreza vmesto nil
	if results == nil {
		results = make([]R, 0)
	}

	return results, nil
}
