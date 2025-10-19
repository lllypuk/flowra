package chat

import (
	"time"

	"github.com/lllypuk/teams-up/internal/domain/chat"
	"github.com/lllypuk/teams-up/internal/domain/uuid"
)

// CreateChatCommand содержит данные для создания нового чата
type CreateChatCommand struct {
	WorkspaceID uuid.UUID
	Title       string    // для typed чатов (Task/Bug/Epic)
	Type        chat.Type // Discussion, Task, Bug, Epic
	IsPublic    bool      // публичный или приватный
	CreatedBy   uuid.UUID
}

// CommandName возвращает имя команды
func (c CreateChatCommand) CommandName() string { return "CreateChat" }

// AddParticipantCommand содержит данные для добавления участника
type AddParticipantCommand struct {
	ChatID  uuid.UUID
	UserID  uuid.UUID
	Role    chat.Role // Admin, Member
	AddedBy uuid.UUID
}

// CommandName возвращает имя команды
func (c AddParticipantCommand) CommandName() string { return "AddParticipant" }

// RemoveParticipantCommand содержит данные для удаления участника
type RemoveParticipantCommand struct {
	ChatID    uuid.UUID
	UserID    uuid.UUID
	RemovedBy uuid.UUID
}

// CommandName возвращает имя команды
func (c RemoveParticipantCommand) CommandName() string { return "RemoveParticipant" }

// ConvertToTaskCommand содержит данные для конвертации чата в Task
type ConvertToTaskCommand struct {
	ChatID      uuid.UUID
	Title       string // новый заголовок
	ConvertedBy uuid.UUID
}

// CommandName возвращает имя команды
func (c ConvertToTaskCommand) CommandName() string { return "ConvertToTask" }

// ConvertToBugCommand содержит данные для конвертации чата в Bug
type ConvertToBugCommand struct {
	ChatID      uuid.UUID
	Title       string // новый заголовок
	ConvertedBy uuid.UUID
}

// CommandName возвращает имя команды
func (c ConvertToBugCommand) CommandName() string { return "ConvertToBug" }

// ConvertToEpicCommand содержит данные для конвертации чата в Epic
type ConvertToEpicCommand struct {
	ChatID      uuid.UUID
	Title       string // новый заголовок
	ConvertedBy uuid.UUID
}

// CommandName возвращает имя команды
func (c ConvertToEpicCommand) CommandName() string { return "ConvertToEpic" }

// ChangeStatusCommand содержит данные для изменения статуса
type ChangeStatusCommand struct {
	ChatID    uuid.UUID
	Status    string // зависит от типа чата
	ChangedBy uuid.UUID
}

// CommandName возвращает имя команды
func (c ChangeStatusCommand) CommandName() string { return "ChangeStatus" }

// AssignUserCommand содержит данные для назначения пользователя
type AssignUserCommand struct {
	ChatID     uuid.UUID
	AssigneeID *uuid.UUID // nil = снять assignee
	AssignedBy uuid.UUID
}

// CommandName возвращает имя команды
func (c AssignUserCommand) CommandName() string { return "AssignUser" }

// SetPriorityCommand содержит данные для установки приоритета
type SetPriorityCommand struct {
	ChatID   uuid.UUID
	Priority string // Low, Medium, High, Critical
	SetBy    uuid.UUID
}

// CommandName возвращает имя команды
func (c SetPriorityCommand) CommandName() string { return "SetPriority" }

// SetDueDateCommand содержит данные для установки дедлайна
type SetDueDateCommand struct {
	ChatID  uuid.UUID
	DueDate *time.Time // nil = снять дедлайн
	SetBy   uuid.UUID
}

// CommandName возвращает имя команды
func (c SetDueDateCommand) CommandName() string { return "SetDueDate" }

// RenameChatCommand содержит данные для переименования чата
type RenameChatCommand struct {
	ChatID    uuid.UUID
	NewTitle  string
	RenamedBy uuid.UUID
}

// CommandName возвращает имя команды
func (c RenameChatCommand) CommandName() string { return "RenameChat" }

// SetSeverityCommand содержит данные для установки severity (только для Bug)
type SetSeverityCommand struct {
	ChatID   uuid.UUID
	Severity string // Minor, Major, Critical, Blocker
	SetBy    uuid.UUID
}

// CommandName возвращает имя команды
func (c SetSeverityCommand) CommandName() string { return "SetSeverity" }
