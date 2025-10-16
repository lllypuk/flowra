package workspace

import (
	"context"

	"github.com/lllypuk/teams-up/internal/domain/uuid"
)

// Repository определяет интерфейс для работы с хранилищем Workspace
type Repository interface {
	// FindByID находит рабочее пространство по ID
	FindByID(ctx context.Context, id uuid.UUID) (*Workspace, error)

	// FindByKeycloakGroup находит рабочее пространство по ID группы Keycloak
	FindByKeycloakGroup(ctx context.Context, keycloakGroupID string) (*Workspace, error)

	// Save сохраняет рабочее пространство
	Save(ctx context.Context, workspace *Workspace) error

	// Delete удаляет рабочее пространство
	Delete(ctx context.Context, id uuid.UUID) error

	// List возвращает список рабочих пространств с пагинацией
	List(ctx context.Context, offset, limit int) ([]*Workspace, error)

	// Count возвращает общее количество рабочих пространств
	Count(ctx context.Context) (int, error)

	// FindInviteByToken находит приглашение по токену
	FindInviteByToken(ctx context.Context, token string) (*Invite, error)
}
