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
	// DefaultPaginationLimit — дефолтный лимит для пагинации запросов.
	DefaultPaginationLimit = 50

	// MaxPaginationLimit — максимальный лимит для пагинации запросов.
	MaxPaginationLimit = 100
)

// HandleMongoError преобразует ошибку MongoDB в доменную ошибку.
// Возвращает:
//   - nil если err == nil
//   - errs.ErrNotFound если документ не найден
//   - errs.ErrAlreadyExists если нарушен unique constraint
//   - wrapped error для остальных случаев
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

// BaseDocument содержит общие поля для всех документов MongoDB.
// Используется как встраиваемая структура для добавления метаданных времени.
type BaseDocument struct {
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

// TouchUpdatedAt обновляет поле UpdatedAt на текущее время UTC.
func (d *BaseDocument) TouchUpdatedAt() {
	d.UpdatedAt = time.Now().UTC()
}

// SetCreatedAt устанавливает CreatedAt на текущее время UTC,
// если оно еще не было установлено (zero value).
func (d *BaseDocument) SetCreatedAt() {
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now().UTC()
	}
}

// SetTimestamps устанавливает оба поля времени.
// CreatedAt устанавливается только если не был установлен ранее.
// UpdatedAt всегда обновляется на текущее время.
func (d *BaseDocument) SetTimestamps() {
	now := time.Now().UTC()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	d.UpdatedAt = now
}

// UpsertOptions возвращает стандартные опции для upsert операции.
// Использование:
//
//	_, err := collection.UpdateOne(ctx, filter, update, UpsertOptions())
func UpsertOptions() *options.UpdateOneOptionsBuilder {
	return options.UpdateOne().SetUpsert(true)
}

// FindWithPagination возвращает опции для find с пагинацией и сортировкой.
// Параметры:
//   - offset: количество документов для пропуска
//   - limit: максимальное количество возвращаемых документов
//   - sortField: поле для сортировки (например, "created_at")
//   - sortOrder: порядок сортировки (1 = ASC, -1 = DESC)
//
// Использование:
//
//	opts := FindWithPagination(0, 50, "created_at", -1)
//	cursor, err := collection.Find(ctx, filter, opts)
func FindWithPagination(offset, limit int, sortField string, sortOrder int) *options.FindOptionsBuilder {
	return options.Find().
		SetSort(bson.D{{Key: sortField, Value: sortOrder}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))
}

// FindWithPaginationDesc возвращает опции для find с пагинацией и сортировкой по убыванию.
// Это convenience функция для частого случая сортировки по created_at DESC.
func FindWithPaginationDesc(offset, limit int) *options.FindOptionsBuilder {
	return FindWithPagination(offset, limit, "created_at", -1)
}

// CountFilter выполняет подсчет документов с указанным фильтром.
// Возвращает количество документов, соответствующих фильтру.
func CountFilter(ctx context.Context, coll *mongo.Collection, filter bson.M) (int, error) {
	count, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// CountAll выполняет подсчет всех документов в коллекции.
// Это convenience функция для подсчета без фильтра.
func CountAll(ctx context.Context, coll *mongo.Collection) (int, error) {
	return CountFilter(ctx, coll, bson.M{})
}

// DefaultLimit возвращает limit с применением дефолтного значения.
// Если limit <= 0, возвращает defaultLimit.
func DefaultLimit(limit, defaultLimit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	return limit
}

// DefaultLimitWithMax возвращает limit с применением дефолтного и максимального значения.
// Если limit <= 0, возвращает defaultLimit.
// Если limit > maxLimit, возвращает maxLimit.
func DefaultLimitWithMax(limit, defaultLimit, maxLimit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}

// StringPtr возвращает указатель на строку.
// Если строка пустая, возвращает nil.
// Полезно для optional string полей в документах.
func StringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// StringValue возвращает значение строки из указателя.
// Если указатель nil, возвращает пустую строку.
func StringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
