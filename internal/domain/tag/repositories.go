package tag

import (
	"context"

	"github.com/flowra/flowra/internal/domain/chat"
	"github.com/flowra/flowra/internal/domain/event"
	"github.com/flowra/flowra/internal/domain/message"
	"github.com/flowra/flowra/internal/domain/user"
	"github.com/flowra/flowra/internal/domain/uuid"
)

// ChatRepository определяет интерфейс для работы с Chat aggregate.
// Интерфейс объявлен здесь (на стороне потребителя - tag domain),
// следуя идиоматичному Go подходу.
type ChatRepository interface {
	Load(ctx context.Context, chatID uuid.UUID) (*chat.Chat, error)
	Save(ctx context.Context, chat *chat.Chat) error
	GetEvents(ctx context.Context, chatID uuid.UUID) ([]event.DomainEvent, error)
}

// UserRepository определяет интерфейс для работы с пользователями.
// Интерфейс объявлен здесь (на стороне потребителя - tag domain),
// следуя идиоматичному Go подходу.
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*user.User, error)
}

// MessageRepository определяет интерфейс для работы с сообщениями.
// Интерфейс объявлен здесь (на стороне потребителя - tag domain),
// следуя идиоматичному Go подходу.
type MessageRepository interface {
	Save(ctx context.Context, message *message.Message) error
}
