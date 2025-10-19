package workspace

import "errors"

var (
	// ErrWorkspaceNotFound возникает когда workspace не найден
	ErrWorkspaceNotFound = errors.New("workspace not found")

	// ErrInviteNotFound возникает когда инвайт не найден
	ErrInviteNotFound = errors.New("invite not found")

	// ErrInviteExpired возникает когда инвайт истек
	ErrInviteExpired = errors.New("invite has expired")

	// ErrInviteRevoked возникает когда инвайт отозван
	ErrInviteRevoked = errors.New("invite has been revoked")

	// ErrInviteMaxUsesReached возникает когда достигнут лимит использований инвайта
	ErrInviteMaxUsesReached = errors.New("invite has reached maximum uses")

	// ErrInvalidInviteToken возникает при невалидном токене инвайта
	ErrInvalidInviteToken = errors.New("invalid invite token")

	// ErrKeycloakGroupCreationFailed возникает при ошибке создания группы в Keycloak
	ErrKeycloakGroupCreationFailed = errors.New("failed to create Keycloak group")

	// ErrKeycloakGroupDeletionFailed возникает при ошибке удаления группы в Keycloak
	ErrKeycloakGroupDeletionFailed = errors.New("failed to delete Keycloak group")

	// ErrKeycloakUserAddFailed возникает при ошибке добавления пользователя в группу Keycloak
	ErrKeycloakUserAddFailed = errors.New("failed to add user to Keycloak group")
)
