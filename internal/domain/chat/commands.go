package chat

import (
	"time"

	"github.com/flowra/flowra/internal/domain/uuid"
)

// Command представляет команду для Chat aggregate
type Command interface {
	CommandType() string
	AggregateID() uuid.UUID
}

// ====== Entity Creation Commands ======

// ConvertToTaskCommand превращает Discussion чат в Task
type ConvertToTaskCommand struct {
	ChatID uuid.UUID
	Title  string
	UserID uuid.UUID // кто выполнил команду
}

// CommandType возвращает тип команды
func (c ConvertToTaskCommand) CommandType() string {
	return "ConvertToTask"
}

// AggregateID возвращает ID aggregate
func (c ConvertToTaskCommand) AggregateID() uuid.UUID {
	return c.ChatID
}

// ConvertToBugCommand превращает Discussion чат в Bug
type ConvertToBugCommand struct {
	ChatID uuid.UUID
	Title  string
	UserID uuid.UUID
}

// CommandType возвращает тип команды
func (c ConvertToBugCommand) CommandType() string {
	return "ConvertToBug"
}

// AggregateID возвращает ID aggregate
func (c ConvertToBugCommand) AggregateID() uuid.UUID {
	return c.ChatID
}

// ConvertToEpicCommand превращает Discussion чат в Epic
type ConvertToEpicCommand struct {
	ChatID uuid.UUID
	Title  string
	UserID uuid.UUID
}

// CommandType возвращает тип команды
func (c ConvertToEpicCommand) CommandType() string {
	return "ConvertToEpic"
}

// AggregateID возвращает ID aggregate
func (c ConvertToEpicCommand) AggregateID() uuid.UUID {
	return c.ChatID
}

// ====== Entity Management Commands ======

// ChangeStatusCommand изменяет статус typed чата
type ChangeStatusCommand struct {
	ChatID uuid.UUID
	Status string
	UserID uuid.UUID
}

// CommandType возвращает тип команды
func (c ChangeStatusCommand) CommandType() string {
	return "ChangeStatus"
}

// AggregateID возвращает ID aggregate
func (c ChangeStatusCommand) AggregateID() uuid.UUID {
	return c.ChatID
}

// AssignUserCommand назначает или снимает исполнителя
type AssignUserCommand struct {
	ChatID     uuid.UUID
	AssigneeID *uuid.UUID // nil = снять assignee
	UserID     uuid.UUID
}

// CommandType возвращает тип команды
func (c AssignUserCommand) CommandType() string {
	return "AssignUser"
}

// AggregateID возвращает ID aggregate
func (c AssignUserCommand) AggregateID() uuid.UUID {
	return c.ChatID
}

// SetPriorityCommand устанавливает приоритет
type SetPriorityCommand struct {
	ChatID   uuid.UUID
	Priority string
	UserID   uuid.UUID
}

// CommandType возвращает тип команды
func (c SetPriorityCommand) CommandType() string {
	return "SetPriority"
}

// AggregateID возвращает ID aggregate
func (c SetPriorityCommand) AggregateID() uuid.UUID {
	return c.ChatID
}

// SetDueDateCommand устанавливает или снимает дедлайн
type SetDueDateCommand struct {
	ChatID  uuid.UUID
	DueDate *time.Time // nil = снять due date
	UserID  uuid.UUID
}

// CommandType возвращает тип команды
func (c SetDueDateCommand) CommandType() string {
	return "SetDueDate"
}

// AggregateID возвращает ID aggregate
func (c SetDueDateCommand) AggregateID() uuid.UUID {
	return c.ChatID
}

// RenameChatCommand изменяет название чата
type RenameChatCommand struct {
	ChatID   uuid.UUID
	NewTitle string
	UserID   uuid.UUID
}

// CommandType возвращает тип команды
func (c RenameChatCommand) CommandType() string {
	return "RenameChat"
}

// AggregateID возвращает ID aggregate
func (c RenameChatCommand) AggregateID() uuid.UUID {
	return c.ChatID
}

// SetSeverityCommand устанавливает severity для Bug
type SetSeverityCommand struct {
	ChatID   uuid.UUID
	Severity string
	UserID   uuid.UUID
}

// CommandType возвращает тип команды
func (c SetSeverityCommand) CommandType() string {
	return "SetSeverity"
}

// AggregateID возвращает ID aggregate
func (c SetSeverityCommand) AggregateID() uuid.UUID {
	return c.ChatID
}
