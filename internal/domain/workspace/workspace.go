package workspace

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Workspace представляет рабочее пространство (команду/организацию)
type Workspace struct {
	id              uuid.UUID
	name            string
	keycloakGroupID string
	createdBy       uuid.UUID
	createdAt       time.Time
	updatedAt       time.Time
	invites         []*Invite
}

// NewWorkspace создает новое рабочее пространство
func NewWorkspace(name, keycloakGroupID string, createdBy uuid.UUID) (*Workspace, error) {
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
		keycloakGroupID: keycloakGroupID,
		createdBy:       createdBy,
		createdAt:       time.Now(),
		updatedAt:       time.Now(),
		invites:         make([]*Invite, 0),
	}, nil
}

// UpdateName обновляет название рабочего пространства
func (w *Workspace) UpdateName(name string) error {
	if name == "" {
		return errs.ErrInvalidInput
	}
	w.name = name
	w.updatedAt = time.Now()
	return nil
}

// CreateInvite создает новое приглашение в рабочее пространство
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

// ID возвращает ID рабочего пространства
func (w *Workspace) ID() uuid.UUID { return w.id }

// Name возвращает название рабочего пространства
func (w *Workspace) Name() string { return w.name }

// KeycloakGroupID возвращает ID группы Keycloak
func (w *Workspace) KeycloakGroupID() string { return w.keycloakGroupID }

// CreatedBy возвращает ID создателя
func (w *Workspace) CreatedBy() uuid.UUID { return w.createdBy }

// CreatedAt возвращает время создания
func (w *Workspace) CreatedAt() time.Time { return w.createdAt }

// UpdatedAt возвращает время последнего обновления
func (w *Workspace) UpdatedAt() time.Time { return w.updatedAt }

// Invites возвращает список приглашений
func (w *Workspace) Invites() []*Invite { return w.invites }

// Invite представляет приглашение в рабочее пространство
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

// NewInvite создает новое приглашение
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

// Use использует приглашение (увеличивает счетчик использований)
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

// IsValid проверяет, валидно ли приглашение
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

// ID возвращает ID приглашения
func (i *Invite) ID() uuid.UUID { return i.id }

// WorkspaceID возвращает ID рабочего пространства
func (i *Invite) WorkspaceID() uuid.UUID { return i.workspaceID }

// Token возвращает токен приглашения
func (i *Invite) Token() string { return i.token }

// CreatedBy возвращает ID создателя
func (i *Invite) CreatedBy() uuid.UUID { return i.createdBy }

// CreatedAt возвращает время создания
func (i *Invite) CreatedAt() time.Time { return i.createdAt }

// ExpiresAt возвращает время истечения
func (i *Invite) ExpiresAt() time.Time { return i.expiresAt }

// MaxUses возвращает максимальное количество использований
func (i *Invite) MaxUses() int { return i.maxUses }

// UsedCount возвращает количество использований
func (i *Invite) UsedCount() int { return i.usedCount }

// IsRevoked возвращает true если приглашение отменено
func (i *Invite) IsRevoked() bool { return i.isRevoked }
