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
	// DefaultPaginationLimit — defoltnyy limit for paginatsii zaprosov.
	DefaultPaginationLimit = 50

	// MaxPaginationLimit — maksimalnyy limit for paginatsii zaprosov.
	MaxPaginationLimit = 100
)

// HandleMongoError preobrazuet error MongoDB in domennuyu error.
// returns:
// - nil if err == nil
// - errs.ErrNotFound if dokument not nayden
// - errs.ErrAlreadyExists if narushen unique constraint
// - wrapped error for ostalnyh sluchaev
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

// BaseDocument contains obschie fields for all dokumentov MongoDB.
// ispolzuetsya as vstraivaemaya struct for add metadannyh time.
type BaseDocument struct {
	CreatedAt time.Time `bson:"created_at"`
	UpdatedAt time.Time `bson:"updated_at"`
}

// TouchUpdatedAt obnovlyaet field UpdatedAt on current time UTC.
func (d *BaseDocument) TouchUpdatedAt() {
	d.UpdatedAt = time.Now().UTC()
}

// SetCreatedAt sets CreatedAt on current time UTC,
// if ono esche not bylo ustanovleno (zero value).
func (d *BaseDocument) SetCreatedAt() {
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now().UTC()
	}
}

// SetTimestamps sets oba fields time.
// CreatedAt sets only if not byl ustanovlen ranee.
// UpdatedAt always obnovlyaetsya on current time.
func (d *BaseDocument) SetTimestamps() {
	now := time.Now().UTC()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	d.UpdatedAt = now
}

// UpsertOptions returns standartnye optsii for upsert operatsii.
// use:
//
//	_, err := collection.UpdateOne(ctx, filter, update, UpsertOptions())
func UpsertOptions() *options.UpdateOneOptionsBuilder {
	return options.UpdateOne().SetUpsert(true)
}

// FindWithPagination returns optsii for find s paginatsiey and sortirovkoy.
// parameters:
// - offset: count dokumentov for propuska
// - limit: maximum count returned documents
// - sortField: field for sortirovki (naprimer, "created_at")
// - sortOrder: order sortirovki (1 = ASC, -1 = DESC)
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

// FindWithPaginationDesc returns optsii for find s paginatsiey and sortirovkoy po ubyvaniyu.
// eto convenience function for chastogo sluchaya sortirovki po created_at DESC.
func FindWithPaginationDesc(offset, limit int) *options.FindOptionsBuilder {
	return FindWithPagination(offset, limit, "created_at", -1)
}

// CountFilter performs podschet dokumentov s ukazannym filtrom.
// returns count dokumentov, response filtru.
func CountFilter(ctx context.Context, coll *mongo.Collection, filter bson.M) (int, error) {
	count, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// CountAll performs podschet all dokumentov in kollektsii.
// eto convenience function for podscheta bez filtra.
func CountAll(ctx context.Context, coll *mongo.Collection) (int, error) {
	return CountFilter(ctx, coll, bson.M{})
}

// DefaultLimit returns limit s applying defoltnogo values.
// if limit <= 0, returns defaultLimit.
func DefaultLimit(limit, defaultLimit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	return limit
}

// DefaultLimitWithMax returns limit s applying defoltnogo and maksimalnogo values.
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

// StringPtr returns pointer on stroku.
// if stroka pustaya, returns nil.
// polezno for optional string poley in dokumentah.
func StringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// StringValue returns value stroki from ukazatelya.
// if pointer nil, returns pustuyu stroku.
func StringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
