package chat

import (
	"errors"
	"slices"
	"time"

	"github.com/lllypuk/flowra/internal/domain/errs"
	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/task"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Type represents the chat type
type Type string

const (
	// TypeDiscussion is a regular discussion
	TypeDiscussion Type = "discussion"
	// TypeTask is a task chat
	TypeTask Type = "task"
	// TypeBug is a bug chat
	TypeBug Type = "bug"
	// TypeEpic is an epic chat
	TypeEpic Type = "epic"
)

// Chat represents the chat aggregate root with Event Sourcing
type Chat struct {
	id           uuid.UUID
	workspaceID  uuid.UUID
	chatType     Type
	isPublic     bool
	createdBy    uuid.UUID
	createdAt    time.Time
	participants []Participant

	// Fields for typed chats (Task/Bug/Epic)
	title      string
	status     string
	priority   string
	assigneeID *uuid.UUID
	dueDate    *time.Time
	severity   string // only for Bug

	// Soft delete
	deleted   bool
	deletedAt *time.Time
	deletedBy *uuid.UUID

	// Event sourcing
	version           int
	uncommittedEvents []event.DomainEvent
}

// NewChat creates a New chat
func NewChat(
	workspaceID uuid.UUID,
	chatType Type,
	isPublic bool,
	createdBy uuid.UUID,
) (*Chat, error) {
	if workspaceID.IsZero() {
		return nil, errs.ErrInvalidInput
	}
	if createdBy.IsZero() {
		return nil, errs.ErrInvalidInput
	}
	if !isValidChatType(chatType) {
		return nil, errs.ErrInvalidInput
	}

	chatID := uuid.NewUUID()
	now := time.Now()

	chat := &Chat{
		participants:      make([]Participant, 0),
		uncommittedEvents: make([]event.DomainEvent, 0),
		version:           0,
	}

	// Raise ChatCreated event (version 1)
	createdEvent := NewChatCreated(
		chatID,
		workspaceID,
		chatType,
		isPublic,
		createdBy,
		now,
		event.Metadata{},
	)
	chat.applyEvent(createdEvent)

	// Creator automatically becomes admin - raise ParticipantAdded event
	// After ChatCreated version = 1, next version = 2
	participantEvent := NewParticipantAdded(
		chatID,
		createdBy,
		RoleAdmin,
		now,
		chat.version+1,
		event.Metadata{},
	)
	chat.applyEvent(participantEvent)

	return chat, nil
}

// AddParticipant adds a participant to the chat
func (c *Chat) AddParticipant(userID uuid.UUID, role Role) error {
	if userID.IsZero() {
		return errs.ErrInvalidInput
	}
	if c.HasParticipant(userID) {
		return errs.ErrAlreadyExists
	}

	// Raise ParticipantAdded event with correct version
	evt := NewParticipantAdded(
		c.id,
		userID,
		role,
		time.Now(),
		c.version+1,
		event.Metadata{},
	)
	c.applyEvent(evt)
	return nil
}

func (c *Chat) addParticipantInternal(userID uuid.UUID, role Role) {
	participant := NewParticipant(userID, role)
	c.participants = append(c.participants, participant)
}

// RemoveParticipant removes a participant from the chat
func (c *Chat) RemoveParticipant(userID uuid.UUID) error {
	if userID.IsZero() {
		return errs.ErrInvalidInput
	}
	if !c.HasParticipant(userID) {
		return errs.ErrNotFound
	}
	if userID == c.createdBy {
		return errs.ErrInvalidInput // Creator cannot leave the chat
	}

	// Raise ParticipantRemoved event
	evt := NewParticipantRemoved(
		c.id,
		userID,
		c.version+1,
		event.Metadata{},
	)
	c.applyEvent(evt)

	return nil
}

// ConvertToTask converts Discussion to Task
func (c *Chat) ConvertToTask(title string, userID uuid.UUID) error {
	// Validation
	if c.chatType != TypeDiscussion {
		return errs.ErrInvalidState
	}
	if title == "" {
		return errs.ErrInvalidInput
	}
	if userID.IsZero() {
		return errs.ErrInvalidInput
	}

	// Create event
	evt := NewChatTypeChanged(
		c.id,
		c.chatType,
		TypeTask,
		title,
		c.version+1,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
			UserID:        userID.String(),
		},
	)

	// Apply and save the event
	c.applyEvent(evt)
	return nil
}

// ConvertToBug converts Discussion to Bug
func (c *Chat) ConvertToBug(title string, userID uuid.UUID) error {
	if c.chatType != TypeDiscussion {
		return errs.ErrInvalidState
	}
	if title == "" {
		return errs.ErrInvalidInput
	}
	if userID.IsZero() {
		return errs.ErrInvalidInput
	}

	evt := NewChatTypeChanged(
		c.id,
		c.chatType,
		TypeBug,
		title,
		c.version+1,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
			UserID:        userID.String(),
		},
	)

	c.applyEvent(evt)
	return nil
}

// ConvertToEpic converts Discussion to Epic
func (c *Chat) ConvertToEpic(title string, userID uuid.UUID) error {
	if c.chatType != TypeDiscussion {
		return errs.ErrInvalidState
	}
	if title == "" {
		return errs.ErrInvalidInput
	}
	if userID.IsZero() {
		return errs.ErrInvalidInput
	}

	evt := NewChatTypeChanged(
		c.id,
		c.chatType,
		TypeEpic,
		title,
		c.version+1,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
			UserID:        userID.String(),
		},
	)

	c.applyEvent(evt)
	return nil
}

// ====== Entity Management Methods ======

// ChangeStatus changes the status of a typed chat
func (c *Chat) ChangeStatus(newStatus string, userID uuid.UUID) error {
	// Validation: only for typed chats
	if c.chatType == TypeDiscussion {
		return errs.ErrInvalidState
	}

	// Status validation
	if err := c.validateStatus(newStatus); err != nil {
		return err
	}

	// If status has not changed
	if c.status == newStatus {
		return nil
	}

	oldStatus := c.status

	evt := NewStatusChanged(
		c.id,
		oldStatus,
		newStatus,
		userID,
		c.version+1,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
			UserID:        userID.String(),
		},
	)

	c.applyEvent(evt)
	return nil
}

// AssignUser assigns an assignee
func (c *Chat) AssignUser(assigneeID *uuid.UUID, userID uuid.UUID) error {
	if c.chatType == TypeDiscussion {
		return errs.ErrInvalidState
	}

	// Removing assignee
	if assigneeID == nil {
		if c.assigneeID == nil {
			return nil // Already no assignee
		}

		evt := NewAssigneeRemoved(
			c.id,
			*c.assigneeID,
			userID,
			c.version+1,
			event.Metadata{
				CorrelationID: uuid.NewUUID().String(),
				CausationID:   uuid.NewUUID().String(),
				UserID:        userID.String(),
			},
		)
		c.applyEvent(evt)
		return nil
	}

	// Check: do not assign the same user
	if c.assigneeID != nil && *c.assigneeID == *assigneeID {
		return nil
	}

	// Assign assignee
	evt := NewUserAssigned(
		c.id,
		*assigneeID,
		userID,
		c.version+1,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
			UserID:        userID.String(),
		},
	)
	c.applyEvent(evt)
	return nil
}

// SetPriority sets the priority
func (c *Chat) SetPriority(priority string, userID uuid.UUID) error {
	if c.chatType == TypeDiscussion {
		return errs.ErrInvalidState
	}

	if err := c.validatePriority(priority); err != nil {
		return err
	}

	if c.priority == priority {
		return nil
	}

	oldPriority := c.priority

	evt := NewPrioritySet(
		c.id,
		oldPriority,
		priority,
		userID,
		c.version+1,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
			UserID:        userID.String(),
		},
	)

	c.applyEvent(evt)
	return nil
}

// SetDueDate sets or removes the deadline
func (c *Chat) SetDueDate(dueDate *time.Time, userID uuid.UUID) error {
	if c.chatType == TypeDiscussion {
		return errs.ErrInvalidState
	}

	// Removing due date
	if dueDate == nil {
		if c.dueDate == nil {
			return nil
		}

		evt := NewDueDateRemoved(
			c.id,
			*c.dueDate,
			userID,
			c.version+1,
			event.Metadata{
				CorrelationID: uuid.NewUUID().String(),
				CausationID:   uuid.NewUUID().String(),
				UserID:        userID.String(),
			},
		)
		c.applyEvent(evt)
		return nil
	}

	// Check: do not set the same date
	if c.dueDate != nil && c.dueDate.Equal(*dueDate) {
		return nil
	}

	// Set due date
	evt := NewDueDateSet(
		c.id,
		c.dueDate,
		*dueDate,
		userID,
		c.version+1,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
			UserID:        userID.String(),
		},
	)
	c.applyEvent(evt)
	return nil
}

// Rename changes the chat title
func (c *Chat) Rename(newTitle string, userID uuid.UUID) error {
	if newTitle == "" {
		return errs.ErrInvalidInput
	}

	if c.title == newTitle {
		return nil
	}

	oldTitle := c.title

	evt := NewChatRenamed(
		c.id,
		oldTitle,
		newTitle,
		userID,
		c.version+1,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
			UserID:        userID.String(),
		},
	)

	c.applyEvent(evt)
	return nil
}

// Delete deletes the chat (soft delete)
func (c *Chat) Delete(deletedBy uuid.UUID) error {
	if c.deleted {
		return errors.New("chat already deleted")
	}

	now := time.Now()
	evt := NewChatDeleted(
		c.id,
		deletedBy,
		now,
		c.version+1,
		event.Metadata{},
	)
	c.applyEvent(evt)
	return nil
}

// SetSeverity sets severity for Bug
func (c *Chat) SetSeverity(severity string, setBy uuid.UUID) error {
	if c.chatType != TypeBug {
		return errs.ErrInvalidState
	}

	if err := c.validateSeverity(severity); err != nil {
		return err
	}

	if c.severity == severity {
		return nil
	}

	oldSeverity := c.severity

	evt := NewSeveritySet(
		c.id,
		oldSeverity,
		severity,
		setBy,
		c.version+1,
		event.Metadata{
			CorrelationID: uuid.NewUUID().String(),
			CausationID:   uuid.NewUUID().String(),
			UserID:        setBy.String(),
		},
	)

	c.applyEvent(evt)
	return nil
}

// HasParticipant checks if the user is a participant
func (c *Chat) HasParticipant(userID uuid.UUID) bool {
	for _, p := range c.participants {
		if p.UserID() == userID {
			return true
		}
	}
	return false
}

// IsParticipantAdmin checks if the participant is an admin
func (c *Chat) IsParticipantAdmin(userID uuid.UUID) bool {
	for _, p := range c.participants {
		if p.UserID() == userID && p.IsAdmin() {
			return true
		}
	}
	return false
}

// FindParticipant finds a participant by ID
func (c *Chat) FindParticipant(userID uuid.UUID) *Participant {
	for _, p := range c.participants {
		if p.UserID() == userID {
			pCopy := p
			return &pCopy
		}
	}
	return nil
}

// IsTyped checks if the chat is typed (not Discussion)
func (c *Chat) IsTyped() bool {
	return c.chatType != TypeDiscussion
}

// GetTaskEntityType returns the corresponding TaskEntity type
func (c *Chat) GetTaskEntityType() (task.EntityType, error) {
	switch c.chatType {
	case TypeTask:
		return task.TypeTask, nil
	case TypeBug:
		return task.TypeBug, nil
	case TypeEpic:
		return task.TypeEpic, nil
	case TypeDiscussion:
		return task.TypeDiscussion, nil
	default:
		return "", errs.ErrInvalidState
	}
}

// Event Sourcing methods

// Apply applies an event to restore state.
// This method is idempotent: it is safe to apply the same event multiple times
// (for example, during event replay).
func (c *Chat) Apply(e event.DomainEvent) error {
	switch evt := e.(type) {
	case *Created:
		c.applyCreated(evt)
	case *ParticipantAdded:
		c.applyParticipantAdded(evt)
	case *ParticipantRemoved:
		c.applyParticipantRemoved(evt)
	case *TypeChanged:
		c.applyTypeChanged(evt)
	case *StatusChanged:
		c.applyStatusChanged(evt)
	case *UserAssigned:
		c.applyUserAssigned(evt)
	case *AssigneeRemoved:
		c.applyAssigneeRemoved(evt)
	case *PrioritySet:
		c.applyPrioritySet(evt)
	case *DueDateSet:
		c.applyDueDateSet(evt)
	case *DueDateRemoved:
		c.applyDueDateRemoved(evt)
	case *Renamed:
		c.applyRenamed(evt)
	case *SeveritySet:
		c.applySeveritySet(evt)
	case *Deleted:
		c.applyDeleted(evt)
	default:
		// Ignore unknown events (forward compatibility)
	}
	return nil
}

func (c *Chat) applyCreated(evt *Created) {
	c.id = uuid.UUID(evt.AggregateID())
	c.workspaceID = evt.WorkspaceID
	c.chatType = evt.Type
	c.isPublic = evt.IsPublic
	c.createdBy = evt.CreatedBy
	c.createdAt = evt.CreatedAt
	c.version = evt.Version()
}

func (c *Chat) applyParticipantAdded(evt *ParticipantAdded) {
	c.addParticipantInternal(evt.UserID, evt.Role)
	c.version = evt.Version()
}

// applyParticipantRemoved removes a participant.
// Idempotent: if the participant does not exist, nothing happens.
func (c *Chat) applyParticipantRemoved(evt *ParticipantRemoved) {
	newParticipants := make([]Participant, 0, len(c.participants))
	for _, p := range c.participants {
		if p.UserID() != evt.UserID {
			newParticipants = append(newParticipants, p)
		}
	}
	c.participants = newParticipants
	c.version = evt.Version()
}

func (c *Chat) applyTypeChanged(evt *TypeChanged) {
	c.chatType = evt.NewType
	c.title = evt.Title
	c.status = c.getDefaultStatus()
	c.version = evt.Version()
}

func (c *Chat) applyStatusChanged(evt *StatusChanged) {
	c.status = evt.NewStatus
	c.version = evt.Version()
}

func (c *Chat) applyUserAssigned(evt *UserAssigned) {
	assigneeID := evt.AssigneeID
	c.assigneeID = &assigneeID
	c.version = evt.Version()
}

func (c *Chat) applyAssigneeRemoved(evt *AssigneeRemoved) {
	c.assigneeID = nil
	c.version = evt.Version()
}

func (c *Chat) applyPrioritySet(evt *PrioritySet) {
	c.priority = evt.NewPriority
	c.version = evt.Version()
}

func (c *Chat) applyDueDateSet(evt *DueDateSet) {
	dueDate := evt.NewDueDate
	c.dueDate = &dueDate
	c.version = evt.Version()
}

func (c *Chat) applyDueDateRemoved(evt *DueDateRemoved) {
	c.dueDate = nil
	c.version = evt.Version()
}

func (c *Chat) applyRenamed(evt *Renamed) {
	c.title = evt.NewTitle
	c.version = evt.Version()
}

func (c *Chat) applySeveritySet(evt *SeveritySet) {
	c.severity = evt.NewSeverity
	c.version = evt.Version()
}

func (c *Chat) applyDeleted(evt *Deleted) {
	c.deleted = true
	c.deletedAt = &evt.DeletedAt
	c.deletedBy = &evt.DeletedBy
	c.version = evt.Version()
}

// getDefaultStatus returns the default status for the chat type
func (c *Chat) getDefaultStatus() string {
	switch c.chatType {
	case TypeTask:
		return "To Do"
	case TypeBug:
		return "New"
	case TypeEpic:
		return "Planned"
	case TypeDiscussion:
		return ""
	default:
		return ""
	}
}

// applyEvent applies an event and adds it to uncommitted
func (c *Chat) applyEvent(evt event.DomainEvent) {
	_ = c.Apply(evt)
	c.uncommittedEvents = append(c.uncommittedEvents, evt)
}

// ApplyAndTrack applies an event and adds it to the list of uncommitted events.
// Used for creating New events in the UseCase layer.
func (c *Chat) ApplyAndTrack(evt event.DomainEvent) error {
	if err := c.Apply(evt); err != nil {
		return err
	}
	c.uncommittedEvents = append(c.uncommittedEvents, evt)
	return nil
}

// GetUncommittedEvents returns новые event
func (c *Chat) GetUncommittedEvents() []event.DomainEvent {
	return c.uncommittedEvents
}

// MarkEventsAsCommitted помечает event as зафиксированные
func (c *Chat) MarkEventsAsCommitted() {
	c.uncommittedEvents = make([]event.DomainEvent, 0)
}

// Getters

// ID returns ID chat
func (c *Chat) ID() uuid.UUID { return c.id }

// WorkspaceID returns ID workspace
func (c *Chat) WorkspaceID() uuid.UUID { return c.workspaceID }

// Type returns type chat
func (c *Chat) Type() Type { return c.chatType }

// IsPublic returns признак публичности
func (c *Chat) IsPublic() bool { return c.isPublic }

// CreatedBy returns creator ID
func (c *Chat) CreatedBy() uuid.UUID { return c.createdBy }

// CreatedAt returns creation time
func (c *Chat) CreatedAt() time.Time { return c.createdAt }

// Participants returns копию list participants
func (c *Chat) Participants() []Participant {
	participants := make([]Participant, len(c.participants))
	copy(participants, c.participants)
	return participants
}

// Version returns version aggregate for optimistic locking
func (c *Chat) Version() int { return c.version }

// Title returns название typed chat
func (c *Chat) Title() string { return c.title }

// Status returns status typed chat
func (c *Chat) Status() string { return c.status }

// Priority returns приоритет
func (c *Chat) Priority() string { return c.priority }

// AssigneeID returns ID наvalueенного user
func (c *Chat) AssigneeID() *uuid.UUID { return c.assigneeID }

// DueDate returns дедлайн
func (c *Chat) DueDate() *time.Time { return c.dueDate }

// severity returns severity for Bug
func (c *Chat) severity() string { return c.severity }

// IsDeleted returns признак removing
func (c *Chat) IsDeleted() bool { return c.deleted }

// DeletedAt returns time removing
func (c *Chat) DeletedAt() *time.Time { return c.deletedAt }

// DeletedBy returns ID удалившего user
func (c *Chat) DeletedBy() *uuid.UUID { return c.deletedBy }

// Validation helpers

// validateStatus validates status for текущего type chat
func (c *Chat) validateStatus(status string) error {
	var validStatuses []string

	switch c.chatType {
	case TypeTask:
		validStatuses = []string{"To Do", "In Progress", "Done"}
	case TypeBug:
		validStatuses = []string{"New", "Investigating", "Fixed", "Verified"}
	case TypeEpic:
		validStatuses = []string{"Planned", "In Progress", "Completed"}
	case TypeDiscussion:
		return errs.ErrInvalidState
	default:
		return errs.ErrInvalidState
	}

	if slices.Contains(validStatuses, status) {
		return nil
	}

	return errs.ErrInvalidInput
}

// validatePriority validates priority
func (c *Chat) validatePriority(priority string) error {
	validPriorities := []string{"Low", "Medium", "High", "Critical"}

	if slices.Contains(validPriorities, priority) {
		return nil
	}

	return errs.ErrInvalidInput
}

// validateSeverity validates severity for Bug
func (c *Chat) validateSeverity(severity string) error {
	validSeverities := []string{"Minor", "Major", "Critical", "Blocker"}

	if slices.Contains(validSeverities, severity) {
		return nil
	}

	return errs.ErrInvalidInput
}

func isValidChatType(t Type) bool {
	return t == TypeDiscussion || t == TypeTask || t == TypeBug || t == TypeEpic
}
