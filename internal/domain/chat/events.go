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
)

// Created событие создания чата
type Created struct {
	event.BaseEvent

	WorkspaceID uuid.UUID
	Type        Type
	IsPublic    bool
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
}

// NewChatCreated создает новое событие ChatCreated
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

// ParticipantAdded событие добавления участника
type ParticipantAdded struct {
	event.BaseEvent

	UserID   uuid.UUID
	Role     Role
	JoinedAt time.Time
}

// NewParticipantAdded создает новое событие ParticipantAdded
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

// ParticipantRemoved событие удаления участника
type ParticipantRemoved struct {
	event.BaseEvent

	UserID uuid.UUID
}

// NewParticipantRemoved создает новое событие ParticipantRemoved
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

// TypeChanged событие изменения типа чата
type TypeChanged struct {
	event.BaseEvent

	OldType Type
	NewType Type
	Title   string
}

// NewChatTypeChanged создает новое событие ChatTypeChanged
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

// StatusChanged событие изменения статуса
type StatusChanged struct {
	event.BaseEvent

	OldStatus string
	NewStatus string
	ChangedBy uuid.UUID
}

// NewStatusChanged создает событие StatusChanged
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

// UserAssigned событие назначения пользователя
type UserAssigned struct {
	event.BaseEvent

	AssigneeID uuid.UUID
	AssignedBy uuid.UUID
}

// NewUserAssigned создает событие UserAssigned
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

// AssigneeRemoved событие снятия assignee
type AssigneeRemoved struct {
	event.BaseEvent

	PreviousAssigneeID uuid.UUID
	RemovedBy          uuid.UUID
}

// NewAssigneeRemoved создает событие AssigneeRemoved
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

// PrioritySet событие установки приоритета
type PrioritySet struct {
	event.BaseEvent

	OldPriority string
	NewPriority string
	ChangedBy   uuid.UUID
}

// NewPrioritySet создает событие PrioritySet
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

// DueDateSet событие установки дедлайна
type DueDateSet struct {
	event.BaseEvent

	OldDueDate *time.Time
	NewDueDate time.Time
	ChangedBy  uuid.UUID
}

// NewDueDateSet создает событие DueDateSet
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

// DueDateRemoved событие снятия дедлайна
type DueDateRemoved struct {
	event.BaseEvent

	PreviousDueDate time.Time
	RemovedBy       uuid.UUID
}

// NewDueDateRemoved создает событие DueDateRemoved
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

// Renamed событие переименования чата
type Renamed struct {
	event.BaseEvent

	OldTitle  string
	NewTitle  string
	RenamedBy uuid.UUID
}

// NewChatRenamed создает событие Renamed
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

// SeveritySet событие установки severity для Bug
type SeveritySet struct {
	event.BaseEvent

	OldSeverity string
	NewSeverity string
	ChangedBy   uuid.UUID
}

// NewSeveritySet создает событие SeveritySet
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
