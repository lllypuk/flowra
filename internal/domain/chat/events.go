package chat

import (
	"time"

	"github.com/lllypuk/teams-up/internal/domain/event"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
)

// Event types
const (
	EventTypeChatCreated        = "chat.created"
	EventTypeParticipantAdded   = "chat.participant_added"
	EventTypeParticipantRemoved = "chat.participant_removed"
	EventTypeChatTypeChanged    = "chat.type_changed"
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
	metadata event.Metadata,
) *ParticipantAdded {
	return &ParticipantAdded{
		BaseEvent: event.NewBaseEvent(EventTypeParticipantAdded, chatID.String(), "Chat", 1, metadata),
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
	metadata event.Metadata,
) *ParticipantRemoved {
	return &ParticipantRemoved{
		BaseEvent: event.NewBaseEvent(EventTypeParticipantRemoved, chatID.String(), "Chat", 1, metadata),
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
	metadata event.Metadata,
) *TypeChanged {
	return &TypeChanged{
		BaseEvent: event.NewBaseEvent(EventTypeChatTypeChanged, chatID.String(), "Chat", 1, metadata),
		OldType:   oldType,
		NewType:   newType,
		Title:     title,
	}
}
