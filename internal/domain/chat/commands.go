package chat

import (
	"time"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// Command represents a command for Chat aggregate
type Command interface {
	CommandType() string
	AggregateID() uuid.UUID
}

// ====== Entity Creation Commands ======

// ConvertToTaskCommand converts a Discussion chat to Task
type ConvertToTaskCommand struct {
	ChatID uuid.UUID
	Title  string
	UserID uuid.UUID // who executed the command
}

// CommandType returns the command type
func (c ConvertToTaskCommand) CommandType() string {
	return "ConvertToTask"
}

// AggregateID returns the aggregate ID
func (c ConvertToTaskCommand) AggregateID() uuid.UUID {
	return c.ChatID
}

// ConvertToBugCommand converts a Discussion chat to Bug
type ConvertToBugCommand struct {
	ChatID uuid.UUID
	Title  string
	UserID uuid.UUID
}

// CommandType returns the command type
func (c ConvertToBugCommand) CommandType() string {
	return "ConvertToBug"
}

// AggregateID returns the aggregate ID
func (c ConvertToBugCommand) AggregateID() uuid.UUID {
	return c.ChatID
}

// ConvertToEpicCommand converts a Discussion chat to Epic
type ConvertToEpicCommand struct {
	ChatID uuid.UUID
	Title  string
	UserID uuid.UUID
}

// CommandType returns the command type
func (c ConvertToEpicCommand) CommandType() string {
	return "ConvertToEpic"
}

// AggregateID returns the aggregate ID
func (c ConvertToEpicCommand) AggregateID() uuid.UUID {
	return c.ChatID
}

// ====== Entity Management Commands ======

// ChangeStatusCommand changes the status of a typed chat
type ChangeStatusCommand struct {
	ChatID uuid.UUID
	Status string
	UserID uuid.UUID
}

// CommandType returns the command type
func (c ChangeStatusCommand) CommandType() string {
	return "ChangeStatus"
}

// AggregateID returns the aggregate ID
func (c ChangeStatusCommand) AggregateID() uuid.UUID {
	return c.ChatID
}

// AssignUserCommand assigns or removes an assignee
type AssignUserCommand struct {
	ChatID     uuid.UUID
	AssigneeID *uuid.UUID // nil = remove assignee
	UserID     uuid.UUID
}

// CommandType returns the command type
func (c AssignUserCommand) CommandType() string {
	return "AssignUser"
}

// AggregateID returns the aggregate ID
func (c AssignUserCommand) AggregateID() uuid.UUID {
	return c.ChatID
}

// SetPriorityCommand sets the priority
type SetPriorityCommand struct {
	ChatID   uuid.UUID
	Priority string
	UserID   uuid.UUID
}

// CommandType returns the command type
func (c SetPriorityCommand) CommandType() string {
	return "SetPriority"
}

// AggregateID returns the aggregate ID
func (c SetPriorityCommand) AggregateID() uuid.UUID {
	return c.ChatID
}

// SetDueDateCommand sets or removes a deadline
type SetDueDateCommand struct {
	ChatID  uuid.UUID
	DueDate *time.Time // nil = remove due date
	UserID  uuid.UUID
}

// CommandType returns the command type
func (c SetDueDateCommand) CommandType() string {
	return "SetDueDate"
}

// AggregateID returns the aggregate ID
func (c SetDueDateCommand) AggregateID() uuid.UUID {
	return c.ChatID
}

// RenameChatCommand changes the chat name
type RenameChatCommand struct {
	ChatID   uuid.UUID
	NewTitle string
	UserID   uuid.UUID
}

// CommandType returns the command type
func (c RenameChatCommand) CommandType() string {
	return "RenameChat"
}

// AggregateID returns the aggregate ID
func (c RenameChatCommand) AggregateID() uuid.UUID {
	return c.ChatID
}

// SetSeverityCommand sets severity for a Bug
type SetSeverityCommand struct {
	ChatID   uuid.UUID
	Severity string
	UserID   uuid.UUID
}

// CommandType returns the command type
func (c SetSeverityCommand) CommandType() string {
	return "SetSeverity"
}

// AggregateID returns the aggregate ID
func (c SetSeverityCommand) AggregateID() uuid.UUID {
	return c.ChatID
}
