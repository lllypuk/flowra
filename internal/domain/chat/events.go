package chat

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/event"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Event types
const (
	EventTypeChatCreated        = "chat.created"
	EventTypeParticipantAdded   = "chat.participant_added"
	EventTypeParticipantRemoved = "chat.participant_removed"
	EventTypeChatTypeChanged    = "chat.type_changed"
	EventTypeStatusChanged      = "chat.status_changed"
	EventTypeUserAssigned       = "chat.user_assigned"
	EventTypeAssigneeRemoved    = "chat.assignee_removed"
	EventTypePrioritySet        = "chat.priority_set"
	EventTypeDueDateSet         = "chat.due_date_set"
	EventTypeDueDateRemoved     = "chat.due_date_removed"
	EventTypeChatRenamed        = "chat.renamed"
	EventTypeSeveritySet        = "chat.severity_set"
	EventTypeChatDeleted        = "chat.deleted"
	EventTypeChatClosed         = "chat.closed"   // Task 007a
	EventTypeChatReopened       = "chat.reopened" // Task 007a
)

// Created event creating chat
type Created struct {
	event.BaseEvent `bson:",inline"`

	WorkspaceID uuid.UUID `json:"workspace_id" bson:"workspace_id"`
	Type        Type      `json:"type"         bson:"type"`
	IsPublic    bool      `json:"is_public"    bson:"is_public"`
	CreatedBy   uuid.UUID `json:"created_by"   bson:"created_by"`
	CreatedAt   time.Time `json:"created_at"   bson:"created_at"`
}

// NewChatCreated creates new event ChatCreated
func NewChatCreated(
	chatID, workspaceID uuid.UUID,
	chatType Type,
	isPublic bool,
	createdBy uuid.UUID,
	createdAt time.Time,
	metadata event.Metadata,
) *Created {
	return &Created{
		BaseEvent:   event.NewBaseEvent(EventTypeChatCreated, chatID.String(), "Chat", 1, metadata),
		WorkspaceID: workspaceID,
		Type:        chatType,
		IsPublic:    isPublic,
		CreatedBy:   createdBy,
		CreatedAt:   createdAt,
	}
}

// ParticipantAdded event adding participant
type ParticipantAdded struct {
	event.BaseEvent `bson:",inline"`

	UserID   uuid.UUID `json:"user_id"   bson:"user_id"`
	Role     Role      `json:"role"      bson:"role"`
	JoinedAt time.Time `json:"joined_at" bson:"joined_at"`
}

// NewParticipantAdded creates new event participantAdded
func NewParticipantAdded(
	chatID, userID uuid.UUID,
	role Role,
	joinedAt time.Time,
	version int,
	metadata event.Metadata,
) *ParticipantAdded {
	return &ParticipantAdded{
		BaseEvent: event.NewBaseEvent(EventTypeParticipantAdded, chatID.String(), "Chat", version, metadata),
		UserID:    userID,
		Role:      role,
		JoinedAt:  joinedAt,
	}
}

// ParticipantRemoved event removing participant
type ParticipantRemoved struct {
	event.BaseEvent `bson:",inline"`

	UserID uuid.UUID `json:"user_id" bson:"user_id"`
}

// NewParticipantRemoved creates new event participantRemoved
func NewParticipantRemoved(
	chatID, userID uuid.UUID,
	version int,
	metadata event.Metadata,
) *ParticipantRemoved {
	return &ParticipantRemoved{
		BaseEvent: event.NewBaseEvent(EventTypeParticipantRemoved, chatID.String(), "Chat", version, metadata),
		UserID:    userID,
	}
}

// TypeChanged event changing type chat
type TypeChanged struct {
	event.BaseEvent `bson:",inline"`

	OldType Type   `json:"old_type" bson:"old_type"`
	NewType Type   `json:"new_type" bson:"new_type"`
	Title   string `json:"title"    bson:"title"`
}

// NewChatTypeChanged creates new event ChatTypeChanged
func NewChatTypeChanged(
	chatID uuid.UUID,
	oldType, newType Type,
	title string,
	version int,
	metadata event.Metadata,
) *TypeChanged {
	return &TypeChanged{
		BaseEvent: event.NewBaseEvent(EventTypeChatTypeChanged, chatID.String(), "Chat", version, metadata),
		OldType:   oldType,
		NewType:   newType,
		Title:     title,
	}
}

// ====== Entity Management Events ======

// StatusChanged event changing status
type StatusChanged struct {
	event.BaseEvent `bson:",inline"`

	OldStatus string    `json:"old_status" bson:"old_status"`
	NewStatus string    `json:"new_status" bson:"new_status"`
	ChangedBy uuid.UUID `json:"changed_by" bson:"changed_by"`
}

// NewStatusChanged creates event StatusChanged
func NewStatusChanged(
	chatID uuid.UUID,
	oldStatus, newStatus string,
	changedBy uuid.UUID,
	version int,
	metadata event.Metadata,
) *StatusChanged {
	return &StatusChanged{
		BaseEvent: event.NewBaseEvent(
			EventTypeStatusChanged,
			chatID.String(),
			"Chat",
			version,
			metadata,
		),
		OldStatus: oldStatus,
		NewStatus: newStatus,
		ChangedBy: changedBy,
	}
}

// UserAssigned event assigning user
type UserAssigned struct {
	event.BaseEvent `bson:",inline"`

	AssigneeID uuid.UUID `json:"assignee_id" bson:"assignee_id"`
	AssignedBy uuid.UUID `json:"assigned_by" bson:"assigned_by"`
}

// NewUserAssigned creates event UserAssigned
func NewUserAssigned(
	chatID, assigneeID, assignedBy uuid.UUID,
	version int,
	metadata event.Metadata,
) *UserAssigned {
	return &UserAssigned{
		BaseEvent: event.NewBaseEvent(
			EventTypeUserAssigned,
			chatID.String(),
			"Chat",
			version,
			metadata,
		),
		AssigneeID: assigneeID,
		AssignedBy: assignedBy,
	}
}

// AssigneeRemoved event snyatiya assignee
type AssigneeRemoved struct {
	event.BaseEvent `bson:",inline"`

	PreviousAssigneeID uuid.UUID `json:"previous_assignee_id" bson:"previous_assignee_id"`
	RemovedBy          uuid.UUID `json:"removed_by"           bson:"removed_by"`
}

// NewAssigneeRemoved creates event AssigneeRemoved
func NewAssigneeRemoved(
	chatID, previousAssigneeID, removedBy uuid.UUID,
	version int,
	metadata event.Metadata,
) *AssigneeRemoved {
	return &AssigneeRemoved{
		BaseEvent: event.NewBaseEvent(
			EventTypeAssigneeRemoved,
			chatID.String(),
			"Chat",
			version,
			metadata,
		),
		PreviousAssigneeID: previousAssigneeID,
		RemovedBy:          removedBy,
	}
}

// PrioritySet event setting priority
type PrioritySet struct {
	event.BaseEvent `bson:",inline"`

	OldPriority string    `json:"old_priority" bson:"old_priority"`
	NewPriority string    `json:"new_priority" bson:"new_priority"`
	ChangedBy   uuid.UUID `json:"changed_by"   bson:"changed_by"`
}

// NewPrioritySet creates event PrioritySet
func NewPrioritySet(
	chatID uuid.UUID,
	oldPriority, newPriority string,
	changedBy uuid.UUID,
	version int,
	metadata event.Metadata,
) *PrioritySet {
	return &PrioritySet{
		BaseEvent: event.NewBaseEvent(
			EventTypePrioritySet,
			chatID.String(),
			"Chat",
			version,
			metadata,
		),
		OldPriority: oldPriority,
		NewPriority: newPriority,
		ChangedBy:   changedBy,
	}
}

// DueDateSet event setting deadline
type DueDateSet struct {
	event.BaseEvent `bson:",inline"`

	OldDueDate *time.Time `json:"old_due_date,omitempty" bson:"old_due_date,omitempty"`
	NewDueDate time.Time  `json:"new_due_date"           bson:"new_due_date"`
	ChangedBy  uuid.UUID  `json:"changed_by"             bson:"changed_by"`
}

// NewDueDateSet creates event DueDateSet
func NewDueDateSet(
	chatID uuid.UUID,
	oldDueDate *time.Time,
	newDueDate time.Time,
	changedBy uuid.UUID,
	version int,
	metadata event.Metadata,
) *DueDateSet {
	return &DueDateSet{
		BaseEvent: event.NewBaseEvent(
			EventTypeDueDateSet,
			chatID.String(),
			"Chat",
			version,
			metadata,
		),
		OldDueDate: oldDueDate,
		NewDueDate: newDueDate,
		ChangedBy:  changedBy,
	}
}

// DueDateRemoved event snyatiya deadline
type DueDateRemoved struct {
	event.BaseEvent `bson:",inline"`

	PreviousDueDate time.Time `json:"previous_due_date" bson:"previous_due_date"`
	RemovedBy       uuid.UUID `json:"removed_by"        bson:"removed_by"`
}

// NewDueDateRemoved creates event DueDateRemoved
func NewDueDateRemoved(
	chatID uuid.UUID,
	previousDueDate time.Time,
	removedBy uuid.UUID,
	version int,
	metadata event.Metadata,
) *DueDateRemoved {
	return &DueDateRemoved{
		BaseEvent: event.NewBaseEvent(
			EventTypeDueDateRemoved,
			chatID.String(),
			"Chat",
			version,
			metadata,
		),
		PreviousDueDate: previousDueDate,
		RemovedBy:       removedBy,
	}
}

// Renamed event pereimenovaniya chat
type Renamed struct {
	event.BaseEvent `bson:",inline"`

	OldTitle  string    `json:"old_title"  bson:"old_title"`
	NewTitle  string    `json:"new_title"  bson:"new_title"`
	RenamedBy uuid.UUID `json:"renamed_by" bson:"renamed_by"`
}

// NewChatRenamed creates event Renamed
func NewChatRenamed(
	chatID uuid.UUID,
	oldTitle, newTitle string,
	renamedBy uuid.UUID,
	version int,
	metadata event.Metadata,
) *Renamed {
	return &Renamed{
		BaseEvent: event.NewBaseEvent(
			EventTypeChatRenamed,
			chatID.String(),
			"Chat",
			version,
			metadata,
		),
		OldTitle:  oldTitle,
		NewTitle:  newTitle,
		RenamedBy: renamedBy,
	}
}

// SeveritySet event setting severity for Bug
type SeveritySet struct {
	event.BaseEvent `bson:",inline"`

	OldSeverity string    `json:"old_severity" bson:"old_severity"`
	NewSeverity string    `json:"new_severity" bson:"new_severity"`
	ChangedBy   uuid.UUID `json:"changed_by"   bson:"changed_by"`
}

// NewSeveritySet creates event SeveritySet
func NewSeveritySet(
	chatID uuid.UUID,
	oldSeverity, newSeverity string,
	changedBy uuid.UUID,
	version int,
	metadata event.Metadata,
) *SeveritySet {
	return &SeveritySet{
		BaseEvent: event.NewBaseEvent(
			EventTypeSeveritySet,
			chatID.String(),
			"Chat",
			version,
			metadata,
		),
		OldSeverity: oldSeverity,
		NewSeverity: newSeverity,
		ChangedBy:   changedBy,
	}
}

// Deleted event removing chat (soft delete)
type Deleted struct {
	event.BaseEvent `bson:",inline"`

	DeletedBy uuid.UUID `json:"deleted_by" bson:"deleted_by"`
	DeletedAt time.Time `json:"deleted_at" bson:"deleted_at"`
}

// NewChatDeleted creates event Deleted
func NewChatDeleted(
	chatID, deletedBy uuid.UUID,
	deletedAt time.Time,
	version int,
	metadata event.Metadata,
) *Deleted {
	return &Deleted{
		BaseEvent: event.NewBaseEvent(
			EventTypeChatDeleted,
			chatID.String(),
			"Chat",
			version,
			metadata,
		),
		DeletedBy: deletedBy,
		DeletedAt: deletedAt,
	}
}

// ====== Task 007a: Chat Lifecycle Events ======

// Closed event when chat is closed/archived
type Closed struct {
	event.BaseEvent `bson:",inline"`

	ClosedBy       uuid.UUID `json:"closed_by"       bson:"closed_by"`
	PreviousStatus string    `json:"previous_status" bson:"previous_status"`
	ClosedAt       time.Time `json:"closed_at"       bson:"closed_at"`
}

// NewChatClosed creates event Closed
func NewChatClosed(
	chatID, closedBy uuid.UUID,
	previousStatus string,
	closedAt time.Time,
	version int,
	metadata event.Metadata,
) *Closed {
	return &Closed{
		BaseEvent: event.NewBaseEvent(
			EventTypeChatClosed,
			chatID.String(),
			"Chat",
			version,
			metadata,
		),
		ClosedBy:       closedBy,
		PreviousStatus: previousStatus,
		ClosedAt:       closedAt,
	}
}

// Reopened event when chat is reopened
type Reopened struct {
	event.BaseEvent `bson:",inline"`

	ReopenedBy uuid.UUID `json:"reopened_by" bson:"reopened_by"`
	NewStatus  string    `json:"new_status"  bson:"new_status"`
	ReopenedAt time.Time `json:"reopened_at" bson:"reopened_at"`
}

// NewChatReopened creates event Reopened
func NewChatReopened(
	chatID, reopenedBy uuid.UUID,
	newStatus string,
	reopenedAt time.Time,
	version int,
	metadata event.Metadata,
) *Reopened {
	return &Reopened{
		BaseEvent: event.NewBaseEvent(
			EventTypeChatReopened,
			chatID.String(),
			"Chat",
			version,
			metadata,
		),
		ReopenedBy: reopenedBy,
		NewStatus:  newStatus,
		ReopenedAt: reopenedAt,
	}
}
