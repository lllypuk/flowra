package notification

import (
	"context"
	"time"

	"github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// CommandRepository определяет интерфейс для команд (изменение состояния) уведомлений
// Интерфейс объявлен на стороне потребителя (application layer)
type CommandRepository interface {
	// Save сохраняет уведомление (создание или обновление)
	Save(ctx context.Context, n *notification.Notification) error

	// SaveBatch сохраняет несколько уведомлений за один запрос
	SaveBatch(ctx context.Context, notifications []*notification.Notification) error

	// Delete удаляет уведомление
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByUserID удаляет все уведомления пользователя
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error

	// DeleteOlderThan удаляет уведомления старше указанной даты
	DeleteOlderThan(ctx context.Context, before time.Time) (int, error)

	// DeleteReadOlderThan удаляет прочитанные уведомления старше указанной даты
	DeleteReadOlderThan(ctx context.Context, before time.Time) (int, error)

	// MarkAsRead отмечает уведомление как прочитанное
	MarkAsRead(ctx context.Context, id uuid.UUID) error

	// MarkAllAsRead отмечает все уведомления пользователя как прочитанные
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error

	// MarkManyAsRead отмечает несколько уведомлений как прочитанные
	MarkManyAsRead(ctx context.Context, ids []uuid.UUID) error
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

	// FindByType находит уведомления определенного типа для пользователя
	FindByType(
		ctx context.Context,
		userID uuid.UUID,
		notificationType notification.Type,
		offset, limit int,
	) ([]*notification.Notification, error)

	// FindByResourceID находит уведомления связанные с ресурсом
	FindByResourceID(ctx context.Context, resourceID string) ([]*notification.Notification, error)

	// CountUnreadByUserID возвращает количество непрочитанных уведомлений
	CountUnreadByUserID(ctx context.Context, userID uuid.UUID) (int, error)

	// CountByType возвращает количество уведомлений по типам для пользователя
	CountByType(ctx context.Context, userID uuid.UUID) (map[notification.Type]int, error)
}

// Repository объединяет Command и Query интерфейсы для удобства
// Используется когда use case нужны оба типа операций
type Repository interface {
	CommandRepository
	QueryRepository
}
