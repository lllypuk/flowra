package message

import (
	"context"

	"github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Pagination represents parameters paginatsii for zaprosov soobscheniy
type Pagination struct {
	Limit  int
	Offset int
}

// CommandRepository defines interface for commands (change state) soobscheniy
// interface declared on the consumer side (application layer)
type CommandRepository interface {
	// Save saves message (creation or update)
	Save(ctx context.Context, msg *message.Message) error

	// Delete fizicheski udalyaet message
	Delete(ctx context.Context, id uuid.UUID) error

	// AddReaction adds reaction to soobscheniyu
	AddReaction(ctx context.Context, messageID uuid.UUID, emojiCode string, userID uuid.UUID) error

	// RemoveReaction udalyaet reaction s messages
	RemoveReaction(ctx context.Context, messageID uuid.UUID, emojiCode string, userID uuid.UUID) error
}

// QueryRepository defines interface for zaprosov (only reading) soobscheniy
// interface declared on the consumer side (application layer)
type QueryRepository interface {
	// FindByID finds message po ID
	FindByID(ctx context.Context, id uuid.UUID) (*message.Message, error)

	// FindByChatID finds messages in chate s paginatsiey
	// soobscheniya vozvraschayutsya otsortirovannymi po time creating (ot New to starym)
	FindByChatID(ctx context.Context, chatID uuid.UUID, pagination Pagination) ([]*message.Message, error)

	// FindThread finds all responses in thread
	// returns messages, u kotoryh ParentMessageID equal ukazannomu
	FindThread(ctx context.Context, parentMessageID uuid.UUID) ([]*message.Message, error)

	// CountByChatID returns count soobscheniy in chate
	CountByChatID(ctx context.Context, chatID uuid.UUID) (int, error)

	// CountThreadReplies returns count response in thread
	CountThreadReplies(ctx context.Context, parentMessageID uuid.UUID) (int, error)

	// GetReactionUsers returns users, postavivshih opredelennuyu reaction
	GetReactionUsers(ctx context.Context, messageID uuid.UUID, emojiCode string) ([]uuid.UUID, error)

	// SearchInChat ischet messages in chate po text
	SearchInChat(ctx context.Context, chatID uuid.UUID, query string, offset, limit int) ([]*message.Message, error)

	// FindByAuthor finds messages avtora in chate
	FindByAuthor(
		ctx context.Context,
		chatID uuid.UUID,
		authorID uuid.UUID,
		offset, limit int,
	) ([]*message.Message, error)
}

// Repository combines Command and Query interfaces for convenience
// Used when use case need both types of operatsiy
type Repository interface {
	CommandRepository
	QueryRepository
}
