package task

import (
	"context"

	"github.com/lllypuk/teams-up/internal/domain/uuid"
)

// Repository определяет интерфейс для работы с хранилищем TaskEntity
type Repository interface {
	// FindByID находит задачу по ID
	FindByID(ctx context.Context, id uuid.UUID) (*Entity, error)

	// FindByChatID находит задачу по ID чата (TaskEntity.ID == ChatID)
	FindByChatID(ctx context.Context, chatID uuid.UUID) (*Entity, error)

	// FindByStatus находит задачи по статусу с пагинацией
	FindByStatus(ctx context.Context, status Status, offset, limit int) ([]*Entity, error)

	// FindByAssignee находит задачи назначенные на пользователя
	FindByAssignee(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*Entity, error)

	// FindByType находит задачи по типу (task/bug/epic)
	FindByType(ctx context.Context, entityType EntityType, offset, limit int) ([]*Entity, error)

	// FindOverdue находит просроченные задачи
	FindOverdue(ctx context.Context, offset, limit int) ([]*Entity, error)

	// GetBoard возвращает задачи для канбан-доски (сгруппированные по статусам)
	GetBoard(ctx context.Context) (map[Status][]*Entity, error)

	// Save сохраняет задачу
	Save(ctx context.Context, task *Entity) error

	// Delete удаляет задачу
	Delete(ctx context.Context, id uuid.UUID) error

	// Count возвращает общее количество задач
	Count(ctx context.Context) (int, error)

	// CountByStatus возвращает количество задач по статусу
	CountByStatus(ctx context.Context, status Status) (int, error)
}
