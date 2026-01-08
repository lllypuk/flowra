package workspace

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Workspace represents workspace space (komandu/organizatsiyu)
type Workspace struct {
	id              uuid.UUID
	name            string
	description     string
	keycloakGroupID string
	createdBy       uuid.UUID
	createdAt       time.Time
	updatedAt       time.Time
	invites         []*Invite
}

// NewWorkspace creates new workspace space
func NewWorkspace(name, description, keycloakGroupID string, createdBy uuid.UUID) (*Workspace, error) {
	if name == "" {
		return nil, errs.ErrInvalidInput
	}
	if keycloakGroupID == "" {
		return nil, errs.ErrInvalidInput
	}
	if createdBy.IsZero() {
		return nil, errs.ErrInvalidInput
	}

	return &Workspace{
		id:              uuid.NewUUID(),
		name:            name,
		description:     description,
		keycloakGroupID: keycloakGroupID,
		createdBy:       createdBy,
		createdAt:       time.Now(),
		updatedAt:       time.Now(),
		invites:         make([]*Invite, 0),
	}, nil
}

// Reconstruct reconstructs workspace space from save.
// Used by repositories for hydration obekta without validation business rules.
// all parameters dolzhny byt valid values from save.
func Reconstruct(
	id uuid.UUID,
	name, description, keycloakGroupID string,
	createdBy uuid.UUID,
	createdAt, updatedAt time.Time,
	invites []*Invite,
) *Workspace {
	if invites == nil {
		invites = make([]*Invite, 0)
	}
	return &Workspace{
		id:              id,
		name:            name,
		description:     description,
		keycloakGroupID: keycloakGroupID,
		createdBy:       createdBy,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
		invites:         invites,
	}
}

// UpdateName obnovlyaet nazvanie workspace prostranstva
func (w *Workspace) UpdateName(name string) error {
	if name == "" {
		return errs.ErrInvalidInput
	}
	w.name = name
	w.updatedAt = time.Now()
	return nil
}

// CreateInvite creates new invitation in workspace space
func (w *Workspace) CreateInvite(createdBy uuid.UUID, expiresAt time.Time, maxUses int) (*Invite, error) {
	if createdBy.IsZero() {
		return nil, errs.ErrInvalidInput
	}
	if expiresAt.Before(time.Now()) {
		return nil, errs.ErrInvalidInput
	}
	if maxUses < 0 {
		return nil, errs.ErrInvalidInput
	}

	invite, err := NewInvite(w.id, createdBy, expiresAt, maxUses)
	if err != nil {
		return nil, err
	}

	w.invites = append(w.invites, invite)
	w.updatedAt = time.Now()
	return invite, nil
}

// FindInviteByToken ischet priglashenie po tokenu
func (w *Workspace) FindInviteByToken(token string) (*Invite, error) {
	for _, invite := range w.invites {
		if invite.token == token {
			return invite, nil
		}
	}
	return nil, errs.ErrNotFound
}

// ID returns ID workspace prostranstva
func (w *Workspace) ID() uuid.UUID { return w.id }

// Name returns nazvanie workspace prostranstva
func (w *Workspace) Name() string { return w.name }

// Description returns opisanie workspace prostranstva
func (w *Workspace) Description() string { return w.description }

// KeycloakGroupID returns ID groups Keycloak
func (w *Workspace) KeycloakGroupID() string { return w.keycloakGroupID }

// CreatedBy returns creator ID
func (w *Workspace) CreatedBy() uuid.UUID { return w.createdBy }

// CreatedAt returns creation time
func (w *Workspace) CreatedAt() time.Time { return w.createdAt }

// UpdatedAt returns time poslednego updating
func (w *Workspace) UpdatedAt() time.Time { return w.updatedAt }

// Invites returns list priglasheniy
func (w *Workspace) Invites() []*Invite { return w.invites }

// Invite represents priglashenie in workspace space
type Invite struct {
	id          uuid.UUID
	workspaceID uuid.UUID
	token       string
	createdBy   uuid.UUID
	createdAt   time.Time
	expiresAt   time.Time
	maxUses     int
	usedCount   int
	isRevoked   bool
}

// NewInvite creates new invitation
func NewInvite(workspaceID, createdBy uuid.UUID, expiresAt time.Time, maxUses int) (*Invite, error) {
	if workspaceID.IsZero() {
		return nil, errs.ErrInvalidInput
	}
	if createdBy.IsZero() {
		return nil, errs.ErrInvalidInput
	}
	if expiresAt.Before(time.Now()) {
		return nil, errs.ErrInvalidInput
	}
	if maxUses < 0 {
		return nil, errs.ErrInvalidInput
	}

	return &Invite{
		id:          uuid.NewUUID(),
		workspaceID: workspaceID,
		token:       uuid.NewUUID().String(),
		createdBy:   createdBy,
		createdAt:   time.Now(),
		expiresAt:   expiresAt,
		maxUses:     maxUses,
		usedCount:   0,
		isRevoked:   false,
	}, nil
}

// Use uses priglashenie (uvelichivaet schetchik ispolzovaniy)
func (i *Invite) Use() error {
	if i.isRevoked {
		return errs.ErrInvalidState
	}
	if time.Now().After(i.expiresAt) {
		return errs.ErrInvalidState
	}
	if i.maxUses > 0 && i.usedCount >= i.maxUses {
		return errs.ErrInvalidState
	}

	i.usedCount++
	return nil
}

// Revoke otmenyaet priglashenie
func (i *Invite) Revoke() error {
	if i.isRevoked {
		return errs.ErrInvalidState
	}
	i.isRevoked = true
	return nil
}

// IsValid checks, valid li priglashenie
func (i *Invite) IsValid() bool {
	if i.isRevoked {
		return false
	}
	if time.Now().After(i.expiresAt) {
		return false
	}
	if i.maxUses > 0 && i.usedCount >= i.maxUses {
		return false
	}
	return true
}

// ID returns ID priglasheniya
func (i *Invite) ID() uuid.UUID { return i.id }

// WorkspaceID returns ID workspace prostranstva
func (i *Invite) WorkspaceID() uuid.UUID { return i.workspaceID }

// Token returns token priglasheniya
func (i *Invite) Token() string { return i.token }

// CreatedBy returns creator ID
func (i *Invite) CreatedBy() uuid.UUID { return i.createdBy }

// CreatedAt returns creation time
func (i *Invite) CreatedAt() time.Time { return i.createdAt }

// ExpiresAt returns time istecheniya
func (i *Invite) ExpiresAt() time.Time { return i.expiresAt }

// MaxUses returns maximum count ispolzovaniy
func (i *Invite) MaxUses() int { return i.maxUses }

// UsedCount returns count ispolzovaniy
func (i *Invite) UsedCount() int { return i.usedCount }

// IsRevoked returns true if priglashenie otmeneno
func (i *Invite) IsRevoked() bool { return i.isRevoked }

// ReconstructInvite reconstructs priglashenie from save.
// Used by repositories for hydration obekta without validation business rules.
// all parameters dolzhny byt valid values from save.
func ReconstructInvite(
	id uuid.UUID,
	workspaceID uuid.UUID,
	token string,
	createdBy uuid.UUID,
	createdAt, expiresAt time.Time,
	maxUses, usedCount int,
	isRevoked bool,
) *Invite {
	return &Invite{
		id:          id,
		workspaceID: workspaceID,
		token:       token,
		createdBy:   createdBy,
		createdAt:   createdAt,
		expiresAt:   expiresAt,
		maxUses:     maxUses,
		usedCount:   usedCount,
		isRevoked:   isRevoked,
	}
}
