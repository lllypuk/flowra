package workspace

import (
	"context"

	"github.com/lllypuk/flowra/internal/application/appcore"
	"github.com/lllypuk/flowra/internal/domain/workspace"
)

// ListUserWorkspacesUseCase - use case for receiv list workspace user
type ListUserWorkspacesUseCase struct {
	appcore.BaseUseCase

	keycloakClient KeycloakClient
	// for receiv workspaces user nuzhno:
	// 1. get list user groups from Keycloak
	// 2. find workspaces for these groups
	// no in tekuschey realizatsii Repository no metoda FindByKeycloakGroups
	// it is ispolzuem existing method FindByKeycloakGroup for kazhdoy groups
}

// NewListUserWorkspacesUseCase creates New ListUserWorkspacesUseCase
func NewListUserWorkspacesUseCase(keycloakClient KeycloakClient) *ListUserWorkspacesUseCase {
	return &ListUserWorkspacesUseCase{
		keycloakClient: keycloakClient,
	}
}

// Execute performs retrieval list workspace user
func (uc *ListUserWorkspacesUseCase) Execute(
	ctx context.Context,
	query ListUserWorkspacesQuery,
) (ListResult, error) {
	// context validation
	if err := uc.ValidateContext(ctx); err != nil {
		return ListResult{}, uc.WrapError("validate context", err)
	}

	// validation request
	if err := uc.validate(query); err != nil {
		return ListResult{}, uc.WrapError("validation failed", err)
	}

	// TODO: realizatsiya trebuet dopolnitelnyh methods in KeycloakClient:
	// - GetUserGroups(ctx, userID) ([]string, error) - get list user groups
	// and in Repository:
	// - FindByKeycloakGroups(ctx, groupIDs []string) ([]*Workspace, error)

	// vremennaya zaglushka for kompilyatsii
	// in realnom proekte nuzhno:
	// 1. Add GetUserGroups in KeycloakClient
	// 2. Add FindByKeycloakGroups in Repository
	// 3. realizovat logiku receiv and filtering workspaces

	return ListResult{
		Workspaces: []*workspace.Workspace{},
		TotalCount: 0,
		Offset:     query.Offset,
		Limit:      query.Limit,
	}, nil
}

// validate validates request
func (uc *ListUserWorkspacesUseCase) validate(query ListUserWorkspacesQuery) error {
	if err := appcore.ValidateUUID("userID", query.UserID); err != nil {
		return err
	}
	if err := appcore.ValidateNonNegative("offset", query.Offset); err != nil {
		return err
	}
	if err := appcore.ValidatePositive("limit", query.Limit); err != nil {
		return err
	}
	const (
		minLimit = 1
		maxLimit = 100
	)
	if err := appcore.ValidateRange("limit", query.Limit, minLimit, maxLimit); err != nil {
		return err
	}
	return nil
}
