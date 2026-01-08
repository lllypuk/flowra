package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// listDocuments performs общую логику receivения list документов с пагинацией.
// T - type документа for декодирования
// R - type result (domain object)
//
// parameters:
//   - ctx: конtext выполнения
//   - collection: MongoDB коллекция
//   - offset: смещение for пагинации
//   - limit: лимит документов (if 0, used DefaultPaginationLimit)
//   - decoder: function conversion документа in domain object
//   - collectionName: название коллекции for сообщений об errorх
//
// returns:
//   - срез domain объектов (never not nil)
//   - error at проблемах с запросом
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
			continue // пропускаем некорректные документы
		}

		item, docErr := decoder(&doc)
		if docErr != nil {
			continue // пропускаем документы, которые not удалось convert
		}

		results = append(results, item)
	}

	if err = cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	// Гарантируем возврат пустого среза вместо nil
	if results == nil {
		results = make([]R, 0)
	}

	return results, nil
}
