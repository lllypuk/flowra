package workspace_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lllypuk/flowra/internal/application/workspace"
	"github.com/lllypuk/flowra/internal/domain/uuid"
	domainworkspace "github.com/lllypuk/flowra/internal/domain/workspace"
)

func TestCreateInviteUseCase_Execute_Success(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewCreateInviteUseCase(repo)

	// Creating existing workspace
	existingWs, _ := domainworkspace.NewWorkspace("Test Workspace", "", "keycloak-group-id", uuid.NewUUID())
	_ = repo.Save(context.Background(), existingWs)

	cmd := workspace.CreateInviteCommand{
		WorkspaceID: existingWs.ID(),
		ExpiresAt:   nil, // будет использовано value by default (7 дней)
		MaxUses:     nil, // будет использовано value by default (0 - unlimited)
		CreatedBy:   uuid.NewUUID(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Value == nil {
		t.Fatal("expected invite to be created")
	}

	if result.Value.WorkspaceID() != cmd.WorkspaceID {
		t.Errorf("expected workspaceID %s, got %s", cmd.WorkspaceID, result.Value.WorkspaceID())
	}

	if result.Value.Token() == "" {
		t.Error("expected invite token to be generated")
	}

	// check, that инвайт добавлен in workspace
	updatedWs, _ := repo.FindByID(context.Background(), existingWs.ID())
	if len(updatedWs.Invites()) != 1 {
		t.Errorf("expected 1 invite in workspace, got %d", len(updatedWs.Invites()))
	}
}

func TestCreateInviteUseCase_Execute_WithCustomExpiresAt(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewCreateInviteUseCase(repo)

	// Creating existing workspace
	existingWs, _ := domainworkspace.NewWorkspace("Test Workspace", "", "keycloak-group-id", uuid.NewUUID())
	_ = repo.Save(context.Background(), existingWs)

	customExpiresAt := time.Now().Add(24 * time.Hour)
	maxUses := 5

	cmd := workspace.CreateInviteCommand{
		WorkspaceID: existingWs.ID(),
		ExpiresAt:   &customExpiresAt,
		MaxUses:     &maxUses,
		CreatedBy:   uuid.NewUUID(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Value == nil {
		t.Fatal("expected invite to be created")
	}

	if result.Value.MaxUses() != maxUses {
		t.Errorf("expected maxUses %d, got %d", maxUses, result.Value.MaxUses())
	}
}

func TestCreateInviteUseCase_Execute_WorkspaceNotFound(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewCreateInviteUseCase(repo)

	cmd := workspace.CreateInviteCommand{
		WorkspaceID: uuid.NewUUID(),
		ExpiresAt:   nil,
		MaxUses:     nil,
		CreatedBy:   uuid.NewUUID(),
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

func TestCreateInviteUseCase_Validate_InvalidWorkspaceID(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewCreateInviteUseCase(repo)

	cmd := workspace.CreateInviteCommand{
		WorkspaceID: uuid.UUID(""),
		ExpiresAt:   nil,
		MaxUses:     nil,
		CreatedBy:   uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for invalid workspaceID")
	}
}

func TestCreateInviteUseCase_Validate_InvalidCreatedBy(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewCreateInviteUseCase(repo)

	cmd := workspace.CreateInviteCommand{
		WorkspaceID: uuid.NewUUID(),
		ExpiresAt:   nil,
		MaxUses:     nil,
		CreatedBy:   uuid.UUID(""),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for invalid createdBy")
	}
}

func TestCreateInviteUseCase_Validate_ExpiresAtInPast(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewCreateInviteUseCase(repo)

	pastTime := time.Now().Add(-24 * time.Hour)
	cmd := workspace.CreateInviteCommand{
		WorkspaceID: uuid.NewUUID(),
		ExpiresAt:   &pastTime,
		MaxUses:     nil,
		CreatedBy:   uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for expiresAt in the past")
	}
}

func TestCreateInviteUseCase_Validate_NegativeMaxUses(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewCreateInviteUseCase(repo)

	negativeMaxUses := -1
	cmd := workspace.CreateInviteCommand{
		WorkspaceID: uuid.NewUUID(),
		ExpiresAt:   nil,
		MaxUses:     &negativeMaxUses,
		CreatedBy:   uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for negative maxUses")
	}
}

func TestCreateInviteUseCase_Execute_SaveError(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()

	// Creating existing workspace
	existingWs, _ := domainworkspace.NewWorkspace("Test Workspace", "", "keycloak-group-id", uuid.NewUUID())
	_ = repo.Save(context.Background(), existingWs)

	// Setting error saving
	repo.saveError = errors.New("database error")

	useCase := workspace.NewCreateInviteUseCase(repo)

	cmd := workspace.CreateInviteCommand{
		WorkspaceID: existingWs.ID(),
		ExpiresAt:   nil,
		MaxUses:     nil,
		CreatedBy:   uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected error from save operation")
	}
}
