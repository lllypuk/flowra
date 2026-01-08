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

func TestRevokeInviteUseCase_Execute_Success(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewRevokeInviteUseCase(repo)

	// Creating workspace with invite
	ws, _ := domainworkspace.NewWorkspace("Test Workspace", "", "keycloak-group-id", uuid.NewUUID())
	expiresAt := time.Now().Add(24 * time.Hour)
	invite, _ := ws.CreateInvite(uuid.NewUUID(), expiresAt, 0)
	_ = repo.Save(context.Background(), ws)

	cmd := workspace.RevokeInviteCommand{
		InviteID:  invite.ID(),
		RevokedBy: uuid.NewUUID(),
	}

	// Act
	result, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Value == nil {
		t.Fatal("expected invite to be returned")
	}

	if !result.Value.IsRevoked() {
		t.Error("expected invite to be revoked")
	}

	// check, that инвайт отозван in workspace
	updatedWs, _ := repo.FindByID(context.Background(), ws.ID())
	updatedInvite, _ := updatedWs.FindInviteByToken(invite.Token())
	if !updatedInvite.IsRevoked() {
		t.Error("expected invite in workspace to be revoked")
	}
}

func TestRevokeInviteUseCase_Execute_InviteNotFound(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewRevokeInviteUseCase(repo)

	cmd := workspace.RevokeInviteCommand{
		InviteID:  uuid.NewUUID(),
		RevokedBy: uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected error for invite not found")
	}

	if !errors.Is(err, workspace.ErrInviteNotFound) {
		t.Errorf("expected ErrInviteNotFound, got: %v", err)
	}
}

func TestRevokeInviteUseCase_Execute_AlreadyRevoked(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewRevokeInviteUseCase(repo)

	// Creating workspace с уже отозванным инвайтом
	ws, _ := domainworkspace.NewWorkspace("Test Workspace", "", "keycloak-group-id", uuid.NewUUID())
	expiresAt := time.Now().Add(24 * time.Hour)
	invite, _ := ws.CreateInvite(uuid.NewUUID(), expiresAt, 0)
	_ = invite.Revoke()
	_ = repo.Save(context.Background(), ws)

	cmd := workspace.RevokeInviteCommand{
		InviteID:  invite.ID(),
		RevokedBy: uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected error for already revoked invite")
	}
}

func TestRevokeInviteUseCase_Validate_InvalidInviteID(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewRevokeInviteUseCase(repo)

	cmd := workspace.RevokeInviteCommand{
		InviteID:  uuid.UUID(""),
		RevokedBy: uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for invalid inviteID")
	}
}

func TestRevokeInviteUseCase_Validate_InvalidRevokedBy(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	useCase := workspace.NewRevokeInviteUseCase(repo)

	cmd := workspace.RevokeInviteCommand{
		InviteID:  uuid.NewUUID(),
		RevokedBy: uuid.UUID(""),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for invalid revokedBy")
	}
}

func TestRevokeInviteUseCase_Execute_SaveError(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()

	// Creating workspace with invite
	ws, _ := domainworkspace.NewWorkspace("Test Workspace", "", "keycloak-group-id", uuid.NewUUID())
	expiresAt := time.Now().Add(24 * time.Hour)
	invite, _ := ws.CreateInvite(uuid.NewUUID(), expiresAt, 0)
	_ = repo.Save(context.Background(), ws)

	// Setting error saving
	repo.saveError = errors.New("database error")

	useCase := workspace.NewRevokeInviteUseCase(repo)

	cmd := workspace.RevokeInviteCommand{
		InviteID:  invite.ID(),
		RevokedBy: uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected error from save operation")
	}
}
