package tag

import (
	"time"

	"github.com/google/uuid"
)

// Command represents a command that should be executed
type Command interface {
	CommandType() string
}

// CreateTaskCommand - command creating Task
type CreateTaskCommand struct {
	ChatID uuid.UUID
	Title  string
}

// CommandType returns type commands
func (c CreateTaskCommand) CommandType() string {
	return "CreateTask"
}

// CreateBugCommand - command creating Bug
type CreateBugCommand struct {
	ChatID uuid.UUID
	Title  string
}

// CommandType returns type commands
func (c CreateBugCommand) CommandType() string {
	return "CreateBug"
}

// CreateEpicCommand - command creating Epic
type CreateEpicCommand struct {
	ChatID uuid.UUID
	Title  string
}

// CommandType returns type commands
func (c CreateEpicCommand) CommandType() string {
	return "CreateEpic"
}

// ====== Task 04: Entity Management Commands ======

// ChangeStatusCommand - command changing status
type ChangeStatusCommand struct {
	ChatID uuid.UUID
	Status string
}

// CommandType returns type commands
func (c ChangeStatusCommand) CommandType() string {
	return "ChangeStatus"
}

// AssignUserCommand - command assigning ispolnitelya
type AssignUserCommand struct {
	ChatID   uuid.UUID
	Username string     // @alex
	UserID   *uuid.UUID // rezolvlennyy ID (mozhet byt nil at snyatii)
}

// CommandType returns type commands
func (c AssignUserCommand) CommandType() string {
	return "AssignUser"
}

// ChangePriorityCommand - command changing priority
type ChangePriorityCommand struct {
	ChatID   uuid.UUID
	Priority string
}

// CommandType returns type commands
func (c ChangePriorityCommand) CommandType() string {
	return "ChangePriority"
}

// SetDueDateCommand - command setting deadline
type SetDueDateCommand struct {
	ChatID  uuid.UUID
	DueDate *time.Time // nil value snyat due date
}

// CommandType returns type commands
func (c SetDueDateCommand) CommandType() string {
	return "SetDueDate"
}

// ChangeTitleCommand - command changing nazvaniya
type ChangeTitleCommand struct {
	ChatID uuid.UUID
	Title  string
}

// CommandType returns type commands
func (c ChangeTitleCommand) CommandType() string {
	return "ChangeTitle"
}

// SetSeverityCommand - command setting bug severity
type SetSeverityCommand struct {
	ChatID   uuid.UUID
	Severity string
}

// CommandType returns type commands
func (c SetSeverityCommand) CommandType() string {
	return "SetSeverity"
}

// ====== Task 007a: Participant Management and Chat Lifecycle Commands ======

// InviteUserCommand - command to add a participant to the chat
type InviteUserCommand struct {
	ChatID   uuid.UUID
	Username string     // @alex format
	UserID   *uuid.UUID // resolved ID (set by executor)
}

// CommandType returns command type
func (c InviteUserCommand) CommandType() string {
	return "InviteUser"
}

// RemoveUserCommand - command to remove a participant from the chat
type RemoveUserCommand struct {
	ChatID   uuid.UUID
	Username string
	UserID   *uuid.UUID // resolved ID (set by executor)
}

// CommandType returns command type
func (c RemoveUserCommand) CommandType() string {
	return "RemoveUser"
}

// CloseChatCommand - command to close/archive a chat
type CloseChatCommand struct {
	ChatID uuid.UUID
}

// CommandType returns command type
func (c CloseChatCommand) CommandType() string {
	return "CloseChat"
}

// ReopenChatCommand - command to reopen a closed chat
type ReopenChatCommand struct {
	ChatID uuid.UUID
}

// CommandType returns command type
func (c ReopenChatCommand) CommandType() string {
	return "ReopenChat"
}

// DeleteChatCommand - command to delete a chat (soft delete)
type DeleteChatCommand struct {
	ChatID uuid.UUID
}

// CommandType returns command type
func (c DeleteChatCommand) CommandType() string {
	return "DeleteChat"
}
