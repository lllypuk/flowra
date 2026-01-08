package workspace_test

import (
	"context"
	"testing"

	"github.com/lllypuk/flowra/internal/application/workspace"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

func TestListUserWorkspacesUseCase_Execute_Success(t *testing.T) {
	// Arrange
	keycloakClient := newMockKeycloakClient()
	useCase := workspace.NewListUserWorkspacesUseCase(keycloakClient)

	query := workspace.ListUserWorkspacesQuery{
		UserID: uuid.NewUUID(),
		Offset: 0,
		Limit:  10,
	}

	// Act
	result, err := useCase.Execute(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Временно checking only that возвращается empty list
	// TODO: Add полноценные tests after реализации GetUserGroups
	if result.Workspaces == nil {
		t.Error("expected workspaces to be initialized")
	}

	if result.Offset != query.Offset {
		t.Errorf("expected offset %d, got %d", query.Offset, result.Offset)
	}

	if result.Limit != query.Limit {
		t.Errorf("expected limit %d, got %d", query.Limit, result.Limit)
	}
}

func TestListUserWorkspacesUseCase_Validate_InvalidUserID(t *testing.T) {
	// Arrange
	keycloakClient := newMockKeycloakClient()
	useCase := workspace.NewListUserWorkspacesUseCase(keycloakClient)

	query := workspace.ListUserWorkspacesQuery{
		UserID: uuid.UUID(""),
		Offset: 0,
		Limit:  10,
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for invalid userID")
	}
}

func TestListUserWorkspacesUseCase_Validate_NegativeOffset(t *testing.T) {
	// Arrange
	keycloakClient := newMockKeycloakClient()
	useCase := workspace.NewListUserWorkspacesUseCase(keycloakClient)

	query := workspace.ListUserWorkspacesQuery{
		UserID: uuid.NewUUID(),
		Offset: -1,
		Limit:  10,
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for negative offset")
	}
}

func TestListUserWorkspacesUseCase_Validate_InvalidLimit(t *testing.T) {
	// Arrange
	keycloakClient := newMockKeycloakClient()
	useCase := workspace.NewListUserWorkspacesUseCase(keycloakClient)

	query := workspace.ListUserWorkspacesQuery{
		UserID: uuid.NewUUID(),
		Offset: 0,
		Limit:  0,
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for invalid limit")
	}
}

func TestListUserWorkspacesUseCase_Validate_LimitTooLarge(t *testing.T) {
	// Arrange
	keycloakClient := newMockKeycloakClient()
	useCase := workspace.NewListUserWorkspacesUseCase(keycloakClient)

	query := workspace.ListUserWorkspacesQuery{
		UserID: uuid.NewUUID(),
		Offset: 0,
		Limit:  101,
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for limit too large")
	}
}
