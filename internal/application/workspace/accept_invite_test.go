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

func TestAcceptInviteUseCase_Execute_Success(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	keycloakClient := newMockKeycloakClient()
	useCase := workspace.NewAcceptInviteUseCase(repo, keycloakClient)

	// Creating workspace with invite
	ws, _ := domainworkspace.NewWorkspace("Test Workspace", "", "keycloak-group-id", uuid.NewUUID())
	expiresAt := time.Now().Add(24 * time.Hour)
	invite, _ := ws.CreateInvite(uuid.NewUUID(), expiresAt, 0)
	_ = repo.Save(context.Background(), ws)

	// Creating group in Keycloak
	keycloakClient.groups[ws.KeycloakGroupID()] = "Test Workspace"
	keycloakClient.groupUsers[ws.KeycloakGroupID()] = []string{}

	userID := uuid.NewUUID()
	cmd := workspace.AcceptInviteCommand{
		Token:  invite.Token(),
		UserID: userID,
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

	// check that user dobavlen in groups Keycloak
	groupUsers := keycloakClient.groupUsers[ws.KeycloakGroupID()]
	if len(groupUsers) != 1 || groupUsers[0] != userID.String() {
		t.Error("expected user to be added to Keycloak group")
	}

	// check that schetchik ispolzovaniy invayta uvelichilsya
	updatedWs, _ := repo.FindByID(context.Background(), ws.ID())
	updatedInvite, _ := updatedWs.FindInviteByToken(invite.Token())
	if updatedInvite.UsedCount() != 1 {
		t.Errorf("expected usedCount 1, got %d", updatedInvite.UsedCount())
	}
}

func TestAcceptInviteUseCase_Execute_InviteNotFound(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	keycloakClient := newMockKeycloakClient()
	useCase := workspace.NewAcceptInviteUseCase(repo, keycloakClient)

	cmd := workspace.AcceptInviteCommand{
		Token:  "invalid-token",
		UserID: uuid.NewUUID(),
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

func TestAcceptInviteUseCase_Execute_InviteExpired(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	keycloakClient := newMockKeycloakClient()
	useCase := workspace.NewAcceptInviteUseCase(repo, keycloakClient)

	// Creating workspace with invite kotoryy skoro istechet
	ws, _ := domainworkspace.NewWorkspace("Test Workspace", "", "keycloak-group-id", uuid.NewUUID())
	// Creating s korotkim srokom deystviya (1 millisekunda in buduschem)
	expiresAt := time.Now().Add(1 * time.Millisecond)
	invite, err := ws.CreateInvite(uuid.NewUUID(), expiresAt, 0)
	if err != nil {
		t.Fatalf("failed to create invite: %v", err)
	}

	// wait for invayt expired
	time.Sleep(5 * time.Millisecond)

	_ = repo.Save(context.Background(), ws)

	cmd := workspace.AcceptInviteCommand{
		Token:  invite.Token(),
		UserID: uuid.NewUUID(),
	}

	// Act
	_, err = useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected error for expired invite")
	}

	if !errors.Is(err, workspace.ErrInviteExpired) {
		t.Errorf("expected ErrInviteExpired, got: %v", err)
	}
}

func TestAcceptInviteUseCase_Execute_InviteRevoked(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	keycloakClient := newMockKeycloakClient()
	useCase := workspace.NewAcceptInviteUseCase(repo, keycloakClient)

	// Creating workspace s otozvannym invaytom
	ws, _ := domainworkspace.NewWorkspace("Test Workspace", "", "keycloak-group-id", uuid.NewUUID())
	expiresAt := time.Now().Add(24 * time.Hour)
	invite, _ := ws.CreateInvite(uuid.NewUUID(), expiresAt, 0)
	_ = invite.Revoke()
	_ = repo.Save(context.Background(), ws)

	cmd := workspace.AcceptInviteCommand{
		Token:  invite.Token(),
		UserID: uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected error for revoked invite")
	}

	if !errors.Is(err, workspace.ErrInviteRevoked) {
		t.Errorf("expected ErrInviteRevoked, got: %v", err)
	}
}

func TestAcceptInviteUseCase_Execute_InviteMaxUsesReached(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	keycloakClient := newMockKeycloakClient()
	useCase := workspace.NewAcceptInviteUseCase(repo, keycloakClient)

	// Creating workspace with invite s limitom ispolzovaniy
	ws, _ := domainworkspace.NewWorkspace("Test Workspace", "", "keycloak-group-id", uuid.NewUUID())
	expiresAt := time.Now().Add(24 * time.Hour)
	invite, _ := ws.CreateInvite(uuid.NewUUID(), expiresAt, 1) // maxUses = 1

	// Creating group in Keycloak
	keycloakClient.groups[ws.KeycloakGroupID()] = "Test Workspace"
	keycloakClient.groupUsers[ws.KeycloakGroupID()] = []string{}

	// ispolzuem invayt one raz
	_ = invite.Use()
	_ = repo.Save(context.Background(), ws)

	// pytaemsya user vtoroy raz
	cmd := workspace.AcceptInviteCommand{
		Token:  invite.Token(),
		UserID: uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected error for invite max uses reached")
	}

	if !errors.Is(err, workspace.ErrInviteExpired) {
		t.Errorf("expected ErrInviteExpired (covers max uses), got: %v", err)
	}
}

func TestAcceptInviteUseCase_Validate_MissingToken(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	keycloakClient := newMockKeycloakClient()
	useCase := workspace.NewAcceptInviteUseCase(repo, keycloakClient)

	cmd := workspace.AcceptInviteCommand{
		Token:  "",
		UserID: uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for missing token")
	}
}

func TestAcceptInviteUseCase_Validate_InvalidUserID(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	keycloakClient := newMockKeycloakClient()
	useCase := workspace.NewAcceptInviteUseCase(repo, keycloakClient)

	cmd := workspace.AcceptInviteCommand{
		Token:  "some-token",
		UserID: uuid.UUID(""),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected validation error for invalid userID")
	}
}

func TestAcceptInviteUseCase_Execute_KeycloakAddUserError(t *testing.T) {
	// Arrange
	repo := newMockWorkspaceRepository()
	keycloakClient := newMockKeycloakClient()
	keycloakClient.addUserError = errors.New("Keycloak error")
	useCase := workspace.NewAcceptInviteUseCase(repo, keycloakClient)

	// Creating workspace with invite
	ws, _ := domainworkspace.NewWorkspace("Test Workspace", "", "keycloak-group-id", uuid.NewUUID())
	expiresAt := time.Now().Add(24 * time.Hour)
	invite, _ := ws.CreateInvite(uuid.NewUUID(), expiresAt, 0)
	_ = repo.Save(context.Background(), ws)

	// Creating group in Keycloak
	keycloakClient.groups[ws.KeycloakGroupID()] = "Test Workspace"
	keycloakClient.groupUsers[ws.KeycloakGroupID()] = []string{}

	cmd := workspace.AcceptInviteCommand{
		Token:  invite.Token(),
		UserID: uuid.NewUUID(),
	}

	// Act
	_, err := useCase.Execute(context.Background(), cmd)

	// Assert
	if err == nil {
		t.Fatal("expected error from Keycloak add user operation")
	}

	if !errors.Is(err, workspace.ErrKeycloakUserAddFailed) {
		t.Errorf("expected ErrKeycloakUserAddFailed, got: %v", err)
	}
}
