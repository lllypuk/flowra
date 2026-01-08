package chat

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/chat"
	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// CreateChatCommand contains data for creating a new chat
type CreateChatCommand struct {
	WorkspaceID uuid.UUID
	Title       string    // for typed chats (Task/Bug/Epic)
	Type        chat.Type // Discussion, Task, Bug, Epic
	IsPublic    bool      // public or private
	CreatedBy   uuid.UUID
}

// CommandName returns the command name
func (c CreateChatCommand) CommandName() string { return "CreateChat" }

// AddParticipantCommand contains data for adding a participant
type AddParticipantCommand struct {
	ChatID  uuid.UUID
	UserID  uuid.UUID
	Role    chat.Role // Admin, Member
	AddedBy uuid.UUID
}

// CommandName returns the command name
func (c AddParticipantCommand) CommandName() string { return "AddParticipant" }

// RemoveParticipantCommand contains data for removing a participant
type RemoveParticipantCommand struct {
	ChatID    uuid.UUID
	UserID    uuid.UUID
	RemovedBy uuid.UUID
}

// CommandName returns the command name
func (c RemoveParticipantCommand) CommandName() string { return "RemoveParticipant" }

// ConvertToTaskCommand contains data for converting a chat to Task
type ConvertToTaskCommand struct {
	ChatID      uuid.UUID
	Title       string // new title
	ConvertedBy uuid.UUID
}

// CommandName returns the command name
func (c ConvertToTaskCommand) CommandName() string { return "ConvertToTask" }

// ConvertToBugCommand contains data for converting a chat to Bug
type ConvertToBugCommand struct {
	ChatID      uuid.UUID
	Title       string // new title
	ConvertedBy uuid.UUID
}

// CommandName returns the command name
func (c ConvertToBugCommand) CommandName() string { return "ConvertToBug" }

// ConvertToEpicCommand contains data for converting a chat to Epic
type ConvertToEpicCommand struct {
	ChatID      uuid.UUID
	Title       string // new title
	ConvertedBy uuid.UUID
}

// CommandName returns the command name
func (c ConvertToEpicCommand) CommandName() string { return "ConvertToEpic" }

// ChangeStatusCommand contains data for changing status
type ChangeStatusCommand struct {
	ChatID    uuid.UUID
	Status    string // depends on chat type
	ChangedBy uuid.UUID
}

// CommandName returns the command name
func (c ChangeStatusCommand) CommandName() string { return "ChangeStatus" }

// AssignUserCommand contains data for assigning a user
type AssignUserCommand struct {
	ChatID     uuid.UUID
	AssigneeID *uuid.UUID // nil = remove assignee
	AssignedBy uuid.UUID
}

// CommandName returns the command name
func (c AssignUserCommand) CommandName() string { return "AssignUser" }

// SetPriorityCommand contains data for setting priority
type SetPriorityCommand struct {
	ChatID   uuid.UUID
	Priority string // Low, Medium, High, Critical
	SetBy    uuid.UUID
}

// CommandName returns the command name
func (c SetPriorityCommand) CommandName() string { return "SetPriority" }

// SetDueDateCommand contains data for setting a deadline
type SetDueDateCommand struct {
	ChatID  uuid.UUID
	DueDate *time.Time // nil = remove deadline
	SetBy   uuid.UUID
}

// CommandName returns the command name
func (c SetDueDateCommand) CommandName() string { return "SetDueDate" }

// RenameChatCommand contains data for renaming a chat
type RenameChatCommand struct {
	ChatID    uuid.UUID
	NewTitle  string
	RenamedBy uuid.UUID
}

// CommandName returns the command name
func (c RenameChatCommand) CommandName() string { return "RenameChat" }

// SetSeverityCommand contains data for setting severity (only for Bug)
type SetSeverityCommand struct {
	ChatID   uuid.UUID
	Severity string // Minor, Major, Critical, Blocker
	SetBy    uuid.UUID
}

// CommandName returns the command name
func (c SetSeverityCommand) CommandName() string { return "SetSeverity" }
