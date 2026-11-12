package workspace

import "errors"

var (
	// ErrWorkspaceNotFound is returned when workspace is not found
	ErrWorkspaceNotFound = errors.New("workspace not found")

	// ErrInviteNotFound is returned when invite is not found
	ErrInviteNotFound = errors.New("invite not found")

	// ErrInviteExpired is returned when invite has expired
	ErrInviteExpired = errors.New("invite has expired")

	// ErrInviteRevoked is returned when invite has been revoked
	ErrInviteRevoked = errors.New("invite has been revoked")

	// ErrInviteMaxUsesReached is returned when invite usage limit is reached
	ErrInviteMaxUsesReached = errors.New("invite has reached maximum uses")

	// ErrInvalidInviteToken is returned when invite token is invalid
	ErrInvalidInviteToken = errors.New("invalid invite token")
)
