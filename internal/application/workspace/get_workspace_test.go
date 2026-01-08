package workspace_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lllypuk/flowra/internal/application/workspace"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	domainworkspace "github.com/lllypuk/flowra/internal/domain/workspace"
)

func TestGetWorkspaceUseCase_Execute_Success(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewGetWorkspaceUseCase(repo)

	// Creating workspace
	ws, _ := domainworkspace.NewWorkspace("Test Workspace", "", "keycloak-group-id", uuid.NewUUID())
	_ = repo.Save(context.Background(), ws)

	query := workspace.GetWorkspaceQuery{
		WorkspaceID: ws.ID(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Value == nil {
		t.Fatal("expected workspace to be returned")
	}

	if result.Value.ID() != ws.ID() {
		t.Errorf("expected workspace ID %s, got %s", ws.ID(), result.Value.ID())
	}

	if result.Value.Name() != ws.Name() {
		t.Errorf("expected workspace name %s, got %s", ws.Name(), result.Value.Name())
	}
}

func TestGetWorkspaceUseCase_Execute_WorkspaceNotFound(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewGetWorkspaceUseCase(repo)

	query := workspace.GetWorkspaceQuery{
		WorkspaceID: uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if err == nil {
		t.Fatal("expected error for workspace not found")
	}

	if !errors.Is(err, workspace.ErrWorkspaceNotFound) {
		t.Errorf("expected ErrWorkspaceNotFound, got: %v", err)
	}
}

func TestGetWorkspaceUseCase_Validate_InvalidWorkspaceID(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewGetWorkspaceUseCase(repo)

	query := workspace.GetWorkspaceQuery{
		WorkspaceID: uuid.UUID(""),
	}

	// Act
	_, err := useCase.Execute(context.Background(), query)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for invalid workspaceID")
	}
}
