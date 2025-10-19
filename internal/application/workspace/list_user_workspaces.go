package workspace

import (
	"context"

	"github.com/lllypuk/teams-up/internal/application/shared"
	"github.com/lllypuk/teams-up/internal/domain/workspace"
)

// ListUserWorkspacesUseCase - use case для получения списка workspace пользователя
type ListUserWorkspacesUseCase struct {
	shared.BaseUseCase

	keycloakClient KeycloakClient
	// Для получения workspaces пользователя нужно:
	// 1. Получить список групп пользователя из Keycloak
	// 2. Найти workspaces по этим группам
	// Но в текущей реализации Repository нет метода FindByKeycloakGroups
	// Поэтому используем существующий метод FindByKeycloakGroup для каждой группы
}

// NewListUserWorkspacesUseCase создает новый ListUserWorkspacesUseCase
func NewListUserWorkspacesUseCase(keycloakClient KeycloakClient) *ListUserWorkspacesUseCase {
	return &ListUserWorkspacesUseCase{
		keycloakClient: keycloakClient,
	}
}

// Execute выполняет получение списка workspace пользователя
func (uc *ListUserWorkspacesUseCase) Execute(
	ctx context.Context,
	query ListUserWorkspacesQuery,
) (ListResult, error) {
	// Валидация контекста
	if err := uc.ValidateContext(ctx); err != nil {
		return ListResult{}, uc.WrapError("validate context", err)
	}

	// Валидация запроса
	if err := uc.validate(query); err != nil {
		return ListResult{}, uc.WrapError("validation failed", err)
	}

	// TODO: Реализация требует дополнительных методов в KeycloakClient:
	// - GetUserGroups(ctx, userID) ([]string, error) - получить список групп пользователя
	// И в Repository:
	// - FindByKeycloakGroups(ctx, groupIDs []string) ([]*Workspace, error)

	// Временная заглушка для компиляции
	// В реальном проекте нужно:
	// 1. Добавить GetUserGroups в KeycloakClient
	// 2. Добавить FindByKeycloakGroups в Repository
	// 3. Реализовать логику получения и фильтрации workspaces

	return ListResult{
		Workspaces: []*workspace.Workspace{},
		TotalCount: 0,
		Offset:     query.Offset,
		Limit:      query.Limit,
	}, nil
}

// validate проверяет валидность запроса
func (uc *ListUserWorkspacesUseCase) validate(query ListUserWorkspacesQuery) error {
	if err := shared.ValidateUUID("userID", query.UserID); err != nil {
		return err
	}
	if err := shared.ValidateNonNegative("offset", query.Offset); err != nil {
		return err
	}
	if err := shared.ValidatePositive("limit", query.Limit); err != nil {
		return err
	}
	const (
		minLimit = 1
		maxLimit = 100
	)
	if err := shared.ValidateRange("limit", query.Limit, minLimit, maxLimit); err != nil {
		return err
	}
	return nil
}
