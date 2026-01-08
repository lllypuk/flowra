package tag

import (
	"time"

	"github.com/google/uuid"
)

// Command represents команду, которая должна быть выполнена
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

// AssignUserCommand - command наvalueения исполнителя
type AssignUserCommand struct {
	ChatID   uuid.UUID
	Username string     // @alex
	UserID   *uuid.UUID // резолвленный ID (может быть nil at снятии)
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
	DueDate *time.Time // nil оvalueает снять due date
}

// CommandType returns type commands
func (c SetDueDateCommand) CommandType() string {
	return "SetDueDate"
}

// ChangeTitleCommand - command changing названия
type ChangeTitleCommand struct {
	ChatID uuid.UUID
	Title  string
}

// CommandType returns type commands
func (c ChangeTitleCommand) CommandType() string {
	return "ChangeTitle"
}

// SetSeverityCommand - command setting серьезности бага
type SetSeverityCommand struct {
	ChatID   uuid.UUID
	severity string
}

// CommandType returns type commands
func (c SetSeverityCommand) CommandType() string {
	return "SetSeverity"
}
