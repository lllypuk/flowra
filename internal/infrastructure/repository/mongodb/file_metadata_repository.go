package mongodb

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// FileMetadata holds ownership information for an uploaded file.
type FileMetadata struct {
	FileID     uuid.UUID
	ChatID     uuid.UUID
	UploaderID uuid.UUID
	UploadedAt time.Time
}

// fileMetadataDocument is the MongoDB representation of file metadata.
type fileMetadataDocument struct {
	FileID     string    `bson:"file_id"`
	ChatID     string    `bson:"chat_id"`
	UploaderID string    `bson:"uploader_id"`
	UploadedAt time.Time `bson:"uploaded_at"`
}

// MongoFileMetadataRepository implements file metadata storage using MongoDB.
type MongoFileMetadataRepository struct {
	collection *mongo.Collection
	logger     *slog.Logger
}

// FileMetadataRepoOption configures MongoFileMetadataRepository.
type FileMetadataRepoOption func(*MongoFileMetadataRepository)

// WithFileMetadataRepoLogger sets the logger for file metadata repository.
func WithFileMetadataRepoLogger(logger *slog.Logger) FileMetadataRepoOption {
	return func(r *MongoFileMetadataRepository) {
		r.logger = logger
	}
}

// NewMongoFileMetadataRepository creates a new file metadata repository.
func NewMongoFileMetadataRepository(
	collection *mongo.Collection,
	opts ...FileMetadataRepoOption,
) *MongoFileMetadataRepository {
	r := &MongoFileMetadataRepository{
		collection: collection,
		logger:     slog.Default(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Save stores file metadata.
func (r *MongoFileMetadataRepository) Save(ctx context.Context, meta FileMetadata) error {
	if meta.FileID.IsZero() {
		return errs.ErrInvalidInput
	}

	doc := fileMetadataDocument{
		FileID:     meta.FileID.String(),
		ChatID:     meta.ChatID.String(),
		UploaderID: meta.UploaderID.String(),
		UploadedAt: meta.UploadedAt,
	}

	_, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to save file metadata",
			slog.String("file_id", meta.FileID.String()),
			slog.String("error", err.Error()),
		)
		return HandleMongoError(err, "file_metadata")
	}

	return nil
}

// FindByFileID retrieves file metadata by file ID.
func (r *MongoFileMetadataRepository) FindByFileID(ctx context.Context, fileID uuid.UUID) (*FileMetadata, error) {
	if fileID.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	filter := bson.M{"file_id": fileID.String()}
	var doc fileMetadataDocument
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		return nil, HandleMongoError(err, "file_metadata")
	}

	return &FileMetadata{
		FileID:     uuid.UUID(doc.FileID),
		ChatID:     uuid.UUID(doc.ChatID),
		UploaderID: uuid.UUID(doc.UploaderID),
		UploadedAt: doc.UploadedAt,
	}, nil
}
