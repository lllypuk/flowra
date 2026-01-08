package workspace

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Workspace represents workspaceее пространство (команду/организацию)
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

// NewWorkspace creates новое workspaceее пространство
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

// Reconstruct восстанавливает workspaceее пространство from storage.
// Used by repositories for hydration объекта without validation business rules.
// all parameters должны быть valid values from storage.
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

// UpdateName обновляет название workspace пространства
func (w *Workspace) UpdateName(name string) error {
	if name == "" {
		return errs.ErrInvalidInput
	}
	w.name = name
	w.updatedAt = time.Now()
	return nil
}

// CreateInvite creates новое приглашение in workspaceее пространство
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

// FindInviteByToken ищет приглашение по токену
func (w *Workspace) FindInviteByToken(token string) (*Invite, error) {
	for _, invite := range w.invites {
		if invite.token == token {
			return invite, nil
		}
	}
	return nil, errs.ErrNotFound
}

// ID returns ID workspace пространства
func (w *Workspace) ID() uuid.UUID { return w.id }

// Name returns название workspace пространства
func (w *Workspace) Name() string { return w.name }

// Description returns описание workspace пространства
func (w *Workspace) Description() string { return w.description }

// KeycloakGroupID returns ID groupsы Keycloak
func (w *Workspace) KeycloakGroupID() string { return w.keycloakGroupID }

// CreatedBy returns creator ID
func (w *Workspace) CreatedBy() uuid.UUID { return w.createdBy }

// CreatedAt returns creation time
func (w *Workspace) CreatedAt() time.Time { return w.createdAt }

// UpdatedAt returns time последнего updating
func (w *Workspace) UpdatedAt() time.Time { return w.updatedAt }

// Invites returns list приглашений
func (w *Workspace) Invites() []*Invite { return w.invites }

// Invite represents приглашение in workspaceее пространство
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

// NewInvite creates новое приглашение
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

// Use uses приглашение (увеличивает счетчик использований)
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

// Revoke отменяет приглашение
func (i *Invite) Revoke() error {
	if i.isRevoked {
		return errs.ErrInvalidState
	}
	i.isRevoked = true
	return nil
}

// IsValid checks, validно ли приглашение
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

// ID returns ID приглашения
func (i *Invite) ID() uuid.UUID { return i.id }

// WorkspaceID returns ID workspace пространства
func (i *Invite) WorkspaceID() uuid.UUID { return i.workspaceID }

// Token returns токен приглашения
func (i *Invite) Token() string { return i.token }

// CreatedBy returns creator ID
func (i *Invite) CreatedBy() uuid.UUID { return i.createdBy }

// CreatedAt returns creation time
func (i *Invite) CreatedAt() time.Time { return i.createdAt }

// ExpiresAt returns time истечения
func (i *Invite) ExpiresAt() time.Time { return i.expiresAt }

// MaxUses returns максимальное count использований
func (i *Invite) MaxUses() int { return i.maxUses }

// UsedCount returns count использований
func (i *Invite) UsedCount() int { return i.usedCount }

// IsRevoked returns true if приглашение отменено
func (i *Invite) IsRevoked() bool { return i.isRevoked }

// ReconstructInvite восстанавливает приглашение from storage.
// Used by repositories for hydration объекта without validation business rules.
// all parameters должны быть valid values from storage.
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
