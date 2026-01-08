package notification

import (
	"context"
	"time"

	"github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// CommandRepository defines interface for commands (change state) уведомлений
// interface declared on the consumer side (application layer)
type CommandRepository interface {
	// Save saves notification (creation or update)
	Save(ctx context.Context, n *notification.Notification) error

	// SaveBatch saves several уведомлений за one query
	SaveBatch(ctx context.Context, notifications []*notification.Notification) error

	// Delete удаляет notification
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByUserID удаляет all уведомления user
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error

	// DeleteOlderThan удаляет уведомления старше указанной даты
	DeleteOlderThan(ctx context.Context, before time.Time) (int, error)

	// DeleteReadOlderThan удаляет прочитанные уведомления старше указанной даты
	DeleteReadOlderThan(ctx context.Context, before time.Time) (int, error)

	// MarkAsRead отмечает notification as прочитанное
	MarkAsRead(ctx context.Context, id uuid.UUID) error

	// MarkAllAsRead отмечает all уведомления user as прочитанные
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error

	// MarkManyAsRead отмечает several уведомлений as прочитанные
	MarkManyAsRead(ctx context.Context, ids []uuid.UUID) error
}

// QueryRepository defines interface for запросов (only reading) уведомлений
// interface declared on the consumer side (application layer)
type QueryRepository interface {
	// FindByID finds notification по ID
	FindByID(ctx context.Context, id uuid.UUID) (*notification.Notification, error)

	// FindByUserID finds all уведомления user с пагинацией
	FindByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*notification.Notification, error)

	// FindUnreadByUserID finds непрочитанные уведомления user
	FindUnreadByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*notification.Notification, error)

	// FindByType finds уведомления specific type for user
	FindByType(
		ctx context.Context,
		userID uuid.UUID,
		notificationType notification.Type,
		offset, limit int,
	) ([]*notification.Notification, error)

	// FindByResourceID finds уведомления связанные с ресурсом
	FindByResourceID(ctx context.Context, resourceID string) ([]*notification.Notification, error)

	// CountUnreadByUserID returns count unread уведомлений
	CountUnreadByUserID(ctx context.Context, userID uuid.UUID) (int, error)

	// CountByType returns count уведомлений по типам for user
	CountByType(ctx context.Context, userID uuid.UUID) (map[notification.Type]int, error)
}

// Repository combines Command and Query interfaces for convenience
// Used when use case need both types of операций
type Repository interface {
	CommandRepository
	QueryRepository
}
