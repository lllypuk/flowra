package message

import (
	"context"

	"github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Pagination represents parameters пагинации for запросов сообщений
type Pagination struct {
	Limit  int
	Offset int
}

// CommandRepository defines interface for commands (change state) сообщений
// interface declared on the consumer side (application layer)
type CommandRepository interface {
	// Save saves message (creation or update)
	Save(ctx context.Context, msg *message.Message) error

	// Delete физически удаляет message
	Delete(ctx context.Context, id uuid.UUID) error

	// AddReaction добавляет реакцию to сообщению
	AddReaction(ctx context.Context, messageID uuid.UUID, emojiCode string, userID uuid.UUID) error

	// RemoveReaction удаляет реакцию с messages
	RemoveReaction(ctx context.Context, messageID uuid.UUID, emojiCode string, userID uuid.UUID) error
}

// QueryRepository defines interface for запросов (only reading) сообщений
// interface declared on the consumer side (application layer)
type QueryRepository interface {
	// FindByID finds message по ID
	FindByID(ctx context.Context, id uuid.UUID) (*message.Message, error)

	// FindByChatID finds messages in чате с пагинацией
	// Сообщения возвращаются отсортированными по time creating (от New to старым)
	FindByChatID(ctx context.Context, chatID uuid.UUID, pagination Pagination) ([]*message.Message, error)

	// FindThread finds all responses in треде
	// returns messages, у которых ParentMessageID equal указанному
	FindThread(ctx context.Context, parentMessageID uuid.UUID) ([]*message.Message, error)

	// CountByChatID returns count сообщений in чате
	CountByChatID(ctx context.Context, chatID uuid.UUID) (int, error)

	// CountThreadReplies returns count responseов in треде
	CountThreadReplies(ctx context.Context, parentMessageID uuid.UUID) (int, error)

	// GetReactionUsers returns users, поставивших определенную реакцию
	GetReactionUsers(ctx context.Context, messageID uuid.UUID, emojiCode string) ([]uuid.UUID, error)

	// SearchInChat ищет messages in чате по textу
	SearchInChat(ctx context.Context, chatID uuid.UUID, query string, offset, limit int) ([]*message.Message, error)

	// FindByAuthor finds messages автора in чате
	FindByAuthor(
		ctx context.Context,
		chatID uuid.UUID,
		authorID uuid.UUID,
		offset, limit int,
	) ([]*message.Message, error)
}

// Repository combines Command and Query interfaces for convenience
// Used when use case need both types of операций
type Repository interface {
	CommandRepository
	QueryRepository
}
