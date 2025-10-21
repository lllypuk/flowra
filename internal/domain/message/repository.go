package message

import (
	"context"

	"github.com/flowra/flowra/internal/domain/uuid"
)

// Pagination параметры пагинации для списка сообщений
type Pagination struct {
	Limit  int
	Offset int
}

// Repository определяет интерфейс репозитория сообщений
type Repository interface {
	// FindByID находит сообщение по ID
	FindByID(ctx context.Context, id uuid.UUID) (*Message, error)

	// FindByChatID находит сообщения в чате с пагинацией
	// Сообщения возвращаются отсортированными по времени создания (от новых к старым)
	FindByChatID(ctx context.Context, chatID uuid.UUID, pagination Pagination) ([]*Message, error)

	// FindThread находит все ответы в треде
	// Возвращает сообщения, у которых ParentMessageID равен указанному
	FindThread(ctx context.Context, parentMessageID uuid.UUID) ([]*Message, error)

	// CountByChatID возвращает количество сообщений в чате
	CountByChatID(ctx context.Context, chatID uuid.UUID) (int, error)

	// Save сохраняет сообщение (создание или обновление)
	Save(ctx context.Context, message *Message) error

	// Delete физически удаляет сообщение (используется редко, обычно используется soft delete через Message.Delete)
	Delete(ctx context.Context, id uuid.UUID) error
}
