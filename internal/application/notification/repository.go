package notification

import (
	"context"
	"time"

	"github.com/lllypuk/flowra/internal/domain/notification"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// CommandRepository defines interface for commands (change state) uvedomleniy
// interface declared on the consumer side (application layer)
type CommandRepository interface {
	// Save saves notification (creation or update)
	Save(ctx context.Context, n *notification.Notification) error

	// SaveBatch saves several uvedomleniy za one query
	SaveBatch(ctx context.Context, notifications []*notification.Notification) error

	// Delete udalyaet notification
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByUserID udalyaet all uvedomleniya user
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error

	// DeleteOlderThan udalyaet uvedomleniya starshe ukazannoy daty
	DeleteOlderThan(ctx context.Context, before time.Time) (int, error)

	// DeleteReadOlderThan udalyaet prochitannye uvedomleniya starshe ukazannoy daty
	DeleteReadOlderThan(ctx context.Context, before time.Time) (int, error)

	// MarkAsRead otmechaet notification as prochitannoe
	MarkAsRead(ctx context.Context, id uuid.UUID) error

	// MarkAllAsRead otmechaet all uvedomleniya user as prochitannye
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error

	// MarkManyAsRead otmechaet several uvedomleniy as prochitannye
	MarkManyAsRead(ctx context.Context, ids []uuid.UUID) error
}

// QueryRepository defines interface for zaprosov (only reading) uvedomleniy
// interface declared on the consumer side (application layer)
type QueryRepository interface {
	// FindByID finds notification po ID
	FindByID(ctx context.Context, id uuid.UUID) (*notification.Notification, error)

	// FindByUserID finds all uvedomleniya user s paginatsiey
	FindByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*notification.Notification, error)

	// FindUnreadByUserID finds neprochitannye uvedomleniya user
	FindUnreadByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*notification.Notification, error)

	// FindByType finds uvedomleniya specific type for user
	FindByType(
		ctx context.Context,
		userID uuid.UUID,
		notificationType notification.Type,
		offset, limit int,
	) ([]*notification.Notification, error)

	// FindByResourceID finds uvedomleniya svyazannye s resursom
	FindByResourceID(ctx context.Context, resourceID string) ([]*notification.Notification, error)

	// CountUnreadByUserID returns count unread uvedomleniy
	CountUnreadByUserID(ctx context.Context, userID uuid.UUID) (int, error)

	// CountByType returns count uvedomleniy po tipam for user
	CountByType(ctx context.Context, userID uuid.UUID) (map[notification.Type]int, error)
}

// Repository combines Command and Query interfaces for convenience
// Used when use case need both types of operatsiy
type Repository interface {
	CommandRepository
	QueryRepository
}
