package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/lllypuk/flowra/internal/domain/errs"
)

const (
	// DefaultPaginationLimit — дефолтный лимит for пагинации запросов.
	DefaultPaginationLimit = 50

	// MaxPaginationLimit — максимальный лимит for пагинации запросов.
	MaxPaginationLimit = 100
)

// HandleMongoError преобразует error MongoDB in доменную error.
// returns:
//   - nil if err == nil
//   - errs.ErrNotFound if документ not найден
//   - errs.ErrAlreadyExists if нарушен unique constraint
//   - wrapped error for остальных случаев
func HandleMongoError(err error, resourceType string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, mongo.ErrNoDocuments) {
		return errs.ErrNotFound
	}

	if mongo.IsDuplicateKeyError(err) {
		return errs.ErrAlreadyExists
	}

	return fmt.Errorf("failed to operate on %s: %w", resourceType, err)
}

// BaseDocument contains общие fields for all документов MongoDB.
// Используется as встраиваемая struct for adding метаданных time.
type BaseDocument struct {
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

// TouchUpdatedAt обновляет field UpdatedAt on current time UTC.
func (d *BaseDocument) TouchUpdatedAt() {
	d.UpdatedAt = time.Now().UTC()
}

// SetCreatedAt устанавливает CreatedAt on current time UTC,
// if оно еще not было установлено (zero value).
func (d *BaseDocument) SetCreatedAt() {
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now().UTC()
	}
}

// SetTimestamps устанавливает оба fields time.
// CreatedAt устанавливается only if not был установлен ранее.
// UpdatedAt always обновляется on current time.
func (d *BaseDocument) SetTimestamps() {
	now := time.Now().UTC()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	d.UpdatedAt = now
}

// UpsertOptions returns стандартные опции for upsert операции.
// use:
//
//	_, err := collection.UpdateOne(ctx, filter, update, UpsertOptions())
func UpsertOptions() *options.UpdateOneOptionsBuilder {
	return options.UpdateOne().SetUpsert(true)
}

// FindWithPagination returns опции for find с пагинацией and сортировкой.
// parameters:
//   - offset: count документов for пропуска
//   - limit: максимальное count возвращаемых документов
//   - sortField: field for сортировки (например, "created_at")
//   - sortOrder: order сортировки (1 = ASC, -1 = DESC)
//
// use:
//
//	opts := FindWithPagination(0, 50, "created_at", -1)
//	cursor, err := collection.Find(ctx, filter, opts)
func FindWithPagination(offset, limit int, sortField string, sortOrder int) *options.FindOptionsBuilder {
	return options.Find().
		SetSort(bson.D{{Key: sortField, Value: sortOrder}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))
}

// FindWithPaginationDesc returns опции for find с пагинацией and сортировкой по убыванию.
// Это convenience function for частого случая сортировки по created_at DESC.
func FindWithPaginationDesc(offset, limit int) *options.FindOptionsBuilder {
	return FindWithPagination(offset, limit, "created_at", -1)
}

// CountFilter performs подсчет документов с указанным фильтром.
// returns count документов, соresponseствующих фильтру.
func CountFilter(ctx context.Context, coll *mongo.Collection, filter bson.M) (int, error) {
	count, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// CountAll performs подсчет all документов in коллекции.
// Это convenience function for подсчета без фильтра.
func CountAll(ctx context.Context, coll *mongo.Collection) (int, error) {
	return CountFilter(ctx, coll, bson.M{})
}

// DefaultLimit returns limit с applyingм дефолтного values.
// if limit <= 0, returns defaultLimit.
func DefaultLimit(limit, defaultLimit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	return limit
}

// DefaultLimitWithMax returns limit с applyingм дефолтного and максимального values.
// if limit <= 0, returns defaultLimit.
// if limit > maxLimit, returns maxLimit.
func DefaultLimitWithMax(limit, defaultLimit, maxLimit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}

// StringPtr returns pointer on строку.
// if строка пустая, returns nil.
// Полезно for optional string полей in документах.
func StringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// StringValue returns value строки from указателя.
// if pointer nil, returns пустую строку.
func StringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
