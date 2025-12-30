package workspace

import (
	"context"

	"github.com/lllypuk/flowra/internal/domain/uuid"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// CommandRepository определяет интерфейс для команд (изменение состояния) рабочих пространств
// Интерфейс объявлен на стороне потребителя (application layer)
type CommandRepository interface {
	// Save сохраняет рабочее пространство (создание или обновление)
	Save(ctx context.Context, ws *workspace.Workspace) error

	// Delete удаляет рабочее пространство
	Delete(ctx context.Context, id uuid.UUID) error
}

// QueryRepository определяет интерфейс для запросов (только чтение) рабочих пространств
// Интерфейс объявлен на стороне потребителя (application layer)
type QueryRepository interface {
	// FindByID находит рабочее пространство по ID
	FindByID(ctx context.Context, id uuid.UUID) (*workspace.Workspace, error)

	// FindByKeycloakGroup находит рабочее пространство по ID группы Keycloak
	FindByKeycloakGroup(ctx context.Context, keycloakGroupID string) (*workspace.Workspace, error)

	// List возвращает список рабочих пространств с пагинацией
	List(ctx context.Context, offset, limit int) ([]*workspace.Workspace, error)

	// Count возвращает общее количество рабочих пространств
	Count(ctx context.Context) (int, error)

	// FindInviteByToken находит приглашение по токену
	FindInviteByToken(ctx context.Context, token string) (*workspace.Invite, error)
}

// Repository объединяет Command и Query интерфейсы для удобства
// Используется когда use case нужны оба типа операций
type Repository interface {
	CommandRepository
	QueryRepository
}
