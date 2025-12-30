package notification

import (
	"context"

	"github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// CommandRepository определяет интерфейс для команд (изменение состояния) уведомлений
// Интерфейс объявлен на стороне потребителя (application layer)
type CommandRepository interface {
	// Save сохраняет уведомление (создание или обновление)
	Save(ctx context.Context, n *notification.Notification) error

	// Delete удаляет уведомление
	Delete(ctx context.Context, id uuid.UUID) error

	// MarkAsRead отмечает уведомление как прочитанное
	MarkAsRead(ctx context.Context, id uuid.UUID) error

	// MarkAllAsRead отмечает все уведомления пользователя как прочитанные
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
}

// QueryRepository определяет интерфейс для запросов (только чтение) уведомлений
// Интерфейс объявлен на стороне потребителя (application layer)
type QueryRepository interface {
	// FindByID находит уведомление по ID
	FindByID(ctx context.Context, id uuid.UUID) (*notification.Notification, error)

	// FindByUserID находит все уведомления пользователя с пагинацией
	FindByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*notification.Notification, error)

	// FindUnreadByUserID находит непрочитанные уведомления пользователя
	FindUnreadByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*notification.Notification, error)

	// CountUnreadByUserID возвращает количество непрочитанных уведомлений
	CountUnreadByUserID(ctx context.Context, userID uuid.UUID) (int, error)
}

// Repository объединяет Command и Query интерфейсы для удобства
// Используется когда use case нужны оба типа операций
type Repository interface {
	CommandRepository
	QueryRepository
}
