package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// listDocuments выполняет общую логику получения списка документов с пагинацией.
// T - тип документа для декодирования
// R - тип результата (domain объект)
//
// Параметры:
//   - ctx: контекст выполнения
//   - collection: MongoDB коллекция
//   - offset: смещение для пагинации
//   - limit: лимит документов (если 0, используется DefaultPaginationLimit)
//   - decoder: функция преобразования документа в domain объект
//   - collectionName: название коллекции для сообщений об ошибках
//
// Возвращает:
//   - срез domain объектов (никогда не nil)
//   - ошибку при проблемах с запросом
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
			continue // пропускаем документы, которые не удалось преобразовать
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
