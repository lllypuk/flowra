package notification

import (
	"context"

	"github.com/lllypuk/teams-up/internal/domain/uuid"
)

// Repository определяет интерфейс для работы с хранилищем Notification
type Repository interface {
	// FindByID находит уведомление по ID
	FindByID(ctx context.Context, id uuid.UUID) (*Notification, error)

	// FindByUserID находит все уведомления пользователя с пагинацией
	FindByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*Notification, error)

	// FindUnreadByUserID находит непрочитанные уведомления пользователя
	FindUnreadByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*Notification, error)

	// CountUnreadByUserID возвращает количество непрочитанных уведомлений
	CountUnreadByUserID(ctx context.Context, userID uuid.UUID) (int, error)

	// Save сохраняет уведомление
	Save(ctx context.Context, notification *Notification) error

	// Delete удаляет уведомление
	Delete(ctx context.Context, id uuid.UUID) error

	// DeleteByUserID удаляет все уведомления пользователя
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}
