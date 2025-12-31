package message

import (
	"context"

	"github.com/lllypuk/flowra/internal/domain/message"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Pagination представляет параметры пагинации для запросов сообщений
type Pagination struct {
	Limit  int
	Offset int
}

// CommandRepository определяет интерфейс для команд (изменение состояния) сообщений
// Интерфейс объявлен на стороне потребителя (application layer)
type CommandRepository interface {
	// Save сохраняет сообщение (создание или обновление)
	Save(ctx context.Context, msg *message.Message) error

	// Delete физически удаляет сообщение
	Delete(ctx context.Context, id uuid.UUID) error

	// AddReaction добавляет реакцию к сообщению
	AddReaction(ctx context.Context, messageID uuid.UUID, emojiCode string, userID uuid.UUID) error

	// RemoveReaction удаляет реакцию с сообщения
	RemoveReaction(ctx context.Context, messageID uuid.UUID, emojiCode string, userID uuid.UUID) error
}

// QueryRepository определяет интерфейс для запросов (только чтение) сообщений
// Интерфейс объявлен на стороне потребителя (application layer)
type QueryRepository interface {
	// FindByID находит сообщение по ID
	FindByID(ctx context.Context, id uuid.UUID) (*message.Message, error)

	// FindByChatID находит сообщения в чате с пагинацией
	// Сообщения возвращаются отсортированными по времени создания (от новых к старым)
	FindByChatID(ctx context.Context, chatID uuid.UUID, pagination Pagination) ([]*message.Message, error)

	// FindThread находит все ответы в треде
	// Возвращает сообщения, у которых ParentMessageID равен указанному
	FindThread(ctx context.Context, parentMessageID uuid.UUID) ([]*message.Message, error)

	// CountByChatID возвращает количество сообщений в чате
	CountByChatID(ctx context.Context, chatID uuid.UUID) (int, error)

	// CountThreadReplies возвращает количество ответов в треде
	CountThreadReplies(ctx context.Context, parentMessageID uuid.UUID) (int, error)

	// GetReactionUsers возвращает пользователей, поставивших определенную реакцию
	GetReactionUsers(ctx context.Context, messageID uuid.UUID, emojiCode string) ([]uuid.UUID, error)

	// SearchInChat ищет сообщения в чате по тексту
	SearchInChat(ctx context.Context, chatID uuid.UUID, query string, offset, limit int) ([]*message.Message, error)

	// FindByAuthor находит сообщения автора в чате
	FindByAuthor(
		ctx context.Context,
		chatID uuid.UUID,
		authorID uuid.UUID,
		offset, limit int,
	) ([]*message.Message, error)
}

// Repository объединяет Command и Query интерфейсы для удобства
// Используется когда use case нужны оба типа операций
type Repository interface {
	CommandRepository
	QueryRepository
}
