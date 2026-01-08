package tag

import (
	"context"

	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/user"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// ChatRepository defines interface for workы с Chat aggregate.
// interface declared здесь (on стороне потребителя - tag domain),
// following idiomatic Go approach.
type ChatRepository interface {
	Load(ctx context.Context, chatID uuid.UUID) (*chat.Chat, error)
	Save(ctx context.Context, chat *chat.Chat) error
	GetEvents(ctx context.Context, chatID uuid.UUID) ([]event.DomainEvent, error)
}

// UserRepository defines interface for workы с userелями.
// interface declared здесь (on стороне потребителя - tag domain),
// following idiomatic Go approach.
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*user.User, error)
}

// MessageRepository defines interface for workы с messagesми.
// interface declared здесь (on стороне потребителя - tag domain),
// following idiomatic Go approach.
type MessageRepository interface {
	Save(ctx context.Context, message *message.Message) error
}
