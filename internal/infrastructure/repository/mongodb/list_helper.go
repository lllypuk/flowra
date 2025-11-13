package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// listDocuments выполняет общую логику получения списка документов с пагинацией
// T - тип документа для декодирования
// R - тип результата (domain объект)
func listDocuments[T any, R any](
	ctx context.Context,
	collection *mongo.Collection,
	offset, limit int,
	decoder func(*T) (R, error),
	collectionName string,
) ([]R, error) {
	if limit == 0 {
		limit = 50
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, HandleMongoError(err, collectionName)
	}
	defer cursor.Close(ctx)

	var results []R
	for cursor.Next(ctx) {
		var doc T
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			continue
		}

		item, docErr := decoder(&doc)
		if docErr != nil {
			continue
		}

		results = append(results, item)
	}

	if err = cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if results == nil {
		results = make([]R, 0)
	}

	return results, nil
}
