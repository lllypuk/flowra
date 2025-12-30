package mongodb

import (
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/lllypuk/flowra/internal/domain/errs"
)

// HandleMongoError преобразует ошибку MongoDB в доменную ошибку
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
