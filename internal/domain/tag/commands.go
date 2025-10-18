package tag

import (
	"time"

	"github.com/google/uuid"
)

// Command представляет команду, которая должна быть выполнена
type Command interface {
	CommandType() string
}

// CreateTaskCommand - команда создания Task
type CreateTaskCommand struct {
	ChatID uuid.UUID
	Title  string
}

// CommandType возвращает тип команды
func (c CreateTaskCommand) CommandType() string {
	return "CreateTask"
}

// CreateBugCommand - команда создания Bug
type CreateBugCommand struct {
	ChatID uuid.UUID
	Title  string
}

// CommandType возвращает тип команды
func (c CreateBugCommand) CommandType() string {
	return "CreateBug"
}

// CreateEpicCommand - команда создания Epic
type CreateEpicCommand struct {
	ChatID uuid.UUID
	Title  string
}

// CommandType возвращает тип команды
func (c CreateEpicCommand) CommandType() string {
	return "CreateEpic"
}

// ====== Task 04: Entity Management Commands ======

// ChangeStatusCommand - команда изменения статуса
type ChangeStatusCommand struct {
	ChatID uuid.UUID
	Status string
}

// CommandType возвращает тип команды
func (c ChangeStatusCommand) CommandType() string {
	return "ChangeStatus"
}

// AssignUserCommand - команда назначения исполнителя
type AssignUserCommand struct {
	ChatID   uuid.UUID
	Username string     // @alex
	UserID   *uuid.UUID // резолвленный ID (может быть nil при снятии)
}

// CommandType возвращает тип команды
func (c AssignUserCommand) CommandType() string {
	return "AssignUser"
}

// ChangePriorityCommand - команда изменения приоритета
type ChangePriorityCommand struct {
	ChatID   uuid.UUID
	Priority string
}

// CommandType возвращает тип команды
func (c ChangePriorityCommand) CommandType() string {
	return "ChangePriority"
}

// SetDueDateCommand - команда установки дедлайна
type SetDueDateCommand struct {
	ChatID  uuid.UUID
	DueDate *time.Time // nil означает снять due date
}

// CommandType возвращает тип команды
func (c SetDueDateCommand) CommandType() string {
	return "SetDueDate"
}

// ChangeTitleCommand - команда изменения названия
type ChangeTitleCommand struct {
	ChatID uuid.UUID
	Title  string
}

// CommandType возвращает тип команды
func (c ChangeTitleCommand) CommandType() string {
	return "ChangeTitle"
}

// SetSeverityCommand - команда установки серьезности бага
type SetSeverityCommand struct {
	ChatID   uuid.UUID
	Severity string
}

// CommandType возвращает тип команды
func (c SetSeverityCommand) CommandType() string {
	return "SetSeverity"
}
