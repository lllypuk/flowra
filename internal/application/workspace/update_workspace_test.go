package workspace_test

import (
	"context"
	"errors"
	"testing"

	"github.com/flowra/flowra/internal/application/workspace"
	"github.com/flowra/flowra/internal/domain/uuid"
	domainworkspace "github.com/flowra/flowra/internal/domain/workspace"
)

func TestUpdateWorkspaceUseCase_Execute_Success(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewUpdateWorkspaceUseCase(repo)

	// Создаем существующий workspace
	existingWs, _ := domainworkspace.NewWorkspace("Old Name", "keycloak-group-id", uuid.NewUUID())
	_ = repo.Save(context.Background(), existingWs)

	cmd := workspace.UpdateWorkspaceCommand{
		WorkspaceID: existingWs.ID(),
		Name:        "New Name",
		UpdatedBy:   uuid.NewUUID(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Value == nil {
		t.Fatal("expected workspace to be returned")
	}

	if result.Value.Name() != cmd.Name {
		t.Errorf("expected name %s, got %s", cmd.Name, result.Value.Name())
	}
}

func TestUpdateWorkspaceUseCase_Execute_WorkspaceNotFound(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewUpdateWorkspaceUseCase(repo)

	cmd := workspace.UpdateWorkspaceCommand{
		WorkspaceID: uuid.NewUUID(),
		Name:        "New Name",
		UpdatedBy:   uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected error for workspace not found")
	}

	if !errors.Is(err, workspace.ErrWorkspaceNotFound) {
		t.Errorf("expected ErrWorkspaceNotFound, got: %v", err)
	}
}

func TestUpdateWorkspaceUseCase_Validate_MissingName(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewUpdateWorkspaceUseCase(repo)

	cmd := workspace.UpdateWorkspaceCommand{
		WorkspaceID: uuid.NewUUID(),
		Name:        "",
		UpdatedBy:   uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for missing name")
	}
}

func TestUpdateWorkspaceUseCase_Validate_InvalidWorkspaceID(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewUpdateWorkspaceUseCase(repo)

	cmd := workspace.UpdateWorkspaceCommand{
		WorkspaceID: uuid.UUID(""),
		Name:        "New Name",
		UpdatedBy:   uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for invalid workspaceID")
	}
}

func TestUpdateWorkspaceUseCase_Execute_SaveError(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()

	// Создаем существующий workspace
	existingWs, _ := domainworkspace.NewWorkspace("Old Name", "keycloak-group-id", uuid.NewUUID())
	_ = repo.Save(context.Background(), existingWs)

	// Устанавливаем ошибку сохранения
	repo.saveError = errors.New("database error")

	useCase := workspace.NewUpdateWorkspaceUseCase(repo)

	cmd := workspace.UpdateWorkspaceCommand{
		WorkspaceID: existingWs.ID(),
		Name:        "New Name",
		UpdatedBy:   uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected error from save operation")
	}
}
